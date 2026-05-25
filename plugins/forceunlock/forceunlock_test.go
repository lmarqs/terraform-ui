package forceunlock

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func newTestPlugin(svc *sdktest.MockService) (*Plugin, *sdktest.PluginDepsHarness) {
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	return p, h
}

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	if p.ID() != "forceunlock" {
		t.Errorf("ID() = %q, want %q", p.ID(), "forceunlock")
	}
	if p.Name() != "Force Unlock" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Force Unlock")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(nil); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	if p.Ready() {
		t.Error("Ready() should be false before activation")
	}
}

func TestActivate_WhenLockInfoPresent_ShouldRequestConfirmation(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "lock-abc-123", Who: "user@host"}

	cmd := p.Activate(Input{})
	if cmd == nil {
		t.Fatal("Activate() with lockInfo should return a cmd")
	}

	msg := cmd()
	reqMsg, ok := msg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("Activate() cmd returned %T, want sdk.RequestInputMsg", msg)
	}
	if reqMsg.Request.Mode != sdk.InputRequestBool {
		t.Errorf("request mode = %v, want InputRequestBool", reqMsg.Request.Mode)
	}
}

func TestActivate_WhenNoLockInfo_ShouldOfferManualEntry(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{})
	if cmd == nil {
		t.Fatal("Activate() without lockInfo should return a cmd")
	}

	msg := cmd()
	reqMsg, ok := msg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("Activate() cmd returned %T, want sdk.RequestInputMsg", msg)
	}
	if reqMsg.Request.Mode != sdk.InputRequestBool {
		t.Errorf("request mode = %v, want InputRequestBool (offer manual entry)", reqMsg.Request.Mode)
	}
}

func TestActivate_WhenAlreadyLoading_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	cmd := p.Activate(Input{})
	if cmd != nil {
		t.Error("Activate() while loading should return nil")
	}
}

func TestUpdate_WhenUnlockSuccess_ShouldSetDoneAndEmitEvents(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	p.lockID = "lock-123"

	result, cmd := p.Update(ForceUnlockResultMsg{LockID: "lock-123", Err: nil})
	pp := result.(*Plugin)

	if pp.status != sdk.StatusDone {
		t.Errorf("status = %v, want StatusDone", pp.status)
	}
	if cmd == nil {
		t.Fatal("success should return cmd (LockClearedEvent + PlanInvalidatedEvent)")
	}
}

func TestUpdate_WhenUnlockError_ShouldSetErrorStatus(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	p.lockID = "lock-123"

	result, cmd := p.Update(ForceUnlockResultMsg{LockID: "lock-123", Err: errors.New("denied")})
	pp := result.(*Plugin)

	if pp.status != sdk.StatusError {
		t.Errorf("status = %v, want StatusError", pp.status)
	}
	if pp.errMsg == "" {
		t.Error("errMsg should be set on error")
	}
	if cmd != nil {
		t.Error("error should return nil cmd")
	}
}

func TestUpdate_WhenQKeyPressed_ShouldDeactivate(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("q cmd() = %T, want sdk.DeactivateMsg", msg)
	}
}

func TestUpdate_WhenEscKeyPressed_ShouldDeactivate(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("esc cmd() = %T, want sdk.DeactivateMsg", msg)
	}
}

func TestUpdate_WhenCtrlRInError_ShouldReactivate(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusError
	p.lockInfo = &sdk.StateLock{ID: "lock-123"}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("ctrl+r in error should return a cmd (re-activate)")
	}
}

func TestUpdate_WhenCtrlRInDone_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("ctrl+r in done state should return nil")
	}
}

func TestView_WhenIdleNoLock_ShouldReturnNonEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() idle with no lock should not be empty")
	}
}

func TestView_WhenIdleWithLock_ShouldReturnNonEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusIdle
	p.lockInfo = &sdk.StateLock{ID: "lock-abc", Who: "user@host"}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() idle with lock should not be empty")
	}
}

func TestView_WhenLoading_ShouldReturnNonEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	p.lockID = "lock-xyz"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() loading should not be empty")
	}
}

func TestView_WhenDone_ShouldReturnNonEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.lockID = "lock-xyz"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() done should not be empty")
	}
}

func TestView_WhenError_ShouldReturnNonEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusError
	p.errMsg = "something went wrong"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() error should not be empty")
	}
}

func TestHints_WhenIdle_ShouldReturnBackAndQuitHints(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusIdle

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() idle should not be empty")
	}
}

func TestHints_WhenLoading_ShouldReturnBackAndQuitHints(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() loading should not be empty")
	}
}

func TestHints_WhenDone_ShouldReturnBackHints(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusDone

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() done should not be empty")
	}
}

func TestHints_WhenError_ShouldReturnRetryHints(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusError

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() error should not be empty")
	}
}

func TestHandleLockDetected_WhenCalled_ShouldStoreLockInfo(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	lock := &sdk.StateLock{ID: "lock-new", Who: "other@host"}
	cmd := p.HandleLockDetected(sdk.LockDetectedEvent{Lock: lock})

	if p.lockInfo == nil {
		t.Fatal("HandleLockDetected should store lockInfo")
	}
	if p.lockInfo.ID != "lock-new" {
		t.Errorf("lockInfo.ID = %q, want %q", p.lockInfo.ID, "lock-new")
	}
	if cmd != nil {
		t.Error("HandleLockDetected should return nil cmd")
	}
}

func TestHandleLockCleared_WhenCalled_ShouldClearLockInfo(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "old-lock"}

	cmd := p.HandleLockCleared(sdk.LockClearedEvent{})

	if p.lockInfo != nil {
		t.Error("HandleLockCleared should clear lockInfo")
	}
	if cmd != nil {
		t.Error("HandleLockCleared should return nil cmd")
	}
}

func TestConfirmUnlock_WhenConfirmed_ShouldCallServiceWithLockID(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "lock-abc"}

	cmd := p.Activate(Input{})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	// Simulate user confirming "y" — returns ForceUnlockStartMsg
	startCmd := reqMsg.Request.Callback("y")
	if startCmd == nil {
		t.Fatal("confirm callback should return a cmd")
	}
	startMsg := startCmd()
	start, ok := startMsg.(ForceUnlockStartMsg)
	if !ok {
		t.Fatalf("callback cmd returned %T, want ForceUnlockStartMsg", startMsg)
	}
	if start.LockID != "lock-abc" {
		t.Errorf("ForceUnlockStartMsg.LockID = %q, want %q", start.LockID, "lock-abc")
	}

	// Feed ForceUnlockStartMsg into Update — triggers the actual unlock
	_, execCmd := p.Update(start)
	if execCmd == nil {
		t.Fatal("Update(ForceUnlockStartMsg) should return exec cmd")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want StatusLoading", p.status)
	}

	// Execute the batch command and find ForceUnlockResultMsg
	execMsg := execCmd()
	batchMsg, ok := execMsg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("exec cmd returned %T, want tea.BatchMsg", execMsg)
	}
	var result ForceUnlockResultMsg
	found := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(ForceUnlockResultMsg); ok {
			result = r
			found = true
		}
	}
	if !found {
		t.Fatal("batch did not contain ForceUnlockResultMsg")
	}
	if result.Err != nil {
		t.Errorf("ForceUnlockResultMsg.Err = %v, want nil", result.Err)
	}
	if result.LockID != "lock-abc" {
		t.Errorf("ForceUnlockResultMsg.LockID = %q, want %q", result.LockID, "lock-abc")
	}
	if len(svc.ForceUnlockCalls) == 0 || svc.ForceUnlockCalls[0] != "lock-abc" {
		t.Errorf("service.ForceUnlock called with %v, want [lock-abc]", svc.ForceUnlockCalls)
	}
}

func TestConfirmUnlock_WhenServiceFails_ShouldReturnError(t *testing.T) {
	svc := &sdktest.MockService{
		ForceUnlockFn: func(_ context.Context, _ string) error {
			return errors.New("denied")
		},
	}
	p, _ := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "lock-err"}

	cmd := p.Activate(Input{})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	startCmd := reqMsg.Request.Callback("y")
	startMsg := startCmd()
	start := startMsg.(ForceUnlockStartMsg)

	_, execCmd := p.Update(start)
	execMsg := execCmd()
	batchMsg := execMsg.(tea.BatchMsg)
	var result ForceUnlockResultMsg
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(ForceUnlockResultMsg); ok {
			result = r
		}
	}

	if result.Err == nil {
		t.Error("ForceUnlockResultMsg.Err = nil, want error")
	}
}

func TestConfirmUnlock_WhenDeclined_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "lock-abc"}

	cmd := p.Activate(Input{})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	// Simulate user declining "n"
	result := reqMsg.Request.Callback("n")
	if result != nil {
		t.Error("declining should return nil cmd")
	}
}

func TestManualEntry_WhenConfirmedWithLockID_ShouldCallService(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	// First confirm: "Enter lock ID manually? (y/n)" — answer yes
	manualCmd := reqMsg.Request.Callback("y")
	if manualCmd == nil {
		t.Fatal("confirming manual entry should return a cmd")
	}

	// Second step: InputText("Lock ID:")
	msg2 := manualCmd()
	reqMsg2, ok := msg2.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("manual entry cmd returned %T, want sdk.RequestInputMsg", msg2)
	}
	if reqMsg2.Request.Mode != sdk.InputRequestText {
		t.Errorf("request mode = %v, want InputRequestText", reqMsg2.Request.Mode)
	}

	// User types a lock ID
	confirmCmd := reqMsg2.Request.Callback("manual-lock-id")
	if confirmCmd == nil {
		t.Fatal("entering lock ID should return a cmd")
	}

	// Third step: confirmation prompt
	msg3 := confirmCmd()
	reqMsg3, ok := msg3.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("confirm cmd returned %T, want sdk.RequestInputMsg", msg3)
	}
	if reqMsg3.Request.Mode != sdk.InputRequestBool {
		t.Errorf("request mode = %v, want InputRequestBool", reqMsg3.Request.Mode)
	}

	// User confirms — returns ForceUnlockStartMsg
	startCmd := reqMsg3.Request.Callback("y")
	if startCmd == nil {
		t.Fatal("confirming unlock should return a cmd")
	}
	startMsg := startCmd()
	start, ok := startMsg.(ForceUnlockStartMsg)
	if !ok {
		t.Fatalf("confirm cmd returned %T, want ForceUnlockStartMsg", startMsg)
	}

	// Feed into Update
	_, execCmd := p.Update(start)
	execMsg := execCmd()
	batchMsg, ok := execMsg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("exec cmd returned %T, want tea.BatchMsg", execMsg)
	}
	var result ForceUnlockResultMsg
	found := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(ForceUnlockResultMsg); ok {
			result = r
			found = true
		}
	}
	if !found {
		t.Fatal("batch did not contain ForceUnlockResultMsg")
	}
	if result.LockID != "manual-lock-id" {
		t.Errorf("LockID = %q, want %q", result.LockID, "manual-lock-id")
	}
	if len(svc.ForceUnlockCalls) == 0 || svc.ForceUnlockCalls[0] != "manual-lock-id" {
		t.Errorf("service called with %v, want [manual-lock-id]", svc.ForceUnlockCalls)
	}
}

func TestManualEntry_WhenEmptyLockID_ShouldDeactivate(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	// Confirm manual entry
	manualCmd := reqMsg.Request.Callback("y")
	msg2 := manualCmd()
	reqMsg2 := msg2.(sdk.RequestInputMsg)

	// User submits empty lock ID
	deactivateCmd := reqMsg2.Request.Callback("")
	if deactivateCmd == nil {
		t.Fatal("empty lock ID should return a deactivate cmd")
	}
	resultMsg := deactivateCmd()
	if _, ok := resultMsg.(sdk.DeactivateMsg); !ok {
		t.Errorf("empty lock ID cmd() = %T, want sdk.DeactivateMsg", resultMsg)
	}
}

func TestManualEntry_WhenDeclined_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	cmd := p.Activate(Input{})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	// Decline manual entry
	result := reqMsg.Request.Callback("n")
	if result != nil {
		t.Error("declining manual entry should return nil cmd")
	}
}

func TestView_WhenUnknownStatus_ShouldReturnEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = 99 // unknown status

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View() with unknown status should return empty, got %q", view)
	}
}

func TestUpdate_WhenUnhandledMsg_ShouldReturnSelfAndNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)

	result, cmd := p.Update(struct{}{})
	if result.(*Plugin) != p {
		t.Error("unhandled msg should return same plugin")
	}
	if cmd != nil {
		t.Error("unhandled msg should return nil cmd")
	}
}

func TestPlugin_WhenCancelWithNilFn_ShouldNotPanic(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.cancelFn = nil
	p.Cancel()
}

func TestPlugin_WhenCancelWithFn_ShouldCallAndClear(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
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

func TestUpdate_WhenTimerTickMsg_ShouldReturnTickCmd(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	p.timer.Start()

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd == nil {
		t.Error("TimerTickMsg while timer running: cmd = nil, want non-nil")
	}
}

func TestUpdate_WhenTimerTickMsgNotRunning_ShouldReturnNilCmd(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusDone

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd != nil {
		t.Error("TimerTickMsg while timer stopped: cmd != nil, want nil")
	}
}

func TestUpdate_WhenKeyOtherThanQEscCtrlR_ShouldDoNothing(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("unhandled key in done: cmd != nil, want nil")
	}
}

func TestUpdate_WhenForceUnlockResultSuccessWithRunningTimer_ShouldStopTimerAndEmitEvents(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	p.lockID = "lock-123"
	p.timer.Start()

	result, cmd := p.Update(ForceUnlockResultMsg{LockID: "lock-123", Err: nil})
	pp := result.(*Plugin)

	if pp.status != sdk.StatusDone {
		t.Errorf("status = %v, want StatusDone", pp.status)
	}
	if pp.timer.Running() {
		t.Error("timer should be stopped after result")
	}
	if pp.lockInfo != nil {
		t.Error("lockInfo should be nil after success")
	}
	if cmd == nil {
		t.Fatal("success should return cmd (LockClearedEvent + PlanInvalidatedEvent)")
	}
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want tea.BatchMsg", msg)
	}
	foundCleared := false
	foundInvalidated := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		subMsg := subCmd()
		switch subMsg.(type) {
		case sdk.LockClearedEvent:
			foundCleared = true
		case sdk.PlanInvalidatedEvent:
			foundInvalidated = true
		}
	}
	if !foundCleared {
		t.Error("batch should contain LockClearedEvent")
	}
	if !foundInvalidated {
		t.Error("batch should contain PlanInvalidatedEvent")
	}
}

func TestUpdate_WhenForceUnlockResultErrorWithRunningTimer_ShouldStopTimer(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	p.lockID = "lock-err"
	p.timer.Start()

	result, _ := p.Update(ForceUnlockResultMsg{LockID: "lock-err", Err: errors.New("denied")})
	pp := result.(*Plugin)

	if pp.timer.Running() {
		t.Error("timer should be stopped after error result")
	}
}

func TestHandleContextChanged_ShouldResetState(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.status = sdk.StatusError
	p.lockID = "abc"
	p.errMsg = "boom"
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: &sdk.Context{Service: svc}})
	if cmd != nil {
		t.Error("HandleContextChanged returned non-nil cmd")
	}
	if p.status != sdk.StatusIdle || p.lockID != "" || p.errMsg != "" {
		t.Errorf("state not reset: status=%v lockID=%q errMsg=%q", p.status, p.lockID, p.errMsg)
	}
}

func TestHandleContextChanged_WhenNextNil_ShouldBeNoOp(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.lockID = "keep"
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil})
	if cmd != nil {
		t.Error("HandleContextChanged with nil Next returned non-nil cmd")
	}
	if p.lockID != "keep" {
		t.Errorf("lockID mutated, got %q", p.lockID)
	}
}
