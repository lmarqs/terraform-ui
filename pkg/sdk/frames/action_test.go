package frames

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Verify ActionFrame satisfies the Frame interface at compile time.
var _ sdk.Frame = (*ActionFrame)(nil)

func TestActionFrame_ID(t *testing.T) {
	f := NewActionFrame("Actions", nil)
	if f.ID() != "actions" {
		t.Fatalf("expected ID 'actions', got %q", f.ID())
	}
}

func TestActionFrame_KeyExecutesHandlerAndPops(t *testing.T) {
	called := false
	f := NewActionFrame("Actions", []Action{
		{Key: "d", Label: "delete", Handler: func() tea.Cmd {
			called = true
			return nil
		}},
	})

	result, _ := f.Update(keyMsg("d"))
	if result != nil {
		t.Fatal("matching key should pop frame (return nil)")
	}
	if !called {
		t.Fatal("handler should be called")
	}
}

func TestActionFrame_KeyWithNilHandlerPops(t *testing.T) {
	f := NewActionFrame("Actions", []Action{
		{Key: "d", Label: "delete", Handler: nil},
	})

	result, _ := f.Update(keyMsg("d"))
	if result != nil {
		t.Fatal("matching key with nil handler should pop frame")
	}
}

func TestActionFrame_KeyReturnsHandlerCmd(t *testing.T) {
	type testMsg struct{}
	f := NewActionFrame("Actions", []Action{
		{Key: "r", Label: "refresh", Handler: func() tea.Cmd {
			return func() tea.Msg { return testMsg{} }
		}},
	})

	_, cmd := f.Update(keyMsg("r"))
	if cmd == nil {
		t.Fatal("handler cmd should be returned")
	}
	msg := cmd()
	if _, ok := msg.(testMsg); !ok {
		t.Fatal("expected testMsg from cmd")
	}
}

func TestActionFrame_DisabledKeyDoesNothing(t *testing.T) {
	called := false
	f := NewActionFrame("Actions", []Action{
		{Key: "d", Label: "delete", Handler: func() tea.Cmd {
			called = true
			return nil
		}, Disabled: true},
	})

	result, cmd := f.Update(keyMsg("d"))
	if result == nil {
		t.Fatal("disabled key should not pop frame")
	}
	if result != f {
		t.Fatal("disabled key should return same frame")
	}
	if cmd != nil {
		t.Fatal("disabled key should not produce a cmd")
	}
	if called {
		t.Fatal("disabled handler should not be called")
	}
}

func TestActionFrame_EscDismisses(t *testing.T) {
	called := false
	f := NewActionFrame("Actions", []Action{
		{Key: "d", Label: "delete", Handler: func() tea.Cmd {
			called = true
			return nil
		}},
	})

	result, cmd := f.Update(keyMsg("esc"))
	if result != nil {
		t.Fatal("esc should pop frame (return nil)")
	}
	if cmd != nil {
		t.Fatal("esc should not produce a cmd")
	}
	if called {
		t.Fatal("esc should not call any handler")
	}
}

func TestActionFrame_UnrecognizedKeysConsumed(t *testing.T) {
	f := NewActionFrame("Actions", []Action{
		{Key: "d", Label: "delete", Handler: func() tea.Cmd { return nil }},
	})

	otherKeys := []string{"a", "b", "x", "q", "enter", " ", "r"}
	for _, key := range otherKeys {
		result, cmd := f.Update(keyMsg(key))
		if result == nil {
			t.Fatalf("key %q should be consumed, not cause pop", key)
		}
		if result != f {
			t.Fatalf("key %q should return same frame", key)
		}
		if cmd != nil {
			t.Fatalf("key %q should not produce a cmd", key)
		}
	}
}

func TestActionFrame_NonKeyMsgIgnored(t *testing.T) {
	f := NewActionFrame("Actions", []Action{
		{Key: "d", Label: "delete", Handler: func() tea.Cmd { return nil }},
	})

	result, cmd := f.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if result != f {
		t.Fatal("non-key msg should return same frame")
	}
	if cmd != nil {
		t.Fatal("non-key msg should not produce a cmd")
	}
}

func TestActionFrame_Hints(t *testing.T) {
	f := NewActionFrame("Actions", []Action{
		{Key: "d", Label: "delete", Handler: func() tea.Cmd { return nil }},
		{Key: "m", Label: "move", Handler: func() tea.Cmd { return nil }},
		{Key: "x", Label: "disabled", Handler: func() tea.Cmd { return nil }, Disabled: true},
	})

	hints := f.Hints()
	// Should have 2 enabled actions + 1 Esc = 3
	if len(hints) != 3 {
		t.Fatalf("expected 3 hints, got %d", len(hints))
	}
	if hints[0].Key != "d" || hints[0].Description != "delete" {
		t.Fatalf("unexpected first hint: %v", hints[0])
	}
	if hints[1].Key != "m" || hints[1].Description != "move" {
		t.Fatalf("unexpected second hint: %v", hints[1])
	}
	if hints[2].Key != "Esc" || hints[2].Description != "cancel" {
		t.Fatalf("unexpected last hint: %v", hints[2])
	}
}

func TestActionFrame_HintsEmpty(t *testing.T) {
	f := NewActionFrame("Actions", nil)

	hints := f.Hints()
	if len(hints) != 1 {
		t.Fatalf("expected 1 hint (Esc), got %d", len(hints))
	}
	if hints[0].Key != "Esc" {
		t.Fatalf("expected Esc hint, got %v", hints[0])
	}
}

func TestActionFrame_View(t *testing.T) {
	f := NewActionFrame("Actions", []Action{
		{Key: "d", Label: "delete", Handler: func() tea.Cmd { return nil }},
		{Key: "m", Label: "move", Handler: func() tea.Cmd { return nil }, Disabled: true},
	})

	view := f.View(80, 24)
	if view == "" {
		t.Fatal("view should not be empty")
	}
}
