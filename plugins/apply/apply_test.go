package apply

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct {
	applyErr error
}

func (m *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return m.applyErr }
func (m *mockService) StateList(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
	return nil, nil
}
func (m *mockService) Show(_ context.Context, _ string) (string, error) { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)      { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"default"}, nil
}
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	return nil
}
func (m *mockService) WorkspaceDelete(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	return nil
}
func (m *mockService) StateRm(_ context.Context, _ string) error                    { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error               { return nil }
func (m *mockService) Import(_ context.Context, _, _ string) error                  { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error                      { return nil }
func (m *mockService) Untaint(_ context.Context, _ string) error                    { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error)         { return nil, nil }
func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) { return nil, nil }
func (m *mockService) Refresh(_ context.Context) error                              { return nil }
func (m *mockService) Init(_ context.Context) error                                 { return nil }
func (m *mockService) ForceUnlock(_ context.Context, _ string) error                { return nil }
func (m *mockService) Version(_ context.Context) (*sdk.VersionInfo, error)          { return nil, nil }
func (m *mockService) WithDir(_ string) sdk.Service                                 { return m }

func TestNew(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	if p.ID() != "apply" {
		t.Errorf("ID() = %q, want %q", p.ID(), "apply")
	}
	if p.Name() != "Apply" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Apply")
	}
	if p.Description() != "Apply terraform changes to infrastructure" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Apply terraform changes to infrastructure")
	}
	if p.Ready() {
		t.Error("Ready() = true before apply completes, want false")
	}
}

func TestConfigure(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	err := p.Configure(map[string]interface{}{"key": "value"})
	if err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestInit(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Workspace:  "default",
		Service:    svc,
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() returned non-nil cmd, want nil")
	}
}

func TestSetTargets(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	targets := []string{"aws_instance.web", "aws_s3_bucket.data"}
	p.SetTargets(targets)
	if len(p.targets) != 2 {
		t.Errorf("SetTargets: len(targets) = %d, want 2", len(p.targets))
	}
}

func TestRequestApply(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	p.RequestApply()
	if p.status != StatusConfirming {
		t.Errorf("status = %v, want StatusConfirming", p.status)
	}
	if p.confirmed {
		t.Error("confirmed = true, want false")
	}
	if p.IsConfirming() != true {
		t.Error("IsConfirming() = false, want true")
	}
}

func TestConfirm(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	cmd := p.Confirm()
	if cmd == nil {
		t.Error("Confirm() returned nil cmd, want non-nil (batch)")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want sdk.StatusLoading", p.status)
	}
	if !p.confirmed {
		t.Error("confirmed = false, want true")
	}
}

func TestCancel(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming
	p.confirmed = true

	p.Cancel()
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want sdk.StatusIdle", p.status)
	}
	if p.confirmed {
		t.Error("confirmed = true after cancel, want false")
	}
}

func TestUpdateApplyResultMsgSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusLoading

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
	if updated.elapsed != 5*time.Second {
		t.Errorf("elapsed = %v, want 5s", updated.elapsed)
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdateApplyResultMsgError(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
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

func TestUpdateTickMsgRunning(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusLoading
	p.startTime = time.Now().Add(-5 * time.Second)

	_, cmd := p.Update(TickMsg(time.Now()))
	if cmd == nil {
		t.Error("Update(TickMsg) in sdk.StatusLoading: cmd = nil, want non-nil (next tick)")
	}
	if p.elapsed < 4*time.Second {
		t.Errorf("elapsed = %v, want >= 4s", p.elapsed)
	}
}

func TestUpdateTickMsgNotRunning(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone

	_, cmd := p.Update(TickMsg(time.Now()))
	if cmd != nil {
		t.Error("Update(TickMsg) in sdk.StatusDone: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgIdle_Enter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.status != StatusConfirming {
		t.Errorf("after enter in idle: status = %v, want StatusConfirming", p.status)
	}
}

func TestUpdateKeyMsgConfirming_Yes(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Error("after y in confirming: cmd = nil, want non-nil")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("after y: status = %v, want sdk.StatusLoading", p.status)
	}
}

func TestUpdateKeyMsgConfirming_YUppercase(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	if cmd == nil {
		t.Error("after Y in confirming: cmd = nil, want non-nil")
	}
}

func TestUpdateKeyMsgConfirming_Enter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("after enter in confirming: cmd = nil, want non-nil")
	}
}

func TestUpdateKeyMsgConfirming_No(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if p.status != sdk.StatusIdle {
		t.Errorf("after n in confirming: status = %v, want sdk.StatusIdle", p.status)
	}
}

func TestUpdateKeyMsgConfirming_NUppercase(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	if p.status != sdk.StatusIdle {
		t.Errorf("after N in confirming: status = %v, want sdk.StatusIdle", p.status)
	}
}

func TestUpdateKeyMsgConfirming_Esc(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.status != sdk.StatusIdle {
		t.Errorf("after esc in confirming: status = %v, want sdk.StatusIdle", p.status)
	}
}

func TestUpdateKeyMsgError_Retry(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusError

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Error("after r in error: cmd = nil, want non-nil (retry)")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("after r in error: status = %v, want sdk.StatusLoading", p.status)
	}
}

func TestUpdateUnknownMsg(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	type unknownMsg struct{}
	result, cmd := p.Update(unknownMsg{})
	if cmd != nil {
		t.Errorf("Update(unknownMsg) cmd = %v, want nil", cmd)
	}
	if result != p {
		t.Error("Update(unknownMsg) returned different plugin reference")
	}
}

func TestViewIdle(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusIdle) returned empty string")
	}
}

func TestViewConfirming(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusConfirming) returned empty string")
	}
}

func TestViewConfirmingWithTargets(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming
	p.targets = []string{"aws_instance.web", "aws_s3_bucket.data"}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusConfirming, with targets) returned empty string")
	}
}

func TestViewRunning(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusLoading
	p.elapsed = 10 * time.Second

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusLoading) returned empty string")
	}
}

func TestViewSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.elapsed = 30 * time.Second

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone) returned empty string")
	}
}

func TestViewError(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusError
	p.errMsg = "something failed"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusError) returned empty string")
	}
}

func TestViewDefaultStatus(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Second, "5s"},
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m0s"},
		{90 * time.Second, "1m30s"},
		{125 * time.Second, "2m5s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestElapsed(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.elapsed = 42 * time.Second
	if p.Elapsed() != 42*time.Second {
		t.Errorf("Elapsed() = %v, want 42s", p.Elapsed())
	}
}

func TestIsConfirming(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	if p.IsConfirming() {
		t.Error("IsConfirming() = true in idle, want false")
	}
	p.status = StatusConfirming
	if !p.IsConfirming() {
		t.Error("IsConfirming() = false in confirming, want true")
	}
}

func TestStatusGetter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	if p.Status() != sdk.StatusIdle {
		t.Errorf("Status() = %v, want sdk.StatusIdle", p.Status())
	}
}

func TestRunApplyCmd(t *testing.T) {
	svc := &mockService{applyErr: nil}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.startTime = time.Now()

	cmd := p.runApply()
	msg := cmd()

	result, ok := msg.(ApplyResultMsg)
	if !ok {
		t.Fatalf("runApply cmd returned %T, want ApplyResultMsg", msg)
	}
	if result.Err != nil {
		t.Errorf("ApplyResultMsg.Err = %v, want nil", result.Err)
	}
}

func TestRunApplyCmdError(t *testing.T) {
	svc := &mockService{applyErr: errors.New("apply failed")}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.startTime = time.Now()

	cmd := p.runApply()
	msg := cmd()

	result, ok := msg.(ApplyResultMsg)
	if !ok {
		t.Fatalf("runApply cmd returned %T, want ApplyResultMsg", msg)
	}
	if result.Err == nil {
		t.Error("ApplyResultMsg.Err = nil, want error")
	}
}

func TestUpdateKeyMsgIdle_OtherKey(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	// A key other than enter in idle should do nothing
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("after x in idle: cmd != nil, want nil")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("after x in idle: status = %v, want sdk.StatusIdle", p.status)
	}
}

func TestUpdateKeyMsgError_OtherKey(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusError

	// A key other than r in error should do nothing
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("after x in error: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgConfirming_OtherKey(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	// A key other than y/n/enter/esc in confirming should do nothing
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("after x in confirming: cmd != nil, want nil")
	}
	if p.status != StatusConfirming {
		t.Errorf("after x in confirming: status changed to %v", p.status)
	}
}

func TestUpdateKeyMsgRunning(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusLoading

	// Keys during running state should do nothing
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("after r in running: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgSuccess_CtrlR_ShouldNavigateToPlan(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
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

func TestFormatDurationZero(t *testing.T) {
	got := formatDuration(0)
	if got != "0s" {
		t.Errorf("formatDuration(0) = %q, want %q", got, "0s")
	}
}

func TestBusy(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

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

func TestTargets(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	if p.Targets() != nil {
		t.Errorf("Targets() = %v, want nil", p.Targets())
	}
	targets := []string{"aws_instance.web", "aws_s3_bucket.data"}
	p.SetTargets(targets)
	got := p.Targets()
	if len(got) != 2 || got[0] != "aws_instance.web" || got[1] != "aws_s3_bucket.data" {
		t.Errorf("Targets() = %v, want %v", got, targets)
	}
}

func TestHints_WhenIdle_ShouldReturnConfirmAndBack(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice in idle state")
	}
	hasBack := false
	hasConfirm := false
	for _, h := range hints {
		if h.Key == "q" && h.Description == "back" {
			hasBack = true
		}
		if h.Key == "Enter" && h.Description == "confirm" {
			hasConfirm = true
		}
	}
	if !hasBack {
		t.Error("Hints() in idle missing 'q back'")
	}
	if !hasConfirm {
		t.Error("Hints() in idle missing 'Enter confirm'")
	}
}

func TestHints_WhenConfirming_ShouldReturnYNAndCancel(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	hints := p.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() in confirming: len = %d, want 2", len(hints))
	}
	if hints[0].Key != "y/n" || hints[0].Description != "confirm" {
		t.Errorf("Hints()[0] = %v, want {y/n confirm}", hints[0])
	}
	if hints[1] != sdk.HintCancel {
		t.Errorf("Hints()[1] = %v, want HintCancel", hints[1])
	}
}

func TestHints_WhenLoading_ShouldReturnCancel(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusLoading

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice in loading state")
	}
	if hints[0].Key != "Esc" || hints[0].Description != "cancel" {
		t.Errorf("Hints()[0] = %v, want {Esc cancel}", hints[0])
	}
}

func TestHints_WhenDone_ShouldReturnRefreshAndCancel(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone

	hints := p.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() length = %d, want 2", len(hints))
	}
	if hints[0].Key != "^r" || hints[0].Description != "refresh" {
		t.Errorf("Hints()[0] = %v, want {^r refresh}", hints[0])
	}
	if hints[1].Key != "Esc" || hints[1].Description != "cancel" {
		t.Errorf("Hints()[1] = %v, want {Esc cancel}", hints[1])
	}
}

func TestHints_WhenError_ShouldReturnRetryAndBack(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusError

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice in error state")
	}
	hasRetry := false
	hasBack := false
	for _, h := range hints {
		if h.Key == "^r" && h.Description == "retry" {
			hasRetry = true
		}
		if h.Key == "Esc" && h.Description == "cancel" {
			hasBack = true
		}
	}
	if !hasRetry {
		t.Error("Hints() in error missing '^r retry'")
	}
	if !hasBack {
		t.Error("Hints() in error missing 'q back'")
	}
}

func TestHints_WhenUnknownStatus_ShouldReturnBack(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.Status(99)

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice for unknown status")
	}
	if hints[0].Key != "q" || hints[0].Description != "back" {
		t.Errorf("Hints()[0] = %v, want {q back}", hints[0])
	}
}

func TestHandleChdirChanged_ShouldResetState(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusError
	p.errMsg = "some error"

	cmd := p.HandleChdirChanged(sdk.ChdirChangedEvent{
		RelPath: "modules/vpc",
		AbsPath: "/abs/modules/vpc",
		Count:   3,
	})
	if cmd != nil {
		t.Error("HandleChdirChanged() returned non-nil cmd")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want sdk.StatusIdle", p.status)
	}
	if p.errMsg != "" {
		t.Errorf("errMsg = %q, want empty", p.errMsg)
	}
	if p.scopedContext != "/abs/modules/vpc" {
		t.Errorf("scopedContext = %q, want %q", p.scopedContext, "/abs/modules/vpc")
	}
}

func TestHandlePlanCompleted_ShouldStoreTotalResources(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	cmd := p.HandlePlanCompleted(sdk.PlanCompletedEvent{
		ResourceCount: 42,
		Summary:       &sdk.PlanSummary{},
	})
	if cmd != nil {
		t.Error("HandlePlanCompleted() returned non-nil cmd")
	}
	if p.TotalResources() != 42 {
		t.Errorf("TotalResources() = %d, want 42", p.TotalResources())
	}
}

func TestTotalResources_WhenNew_ShouldBeZero(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	if p.TotalResources() != 0 {
		t.Errorf("TotalResources() = %d, want 0", p.TotalResources())
	}
}

func TestActivate_ShouldReturnNil(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() returned non-nil cmd, want nil")
	}
}

func TestTick_ShouldReturnTickMsg(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	cmd := p.tick()
	if cmd == nil {
		t.Fatal("tick() returned nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(TickMsg); !ok {
		t.Errorf("tick() cmd returned %T, want TickMsg", msg)
	}
}

func TestPlugin_WhenApplySucceeds_ShouldEmitPlanInvalidatedEvent(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusLoading

	_, cmd := p.Update(ApplyResultMsg{Err: nil, Duration: 5 * time.Second})
	if cmd == nil {
		t.Fatal("Update(ApplyResultMsg{Err: nil}) returned nil cmd, want cmd that emits PlanInvalidatedEvent")
	}

	msg := cmd()
	if _, ok := msg.(sdk.PlanInvalidatedEvent); !ok {
		t.Errorf("cmd() returned %T, want sdk.PlanInvalidatedEvent", msg)
	}
}

func TestPlugin_WhenDoneAndCtrlR_ShouldNavigateToPlan(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("Update(ctrl+r in StatusDone) returned nil cmd, want cmd that emits NavigateMsg{PluginID: \"plan\"}")
	}

	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want sdk.NavigateMsg", msg)
	}
	if nav.PluginID != "plan" {
		t.Errorf("NavigateMsg.PluginID = %q, want %q", nav.PluginID, "plan")
	}
}
