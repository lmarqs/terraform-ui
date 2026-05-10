package components

import (
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
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

func TestStatusBar_WithBinaryName_Render(t *testing.T) {
	sb := NewStatusBar().WithBinaryName("terraform")
	output := sb.Render(120)
	if !strings.Contains(output, "terraform") {
		t.Error("Render() should contain the binary name")
	}
}

func TestStatusBar_WithBinaryName_RenderHints(t *testing.T) {
	sb := NewStatusBar().WithBinaryName("tofu")
	hints := []sdk.KeyHint{
		{Key: "q", Description: "quit"},
		{Key: "?", Description: "help"},
	}
	output := sb.RenderHints(hints, 120)
	if !strings.Contains(output, "tofu") {
		t.Error("RenderHints() with binary name should contain 'tofu'")
	}
}

func TestStatusBar_WithBinaryName_Empty(t *testing.T) {
	sb := NewStatusBar().WithBinaryName("")
	output := sb.Render(80)
	// Should still render normally without binary name
	if output == "" {
		t.Error("Render() should not be empty even with no binary name")
	}
}

func TestStatusBar_WithBinaryName_NarrowWidth(t *testing.T) {
	sb := NewStatusBar().WithBinaryName("terraform")
	// With a very narrow width, binary name should be omitted (not enough gap)
	output := sb.Render(20)
	if output == "" {
		t.Error("Render() should not be empty even with narrow width")
	}
}
