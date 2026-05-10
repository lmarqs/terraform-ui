package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// ContentBorder renders content in a rounded border box with a title
// and optional item count embedded in the top border line.
type ContentBorder struct{}

// NewContentBorder creates a new content border component.
func NewContentBorder() ContentBorder {
	return ContentBorder{}
}

// Render wraps content in a bordered box with title in the top border.
// If filtered != total, shows "(filtered/total)". If equal and > 0, shows "(total)".
// Width is the outer box width. Height is the outer box height (including borders).
func (c ContentBorder) Render(content, title string, filtered, total, width, height int) string {
	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	titleStr := formatBorderTitle(title, filtered, total)

	topBorder := buildTopBorder(titleStr, width)
	bottomBorder := "└" + strings.Repeat("─", innerWidth) + "┘"

	borderFg := lipgloss.NewStyle().Foreground(sdk.ColorPrimary)
	topBorder = borderFg.Render(topBorder)
	bottomBorder = borderFg.Render(bottomBorder)

	contentStyle := lipgloss.NewStyle().
		Width(innerWidth).
		Height(height - 2).
		MaxHeight(height - 2)

	rendered := contentStyle.Render(content)

	var lines []string
	for _, line := range strings.Split(rendered, "\n") {
		lines = append(lines, borderFg.Render("│")+line+strings.Repeat(" ", max(0, innerWidth-lipgloss.Width(line)))+borderFg.Render("│"))
	}

	return topBorder + "\n" + strings.Join(lines, "\n") + "\n" + bottomBorder
}

func formatBorderTitle(title string, filtered, total int) string {
	if total <= 0 {
		return title
	}
	if filtered == total {
		return fmt.Sprintf("%s (%d)", title, total)
	}
	return fmt.Sprintf("%s (%d/%d)", title, filtered, total)
}

func buildTopBorder(title string, width int) string {
	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = 1
	}

	titleLen := len(title)
	if titleLen == 0 {
		return "┌" + strings.Repeat("─", innerWidth) + "┐"
	}

	decorated := " " + title + " "
	decoratedLen := len(decorated)

	if decoratedLen >= innerWidth {
		return "┌" + decorated[:innerWidth] + "┐"
	}

	leftPad := (innerWidth - decoratedLen) / 2
	rightPad := innerWidth - decoratedLen - leftPad

	return "┌" + strings.Repeat("─", leftPad) + decorated + strings.Repeat("─", rightPad) + "┐"
}

