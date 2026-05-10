package plan

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
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
	Summary *sdk.PlanSummary
	Err     error
}

// Plugin implements the plan review feature.
type Plugin struct {
	svc           sdk.Service
	log           *slog.Logger
	session       *sdk.Session
	stack         *sdk.Stack
	status        Status
	summary       *sdk.PlanSummary
	errMsg        string
	lockInfo      *sdk.StateLock
	selected      int
	targets       []string
	expanded      map[int]bool
	scopedContext string // tracks which context the service was scoped to
}

// New creates a new plan plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		expanded: make(map[int]bool),
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
func (e *Plugin) KeyBinding() string  { return "p" }
func (e *Plugin) Ready() bool         { return e.status == StatusDone }
func (e *Plugin) Status() Status      { return e.status }
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
	e.session = ctx.Session
	e.status = StatusIdle
	e.summary = nil
	e.errMsg = ""
	e.selected = 0
	e.expanded = make(map[int]bool)
	return nil
}

// Activate triggers the plan when the user enters the plugin view.
func (e *Plugin) Activate() tea.Cmd {
	// Check if the active context changed since last activation
	if e.session != nil {
		currentContext, _ := sdk.GetTyped[string](e.session, sdk.SessionKeyActiveContextAbs)
		if currentContext != e.scopedContext {
			// Context changed — reset state and re-scope
			e.status = StatusIdle
			e.summary = nil
			e.errMsg = ""
			e.selected = 0
			e.expanded = make(map[int]bool)
			e.scopedContext = currentContext
			if currentContext != "" {
				e.svc = e.svc.WithDir(currentContext)
			}
		}
	}

	if e.status == StatusIdle || e.status == StatusError {
		// Check if there's an active context to scope to
		if e.session != nil {
			if dir, ok := sdk.GetTyped[string](e.session, sdk.SessionKeyActiveContextAbs); ok && dir != "" {
				e.svc = e.svc.WithDir(dir)
				e.scopedContext = dir
			} else if count, ok := sdk.GetTyped[int](e.session, sdk.SessionKeyContextCount); ok && count > 1 {
				// Multi-context mode but no context selected
				e.status = StatusError
				e.errMsg = "Select a context first (press c)"
				return nil
			}
		}
		e.status = StatusLoading
		e.log.Debug("plan.start", "targets", e.targets)
		return e.runPlan()
	}
	return nil
}

// Refresh re-runs the plan.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = StatusLoading
	e.summary = nil
	e.errMsg = ""
	e.lockInfo = nil
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
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case PlanResultMsg:
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
			e.lockInfo = sdk.ParseLockError(e.errMsg)
			e.log.Debug("plan.error", "error", msg.Err.Error())
		} else {
			e.status = StatusDone
			e.summary = msg.Summary
			changes := 0
			if msg.Summary != nil {
				changes = len(msg.Summary.Changes)
			}
			e.log.Debug("plan.complete", "changes", changes)
			if e.session != nil && msg.Summary != nil {
				e.session.Set(sdk.SessionKeyPlanSummary, msg.Summary)
				e.session.Set(sdk.SessionKeyResourceCount, len(msg.Summary.Changes))
			}
		}
		return e, nil

	case ForceUnlockResultMsg:
		if msg.Err != nil {
			e.errMsg = fmt.Sprintf("Force-unlock failed: %s", msg.Err.Error())
			e.lockInfo = nil
			e.log.Debug("plan.force-unlock.error", "error", msg.Err.Error())
		} else {
			e.lockInfo = nil
			e.log.Debug("plan.force-unlock.success")
			return e, e.Refresh()
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
	e.expanded[e.selected] = !e.expanded[e.selected]
}

// IsExpanded returns whether a change row is expanded.
func (e *Plugin) IsExpanded(idx int) bool {
	return e.expanded[idx]
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
	case StatusIdle:
		return sdk.StyleFaintItalic.Render("Press Enter to run terraform plan...")

	case StatusLoading:
		return sdk.StyleFaintItalic.Render("Running terraform plan...")

	case StatusError:
		if e.lockInfo != nil {
			return sdk.FormatLockInfo(e.lockInfo)
		}
		return sdk.StyleError.Render("Error: " + e.errMsg)

	case StatusDone:
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
		if e.expanded[i] && len(change.AttributeDiffs) > 0 {
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
		if e.expanded[e.selected] {
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
	if e.session == nil {
		return
	}
	pinned, _ := sdk.GetTyped[[]string](e.session, "terraform.pinned")
	for i, a := range pinned {
		if a == address {
			pinned = append(pinned[:i], pinned[i+1:]...)
			e.session.Set("terraform.pinned", pinned)
			e.log.Debug("plan.unpin", "address", address)
			return
		}
	}
	pinned = append(pinned, address)
	e.session.Set("terraform.pinned", pinned)
	e.log.Debug("plan.pin", "address", address)
}

func (e *Plugin) isPinnedAddress(address string) bool {
	if e.session == nil {
		return false
	}
	pinned, _ := sdk.GetTyped[[]string](e.session, "terraform.pinned")
	for _, a := range pinned {
		if a == address {
			return true
		}
	}
	return false
}

// ForceUnlockResultMsg is sent when a force-unlock operation completes.
type ForceUnlockResultMsg struct {
	Err error
}

// ApplyRequestMsg signals the app to start applying the plan.
type ApplyRequestMsg struct{}

func (e *Plugin) requestApply() tea.Cmd {
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Apply plan (%d changes)?", len(e.summary.Changes)),
				func() tea.Cmd {
					return func() tea.Msg {
						return ApplyRequestMsg{}
					}
				},
			),
		}
	}
}

func (e *Plugin) requestForceUnlock() tea.Cmd {
	lockID := e.lockInfo.ID
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Force-unlock %s? This is dangerous if another operation is running.", lockID),
				func() tea.Cmd {
					return func() tea.Msg {
						err := svc.ForceUnlock(context.Background(), lockID)
						if err != nil {
							log.Debug("plan.force-unlock.error", "lockID", lockID, "error", err.Error())
						} else {
							log.Debug("plan.force-unlock.success", "lockID", lockID)
						}
						return ForceUnlockResultMsg{Err: err}
					}
				},
			),
		}
	}
}
