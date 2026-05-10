package components

import (
	"strings"
	"testing"
)

func TestCommandBar_Render_ContainsBorders(t *testing.T) {
	cb := NewCommandBar()
	output := cb.Render("context", nil, 60)

	if !strings.Contains(output, "┌") {
		t.Error("should contain top border")
	}
	if !strings.Contains(output, "┐") {
		t.Error("should contain top-right border")
	}
	if !strings.Contains(output, "└") {
		t.Error("should contain bottom border")
	}
	if !strings.Contains(output, "┘") {
		t.Error("should contain bottom-right border")
	}
}

func TestCommandBar_Render_ContainsInput(t *testing.T) {
	cb := NewCommandBar()
	output := cb.Render("plan", nil, 60)

	if !strings.Contains(output, ":plan") {
		t.Error("should contain ':plan' input")
	}
	if !strings.Contains(output, "█") {
		t.Error("should contain cursor")
	}
}

func TestCommandBar_Render_IsThreeLines(t *testing.T) {
	cb := NewCommandBar()
	output := cb.Render("test", nil, 60)

	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestCommandBar_Render_WithMatches(t *testing.T) {
	cb := NewCommandBar()
	output := cb.Render("st", []string{"state", "status"}, 80)

	if !strings.Contains(output, "state") {
		t.Error("should contain match 'state'")
	}
	if !strings.Contains(output, "status") {
		t.Error("should contain match 'status'")
	}
	if !strings.Contains(output, "|") {
		t.Error("matches should be separated by |")
	}
}

func TestCommandBar_Render_EmptyInput(t *testing.T) {
	cb := NewCommandBar()
	output := cb.Render("", nil, 60)

	if !strings.Contains(output, ":█") {
		t.Error("empty input should show ':' with cursor")
	}
}

func TestCommandBar_Render_VariousWidths(t *testing.T) {
	cb := NewCommandBar()
	widths := []int{20, 40, 80, 120}
	for _, w := range widths {
		output := cb.Render("test", []string{"testing"}, w)
		if output == "" {
			t.Errorf("Render with width %d returned empty", w)
		}
	}
}
