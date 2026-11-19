package risk

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// riskItem is one reviewed change joined to its assessment row.
type riskItem struct {
	change sdk.PlanChange
	row    sdk.PolicyResourceAssessment
}

// riskGroup holds the items classified at one level, or those whose assessment failed.
type riskGroup struct {
	level       sdk.RiskLevel
	unavailable bool
	items       []riskItem
}

// Plugin implements the risk analysis feature.
type Plugin struct {
	sdk.PluginBase
	stack           *sdk.Stack
	panel           *ui.ContentPanel
	offset          int
	outcome         outcome
	groups          []riskGroup
	items           []riskItem
	selected        int
	summary         *sdk.PlanSummary
	assessment      *sdk.PolicyAssessment
	requestRisk     func(*sdk.PlanSummary) (sdk.AssessmentID, tea.Cmd)
	pending         sdk.AssessmentID
	requested       *sdk.Context
	input           Input
	configured      *sdk.RiskLevel
	threshold       *sdk.RiskLevel
	settingsInvalid bool
	timer           ui.Timer
}

// New creates a new risk analysis plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{PluginBase: sdk.NewPluginBase("risk", "Risk Analysis", "Analyze risk levels of planned changes")}
	p.Svc = svc
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	p.panel = ui.NewContentPanel()
	p.panel.SelectedStyle = func(s string, width int) string { return sdk.StyleSelected.Width(width).Render(s) }
	return p
}

func (e *Plugin) Ready() bool   { return e.outcome.kind > outcomeLoading }
func (e *Plugin) Selected() int { return e.selected }

func (e *Plugin) Status() sdk.Status {
	switch e.outcome.kind {
	case outcomeIdle:
		return sdk.StatusIdle
	case outcomeLoading:
		return sdk.StatusLoading
	case outcomeDone:
		return sdk.StatusDone
	}
	return sdk.StatusError
}

// overall is the assessment's aggregate level, which only a complete assessment has.
func (e *Plugin) overall() sdk.RiskLevel {
	if e.assessment.OverallLevel == nil {
		return sdk.RiskNone
	}
	return *e.assessment.OverallLevel
}

// Hints returns context-sensitive key hints for the status bar.
func (e *Plugin) Hints() []sdk.KeyHint { return e.stack.Hints() }

func (e *Plugin) Stack() *sdk.Stack { return e.stack }

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return e.configureReport(cfg)
}

// Init wires the plugin to its shared dependencies.
func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	e.InitBase(deps)
	e.requestRisk = deps.RequestAssessment
	return nil
}

// Activate requests the host assessment of the retained reviewed plan, or of a
// plan the host acquires when none is retained, with input's output and
// comparison options.
func (e *Plugin) Activate(input Input) tea.Cmd {
	if e.outcome.kind == outcomeLoading {
		return nil
	}
	e.input = input
	e.resetDerived()
	e.threshold = e.configured
	// The host never sees the CLI threshold. Invalid settings defer to the host,
	// whose answer carries their hint.
	if input.Threshold != nil && !e.settingsInvalid {
		threshold, valid := parseThreshold(*input.Threshold)
		if !valid {
			e.outcome = outcome{kind: outcomeInvalidThreshold, diagnostic: unavailable(string(sdk.ErrorConfigurationInvalid))}
			return nil
		}
		e.threshold = threshold
	}
	return e.requestAssessment()
}

// analyze joins each change to its assessment row and groups the changes the
// assessment classified or failed, failures first, then by level.
func (e *Plugin) analyze(summary *sdk.PlanSummary, rows sdk.AssessmentRows) {
	var unavailable []riskItem
	byLevel := map[sdk.RiskLevel][]riskItem{}
	if summary != nil {
		for _, change := range summary.Changes {
			row, ok := rows[change.Resource.InstanceKey()]
			switch {
			case !ok:
			case row.Status == sdk.ResourceUnavailable:
				unavailable = append(unavailable, riskItem{change: change, row: row})
			case row.Level != nil:
				byLevel[*row.Level] = append(byLevel[*row.Level], riskItem{change: change, row: row})
			}
		}
	}
	groups := []riskGroup{{unavailable: true, items: unavailable}}
	for _, level := range []sdk.RiskLevel{sdk.RiskCritical, sdk.RiskHigh, sdk.RiskMedium, sdk.RiskLow, sdk.RiskNone} {
		groups = append(groups, riskGroup{level: level, items: byLevel[level]})
	}
	e.groups, e.items = nil, nil
	for _, group := range groups {
		if len(group.items) > 0 {
			slices.SortFunc(group.items, compareItems)
			e.groups = append(e.groups, group)
			e.items = append(e.items, group.items...)
		}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return e, e.timer.Tick()
	case tea.KeyMsg:
		return e, e.stack.Update(msg)
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "j", "down":
		e.moveDown()
	case "k", "up":
		e.moveUp()
	case "g":
		e.selected = 0
	case "G":
		e.selected = max(0, len(e.items)-1)
	case "left", "right", "ctrl+w":
		if e.navigable() {
			e.panel.HandleKey(msg)
		}
	}
}

// navigable reports whether the list shows rows to move through and pan.
func (e *Plugin) navigable() bool { return e.Ready() && len(e.items) > 0 }

func (e *Plugin) moveUp() {
	if e.selected > 0 {
		e.selected--
	}
}

func (e *Plugin) moveDown() {
	if e.selected < len(e.items)-1 {
		e.selected++
	}
}

func (e *Plugin) CursorPosition() (int, int) {
	if !e.Ready() || len(e.items) == 0 {
		return 0, 0
	}
	return e.selected + 1, len(e.items)
}

// View renders the risk analysis plugin.
func (e *Plugin) View(width, height int) string {
	if e.stack.Depth() > 1 {
		return e.stack.View(width, height)
	}
	switch e.outcome.kind {
	case outcomeIdle:
		return sdk.StyleFaintItalic.Render("Run a plan first to analyze risk...")
	case outcomeLoading:
		return sdk.StyleFaintItalic.Render("Loading risk assessment... " + e.timer.FormatElapsed())
	case outcomeDone:
		return e.renderAnalysis(width, height)
	case outcomeUnusable:
		if len(e.items) > 0 {
			return e.renderAnalysis(width, height)
		}
		return sdk.StyleError.Render(e.outcome.diagnostic)
	case outcomeInvalidThreshold, outcomePlanFailed:
		return sdk.StyleError.Render(e.outcome.diagnostic)

	default:
		return ""
	}
}

func (e *Plugin) renderAnalysis(width, height int) string {
	if len(e.items) == 0 {
		return sdk.StyleFaint.Render(fmt.Sprintf("No eligible mutations (%d changes not assessed).", countRows(e.assessment).notAssessed))
	}

	rows := []string{}
	selected := min(e.selected, len(e.items)-1)
	index, start, end := 0, 0, 0
	for _, group := range e.groups {
		rows = append(rows, e.renderGroupHeader(group))
		for _, item := range group.items {
			if index == selected {
				start = len(rows)
			}
			rows = append(rows, e.renderChangeRow(item))
			rows = append(rows, sdk.RenderCompactAttributeDiffs(item.change.AttributeDiffs, max(0, width-ui.GutterWidth))...)
			if index == selected {
				end = len(rows)
			}
			index++
		}
	}
	content, offset := e.panel.RenderWindow(ui.RenderParams{Rows: rows, Width: width, Height: max(1, height-4), Cursor: start, CursorEnd: end, ScrollOffset: e.offset})
	e.offset = offset
	banner := ansi.Truncate(e.renderOverallBanner(), max(0, width), "")
	stats := ansi.Truncate(e.renderStats(), max(0, width), "")
	result := banner + "\n\n" + content + "\n\n" + stats
	lines := strings.Split(result, "\n")
	return strings.Join(lines[:min(max(0, height), len(lines))], "\n")
}

func (e *Plugin) renderOverallBanner() string {
	if e.assessment.Status == sdk.AssessmentPartial || e.assessment.Status == sdk.AssessmentUnavailable {
		counts := countRows(e.assessment)
		message := fmt.Sprintf("Risk unavailable: %d of %d eligible mutations failed", counts.unavailable, counts.unavailable+counts.classified)
		if counts.classified > 0 {
			message += fmt.Sprintf("; %d classified", counts.classified)
		}
		return sdk.StyleError.Render(message + ".")
	}
	return "Risk: " + e.overall().String() + " — explicit policy classifications."
}

func (e *Plugin) renderGroupHeader(group riskGroup) string {
	if group.unavailable {
		return "--- " + sdk.StyleError.Render(fmt.Sprintf("Risk unavailable (%d)", len(group.items))) + " ---"
	}
	var label string
	switch group.level {
	case sdk.RiskCritical:
		label = sdk.StyleRiskCritical.Render(fmt.Sprintf("CRITICAL (%d)", len(group.items)))
	case sdk.RiskHigh:
		label = sdk.StyleRiskHigh.Render(fmt.Sprintf("HIGH (%d)", len(group.items)))
	case sdk.RiskMedium:
		label = sdk.StyleRiskMedium.Render(fmt.Sprintf("MEDIUM (%d)", len(group.items)))
	case sdk.RiskLow:
		label = sdk.StyleRiskLow.Render(fmt.Sprintf("LOW (%d)", len(group.items)))
	default:
		label = sdk.StyleFaint.Render(fmt.Sprintf("NONE (%d)", len(group.items)))
	}
	return "--- " + label + " ---"
}

func (e *Plugin) renderChangeRow(item riskItem) string {
	reason := ""
	if item.row.Status == sdk.ResourceUnavailable {
		reason = "risk unavailable: " + string(item.row.ErrorCode)
	}
	for _, evidence := range item.row.Reasons {
		if reason != "" {
			reason += "; "
		}
		reason += evidence.Message
	}

	row := fmt.Sprintf("   %s %s", sdk.ActionSymbol(item.change.Action), item.change.Resource.Address)
	if reason != "" {
		row += "  " + sdk.StyleFaint.Render(reason)
	}
	return row
}

func (e *Plugin) renderStats() string {
	var parts []string
	for _, g := range e.groups {
		label := g.level.String()
		if g.unavailable {
			label = string(sdk.ResourceUnavailable)
		}
		parts = append(parts, fmt.Sprintf("%s: %d", label, len(g.items)))
	}
	return sdk.StyleFaint.Render(fmt.Sprintf("Total: %d changes  [%s]", len(e.items), strings.Join(parts, " | ")))
}

func compareItems(x, y riskItem) int {
	return x.change.Resource.InstanceKey().Compare(y.change.Resource.InstanceKey())
}

// rowCounts is how many of an assessment's rows were classified, failed or left out.
type rowCounts struct {
	classified, unavailable, notAssessed int
}

// countRows counts a.Resources by status; a nil assessment has no rows.
func countRows(a *sdk.PolicyAssessment) rowCounts {
	var counts rowCounts
	if a == nil {
		return counts
	}
	for _, row := range a.Resources {
		switch row.Status {
		case sdk.ResourceClassified:
			counts.classified++
		case sdk.ResourceUnavailable:
			counts.unavailable++
		case sdk.ResourceNotAssessed:
			counts.notAssessed++
		}
	}
	return counts
}
