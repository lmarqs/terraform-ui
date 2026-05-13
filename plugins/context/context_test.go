package context

import (
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
	p.session = sdk.NewSession()
	p.Activate()

	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1", p.stack.Depth())
	}
}

func TestFormNavigation_EnterOnScope_NavigatesToScope(t *testing.T) {
	p := New(nil).(*Plugin)
	p.session = sdk.NewSession()
	p.Activate()

	// First selectable field is Scope (cursor starts there)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from Enter on scope field")
	}

	msg := cmd()
	nav, ok := msg.(NavigateToMsg)
	if !ok {
		t.Fatalf("expected NavigateToMsg, got %T", msg)
	}
	if nav.PluginID != "scope" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "scope")
	}
}

func TestFormNavigation_EnterOnWorkspace_NavigatesToWorkspaces(t *testing.T) {
	p := New(nil).(*Plugin)
	p.session = sdk.NewSession()
	p.Activate()

	// Move down to workspace field
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from Enter on workspace field")
	}

	msg := cmd()
	nav, ok := msg.(NavigateToMsg)
	if !ok {
		t.Fatalf("expected NavigateToMsg, got %T", msg)
	}
	if nav.PluginID != "workspaces" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "workspaces")
	}
}

func TestView_ShowsProjectScopeWorkspace(t *testing.T) {
	p := New(nil).(*Plugin)
	p.session = sdk.NewSession()
	p.cfg.Dir = "/my/project"
	p.session.Set(sdk.SessionKeyActiveChdir, "modules/east")
	p.Activate()

	output := p.View(80, 20)
	if output == "" {
		t.Error("view should not be empty")
	}
}

func TestConfigure(t *testing.T) {
	p := New(nil).(*Plugin)
	if err := p.Configure(nil); err != nil {
		t.Errorf("Configure() = %v", err)
	}
}
