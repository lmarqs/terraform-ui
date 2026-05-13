package chdir

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestPlugin_ID(t *testing.T) {
	p := New(nil)
	if p.ID() != "chdir" {
		t.Errorf("ID() = %q, want chdir", p.ID())
	}
}

func TestPlugin_WhenNoMembers_ShouldBeReadyImmediately(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()
	if !p.Ready() {
		t.Error("Ready() = false, want true when no members")
	}
}

func TestPlugin_WhenMembers_ShouldNotBeReadyUntilSelection(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()
	if p.Ready() {
		t.Error("Ready() = true before selection, want false")
	}
}

func TestPlugin_WhenEnterPressed_ShouldSelectMember(t *testing.T) {
	p := New(nil).(*Plugin)
	session := sdk.NewSession()
	p.session = session
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")

	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !p.Ready() {
		t.Error("Ready() = false after enter, want true")
	}

	chdir, ok := sdk.GetTyped[string](session, sdk.SessionKeyActiveChdir)
	if !ok || chdir != "modules/vpc" {
		t.Errorf("SessionKeyActiveChdir = %q, want modules/vpc", chdir)
	}

	abs, ok := sdk.GetTyped[string](session, sdk.SessionKeyActiveChdirAbs)
	if !ok || abs != "/project/modules/vpc" {
		t.Errorf("SessionKeyActiveChdirAbs = %q, want /project/modules/vpc", abs)
	}
}

func TestPlugin_WhenNavigateDown_ShouldSelectSecondMember(t *testing.T) {
	p := New(nil).(*Plugin)
	session := sdk.NewSession()
	p.session = session
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")

	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	chdir, _ := sdk.GetTyped[string](session, sdk.SessionKeyActiveChdir)
	if chdir != "modules/ecs" {
		t.Errorf("SessionKeyActiveChdir = %q, want modules/ecs", chdir)
	}
}

func TestPlugin_View_WhenNoMembers_ShouldShowMessage(t *testing.T) {
	p := New(nil).(*Plugin)
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() should not be empty when no members")
	}
}

func TestPlugin_View_WhenMembers_ShouldRenderList(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() should not be empty with members")
	}
}

func TestPlugin_Hints_ShouldIncludeEnterAndBack(t *testing.T) {
	p := New(nil).(*Plugin)
	hints := p.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() length = %d, want 2", len(hints))
	}
	if hints[0].Key != "enter" {
		t.Errorf("Hints()[0].Key = %q, want enter", hints[0].Key)
	}
}
