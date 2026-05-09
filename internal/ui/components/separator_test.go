package components

import (
	"strings"
	"testing"
)

func TestNewSeparator(t *testing.T) {
	s := NewSeparator()
	_ = s
}

func TestSeparator_Render_CorrectWidth(t *testing.T) {
	s := NewSeparator()

	output := s.Render(40)
	if !strings.Contains(output, strings.Repeat("═", 40)) {
		t.Error("Render(40) should contain 40 ═ characters")
	}
}

func TestSeparator_Render_ContainsDoubleLineChars(t *testing.T) {
	s := NewSeparator()

	output := s.Render(10)
	if !strings.Contains(output, "═") {
		t.Error("Render() should contain ═ characters")
	}
}

func TestSeparator_Render_NonEmpty(t *testing.T) {
	s := NewSeparator()

	output := s.Render(20)
	if output == "" {
		t.Error("Render(20) should return non-empty string")
	}
}

func TestSeparator_Render_VariousWidths(t *testing.T) {
	s := NewSeparator()

	widths := []int{1, 10, 40, 80, 120}
	for _, w := range widths {
		output := s.Render(w)
		if output == "" {
			t.Errorf("Render(%d) returned empty string", w)
		}
	}
}
