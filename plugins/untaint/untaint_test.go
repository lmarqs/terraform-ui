package untaint

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

type mockService struct {
	untaintErr    error
	untaintCalled []string
}

func (m *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return nil, nil
}
func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (m *mockService) StateList(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
	return nil, nil
}
func (m *mockService) Show(_ context.Context, _ string) (string, error)  { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)       { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	return nil
}
func (m *mockService) WorkspaceDelete(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	return nil
}
func (m *mockService) StateRm(_ context.Context, _ string) error            { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error       { return nil }
func (m *mockService) Import(_ context.Context, _, _ string) error          { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error              { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error) { return nil, nil }
func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}
func (m *mockService) Refresh(_ context.Context) error                     { return nil }
func (m *mockService) Init(_ context.Context, _ sdk.InitOptions) error     { return nil }
func (m *mockService) ForceUnlock(_ context.Context, _ string) error       { return nil }
func (m *mockService) Version(_ context.Context) (*sdk.VersionInfo, error) { return nil, nil }
func (m *mockService) WithDir(_ string) sdk.Service                        { return m }
func (m *mockService) Untaint(_ context.Context, addr string) error {
	m.untaintCalled = append(m.untaintCalled, addr)
	return m.untaintErr
}

func newTestPlugin(svc *mockService) *Plugin {
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	return p
}

func TestPlugin_WhenCreated_ShouldHaveCorrectMetadata(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	if p.ID() != "untaint" {
		t.Errorf("ID() = %q, want %q", p.ID(), "untaint")
	}
	if p.Name() != "Untaint" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Untaint")
	}
	if p.Description() != "Remove taint mark from resources" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Remove taint mark from resources")
	}
	if p.Ready() {
		t.Error("Ready() = true, want false for new plugin")
	}
}

func TestPlugin_WhenConfigured_ShouldAcceptAnyConfig(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	if err := p.Configure(map[string]interface{}{"unknown": "value"}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestPlugin_WhenInitialized_ShouldStoreContext(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx := &sdk.Context{
		Service: svc,
		Logger:  logger,
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() should return nil cmd")
	}

	pp := p.(*Plugin)
	if pp.svc != svc {
		t.Error("Init() should store service from context")
	}
	if pp.log != logger {
		t.Error("Init() should store logger from context")
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
			svc := &mockService{}
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
	svc := &mockService{}
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
			svc := &mockService{untaintErr: tt.untaintErr}
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
	svc := &mockService{}
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
			svc := &mockService{}
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
	svc := &mockService{}
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
			svc := &mockService{}
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

func TestPlugin_WhenHandleChdirChanged_ShouldReset(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.addresses = []string{"aws_instance.web"}
	p.errMsg = "old error"

	cmd := p.HandleChdirChanged(sdk.ChdirChangedEvent{AbsPath: "/new/path"})

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
		t.Error("HandleChdirChanged should return nil cmd")
	}
}

func TestPlugin_WhenUntaintPartiallyFails_ShouldReportUntaintedAndError(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.addresses = []string{"aws_instance.a", "aws_instance.b", "aws_instance.c"}

	customSvc := &failOnNthService{failOn: 2}
	p.svc = customSvc

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
	svc := &mockService{}
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
	svc := &mockService{}
	p := newTestPlugin(svc)

	result, cmd := p.Update(struct{}{})
	if result.(*Plugin) != p {
		t.Error("unhandled msg should return same plugin")
	}
	if cmd != nil {
		t.Error("unhandled msg should return nil cmd")
	}
}

type failOnNthService struct {
	mockService
	failOn int
	calls  int
}

func (m *failOnNthService) Untaint(_ context.Context, addr string) error {
	m.calls++
	if m.calls >= m.failOn {
		return errors.New("untaint failed on " + addr)
	}
	return nil
}

func TestPlugin_WhenCancelCalledWithNilFn_ShouldNotPanic(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.cancelFn = nil
	p.Cancel()
}

func TestPlugin_WhenCancelCalledWithFn_ShouldCallAndClear(t *testing.T) {
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
