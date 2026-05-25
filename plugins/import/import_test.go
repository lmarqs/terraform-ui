package tfimport

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func newTestPlugin(svc sdk.Service) (*Plugin, *sdktest.PluginDepsHarness) {
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	return p, h
}

// TestPlugin_WhenActivatedWithBothAddrAndID_ShouldSkipFormAndConfirm verifies
// the cmd-side path: when both Addr and ID are provided, the plugin bypasses
// the form and goes straight to the confirm step.
func TestPlugin_WhenActivatedWithBothAddrAndID_ShouldSkipFormAndConfirm(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	input := Input{Addr: "aws_instance.web", ID: "i-abc", JSON: true}
	cmd := p.Activate(input)
	if cmd == nil {
		t.Fatal("Activate() should return a confirm cmd")
	}
	if p.address != "aws_instance.web" || p.id != "i-abc" {
		t.Errorf("address/id = %q/%q, want aws_instance.web/i-abc", p.address, p.id)
	}
	if !p.input.JSON {
		t.Error("Input.JSON should be stored on plugin state")
	}
	msg := cmd()
	reqMsg, ok := msg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("expected sdk.RequestInputMsg (confirm), got %T", msg)
	}
	if reqMsg.Request.Mode != sdk.InputRequestBool {
		t.Errorf("request mode = %v, want InputRequestBool (confirm)", reqMsg.Request.Mode)
	}
}

// TestPlugin_WhenActivatedWithOnlyAddr_ShouldRunForm verifies the TUI path:
// when only Addr is provided the plugin still runs the form (address
// pre-filled, ID prompted).
func TestPlugin_WhenActivatedWithOnlyAddr_ShouldRunForm(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{Addr: "aws_instance.web"})
	if cmd == nil {
		t.Fatal("Activate() should return a form cmd")
	}
	msg := cmd()
	reqMsg, ok := msg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
	}
	if reqMsg.Request.Mode != sdk.InputRequestText {
		t.Errorf("request mode = %v, want InputRequestText", reqMsg.Request.Mode)
	}
	if reqMsg.Request.Default != "aws_instance.web" {
		t.Errorf("default = %q, want %q", reqMsg.Request.Default, "aws_instance.web")
	}
}

// TestPlugin_DoesNotImplementStdoutEmitter pins the contract: import emits no
// stdout content today.
func TestPlugin_DoesNotImplementStdoutEmitter(t *testing.T) {
	p := New(&sdktest.MockService{})
	if _, ok := p.(sdk.StdoutEmitter); ok {
		t.Error("import must not implement sdk.StdoutEmitter (no stdout content)")
	}
}

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "import" {
		t.Errorf("ID() = %q, want %q", p.ID(), "import")
	}
	if p.Name() != "Import" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Import")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(map[string]interface{}{}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	if cmd := p.Init(sdktest.NewDeps(svc).Deps); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
	if p.Ready() {
		t.Error("Ready() should be false before activation")
	}
}

func TestPlugin_WhenActivated_ShouldRequestAddress(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{Addr: "aws_instance.web"})
	if cmd == nil {
		t.Fatal("Activate() should return a cmd")
	}

	msg := cmd()
	reqMsg, ok := msg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
	}
	if reqMsg.Request.Mode != sdk.InputRequestText {
		t.Errorf("request mode = %v, want InputRequestText", reqMsg.Request.Mode)
	}
	if reqMsg.Request.Default != "aws_instance.web" {
		t.Errorf("request default = %q, want %q", reqMsg.Request.Default, "aws_instance.web")
	}
}

func TestPlugin_WhenActivatedWhileLoading_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	cmd := p.Activate(Input{Addr: "aws_instance.web"})
	if cmd != nil {
		t.Error("Activate() while loading should return nil")
	}
}

func TestPlugin_WhenActivatedWithNoAddress_ShouldRequestEmptyDefault(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{})
	if cmd == nil {
		t.Fatal("Activate() should return a cmd")
	}

	msg := cmd()
	reqMsg, ok := msg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
	}
	if reqMsg.Request.Default != "" {
		t.Errorf("request default = %q, want empty", reqMsg.Request.Default)
	}
}

func TestPlugin_WhenAddressSubmittedEmpty_ShouldDeactivate(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	deactivateCmd := reqMsg.Request.Callback("")
	if deactivateCmd == nil {
		t.Fatal("empty address should return a cmd")
	}
	resultMsg := deactivateCmd()
	if _, ok := resultMsg.(sdk.DeactivateMsg); !ok {
		t.Errorf("expected sdk.DeactivateMsg, got %T", resultMsg)
	}
}

func TestPlugin_WhenIDSubmittedEmpty_ShouldDeactivate(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	idCmd := reqMsg.Request.Callback("aws_instance.web")
	if idCmd == nil {
		t.Fatal("address callback should return a cmd")
	}
	idMsg := idCmd()
	idReqMsg, ok := idMsg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("expected sdk.RequestInputMsg for ID, got %T", idMsg)
	}
	if idReqMsg.Request.Mode != sdk.InputRequestText {
		t.Errorf("request mode = %v, want InputRequestText", idReqMsg.Request.Mode)
	}

	deactivateCmd := idReqMsg.Request.Callback("")
	if deactivateCmd == nil {
		t.Fatal("empty ID should return a cmd")
	}
	resultMsg := deactivateCmd()
	if _, ok := resultMsg.(sdk.DeactivateMsg); !ok {
		t.Errorf("expected sdk.DeactivateMsg, got %T", resultMsg)
	}
}

func TestPlugin_WhenFormCompleted_ShouldRequestConfirmation(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	idCmd := reqMsg.Request.Callback("aws_instance.web")
	idMsg := idCmd()
	idReqMsg := idMsg.(sdk.RequestInputMsg)

	submitCmd := idReqMsg.Request.Callback("i-1234567890")
	if submitCmd == nil {
		t.Fatal("ID callback should return a cmd")
	}
	submitMsg := submitCmd()
	importSubmit, ok := submitMsg.(importSubmitMsg)
	if !ok {
		t.Fatalf("expected importSubmitMsg, got %T", submitMsg)
	}
	if importSubmit.Address != "aws_instance.web" {
		t.Errorf("Address = %q, want %q", importSubmit.Address, "aws_instance.web")
	}
	if importSubmit.ID != "i-1234567890" {
		t.Errorf("ID = %q, want %q", importSubmit.ID, "i-1234567890")
	}

	_, confirmCmd := p.Update(importSubmit)
	if confirmCmd == nil {
		t.Fatal("Update(importSubmitMsg) should return confirm cmd")
	}
	confirmMsg := confirmCmd()
	confirmReqMsg, ok := confirmMsg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("expected sdk.RequestInputMsg for confirm, got %T", confirmMsg)
	}
	if confirmReqMsg.Request.Mode != sdk.InputRequestBool {
		t.Errorf("request mode = %v, want InputRequestBool", confirmReqMsg.Request.Mode)
	}
}

func TestPlugin_WhenConfirmed_ShouldExecuteImport(t *testing.T) {
	tests := []struct {
		name       string
		importErr  error
		wantStatus sdk.Status
	}{
		{"ShouldImportSuccessfully", nil, sdk.StatusDone},
		{"ShouldSetErrorOnFailure", errors.New("import failed"), sdk.StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &sdktest.MockService{
				ImportFn: func(_ context.Context, _, _ string) error {
					return tt.importErr
				},
			}
			p, _ := newTestPlugin(svc)
			p.address = "aws_instance.web"
			p.id = "i-1234567890"

			_, confirmCmd := p.Update(importSubmitMsg{Address: "aws_instance.web", ID: "i-1234567890"})
			confirmMsg := confirmCmd()
			confirmReqMsg := confirmMsg.(sdk.RequestInputMsg)

			startCmd := confirmReqMsg.Request.Callback("y")
			if startCmd == nil {
				t.Fatal("confirm callback should return a cmd")
			}
			startMsg := startCmd()
			if _, ok := startMsg.(importStartMsg); !ok {
				t.Fatalf("expected importStartMsg, got %T", startMsg)
			}

			_, execCmd := p.Update(startMsg)
			if execCmd == nil {
				t.Fatal("Update(importStartMsg) should return exec cmd")
			}
			if p.status != sdk.StatusLoading {
				t.Errorf("status = %v, want StatusLoading", p.status)
			}

			execMsg := execCmd()
			batchMsg, ok := execMsg.(tea.BatchMsg)
			if !ok {
				t.Fatalf("exec cmd returned %T, want tea.BatchMsg", execMsg)
			}
			var result importResultMsg
			found := false
			for _, subCmd := range batchMsg {
				if subCmd == nil {
					continue
				}
				if r, ok := subCmd().(importResultMsg); ok {
					result = r
					found = true
				}
			}
			if !found {
				t.Fatal("batch did not contain importResultMsg")
			}

			_, eventCmd := p.Update(result)
			if p.status != tt.wantStatus {
				t.Errorf("status = %v, want %v", p.status, tt.wantStatus)
			}

			if tt.importErr == nil {
				if eventCmd == nil {
					t.Fatal("success should return event cmd")
				}
				batchEvt := eventCmd()
				evtBatch, ok := batchEvt.(tea.BatchMsg)
				if !ok {
					t.Fatalf("expected tea.BatchMsg for events, got %T", batchEvt)
				}
				var hasStateRefreshed, hasPlanInvalidated bool
				for _, subCmd := range evtBatch {
					if subCmd == nil {
						continue
					}
					msg := subCmd()
					switch msg.(type) {
					case sdk.StateRefreshedEvent:
						hasStateRefreshed = true
					case sdk.PlanInvalidatedEvent:
						hasPlanInvalidated = true
					}
				}
				if !hasStateRefreshed {
					t.Error("expected sdk.StateRefreshedEvent in batch")
				}
				if !hasPlanInvalidated {
					t.Error("expected sdk.PlanInvalidatedEvent in batch")
				}
				if len(svc.ImportCalls) == 0 || svc.ImportCalls[0][0] != "aws_instance.web" {
					t.Errorf("service.Import addr = %v, want [aws_instance.web ...]", svc.ImportCalls)
				}
				if len(svc.ImportCalls) == 0 || svc.ImportCalls[0][1] != "i-1234567890" {
					t.Errorf("service.Import id = %v, want [... i-1234567890]", svc.ImportCalls)
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
	p, _ := newTestPlugin(svc)
	p.address = "aws_instance.web"
	p.id = "i-123"

	_, confirmCmd := p.Update(importSubmitMsg{Address: "aws_instance.web", ID: "i-123"})
	confirmMsg := confirmCmd()
	confirmReqMsg := confirmMsg.(sdk.RequestInputMsg)

	result := confirmReqMsg.Request.Callback("n")
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
		{"ShouldDeactivateOnEsc_Form", StatusForm, tea.KeyMsg{Type: tea.KeyEsc}, sdk.DeactivateMsg{}, false},
		{"ShouldDeactivateOnEsc_Idle", sdk.StatusIdle, tea.KeyMsg{Type: tea.KeyEsc}, sdk.DeactivateMsg{}, false},
		{"ShouldRetryOnCtrlR_Error", sdk.StatusError, tea.KeyMsg{Type: tea.KeyCtrlR}, nil, false},
		{"ShouldIgnoreKeysInLoading", sdk.StatusLoading, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &sdktest.MockService{}
			p, _ := newTestPlugin(svc)
			p.status = tt.status
			p.address = "aws_instance.web"
			p.id = "i-123"

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
	p, _ := newTestPlugin(svc)
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
		{"ShouldShowFormInStatusForm", func(p *Plugin) {
			p.status = StatusForm
		}, ""},
		{"ShouldShowFormInIdle", func(p *Plugin) {
			p.status = sdk.StatusIdle
		}, ""},
		{"ShouldShowProgressInLoading", func(p *Plugin) {
			p.status = sdk.StatusLoading
			p.address = "aws_instance.web"
		}, ""},
		{"ShouldShowSuccessInDone", func(p *Plugin) {
			p.status = sdk.StatusDone
			p.address = "aws_instance.web"
			p.id = "i-123"
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
			p, _ := newTestPlugin(svc)
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

func TestPlugin_WhenBusy_ShouldReportLoadingState(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

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
	p, _ := newTestPlugin(svc)

	result, cmd := p.Update(struct{}{})
	if result.(*Plugin) != p {
		t.Error("unhandled msg should return same plugin")
	}
	if cmd != nil {
		t.Error("unhandled msg should return nil cmd")
	}
}

func TestPlugin_WhenActivatedWithAddr_ShouldPreFillForm(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{Addr: "aws_instance.web"})
	if cmd == nil {
		t.Fatal("Activate() should return a cmd")
	}
	if p.address != "aws_instance.web" {
		t.Errorf("address = %q, want %q", p.address, "aws_instance.web")
	}
}

func TestPlugin_WhenImportSubmitMsg_ShouldStoreAddressAndID(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	_, cmd := p.Update(importSubmitMsg{Address: "aws_instance.web", ID: "i-abc"})
	if cmd == nil {
		t.Fatal("Update(importSubmitMsg) should return confirm cmd")
	}
	if p.address != "aws_instance.web" {
		t.Errorf("address = %q, want %q", p.address, "aws_instance.web")
	}
	if p.id != "i-abc" {
		t.Errorf("id = %q, want %q", p.id, "i-abc")
	}
}

func TestPlugin_WhenCancelWithNilFn_ShouldNotPanic(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.cancelFn = nil
	p.Cancel()
}

func TestPlugin_WhenCancelWithFn_ShouldCallAndClear(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
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
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	hints := p.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() in Done: len = %d, want 2", len(hints))
	}
	if hints[0].Key != "p" {
		t.Errorf("hints[0].Key = %q, want %q", hints[0].Key, "p")
	}
}

func TestPlugin_WhenHintsInError_ShouldReturnRetryAndBack(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusError
	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() in Error returned empty")
	}
}

func TestPlugin_WhenHintsInIdle_ShouldReturnBack(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusIdle
	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() in Idle returned empty")
	}
}

func TestHandleContextChanged_ShouldResetState(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusError
	p.address = "old.addr"
	p.id = "old-id"
	p.errMsg = "boom"
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: &sdk.Context{Service: svc}})
	if cmd != nil {
		t.Error("HandleContextChanged returned non-nil cmd")
	}
	if p.status != sdk.StatusIdle || p.address != "" || p.id != "" || p.errMsg != "" {
		t.Errorf("state not reset: status=%v addr=%q id=%q errMsg=%q", p.status, p.address, p.id, p.errMsg)
	}
}

func TestHandleContextChanged_WhenNextNil_ShouldBeNoOp(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.address = "keep"
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil})
	if cmd != nil {
		t.Error("HandleContextChanged with nil Next returned non-nil cmd")
	}
	if p.address != "keep" {
		t.Errorf("address mutated, got %q", p.address)
	}
}
