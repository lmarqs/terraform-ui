package components

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

var logo = [3]string{
	"╔╦╗╔═╗╦ ╦╦",
	" ║ ╠╣ ║ ║║",
	" ╩ ╚  ╚═╝╩",
}

// Header renders a 3-line info block with project, chdir, workspace on
// the left, and an ASCII logo on the right.
type Header struct {
	dir         string
	workspace   string
	chdir       string
	pinnedCount int
	lockInfo    *sdk.StateLock
	stale       bool
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

// WithWorkspace returns a copy with the workspace updated.
func (h Header) WithWorkspace(ws string) Header {
	h.workspace = ws
	return h
}

// WithPinnedCount returns a copy with the pinned targets count.
func (h Header) WithPinnedCount(count int) Header {
	h.pinnedCount = count
	return h
}

// WithLockInfo returns a copy with the lock indicator set (nil to clear).
func (h Header) WithLockInfo(lock *sdk.StateLock) Header {
	h.lockInfo = lock
	return h
}

// WithStale returns a copy with the stale indicator set.
func (h Header) WithStale(stale bool) Header {
	h.stale = stale
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

	wsParts := []string{h.workspace}
	if h.stale {
		wsParts = append(wsParts, sdk.StyleUpdate.Render("stale"))
	}
	if h.lockInfo != nil {
		wsParts = append(wsParts, sdk.StyleError.Render(formatLockBadge(h.lockInfo)))
	}
	line3Left := headerLabelStyle.Render(" Workspace:") + " " + headerValueStyle.Render(strings.Join(wsParts, " │ "))

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

func formatLockBadge(lock *sdk.StateLock) string {
	parts := []string{"locked"}
	detail := ""
	if lock.Who != "" {
		detail = lock.Who
	}
	if !lock.Created.IsZero() {
		age := time.Since(lock.Created)
		ageStr := formatBadgeAge(age)
		if detail != "" {
			detail += " " + ageStr
		} else {
			detail = ageStr
		}
	}
	if detail != "" {
		parts = append(parts, "("+detail+")")
	}
	return strings.Join(parts, " ")
}

func formatBadgeAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	hours := int(d.Hours())
	if hours >= 24 {
		return fmt.Sprintf("%dd ago", hours/24)
	}
	return fmt.Sprintf("%dh ago", hours)
}
