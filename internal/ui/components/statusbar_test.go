package components

import (
	"strings"
	"testing"
)

func TestNewStatusBar(t *testing.T) {
	sb := NewStatusBar()
	// StatusBar is a zero-value struct, just verify it doesn't panic
	_ = sb
}

func TestStatusBar_Render_ReturnsNonEmpty(t *testing.T) {
	sb := NewStatusBar()

	output := sb.Render(80)
	if output == "" {
		t.Fatal("Render(80) returned empty string")
	}
}

func TestStatusBar_Render_ContainsKeyBindings(t *testing.T) {
	sb := NewStatusBar()

	output := sb.Render(120)

	expectedBindings := []string{"q", "esc", "?", "/"}
	for _, key := range expectedBindings {
		if !strings.Contains(output, key) {
			t.Errorf("Render() should contain key binding %q", key)
		}
	}
}

func TestStatusBar_Render_ContainsLabels(t *testing.T) {
	sb := NewStatusBar()

	output := sb.Render(120)

	expectedLabels := []string{"quit", "back", "wrap", "help", "search", "navigate"}
	for _, label := range expectedLabels {
		if !strings.Contains(output, label) {
			t.Errorf("Render() should contain label %q", label)
		}
	}
}

func TestStatusBar_Render_VariousWidths(t *testing.T) {
	sb := NewStatusBar()

	widths := []int{20, 40, 80, 120, 200}
	for _, w := range widths {
		output := sb.Render(w)
		if output == "" {
			t.Errorf("Render(%d) returned empty string", w)
		}
	}
}
