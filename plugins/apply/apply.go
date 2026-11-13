// Package apply runs terraform apply in two modes:
//   - Plan-file mode: consumes the plan artifact the plan plugin hands over (ADR-0019),
//     after the user reviewed its diff, and asks for confirmation. Both the TUI and the
//     interactive `tfui apply` reach apply this way.
//   - One-shot mode (`tfui apply --auto-approve`): terraform plans and applies in one
//     run, with any --target flags, and no review or prompt.
package apply

import (
	"context"
	"fmt"
	"io"
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
	// Counts is the resource tally terraform printed on completion, or nil when
	// it printed none. A failed run never prints one.
	Counts *sdk.ApplyCounts
	// Applied is what terraform reported finishing, resource by resource. It is
	// the only account of a run that stopped partway.
	Applied sdk.ApplyCounts
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
	// appliedPlanFile records whether the latest run consumed a plan file. A
	// failed plan-file run offers no retry: with the plan file spent, a retry
	// would apply everything unreviewed.
	appliedPlanFile bool
	cancelFn   context.CancelFunc
	stack      *sdk.Stack
	lastStream *frames.StreamFrame
	// counts is terraform's own post-apply tally, read off the output stream.
	// nil until terraform reports one — an unreported tally is unknown, which is
	// not the same answer as zeros.
	counts *sdk.ApplyCounts
	// applied accumulates the per-resource completion notices. terraform prints
	// no tally when a run fails, so this is the only account of what a partial
	// apply managed to do before it stopped.
	applied sdk.ApplyCounts
	// cfgTargets is the project-level default target list from
	// `plugin "apply" { targets = [...] }`, used only when the caller passed no
	// --target. It keeps apply scoped like plan when both read the same config.
	cfgTargets []string
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

// SetLockOverrides stages the per-run --lock / --lock-timeout overrides the
// plan ran with, so the apply of that plan locks state the same way. Called by
// the app when routing ApplyRequestMsg from plan.
func (e *Plugin) SetLockOverrides(mode sdk.LockMode, timeout sdk.LockTimeout) {
	e.input.Lock = mode
	e.input.LockTimeout = timeout
}

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
		hints := []sdk.KeyHint{
			{Key: "y/n", Description: "confirm"},
			sdk.HintCancel,
		}
		if e.lastStream != nil {
			hints = append(hints, sdk.KeyHint{Key: "l", Description: "log"})
		}
		return hints
	case sdk.StatusDone:
		hints := (sdk.HintSetRefresh | sdk.HintSetCancel).Hints()
		if e.lastStream != nil {
			hints = append(hints, sdk.KeyHint{Key: "l", Description: "log"})
		}
		return hints
	case sdk.StatusError:
		set := sdk.HintSetCancel
		if !e.appliedPlanFile {
			set |= sdk.HintSetRetry
		}
		hints := set.Hints()
		if e.lastStream != nil {
			hints = append(hints, sdk.KeyHint{Key: "l", Description: "log"})
		}
		return hints
	default:
		return sdk.HintSetQuit.Hints()
	}
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	e.cfgTargets = sdk.ConfigStrings(cfg, "targets", nil)
	return nil
}

// targets resolves the target list for a run: the explicit --target selection
// when there is one, the project default otherwise.
func (e *Plugin) targets() []string {
	if len(e.input.Targets) > 0 {
		return e.input.Targets
	}
	return e.cfgTargets
}

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
	// onPins is nil: a pure pin toggle is a no-op — apply has no targets to
	// refresh (it consumes the saved plan file, ADR-0019).
	return e.ReactToContext(ev, nil, func() {
		e.planFile = ""
		e.confirmed = false
		e.status = sdk.StatusIdle
		e.errMsg = ""
	})
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
	e.counts = nil
	e.applied = sdk.ApplyCounts{}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel

	lw, ch := frames.NewLineWriter()
	sf := frames.NewStreamFrame("terraform apply", ch, cancel).WithElapsed(e.timer.FormatElapsed)
	e.lastStream = sf
	e.stack.Clear()
	e.stack.Push(sf)

	svc := e.Svc
	var opts sdk.ApplyOptions
	if e.GetCtx != nil {
		opts = e.GetCtx().ApplyOptions()
	}
	e.appliedPlanFile = e.planFile != ""
	if e.appliedPlanFile {
		opts.PlanFile = e.planFile
	} else {
		opts.Targets = e.targets()
	}
	opts.AutoApprove = e.input.AutoApprove
	opts.Lock = e.input.Lock.Or(opts.Lock)
	opts.LockTimeout = e.input.LockTimeout.Or(opts.LockTimeout)
	// The tally rides the same output terraform streams to the UI, so the result
	// message can describe what happened without waiting on the message loop.
	tally := &sdk.ApplyTally{}
	opts.Writer = io.MultiWriter(lw, tally)
	start := time.Now()
	return tea.Batch(
		func() tea.Msg {
			err := svc.Apply(ctx, opts)
			lw.Close()
			return ApplyResultMsg{
				Err:      err,
				Duration: time.Since(start),
				Counts:   tally.Reported(),
				Applied:  tally.Applied(),
			}
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
		e.counts = msg.Counts
		e.applied = msg.Applied
		// Drop our reference; cleanup is owned by the plan plugin (ADR-0020).
		e.planFile = ""
		// The targeting selection is spent once apply reaches a terminal state,
		// success or error — clear pins so the next plan isn't silently scoped
		// to the just-applied set. The app's empty-pins guard makes this a no-op
		// for an un-targeted apply.
		if msg.Err != nil {
			e.status = sdk.StatusError
			e.errMsg = msg.Err.Error()
			return e, e.ClearPinsFn()
		}
		e.status = sdk.StatusDone
		return e, tea.Batch(
			func() tea.Msg { return sdk.PlanInvalidatedEvent{} },
			e.ClearPinsFn(),
		)

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
		// Enter/Esc parity: Enter confirms (yes), Esc cancels (no).
		case "y", "Y", "enter":
			return e.Confirm()
		case "n", "N", "esc":
			e.Abort()
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		case "l":
			// Re-open the streamed plan to review the full diff before deciding.
			if e.lastStream != nil {
				e.stack.Push(e.lastStream)
			}
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
			if e.appliedPlanFile {
				return nil
			}
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

// Stderr reports the apply outcome. Apply deliberately emits no stdout —
// exit code 0 is the machine signal — but headless callers still need a
// human-readable confirmation or the terraform error, on the status channel.
//
// The outcome carries terraform's resource tally when terraform reported one:
// it is what a caller reconciles the plan against, and after a partial failure
// it separates "nothing happened" from "half applied". A run that reported no
// tally says so by omission rather than printing zeros.
func (e *Plugin) Stderr() []byte {
	switch e.status {
	case sdk.StatusError:
		if e.applied.Empty() {
			return []byte(e.errMsg + "\n")
		}
		return []byte(fmt.Sprintf("%s\nApplied before the failure: %s.\n", e.errMsg, e.applied))
	case sdk.StatusDone:
		if e.counts != nil {
			return []byte(fmt.Sprintf("Apply complete. Resources: %s.\n", e.counts))
		}
		return []byte("Apply complete.\n")
	default:
		return nil
	}
}

// ExitCode returns the process exit code: 1 if the apply failed, 0 otherwise.
func (e *Plugin) ExitCode() int {
	if e.status == sdk.StatusError {
		return 1
	}
	return 0
}
