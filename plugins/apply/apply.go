// Package apply runs terraform apply in two modes:
//   - Plan-file mode (TUI flow): consumes a plan artifact from the plan plugin (ADR-0019).
//   - Auto-plan mode (CLI standalone): targets provided directly, terraform plans+applies in one shot.
package apply

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// StatusConfirming is the apply-specific state where the user is asked y/n
// before the plan file is applied.
const StatusConfirming = sdk.Status(10)

// ApplyResultMsg is sent when apply completes.
type ApplyResultMsg struct {
	Err      error
	Duration time.Duration
}

// Plugin implements the terraform apply feature.
type Plugin struct {
	sdk.PluginBase
	status     sdk.Status
	errMsg     string
	timer      ui.Timer
	confirmed  bool
	input      Input
	planFile   string
	cancelFn   context.CancelFunc
	stack      *sdk.Stack
	lastStream *frames.StreamFrame
}

// New creates a new apply plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		PluginBase: sdk.NewPluginBase("apply", "Apply", "Apply terraform changes to infrastructure"),
		stack:      sdk.NewStack(),
	}
	p.Svc = svc
	return p
}

func (e *Plugin) Stack() *sdk.Stack { return e.stack }

func (e *Plugin) Ready() bool            { return e.status == sdk.StatusDone }
func (e *Plugin) Status() sdk.Status     { return e.status }
func (e *Plugin) Elapsed() time.Duration { return e.timer.Elapsed() }
func (e *Plugin) IsConfirming() bool     { return e.status == StatusConfirming }
func (e *Plugin) Busy() bool             { return e.status == sdk.StatusLoading }

// SetPlanFile stages the plan artifact that the next RequestApply / AutoApply
// will consume. Called by the app when routing ApplyRequestMsg from plan.
func (e *Plugin) SetPlanFile(path string) { e.planFile = path }

// PlanFile returns the staged plan artifact path (used by app for logging).
func (e *Plugin) PlanFile() string { return e.planFile }

// Hints returns context-sensitive key hints for the status bar.
func (e *Plugin) Hints() []sdk.KeyHint {
	if top := e.stack.Peek(); top != nil {
		return top.Hints()
	}

	switch e.status {
	case sdk.StatusIdle:
		return (sdk.HintSetConfirm | sdk.HintSetQuit).Hints()
	case sdk.StatusLoading:
		return sdk.HintSetCancel.Hints()
	case StatusConfirming:
		return []sdk.KeyHint{
			{Key: "y/n", Description: "confirm"},
			sdk.HintCancel,
		}
	case sdk.StatusDone:
		hints := (sdk.HintSetRefresh | sdk.HintSetCancel).Hints()
		if e.lastStream != nil {
			hints = append(hints, sdk.KeyHint{Key: "l", Description: "log"})
		}
		return hints
	case sdk.StatusError:
		hints := (sdk.HintSetRetry | sdk.HintSetCancel).Hints()
		if e.lastStream != nil {
			hints = append(hints, sdk.KeyHint{Key: "l", Description: "log"})
		}
		return hints
	default:
		return sdk.HintSetQuit.Hints()
	}
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(_ map[string]interface{}) error { return nil }

// Init wires the plugin to its shared dependencies. Apply rebinds its
// Service via HandleContextChanged on every chdir/workspace switch and
// reads var-files / vars / parallelism / lock fresh from deps.Context()
// at every apply.
func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	e.InitBase(deps)
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler. A chdir or
// workspace switch invalidates any staged plan file, since it referenced the
// previous context's resources (the safety bug ADR-0018 fixes). Pure pin
// toggles are no-ops — apply has no targets to refresh.
func (e *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if ev.Next == nil {
		return nil
	}
	if ev.OnlyPinsChanged() {
		return nil
	}
	e.HandleContextChangedDefault(ev)
	e.planFile = ""
	e.confirmed = false
	e.status = sdk.StatusIdle
	e.errMsg = ""
	return nil
}

// Activate is the input port: cmd/tfui parses CLI flags into Input and hands
// the typed value to the plugin. The plugin stores the input on its state and
// returns the initial command that drives its lifecycle.
func (e *Plugin) Activate(input Input) tea.Cmd {
	e.input = input
	if input.AutoApprove {
		return e.AutoApply()
	}
	return e.RequestApply()
}

// RequestApply transitions to confirmation. The plan file must be staged via
// SetPlanFile before this is called.
func (e *Plugin) RequestApply() tea.Cmd {
	e.confirmed = false
	e.errMsg = ""
	e.status = StatusConfirming
	return nil
}

// AutoApply skips confirmation and begins apply immediately.
func (e *Plugin) AutoApply() tea.Cmd {
	e.confirmed = true
	e.errMsg = ""
	e.status = sdk.StatusLoading
	return tea.Batch(e.runApply(), e.timer.Start())
}

// Cancel aborts any in-flight terraform operation.
func (e *Plugin) Cancel() {
	if e.cancelFn != nil {
		e.cancelFn()
		e.cancelFn = nil
	}
}

// Confirm executes the apply after user confirmation.
func (e *Plugin) Confirm() tea.Cmd {
	e.confirmed = true
	e.status = sdk.StatusLoading
	e.errMsg = ""
	return tea.Batch(e.runApply(), e.timer.Start())
}

// Abort resets confirmation state without cancelling in-flight operations.
func (e *Plugin) Abort() {
	e.status = sdk.StatusIdle
	e.confirmed = false
}

func (e *Plugin) runApply() tea.Cmd {
	e.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel

	lw, ch := frames.NewLineWriter()
	sf := frames.NewStreamFrame("terraform apply", ch, cancel)
	e.lastStream = sf
	e.stack.Clear()
	e.stack.Push(sf)

	svc := e.Svc
	var opts sdk.ApplyOptions
	if e.GetCtx != nil {
		opts = e.GetCtx().ApplyOptions()
	}
	if e.planFile != "" {
		opts.PlanFile = e.planFile
	} else {
		opts.Targets = e.input.Targets
	}
	opts.AutoApprove = e.input.AutoApprove
	opts.Writer = lw
	start := time.Now()
	return tea.Batch(
		func() tea.Msg {
			err := svc.Apply(ctx, opts)
			lw.Close()
			return ApplyResultMsg{Err: err, Duration: time.Since(start)}
		},
		frames.WaitForLine(ch),
	)
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg.(type) {
	case frames.StreamLineMsg, frames.StreamDoneMsg:
		cmd := e.stack.Update(msg)
		return e, cmd
	}

	switch msg := msg.(type) {
	case ApplyResultMsg:
		e.timer.Stop()
		e.stack.Reset()
		// Drop our reference; cleanup is owned by the plan plugin (ADR-0020).
		e.planFile = ""
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
		if top := e.stack.Peek(); top != nil {
			cmd := e.stack.Update(msg)
			return e, cmd
		}
		cmd := e.handleKey(msg)
		return e, cmd
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch e.status {
	case sdk.StatusIdle:
		if msg.String() == "enter" {
			return e.RequestApply()
		}
	case StatusConfirming:
		switch msg.String() {
		case "y", "Y", "enter":
			return e.Confirm()
		case "n", "N", "esc":
			e.Abort()
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	case sdk.StatusDone:
		switch msg.String() {
		case "ctrl+r":
			return func() tea.Msg { return sdk.NavigateMsg{PluginID: "plan"} }
		case "esc":
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		case "l":
			if e.lastStream != nil {
				e.stack.Push(e.lastStream)
			}
		}
	case sdk.StatusError:
		switch msg.String() {
		case "ctrl+r":
			return e.Confirm()
		case "esc":
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		case "l":
			if e.lastStream != nil {
				e.stack.Push(e.lastStream)
			}
		}
	}
	return nil
}

// View renders the apply plugin.
func (e *Plugin) View(width, height int) string {
	if top := e.stack.Peek(); top != nil {
		return top.View(width, height)
	}

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

func (e *Plugin) renderConfirmation(_, _ int) string {
	header := sdk.StyleRiskHigh.Render("Are you sure you want to apply these changes?")
	detail := sdk.StyleFaint.Render("This will modify your infrastructure.")
	prompt := sdk.StyleKey.Render("[y]es") + " / " + sdk.StyleFaint.Render("[n]o")
	return header + "\n" + detail + "\n\n" + prompt
}

// ExitCode returns the process exit code: 1 if the apply failed, 0 otherwise.
func (e *Plugin) ExitCode() int {
	if e.status == sdk.StatusError {
		return 1
	}
	return 0
}
