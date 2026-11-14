package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// labelQAsHome relabels the plugins' shared "q quit" hint for the full TUI,
// where q goes home; only a standalone run and the home screen quit on q.
func labelQAsHome(hints []sdk.KeyHint) []sdk.KeyHint {
	out := make([]sdk.KeyHint, len(hints))
	for i, h := range hints {
		if h == sdk.HintQuit {
			h.Description = "home"
		}
		out[i] = h
	}
	return out
}

// activeHints returns the frame stack's hints, falling back to the plugin's own
// when the stack has none (an empty stack, as on apply's confirmation screen).
func activeHints(p sdk.Plugin) []sdk.KeyHint {
	if stackable, ok := p.(sdk.Stackable); ok {
		if hints := stackable.Stack().Hints(); hints != nil {
			return hints
		}
	}
	if hintable, ok := p.(sdk.Hintable); ok {
		return hintable.Hints()
	}
	return nil
}

// homeHints are the status bar's hints on the home screen.
var homeHints = []sdk.KeyHint{
	{Key: "Enter", Description: "select"},
	{Key: ":", Description: "command"},
	{Key: "q", Description: "quit"},
}

// renderStatusBar renders the bottom row: a command error, the input prompt,
// the active plugin's hints, or idleHints when no plugin is active (nothing
// when idleHints is nil).
func (a App) renderStatusBar(idleHints []sdk.KeyHint) string {
	switch {
	case a.commandError != "":
		return lipgloss.NewStyle().
			Background(sdk.ColorBg).
			Foreground(sdk.ColorDanger).
			Padding(0, 1).
			Width(a.width).
			Render(a.commandError)
	case a.inputActive:
		return lipgloss.NewStyle().
			Background(sdk.ColorBg).
			Foreground(sdk.ColorText).
			Bold(true).
			Padding(0, 1).
			Width(a.width).
			Render(a.inputPrompt + " " + a.inputAnswer + "█")
	case a.activePlugin != nil:
		if hints := activeHints(a.activePlugin); hints != nil {
			if a.standalone == nil {
				hints = labelQAsHome(hints)
			}
			return a.statusBar.RenderHints(hints, a.width)
		}
		return a.statusBar.Render(a.width)
	case idleHints != nil:
		return a.statusBar.RenderHints(idleHints, a.width)
	}
	return ""
}

// renderOverlay renders the active overlay centered over the screen, above
// statusBar or the overlay's own hints.
func (a App) renderOverlay(statusBar string) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sdk.ColorPrimary).
		Padding(1, 2).
		Width(a.width / 2).
		MaxHeight(a.height - 4).
		Render(a.activeOverlay.View(a.width, a.height))
	overlayView := lipgloss.Place(a.width, a.height-1, lipgloss.Center, lipgloss.Center, box)
	if hints := a.activeOverlay.Hints(); hints != nil {
		statusBar = a.statusBar.RenderHints(hints, a.width)
	}
	return overlayView + "\n" + statusBar
}

func (a App) View() string {
	// Only before the first WindowSizeMsg arrives: an unusable size reported by
	// the terminal is substituted, never left at zero.
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	// Standalone mode: minimal chrome (single header line + hint bar)
	if a.standalone != nil {
		return a.viewStandalone()
	}

	headerHeight := 3
	footerHeight := 1
	borderChrome := 2
	commandBarHeight := 0
	if a.commandMode {
		commandBarHeight = 3
	}
	contentHeight := a.height - headerHeight - commandBarHeight - borderChrome - footerHeight

	title := "Home"
	filtered, total, pinned := 0, 0, 0
	cursor, navigable := 0, 0
	if a.activePlugin != nil {
		title = a.activePlugin.Name()
		if c, ok := a.activePlugin.(sdk.Countable); ok {
			filtered, total = c.Count()
		}
		if p, ok := a.activePlugin.(sdk.Pinnable); ok {
			pinned = p.PinnedCount()
		}
		if pos, ok := a.activePlugin.(sdk.Positionable); ok {
			cursor, navigable = pos.CursorPosition()
		}
	}

	var content string
	innerWidth := a.width - 2
	if a.activePlugin != nil {
		content = a.activePlugin.View(innerWidth, contentHeight)
	} else {
		content = a.homeView.Render(innerWidth, contentHeight)
	}

	header := a.header.Render(a.width)
	bordered := a.contentBorder.Render(content, title, filtered, total, pinned, cursor, navigable, a.width, contentHeight+borderChrome)

	statusBar := a.renderStatusBar(homeHints)

	if a.activeOverlay != nil {
		return a.renderOverlay(statusBar)
	}

	var parts []string
	parts = append(parts, header)
	if a.commandMode {
		parts = append(parts, a.commandBar.Render(a.commandInput, a.commandMatches(), a.width))
	}
	parts = append(parts, bordered)
	parts = append(parts, statusBar)
	return strings.Join(parts, "\n")
}

func (a App) viewStandalone() string {
	headerHeight := 1
	footerHeight := 1
	padding := 2 // blank row after header + blank row before footer
	contentHeight := a.height - headerHeight - footerHeight - padding

	// Minimal header: context info on left, tfui on right
	headerStyle := lipgloss.NewStyle().
		Width(a.width)
	sep := sdk.StyleFaint.Render(" › ")
	projectStyle := lipgloss.NewStyle().Foreground(sdk.ColorPrimary).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(sdk.ColorText)

	var leftParts []string
	leftParts = append(leftParts, projectStyle.Render(filepath.Base(a.cfg.WorkingDir())))
	if !a.activeChdir.IsZero() {
		leftParts = append(leftParts, valueStyle.Render(a.activeChdir.String()))
	}
	if !a.activeWorkspace.IsZero() {
		leftParts = append(leftParts, valueStyle.Render(a.activeWorkspace.String()))
	}
	if a.lockInfo != nil {
		leftParts = append(leftParts, sdk.StyleError.Render("[locked]"))
	}
	left := strings.Join(leftParts, sep)
	right := sdk.StyleFaint.Render("tfui")
	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	header := headerStyle.Render(left + strings.Repeat(" ", gap) + right)

	// Content
	var content string
	if a.activePlugin != nil {
		content = a.activePlugin.View(a.width, contentHeight)
	}

	statusBar := a.renderStatusBar(nil)
	if a.activeOverlay != nil {
		return a.renderOverlay(statusBar)
	}

	return header + "\n\n" + content + "\n\n" + statusBar
}
