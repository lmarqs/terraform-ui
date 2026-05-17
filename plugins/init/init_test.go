package init

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

type mockService struct {
	initErr  error
	initOpts sdk.InitOptions
}

func (m *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return nil, nil
}
func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (m *mockService) StateList(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
	return nil, nil
}
func (m *mockService) Show(_ context.Context, _ string) (string, error) { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)      { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"default"}, nil
}
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	return nil
}
func (m *mockService) WorkspaceDelete(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	return nil
}
func (m *mockService) StateRm(_ context.Context, _ string) error      { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error { return nil }
func (m *mockService) Import(_ context.Context, _, _ string) error    { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error        { return nil }
func (m *mockService) Untaint(_ context.Context, _ string) error      { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error) {
	return nil, nil
}
func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}
func (m *mockService) Refresh(_ context.Context) error { return nil }
func (m *mockService) Init(_ context.Context, opts sdk.InitOptions) error {
	m.initOpts = opts
	return m.initErr
}
func (m *mockService) ForceUnlock(_ context.Context, _ string) error       { return nil }
func (m *mockService) Version(_ context.Context) (*sdk.VersionInfo, error) { return nil, nil }
func (m *mockService) WithDir(_ string) sdk.Service                        { return m }

func TestNew(t *testing.T) {
	p := New(&mockService{}).(*Plugin)

	if p.ID() != "init" {
		t.Errorf("ID() = %q, want %q", p.ID(), "init")
	}
	if p.Name() != "Init" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Init")
	}
	if p.Description() != "Initialize terraform working directory" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Initialize terraform working directory")
	}
	if !p.Ready() {
		t.Error("Ready() should always be true")
	}
	if !p.backend {
		t.Error("backend should default to true")
	}
}

func TestActivate_ShowsForm(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()

	if p.stack.Peek() == nil {
		t.Fatal("Activate() should push form onto stack")
	}
	if p.stack.Peek().ID() != "form" {
		t.Errorf("top frame ID = %q, want %q", p.stack.Peek().ID(), "form")
	}
}

func TestToggleFields(t *testing.T) {
	p := New(&mockService{}).(*Plugin)

	if p.upgrade {
		t.Fatal("upgrade should start false")
	}
	p.upgrade = !p.upgrade
	if !p.upgrade {
		t.Fatal("toggle should flip to true")
	}

	if p.reconfigure {
		t.Fatal("reconfigure should start false")
	}
	if !p.backend {
		t.Fatal("backend should start true")
	}
}

func TestSubmit_PushesResultFrame(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()

	p.Update(initSubmitMsg{})

	top := p.stack.Peek()
	if top == nil {
		t.Fatal("submit should push result frame onto stack")
	}
	if top.ID() != "result" {
		t.Errorf("top frame ID = %q, want %q", top.ID(), "result")
	}
}

func TestInitResultMsg_Success_EmitsDeactivate(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})

	_, cmd := p.Update(InitResultMsg{Err: nil})
	if cmd == nil {
		t.Fatal("success should return a command")
	}
}

func TestInitResultMsg_Error_StaysOnResultFrame(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})

	p.Update(InitResultMsg{Err: errors.New("backend error")})

	top := p.stack.Peek()
	if top == nil {
		t.Fatal("error should keep result frame on stack")
	}
	rf, ok := top.(*resultFrame)
	if !ok {
		t.Fatalf("top frame is %T, want *resultFrame", top)
	}
	if rf.status != sdk.StatusError {
		t.Errorf("result frame status = %v, want StatusError", rf.status)
	}
	if rf.errMsg != "backend error" {
		t.Errorf("result frame errMsg = %q, want %q", rf.errMsg, "backend error")
	}
}

func TestError_EnterReturnsToForm(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})
	p.Update(InitResultMsg{Err: errors.New("fail")})

	// Enter on error frame should pop back to form
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e', 'n', 't', 'e', 'r'}})

	// Actually send as proper enter key
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	top := p.stack.Peek()
	if top == nil {
		t.Fatal("after Enter on error, form should be visible")
	}
	if top.ID() != "form" {
		t.Errorf("top frame ID = %q, want %q (form should be back)", top.ID(), "form")
	}
}

func TestHandleChdirChanged(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()

	p.HandleChdirChanged(sdk.ChdirChangedEvent{AbsPath: "/new/dir"})

	if !p.stack.IsEmpty() {
		t.Error("chdir should reset the stack")
	}
}

func TestTimerTick(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})

	_, cmd := p.Update(ui.TimerTickMsg{})
	_ = cmd
}

func TestView_DelegatesToStack(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()

	view := p.View(80, 24)
	if view == "" {
		t.Error("View should delegate to form frame")
	}
}

func TestView_ResultFrame_Loading(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})

	view := p.View(80, 24)
	if view == "" {
		t.Error("View in Loading state should not be empty")
	}
}

func TestView_ResultFrame_Error(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})
	p.Update(InitResultMsg{Err: errors.New("something failed")})

	view := p.View(80, 24)
	if view == "" {
		t.Error("View in Error state should not be empty")
	}
}

func TestBusy(t *testing.T) {
	p := New(&mockService{}).(*Plugin)

	if p.Busy() {
		t.Error("should not be busy before submit")
	}

	p.Activate()
	p.Update(initSubmitMsg{})

	if !p.Busy() {
		t.Error("should be busy during loading")
	}

	p.Update(InitResultMsg{Err: errors.New("fail")})

	if p.Busy() {
		t.Error("should not be busy after result")
	}
}

func TestConfigure(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	if err := p.Configure(map[string]interface{}{"key": "value"}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestPlugin_WhenInit_ShouldStoreService(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	newSvc := &mockService{}

	cmd := p.Init(&sdk.Context{Service: newSvc})
	if cmd != nil {
		t.Error("Init() should return nil")
	}
	if p.svc != newSvc {
		t.Error("Init() should store ctx.Service")
	}
}

func TestPlugin_WhenStack_ShouldReturnNonNil(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	if p.Stack() == nil {
		t.Fatal("Stack() = nil, want non-nil")
	}
}

func TestPlugin_WhenSubmitFromForm_ShouldEmitInitSubmitMsg(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	cmd := p.submitFromForm()
	if cmd == nil {
		t.Fatal("submitFromForm() = nil")
	}
	msg := cmd()
	if _, ok := msg.(initSubmitMsg); !ok {
		t.Errorf("cmd() = %T, want initSubmitMsg", msg)
	}
}

func TestPlugin_WhenEditExtraArgs_ShouldEmitRequestInputMsg(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	cmd := p.editExtraArgs()
	if cmd == nil {
		t.Fatal("editExtraArgs() = nil")
	}
	msg := cmd()
	if _, ok := msg.(sdk.RequestInputMsg); !ok {
		t.Errorf("cmd() = %T, want RequestInputMsg", msg)
	}
}

func TestPlugin_WhenCancelWithNilFn_ShouldNotPanic(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.cancelFn = nil
	p.Cancel()
}

func TestPlugin_WhenCancelWithFn_ShouldCallAndClear(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	called := false
	p.cancelFn = func() { called = true }
	p.Cancel()
	if !called {
		t.Error("Cancel() should call cancelFn")
	}
	if p.cancelFn != nil {
		t.Error("Cancel() should clear cancelFn")
	}
}

func TestPlugin_WhenSubmitWithBackendFalse_ShouldSetBackendPtr(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.backend = false
	p.Activate()
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		msgs := executeBatch(cmd)
		for _, msg := range msgs {
			p.Update(msg)
		}
	}

	if svc.initOpts.Backend == nil {
		t.Fatal("Backend should be non-nil when backend=false")
	}
	if *svc.initOpts.Backend != false {
		t.Error("Backend should be &false")
	}
}

func TestPlugin_WhenSubmitWithBackendTrue_ShouldLeaveBackendNil(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.backend = true
	p.Activate()
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		msgs := executeBatch(cmd)
		for _, msg := range msgs {
			p.Update(msg)
		}
	}

	if svc.initOpts.Backend != nil {
		t.Error("Backend should be nil when backend=true")
	}
}

func TestPlugin_WhenSubmitWithExtraArgs_ShouldSplitFields(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.extraArgs = "-lock=false -input=false"
	p.Activate()
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		msgs := executeBatch(cmd)
		for _, msg := range msgs {
			p.Update(msg)
		}
	}

	if len(svc.initOpts.ExtraArgs) != 2 {
		t.Fatalf("ExtraArgs len = %d, want 2", len(svc.initOpts.ExtraArgs))
	}
	if svc.initOpts.ExtraArgs[0] != "-lock=false" {
		t.Errorf("ExtraArgs[0] = %q, want %q", svc.initOpts.ExtraArgs[0], "-lock=false")
	}
}

func TestPlugin_WhenExtraArgsDisplayEmpty_ShouldReturnNone(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.extraArgs = ""
	if got := p.extraArgsDisplay(); got != "(none)" {
		t.Errorf("extraArgsDisplay() = %q, want %q", got, "(none)")
	}
}

func TestPlugin_WhenExtraArgsDisplaySet_ShouldReturnValue(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.extraArgs = "-lock=false"
	if got := p.extraArgsDisplay(); got != "-lock=false" {
		t.Errorf("extraArgsDisplay() = %q, want %q", got, "-lock=false")
	}
}

func TestPlugin_WhenOutput_ShouldReturnSuccessMessage(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if string(data) != "Initialized successfully.\n" {
		t.Errorf("Output() = %q, want %q", string(data), "Initialized successfully.\n")
	}
}

func TestResultFrame_WhenViewInDone_ShouldShowSuccess(t *testing.T) {
	var timer ui.Timer
	rf := newResultFrame(&timer)
	rf.status = sdk.StatusDone
	view := rf.View(80, 24)
	if view == "" {
		t.Error("View in Done should not be empty")
	}
}

func TestResultFrame_WhenHintsInLoading_ShouldReturnBack(t *testing.T) {
	var timer ui.Timer
	rf := newResultFrame(&timer)
	hints := rf.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints in Loading returned empty")
	}
}

func TestResultFrame_WhenHintsInError_ShouldReturnEnterAndBack(t *testing.T) {
	var timer ui.Timer
	rf := newResultFrame(&timer)
	rf.status = sdk.StatusError
	hints := rf.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints in Error: len = %d, want 2", len(hints))
	}
	if hints[0].Key != "Enter" {
		t.Errorf("hints[0].Key = %q, want %q", hints[0].Key, "Enter")
	}
}

func TestResultFrame_WhenHintsInDone_ShouldReturnNil(t *testing.T) {
	var timer ui.Timer
	rf := newResultFrame(&timer)
	rf.status = sdk.StatusDone
	if hints := rf.Hints(); hints != nil {
		t.Errorf("Hints in Done = %v, want nil", hints)
	}
}

func TestResultFrame_WhenSuccess_ShouldReturnCmd(t *testing.T) {
	var timer ui.Timer
	rf := newResultFrame(&timer)
	_, cmd := rf.Update(InitResultMsg{Err: nil, Duration: time.Second})
	if cmd == nil {
		t.Fatal("success should return cmd")
	}
	if rf.status != sdk.StatusDone {
		t.Errorf("status = %v, want Done", rf.status)
	}
}

func TestResultFrame_WhenError_ShouldSetErrorStatus(t *testing.T) {
	var timer ui.Timer
	rf := newResultFrame(&timer)
	_, cmd := rf.Update(InitResultMsg{Err: errors.New("fail"), Duration: time.Second})
	if cmd != nil {
		t.Error("error should return nil cmd")
	}
	if rf.status != sdk.StatusError {
		t.Errorf("status = %v, want Error", rf.status)
	}
	if rf.errMsg != "fail" {
		t.Errorf("errMsg = %q, want %q", rf.errMsg, "fail")
	}
}

func TestResultFrame_WhenEnterInError_ShouldPop(t *testing.T) {
	var timer ui.Timer
	rf := newResultFrame(&timer)
	rf.status = sdk.StatusError
	frame, _ := rf.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if frame != nil {
		t.Error("Enter in error should return nil to pop")
	}
}

func TestResultFrame_WhenEnterInLoading_ShouldNotPop(t *testing.T) {
	var timer ui.Timer
	rf := newResultFrame(&timer)
	frame, _ := rf.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if frame == nil {
		t.Error("Enter in loading should not pop")
	}
}

func TestPlugin_WhenSubmitWithUpgrade_ShouldPassUpgradeOption(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.upgrade = true
	p.reconfigure = true
	p.Activate()
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		msgs := executeBatch(cmd)
		for _, msg := range msgs {
			p.Update(msg)
		}
	}

	if !svc.initOpts.Upgrade {
		t.Error("Upgrade should be true")
	}
	if !svc.initOpts.Reconfigure {
		t.Error("Reconfigure should be true")
	}
}

func executeBatch(cmd tea.Cmd) []tea.Msg {
	msg := cmd()
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batchMsg {
			if c != nil {
				msgs = append(msgs, c())
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}
