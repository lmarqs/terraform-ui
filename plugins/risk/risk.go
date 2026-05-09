package risk

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Status represents the current state of the risk sdk.
type Status int

const (
	StatusIdle Status = iota
	StatusReady
)

// RiskGroup holds changes grouped by risk level.
type RiskGroup struct {
	Level   sdk.RiskLevel
	Changes []sdk.PlanChange
}

// Plugin implements the risk analysis feature.
type Plugin struct {
	svc      sdk.Service
	status   Status
	groups   []RiskGroup
	overall  sdk.RiskLevel
	selected int
	total    int
}

// New creates a new risk analysis sdk.
func New(svc sdk.Service) sdk.Plugin {
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
func (e *Plugin) Overall() sdk.RiskLevel {
	return e.overall
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	return nil
}

// Analyze processes a plan summary and groups changes by risk.
func (e *Plugin) Analyze(summary *sdk.PlanSummary) {
	if summary == nil || len(summary.Changes) == 0 {
		e.status = StatusReady
		e.groups = nil
		e.overall = sdk.RiskNone
		e.total = 0
		return
	}

	e.overall = sdk.OverallRisk(summary.Changes)
	e.total = len(summary.Changes)

	// Group changes by risk level (highest first)
	byLevel := map[sdk.RiskLevel][]sdk.PlanChange{
		sdk.RiskCritical: {},
		sdk.RiskHigh:     {},
		sdk.RiskMedium:   {},
		sdk.RiskLow:      {},
		sdk.RiskNone:     {},
	}

	for _, c := range summary.Changes {
		byLevel[c.Risk] = append(byLevel[c.Risk], c)
	}

	e.groups = make([]RiskGroup, 0)
	levels := []sdk.RiskLevel{
		sdk.RiskCritical,
		sdk.RiskHigh,
		sdk.RiskMedium,
		sdk.RiskLow,
		sdk.RiskNone,
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

// Update processes messages and returns the updated sdk.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
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

// View renders the risk analysis sdk.
func (e *Plugin) View(width, height int) string {
	title := sdk.StyleTitle.Render("Risk Analysis")

	switch e.status {
	case StatusIdle:
		placeholder := sdk.StyleFaintItalic.Render("Run a plan first to analyze risk...")
		return sdk.StylePadded.Render(title + "\n\n" + placeholder)

	case StatusReady:
		return e.renderAnalysis(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderAnalysis(width, height int) string {
	title := sdk.StyleTitle.Render("Risk Analysis")

	if len(e.groups) == 0 {
		noRisk := sdk.StyleSuccess.Render("No changes to analyze.")
		return sdk.StylePadded.Render(title + "\n\n" + noRisk)
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
				header = sdk.StyleSelected.Width(width - 6).Render(header)
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
				row = sdk.StyleSelected.Width(width - 6).Render(row)
			}
			b.WriteString(row)
			b.WriteByte('\n')
			itemIdx++
		}
		b.WriteByte('\n')
	}

	// Statistics
	stats := e.renderStats()
	hint := sdk.StyleFaintItalic.Render("j/k navigate  Esc back")

	content := title + "\n\n" + b.String() + stats + "\n" + hint
	return sdk.StylePadded.Render(content)
}

func (e *Plugin) renderOverallBanner() string {
	switch e.overall {
	case sdk.RiskCritical:
		return sdk.StyleRiskCritical.Render("!! CRITICAL RISK DETECTED !!")
	case sdk.RiskHigh:
		return sdk.StyleRiskHigh.Render("! HIGH RISK DETECTED !")
	case sdk.RiskMedium:
		return sdk.StyleRiskMedium.Render("Medium risk - review recommended")
	case sdk.RiskLow:
		return sdk.StyleRiskLow.Render("Low risk - changes look safe")
	default:
		return sdk.StyleSuccess.Render("No risk detected")
	}
}

func (e *Plugin) renderGroupHeader(group RiskGroup) string {
	var label string
	switch group.Level {
	case sdk.RiskCritical:
		label = sdk.StyleRiskCritical.Render(fmt.Sprintf("CRITICAL (%d)", len(group.Changes)))
	case sdk.RiskHigh:
		label = sdk.StyleRiskHigh.Render(fmt.Sprintf("HIGH (%d)", len(group.Changes)))
	case sdk.RiskMedium:
		label = sdk.StyleRiskMedium.Render(fmt.Sprintf("MEDIUM (%d)", len(group.Changes)))
	case sdk.RiskLow:
		label = sdk.StyleRiskLow.Render(fmt.Sprintf("LOW (%d)", len(group.Changes)))
	default:
		label = sdk.StyleFaint.Render(fmt.Sprintf("NONE (%d)", len(group.Changes)))
	}
	return "--- " + label + " ---"
}

func (e *Plugin) renderChangeRow(change sdk.PlanChange) string {
	symbol := actionSymbol(change.Action)
	address := change.Resource.Address
	reason := riskReason(change)

	row := fmt.Sprintf("   %s %s", symbol, address)
	if reason != "" {
		row += "  " + sdk.StyleFaint.Render(reason)
	}
	return row
}

func (e *Plugin) renderStats() string {
	var parts []string
	for _, g := range e.groups {
		parts = append(parts, fmt.Sprintf("%s: %d", g.Level.String(), len(g.Changes)))
	}
	return sdk.StyleFaint.Render(fmt.Sprintf("Total: %d changes  [%s]", e.total, strings.Join(parts, " | ")))
}

func riskReason(change sdk.PlanChange) string {
	switch {
	case change.Action == sdk.ActionDelete || change.Action == sdk.ActionDeleteThenCreate:
		return "destructive operation"
	case change.Action == sdk.ActionUpdate && change.Risk >= sdk.RiskHigh:
		return "modification of critical resource"
	case change.IsPhantom:
		return "phantom change (cosmetic only)"
	default:
		return ""
	}
}

func actionSymbol(action sdk.Action) string {
	switch action {
	case sdk.ActionCreate:
		return sdk.StyleCreate.Render("+")
	case sdk.ActionUpdate:
		return sdk.StyleUpdate.Render("~")
	case sdk.ActionDelete:
		return sdk.StyleDelete.Render("-")
	case sdk.ActionDeleteThenCreate, sdk.ActionCreateThenDelete:
		return sdk.StyleReplace.Render("-/+")
	default:
		return " "
	}
}
