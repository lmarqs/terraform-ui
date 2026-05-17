package ui_test

import (
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
			name:  "exact fit returns lines unchanged",
			lines: []string{"a", "b", "c"},
			opts:  ui.ScrollGutterOpts{ViewOffset: 0, TotalItems: 3, ViewportHeight: 3},
			want: func(t *testing.T, got []string) {
				if got[0] != "a" {
					t.Errorf("expected unchanged, got %q", got[0])
				}
			},
		},
		{
			name:  "overflow at top shows arrow cap and thumb",
			lines: []string{"row1", "row2", "row3", "row4", "row5"},
			opts:  ui.ScrollGutterOpts{ViewOffset: 0, TotalItems: 20, ViewportHeight: 5},
			want: func(t *testing.T, got []string) {
				if len(got) != 5 {
					t.Fatalf("expected 5 lines, got %d", len(got))
				}
				last := got[len(got)-1]
				if last[len(last)-len("▼"):] != "▼" {
					t.Errorf("last line should end with ▼, got %q", last)
				}
				first := got[0]
				if first[len(first)-len("▲"):] != "▲" {
					t.Errorf("first line should end with ▲, got %q", first)
				}
			},
		},
		{
			name:  "overflow at bottom shows bottom arrow",
			lines: []string{"row1", "row2", "row3"},
			opts:  ui.ScrollGutterOpts{ViewOffset: 7, TotalItems: 10, ViewportHeight: 3},
			want: func(t *testing.T, got []string) {
				last := got[len(got)-1]
				if last[len(last)-len("▼"):] != "▼" {
					t.Errorf("last line should end with ▼, got %q", last)
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
			name:  "thumb position moves with offset",
			lines: []string{"a", "b", "c", "d", "e"},
			opts:  ui.ScrollGutterOpts{ViewOffset: 10, TotalItems: 20, ViewportHeight: 5},
			want: func(t *testing.T, got []string) {
				// Thumb should be roughly in the middle
				hasThumb := false
				for i := 1; i < len(got)-1; i++ {
					line := got[i]
					if line[len(line)-len("┃"):] == "┃" {
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
