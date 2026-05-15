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
	if p.Ready() {
		t.Error("Ready() = true before init runs, want false")
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

func TestSubmit_TransitionsToLoading(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.upgrade = true

	cmd := p.submit()
	if cmd == nil {
		t.Fatal("submit() should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want StatusLoading", p.status)
	}
}

func TestInitResultMsg_Success(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	updated, cmd := p.Update(InitResultMsg{Err: nil})
	plug := updated.(*Plugin)

	if plug.status != sdk.StatusDone {
		t.Errorf("status = %v, want StatusDone", plug.status)
	}
	if cmd == nil {
		t.Fatal("success should return PlanInvalidatedEvent command")
	}
	msg := cmd()
	if _, ok := msg.(sdk.PlanInvalidatedEvent); !ok {
		t.Errorf("cmd() = %T, want PlanInvalidatedEvent", msg)
	}
}

func TestInitResultMsg_Error(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	updated, cmd := p.Update(InitResultMsg{Err: errors.New("backend error")})
	plug := updated.(*Plugin)

	if plug.status != sdk.StatusError {
		t.Errorf("status = %v, want StatusError", plug.status)
	}
	if plug.errMsg != "backend error" {
		t.Errorf("errMsg = %q, want %q", plug.errMsg, "backend error")
	}
	if cmd != nil {
		t.Error("error should not emit a command")
	}
}

func TestRerun_FromDone(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})

	if p.stack.Peek() == nil {
		t.Fatal("Enter from Done should push form onto stack")
	}
}

func TestRerun_FromError(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusError

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})

	if p.stack.Peek() == nil {
		t.Fatal("Enter from Error should push form onto stack")
	}
}

func TestHandleChdirChanged(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.errMsg = "old error"

	p.HandleChdirChanged(sdk.ChdirChangedEvent{AbsPath: "/new/dir"})

	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want StatusIdle after chdir", p.status)
	}
	if p.errMsg != "" {
		t.Errorf("errMsg should be cleared after chdir")
	}
}

func TestTimerTick(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	_, cmd := p.Update(ui.TimerTickMsg{})
	// Timer.Tick() returns nil when timer isn't running — that's fine
	_ = cmd
}

func TestView_Loading(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	view := p.View(80, 24)
	if view == "" {
		t.Error("View in Loading state should not be empty")
	}
}

func TestView_Done(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone

	view := p.View(80, 24)
	if view == "" {
		t.Error("View in Done state should not be empty")
	}
}

func TestView_Error(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusError
	p.errMsg = "something failed"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View in Error state should not be empty")
	}
}

func TestConfigure(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	if err := p.Configure(map[string]interface{}{"key": "value"}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}
