package components

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// PlanSummary holds counts of planned resource changes.
type PlanSummary struct {
	Create  int
	Update  int
	Delete  int
	Replace int
}

type Header struct {
	dir           string
	workspace     string
	resourceCount int
	binaryName    string
	context       string
	activeView    string
	planSummary   *PlanSummary
	pinnedCount   int
	expanded      bool
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

// WithActiveView returns a copy of the Header with the active view name.
func (h Header) WithActiveView(view string) Header {
	h.activeView = view
	return h
}

// WithPlanSummary returns a copy of the Header with the plan summary counts.
func (h Header) WithPlanSummary(create, update, delete, replace int) Header {
	h.planSummary = &PlanSummary{
		Create:  create,
		Update:  update,
		Delete:  delete,
		Replace: replace,
	}
	return h
}

// WithPinnedCount returns a copy of the Header with the pinned targets count.
func (h Header) WithPinnedCount(count int) Header {
	h.pinnedCount = count
	return h
}

// WithExpanded returns a copy of the Header with the expanded flag set.
func (h Header) WithExpanded(expanded bool) Header {
	h.expanded = expanded
	return h
}

// Toggle returns a copy of the Header with the expanded flag toggled.
func (h Header) Toggle() Header {
	h.expanded = !h.expanded
	return h
}

var headerStyle = lipgloss.NewStyle().
	Background(sdk.ColorBg).
	Foreground(sdk.ColorText).
	Bold(true).
	Padding(0, 1)

func (h Header) Render(width int) string {
	if h.expanded {
		return h.renderExpanded(width)
	}
	return h.renderCompact(width)
}

func (h Header) renderCompact(width int) string {
	parts := []string{
		sdk.StyleKey.Render("⎈") + " " + h.workspace,
		h.dir,
		sdk.StyleFaint.Render(h.binaryName),
	}

	if h.context != "" {
		parts = append(parts, sdk.StyleKey.Render("ctx:")+" "+h.context)
	}

	if h.planSummary != nil {
		plan := h.formatPlanCompact()
		if plan != "" {
			parts = append(parts, plan)
		}
	}

	if h.pinnedCount > 0 {
		parts = append(parts, sdk.StyleSuccess.Render(fmt.Sprintf("\U0001F4CC %d pinned", h.pinnedCount)))
	}

	left := strings.Join(parts, " │ ")

	right := h.formatRight()

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	content := left + fmt.Sprintf("%*s", gap, "") + right
	return headerStyle.Width(width).Render(content)
}

func (h Header) renderExpanded(width int) string {
	// Line 1: workspace, dir, binary, context
	line1Parts := []string{
		sdk.StyleKey.Render("⎈") + " " + sdk.StyleKey.Render("workspace:") + " " + h.workspace,
		sdk.StyleKey.Render("dir:") + " " + h.dir,
		sdk.StyleKey.Render("binary:") + " " + sdk.StyleFaint.Render(h.binaryName),
	}
	if h.context != "" {
		line1Parts = append(line1Parts, sdk.StyleKey.Render("context:")+" "+h.context)
	}
	line1 := strings.Join(line1Parts, "   ")

	// Line 2: plan summary + pinned
	var line2Parts []string
	if h.planSummary != nil {
		line2Parts = append(line2Parts, h.formatPlanExpanded())
	}
	if h.pinnedCount > 0 {
		line2Parts = append(line2Parts, sdk.StyleSuccess.Render(fmt.Sprintf("\U0001F4CC Pinned: %d targets", h.pinnedCount)))
	}
	line2 := strings.Join(line2Parts, "   ")

	// Line 3: separator + active view
	right := h.formatRight()
	separatorLen := width - lipgloss.Width(right) - 3
	if separatorLen < 4 {
		separatorLen = 4
	}
	line3 := strings.Repeat("─", separatorLen) + " " + right

	var lines []string
	lines = append(lines, line1)
	if line2 != "" {
		lines = append(lines, line2)
	}
	lines = append(lines, line3)

	content := strings.Join(lines, "\n")
	return headerStyle.Width(width).Render(content)
}

func (h Header) formatPlanCompact() string {
	if h.planSummary == nil {
		return ""
	}
	s := h.planSummary
	var parts []string
	if s.Create > 0 {
		parts = append(parts, sdk.StyleCreate.Render(fmt.Sprintf("+%d", s.Create)))
	}
	if s.Update > 0 {
		parts = append(parts, sdk.StyleUpdate.Render(fmt.Sprintf("~%d", s.Update)))
	}
	if s.Delete > 0 {
		parts = append(parts, sdk.StyleDelete.Render(fmt.Sprintf("-%d", s.Delete)))
	}
	if s.Replace > 0 {
		parts = append(parts, sdk.StyleReplace.Render(fmt.Sprintf("±%d", s.Replace)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Plan: " + strings.Join(parts, " ")
}

func (h Header) formatPlanExpanded() string {
	if h.planSummary == nil {
		return ""
	}
	s := h.planSummary
	var parts []string
	if s.Create > 0 {
		parts = append(parts, sdk.StyleCreate.Render(fmt.Sprintf("%d to add", s.Create)))
	}
	if s.Update > 0 {
		parts = append(parts, sdk.StyleUpdate.Render(fmt.Sprintf("%d to change", s.Update)))
	}
	if s.Delete > 0 {
		parts = append(parts, sdk.StyleDelete.Render(fmt.Sprintf("%d to destroy", s.Delete)))
	}
	if s.Replace > 0 {
		parts = append(parts, sdk.StyleReplace.Render(fmt.Sprintf("%d to replace", s.Replace)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "\U0001F4CB Plan: " + strings.Join(parts, ", ")
}

func (h Header) formatRight() string {
	if h.activeView != "" {
		return sdk.StyleTitle.Render(h.activeView)
	}
	return fmt.Sprintf("%s %d",
		sdk.StyleKey.Render("resources:"),
		h.resourceCount,
	)
}
