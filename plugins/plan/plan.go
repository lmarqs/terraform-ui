package plan

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// PlanResultMsg is sent when the plan operation completes.
type PlanResultMsg struct {
	Summary *sdk.PlanSummary
	Err     error
}

// Plugin implements the plan review feature.
type Plugin struct {
	svc           sdk.Service
	log           *slog.Logger
	options       *sdk.ResolvedOptions
	stack         *sdk.Stack
	pins          *sdk.PinService
	expander      *ui.ExpandSet
	status        sdk.Status
	summary       *sdk.PlanSummary
	errMsg        string
	lockInfo      *sdk.StateLock
	selected      int
	targets       []string
	scopedContext string // tracks which context the service was scoped to
}

// New creates a new plan plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		expander: ui.NewExpandSet(),
		svc:      svc,
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	return p
}

func (e *Plugin) ID() string          { return "plan" }
func (e *Plugin) Name() string        { return "Plan" }
func (e *Plugin) Description() string { return "Review terraform plan changes" }
func (e *Plugin) Ready() bool         { return e.status == sdk.StatusDone }
func (e *Plugin) Status() sdk.Status  { return e.status }
func (e *Plugin) Busy() bool          { return e.status == sdk.StatusLoading }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) Targets() []string   { return e.targets }
func (e *Plugin) Stack() *sdk.Stack   { return e.stack }
func (e *Plugin) Summary() *sdk.PlanSummary {
	return e.summary
}
func (e *Plugin) Count() (int, int) {
	if e.summary == nil {
		return 0, 0
	}
	return len(e.summary.Changes), len(e.summary.Changes)
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// SetTargets configures resource targets for the plan.
func (e *Plugin) SetTargets(targets []string) {
	e.targets = targets
}

// Init initializes the plugin with shared context. Does not auto-run plan —
// the user must explicitly activate the plugin to trigger a plan.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	e.log = ctx.Logger
	e.options = ctx.Options
	e.pins = ctx.Pins
	e.reset()
	return nil
}

// HandleLockCleared implements sdk.LockClearedHandler.
func (e *Plugin) HandleLockCleared(_ sdk.LockClearedEvent) tea.Cmd {
	e.lockInfo = nil
	return e.Refresh()
}

// HandleChdirChanged implements sdk.ChdirHandler.
func (e *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	e.svc = e.svc.WithDir(evt.AbsPath)
	e.scopedContext = evt.AbsPath
	e.reset()
	return nil
}

// HandlePlanInvalidated implements sdk.PlanInvalidatedHandler.
func (e *Plugin) HandlePlanInvalidated(_ sdk.PlanInvalidatedEvent) tea.Cmd {
	e.reset()
	return nil
}

// reset clears all plugin state to initial values.
func (e *Plugin) reset() {
	e.status = sdk.StatusIdle
	e.summary = nil
	e.errMsg = ""
	e.selected = 0
	e.expander = ui.NewExpandSet()
}

// Activate triggers the plan when the user enters the plugin view.
func (e *Plugin) Activate() tea.Cmd {
	if e.status == sdk.StatusIdle || e.status == sdk.StatusError {
		e.status = sdk.StatusLoading
		e.log.Debug("plan.start", "targets", e.targets)
		return e.runPlan()
	}
	return nil
}

// Refresh re-runs the plan.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = sdk.StatusLoading
	e.summary = nil
	e.errMsg = ""
	e.lockInfo = nil
	e.selected = 0
	e.expander = ui.NewExpandSet()
	return e.runPlan()
}

func (e *Plugin) runPlan() tea.Cmd {
	svc := e.svc
	opts := sdk.BuildPlanOptions(e.options, e.targets)
	return func() tea.Msg {
		summary, err := svc.Plan(context.Background(), opts)
		return PlanResultMsg{Summary: summary, Err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case PlanResultMsg:
		if msg.Err != nil {
			e.status = sdk.StatusError
			e.errMsg = msg.Err.Error()
			e.lockInfo = sdk.ParseLockError(e.errMsg)
			e.log.Debug("plan.error", "error", msg.Err.Error())
			if e.lockInfo != nil {
				return e, func() tea.Msg { return sdk.LockDetectedEvent{Lock: e.lockInfo} }
			}
		} else {
			e.status = sdk.StatusDone
			e.summary = msg.Summary
			changes := 0
			if msg.Summary != nil {
				changes = len(msg.Summary.Changes)
			}
			e.log.Debug("plan.complete", "changes", changes)
			if msg.Summary != nil {
				return e, tea.Batch(
					func() tea.Msg {
						return sdk.PlanCompletedEvent{
							Summary:       msg.Summary,
							ResourceCount: changes,
						}
					},
					func() tea.Msg { return sdk.StateRefreshedEvent{} },
				)
			}
			return e, func() tea.Msg { return sdk.StateRefreshedEvent{} }
		}
		return e, nil

	}
	return e, nil
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
	e.expander.Toggle(e.selected)
}

// IsExpanded returns whether a change row is expanded.
func (e *Plugin) IsExpanded(idx int) bool {
	return e.expander.IsExpanded(idx)
}

// SelectedChange returns the currently selected change, if any.
func (e *Plugin) SelectedChange() *sdk.PlanChange {
	if e.summary == nil || e.selected >= len(e.summary.Changes) {
		return nil
	}
	return &e.summary.Changes[e.selected]
}

// View renders the plan plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Ready to plan.")

	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Running terraform plan...")

	case sdk.StatusError:
		if e.lockInfo != nil {
			return sdk.FormatLockInfo(e.lockInfo)
		}
		return sdk.StyleError.Render("Error: " + e.errMsg)

	case sdk.StatusDone:
		return e.renderResults(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderResults(width, height int) string {
	if e.summary == nil || len(e.summary.Changes) == 0 {
		return sdk.StyleSuccess.Render("No changes. Infrastructure is up-to-date.")
	}

	var b strings.Builder

	// Calculate visible area (summary + hint take ~5 lines)
	maxVisible := height - 5
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
			row = sdk.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')

		// Render expanded attribute diffs
		if e.expander.IsExpanded(i) && len(change.AttributeDiffs) > 0 {
			b.WriteString(e.renderAttributeDiffs(change.AttributeDiffs, width))
		}
	}

	summary := e.renderSummaryLine()
	riskLine := e.renderOverallRisk()

	content := b.String() + "\n" + summary
	if riskLine != "" {
		content += "\n" + riskLine
	}
	return content
}

func (e *Plugin) renderChangeRow(change sdk.PlanChange, width int) string {
	symbol := sdk.ActionSymbol(change.Action)
	address := change.Resource.Address
	risk := sdk.RiskBadge(change.Risk)

	if change.IsPhantom {
		address = sdk.StylePhantom.Render(address)
		symbol = sdk.StylePhantom.Render(symbol)
	}

	pinMark := " "
	if e.isPinnedAddress(change.Resource.Address) {
		pinMark = sdk.StyleSuccess.Render("*")
	}

	expandIndicator := " "
	if len(change.AttributeDiffs) > 0 {
		if e.expander.IsExpanded(e.selected) {
			expandIndicator = "v"
		} else {
			expandIndicator = ">"
		}
	}

	row := fmt.Sprintf(" %s%s %s %s", pinMark, expandIndicator, symbol, address)
	if risk != "" {
		row += " " + risk
	}
	if change.IsPhantom {
		row += " " + sdk.StylePhantom.Render("(phantom)")
	}
	return row
}

func (e *Plugin) renderAttributeDiffs(diffs []sdk.AttributeDiff, width int) string {
	var b strings.Builder
	for _, diff := range diffs {
		key := sdk.StyleKey.Render("    " + diff.Key + ":")
		if diff.Sensitive {
			b.WriteString(key + " " + sdk.StyleFaintItalic.Render("(sensitive)") + "\n")
			continue
		}
		old := sdk.StyleDelete.Render(sdk.Truncate(diff.OldValue, width/3))
		new := sdk.StyleCreate.Render(sdk.Truncate(diff.NewValue, width/3))
		b.WriteString(key + " " + old + " -> " + new + "\n")
	}
	return b.String()
}

func (e *Plugin) renderSummaryLine() string {
	s := e.summary
	parts := []string{}
	if s.ToCreate > 0 {
		parts = append(parts, sdk.StyleCreate.Render(fmt.Sprintf("%d to add", s.ToCreate)))
	}
	if s.ToUpdate > 0 {
		parts = append(parts, sdk.StyleUpdate.Render(fmt.Sprintf("%d to change", s.ToUpdate)))
	}
	if s.ToDelete > 0 {
		parts = append(parts, sdk.StyleDelete.Render(fmt.Sprintf("%d to destroy", s.ToDelete)))
	}
	if s.ToReplace > 0 {
		parts = append(parts, sdk.StyleReplace.Render(fmt.Sprintf("%d to replace", s.ToReplace)))
	}

	if len(parts) == 0 {
		return sdk.StyleFaint.Render("Plan: no changes")
	}
	return "Plan: " + strings.Join(parts, ", ")
}

func (e *Plugin) renderOverallRisk() string {
	if e.summary == nil || len(e.summary.Changes) == 0 {
		return ""
	}
	overall := sdk.OverallRisk(e.summary.Changes)
	switch overall {
	case sdk.RiskCritical:
		return sdk.StyleRiskCritical.Render("Overall risk: CRITICAL")
	case sdk.RiskHigh:
		return sdk.StyleRiskHigh.Render("Overall risk: HIGH")
	case sdk.RiskMedium:
		return sdk.StyleRiskMedium.Render("Overall risk: medium")
	case sdk.RiskLow:
		return sdk.StyleRiskLow.Render("Overall risk: low")
	default:
		return ""
	}
}

func (e *Plugin) togglePin(address string) {
	if e.pins != nil {
		e.pins.Toggle(address)
		e.log.Debug("plan.pin.toggle", "address", address)
	}
}

func (e *Plugin) isPinnedAddress(address string) bool {
	if e.pins != nil {
		return e.pins.IsPinned(address)
	}
	return false
}

// ApplyRequestMsg signals the app to start applying the plan.
type ApplyRequestMsg struct{}

func (e *Plugin) requestApply() tea.Cmd {
	msg := fmt.Sprintf("Apply plan (%d changes)?", len(e.summary.Changes))
	if n := e.pins.Count(); n > 0 {
		msg = fmt.Sprintf("Apply %d targeted resource(s)?", n)
	}
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				msg,
				func() tea.Cmd {
					return func() tea.Msg {
						return ApplyRequestMsg{}
					}
				},
			),
		}
	}
}
