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

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !p.Ready() {
		t.Error("Ready() = false after enter, want true")
	}
	if cmd == nil {
		t.Fatal("Update returned nil cmd, want ChdirChangedEvent cmd")
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

	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Update returned nil cmd, want ChdirChangedEvent cmd")
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
