package styles

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary = lipgloss.Color("39")
	ColorFaint   = lipgloss.Color("241")
	ColorText    = lipgloss.Color("252")
	ColorBg      = lipgloss.Color("236")
	ColorSuccess = lipgloss.Color("40")
	ColorWarning = lipgloss.Color("214")

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleFaint = lipgloss.NewStyle().
			Foreground(ColorFaint)

	StyleFaintItalic = lipgloss.NewStyle().
				Foreground(ColorFaint).
				Italic(true)

	StyleKey = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	StyleSelected = lipgloss.NewStyle().
			Background(ColorBg)

	StylePadded = lipgloss.NewStyle().
			Padding(1, 2)
)
