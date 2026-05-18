package sdk

import "github.com/charmbracelet/lipgloss"

// Color palette constants — Dracula theme.
var (
	// ColorPrimary is the main accent color for titles and key highlights.
	ColorPrimary = lipgloss.Color("#bd93f9")
	// ColorFaint is used for de-emphasized or secondary text.
	ColorFaint = lipgloss.Color("#6272a4")
	// ColorText is the default text color.
	ColorText = lipgloss.Color("#f8f8f2")
	// ColorBg is the background color for selected/highlighted rows.
	ColorBg = lipgloss.Color("#44475a")
	// ColorSuccess indicates successful operations or low-risk items.
	ColorSuccess = lipgloss.Color("#50fa7b")
	// ColorWarning indicates medium-risk or cautionary items.
	ColorWarning = lipgloss.Color("#f1fa8c")
	// ColorDanger indicates high-risk or error states.
	ColorDanger = lipgloss.Color("#ff5555")
	// ColorCritical indicates critical-risk items requiring immediate attention.
	ColorCritical = lipgloss.Color("#ff79c6")
	// ColorCreate is the color for resource creation actions.
	ColorCreate = lipgloss.Color("#50fa7b")
	// ColorUpdate is the color for resource update actions.
	ColorUpdate = lipgloss.Color("#f1fa8c")
	// ColorDelete is the color for resource deletion actions.
	ColorDelete = lipgloss.Color("#ff5555")
	// ColorReplace is the color for resource replace actions.
	ColorReplace = lipgloss.Color("#ff79c6")

	// StyleTitle renders bold primary-colored section headings.
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// StyleFaint renders de-emphasized secondary text.
	StyleFaint = lipgloss.NewStyle().
			Foreground(ColorFaint)

	// StyleFaintItalic renders de-emphasized italic text for hints and placeholders.
	StyleFaintItalic = lipgloss.NewStyle().
				Foreground(ColorFaint).
				Italic(true)

	// StyleKey renders bold primary-colored text for key labels and prompts.
	StyleKey = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// StyleSelected renders highlighted background for the currently selected row.
	StyleSelected = lipgloss.NewStyle().
			Background(ColorBg)

	// StylePadded adds uniform padding around content blocks.
	StylePadded = lipgloss.NewStyle().
			Padding(1, 2)

	// StyleCreate renders bold green text for resource creation indicators.
	StyleCreate = lipgloss.NewStyle().
			Foreground(ColorCreate).
			Bold(true)

	// StyleUpdate renders bold amber text for resource update indicators.
	StyleUpdate = lipgloss.NewStyle().
			Foreground(ColorUpdate).
			Bold(true)

	// StyleDelete renders bold red text for resource deletion indicators.
	StyleDelete = lipgloss.NewStyle().
			Foreground(ColorDelete).
			Bold(true)

	// StyleReplace renders bold pink text for resource replace indicators.
	StyleReplace = lipgloss.NewStyle().
			Foreground(ColorReplace).
			Bold(true)

	// StyleRiskLow renders green text for low-risk items.
	StyleRiskLow = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// StyleRiskMedium renders amber text for medium-risk items.
	StyleRiskMedium = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// StyleRiskHigh renders bold red text for high-risk items.
	StyleRiskHigh = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	// StyleRiskCritical renders bold magenta text for critical-risk items.
	StyleRiskCritical = lipgloss.NewStyle().
				Foreground(ColorCritical).
				Bold(true)

	// StylePhantom renders faint italic text for phantom (cosmetic-only) changes.
	StylePhantom = lipgloss.NewStyle().
			Foreground(ColorFaint).
			Italic(true)

	// StyleSuccess renders bold green text for success messages.
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// StyleError renders bold red text for error messages.
	StyleError = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)
)
