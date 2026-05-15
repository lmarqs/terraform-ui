package apply

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

const StatusConfirming = sdk.Status(10)

// ApplyResultMsg is sent when apply completes.
type ApplyResultMsg struct {
	Err      error
	Duration time.Duration
}

// Plugin implements the terraform apply feature.
type Plugin struct {
	svc            sdk.Service
	options        *sdk.ResolvedOptions
	status         sdk.Status
	errMsg         string
	targets        []string
	timer          ui.Timer
	confirmed      bool
	totalResources int
	scopedContext  string
}

// New creates a new apply plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc: svc,
	}
}

func (e *Plugin) ID() string          { return "apply" }
func (e *Plugin) Name() string        { return "Apply" }
func (e *Plugin) Description() string { return "Apply terraform changes to infrastructure" }
func (e *Plugin) Ready() bool         { return e.status == sdk.StatusDone }
func (e *Plugin) Status() sdk.Status  { return e.status }
func (e *Plugin) Elapsed() time.Duration {
	return e.timer.Elapsed()
}
func (e *Plugin) IsConfirming() bool { return e.status == StatusConfirming }
func (e *Plugin) Busy() bool         { return e.status == sdk.StatusLoading }

// Hints returns context-sensitive key hints for the status bar.
func (e *Plugin) Hints() []sdk.KeyHint {
	switch e.status {
	case sdk.StatusIdle:
		return (sdk.HintSetConfirm | sdk.HintSetBack).Hints()
	case StatusConfirming:
		return []sdk.KeyHint{
			{Key: "y/n", Description: "confirm"},
			sdk.HintCancel,
		}
	case sdk.StatusLoading:
		return (sdk.HintSetCancel).Hints()
	case sdk.StatusDone:
		return (sdk.HintSetRefresh | sdk.HintSetCancel).Hints()
	case sdk.StatusError:
		return (sdk.HintSetRetry | sdk.HintSetCancel).Hints()
	default:
		return (sdk.HintSetBack).Hints()
	}
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// SetTargets configures resource targets for apply.
func (e *Plugin) SetTargets(targets []string) {
	e.targets = targets
}

// Targets returns the currently configured targets.
func (e *Plugin) Targets() []string {
	return e.targets
}

// Init initializes the plugin with shared context.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	e.options = ctx.Options
	return nil
}

// HandleChdirChanged implements sdk.ChdirHandler.
func (e *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	e.svc = e.svc.WithDir(evt.AbsPath)
	e.scopedContext = evt.AbsPath
	// Apply intentionally preserves targets/confirmed/totalResources across scope changes
	e.status = sdk.StatusIdle
	e.errMsg = ""
	return nil
}

// HandlePlanCompleted implements sdk.PlanCompletedHandler.
func (e *Plugin) HandlePlanCompleted(evt sdk.PlanCompletedEvent) tea.Cmd {
	e.totalResources = evt.ResourceCount
	return nil
}

// Activate resets terminal states when re-entered via navigation.
func (e *Plugin) Activate() tea.Cmd {
	if e.status == sdk.StatusError || e.status == sdk.StatusDone {
		e.status = sdk.StatusIdle
		e.errMsg = ""
	}
	return nil
}

// TotalResources returns the total resource count from the last completed plan.
func (e *Plugin) TotalResources() int {
	return e.totalResources
}

// RequestApply transitions to the confirmation state.
func (e *Plugin) RequestApply() {
	e.status = StatusConfirming
	e.confirmed = false
	e.errMsg = ""
}

// Confirm executes the apply after user confirmation.
func (e *Plugin) Confirm() tea.Cmd {
	e.confirmed = true
	e.status = sdk.StatusLoading
	e.errMsg = ""
	return tea.Batch(e.runApply(), e.timer.Start())
}

// Cancel aborts the apply confirmation.
func (e *Plugin) Cancel() {
	e.status = sdk.StatusIdle
	e.confirmed = false
}

func (e *Plugin) runApply() tea.Cmd {
	svc := e.svc
	opts := sdk.BuildApplyOptions(e.options, e.targets)
	start := time.Now()
	return func() tea.Msg {
		err := svc.Apply(context.Background(), opts)
		return ApplyResultMsg{Err: err, Duration: time.Since(start)}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ApplyResultMsg:
		e.timer.Stop()
		if msg.Err != nil {
			e.status = sdk.StatusError
			e.errMsg = msg.Err.Error()
			return e, nil
		}
		e.status = sdk.StatusDone
		return e, func() tea.Msg { return sdk.PlanInvalidatedEvent{} }

	case ui.TimerTickMsg:
		return e, e.timer.Tick()

	case tea.KeyMsg:
		cmd := e.handleKey(msg)
		return e, cmd
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch e.status {
	case sdk.StatusIdle:
		switch msg.String() {
		case "enter":
			e.RequestApply()
		}
	case StatusConfirming:
		switch msg.String() {
		case "y", "Y", "enter":
			return e.Confirm()
		case "n", "N", "esc":
			e.Cancel()
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	case sdk.StatusDone:
		switch msg.String() {
		case "ctrl+r":
			return func() tea.Msg { return sdk.NavigateMsg{PluginID: "plan"} }
		case "esc":
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	case sdk.StatusError:
		switch msg.String() {
		case "ctrl+r":
			return e.Confirm()
		case "esc":
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	}
	return nil
}

// View renders the apply plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Run plan first, then apply changes here.")

	case StatusConfirming:
		return e.renderConfirmation(width, height)

	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Applying changes... " + e.timer.FormatElapsed())

	case sdk.StatusDone:
		success := sdk.StyleSuccess.Render("Apply complete! Resources are up-to-date.")
		duration := sdk.StyleFaint.Render("Duration: " + e.timer.FormatElapsed())
		return success + "\n" + duration

	case sdk.StatusError:
		return sdk.StyleError.Render("Apply failed: " + e.errMsg)

	default:
		return ""
	}
}

func (e *Plugin) renderConfirmation(width, height int) string {
	warning := sdk.StyleRiskHigh.Render("Are you sure you want to apply these changes?")
	detail := sdk.StyleFaint.Render("This will modify your infrastructure.")

	if len(e.targets) > 0 {
		detail = sdk.StyleFaint.Render(fmt.Sprintf("Targeting %d resource(s).", len(e.targets)))
	}

	prompt := sdk.StyleKey.Render("[y]es") + " / " + sdk.StyleFaint.Render("[n]o")

	return warning + "\n" + detail + "\n\n" + prompt
}
