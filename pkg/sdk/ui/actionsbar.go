package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// ActionChip describes a single terraform action shown in the actions bar.
type ActionChip struct {
	Key   string
	Label string
}

// ActionsBarHeight is the number of rows consumed by the actions bar
// (blank separator line + chip row).
const ActionsBarHeight = 2

var chipStyle = lipgloss.NewStyle().
	Background(sdk.ColorPrimary).
	Foreground(lipgloss.Color("0"))

// RenderActionsBar renders a row of styled action chips.
// Returns empty string if actions is empty.
func RenderActionsBar(actions []ActionChip, width int) string {
	if len(actions) == 0 {
		return ""
	}

	var chips []string
	for _, a := range actions {
		chips = append(chips, chipStyle.Render(a.Key+" "+a.Label))
	}

	row := " " + strings.Join(chips, " ")

	if lipgloss.Width(row) > width {
		row = lipgloss.NewStyle().MaxWidth(width).Render(row)
	}

	return "\n" + row
}
