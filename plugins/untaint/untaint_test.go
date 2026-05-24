package untaint

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func newTestPlugin(svc *sdktest.MockService) *Plugin {
	p := New(svc).(*Plugin)
	p.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	return p
}

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "untaint" {
		t.Errorf("ID() = %q, want %q", p.ID(), "untaint")
	}
	if p.Name() != "Untaint" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Untaint")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(map[string]interface{}{}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	ctx := &sdk.PluginDeps{
		Service: svc,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if cmd := p.Init(ctx); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
	if p.Ready() {
		t.Error("Ready() should be false before activation")
	}
}

func TestPlugin_WhenActivated_ShouldRequestConfirmation(t *testing.T) {
	tests := []struct {
		name      string
		addresses []string
		wantMode  sdk.InputRequestMode
	}{
		{"ShouldDeactivateWhenNoAddresses", nil, 0},
		{"ShouldConfirmSingleAddress", []string{"aws_instance.web"}, sdk.InputRequestBool},
		{"ShouldConfirmMultipleAddresses", []string{"aws_instance.a", "aws_instance.b"}, sdk.InputRequestBool},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &sdktest.MockService{}
			p := newTestPlugin(svc)
			p.SetTargets(tt.addresses)

			cmd := p.Activate()
			if cmd == nil {
				t.Fatal("Activate() should return a cmd")
			}

			msg := cmd()
			if len(tt.addresses) == 0 {
				if _, ok := msg.(sdk.DeactivateMsg); !ok {
					t.Fatalf("expected sdk.DeactivateMsg for empty addresses, got %T", msg)
				}
				return
			}

			reqMsg, ok := msg.(sdk.RequestInputMsg)
			if !ok {
				t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
			}
			if reqMsg.Request.Mode != tt.wantMode {
				t.Errorf("request mode = %v, want %v", reqMsg.Request.Mode, tt.wantMode)
			}
		})
	}
}

func TestPlugin_WhenActivatedWhileLoading_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() while loading should return nil")
	}
}

func TestPlugin_WhenConfirmed_ShouldExecuteUntaint(t *testing.T) {
	tests := []struct {
		name          string
		addresses     []string
		untaintErr    error
		wantStatus    sdk.Status
		wantUntainted int
	}{
		{"ShouldUntaintSingleAddress", []string{"aws_instance.web"}, nil, sdk.StatusDone, 1},
		{"ShouldUntaintMultipleAddresses", []string{"aws_instance.a", "aws_instance.b"}, nil, sdk.StatusDone, 2},
		{"ShouldSetErrorOnFailure", []string{"aws_instance.web"}, errors.New("untaint failed"), sdk.StatusError, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &sdktest.MockService{
				UntaintFn: func(_ context.Context, _ string) error {
					return tt.untaintErr
				},
			}
			p := newTestPlugin(svc)
			p.SetTargets(tt.addresses)

			cmd := p.Activate()
			msg := cmd()
			reqMsg := msg.(sdk.RequestInputMsg)

			startCmd := reqMsg.Request.Callback("y")
			if startCmd == nil {
				t.Fatal("confirm callback should return a cmd")
			}
			startMsg := startCmd()
			if _, ok := startMsg.(untaintStartMsg); !ok {
				t.Fatalf("expected untaintStartMsg, got %T", startMsg)
			}

			_, execCmd := p.Update(startMsg)
			if execCmd == nil {
				t.Fatal("Update(untaintStartMsg) should return exec cmd")
			}
			if p.status != sdk.StatusLoading {
				t.Errorf("status = %v, want StatusLoading", p.status)
			}

			execMsg := execCmd()
			batchMsg, ok := execMsg.(tea.BatchMsg)
			if !ok {
				t.Fatalf("exec cmd returned %T, want tea.BatchMsg", execMsg)
			}
			var result untaintResultMsg
			found := false
			for _, subCmd := range batchMsg {
				if subCmd == nil {
					continue
				}
				if r, ok := subCmd().(untaintResultMsg); ok {
					result = r
					found = true
				}
			}
			if !found {
				t.Fatal("batch did not contain untaintResultMsg")
			}

			_, eventCmd := p.Update(result)
			if p.status != tt.wantStatus {
				t.Errorf("status = %v, want %v", p.status, tt.wantStatus)
			}
			if len(p.untainted) != tt.wantUntainted {
				t.Errorf("untainted count = %d, want %d", len(p.untainted), tt.wantUntainted)
			}

			if tt.untaintErr == nil {
				if eventCmd == nil {
					t.Fatal("success should return event cmd")
				}
				eventMsg := eventCmd()
				if _, ok := eventMsg.(sdk.PlanInvalidatedEvent); !ok {
					t.Errorf("expected sdk.PlanInvalidatedEvent, got %T", eventMsg)
				}
			} else {
				if eventCmd != nil {
					t.Error("error should return nil cmd")
				}
				if p.errMsg == "" {
					t.Error("errMsg should be set on error")
				}
			}
		})
	}
}

func TestPlugin_WhenConfirmDeclined_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.SetTargets([]string{"aws_instance.web"})

	cmd := p.Activate()
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	result := reqMsg.Request.Callback("n")
	if result != nil {
		t.Error("declining should return nil cmd")
	}
}

func TestPlugin_WhenReceivingKeys_ShouldNavigate(t *testing.T) {
	tests := []struct {
		name       string
		status     sdk.Status
		key        tea.KeyMsg
		wantMsg    interface{}
		wantNilCmd bool
	}{
		{"ShouldNavigateToPlanOnP_Done", sdk.StatusDone, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}, sdk.NavigateMsg{PluginID: "plan"}, false},
		{"ShouldDeactivateOnEsc_Done", sdk.StatusDone, tea.KeyMsg{Type: tea.KeyEsc}, sdk.DeactivateMsg{}, false},
		{"ShouldDeactivateOnEsc_Error", sdk.StatusError, tea.KeyMsg{Type: tea.KeyEsc}, sdk.DeactivateMsg{}, false},
		{"ShouldDeactivateOnEsc_Idle", sdk.StatusIdle, tea.KeyMsg{Type: tea.KeyEsc}, sdk.DeactivateMsg{}, false},
		{"ShouldRetryOnCtrlR_Error", sdk.StatusError, tea.KeyMsg{Type: tea.KeyCtrlR}, nil, false},
		{"ShouldIgnoreKeysInLoading", sdk.StatusLoading, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &sdktest.MockService{}
			p := newTestPlugin(svc)
			p.status = tt.status
			p.addresses = []string{"aws_instance.web"}

			_, cmd := p.Update(tt.key)

			if tt.wantNilCmd {
				if cmd != nil {
					t.Error("expected nil cmd")
				}
				return
			}

			if tt.name == "ShouldRetryOnCtrlR_Error" {
				if cmd == nil {
					t.Fatal("ctrl+r in error should return a cmd")
				}
				if p.status != sdk.StatusLoading {
					t.Errorf("status = %v, want StatusLoading after retry", p.status)
				}
				return
			}

			if cmd == nil {
				t.Fatal("expected non-nil cmd")
			}
			msg := cmd()
			switch expected := tt.wantMsg.(type) {
			case sdk.NavigateMsg:
				nav, ok := msg.(sdk.NavigateMsg)
				if !ok {
					t.Fatalf("expected sdk.NavigateMsg, got %T", msg)
				}
				if nav.PluginID != expected.PluginID {
					t.Errorf("PluginID = %q, want %q", nav.PluginID, expected.PluginID)
				}
			case sdk.DeactivateMsg:
				if _, ok := msg.(sdk.DeactivateMsg); !ok {
					t.Fatalf("expected sdk.DeactivateMsg, got %T", msg)
				}
			}
		})
	}
}

func TestPlugin_WhenTimerTicks_ShouldPropagate(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	p.timer.Start()

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd == nil {
		t.Error("TimerTickMsg should return tick cmd while running")
	}
}

func TestPlugin_WhenViewRendered_ShouldShowCorrectContent(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(p *Plugin)
		wantNe string
	}{
		{"ShouldShowWaitingInIdle", func(p *Plugin) {
			p.status = sdk.StatusIdle
		}, ""},
		{"ShouldShowProgressInLoading", func(p *Plugin) {
			p.status = sdk.StatusLoading
			p.addresses = []string{"aws_instance.web"}
		}, ""},
		{"ShouldShowSuccessInDone", func(p *Plugin) {
			p.status = sdk.StatusDone
			p.untainted = []string{"aws_instance.web"}
		}, ""},
		{"ShouldShowMultiSuccessInDone", func(p *Plugin) {
			p.status = sdk.StatusDone
			p.untainted = []string{"a", "b", "c"}
		}, ""},
		{"ShouldShowErrorInError", func(p *Plugin) {
			p.status = sdk.StatusError
			p.errMsg = "something went wrong"
		}, ""},
		{"ShouldReturnEmptyForUnknownStatus", func(p *Plugin) {
			p.status = sdk.Status(99)
		}, "NONE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &sdktest.MockService{}
			p := newTestPlugin(svc)
			tt.setup(p)

			view := p.View(80, 24)
			if tt.wantNe == "NONE" {
				if view != "" {
					t.Errorf("View() = %q, want empty", view)
				}
			} else {
				if view == "" {
					t.Error("View() returned empty string")
				}
			}
		})
	}
}

func TestPlugin_WhenHandleContextChanged_ShouldReset(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.addresses = []string{"aws_instance.web"}
	p.errMsg = "old error"

	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: &sdk.Context{Service: svc}})

	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want StatusIdle", p.status)
	}
	if p.addresses != nil {
		t.Error("addresses should be cleared")
	}
	if p.errMsg != "" {
		t.Error("errMsg should be cleared")
	}
	if cmd != nil {
		t.Error("HandleContextChanged should return nil cmd")
	}
}

func TestPlugin_WhenUntaintPartiallyFails_ShouldReportUntaintedAndError(t *testing.T) {
	calls := 0
	svc := &sdktest.MockService{
		UntaintFn: func(_ context.Context, addr string) error {
			calls++
			if calls >= 2 {
				return errors.New("untaint failed on " + addr)
			}
			return nil
		},
	}
	p := newTestPlugin(svc)
	p.Svc = svc
	p.addresses = []string{"aws_instance.a", "aws_instance.b", "aws_instance.c"}

	cmd := p.executeUntaint()
	msg := cmd()
	batchMsg := msg.(tea.BatchMsg)
	var result untaintResultMsg
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(untaintResultMsg); ok {
			result = r
		}
	}

	p.Update(result)
	if p.status != sdk.StatusError {
		t.Errorf("status = %v, want StatusError", p.status)
	}
	if len(p.untainted) != 1 {
		t.Errorf("untainted = %d, want 1 (first succeeded before failure)", len(p.untainted))
	}
}

func TestPlugin_WhenBusy_ShouldReportLoadingState(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)

	p.status = sdk.StatusIdle
	if p.Busy() {
		t.Error("Busy() = true in idle, want false")
	}

	p.status = sdk.StatusLoading
	if !p.Busy() {
		t.Error("Busy() = false in loading, want true")
	}

	p.status = sdk.StatusDone
	if p.Busy() {
		t.Error("Busy() = true in done, want false")
	}
}

func TestPlugin_WhenUnhandledMsg_ShouldReturnSelf(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)

	result, cmd := p.Update(struct{}{})
	if result.(*Plugin) != p {
		t.Error("unhandled msg should return same plugin")
	}
	if cmd != nil {
		t.Error("unhandled msg should return nil cmd")
	}
}

func TestPlugin_WhenCancelCalledWithNilFn_ShouldNotPanic(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.cancelFn = nil
	p.Cancel()
}

func TestPlugin_WhenCancelCalledWithFn_ShouldCallAndClear(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	called := false
	p.cancelFn = func() { called = true }
	p.Cancel()
	if !called {
		t.Error("Cancel() should call cancelFn")
	}
	if p.cancelFn != nil {
		t.Error("Cancel() should set cancelFn to nil")
	}
}

func TestPlugin_WhenHintsInDone_ShouldReturnPlanAndCancel(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone

	hints := p.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() in Done: len = %d, want 2", len(hints))
	}
	if hints[0].Key != "p" || hints[0].Description != "plan" {
		t.Errorf("hints[0] = {%q, %q}, want {p, plan}", hints[0].Key, hints[0].Description)
	}
	if hints[1].Key != "Esc" || hints[1].Description != "cancel" {
		t.Errorf("hints[1] = {%q, %q}, want {Esc, cancel}", hints[1].Key, hints[1].Description)
	}
}

func TestPlugin_WhenHintsInError_ShouldReturnRetryAndBack(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusError

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() in Error returned empty slice")
	}
	hasRetry := false
	for _, h := range hints {
		if h.Description == "retry" {
			hasRetry = true
		}
	}
	if !hasRetry {
		t.Error("Hints() in Error should contain 'retry'")
	}
}

func TestPlugin_WhenHintsInIdle_ShouldReturnBack(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusIdle

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() in Idle returned empty slice")
	}
	hasBack := false
	for _, h := range hints {
		if h.Description == "back" {
			hasBack = true
		}
	}
	if !hasBack {
		t.Error("Hints() in Idle should contain 'back'")
	}
}

func TestPlugin_WhenViewInLoadingWithMultipleAddresses_ShouldShowResourceCount(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading
	p.addresses = []string{"aws_instance.a", "aws_instance.b", "aws_instance.c"}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() in loading with multiple addresses returned empty")
	}
	if !strings.Contains(view, "3 resources") {
		t.Errorf("View() should show '3 resources', got %q", view)
	}
}

func TestHandleContextChanged_ShouldClearAddressesAndReset(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusError
	p.addresses = []string{"a", "b"}
	p.errMsg = "boom"
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: &sdk.Context{Service: svc}})
	if cmd != nil {
		t.Error("HandleContextChanged returned non-nil cmd")
	}
	if p.status != sdk.StatusIdle || len(p.addresses) != 0 || p.errMsg != "" {
		t.Errorf("state not reset: status=%v addrs=%v errMsg=%q", p.status, p.addresses, p.errMsg)
	}
}

func TestHandleContextChanged_WhenNextNil_ShouldBeNoOp(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.addresses = []string{"keep"}
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil})
	if cmd != nil {
		t.Error("HandleContextChanged with nil Next returned non-nil cmd")
	}
	if len(p.addresses) != 1 {
		t.Errorf("addresses mutated, got %v", p.addresses)
	}
}
