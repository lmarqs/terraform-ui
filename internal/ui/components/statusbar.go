package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type StatusBar struct{}

func NewStatusBar() StatusBar { return StatusBar{} }

var statusStyle = lipgloss.NewStyle().
	Background(sdk.ColorBg).
	Foreground(sdk.ColorText).
	Padding(0, 1)

func (s StatusBar) Render(width int) string {
	bindings := sdk.StyleKey.Render("q") + " quit  " +
		sdk.StyleKey.Render("esc") + " back  " +
		sdk.StyleKey.Render("^w") + " wrap  " +
		sdk.StyleKey.Render("←→") + " pan  " +
		sdk.StyleKey.Render("?") + " help  " +
		sdk.StyleKey.Render("/") + " search  " +
		sdk.StyleKey.Render("↑↓") + " navigate"

	return statusStyle.Width(width).Render(bindings)
}
