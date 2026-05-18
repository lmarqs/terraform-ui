package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ActionChip describes a single terraform action shown in the actions bar.
type ActionChip struct {
	Key   string
	Label string
}

// ActionsBarHeight is the number of rows consumed by the actions bar
// (blank separator line + chip row).
const ActionsBarHeight = 2

var chipKeyStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#bd93f9")).
	Foreground(lipgloss.Color("#ffffff")).
	Bold(true)

var chipLabelStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#644e84")).
	Foreground(lipgloss.Color("#f8f8f2"))

// RenderActionsBar renders a row of styled action chips.
// Returns empty string if actions is empty.
func RenderActionsBar(actions []ActionChip, width int) string {
	if len(actions) == 0 {
		return ""
	}

	var chips []string
	for _, a := range actions {
		chips = append(chips, chipKeyStyle.Render(a.Key+" ")+chipLabelStyle.Render(a.Label))
	}

	row := " " + strings.Join(chips, " ")

	if lipgloss.Width(row) > width {
		row = lipgloss.NewStyle().MaxWidth(width).Render(row)
	}

	return "\n\n" + row
}
