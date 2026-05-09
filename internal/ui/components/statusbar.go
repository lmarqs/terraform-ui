package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

type StatusBar struct{}

func NewStatusBar() StatusBar { return StatusBar{} }

var statusStyle = lipgloss.NewStyle().
	Background(styles.ColorBg).
	Foreground(styles.ColorText).
	Padding(0, 1)

func (s StatusBar) Render(width int) string {
	bindings := styles.StyleKey.Render("q") + " quit  " +
		styles.StyleKey.Render("esc") + " back  " +
		styles.StyleKey.Render("?") + " help  " +
		styles.StyleKey.Render("/") + " search  " +
		styles.StyleKey.Render("↑↓") + " navigate"

	return statusStyle.Width(width).Render(bindings)
}
