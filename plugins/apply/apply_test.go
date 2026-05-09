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

func (m *mockService) Plan(_ context.Context, _ []string) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (m *mockService) Apply(_ context.Context, _ []string) error           { return m.applyErr }
func (m *mockService) StateList(_ context.Context) ([]sdk.Resource, error) { return nil, nil }
func (m *mockService) Show(_ context.Context, _ string) (string, error)    { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)         { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"default"}, nil
}

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
	if p.KeyBinding() != "a" {
		t.Errorf("KeyBinding() = %q, want %q", p.KeyBinding(), "a")
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
		Dir:       "/tmp",
		Workspace: "default",
		Service:   svc,
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
	if p.status != StatusRunning {
		t.Errorf("status = %v, want StatusRunning", p.status)
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
	if p.status != StatusIdle {
		t.Errorf("status = %v, want StatusIdle", p.status)
	}
	if p.confirmed {
		t.Error("confirmed = true after cancel, want false")
	}
}

func TestUpdateApplyResultMsgSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusRunning

	result, cmd := p.Update(ApplyResultMsg{Err: nil, Duration: 5 * time.Second})
	if cmd != nil {
		t.Errorf("Update(ApplyResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusSuccess {
		t.Errorf("status = %v, want StatusSuccess", updated.status)
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
	p.status = StatusRunning

	result, cmd := p.Update(ApplyResultMsg{Err: errors.New("apply failed"), Duration: 3 * time.Second})
	if cmd != nil {
		t.Errorf("Update(ApplyResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusError {
		t.Errorf("status = %v, want StatusError", updated.status)
	}
	if updated.errMsg != "apply failed" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "apply failed")
	}
}

func TestUpdateTickMsgRunning(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusRunning
	p.startTime = time.Now().Add(-5 * time.Second)

	_, cmd := p.Update(TickMsg(time.Now()))
	if cmd == nil {
		t.Error("Update(TickMsg) in StatusRunning: cmd = nil, want non-nil (next tick)")
	}
	if p.elapsed < 4*time.Second {
		t.Errorf("elapsed = %v, want >= 4s", p.elapsed)
	}
}

func TestUpdateTickMsgNotRunning(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusSuccess

	_, cmd := p.Update(TickMsg(time.Now()))
	if cmd != nil {
		t.Error("Update(TickMsg) in StatusSuccess: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgIdle_Enter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusIdle

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
	if p.status != StatusRunning {
		t.Errorf("after y: status = %v, want StatusRunning", p.status)
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
	if p.status != StatusIdle {
		t.Errorf("after n in confirming: status = %v, want StatusIdle", p.status)
	}
}

func TestUpdateKeyMsgConfirming_NUppercase(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	if p.status != StatusIdle {
		t.Errorf("after N in confirming: status = %v, want StatusIdle", p.status)
	}
}

func TestUpdateKeyMsgConfirming_Esc(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusConfirming

	p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.status != StatusIdle {
		t.Errorf("after esc in confirming: status = %v, want StatusIdle", p.status)
	}
}

func TestUpdateKeyMsgError_Retry(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusError

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("after r in error: cmd = nil, want non-nil (retry)")
	}
	if p.status != StatusRunning {
		t.Errorf("after r in error: status = %v, want StatusRunning", p.status)
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
	p.status = StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusIdle) returned empty string")
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
	p.status = StatusRunning
	p.elapsed = 10 * time.Second

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusRunning) returned empty string")
	}
}

func TestViewSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusSuccess
	p.elapsed = 30 * time.Second

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusSuccess) returned empty string")
	}
}

func TestViewError(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusError
	p.errMsg = "something failed"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusError) returned empty string")
	}
}

func TestViewDefaultStatus(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = Status(99)

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
	if p.Status() != StatusIdle {
		t.Errorf("Status() = %v, want StatusIdle", p.Status())
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
	p.status = StatusIdle

	// A key other than enter in idle should do nothing
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("after x in idle: cmd != nil, want nil")
	}
	if p.status != StatusIdle {
		t.Errorf("after x in idle: status = %v, want StatusIdle", p.status)
	}
}

func TestUpdateKeyMsgError_OtherKey(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusError

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
	p.status = StatusRunning

	// Keys during running state should do nothing
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil {
		t.Error("after r in running: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusSuccess

	// Keys during success state should do nothing (no handler)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil {
		t.Error("after r in success: cmd != nil, want nil")
	}
}

func TestFormatDurationZero(t *testing.T) {
	got := formatDuration(0)
	if got != "0s" {
		t.Errorf("formatDuration(0) = %q, want %q", got, "0s")
	}
}
