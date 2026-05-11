package components

import (
	"strings"
	"testing"
)

func TestContentBorder_Render_ContainsBorderChars(t *testing.T) {
	cb := NewContentBorder()
	output := cb.Render("hello", "Test", 0, 0, 0, 40, 5)

	if !strings.Contains(output, "┌") {
		t.Error("should contain top-left border")
	}
	if !strings.Contains(output, "┐") {
		t.Error("should contain top-right border")
	}
	if !strings.Contains(output, "└") {
		t.Error("should contain bottom-left border")
	}
	if !strings.Contains(output, "┘") {
		t.Error("should contain bottom-right border")
	}
	if !strings.Contains(output, "│") {
		t.Error("should contain side borders")
	}
}

func TestContentBorder_Render_TitleInTopBorder(t *testing.T) {
	cb := NewContentBorder()
	output := cb.Render("content", "State Browser", 0, 0, 0, 60, 5)

	lines := strings.Split(output, "\n")
	if len(lines) < 1 {
		t.Fatal("expected at least one line")
	}
	if !strings.Contains(lines[0], "State Browser") {
		t.Error("top border should contain title")
	}
}

func TestContentBorder_Render_TitleWithFilteredTotal(t *testing.T) {
	cb := NewContentBorder()
	output := cb.Render("content", "State Browser", 30, 1549, 0, 60, 5)

	if !strings.Contains(output, "(30/1549)") {
		t.Error("should show filtered/total count")
	}
}

func TestContentBorder_Render_TitleWithTotalOnly(t *testing.T) {
	cb := NewContentBorder()
	output := cb.Render("content", "State Browser", 1549, 1549, 0, 60, 5)

	if !strings.Contains(output, "(1549)") {
		t.Error("should show total-only count")
	}
	if strings.Contains(output, "1549/1549") {
		t.Error("should not show redundant filtered/total")
	}
}

func TestContentBorder_Render_NoCountWhenZero(t *testing.T) {
	cb := NewContentBorder()
	output := cb.Render("content", "Home", 0, 0, 0, 60, 5)

	if strings.Contains(output, "(") {
		t.Error("should not show count when total is 0")
	}
}

func TestContentBorder_Render_ContainsContent(t *testing.T) {
	cb := NewContentBorder()
	output := cb.Render("my content here", "Title", 0, 0, 0, 40, 5)

	if !strings.Contains(output, "my content here") {
		t.Error("should contain the content")
	}
}

func TestContentBorder_Render_EmptyTitle(t *testing.T) {
	cb := NewContentBorder()
	output := cb.Render("content", "", 0, 0, 0, 40, 5)

	lines := strings.Split(output, "\n")
	if !strings.Contains(lines[0], "────") {
		t.Error("empty title should produce plain border")
	}
}

func TestFormatBorderTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		filtered int
		total    int
		pinned   int
		want     string
	}{
		{"NoCount", "Home", 0, 0, 0, "Home"},
		{"TotalOnly", "State", 100, 100, 0, "State (100)"},
		{"FilteredTotal", "State", 30, 1549, 0, "State (30/1549)"},
		{"ZeroTotal", "Plan", 0, 0, 0, "Plan"},
		{"WithPinned", "State", 1549, 1549, 5, "State (1549) 📌5"},
		{"FilteredWithPinned", "State", 30, 1549, 3, "State (30/1549) 📌3"},
		{"PinnedZeroNotShown", "State", 100, 100, 0, "State (100)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBorderTitle(tt.title, tt.filtered, tt.total, tt.pinned)
			if got != tt.want {
				t.Errorf("formatBorderTitle(%q, %d, %d, %d) = %q, want %q", tt.title, tt.filtered, tt.total, tt.pinned, got, tt.want)
			}
		})
	}
}

func TestBuildTopBorder(t *testing.T) {
	tests := []struct {
		name  string
		title string
		width int
	}{
		{"EmptyTitle", "", 40},
		{"ShortTitle", "Hi", 40},
		{"LongTitle", "This is a very long title that might overflow", 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTopBorder(tt.title, tt.width)
			if !strings.HasPrefix(got, "┌") {
				t.Error("should start with ┌")
			}
			if !strings.HasSuffix(got, "┐") {
				t.Error("should end with ┐")
			}
		})
	}
}
