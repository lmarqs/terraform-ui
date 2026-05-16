package tfimport

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
	importErr  error
	importAddr string
	importID   string
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
func (m *mockService) StateRm(_ context.Context, _ string) error      { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error  { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error         { return nil }
func (m *mockService) Untaint(_ context.Context, _ string) error       { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error) { return nil, nil }
func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}
func (m *mockService) Refresh(_ context.Context) error                     { return nil }
func (m *mockService) Init(_ context.Context, _ sdk.InitOptions) error     { return nil }
func (m *mockService) ForceUnlock(_ context.Context, _ string) error       { return nil }
func (m *mockService) Version(_ context.Context) (*sdk.VersionInfo, error) { return nil, nil }
func (m *mockService) WithDir(_ string) sdk.Service                        { return m }
func (m *mockService) Import(_ context.Context, addr, id string) error {
	m.importAddr = addr
	m.importID = id
	return m.importErr
}

func newTestPlugin(svc *mockService) *Plugin {
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	return p
}

func TestPlugin_WhenCreated_ShouldHaveCorrectMetadata(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	if p.ID() != "import" {
		t.Errorf("ID() = %q, want %q", p.ID(), "import")
	}
	if p.Name() != "Import" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Import")
	}
	if p.Description() != "Import existing infrastructure into terraform state" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Import existing infrastructure into terraform state")
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

func TestPlugin_WhenActivated_ShouldRequestAddress(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.SetAddress("aws_instance.web")

	cmd := p.Activate()
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
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() while loading should return nil")
	}
}

func TestPlugin_WhenActivatedWithNoAddress_ShouldRequestEmptyDefault(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)

	cmd := p.Activate()
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
	svc := &mockService{}
	p := newTestPlugin(svc)

	cmd := p.Activate()
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
	svc := &mockService{}
	p := newTestPlugin(svc)

	cmd := p.Activate()
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
	svc := &mockService{}
	p := newTestPlugin(svc)

	cmd := p.Activate()
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
			svc := &mockService{importErr: tt.importErr}
			p := newTestPlugin(svc)
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
				if svc.importAddr != "aws_instance.web" {
					t.Errorf("service.Import addr = %q, want %q", svc.importAddr, "aws_instance.web")
				}
				if svc.importID != "i-1234567890" {
					t.Errorf("service.Import id = %q, want %q", svc.importID, "i-1234567890")
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
			svc := &mockService{}
			p := newTestPlugin(svc)
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
	p.address = "aws_instance.web"
	p.id = "i-123"
	p.errMsg = "old error"

	cmd := p.HandleChdirChanged(sdk.ChdirChangedEvent{AbsPath: "/new/path"})

	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want StatusIdle", p.status)
	}
	if p.address != "" {
		t.Error("address should be cleared")
	}
	if p.id != "" {
		t.Error("id should be cleared")
	}
	if p.errMsg != "" {
		t.Error("errMsg should be cleared")
	}
	if cmd != nil {
		t.Error("HandleChdirChanged should return nil cmd")
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

func TestPlugin_WhenSetAddress_ShouldPreFillForm(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.SetAddress("aws_instance.web")

	if p.address != "aws_instance.web" {
		t.Errorf("address = %q, want %q", p.address, "aws_instance.web")
	}
}

func TestPlugin_WhenImportSubmitMsg_ShouldStoreAddressAndID(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)

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
