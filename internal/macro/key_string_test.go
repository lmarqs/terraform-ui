package macro

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyToString_roundtrip(t *testing.T) {
	keys := []string{
		"a", "z", "0", "9",
		"/", ":", "!", "~",
		"enter", "esc", "tab", "backspace",
		"up", "down", "left", "right",
		"space",
		"ctrl+c", "ctrl+w", "ctrl+t", "ctrl+s",
	}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			msg := keyToMsg(key)
			got := KeyToString(msg)
			if got != key {
				t.Errorf("KeyToString(keyToMsg(%q)) = %q, want %q", key, got, key)
			}
		})
	}
}

func TestKeyToString_unknown(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyF1}
	got := KeyToString(msg)
	if got != "" {
		t.Errorf("KeyToString(F1) = %q, want empty string", got)
	}
}
