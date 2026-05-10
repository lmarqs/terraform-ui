package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// CommandBar renders a bordered command input box for `:` command mode.
type CommandBar struct{}

// NewCommandBar creates a new command bar component.
func NewCommandBar() CommandBar {
	return CommandBar{}
}

// Render returns the bordered command bar with input and optional autocomplete matches.
func (c CommandBar) Render(input string, matches []string, width int) string {
	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	content := ":" + input + "█"
	if len(matches) > 0 {
		content += "  " + sdk.StyleFaint.Render(strings.Join(matches, " | "))
	}

	borderFg := lipgloss.NewStyle().Foreground(sdk.ColorPrimary)

	top := borderFg.Render("┌" + strings.Repeat("─", innerWidth) + "┐")
	bottom := borderFg.Render("└" + strings.Repeat("─", innerWidth) + "┘")

	contentWidth := lipgloss.Width(content)
	padding := ""
	if contentWidth < innerWidth {
		padding = strings.Repeat(" ", innerWidth-contentWidth)
	}
	middle := borderFg.Render("│") + content + padding + borderFg.Render("│")

	return top + "\n" + middle + "\n" + bottom
}
