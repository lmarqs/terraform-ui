package risk

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// RiskGroup holds changes grouped by risk level.
type RiskGroup struct {
	Level   sdk.RiskLevel
	Changes []sdk.PlanChange
}

// Plugin implements the risk analysis feature.
type Plugin struct {
	sdk.PluginBase
	status         sdk.Status
	groups         []RiskGroup
	overall        sdk.RiskLevel
	selected       int
	total          int
	riskDiagnostic string
	summary        *sdk.PlanSummary
	assessment     *sdk.PolicyAssessment
	assess         func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment
	generation     uint64
	cancel         context.CancelFunc
	planError      string
}

// New creates a new risk analysis plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{PluginBase: sdk.NewPluginBase("risk", "Risk Analysis", "Analyze risk levels of planned changes")}
	p.Svc = svc
	return p
}

func (e *Plugin) Ready() bool        { return e.status == sdk.StatusDone || e.status == sdk.StatusError }
func (e *Plugin) Status() sdk.Status { return e.status }
func (e *Plugin) Selected() int      { return e.selected }
func (e *Plugin) Overall() sdk.RiskLevel {
	return e.overall
}

// Hints returns context-sensitive key hints for the status bar.
func (e *Plugin) Hints() []sdk.KeyHint {
	if e.status == sdk.StatusDone && len(e.groups) > 0 {
		return (sdk.HintSetQuit).Hints()
	}
	return (sdk.HintSetQuit).Hints()
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init wires the plugin to its shared dependencies.
func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	e.InitBase(deps)
	e.assess = deps.AssessRisk
	return nil
}

// Activate assesses the retained reviewed plan, acquiring one only when absent.
func (e *Plugin) Activate() tea.Cmd {
	if e.Busy() {
		return nil
	}
	e.resetDerived()
	if e.checkPolicySource() {
		return nil
	}
	return e.startAssessment(e.summary)
}

func (e *Plugin) checkPolicySource() bool {
	e.riskDiagnostic = e.PolicyDiagnostic()
	if e.riskDiagnostic == "" {
		return false
	}
	e.status = sdk.StatusError
	e.groups = nil
	e.overall = sdk.RiskNone
	e.selected = 0
	e.total = 0
	return true
}

func (e *Plugin) Stderr() []byte {
	if e.planError != "" {
		return []byte(e.planError + "\n")
	}
	if e.riskDiagnostic == "" {
		return nil
	}
	return []byte("risk unavailable: " + e.riskDiagnostic + "\n")
}

func (e *Plugin) ExitCode() int {
	if e.status == sdk.StatusError {
		return 1
	}
	return 0
}

// Analyze processes a plan summary and groups changes by risk.
func (e *Plugin) Analyze(summary *sdk.PlanSummary) {
	if e.checkPolicySource() {
		return
	}
	if summary == nil || len(summary.Changes) == 0 {
		e.status = sdk.StatusDone
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

	e.status = sdk.StatusDone
	e.selected = 0
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case assessmentResultMsg:
		e.acceptAssessment(msg)
		return e, nil
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

func (e *Plugin) CursorPosition() (int, int) {
	if e.status != sdk.StatusDone || len(e.groups) == 0 {
		return 0, 0
	}
	return e.selected + 1, e.totalItems()
}

// View renders the risk analysis plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Run a plan first to analyze risk...")
	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Loading risk assessment...")

	case sdk.StatusDone:
		return e.renderAnalysis(width, height)
	case sdk.StatusError:
		if e.planError != "" {
			return sdk.StyleError.Render(e.planError)
		}
		return sdk.StyleError.Render("risk unavailable: " + e.riskDiagnostic)

	default:
		return ""
	}
}

func (e *Plugin) renderAnalysis(width, height int) string {
	if len(e.groups) == 0 {
		return sdk.StyleSuccess.Render("No changes to analyze.")
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
				header = sdk.StyleSelected.Width(width).Render(header)
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
				row = sdk.StyleSelected.Width(width).Render(row)
			}
			b.WriteString(row)
			b.WriteByte('\n')
			itemIdx++
		}
		b.WriteByte('\n')
	}

	stats := e.renderStats()

	return b.String() + stats
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
	symbol := sdk.ActionSymbol(change.Action)
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
