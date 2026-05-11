package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type StatusBar struct {
	shortcuts  string
	binaryName string
}

func NewStatusBar() StatusBar { return StatusBar{} }

// WithShortcuts returns a StatusBar that displays the given shortcut hints.
func (s StatusBar) WithShortcuts(shortcuts string) StatusBar {
	s.shortcuts = shortcuts
	return s
}

// WithBinaryName returns a StatusBar that displays the binary name right-aligned.
func (s StatusBar) WithBinaryName(name string) StatusBar {
	s.binaryName = name
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
			sdk.StyleKey.Render("?") + " help"
	}

	content := s.appendBinaryName(bindings, width)
	return statusStyle.Width(width).Render(content)
}

// RenderHints formats a slice of KeyHint into a styled status bar string.
func (s StatusBar) RenderHints(hints []sdk.KeyHint, width int) string {
	var parts []string
	for _, h := range hints {
		if h.Key == "" {
			parts = append(parts, sdk.StyleFaint.Render(h.Description))
		} else {
			parts = append(parts, sdk.StyleKey.Render(h.Key)+" "+h.Description)
		}
	}
	bindings := strings.Join(parts, "  ")
	content := s.appendBinaryName(bindings, width)
	return statusStyle.Width(width).Render(content)
}

// appendBinaryName appends the binary name right-aligned if set.
func (s StatusBar) appendBinaryName(bindings string, width int) string {
	if s.binaryName == "" {
		return bindings
	}
	binaryLabel := sdk.StyleFaint.Render(s.binaryName)
	bindingsWidth := lipgloss.Width(bindings)
	binaryWidth := lipgloss.Width(binaryLabel)
	// Account for statusStyle padding (1 on each side)
	gap := width - bindingsWidth - binaryWidth - 2
	if gap < 2 {
		return bindings
	}
	return bindings + strings.Repeat(" ", gap) + binaryLabel
}
