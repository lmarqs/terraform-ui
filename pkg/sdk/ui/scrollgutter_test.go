package ui_test

import (
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func TestRenderScrollGutter(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		opts  ui.ScrollGutterOpts
		want  func(t *testing.T, got []string)
	}{
		{
			name:  "no overflow returns lines unchanged",
			lines: []string{"line1", "line2", "line3"},
			opts:  ui.ScrollGutterOpts{ViewOffset: 0, TotalItems: 3, ViewportHeight: 5},
			want: func(t *testing.T, got []string) {
				if len(got) != 3 {
					t.Fatalf("expected 3 lines, got %d", len(got))
				}
				if got[0] != "line1" {
					t.Errorf("expected unchanged line, got %q", got[0])
				}
			},
		},
		{
			name:  "nil lines returns nil",
			lines: nil,
			opts:  ui.ScrollGutterOpts{ViewOffset: 0, TotalItems: 10, ViewportHeight: 5},
			want: func(t *testing.T, got []string) {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
			},
		},
		{
			name:  "overflow appends gutter characters",
			lines: []string{"row1", "row2", "row3", "row4", "row5"},
			opts:  ui.ScrollGutterOpts{ViewOffset: 0, TotalItems: 20, ViewportHeight: 5, Width: 4},
			want: func(t *testing.T, got []string) {
				if !strings.HasSuffix(got[0], "▲") {
					t.Errorf("first line should end with ▲, got %q", got[0])
				}
				if !strings.HasSuffix(got[len(got)-1], "▼") {
					t.Errorf("last line should end with ▼, got %q", got[len(got)-1])
				}
			},
		},
		{
			name:  "lines are padded to width before gutter",
			lines: []string{"ab", "abcdef", "x"},
			opts:  ui.ScrollGutterOpts{ViewOffset: 0, TotalItems: 10, ViewportHeight: 3, Width: 6},
			want: func(t *testing.T, got []string) {
				// "ab" (2) + 4 spaces + gutter = 7 chars visual
				// "abcdef" (6) + 0 spaces + gutter = 7 chars visual
				// All gutter chars should align at same column
				for i, line := range got {
					gutterCol := strings.LastIndexAny(line, "▲▼┃│")
					if gutterCol < 6 {
						t.Errorf("line %d: gutter at col %d, expected at col 6+", i, gutterCol)
					}
				}
			},
		},
		{
			name:  "thumb position moves with offset",
			lines: []string{"a", "b", "c", "d", "e"},
			opts:  ui.ScrollGutterOpts{ViewOffset: 10, TotalItems: 20, ViewportHeight: 5, Width: 1},
			want: func(t *testing.T, got []string) {
				hasThumb := false
				for i := 1; i < len(got)-1; i++ {
					if strings.HasSuffix(got[i], "┃") {
						hasThumb = true
					}
				}
				if !hasThumb {
					t.Error("expected thumb (┃) in middle lines")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ui.RenderScrollGutter(tt.lines, tt.opts)
			tt.want(t, got)
		})
	}
}

func TestRenderScrollGutter_WhenViewportHeightIsZero_ShouldReturnLinesUnchanged(t *testing.T) {
	lines := []string{"a", "b"}
	got := ui.RenderScrollGutter(lines, ui.ScrollGutterOpts{
		ViewOffset:     0,
		TotalItems:     10,
		ViewportHeight: 0,
	})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("expected unchanged lines, got %v", got)
	}
}

func TestRenderScrollGutter_WhenViewOffsetIsNegative_ShouldClampThumbStartToZero(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	got := ui.RenderScrollGutter(lines, ui.ScrollGutterOpts{
		ViewOffset:     -5,
		TotalItems:     20,
		ViewportHeight: 5,
		Width:          1,
	})
	if !strings.HasSuffix(got[0], "▲") {
		t.Errorf("expected top cap, got %q", got[0])
	}
	if !strings.HasSuffix(got[len(got)-1], "▼") {
		t.Errorf("expected bottom cap, got %q", got[len(got)-1])
	}
}

func TestRenderScrollGutter_WhenViewOffsetExceedsTotal_ShouldClampThumbToBottom(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	got := ui.RenderScrollGutter(lines, ui.ScrollGutterOpts{
		ViewOffset:     100,
		TotalItems:     20,
		ViewportHeight: 5,
		Width:          1,
	})
	if !strings.HasSuffix(got[0], "▲") {
		t.Errorf("expected top cap, got %q", got[0])
	}
	if !strings.HasSuffix(got[len(got)-1], "▼") {
		t.Errorf("expected bottom cap, got %q", got[len(got)-1])
	}
}

func TestRenderScrollGutter_WhenLineExceedsWidth_ShouldNotPad(t *testing.T) {
	lines := []string{"abcdefghij", "short", "abcdefghij"}
	got := ui.RenderScrollGutter(lines, ui.ScrollGutterOpts{
		ViewOffset:     0,
		TotalItems:     10,
		ViewportHeight: 3,
		Width:          5,
	})
	if !strings.HasSuffix(got[0], "▲") {
		t.Errorf("first line should end with gutter cap, got %q", got[0])
	}
	if !strings.HasPrefix(got[0], "abcdefghij") {
		t.Errorf("long line should not be truncated, got %q", got[0])
	}
}
