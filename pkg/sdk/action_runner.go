package sdk

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// ActionSpec is the per-operation adapter surface for ActionRunner — the only
// thing that differs between plugins in action mode (today: taint, untaint,
// import, forceunlock). The plugin builds one in Activate, arms the runner with
// it, and the runner owns everything that happens afterwards.
type ActionSpec struct {
	// Name identifies the action (e.g. "taint"). Used to tag result log
	// lines: "<Name>.success" / "<Name>.error".
	Name string
	// Run performs the terraform mutation against the supplied cancellable
	// context and returns the addresses that completed. It may loop (taint over
	// many addresses, accumulating partial success) or call once (import,
	// force-unlock) — the runner does not care which.
	Run func(ctx context.Context) ([]string, error)
	// OnSuccess lists the events emitted when Run succeeds (e.g.
	// PlanInvalidatedEvent, StateRefreshedEvent).
	OnSuccess []tea.Msg

	// The remaining fields drive the runner's default View/Hints. Plugins that
	// render their own (forceunlock) may leave them zero.

	// Idle is the placeholder shown before the action runs.
	Idle string
	// Running returns the in-progress phrase, e.g. "Tainting 3 resources".
	Running func() string
	// Done returns the success message given the completed addresses, e.g.
	// "✓ Tainted 3 resources".
	Done func(done []string) string
	// ErrorLabel prefixes the failure message, e.g. "Taint failed".
	ErrorLabel string
	// OfferPlan makes the Done state offer the `p` → plan shortcut.
	OfferPlan bool
}

// actionResultMsg carries the outcome of ActionSpec.Run back into Update.
type actionResultMsg struct {
	done []string
	err  error
}

// ActionRunner is the deep, embeddable capability providing the action-mode
// lifecycle: the execute-and-report phase a plugin enters after gathering input.
// It owns the cancellable execution, the elapsed timer, the Idle→Loading→
// Done/Error status machine, result handling, retry, and success-event emission
// — so each plugin only supplies an ActionSpec and its own input prelude
// (confirm / form / manual entry). A plugin composes it to gain the mode.
//
// Embed it next to PluginBase:
//
//	type Plugin struct {
//	    sdk.PluginBase
//	    sdk.ActionRunner
//	}
//
// Busy(), Ready() and Cancel() are promoted, so the plugin satisfies sdk.Busy
// and sdk.Cancellable for free.
type ActionRunner struct {
	log      *slog.Logger
	timer    ui.Timer
	status   Status
	spec     ActionSpec
	done     []string
	errMsg   string
	cancelFn context.CancelFunc
}

// InitRunner wires the logger. Embedders call it from Init after InitBase.
// A nil logger is replaced with a discard logger so the runner is always safe.
func (a *ActionRunner) InitRunner(log *slog.Logger) {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	a.log = log
}

// Arm stores the spec and resets the runner to Idle. Called from Activate,
// before the confirm/form prelude runs.
func (a *ActionRunner) Arm(spec ActionSpec) {
	a.spec = spec
	a.Reset()
}

// Start begins executing the armed spec: it cancels any prior run, opens a
// fresh cancellable context, enters Loading, and returns a command batch that
// runs the work and ticks the timer. Re-invoking it is how retry works.
func (a *ActionRunner) Start() tea.Cmd {
	a.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelFn = cancel
	a.status = StatusLoading
	a.done = nil
	a.errMsg = ""

	spec := a.spec
	log := a.log
	return tea.Batch(
		func() tea.Msg {
			done, err := spec.Run(ctx)
			if err != nil {
				log.Debug(spec.Name+".error", "error", err.Error())
			} else {
				log.Debug(spec.Name+".success", "count", len(done))
			}
			return actionResultMsg{done: done, err: err}
		},
		a.timer.Start(),
	)
}

// Update handles the runner's async messages (timer ticks and the action
// result). It reports whether the message was consumed so embedders can fall
// through to their own (key/prelude) handling.
func (a *ActionRunner) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch m := msg.(type) {
	case ui.TimerTickMsg:
		return true, a.timer.Tick()
	case actionResultMsg:
		a.timer.Stop()
		a.done = m.done
		if m.err != nil {
			a.status = StatusError
			a.errMsg = m.err.Error()
			return true, nil
		}
		a.status = StatusDone
		return true, a.successCmd()
	}
	return false, nil
}

func (a *ActionRunner) successCmd() tea.Cmd {
	if len(a.spec.OnSuccess) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, len(a.spec.OnSuccess))
	for i := range a.spec.OnSuccess {
		ev := a.spec.OnSuccess[i]
		cmds[i] = func() tea.Msg { return ev }
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}

// StandardKeys handles the key bindings shared by the standard-shaped verbs
// (taint, untaint, import): `p` → plan from Done (when OfferPlan), `ctrl+r` →
// retry from Error, `esc` → deactivate. Plugins with bespoke key maps
// (forceunlock) ignore this and handle keys themselves.
func (a *ActionRunner) StandardKeys(key tea.KeyMsg) tea.Cmd {
	switch a.status {
	case StatusDone:
		switch key.String() {
		case "p":
			if a.spec.OfferPlan {
				return func() tea.Msg { return NavigateMsg{PluginID: "plan"} }
			}
		case "esc":
			return deactivate()
		}
	case StatusError:
		switch key.String() {
		case "ctrl+r":
			return a.Start()
		case "esc":
			return deactivate()
		}
	case StatusIdle:
		if key.String() == "esc" {
			return deactivate()
		}
	}
	return nil
}

func deactivate() tea.Cmd {
	return func() tea.Msg { return DeactivateMsg{} }
}

// View renders the standard Idle/Loading/Done/Error states from the spec.
// Plugins that need a custom look (forceunlock's lock info) render their own
// and read CurrentStatus / Elapsed / ErrMessage instead.
func (a *ActionRunner) View() string {
	switch a.status {
	case StatusLoading:
		return StyleFaintItalic.Render(fmt.Sprintf("%s... %s", a.spec.Running(), a.timer.FormatElapsed()))
	case StatusDone:
		return StyleSuccess.Render(a.spec.Done(a.done)) +
			"\n" + StyleFaint.Render("Duration: "+a.timer.FormatElapsed())
	case StatusError:
		return StyleError.Render("✗ " + a.spec.ErrorLabel + ": " + a.errMsg)
	}
	return StyleFaintItalic.Render(a.spec.Idle) // idle / pre-run placeholder
}

// Hints returns the standard hint sets for the Done/Error/default states.
func (a *ActionRunner) Hints() []KeyHint {
	switch a.status {
	case StatusDone:
		if a.spec.OfferPlan {
			return []KeyHint{{Key: "p", Description: "plan"}, HintCancel}
		}
		return HintSetBack.Hints()
	case StatusError:
		return (HintSetRetry | HintSetBack).Hints()
	default:
		return HintSetBack.Hints()
	}
}

// Cancel aborts any in-flight run. Promoted to satisfy sdk.Cancellable.
func (a *ActionRunner) Cancel() {
	if a.cancelFn != nil {
		a.cancelFn()
		a.cancelFn = nil
	}
}

// Reset returns the runner to Idle, clearing the last result and stopping the
// timer. Embedders call it from HandleContextChanged.
func (a *ActionRunner) Reset() {
	a.status = StatusIdle
	a.done = nil
	a.errMsg = ""
	a.timer.Stop()
}

// Busy reports an in-flight run. Promoted to satisfy sdk.Busy.
func (a *ActionRunner) Busy() bool { return a.status == StatusLoading }

// Ready reports a completed run. Promoted to satisfy Plugin.Ready.
func (a *ActionRunner) Ready() bool { return a.status == StatusDone }

// CurrentStatus exposes the lifecycle status for plugins that render their own
// view. It is deliberately NOT named Status() to avoid implementing
// sdk.Statusable by promotion, which would change CI completion detection.
func (a *ActionRunner) CurrentStatus() Status { return a.status }

// Elapsed returns the formatted elapsed time of the current/last run.
func (a *ActionRunner) Elapsed() string { return a.timer.FormatElapsed() }

// ErrMessage returns the last failure message (empty when not in error).
func (a *ActionRunner) ErrMessage() string { return a.errMsg }
