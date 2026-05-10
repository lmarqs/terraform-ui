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

// Header renders a 3-line info block with context, workspace, dir+binary on
// the left, and an ASCII logo on the right.
type Header struct {
	dir         string
	workspace   string
	binaryName  string
	context     string
	pinnedCount int
}

// NewHeader creates a header. binaryPath is resolved to its base name.
func NewHeader(dir, workspace, binaryPath string) Header {
	return Header{
		dir:        dir,
		workspace:  workspace,
		binaryName: filepath.Base(binaryPath),
	}
}

// WithContext returns a copy with the active context set.
func (h Header) WithContext(context string) Header {
	h.context = context
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
	ctxVal := h.context
	if ctxVal == "" {
		ctxVal = "-"
	}

	line1Left := headerLabelStyle.Render(" Context:") + " " + headerValueStyle.Render(ctxVal)
	line2Left := headerLabelStyle.Render(" Workspace:") + " " + headerValueStyle.Render(h.workspace)

	line3Parts := []string{h.dir}
	line3Parts = append(line3Parts, sdk.StyleFaint.Render(h.binaryName))
	if h.pinnedCount > 0 {
		line3Parts = append(line3Parts, sdk.StyleSuccess.Render(fmt.Sprintf("%d pinned", h.pinnedCount)))
	}
	line3Left := headerLabelStyle.Render(" Dir:") + " " + headerValueStyle.Render(strings.Join(line3Parts, " │ "))

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
