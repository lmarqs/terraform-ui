// Package styles defines the shared lipgloss color palette and style constants
// used by all tfui views and plugins for consistent terminal rendering.
package styles

import "github.com/charmbracelet/lipgloss"

// Color palette constants used across the theme.
var (
	// ColorPrimary is the main accent color for titles and key highlights.
	ColorPrimary = lipgloss.Color("39")
	// ColorFaint is used for de-emphasized or secondary text.
	ColorFaint = lipgloss.Color("241")
	// ColorText is the default text color.
	ColorText = lipgloss.Color("252")
	// ColorBg is the background color for selected/highlighted rows.
	ColorBg = lipgloss.Color("236")
	// ColorSuccess indicates successful operations or low-risk items.
	ColorSuccess = lipgloss.Color("40")
	// ColorWarning indicates medium-risk or cautionary items.
	ColorWarning = lipgloss.Color("214")
	// ColorDanger indicates high-risk or error states.
	ColorDanger = lipgloss.Color("196")
	// ColorCritical indicates critical-risk items requiring immediate attention.
	ColorCritical = lipgloss.Color("201")
	// ColorCreate is the color for resource creation actions.
	ColorCreate = lipgloss.Color("40")
	// ColorUpdate is the color for resource update actions.
	ColorUpdate = lipgloss.Color("214")
	// ColorDelete is the color for resource deletion actions.
	ColorDelete = lipgloss.Color("196")
	// ColorReplace is the color for resource replace actions.
	ColorReplace = lipgloss.Color("213")

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
