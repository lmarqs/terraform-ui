package sdk

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type mockFrame struct {
	id        string
	updated   bool
	returnNil bool
	lastMsg   tea.Msg
}

func (f *mockFrame) ID() string { return f.id }
func (f *mockFrame) Update(msg tea.Msg) (Frame, tea.Cmd) {
	f.updated = true
	f.lastMsg = msg
	if f.returnNil {
		return nil, nil
	}
	return f, nil
}
func (f *mockFrame) View(width, height int) string { return f.id + "-view" }
func (f *mockFrame) Hints() []KeyHint {
	return []KeyHint{{Key: "x", Description: f.id}}
}

func TestStack_PushPop(t *testing.T) {
	s := NewStack()

	if !s.IsEmpty() {
		t.Fatal("new stack should be empty")
	}
	if s.Depth() != 0 {
		t.Fatalf("expected depth 0, got %d", s.Depth())
	}

	f1 := &mockFrame{id: "one"}
	f2 := &mockFrame{id: "two"}

	s.Push(f1)
	s.Push(f2)

	if s.Depth() != 2 {
		t.Fatalf("expected depth 2, got %d", s.Depth())
	}
	if s.Peek() != f2 {
		t.Fatal("peek should return top frame")
	}

	popped := s.Pop()
	if popped != f2 {
		t.Fatal("pop should return top frame")
	}
	if s.Depth() != 1 {
		t.Fatalf("expected depth 1 after pop, got %d", s.Depth())
	}
	if s.Peek() != f1 {
		t.Fatal("peek after pop should return previous frame")
	}
}

func TestStack_PopEmpty(t *testing.T) {
	s := NewStack()
	if s.Pop() != nil {
		t.Fatal("pop on empty stack should return nil")
	}
}

func TestStack_PeekEmpty(t *testing.T) {
	s := NewStack()
	if s.Peek() != nil {
		t.Fatal("peek on empty stack should return nil")
	}
}

func TestStack_UpdateRoutesToTop(t *testing.T) {
	s := NewStack()
	f1 := &mockFrame{id: "bottom"}
	f2 := &mockFrame{id: "top"}
	s.Push(f1)
	s.Push(f2)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	s.Update(msg)

	if !f2.updated {
		t.Fatal("top frame should receive the message")
	}
	if f1.updated {
		t.Fatal("bottom frame should NOT receive the message")
	}
}

func TestStack_UpdatePopOnNil(t *testing.T) {
	s := NewStack()
	f1 := &mockFrame{id: "bottom"}
	f2 := &mockFrame{id: "top", returnNil: true}
	s.Push(f1)
	s.Push(f2)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	s.Update(msg)

	if s.Depth() != 1 {
		t.Fatalf("expected depth 1 after pop, got %d", s.Depth())
	}
	if s.Peek() != f1 {
		t.Fatal("bottom frame should be on top after pop")
	}
}

func TestStack_UpdateEmptyIsNoop(t *testing.T) {
	s := NewStack()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	cmd := s.Update(msg)
	if cmd != nil {
		t.Fatal("update on empty stack should return nil cmd")
	}
}

func TestStack_ViewReturnsTopFrame(t *testing.T) {
	s := NewStack()
	if s.View(80, 24) != "" {
		t.Fatal("view on empty stack should return empty string")
	}

	s.Push(&mockFrame{id: "bottom"})
	s.Push(&mockFrame{id: "top"})

	if s.View(80, 24) != "top-view" {
		t.Fatalf("expected top-view, got %q", s.View(80, 24))
	}
}

func TestStack_HintsReturnsTopFrame(t *testing.T) {
	s := NewStack()
	if s.Hints() != nil {
		t.Fatal("hints on empty stack should return nil")
	}

	s.Push(&mockFrame{id: "bottom"})
	s.Push(&mockFrame{id: "top"})

	hints := s.Hints()
	if len(hints) != 1 || hints[0].Description != "top" {
		t.Fatalf("expected top frame hints, got %v", hints)
	}
}

func TestStack_Clear(t *testing.T) {
	s := NewStack()
	s.Push(&mockFrame{id: "root"})
	s.Push(&mockFrame{id: "mid"})
	s.Push(&mockFrame{id: "top"})

	s.Clear()
	if s.Depth() != 1 {
		t.Fatalf("expected depth 1 after clear, got %d", s.Depth())
	}
	if s.Peek().ID() != "root" {
		t.Fatal("clear should preserve root frame")
	}
}

func TestStack_ClearSingleFrame(t *testing.T) {
	s := NewStack()
	s.Push(&mockFrame{id: "root"})
	s.Clear()
	if s.Depth() != 1 {
		t.Fatalf("clear on single frame should be no-op, got depth %d", s.Depth())
	}
}

func TestStack_ClearEmpty(t *testing.T) {
	s := NewStack()
	s.Clear()
	if s.Depth() != 0 {
		t.Fatal("clear on empty stack should be no-op")
	}
}

func TestStack_Reset(t *testing.T) {
	s := NewStack()
	s.Push(&mockFrame{id: "root"})
	s.Push(&mockFrame{id: "top"})
	s.Reset()
	if !s.IsEmpty() {
		t.Fatal("reset should empty the stack")
	}
}

type replacingFrame struct {
	id          string
	replacement Frame
}

func (f *replacingFrame) ID() string { return f.id }
func (f *replacingFrame) Update(msg tea.Msg) (Frame, tea.Cmd) {
	return f.replacement, nil
}
func (f *replacingFrame) View(width, height int) string { return f.id }
func (f *replacingFrame) Hints() []KeyHint              { return nil }

func TestStack_UpdateReplacesFrame(t *testing.T) {
	s := NewStack()
	newFrame := &mockFrame{id: "new"}
	s.Push(&replacingFrame{id: "old", replacement: newFrame})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	s.Update(msg)

	if s.Peek() != newFrame {
		t.Fatal("frame should be replaced in-place")
	}
	if s.Depth() != 1 {
		t.Fatalf("depth should remain 1, got %d", s.Depth())
	}
}
