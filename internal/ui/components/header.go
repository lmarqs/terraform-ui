package components

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

var logo = [3]string{
	"╔╦╗╔═╗╦ ╦╦",
	" ║ ╠╣ ║ ║║",
	" ╩ ╚  ╚═╝╩",
}

// Header renders a 3-line info block with project, scope, workspace on
// the left, and an ASCII logo on the right.
type Header struct {
	dir         string
	workspace   string
	chdir       string
	pinnedCount int
}

// NewHeader creates a header.
func NewHeader(dir, workspace string) Header {
	return Header{
		dir:       dir,
		workspace: workspace,
	}
}

// WithChdir returns a copy with the active chdir set.
func (h Header) WithChdir(chdir string) Header {
	h.chdir = chdir
	return h
}

// WithPinnedCount returns a copy with the pinned targets count.
func (h Header) WithPinnedCount(count int) Header {
	h.pinnedCount = count
	return h
}

var headerLabelStyle = lipgloss.NewStyle().
	Foreground(sdk.ColorFaint)

var headerValueStyle = lipgloss.NewStyle().
	Foreground(sdk.ColorText)

var logoStyle = lipgloss.NewStyle().
	Foreground(sdk.ColorPrimary).
	Bold(true)

// Render produces the 3-line header at the given width.
func (h Header) Render(width int) string {
	chdirVal := h.chdir
	if chdirVal == "" {
		chdirVal = "-"
	}

	projectParts := []string{filepath.Base(h.dir)}
	if h.pinnedCount > 0 {
		projectParts = append(projectParts, sdk.StyleSuccess.Render(fmt.Sprintf("%d pinned", h.pinnedCount)))
	}
	line1Left := headerLabelStyle.Render(" Project:") + " " + headerValueStyle.Render(strings.Join(projectParts, " │ "))
	line2Left := headerLabelStyle.Render(" Chdir:") + " " + headerValueStyle.Render(chdirVal)
	line3Left := headerLabelStyle.Render(" Workspace:") + " " + headerValueStyle.Render(h.workspace)

	logoWidth := lipgloss.Width(logo[0])

	lines := [3]string{line1Left, line2Left, line3Left}
	var result []string
	for i, left := range lines {
		leftWidth := lipgloss.Width(left)
		gap := width - leftWidth - logoWidth
		if gap < 1 {
			gap = 1
		}
		right := logoStyle.Render(logo[i])
		result = append(result, left+strings.Repeat(" ", gap)+right)
	}

	return strings.Join(result, "\n")
}
