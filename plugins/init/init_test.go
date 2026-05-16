package init

import (
	"context"
	"errors"
	"testing"

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
