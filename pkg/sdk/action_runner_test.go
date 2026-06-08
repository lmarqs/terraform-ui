package sdk

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// runResult drains the batch returned by Start (or Retry) and returns the
// single actionResultMsg it carries, mirroring how the framework would deliver
// the async result back into Update.
func runResult(t *testing.T, cmd tea.Cmd) actionResultMsg {
	t.Helper()
	if cmd == nil {
		t.Fatal("Start() returned nil cmd")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Start() cmd produced %T, want tea.BatchMsg", msg)
	}
	for _, sub := range batch {
		if sub == nil {
			continue
		}
		if r, ok := sub().(actionResultMsg); ok {
			return r
		}
	}
	t.Fatal("batch did not contain an actionResultMsg")
	return actionResultMsg{}
}

func taintLikeSpec(run func(ctx context.Context) ([]string, error)) ActionSpec {
	return ActionSpec{
		Verb:       "taint",
		Run:        run,
		OnSuccess:  []tea.Msg{PlanInvalidatedEvent{}},
		Idle:       "Waiting for confirmation...",
		Running:    func() string { return "Tainting 1 resource" },
		Done:       func(done []string) string { return "✓ Tainted " + strings.Join(done, ", ") },
		ErrorLabel: "Taint failed",
		OfferPlan:  true,
	}
}

func newRunner(spec ActionSpec) *ActionRunner {
	a := &ActionRunner{}
	a.InitRunner(nil) // nil → discard logger, must not panic
	a.Arm(spec)
	return a
}

func TestActionRunner_WhenArmed_ShouldBeIdle(t *testing.T) {
	a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
	if a.CurrentStatus() != StatusIdle {
		t.Errorf("CurrentStatus() = %v, want StatusIdle", a.CurrentStatus())
	}
	if a.Busy() {
		t.Error("Busy() = true when idle, want false")
	}
	if a.Ready() {
		t.Error("Ready() = true when idle, want false")
	}
}

func TestActionRunner_WhenStarted_ShouldRunAndComplete(t *testing.T) {
	var gotCtx context.Context
	a := newRunner(taintLikeSpec(func(ctx context.Context) ([]string, error) {
		gotCtx = ctx
		return []string{"aws_instance.web"}, nil
	}))

	cmd := a.Start()
	if a.CurrentStatus() != StatusLoading {
		t.Fatalf("CurrentStatus() = %v after Start, want StatusLoading", a.CurrentStatus())
	}
	if !a.Busy() {
		t.Error("Busy() = false while loading, want true")
	}

	result := runResult(t, cmd)
	if gotCtx == nil {
		t.Error("spec.Run must receive a non-nil context (cancellable, not Background)")
	}
	if gotCtx.Done() == nil {
		t.Error("spec.Run context must be cancellable (Done() channel non-nil)")
	}

	handled, eventCmd := a.Update(result)
	if !handled {
		t.Error("Update(actionResultMsg) should report handled")
	}
	if a.CurrentStatus() != StatusDone {
		t.Errorf("CurrentStatus() = %v after success, want StatusDone", a.CurrentStatus())
	}
	if !a.Ready() {
		t.Error("Ready() = false after Done, want true")
	}
	if eventCmd == nil {
		t.Fatal("success should emit OnSuccess events")
	}
	if _, ok := eventCmd().(PlanInvalidatedEvent); !ok {
		t.Errorf("OnSuccess event = %T, want PlanInvalidatedEvent", eventCmd())
	}
}

func TestActionRunner_WhenRunFails_ShouldEnterErrorWithMessage(t *testing.T) {
	a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) {
		return nil, errors.New("boom")
	}))

	result := runResult(t, a.Start())
	_, eventCmd := a.Update(result)

	if a.CurrentStatus() != StatusError {
		t.Errorf("CurrentStatus() = %v, want StatusError", a.CurrentStatus())
	}
	if a.ErrMessage() != "boom" {
		t.Errorf("ErrMessage() = %q, want %q", a.ErrMessage(), "boom")
	}
	if eventCmd != nil {
		t.Error("failure must not emit success events")
	}
}

func TestActionRunner_WhenMultipleOnSuccess_ShouldBatchAll(t *testing.T) {
	spec := taintLikeSpec(func(context.Context) ([]string, error) { return []string{"a"}, nil })
	spec.OnSuccess = []tea.Msg{StateRefreshedEvent{}, PlanInvalidatedEvent{}}
	a := newRunner(spec)

	result := runResult(t, a.Start())
	_, eventCmd := a.Update(result)
	if eventCmd == nil {
		t.Fatal("expected batched success events")
	}
	batch, ok := eventCmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("multiple OnSuccess should batch, got %T", eventCmd())
	}
	if len(batch) != 2 {
		t.Errorf("batch len = %d, want 2", len(batch))
	}
}

func TestActionRunner_WhenNoOnSuccess_ShouldReturnNilOnDone(t *testing.T) {
	spec := taintLikeSpec(func(context.Context) ([]string, error) { return []string{"a"}, nil })
	spec.OnSuccess = nil
	a := newRunner(spec)

	result := runResult(t, a.Start())
	if _, eventCmd := a.Update(result); eventCmd != nil {
		t.Error("no OnSuccess should yield nil cmd on Done")
	}
}

func TestActionRunner_WhenTimerTicks_ShouldHandleAndContinue(t *testing.T) {
	a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
	a.Start() // starts the timer

	handled, cmd := a.Update(ui.TimerTickMsg{})
	if !handled {
		t.Error("TimerTickMsg should be handled by the runner")
	}
	if cmd == nil {
		t.Error("TimerTickMsg while running should return the next tick cmd")
	}
}

func TestActionRunner_WhenUnknownMsg_ShouldNotHandle(t *testing.T) {
	a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
	if handled, cmd := a.Update(struct{}{}); handled || cmd != nil {
		t.Errorf("unknown msg: handled=%v cmd=%v, want false/nil", handled, cmd)
	}
}

func TestActionRunner_Cancel_ShouldCancelContext(t *testing.T) {
	var capturedCtx context.Context
	a := newRunner(taintLikeSpec(func(ctx context.Context) ([]string, error) {
		capturedCtx = ctx
		return nil, nil
	}))

	// Draining the batch runs the work func, which captures the context.
	// Start stores the cancel func; it is not cleared until Cancel runs.
	runResult(t, a.Start())
	if capturedCtx == nil {
		t.Fatal("spec.Run did not receive a context")
	}

	a.Cancel()
	select {
	case <-capturedCtx.Done():
		// cancelled as expected
	default:
		t.Error("Cancel() did not cancel the in-flight context")
	}

	// Cancel again with no in-flight op must be a no-op (not panic).
	a.Cancel()
}

func TestActionRunner_StandardKeys(t *testing.T) {
	keyP := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	keyEsc := tea.KeyMsg{Type: tea.KeyEsc}
	keyCtrlR := tea.KeyMsg{Type: tea.KeyCtrlR}

	t.Run("Done+OfferPlan+p navigates to plan", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
		a.Update(runResult(t, a.Start()))
		cmd := a.StandardKeys(keyP)
		if cmd == nil {
			t.Fatal("p in Done should return a cmd")
		}
		nav, ok := cmd().(NavigateMsg)
		if !ok || nav.PluginID != "plan" {
			t.Errorf("got %#v, want NavigateMsg{plan}", cmd())
		}
	})

	t.Run("Done without OfferPlan ignores p", func(t *testing.T) {
		spec := taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil })
		spec.OfferPlan = false
		a := newRunner(spec)
		a.Update(runResult(t, a.Start()))
		if cmd := a.StandardKeys(keyP); cmd != nil {
			t.Error("p without OfferPlan should be ignored")
		}
	})

	t.Run("Done+esc deactivates", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
		a.Update(runResult(t, a.Start()))
		cmd := a.StandardKeys(keyEsc)
		if cmd == nil {
			t.Fatal("esc should return a cmd")
		}
		if _, ok := cmd().(DeactivateMsg); !ok {
			t.Errorf("got %T, want DeactivateMsg", cmd())
		}
	})

	t.Run("Error+ctrl+r retries (re-runs spec, back to loading)", func(t *testing.T) {
		calls := 0
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) {
			calls++
			return nil, errors.New("fail")
		}))
		a.Update(runResult(t, a.Start()))
		if a.CurrentStatus() != StatusError {
			t.Fatalf("precondition: want StatusError, got %v", a.CurrentStatus())
		}
		cmd := a.StandardKeys(keyCtrlR)
		if cmd == nil {
			t.Fatal("ctrl+r in Error should return retry cmd")
		}
		if a.CurrentStatus() != StatusLoading {
			t.Errorf("after retry CurrentStatus() = %v, want StatusLoading", a.CurrentStatus())
		}
		runResult(t, cmd) // drains the retry's work; must re-invoke spec.Run
		if calls != 2 {
			t.Errorf("spec.Run called %d times, want 2 (initial + retry)", calls)
		}
	})

	t.Run("Error+esc deactivates", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, errors.New("x") }))
		a.Update(runResult(t, a.Start()))
		if cmd := a.StandardKeys(keyEsc); cmd == nil {
			t.Error("esc in Error should deactivate")
		}
	})

	t.Run("Idle+esc deactivates", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
		cmd := a.StandardKeys(keyEsc)
		if cmd == nil {
			t.Fatal("esc in Idle should deactivate")
		}
		if _, ok := cmd().(DeactivateMsg); !ok {
			t.Errorf("got %T, want DeactivateMsg", cmd())
		}
	})

	t.Run("Loading ignores keys", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
		a.Start()
		if cmd := a.StandardKeys(keyP); cmd != nil {
			t.Error("keys during Loading should be ignored")
		}
	})

	t.Run("unrecognized key returns nil", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
		a.Update(runResult(t, a.Start()))
		if cmd := a.StandardKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}); cmd != nil {
			t.Error("unrecognized key should return nil")
		}
	})
}

func TestActionRunner_View(t *testing.T) {
	makeRunner := func() *ActionRunner {
		return newRunner(taintLikeSpec(func(context.Context) ([]string, error) {
			return []string{"aws_instance.web"}, nil
		}))
	}

	t.Run("Idle shows the idle text", func(t *testing.T) {
		a := makeRunner()
		if !strings.Contains(a.View(), "Waiting for confirmation") {
			t.Errorf("Idle view = %q, want idle text", a.View())
		}
	})

	t.Run("Loading shows running phrase", func(t *testing.T) {
		a := makeRunner()
		a.Start()
		if !strings.Contains(a.View(), "Tainting") {
			t.Errorf("Loading view = %q, want running phrase", a.View())
		}
	})

	t.Run("Done shows done label and duration", func(t *testing.T) {
		a := makeRunner()
		a.Update(runResult(t, a.Start()))
		view := a.View()
		if !strings.Contains(view, "✓ Tainted") {
			t.Errorf("Done view = %q, want done label", view)
		}
		if !strings.Contains(view, "Duration:") {
			t.Errorf("Done view = %q, want duration line", view)
		}
	})

	t.Run("Error shows error label and message", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) {
			return nil, errors.New("permission denied")
		}))
		a.Update(runResult(t, a.Start()))
		view := a.View()
		if !strings.Contains(view, "Taint failed") || !strings.Contains(view, "permission denied") {
			t.Errorf("Error view = %q, want label + message", view)
		}
	})

	t.Run("unknown status renders empty", func(t *testing.T) {
		a := makeRunner()
		a.status = Status(99)
		if a.View() != "" {
			t.Errorf("unknown status view = %q, want empty", a.View())
		}
	})
}

func TestActionRunner_Hints(t *testing.T) {
	t.Run("Done with OfferPlan offers plan and cancel", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
		a.Update(runResult(t, a.Start()))
		hints := a.Hints()
		if len(hints) != 2 || hints[0].Key != "p" || hints[1].Description != "cancel" {
			t.Errorf("Done hints = %+v, want [{p plan} {Esc cancel}]", hints)
		}
	})

	t.Run("Done without OfferPlan shows back", func(t *testing.T) {
		spec := taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil })
		spec.OfferPlan = false
		a := newRunner(spec)
		a.Update(runResult(t, a.Start()))
		hints := a.Hints()
		if len(hints) != 1 || hints[0].Description != "back" {
			t.Errorf("Done hints (no plan) = %+v, want [{Esc back}]", hints)
		}
	})

	t.Run("Error shows retry", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, errors.New("x") }))
		a.Update(runResult(t, a.Start()))
		found := false
		for _, h := range a.Hints() {
			if h.Description == "retry" {
				found = true
			}
		}
		if !found {
			t.Error("Error hints should contain retry")
		}
	})

	t.Run("default shows back", func(t *testing.T) {
		a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
		hints := a.Hints() // idle
		if len(hints) == 0 || hints[len(hints)-1].Description != "back" {
			t.Errorf("idle hints = %+v, want back", hints)
		}
	})
}

func TestActionRunner_Reset_ShouldReturnToIdle(t *testing.T) {
	a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) {
		return nil, errors.New("x")
	}))
	a.Update(runResult(t, a.Start()))
	if a.CurrentStatus() != StatusError || a.ErrMessage() == "" {
		t.Fatal("precondition: expected error state with message")
	}

	a.Reset()
	if a.CurrentStatus() != StatusIdle {
		t.Errorf("CurrentStatus() = %v after Reset, want StatusIdle", a.CurrentStatus())
	}
	if a.ErrMessage() != "" {
		t.Error("Reset should clear the error message")
	}
}

func TestActionRunner_InitRunner_AcceptsExplicitLogger(t *testing.T) {
	a := &ActionRunner{}
	a.InitRunner(slog.New(slog.NewTextHandler(io.Discard, nil)))
	a.Arm(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
	// A real (non-nil) logger must drive the run without panicking.
	a.Update(runResult(t, a.Start()))
	if !a.Ready() {
		t.Error("run with explicit logger should reach Done")
	}
}

func TestActionRunner_Elapsed_ShouldBeFormatted(t *testing.T) {
	a := newRunner(taintLikeSpec(func(context.Context) ([]string, error) { return nil, nil }))
	a.Start()
	if a.Elapsed() == "" {
		t.Error("Elapsed() should be non-empty while/after running")
	}
}
