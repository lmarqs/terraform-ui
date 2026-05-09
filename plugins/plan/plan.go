package plan

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

// Status represents the current state of the plan plugin.
type Status int

const (
	StatusIdle Status = iota
	StatusLoading
	StatusDone
	StatusError
)

// PlanResultMsg is sent when the plan operation completes.
type PlanResultMsg struct {
	Summary *terraform.PlanSummary
	Err     error
}

// Plugin implements the plan review feature.
type Plugin struct {
	svc      terraform.Service
	status   Status
	summary  *terraform.PlanSummary
	errMsg   string
	selected int
	targets  []string
	expanded map[int]bool
}

// New creates a new plan plugin.
func New(svc terraform.Service) plugin.Plugin {
	return &Plugin{
		expanded: make(map[int]bool),
		svc:      svc,
	}
}

func (e *Plugin) ID() string          { return "plan" }
func (e *Plugin) Name() string        { return "Plan" }
func (e *Plugin) Description() string { return "Review terraform plan changes" }
func (e *Plugin) KeyBinding() string  { return "p" }
func (e *Plugin) Ready() bool         { return e.status == StatusDone }
func (e *Plugin) Status() Status      { return e.status }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) Targets() []string   { return e.targets }
func (e *Plugin) Summary() *terraform.PlanSummary {
	return e.summary
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// SetTargets configures resource targets for the plan.
func (e *Plugin) SetTargets(targets []string) {
	e.targets = targets
}

// Init initializes the plugin with shared context and triggers a plan.
func (e *Plugin) Init(ctx *plugin.Context) tea.Cmd {
	e.svc = ctx.Service
	e.status = StatusLoading
	e.summary = nil
	e.errMsg = ""
	e.selected = 0
	e.expanded = make(map[int]bool)
	return e.runPlan()
}

// Refresh re-runs the plan.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = StatusLoading
	e.summary = nil
	e.errMsg = ""
	e.selected = 0
	e.expanded = make(map[int]bool)
	return e.runPlan()
}

func (e *Plugin) runPlan() tea.Cmd {
	svc := e.svc
	targets := e.targets
	return func() tea.Msg {
		summary, err := svc.Plan(context.Background(), targets)
		return PlanResultMsg{Summary: summary, Err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case PlanResultMsg:
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
		} else {
			e.status = StatusDone
			e.summary = msg.Summary
		}
		return e, nil

	case tea.KeyMsg:
		cmd := e.handleKey(msg)
		return e, cmd
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		e.MoveDown()
	case "k", "up":
		e.MoveUp()
	case "enter", " ":
		e.ToggleExpand()
	case "r":
		if e.status == StatusError || e.status == StatusDone {
			return e.Refresh()
		}
	case "G":
		e.MoveToEnd()
	case "g":
		e.MoveToStart()
	}
	return nil
}

// MoveUp moves selection up.
func (e *Plugin) MoveUp() {
	if e.selected > 0 {
		e.selected--
	}
}

// MoveDown moves selection down.
func (e *Plugin) MoveDown() {
	if e.summary != nil && e.selected < len(e.summary.Changes)-1 {
		e.selected++
	}
}

// MoveToStart moves selection to the first item.
func (e *Plugin) MoveToStart() {
	e.selected = 0
}

// MoveToEnd moves selection to the last item.
func (e *Plugin) MoveToEnd() {
	if e.summary != nil && len(e.summary.Changes) > 0 {
		e.selected = len(e.summary.Changes) - 1
	}
}

// ToggleExpand toggles attribute diff expansion for the selected change.
func (e *Plugin) ToggleExpand() {
	e.expanded[e.selected] = !e.expanded[e.selected]
}

// IsExpanded returns whether a change row is expanded.
func (e *Plugin) IsExpanded(idx int) bool {
	return e.expanded[idx]
}

// SelectedChange returns the currently selected change, if any.
func (e *Plugin) SelectedChange() *terraform.PlanChange {
	if e.summary == nil || e.selected >= len(e.summary.Changes) {
		return nil
	}
	return &e.summary.Changes[e.selected]
}

// View renders the plan plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case StatusIdle:
		title := styles.StyleTitle.Render("Plan Review")
		placeholder := styles.StyleFaintItalic.Render("Press Enter to run terraform plan...")
		return styles.StylePadded.Render(title + "\n\n" + placeholder)

	case StatusLoading:
		title := styles.StyleTitle.Render("Plan Review")
		loading := styles.StyleFaintItalic.Render("Running terraform plan...")
		return styles.StylePadded.Render(title + "\n\n" + loading)

	case StatusError:
		title := styles.StyleTitle.Render("Plan Review")
		errText := styles.StyleError.Render("Error: " + e.errMsg)
		hint := styles.StyleFaintItalic.Render("Press r to retry, Esc to go back")
		return styles.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	case StatusDone:
		return e.renderResults(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderResults(width, height int) string {
	title := styles.StyleTitle.Render("Plan Review")

	if e.summary == nil || len(e.summary.Changes) == 0 {
		noChanges := styles.StyleSuccess.Render("No changes. Infrastructure is up-to-date.")
		return styles.StylePadded.Render(title + "\n\n" + noChanges)
	}

	var b strings.Builder

	// Calculate visible area (title + summary + hint take ~5 lines)
	maxVisible := height - 7
	if maxVisible < 3 {
		maxVisible = 3
	}

	// Determine scroll window
	startIdx := 0
	if e.selected >= maxVisible {
		startIdx = e.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(e.summary.Changes) {
		endIdx = len(e.summary.Changes)
	}

	for i := startIdx; i < endIdx; i++ {
		change := e.summary.Changes[i]
		row := e.renderChangeRow(change, width)
		if i == e.selected {
			row = styles.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')

		// Render expanded attribute diffs
		if e.expanded[i] && len(change.AttributeDiffs) > 0 {
			b.WriteString(e.renderAttributeDiffs(change.AttributeDiffs, width))
		}
	}

	summary := e.renderSummaryLine()
	riskLine := e.renderOverallRisk()
	hint := styles.StyleFaintItalic.Render("j/k navigate  Enter expand  r refresh  a apply  Esc back")

	content := title + "\n\n" + b.String() + "\n" + summary
	if riskLine != "" {
		content += "\n" + riskLine
	}
	content += "\n" + hint
	return styles.StylePadded.Render(content)
}

func (e *Plugin) renderChangeRow(change terraform.PlanChange, width int) string {
	symbol := actionSymbol(change.Action)
	address := change.Resource.Address
	risk := riskBadge(change.Risk)

	if change.IsPhantom {
		address = styles.StylePhantom.Render(address)
		symbol = styles.StylePhantom.Render(symbol)
	}

	expandIndicator := " "
	if len(change.AttributeDiffs) > 0 {
		if e.expanded[e.selected] {
			expandIndicator = "v"
		} else {
			expandIndicator = ">"
		}
	}

	row := fmt.Sprintf(" %s %s %s", expandIndicator, symbol, address)
	if risk != "" {
		row += " " + risk
	}
	if change.IsPhantom {
		row += " " + styles.StylePhantom.Render("(phantom)")
	}
	return row
}

func (e *Plugin) renderAttributeDiffs(diffs []terraform.AttributeDiff, width int) string {
	var b strings.Builder
	for _, diff := range diffs {
		key := styles.StyleKey.Render("    " + diff.Key + ":")
		if diff.Sensitive {
			b.WriteString(key + " " + styles.StyleFaintItalic.Render("(sensitive)") + "\n")
			continue
		}
		old := styles.StyleDelete.Render(truncateValue(diff.OldValue, width/3))
		new := styles.StyleCreate.Render(truncateValue(diff.NewValue, width/3))
		b.WriteString(key + " " + old + " -> " + new + "\n")
	}
	return b.String()
}

func truncateValue(s string, maxLen int) string {
	if maxLen < 10 {
		maxLen = 10
	}
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func (e *Plugin) renderSummaryLine() string {
	s := e.summary
	parts := []string{}
	if s.ToCreate > 0 {
		parts = append(parts, styles.StyleCreate.Render(fmt.Sprintf("%d to add", s.ToCreate)))
	}
	if s.ToUpdate > 0 {
		parts = append(parts, styles.StyleUpdate.Render(fmt.Sprintf("%d to change", s.ToUpdate)))
	}
	if s.ToDelete > 0 {
		parts = append(parts, styles.StyleDelete.Render(fmt.Sprintf("%d to destroy", s.ToDelete)))
	}
	if s.ToReplace > 0 {
		parts = append(parts, styles.StyleReplace.Render(fmt.Sprintf("%d to replace", s.ToReplace)))
	}

	if len(parts) == 0 {
		return styles.StyleFaint.Render("Plan: no changes")
	}
	return "Plan: " + strings.Join(parts, ", ")
}

func (e *Plugin) renderOverallRisk() string {
	if e.summary == nil || len(e.summary.Changes) == 0 {
		return ""
	}
	overall := terraform.OverallRisk(e.summary.Changes)
	switch overall {
	case terraform.RiskCritical:
		return styles.StyleRiskCritical.Render("Overall risk: CRITICAL")
	case terraform.RiskHigh:
		return styles.StyleRiskHigh.Render("Overall risk: HIGH")
	case terraform.RiskMedium:
		return styles.StyleRiskMedium.Render("Overall risk: medium")
	case terraform.RiskLow:
		return styles.StyleRiskLow.Render("Overall risk: low")
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
	case terraform.ActionRead:
		return styles.StyleFaint.Render("<=")
	default:
		return " "
	}
}

func riskBadge(risk terraform.RiskLevel) string {
	switch risk {
	case terraform.RiskLow:
		return styles.StyleRiskLow.Render("[low]")
	case terraform.RiskMedium:
		return styles.StyleRiskMedium.Render("[medium]")
	case terraform.RiskHigh:
		return styles.StyleRiskHigh.Render("[HIGH]")
	case terraform.RiskCritical:
		return styles.StyleRiskCritical.Render("[CRITICAL]")
	default:
		return ""
	}
}
