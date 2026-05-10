package frames

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func keyMsg(key string) tea.Msg {
	if len(key) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	switch key {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func TestFilterFrame_AppendsPrintableKeys(t *testing.T) {
	var lastFilter string
	f := NewFilterFrame(FilterOpts{
		OnFilter: func(q string) { lastFilter = q },
	})

	tests := []struct {
		key      string
		expected string
	}{
		{"a", "a"},
		{"u", "au"},
		{"r", "aur"},
		{"o", "auro"},
		{"r", "auror"},
		{"a", "aurora"},
	}

	for _, tt := range tests {
		result, _ := f.Update(keyMsg(tt.key))
		if result == nil {
			t.Fatalf("frame should not pop on key %q", tt.key)
		}
		if f.Query != tt.expected {
			t.Fatalf("after %q: expected query %q, got %q", tt.key, tt.expected, f.Query)
		}
		if lastFilter != tt.expected {
			t.Fatalf("OnFilter not called with %q", tt.expected)
		}
	}
}

func TestFilterFrame_KeybindingKeysAreTextInput(t *testing.T) {
	var lastFilter string
	f := NewFilterFrame(FilterOpts{
		OnFilter: func(q string) { lastFilter = q },
	})

	// These are keybindings in normal mode but should be text in filter mode
	keybindingKeys := []string{"i", "d", "e", "r", "s", "q", "w", "u", "g", "G"}
	for _, key := range keybindingKeys {
		result, _ := f.Update(keyMsg(key))
		if result == nil {
			t.Fatalf("key %q should be text input, not cause pop", key)
		}
	}

	expected := "idersqwugG"
	if f.Query != expected {
		t.Fatalf("expected query %q, got %q", expected, f.Query)
	}
	if lastFilter != expected {
		t.Fatalf("OnFilter should track all keys, got %q", lastFilter)
	}
}

func TestFilterFrame_EscPops(t *testing.T) {
	f := NewFilterFrame(FilterOpts{})
	f.Query = "test"

	result, _ := f.Update(keyMsg("esc"))
	if result != nil {
		t.Fatal("esc should pop the filter frame (return nil)")
	}
}

func TestFilterFrame_EnterCallsOnSelect(t *testing.T) {
	called := false
	f := NewFilterFrame(FilterOpts{
		OnSelect: func() tea.Cmd {
			called = true
			return nil
		},
	})

	result, _ := f.Update(keyMsg("enter"))
	if result == nil {
		t.Fatal("enter should not pop the frame")
	}
	if !called {
		t.Fatal("enter should call OnSelect")
	}
}

func TestFilterFrame_Backspace(t *testing.T) {
	var lastFilter string
	f := NewFilterFrame(FilterOpts{
		OnFilter: func(q string) { lastFilter = q },
	})
	f.Query = "abc"

	result, _ := f.Update(keyMsg("backspace"))
	if result == nil {
		t.Fatal("backspace should not pop")
	}
	if f.Query != "ab" {
		t.Fatalf("expected query %q, got %q", "ab", f.Query)
	}
	if lastFilter != "ab" {
		t.Fatal("OnFilter should be called on backspace")
	}
}

func TestFilterFrame_BackspaceOnEmpty(t *testing.T) {
	f := NewFilterFrame(FilterOpts{})
	f.Query = ""

	result, _ := f.Update(keyMsg("backspace"))
	if result == nil {
		t.Fatal("backspace on empty should not pop")
	}
	if f.Query != "" {
		t.Fatal("query should stay empty")
	}
}

func TestFilterFrame_Navigation(t *testing.T) {
	navDir := 0
	f := NewFilterFrame(FilterOpts{
		OnNavigate: func(dir int) { navDir = dir },
	})

	f.Update(keyMsg("down"))
	if navDir != 1 {
		t.Fatalf("down should navigate +1, got %d", navDir)
	}

	f.Update(keyMsg("up"))
	if navDir != -1 {
		t.Fatalf("up should navigate -1, got %d", navDir)
	}
}

func TestFilterFrame_SpaceIsTextInput(t *testing.T) {
	var lastFilter string
	f := NewFilterFrame(FilterOpts{
		OnFilter: func(q string) { lastFilter = q },
	})
	f.Query = "foo"

	result, _ := f.Update(keyMsg(" "))
	if result == nil {
		t.Fatal("space should be text input, not pop")
	}
	if f.Query != "foo " {
		t.Fatalf("expected %q, got %q", "foo ", f.Query)
	}
	if lastFilter != "foo " {
		t.Fatal("OnFilter should be called with space")
	}
}

func TestFilterFrame_Hints(t *testing.T) {
	f := NewFilterFrame(FilterOpts{})
	hints := f.Hints()
	if len(hints) != 2 {
		t.Fatalf("expected 2 hints, got %d", len(hints))
	}
	found := map[string]bool{}
	for _, h := range hints {
		found[h.Key] = true
	}
	if !found["Esc"] || !found["Enter"] {
		t.Fatalf("expected Esc and Enter hints, got %v", hints)
	}
}

func TestFilterFrame_ID(t *testing.T) {
	f := NewFilterFrame(FilterOpts{})
	if f.ID() != "filter" {
		t.Fatalf("expected ID 'filter', got %q", f.ID())
	}
}

func TestFilterFrame_View(t *testing.T) {
	f := NewFilterFrame(FilterOpts{})
	f.Query = "test"
	view := f.View(80, 24)
	if view != "/ test█" {
		t.Fatalf("expected '/ test█', got %q", view)
	}
}

// Verify FilterFrame satisfies the Frame interface at compile time.
var _ sdk.Frame = (*FilterFrame)(nil)
