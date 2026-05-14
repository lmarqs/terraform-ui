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

func TestPlugin_WhenEnterPressed_ShouldPublishChdirChangedEvent(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !p.Ready() {
		t.Error("Ready() = false after enter, want true")
	}
	if cmd == nil {
		t.Fatal("expected ChdirChangedEvent cmd")
	}

	msg := cmd()
	evt, ok := msg.(sdk.ChdirChangedEvent)
	if !ok {
		t.Fatalf("cmd() returned %T, want sdk.ChdirChangedEvent", msg)
	}
	if evt.RelPath != "modules/vpc" {
		t.Errorf("event.RelPath = %q, want modules/vpc", evt.RelPath)
	}
	if evt.AbsPath != "/project/modules/vpc" {
		t.Errorf("event.AbsPath = %q, want /project/modules/vpc", evt.AbsPath)
	}
	if evt.Count != 2 {
		t.Errorf("event.Count = %d, want 2", evt.Count)
	}
}

func TestPlugin_WhenNavigateDown_ShouldSelectSecondMember(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("expected ChdirChangedEvent cmd")
	}
	msg := cmd()
	evt, ok := msg.(sdk.ChdirChangedEvent)
	if !ok {
		t.Fatalf("cmd() returned %T, want sdk.ChdirChangedEvent", msg)
	}
	if evt.RelPath != "modules/ecs" {
		t.Errorf("event.RelPath = %q, want modules/ecs", evt.RelPath)
	}
}

func TestPlugin_EscPopsFrame(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc"}, "/project")
	p.Activate()

	if p.stack.Depth() != 1 {
		t.Fatalf("stack depth = %d, want 1", p.stack.Depth())
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if p.stack.Depth() != 0 {
		t.Errorf("stack depth = %d, want 0 after esc", p.stack.Depth())
	}
}

func TestPlugin_View_WhenNoMembers_ShouldShowMessage(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()
	view := p.View(80, 24)
	if view != "" {
		t.Error("View() should be empty when no members (stack is empty)")
	}
}

func TestPlugin_View_WhenMembers_ShouldRenderList(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() should not be empty with members")
	}
}

func TestPlugin_Hints_ShouldIncludeEnterAndBack(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc"}, "/project")
	p.Activate()

	hints := p.stack.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() length = %d, want 2", len(hints))
	}
	if hints[0].Key != "enter" {
		t.Errorf("Hints()[0].Key = %q, want enter", hints[0].Key)
	}
	if hints[1].Key != "esc" {
		t.Errorf("Hints()[1].Key = %q, want esc", hints[1].Key)
	}
}

func TestPlugin_Stackable(t *testing.T) {
	p := New(nil).(*Plugin)
	if p.Stack() == nil {
		t.Error("Stack() should not be nil")
	}
}
