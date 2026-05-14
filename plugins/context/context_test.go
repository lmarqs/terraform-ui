package context

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct{}

func (m *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return nil, nil
}

func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (m *mockService) StateList(_ context.Context) ([]sdk.Resource, error) {
	return nil, nil
}

func (m *mockService) Show(_ context.Context, _ string) (string, error)  { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)       { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string) error    { return nil }
func (m *mockService) WorkspaceDelete(_ context.Context, _ string) error { return nil }
func (m *mockService) StateRm(_ context.Context, _ string) error         { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error    { return nil }
func (m *mockService) Import(_ context.Context, _, _ string) error       { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error           { return nil }
func (m *mockService) Untaint(_ context.Context, _ string) error         { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error) {
	return nil, nil
}

func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}

func (m *mockService) Refresh(_ context.Context) error               { return nil }
func (m *mockService) Init(_ context.Context) error                  { return nil }
func (m *mockService) ForceUnlock(_ context.Context, _ string) error { return nil }
func (m *mockService) WithDir(_ string) sdk.Service                  { return m }

func TestNew(t *testing.T) {
	p := New(nil).(*Plugin)
	if p.ID() != "context" {
		t.Errorf("ID() = %q, want %q", p.ID(), "context")
	}
	if p.Name() != "Context" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Context")
	}
}

func TestReady(t *testing.T) {
	p := New(nil).(*Plugin)
	if !p.Ready() {
		t.Error("should always be ready")
	}
}

func TestActivate_PushesFormFrame(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1", p.stack.Depth())
	}
}

func TestFormNavigation_EnterOnChdir_PushesPickerFrame(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()

	// First selectable field is Chdir (cursor starts there)
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if p.stack.Depth() != 2 {
		t.Errorf("stack depth = %d, want 2 (form + picker)", p.stack.Depth())
	}
}

func TestFormNavigation_EnterOnWorkspace_TriggersFetch(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()

	// No members configured, so first selectable is Workspace
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command to fetch workspaces")
	}

	// Picker not pushed yet — waiting for response
	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1 (fetch in progress)", p.stack.Depth())
	}
}

func TestChdirPicker_SelectEmitsEvent(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()

	// Open chdir picker
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Select first item
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from selection")
	}

	msg := cmd()
	evt, ok := msg.(sdk.ChdirChangedEvent)
	if !ok {
		t.Fatalf("expected ChdirChangedEvent, got %T", msg)
	}
	if evt.RelPath != "modules/vpc" {
		t.Errorf("RelPath = %q, want %q", evt.RelPath, "modules/vpc")
	}
}

func TestChdirPicker_EscPopsBack(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()

	// Open chdir picker
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.stack.Depth() != 2 {
		t.Fatalf("stack depth = %d, want 2", p.stack.Depth())
	}

	// Press esc
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1 after esc", p.stack.Depth())
	}
}

func TestChdirPicker_SelectPopsBack(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()

	// Open chdir picker
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Select
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1 after selection", p.stack.Depth())
	}
}

func TestWorkspacePicker_PushedOnResponse(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	// Simulate workspace list response from user-triggered fetch
	p.Update(workspaceListMsg{workspaces: []string{"default", "staging"}})

	if p.stack.Depth() != 2 {
		t.Errorf("stack depth = %d, want 2 (form + picker)", p.stack.Depth())
	}
}

func TestWorkspacePicker_SelectEmitsEvent(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.workspace = "default"
	p.Activate()

	// Simulate workspace list arriving (user triggered fetch)
	p.Update(workspaceListMsg{workspaces: []string{"default", "staging"}})

	// Move down to "staging"
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from workspace selection")
	}

	msg := cmd()
	evt, ok := msg.(sdk.WorkspaceChangedEvent)
	if !ok {
		t.Fatalf("expected WorkspaceChangedEvent, got %T", msg)
	}
	if evt.Name != "staging" {
		t.Errorf("Name = %q, want %q", evt.Name, "staging")
	}
}

func TestView_ShowsProjectChdirWorkspace(t *testing.T) {
	p := New(nil).(*Plugin)
	p.cfg.Dir = "/my/project"
	p.chdir = "modules/east"
	p.Activate()

	output := p.View(80, 20)
	if output == "" {
		t.Error("view should not be empty")
	}
	if !strings.Contains(output, "modules/east") {
		t.Error("should show chdir value")
	}
}

func TestConfigure(t *testing.T) {
	p := New(nil).(*Plugin)
	if err := p.Configure(nil); err != nil {
		t.Errorf("Configure() = %v", err)
	}
}

func TestChdirNotSelectable_WhenNoMembers(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.Activate()

	// Enter should trigger workspace fetch (first selectable), not chdir
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected workspace fetch command")
	}

	// Picker not pushed yet — async fetch in progress
	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1", p.stack.Depth())
	}
}
