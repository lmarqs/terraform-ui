package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Separator renders a horizontal double-line separator (═) in the accent color.
type Separator struct{}

// NewSeparator creates a new separator component.
func NewSeparator() Separator {
	return Separator{}
}

var separatorStyle = lipgloss.NewStyle().
	Foreground(sdk.ColorPrimary)

// Render returns a full-width separator line.
func (s Separator) Render(width int) string {
	line := strings.Repeat("═", width)
	return separatorStyle.Render(line)
}
