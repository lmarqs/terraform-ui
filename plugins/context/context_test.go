package context

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

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

func TestFormNavigation_EnterOnChdir_EmitsNavigateMsg(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()

	// First selectable field is Chdir
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from chdir selection")
	}

	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.PluginID != "chdir" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "chdir")
	}
}

func TestFormNavigation_EnterOnWorkspace_EmitsNavigateMsg(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	// No members configured, so first selectable is Workspace
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from workspace selection")
	}

	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.PluginID != "workspaces" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "workspaces")
	}
}

func TestFormNavigation_EscPopsForm(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.stack.Depth() != 0 {
		t.Errorf("stack depth = %d, want 0 after esc", p.stack.Depth())
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
	p := New(nil).(*Plugin)
	p.Activate()

	// Enter should trigger workspace navigate (first selectable), not chdir
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected workspace navigate command")
	}

	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.PluginID != "workspaces" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "workspaces")
	}
}

func TestHandleChdirChanged(t *testing.T) {
	p := New(nil).(*Plugin)
	p.HandleChdirChanged(sdk.ChdirChangedEvent{RelPath: "modules/vpc"})
	if p.chdir != "modules/vpc" {
		t.Errorf("chdir = %q, want %q", p.chdir, "modules/vpc")
	}
}

func TestHandleWorkspaceChanged(t *testing.T) {
	p := New(nil).(*Plugin)
	p.HandleWorkspaceChanged(sdk.WorkspaceChangedEvent{Name: "staging"})
	if p.workspace != "staging" {
		t.Errorf("workspace = %q, want %q", p.workspace, "staging")
	}
}
