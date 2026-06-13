package apply

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "apply" {
		t.Errorf("ID() = %q, want %q", p.ID(), "apply")
	}
	if p.Name() != "Apply" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Apply")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(map[string]interface{}{}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	if cmd := p.Init(sdktest.NewDeps(svc).Deps); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
	if p.Ready() {
		t.Error("Ready() should be false before activation")
	}
}

func TestRequestApply_WhenCalled_ShouldEnterConfirmingState(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	p.SetPlanFile("/tmp/foo.tfplan")
	p.RequestApply()

	if p.status != StatusConfirming {
		t.Errorf("status = %v, want StatusConfirming", p.status)
	}
	if p.confirmed {
		t.Error("confirmed = true, want false")
	}
	if !p.IsConfirming() {
		t.Error("IsConfirming() = false, want true")
	}
}

func TestSetPlanFile_WhenSet_ShouldBeStaged(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.SetPlanFile("/tmp/staged.tfplan")
	if p.PlanFile() != "/tmp/staged.tfplan" {
		t.Errorf("PlanFile() = %q, want /tmp/staged.tfplan", p.PlanFile())
	}
}

func TestConfirm_WhenCalled_ShouldRunApplyWithStagedPlanFile(t *testing.T) {
	var got sdk.ApplyOptions
	svc := &sdktest.MockService{
		ApplyFn: func(_ context.Context, opts sdk.ApplyOptions) error {
			got = opts
			return nil
		},
	}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.SetPlanFile("/tmp/foo.tfplan")
	p.status = StatusConfirming

	cmd := p.Confirm()
	if cmd == nil {
		t.Fatal("Confirm() returned nil cmd")
	}
	drainBatchUntilApplyResult(t, cmd)

	if got.PlanFile != "/tmp/foo.tfplan" {
		t.Errorf("ApplyOptions.PlanFile = %q, want /tmp/foo.tfplan", got.PlanFile)
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want StatusLoading", p.status)
	}
	if !p.confirmed {
		t.Error("confirmed = false, want true")
	}
}

// drainBatchUntilApplyResult fans out the cmds from a (possibly nested)
// tea.Batch, runs each in a goroutine, and returns once the apply leaf has
// produced ApplyResultMsg. We can't run cmds sequentially because the batch
// pairs the service call with a WaitForLine that blocks until the writer
// closes — bubbletea only sees them concurrently in a real run loop.
func drainBatchUntilApplyResult(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	done := make(chan ApplyResultMsg, 1)
	var run func(c tea.Cmd)
	run = func(c tea.Cmd) {
		go func() {
			msg := c()
			if r, ok := msg.(ApplyResultMsg); ok {
				select {
				case done <- r:
				default:
				}
				return
			}
			if batch, ok := msg.(tea.BatchMsg); ok {
				for _, sub := range batch {
					run(sub)
				}
			}
		}()
	}
	run(cmd)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for ApplyResultMsg")
	}
}

func TestAbort_WhenCalled_ShouldResetToIdle(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusConfirming
	p.confirmed = true

	p.Abort()
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want sdk.StatusIdle", p.status)
	}
	if p.confirmed {
		t.Error("confirmed = true after abort, want false")
	}
}

func TestUpdate_WhenApplyResultSuccess_ShouldEmitPlanInvalidated(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.planFile = "/tmp/foo.tfplan"
	p.timer.Start()

	result, cmd := p.Update(ApplyResultMsg{Err: nil, Duration: 5 * time.Second})
	if cmd == nil {
		t.Fatal("Update(ApplyResultMsg) cmd = nil, want PlanInvalidatedEvent emitter")
	}
	msg := cmd()
	if _, ok := msg.(sdk.PlanInvalidatedEvent); !ok {
		t.Errorf("cmd() = %T, want sdk.PlanInvalidatedEvent", msg)
	}

	updated := result.(*Plugin)
	if updated.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", updated.status)
	}
	if updated.timer.Running() {
		t.Error("timer still running after result, want stopped")
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
	if updated.planFile != "" {
		t.Errorf("planFile = %q, want cleared after consumed", updated.planFile)
	}
}

func TestUpdate_WhenApplyResultError_ShouldTransitionToError(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	result, cmd := p.Update(ApplyResultMsg{Err: errors.New("apply failed"), Duration: 3 * time.Second})
	if cmd != nil {
		t.Errorf("Update(ApplyResultMsg) cmd = %v, want nil", cmd)
	}
	updated := result.(*Plugin)
	if updated.status != sdk.StatusError {
		t.Errorf("status = %v, want sdk.StatusError", updated.status)
	}
	if updated.errMsg != "apply failed" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "apply failed")
	}
}

func TestUpdate_WhenTimerTickWhileRunning_ShouldReturnNextTickCmd(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.timer.Start()

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd == nil {
		t.Error("Update(TimerTickMsg) while timer running: cmd = nil, want non-nil (next tick)")
	}
}

func TestUpdate_WhenTimerTickWhileStopped_ShouldReturnNilCmd(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd != nil {
		t.Error("Update(TimerTickMsg) while timer stopped: cmd != nil, want nil")
	}
}

func TestUpdate_WhenEnterInIdle_ShouldTransitionToConfirming(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusIdle

	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.status != StatusConfirming {
		t.Errorf("after enter in idle: status = %v, want StatusConfirming", p.status)
	}
}

func TestUpdate_WhenYInConfirming_ShouldStartApply(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Init(sdktest.NewDeps(&sdktest.MockService{}).Deps)
	p.status = StatusConfirming

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Error("after y in confirming: cmd = nil, want non-nil")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("after y: status = %v, want sdk.StatusLoading", p.status)
	}
}

func TestUpdate_WhenUpperYInConfirming_ShouldStartApply(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Init(sdktest.NewDeps(&sdktest.MockService{}).Deps)
	p.status = StatusConfirming

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	if cmd == nil {
		t.Error("after Y in confirming: cmd = nil, want non-nil")
	}
}

// Enter must NOT confirm a destructive apply: the Enter keystroke that launches
// `tfui apply` leaks into the alt-screen TUI, and accepting it would auto-apply
// without the user ever seeing the prompt. Confirmation requires an explicit
// affirmative key (y/Y), matching the ConfirmFrame convention (y/n/esc only).
func TestUpdate_WhenEnterInConfirming_ShouldNotStartApply(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.status = StatusConfirming

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("after enter in confirming: cmd = non-nil, want nil (enter must not confirm)")
	}
	if p.status != StatusConfirming {
		t.Errorf("after enter: status = %v, want StatusConfirming (still awaiting y/n)", p.status)
	}
	if len(svc.ApplyCalls) != 0 {
		t.Errorf("after enter: apply called %d times, want 0", len(svc.ApplyCalls))
	}
}

func TestUpdate_WhenNInConfirming_ShouldAbort(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusConfirming

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if p.status != sdk.StatusIdle {
		t.Errorf("after n in confirming: status = %v, want sdk.StatusIdle", p.status)
	}
}

func TestUpdate_WhenUpperNInConfirming_ShouldAbort(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusConfirming

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	if p.status != sdk.StatusIdle {
		t.Errorf("after N in confirming: status = %v, want sdk.StatusIdle", p.status)
	}
}

func TestUpdate_WhenEscInConfirming_ShouldAbort(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusConfirming

	p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.status != sdk.StatusIdle {
		t.Errorf("after esc in confirming: status = %v, want sdk.StatusIdle", p.status)
	}
}

func TestUpdate_WhenCtrlRInError_ShouldRetry(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Init(sdktest.NewDeps(&sdktest.MockService{}).Deps)
	p.status = sdk.StatusError

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Error("after r in error: cmd = nil, want non-nil (retry)")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("after r in error: status = %v, want sdk.StatusLoading", p.status)
	}
}

func TestUpdate_WhenUnknownMsg_ShouldReturnSelfWithNoCmd(t *testing.T) {
	p := New(&sdktest.MockService{})

	type unknownMsg struct{}
	result, cmd := p.Update(unknownMsg{})
	if cmd != nil {
		t.Errorf("Update(unknownMsg) cmd = %v, want nil", cmd)
	}
	if result != p {
		t.Error("Update(unknownMsg) returned different plugin reference")
	}
}

func TestView_WhenIdle_ShouldShowReadyMessage(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if !strings.Contains(view, "plan") || !strings.Contains(view, "apply") {
		t.Errorf("view should indicate run plan first then apply, got %q", view)
	}
}

func TestView_WhenConfirming_ShouldShowConfirmationPrompt(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusConfirming

	view := p.View(80, 24)
	if !strings.Contains(view, "apply") {
		t.Errorf("view should ask about applying changes, got %q", view)
	}
	if !strings.Contains(view, "y") || !strings.Contains(view, "n") {
		t.Errorf("view should show y/n prompt options, got %q", view)
	}
}

func TestView_WhenLoading_ShouldShowRunningState(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.timer.Start()

	view := p.View(80, 24)
	if !strings.Contains(view, "Applying") {
		t.Errorf("view should indicate applying changes, got %q", view)
	}
}

func TestView_WhenDone_ShouldShowSuccessMessage(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.timer.Start()
	p.timer.Stop()

	view := p.View(80, 24)
	if !strings.Contains(view, "Apply complete") {
		t.Errorf("view should indicate apply complete, got %q", view)
	}
	if !strings.Contains(view, "Duration") {
		t.Errorf("view should show duration, got %q", view)
	}
}

func TestView_WhenError_ShouldShowErrorMessage(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError
	p.errMsg = "something failed"

	view := p.View(80, 24)
	if !strings.Contains(view, "Apply failed") || !strings.Contains(view, "something failed") {
		t.Errorf("view should show error message 'Apply failed: something failed', got %q", view)
	}
}

func TestView_WhenInvalidStatus_ShouldReturnEmptyString(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestConfirm_WhenCalled_ShouldStartTimer(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Init(sdktest.NewDeps(&sdktest.MockService{}).Deps)
	p.status = StatusConfirming

	p.Confirm()
	if !p.timer.Running() {
		t.Error("timer not running after Confirm(), want running")
	}
}

func TestElapsed_WhenTimerRunning_ShouldReturnPositiveDuration(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.timer.Start()
	time.Sleep(10 * time.Millisecond)
	if p.Elapsed() == 0 {
		t.Error("Elapsed() = 0 while timer running, want > 0")
	}
}

func TestIsConfirming_GivenStatus_ShouldReturnCorrectBool(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	if p.IsConfirming() {
		t.Error("IsConfirming() = true in idle, want false")
	}
	p.status = StatusConfirming
	if !p.IsConfirming() {
		t.Error("IsConfirming() = false in confirming, want true")
	}
}

func TestStatus_WhenNew_ShouldReturnIdle(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	if p.Status() != sdk.StatusIdle {
		t.Errorf("Status() = %v, want sdk.StatusIdle", p.Status())
	}
}

func TestRunApply_WhenServiceSucceeds_ShouldReturnSuccessMsg(t *testing.T) {
	var captured ApplyResultMsg
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.planFile = "/tmp/foo.tfplan"

	cmd := p.runApply()
	captured = collectApplyResult(t, cmd)
	if captured.Err != nil {
		t.Errorf("ApplyResultMsg.Err = %v, want nil", captured.Err)
	}
}

func TestRunApply_WhenServiceFails_ShouldReturnErrorMsg(t *testing.T) {
	svc := &sdktest.MockService{
		ApplyFn: func(_ context.Context, _ sdk.ApplyOptions) error {
			return errors.New("apply failed")
		},
	}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.planFile = "/tmp/foo.tfplan"

	cmd := p.runApply()
	result := collectApplyResult(t, cmd)
	if result.Err == nil {
		t.Error("ApplyResultMsg.Err = nil, want error")
	}
}

func TestRunApply_WhenOptionsResolved_ShouldPassVarsAndVarFiles(t *testing.T) {
	var got sdk.ApplyOptions
	svc := &sdktest.MockService{
		ApplyFn: func(_ context.Context, opts sdk.ApplyOptions) error {
			got = opts
			return nil
		},
	}
	p := New(svc).(*Plugin)
	h := sdktest.NewDeps(svc)
	h.Ctx.VarFiles = []string{"prod.tfvars"}
	h.Ctx.Vars = map[string]string{"env": "prod"}
	h.Ctx.ExtraArgs = []string{"-no-color"}
	p.Init(h.Deps)
	p.planFile = "/tmp/x.tfplan"

	cmd := p.runApply()
	collectApplyResult(t, cmd)

	if len(got.VarFiles) != 1 || got.VarFiles[0] != "prod.tfvars" {
		t.Errorf("VarFiles = %v, want [prod.tfvars]", got.VarFiles)
	}
	if got.Vars["env"] != "prod" {
		t.Errorf("Vars[env] = %q, want prod", got.Vars["env"])
	}
	if len(got.ExtraArgs) != 1 || got.ExtraArgs[0] != "-no-color" {
		t.Errorf("ExtraArgs = %v, want [-no-color]", got.ExtraArgs)
	}
	if got.PlanFile != "/tmp/x.tfplan" {
		t.Errorf("PlanFile = %q, want /tmp/x.tfplan", got.PlanFile)
	}
}

func collectApplyResult(t *testing.T, cmd tea.Cmd) ApplyResultMsg {
	t.Helper()
	done := make(chan ApplyResultMsg, 1)
	var run func(c tea.Cmd)
	run = func(c tea.Cmd) {
		go func() {
			msg := c()
			if r, ok := msg.(ApplyResultMsg); ok {
				select {
				case done <- r:
				default:
				}
				return
			}
			if batch, ok := msg.(tea.BatchMsg); ok {
				for _, sub := range batch {
					run(sub)
				}
			}
		}()
	}
	run(cmd)
	select {
	case r := <-done:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("runApply did not produce ApplyResultMsg")
		return ApplyResultMsg{}
	}
}

func TestUpdate_WhenUnhandledKeyInIdle_ShouldDoNothing(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusIdle

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("after x in idle: cmd != nil, want nil")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("after x in idle: status = %v, want sdk.StatusIdle", p.status)
	}
}

func TestUpdate_WhenUnhandledKeyInError_ShouldDoNothing(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("after x in error: cmd != nil, want nil")
	}
}

func TestUpdate_WhenUnhandledKeyInConfirming_ShouldDoNothing(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusConfirming

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("after x in confirming: cmd != nil, want nil")
	}
	if p.status != StatusConfirming {
		t.Errorf("after x in confirming: status changed to %v", p.status)
	}
}

func TestUpdate_WhenKeyInLoading_ShouldDoNothing(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("after r in running: cmd != nil, want nil")
	}
}

func TestUpdate_WhenCtrlRInDone_ShouldNavigateToPlan(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("after ctrl+r in success: cmd = nil, want NavigateMsg")
	}
	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want sdk.NavigateMsg", msg)
	}
	if nav.PluginID != "plan" {
		t.Errorf("NavigateMsg.PluginID = %q, want %q", nav.PluginID, "plan")
	}
}

func TestBusy_GivenStatus_ShouldReturnTrueOnlyWhenLoading(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	if p.Busy() {
		t.Error("Busy() = true in idle, want false")
	}
	p.status = sdk.StatusLoading
	if !p.Busy() {
		t.Error("Busy() = false in loading, want true")
	}
	p.status = sdk.StatusDone
	if p.Busy() {
		t.Error("Busy() = true in done, want false")
	}
}

func TestHints_WhenIdle_ShouldReturnConfirmAndQuit(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusIdle

	hints := p.Hints()
	hasQ, hasEnter := false, false
	for _, h := range hints {
		if h.Key == "q" {
			hasQ = true
		}
		if h.Key == "Enter" {
			hasEnter = true
		}
	}
	if !hasQ {
		t.Error("Hints() in idle missing 'q quit'")
	}
	if !hasEnter {
		t.Error("Hints() in idle missing 'Enter confirm'")
	}
}

func TestHints_WhenConfirming_ShouldReturnYNAndCancel(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusConfirming

	hints := p.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() in confirming: len = %d, want 2", len(hints))
	}
	if hints[0].Key != "y/n" {
		t.Errorf("Hints()[0].Key = %q, want y/n", hints[0].Key)
	}
	if hints[1] != sdk.HintCancel {
		t.Errorf("Hints()[1] = %v, want HintCancel", hints[1])
	}
}

func TestHints_WhenLoading_ShouldReturnCancel(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice in loading state")
	}
	if hints[0].Key != "Esc" {
		t.Errorf("Hints()[0].Key = %q, want Esc", hints[0].Key)
	}
}

func TestHints_WhenDone_ShouldReturnRefreshAndCancel(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone

	hints := p.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() length = %d, want 2", len(hints))
	}
	if hints[0].Key != "^r" {
		t.Errorf("Hints()[0].Key = %q, want ^r", hints[0].Key)
	}
}

func TestHints_WhenError_ShouldReturnRetryAndCancel(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError

	hints := p.Hints()
	hasRetry := false
	for _, h := range hints {
		if h.Key == "^r" {
			hasRetry = true
		}
	}
	if !hasRetry {
		t.Error("Hints() in error missing '^r retry'")
	}
}

func TestHints_WhenUnknownStatus_ShouldReturnQuit(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.Status(99)

	hints := p.Hints()
	if len(hints) == 0 || hints[0].Key != "q" {
		t.Errorf("Hints() unknown status = %v, want first key 'q'", hints)
	}
}

func TestHandleContextChanged_WhenChdirChanges_ShouldClearPlanFileAndReset(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusError
	p.errMsg = "boom"
	p.planFile = "/tmp/from_old_chdir.tfplan"
	p.confirmed = true

	prev := &sdk.Context{Workspace: "default"}
	next := &sdk.Context{Service: svc}

	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Prev: prev, Next: next})
	if cmd != nil {
		t.Error("HandleContextChanged returned non-nil cmd")
	}
	if p.planFile != "" {
		t.Errorf("planFile = %q, want empty after chdir change (the leak bug ADR-0018 fixes)", p.planFile)
	}
	if p.confirmed {
		t.Error("confirmed should be reset to false on chdir change")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want StatusIdle", p.status)
	}
}

func TestHandleContextChanged_WhenOnlyPinsChange_ShouldBeNoOp(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.confirmed = true
	p.planFile = "/tmp/foo.tfplan"

	prev := &sdk.Context{Pins: []string{"a"}}
	next := &sdk.Context{Pins: []string{"a", "b"}}

	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Prev: prev, Next: next})
	if cmd != nil {
		t.Error("HandleContextChanged returned non-nil cmd")
	}
	if p.planFile != "/tmp/foo.tfplan" {
		t.Error("planFile cleared on pure target change, want preserved")
	}
	if !p.confirmed {
		t.Error("confirmed should be preserved when only targets change")
	}
}

func TestHandleContextChanged_WhenNextNil_ShouldBeNoOp(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.planFile = "/tmp/x.tfplan"

	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil})
	if cmd != nil {
		t.Error("HandleContextChanged with nil Next returned non-nil cmd")
	}
	if p.planFile == "" {
		t.Error("planFile cleared on nil Next, want preserved")
	}
}

func TestHandleContextChanged_WhenNextHasService_ShouldRebindService(t *testing.T) {
	oldSvc := &sdktest.MockService{}
	newSvc := &sdktest.MockService{}
	p := New(oldSvc).(*Plugin)
	p.Svc = oldSvc

	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{
		Next: &sdk.Context{Service: newSvc},
	})
	if cmd != nil {
		t.Error("HandleContextChanged returned non-nil cmd")
	}
	if p.Svc != newSvc {
		t.Error("svc not rebound to next.Service after chdir change")
	}
}

func TestActivate_WhenAutoApprove_ShouldStartApplyImmediately(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)

	cmd := p.Activate(Input{AutoApprove: true})
	if cmd == nil {
		t.Fatal("Activate(AutoApprove:true) cmd = nil, want non-nil")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want StatusLoading", p.status)
	}
	if !p.confirmed {
		t.Error("confirmed = false, want true after AutoApprove")
	}
}

func TestActivate_WhenNoAutoApprove_ShouldEnterConfirming(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	cmd := p.Activate(Input{})
	if cmd != nil {
		t.Errorf("Activate(empty) cmd = %v, want nil", cmd)
	}
	if p.status != StatusConfirming {
		t.Errorf("status = %v, want StatusConfirming", p.status)
	}
}

func TestActivate_WhenTargetsProvided_ShouldPassThroughToApply(t *testing.T) {
	var got sdk.ApplyOptions
	svc := &sdktest.MockService{
		ApplyFn: func(_ context.Context, opts sdk.ApplyOptions) error {
			got = opts
			return nil
		},
	}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)

	cmd := p.Activate(Input{AutoApprove: true, Targets: []string{"aws_instance.web"}})
	collectApplyResult(t, cmd)

	if len(got.Targets) != 1 || got.Targets[0] != "aws_instance.web" {
		t.Errorf("ApplyOptions.Targets = %v, want [aws_instance.web]", got.Targets)
	}
}

func TestActivate_WhenInputJSONSet_ShouldStoreOnPlugin(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	p.Activate(Input{JSON: true})
	if !p.input.JSON {
		t.Error("input.JSON = false, want true after Activate(Input{JSON: true})")
	}
}

func TestPlugin_WhenCancelWithNilCancelFn_ShouldNotPanic(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.cancelFn = nil
	p.Cancel()
}

func TestPlugin_WhenCancelWithCancelFn_ShouldCallIt(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	called := false
	p.cancelFn = func() { called = true }
	p.Cancel()
	if !called {
		t.Error("Cancel() should call cancelFn")
	}
	if p.cancelFn != nil {
		t.Error("Cancel() should set cancelFn to nil")
	}
}

func TestPlugin_WhenAutoApply_ShouldEnterLoadingAndStartTimer(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Init(sdktest.NewDeps(svc).Deps)
	p.SetPlanFile("/tmp/x.tfplan")

	cmd := p.AutoApply()
	if cmd == nil {
		t.Fatal("AutoApply() should return non-nil cmd")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want StatusLoading", p.status)
	}
	if !p.confirmed {
		t.Error("confirmed = false, want true")
	}
	if !p.timer.Running() {
		t.Error("timer not running after AutoApply")
	}
}

func TestPlugin_WhenKeyInLoadingStatus_ShouldDoNothing(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd != nil {
		t.Error("key in loading: cmd != nil, want nil")
	}

	_, cmd = p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Error("esc in loading: cmd != nil, want nil")
	}
}

func TestPlugin_WhenEscInDone_ShouldEmitDeactivateMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc in done should return cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd() = %T, want DeactivateMsg", msg)
	}
}

func TestPlugin_WhenEscInError_ShouldEmitDeactivateMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc in error should return cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd() = %T, want DeactivateMsg", msg)
	}
}

func TestPlugin_WhenNoInConfirming_ShouldEmitDeactivateMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = StatusConfirming

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("n in confirming: cmd = nil, want DeactivateMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd() = %T, want DeactivateMsg", msg)
	}
}

func TestPlugin_WhenOtherKeyInDone_ShouldDoNothing(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("unhandled key in done: cmd != nil, want nil")
	}
}

func TestExitCode_WhenStatusError_ShouldReturn1(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError

	if got := p.ExitCode(); got != 1 {
		t.Errorf("ExitCode() = %d, want 1", got)
	}
}

func TestExitCode_WhenStatusDone_ShouldReturn0(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone

	if got := p.ExitCode(); got != 0 {
		t.Errorf("ExitCode() = %d, want 0", got)
	}
}

func TestStack_WhenCalled_ShouldReturnInternalStack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	if p.Stack() == nil {
		t.Fatal("Stack() = nil, want non-nil")
	}
	if p.Stack() != p.stack {
		t.Error("Stack() should return the internal stack field")
	}
}

func TestHints_WhenStackHasFrame_ShouldDelegateToTopFrame(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	lw, ch := frames.NewLineWriter()
	lw.Close()
	p.stack.Push(frames.NewStreamFrame("test", ch, nil))

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() with frame on stack should delegate to top frame, got empty")
	}
}

func TestHints_WhenDoneWithLastStream_ShouldIncludeLHint(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	lw, ch := frames.NewLineWriter()
	lw.Close()
	p.lastStream = frames.NewStreamFrame("test", ch, nil)

	hints := p.Hints()
	hasL := false
	for _, h := range hints {
		if h.Key == "l" {
			hasL = true
		}
	}
	if !hasL {
		t.Error("Hints() when Done with lastStream should include 'l' hint")
	}
}

func TestHints_WhenErrorWithLastStream_ShouldIncludeLHint(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError
	lw, ch := frames.NewLineWriter()
	lw.Close()
	p.lastStream = frames.NewStreamFrame("test", ch, nil)

	hints := p.Hints()
	hasL := false
	for _, h := range hints {
		if h.Key == "l" {
			hasL = true
		}
	}
	if !hasL {
		t.Error("Hints() when Error with lastStream should include 'l' hint")
	}
}

func TestUpdate_WhenStreamLineMsgArrives_ShouldRouteToStack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	lw, ch := frames.NewLineWriter()
	p.stack.Push(frames.NewStreamFrame("test", ch, nil))

	_, cmd := p.Update(frames.StreamLineMsg{Line: "hello"})
	if cmd == nil {
		t.Fatal("Update(StreamLineMsg) with StreamFrame on stack should return non-nil cmd")
	}
	lw.Close()
}

func TestUpdate_WhenKeyMsgAndStackHasFrame_ShouldRouteToStack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	lw, ch := frames.NewLineWriter()
	lw.Close()
	p.stack.Push(frames.NewStreamFrame("test", ch, nil))

	next, _ := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next != p {
		t.Fatal("Update(KeyMsg) with stack frame should route to stack and return same plugin")
	}
}

func TestHandleKey_WhenDoneAndLKey_ShouldPushLastStreamOnStack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	lw, ch := frames.NewLineWriter()
	lw.Close()
	sf := frames.NewStreamFrame("test", ch, nil)
	p.lastStream = sf
	depthBefore := p.stack.Depth()

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if p.stack.Depth() != depthBefore+1 {
		t.Errorf("stack depth = %d, want %d after l key in Done", p.stack.Depth(), depthBefore+1)
	}
}

func TestHandleKey_WhenErrorAndLKey_ShouldPushLastStreamOnStack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError
	lw, ch := frames.NewLineWriter()
	lw.Close()
	sf := frames.NewStreamFrame("test", ch, nil)
	p.lastStream = sf
	depthBefore := p.stack.Depth()

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if p.stack.Depth() != depthBefore+1 {
		t.Errorf("stack depth = %d, want %d after l key in Error", p.stack.Depth(), depthBefore+1)
	}
}

func TestHandleKey_WhenDoneAndLKey_NoLastStream_ShouldDoNothing(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	depthBefore := p.stack.Depth()

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if p.stack.Depth() != depthBefore {
		t.Errorf("stack depth changed without lastStream; got %d, want %d", p.stack.Depth(), depthBefore)
	}
}

func TestHandleKey_WhenErrorAndLKey_NoLastStream_ShouldDoNothing(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError
	depthBefore := p.stack.Depth()

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if p.stack.Depth() != depthBefore {
		t.Errorf("stack depth changed without lastStream; got %d, want %d", p.stack.Depth(), depthBefore)
	}
}

func TestView_WhenStackHasFrame_ShouldDelegateToTopFrame(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	lw, ch := frames.NewLineWriter()
	lw.Close()
	p.stack.Push(frames.NewStreamFrame("test", ch, nil))
	_ = p.View(80, 24)
}
