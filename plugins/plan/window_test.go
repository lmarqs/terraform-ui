package plan

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// gutterColumn returns the gutter glyph ending each line of view that has one.
func gutterColumn(view string) []string {
	var gutter []string
	for _, line := range strings.Split(view, "\n") {
		runes := []rune(line)
		if len(runes) == 0 {
			continue
		}
		if g := string(runes[len(runes)-1]); strings.Contains("▲▼┃│", g) {
			gutter = append(gutter, g)
		}
	}
	return gutter
}

func TestView_GivenWrapOnAndCursorOnLastRow_ShouldDrawItAndPutTheThumbAtTheBottom(t *testing.T) {
	changes := make([]sdk.PlanChange, 20)
	for i := range changes {
		changes[i] = sdk.PlanChange{
			Resource: sdk.Resource{Address: fmt.Sprintf("aws_subnet.row%02d_%s", i, strings.Repeat("x", 60))},
			Action:   sdk.ActionUpdate,
		}
	}
	p, _ := newPlanPluginWithChanges(changes)
	p.listPanel.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlW})

	p.MoveToEnd()
	view := ansi.Strip(p.View(60, 20))

	if !strings.Contains(view, "row19") {
		t.Fatalf("cursor on the last row should draw it:\n%s", view)
	}
	gutter := gutterColumn(view)
	if len(gutter) < 3 || gutter[len(gutter)-1] != "▼" || gutter[len(gutter)-2] != "┃" {
		t.Fatalf("thumb should end at the bottom of the gutter, got %v:\n%s", gutter, view)
	}

	p.MoveUp()
	view = ansi.Strip(p.View(60, 20))
	if !strings.Contains(view, "row18") || !strings.Contains(view, "row19") {
		t.Fatalf("moving up one row should draw it and keep the window, which counts wrapped lines, on the last row:\n%s", view)
	}
}
