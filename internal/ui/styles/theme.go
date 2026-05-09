package styles

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary  = lipgloss.Color("39")
	ColorFaint    = lipgloss.Color("241")
	ColorText     = lipgloss.Color("252")
	ColorBg       = lipgloss.Color("236")
	ColorSuccess  = lipgloss.Color("40")
	ColorWarning  = lipgloss.Color("214")
	ColorDanger   = lipgloss.Color("196")
	ColorCritical = lipgloss.Color("201")
	ColorCreate   = lipgloss.Color("40")
	ColorUpdate   = lipgloss.Color("214")
	ColorDelete   = lipgloss.Color("196")
	ColorReplace  = lipgloss.Color("213")

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

	StyleCreate = lipgloss.NewStyle().
			Foreground(ColorCreate).
			Bold(true)

	StyleUpdate = lipgloss.NewStyle().
			Foreground(ColorUpdate).
			Bold(true)

	StyleDelete = lipgloss.NewStyle().
			Foreground(ColorDelete).
			Bold(true)

	StyleReplace = lipgloss.NewStyle().
			Foreground(ColorReplace).
			Bold(true)

	StyleRiskLow = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StyleRiskMedium = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StyleRiskHigh = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	StyleRiskCritical = lipgloss.NewStyle().
				Foreground(ColorCritical).
				Bold(true)

	StylePhantom = lipgloss.NewStyle().
			Foreground(ColorFaint).
			Italic(true)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)
)
