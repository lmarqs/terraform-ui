package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type StatusBar struct {
	shortcuts string
}

func NewStatusBar() StatusBar { return StatusBar{} }

// WithShortcuts returns a StatusBar that displays the given shortcut hints.
func (s StatusBar) WithShortcuts(shortcuts string) StatusBar {
	s.shortcuts = shortcuts
	return s
}

var statusStyle = lipgloss.NewStyle().
	Background(sdk.ColorBg).
	Foreground(sdk.ColorText).
	Padding(0, 1)

func (s StatusBar) Render(width int) string {
	var bindings string
	if s.shortcuts != "" {
		bindings = s.shortcuts
	} else {
		bindings = sdk.StyleKey.Render("q") + " quit  " +
			sdk.StyleKey.Render("esc") + " back  " +
			sdk.StyleKey.Render("^w") + " wrap  " +
			sdk.StyleKey.Render("/") + " search  " +
			sdk.StyleKey.Render("↑↓") + " navigate  " +
			sdk.StyleKey.Render("←→") + " pan  " +
			sdk.StyleKey.Render("?") + " help"
	}

	return statusStyle.Width(width).Render(bindings)
}
