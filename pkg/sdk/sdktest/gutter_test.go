package sdktest

import (
	"slices"
	"testing"
)

func TestGutterColumn_ShouldReturnTheGutterGlyphEndingEachLine(t *testing.T) {
	tests := []struct {
		name string
		view string
		want []string
	}{
		{"every glyph in order", "a▲\nb┃\nc│\nd▼", []string{"▲", "┃", "│", "▼"}},
		{"skips empty lines and lines without a glyph", "a▲\n\nplain\nb▼", []string{"▲", "▼"}},
		{"ignores a glyph that does not end the line", "▲ rest", nil},
		{"no gutter", "one\ntwo", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GutterColumn(tt.view); !slices.Equal(got, tt.want) {
				t.Errorf("GutterColumn(%q) = %v, want %v", tt.view, got, tt.want)
			}
		})
	}
}
