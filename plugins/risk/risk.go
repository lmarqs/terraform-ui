package risk

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

// Status represents the current state of the risk plugin.
type Status int

const (
	StatusIdle Status = iota
	StatusReady
)

// RiskGroup holds changes grouped by risk level.
type RiskGroup struct {
	Level   terraform.RiskLevel
	Changes []terraform.PlanChange
}

// Plugin implements the risk analysis feature.
type Plugin struct {
	svc      terraform.Service
	status   Status
	groups   []RiskGroup
	overall  terraform.RiskLevel
	selected int
	total    int
}

// New creates a new risk analysis plugin.
func New(svc terraform.Service) plugin.Plugin {
	return &Plugin{
		svc: svc,
	}
}

func (e *Plugin) ID() string          { return "risk" }
func (e *Plugin) Name() string        { return "Risk Analysis" }
func (e *Plugin) Description() string { return "Analyze risk levels of planned changes" }
func (e *Plugin) KeyBinding() string  { return "R" }
func (e *Plugin) Ready() bool         { return e.status == StatusReady }
func (e *Plugin) Status() Status      { return e.status }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) Overall() terraform.RiskLevel {
	return e.overall
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context.
func (e *Plugin) Init(ctx *plugin.Context) tea.Cmd {
	e.svc = ctx.Service
	return nil
}

// Analyze processes a plan summary and groups changes by risk.
func (e *Plugin) Analyze(summary *terraform.PlanSummary) {
	if summary == nil || len(summary.Changes) == 0 {
		e.status = StatusReady
		e.groups = nil
		e.overall = terraform.RiskNone
		e.total = 0
		return
	}

	e.overall = terraform.OverallRisk(summary.Changes)
	e.total = len(summary.Changes)

	// Group changes by risk level (highest first)
	byLevel := map[terraform.RiskLevel][]terraform.PlanChange{
		terraform.RiskCritical: {},
		terraform.RiskHigh:     {},
		terraform.RiskMedium:   {},
		terraform.RiskLow:      {},
		terraform.RiskNone:     {},
	}

	for _, c := range summary.Changes {
		byLevel[c.Risk] = append(byLevel[c.Risk], c)
	}

	e.groups = make([]RiskGroup, 0)
	levels := []terraform.RiskLevel{
		terraform.RiskCritical,
		terraform.RiskHigh,
		terraform.RiskMedium,
		terraform.RiskLow,
		terraform.RiskNone,
	}
	for _, level := range levels {
		if len(byLevel[level]) > 0 {
			e.groups = append(e.groups, RiskGroup{
				Level:   level,
				Changes: byLevel[level],
			})
		}
	}

	e.status = StatusReady
	e.selected = 0
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		e.handleKey(msg)
		return e, nil
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "j", "down":
		e.MoveDown()
	case "k", "up":
		e.MoveUp()
	}
}

// MoveUp moves selection up.
func (e *Plugin) MoveUp() {
	if e.selected > 0 {
		e.selected--
	}
}

// MoveDown moves selection down.
func (e *Plugin) MoveDown() {
	max := e.totalItems() - 1
	if e.selected < max {
		e.selected++
	}
}

func (e *Plugin) totalItems() int {
	count := 0
	for _, g := range e.groups {
		count++ // group header
		count += len(g.Changes)
	}
	return count
}

// View renders the risk analysis plugin.
func (e *Plugin) View(width, height int) string {
	title := styles.StyleTitle.Render("Risk Analysis")

	switch e.status {
	case StatusIdle:
		placeholder := styles.StyleFaintItalic.Render("Run a plan first to analyze risk...")
		return styles.StylePadded.Render(title + "\n\n" + placeholder)

	case StatusReady:
		return e.renderAnalysis(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderAnalysis(width, height int) string {
	title := styles.StyleTitle.Render("Risk Analysis")

	if len(e.groups) == 0 {
		noRisk := styles.StyleSuccess.Render("No changes to analyze.")
		return styles.StylePadded.Render(title + "\n\n" + noRisk)
	}

	var b strings.Builder

	// Overall risk summary
	overallLine := e.renderOverallBanner()
	b.WriteString(overallLine)
	b.WriteString("\n\n")

	// Render each risk group
	itemIdx := 0
	maxVisible := height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}

	for _, group := range e.groups {
		header := e.renderGroupHeader(group)
		if itemIdx < maxVisible {
			if itemIdx == e.selected {
				header = styles.StyleSelected.Width(width - 6).Render(header)
			}
			b.WriteString(header)
			b.WriteByte('\n')
		}
		itemIdx++

		for _, change := range group.Changes {
			if itemIdx >= maxVisible {
				break
			}
			row := e.renderChangeRow(change)
			if itemIdx == e.selected {
				row = styles.StyleSelected.Width(width - 6).Render(row)
			}
			b.WriteString(row)
			b.WriteByte('\n')
			itemIdx++
		}
		b.WriteByte('\n')
	}

	// Statistics
	stats := e.renderStats()
	hint := styles.StyleFaintItalic.Render("j/k navigate  Esc back")

	content := title + "\n\n" + b.String() + stats + "\n" + hint
	return styles.StylePadded.Render(content)
}

func (e *Plugin) renderOverallBanner() string {
	switch e.overall {
	case terraform.RiskCritical:
		return styles.StyleRiskCritical.Render("!! CRITICAL RISK DETECTED !!")
	case terraform.RiskHigh:
		return styles.StyleRiskHigh.Render("! HIGH RISK DETECTED !")
	case terraform.RiskMedium:
		return styles.StyleRiskMedium.Render("Medium risk - review recommended")
	case terraform.RiskLow:
		return styles.StyleRiskLow.Render("Low risk - changes look safe")
	default:
		return styles.StyleSuccess.Render("No risk detected")
	}
}

func (e *Plugin) renderGroupHeader(group RiskGroup) string {
	var label string
	switch group.Level {
	case terraform.RiskCritical:
		label = styles.StyleRiskCritical.Render(fmt.Sprintf("CRITICAL (%d)", len(group.Changes)))
	case terraform.RiskHigh:
		label = styles.StyleRiskHigh.Render(fmt.Sprintf("HIGH (%d)", len(group.Changes)))
	case terraform.RiskMedium:
		label = styles.StyleRiskMedium.Render(fmt.Sprintf("MEDIUM (%d)", len(group.Changes)))
	case terraform.RiskLow:
		label = styles.StyleRiskLow.Render(fmt.Sprintf("LOW (%d)", len(group.Changes)))
	default:
		label = styles.StyleFaint.Render(fmt.Sprintf("NONE (%d)", len(group.Changes)))
	}
	return "--- " + label + " ---"
}

func (e *Plugin) renderChangeRow(change terraform.PlanChange) string {
	symbol := actionSymbol(change.Action)
	address := change.Resource.Address
	reason := riskReason(change)

	row := fmt.Sprintf("   %s %s", symbol, address)
	if reason != "" {
		row += "  " + styles.StyleFaint.Render(reason)
	}
	return row
}

func (e *Plugin) renderStats() string {
	var parts []string
	for _, g := range e.groups {
		parts = append(parts, fmt.Sprintf("%s: %d", g.Level.String(), len(g.Changes)))
	}
	return styles.StyleFaint.Render(fmt.Sprintf("Total: %d changes  [%s]", e.total, strings.Join(parts, " | ")))
}

func riskReason(change terraform.PlanChange) string {
	switch {
	case change.Action == terraform.ActionDelete || change.Action == terraform.ActionDeleteThenCreate:
		return "destructive operation"
	case change.Action == terraform.ActionUpdate && change.Risk >= terraform.RiskHigh:
		return "modification of critical resource"
	case change.IsPhantom:
		return "phantom change (cosmetic only)"
	default:
		return ""
	}
}

func actionSymbol(action terraform.Action) string {
	switch action {
	case terraform.ActionCreate:
		return styles.StyleCreate.Render("+")
	case terraform.ActionUpdate:
		return styles.StyleUpdate.Render("~")
	case terraform.ActionDelete:
		return styles.StyleDelete.Render("-")
	case terraform.ActionDeleteThenCreate, terraform.ActionCreateThenDelete:
		return styles.StyleReplace.Render("-/+")
	default:
		return " "
	}
}
