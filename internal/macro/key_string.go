package macro

import tea "github.com/charmbracelet/bubbletea"

// KeyToString converts a tea.KeyMsg back to the string representation
// used by the tape DSL. Returns empty string for unrecognized keys.
func KeyToString(msg tea.KeyMsg) string {
	switch msg.Type {
	case tea.KeyEnter:
		return "enter"
	case tea.KeyEsc:
		return "esc"
	case tea.KeyTab:
		return "tab"
	case tea.KeyBackspace:
		return "backspace"
	case tea.KeyUp:
		return "up"
	case tea.KeyDown:
		return "down"
	case tea.KeyLeft:
		return "left"
	case tea.KeyRight:
		return "right"
	case tea.KeySpace:
		return "space"
	case tea.KeyCtrlC:
		return "ctrl+c"
	case tea.KeyCtrlW:
		return "ctrl+w"
	case tea.KeyCtrlT:
		return "ctrl+t"
	case tea.KeyCtrlS:
		return "ctrl+s"
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			return string(msg.Runes)
		}
	}
	return ""
}
