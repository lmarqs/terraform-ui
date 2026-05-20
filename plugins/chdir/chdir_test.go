package chdir

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "chdir" {
		t.Errorf("ID() = %q, want %q", p.ID(), "chdir")
	}
	if p.Name() != "Chdir" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Chdir")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(map[string]interface{}{}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	if cmd := p.Init(&sdk.Context{WorkingDir: "/tmp", Workspace: "default", Service: svc}); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
	if p.Ready() {
		t.Error("Ready() should be false before activation")
	}
}

func TestPlugin_WhenUpdated_ShouldReturnSelfAndNil(t *testing.T) {
	p := New(nil)
	result, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if result != p {
		t.Error("Update() returned different plugin reference")
	}
	if cmd != nil {
		t.Error("Update() returned non-nil cmd, want nil")
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

func TestPlugin_WhenNavigateWithJK_ShouldMoveCursor(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs", "modules/rds"}, "/project")
	p.Activate()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.cursor.Pos() != 1 {
		t.Errorf("after j: cursor.Pos() = %d, want 1", p.cursor.Pos())
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.cursor.Pos() != 2 {
		t.Errorf("after second j: cursor.Pos() = %d, want 2", p.cursor.Pos())
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.cursor.Pos() != 1 {
		t.Errorf("after k: cursor.Pos() = %d, want 1", p.cursor.Pos())
	}
}

func TestPlugin_WhenNavigateWithUpKey_ShouldMoveCursorUp(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"}, "/project")
	p.Activate()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.cursor.Pos() != 0 {
		t.Errorf("after up: cursor.Pos() = %d, want 0", p.cursor.Pos())
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

func TestPlugin_WhenNonKeyMsg_ShouldReturnFrameUnchanged(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc"}, "/project")
	p.Activate()

	type customMsg struct{}
	cmd := p.stack.Update(customMsg{})
	if cmd != nil {
		t.Error("Update(non-key msg) returned non-nil cmd, want nil")
	}
	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1 (frame unchanged)", p.stack.Depth())
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

func TestPlugin_View_WhenMembersEmpty_ShouldRenderEmptyMessage(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{}, "/project")
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	view := p.View(80, 24)
	if view == "" {
		t.Error("View() should show empty message when members is empty but frame is present")
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

func TestListFrame_WhenCreated_ShouldHaveCorrectID(t *testing.T) {
	p := New(nil).(*Plugin)
	f := &listFrame{plugin: p}
	if f.ID() != "list" {
		t.Errorf("ID() = %q, want list", f.ID())
	}
}

func TestSelectMember_WhenCursorBeyondMembers_ShouldReturnNil(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc"}, "/project")
	p.cursor.SetCount(1)
	// Manually force cursor beyond bounds by setting count to a higher value then back
	p.cursor.SetCount(5)
	// Move cursor to position 4
	p.cursor.MoveDown()
	p.cursor.MoveDown()
	p.cursor.MoveDown()
	p.cursor.MoveDown()
	// Now set members to fewer than cursor position
	p.members = []string{"modules/vpc"}

	cmd := p.selectMember()
	if cmd != nil {
		t.Error("selectMember() with cursor beyond members should return nil")
	}
}

func TestSelectMember_WhenNoMembers_ShouldReturnNil(t *testing.T) {
	p := New(nil).(*Plugin)
	p.members = nil
	cmd := p.selectMember()
	if cmd != nil {
		t.Error("selectMember() with no members should return nil")
	}
}

func TestPlugin_WhenHandleChdirChanged_ShouldMarkSelected(t *testing.T) {
	p := New(nil).(*Plugin)
	p.selected = false

	cmd := p.HandleChdirChanged(sdk.ChdirChangedEvent{RelPath: "mod/a", AbsPath: "/x"})
	if cmd != nil {
		t.Error("HandleChdirChanged() should return nil")
	}
	if !p.selected {
		t.Error("selected should be true after HandleChdirChanged")
	}
}
