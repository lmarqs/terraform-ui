package forceunlock

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct {
	forceUnlockErr error
	forceUnlockID  string
}

func (m *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return nil, nil
}
func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (m *mockService) StateList(_ context.Context) ([]sdk.Resource, error) {
	return nil, nil
}
func (m *mockService) Show(_ context.Context, _ string) (string, error)  { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)       { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	return nil
}
func (m *mockService) WorkspaceDelete(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	return nil
}
func (m *mockService) StateRm(_ context.Context, _ string) error            { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error       { return nil }
func (m *mockService) Import(_ context.Context, _, _ string) error          { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error              { return nil }
func (m *mockService) Untaint(_ context.Context, _ string) error            { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error) { return nil, nil }
func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}
func (m *mockService) Refresh(_ context.Context) error                     { return nil }
func (m *mockService) Init(_ context.Context) error                        { return nil }
func (m *mockService) Version(_ context.Context) (*sdk.VersionInfo, error) { return nil, nil }
func (m *mockService) ForceUnlock(_ context.Context, lockID string) error {
	m.forceUnlockID = lockID
	return m.forceUnlockErr
}
func (m *mockService) WithDir(_ string) sdk.Service { return m }

func newTestPlugin(svc sdk.Service) *Plugin {
	p := New(svc).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	return p
}

func TestNew(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	if p.ID() != "forceunlock" {
		t.Errorf("ID() = %q, want %q", p.ID(), "forceunlock")
	}
	if p.Name() != "Force Unlock" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Force Unlock")
	}
	if p.Description() != "Remove a stale state lock" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Remove a stale state lock")
	}
	if p.Ready() {
		t.Error("Ready() = true, want false for new plugin")
	}
}

func TestConfigure(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	if err := p.Configure(nil); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestInit(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx := &sdk.Context{
		Service: svc,
		Logger:  logger,
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() should return nil cmd")
	}

	pp := p.(*Plugin)
	if pp.svc != svc {
		t.Error("Init() should store service from context")
	}
	if pp.log != logger {
		t.Error("Init() should store logger from context")
	}
}

func TestActivate_WithLockInfo(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "lock-abc-123", Who: "user@host"}

	cmd := p.Activate()
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

func TestActivate_WithoutLockInfo(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)

	cmd := p.Activate()
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

func TestActivate_WhenLoading_Noop(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() while loading should return nil")
	}
}

func TestUpdate_ForceUnlockResult_Success(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
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

func TestUpdate_ForceUnlockResult_Error(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
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

func TestUpdate_KeyQ_Deactivates(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
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

func TestUpdate_KeyEsc_Deactivates(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
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

func TestUpdate_CtrlR_InError_Reactivates(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusError
	p.lockInfo = &sdk.StateLock{ID: "lock-123"}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("ctrl+r in error should return a cmd (re-activate)")
	}
}

func TestUpdate_CtrlR_InDone_Noop(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("ctrl+r in done state should return nil")
	}
}

func TestView_Idle_NoLock(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() idle with no lock should not be empty")
	}
}

func TestView_Idle_WithLock(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusIdle
	p.lockInfo = &sdk.StateLock{ID: "lock-abc", Who: "user@host"}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() idle with lock should not be empty")
	}
}

func TestView_Loading(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	p.lockID = "lock-xyz"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() loading should not be empty")
	}
}

func TestView_Done(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.lockID = "lock-xyz"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() done should not be empty")
	}
}

func TestView_Error(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusError
	p.errMsg = "something went wrong"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View() error should not be empty")
	}
}

func TestHints_Done(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() done should not be empty")
	}
}

func TestHints_Error(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusError

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() error should not be empty")
	}
}

func TestHandleLockDetected(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)

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

func TestHandleLockCleared(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "old-lock"}

	cmd := p.HandleLockCleared(sdk.LockClearedEvent{})

	if p.lockInfo != nil {
		t.Error("HandleLockCleared should clear lockInfo")
	}
	if cmd != nil {
		t.Error("HandleLockCleared should return nil cmd")
	}
}

func TestHandleChdirChanged(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.lockInfo = &sdk.StateLock{ID: "old"}
	p.lockID = "old"
	p.errMsg = "something"

	cmd := p.HandleChdirChanged(sdk.ChdirChangedEvent{AbsPath: "/new/path"})

	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want StatusIdle after chdir change", p.status)
	}
	if p.lockInfo != nil {
		t.Error("lockInfo should be cleared on chdir change")
	}
	if p.lockID != "" {
		t.Error("lockID should be cleared on chdir change")
	}
	if p.errMsg != "" {
		t.Error("errMsg should be cleared on chdir change")
	}
	if cmd != nil {
		t.Error("HandleChdirChanged should return nil cmd")
	}
}

func TestConfirmUnlock_Callback_CallsService(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "lock-abc"}

	cmd := p.Activate()
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	// Simulate user confirming "y"
	unlockCmd := reqMsg.Request.Callback("y")
	if unlockCmd == nil {
		t.Fatal("confirm callback should return a cmd")
	}

	// Execute the unlock command — it calls ForceUnlock
	resultMsg := unlockCmd()
	result, ok := resultMsg.(ForceUnlockResultMsg)
	if !ok {
		t.Fatalf("unlock cmd returned %T, want ForceUnlockResultMsg", resultMsg)
	}
	if result.Err != nil {
		t.Errorf("ForceUnlockResultMsg.Err = %v, want nil", result.Err)
	}
	if result.LockID != "lock-abc" {
		t.Errorf("ForceUnlockResultMsg.LockID = %q, want %q", result.LockID, "lock-abc")
	}
	if svc.forceUnlockID != "lock-abc" {
		t.Errorf("service.ForceUnlock called with %q, want %q", svc.forceUnlockID, "lock-abc")
	}
}

func TestConfirmUnlock_Callback_ReturnsError(t *testing.T) {
	svc := &mockService{forceUnlockErr: errors.New("denied")}
	p := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "lock-err"}

	cmd := p.Activate()
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	unlockCmd := reqMsg.Request.Callback("y")
	resultMsg := unlockCmd()
	result := resultMsg.(ForceUnlockResultMsg)

	if result.Err == nil {
		t.Error("ForceUnlockResultMsg.Err = nil, want error")
	}
}

func TestConfirmUnlock_Callback_Declined(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "lock-abc"}

	cmd := p.Activate()
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	// Simulate user declining "n"
	result := reqMsg.Request.Callback("n")
	if result != nil {
		t.Error("declining should return nil cmd")
	}
}

func TestManualEntry_Callback_ConfirmYes(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)

	cmd := p.Activate()
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

	// User confirms
	unlockCmd := reqMsg3.Request.Callback("y")
	if unlockCmd == nil {
		t.Fatal("confirming unlock should return a cmd")
	}

	// Execute the actual unlock
	resultMsg := unlockCmd()
	result, ok := resultMsg.(ForceUnlockResultMsg)
	if !ok {
		t.Fatalf("unlock cmd returned %T, want ForceUnlockResultMsg", resultMsg)
	}
	if result.LockID != "manual-lock-id" {
		t.Errorf("LockID = %q, want %q", result.LockID, "manual-lock-id")
	}
	if svc.forceUnlockID != "manual-lock-id" {
		t.Errorf("service called with %q, want %q", svc.forceUnlockID, "manual-lock-id")
	}
}

func TestManualEntry_Callback_EmptyLockID(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)

	cmd := p.Activate()
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

func TestManualEntry_Callback_DeclinedManualEntry(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)

	cmd := p.Activate()
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	// Decline manual entry
	result := reqMsg.Request.Callback("n")
	if result != nil {
		t.Error("declining manual entry should return nil cmd")
	}
}

func TestView_DefaultStatus(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = 99 // unknown status

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View() with unknown status should return empty, got %q", view)
	}
}

func TestUpdate_UnhandledMsg(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)

	result, cmd := p.Update(struct{}{})
	if result.(*Plugin) != p {
		t.Error("unhandled msg should return same plugin")
	}
	if cmd != nil {
		t.Error("unhandled msg should return nil cmd")
	}
}
