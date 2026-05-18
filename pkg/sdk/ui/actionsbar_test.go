package ui_test

import (
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func TestRenderActionsBar(t *testing.T) {
	tests := []struct {
		name    string
		actions []ui.ActionChip
		width   int
		wantFn  func(t *testing.T, got string)
	}{
		{
			name:    "empty actions returns empty string",
			actions: nil,
			width:   80,
			wantFn: func(t *testing.T, got string) {
				if got != "" {
					t.Errorf("expected empty string, got %q", got)
				}
			},
		},
		{
			name:    "single chip renders with blank line prefix",
			actions: []ui.ActionChip{{Key: "d", Label: "delete"}},
			width:   80,
			wantFn: func(t *testing.T, got string) {
				if !strings.HasPrefix(got, "\n") {
					t.Error("expected leading newline (blank separator)")
				}
				if !strings.Contains(got, "d") {
					t.Error("expected key 'd' in output")
				}
				if !strings.Contains(got, "delete") {
					t.Error("expected label 'delete' in output")
				}
			},
		},
		{
			name: "multiple chips separated by space",
			actions: []ui.ActionChip{
				{Key: "d", Label: "delete"},
				{Key: "t", Label: "taint"},
				{Key: "T", Label: "untaint"},
			},
			width: 80,
			wantFn: func(t *testing.T, got string) {
				lines := strings.Split(got, "\n")
				if len(lines) != 3 {
					t.Fatalf("expected 3 parts (join + blank + chips), got %d", len(lines))
				}
				if lines[0] != "" || lines[1] != "" {
					t.Errorf("first two parts should be empty, got %q and %q", lines[0], lines[1])
				}
			},
		},
		{
			name: "chips contain both key and label text",
			actions: []ui.ActionChip{
				{Key: "a", Label: "apply"},
				{Key: "!", Label: "batch"},
			},
			width: 80,
			wantFn: func(t *testing.T, got string) {
				if !strings.Contains(got, "a") || !strings.Contains(got, "apply") {
					t.Error("expected 'a apply' chip content")
				}
				if !strings.Contains(got, "!") || !strings.Contains(got, "batch") {
					t.Error("expected '! batch' chip content")
				}
			},
		},
		{
			name: "height is ActionsBarHeight plus join char",
			actions: []ui.ActionChip{
				{Key: "d", Label: "delete"},
			},
			width: 80,
			wantFn: func(t *testing.T, got string) {
				lines := strings.Split(got, "\n")
				if len(lines) != ui.ActionsBarHeight+1 {
					t.Errorf("expected %d parts (join + blank separator + chips), got %d", ui.ActionsBarHeight+1, len(lines))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ui.RenderActionsBar(tt.actions, tt.width)
			tt.wantFn(t, got)
		})
	}
}
