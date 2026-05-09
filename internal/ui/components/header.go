package components

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type Header struct {
	dir           string
	workspace     string
	resourceCount int
	binaryName    string
	context       string
}

func NewHeader(dir, workspace, binaryPath string, resourceCount int) Header {
	name := filepath.Base(binaryPath)
	return Header{dir: dir, workspace: workspace, binaryName: name, resourceCount: resourceCount}
}

// WithContext returns a copy of the Header with the active context set.
func (h Header) WithContext(context string) Header {
	h.context = context
	return h
}

var headerStyle = lipgloss.NewStyle().
	Background(sdk.ColorBg).
	Foreground(sdk.ColorText).
	Bold(true).
	Padding(0, 1)

func (h Header) Render(width int) string {
	left := fmt.Sprintf("%s %s  %s %s  %s %s",
		sdk.StyleKey.Render("workspace:"),
		h.workspace,
		sdk.StyleKey.Render("dir:"),
		h.dir,
		sdk.StyleKey.Render("binary:"),
		h.binaryName,
	)

	if h.context != "" {
		left += fmt.Sprintf("  %s %s",
			sdk.StyleKey.Render("context:"),
			h.context,
		)
	}

	right := fmt.Sprintf("%s %d",
		sdk.StyleKey.Render("resources:"),
		h.resourceCount,
	)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	content := left + fmt.Sprintf("%*s", gap, "") + right
	return headerStyle.Width(width).Render(content)
}
