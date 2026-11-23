package plan

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestView_GivenWrapOn_ShouldWindowByWrappedLinesAndPlaceTheGutterThumbByThem(t *testing.T) {
	const width, height = 60, 20
	tests := []struct {
		name      string
		down      int
		toEnd     bool
		up        int
		wantFirst string
		wantLast  string
		// The thumb is sized and placed by wrapped lines, not rows: 20 rows of about 2.5
		// lines each. Counting rows would draw a thumb one cell taller.
		wantGutter string
	}{
		{name: "cursor on the first row", wantFirst: "row00", wantLast: "row07", wantGutter: "▲┃┃┃┃│││││││││▼"},
		{name: "cursor ten rows down", down: 10, wantFirst: "row04", wantLast: "row10", wantGutter: "▲││┃┃┃┃│││││││▼"},
		{name: "cursor fifteen rows down", down: 15, wantFirst: "row09", wantLast: "row15", wantGutter: "▲││││││┃┃┃┃│││▼"},
		{name: "cursor on the last row", toEnd: true, wantFirst: "row13", wantLast: "row19", wantGutter: "▲│││││││││┃┃┃┃▼"},
		{name: "cursor one row above the last", toEnd: true, up: 1, wantFirst: "row12", wantLast: "row19", wantGutter: "▲││││││││┃┃┃┃│▼"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := make([]sdk.PlanChange, 20)
			for i := range changes {
				changes[i] = sdk.PlanChange{
					Resource: sdk.Resource{Address: fmt.Sprintf("aws_subnet.row%02d_%s", i, strings.Repeat("x", 60))},
					Action:   sdk.ActionUpdate,
				}
			}
			p, _ := newPlanPluginWithChanges(changes)
			p.listPanel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
			p.View(width, height)

			// The window follows the frames drawn, so each move draws one.
			for range tt.down {
				p.MoveDown()
				p.View(width, height)
			}
			if tt.toEnd {
				p.MoveToEnd()
				p.View(width, height)
			}
			for range tt.up {
				p.MoveUp()
				p.View(width, height)
			}
			view := ansi.Strip(p.View(width, height))

			for _, row := range []string{tt.wantFirst, tt.wantLast} {
				if !strings.Contains(view, row) {
					t.Errorf("view should draw %s:\n%s", row, view)
				}
			}
			if got := strings.Join(sdktest.GutterColumn(view), ""); got != tt.wantGutter {
				t.Errorf("gutter = %s, want %s:\n%s", got, tt.wantGutter, view)
			}
		})
	}
}
