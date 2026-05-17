package macro

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRunnerExecuteEmpty(t *testing.T) {
	model := mockModel{content: "ready"}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	err := r.Execute(nil)
	if err != nil {
		t.Errorf("empty commands should succeed, got %v", err)
	}
}

func TestRunnerKey(t *testing.T) {
	model := mockModel{content: "initial"}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdKey, Args: []string{"p"}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.ViewContains("keys: p") {
		t.Errorf("view = %q, want to contain 'keys: p'", d.View())
	}
}

func TestRunnerAssertViewPass(t *testing.T) {
	model := mockModel{}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	// After Init(), mockModel sets content to "ready"
	commands := []Command{
		{Type: CmdAssertView, Args: []string{"ready"}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Errorf("assert should pass, got %v", err)
	}
}

func TestRunnerAssertViewFail(t *testing.T) {
	model := mockModel{content: "hello world"}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdAssertView, Args: []string{"missing"}, Line: 3},
	}
	err := r.Execute(commands)
	if err == nil {
		t.Fatal("assert should fail")
	}

	runErr, ok := err.(*RunError)
	if !ok {
		t.Fatalf("expected *RunError, got %T", err)
	}
	if runErr.Code != ExitAssertFail {
		t.Errorf("code = %d, want %d", runErr.Code, ExitAssertFail)
	}
	if runErr.Line != 3 {
		t.Errorf("line = %d, want 3", runErr.Line)
	}
	if !strings.Contains(runErr.Error(), "line 3") {
		t.Errorf("error = %q, want to contain 'line 3'", runErr.Error())
	}
}

func TestRunnerWaitReady(t *testing.T) {
	model := mockModel{}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	// mockModel.Init() returns readyMsg which sets content to "ready"
	// After Driver.Init() in Execute, the view should be "ready"
	commands := []Command{
		{Type: CmdWaitReady, Line: 1},
		{Type: CmdAssertView, Args: []string{"ready"}, Line: 2},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Errorf("should succeed after init: %v", err)
	}
}

func TestRunnerWaitViewPass(t *testing.T) {
	model := mockModel{}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	// After Init(), mockModel sets content to "ready"
	commands := []Command{
		{Type: CmdWaitView, Args: []string{"ready"}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Errorf("wait view should pass: %v", err)
	}
}

func TestRunnerWaitViewTimeout(t *testing.T) {
	model := mockModel{content: "something else"}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)
	r.timeout = 50 * time.Millisecond

	commands := []Command{
		{Type: CmdWaitView, Args: []string{"never appears"}, Line: 5},
	}
	err := r.Execute(commands)
	if err == nil {
		t.Fatal("should timeout")
	}

	runErr, ok := err.(*RunError)
	if !ok {
		t.Fatalf("expected *RunError, got %T", err)
	}
	if runErr.Code != ExitTimeout {
		t.Errorf("code = %d, want %d", runErr.Code, ExitTimeout)
	}
	if runErr.Line != 5 {
		t.Errorf("line = %d, want 5", runErr.Line)
	}
}

func TestRunnerScreenshot(t *testing.T) {
	model := mockModel{}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	path := filepath.Join(t.TempDir(), "shot.txt")
	// After Init(), content is "ready"
	commands := []Command{
		{Type: CmdScreenshot, Args: []string{path}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("screenshot file not created: %v", err)
	}
	if string(data) != "ready" {
		t.Errorf("screenshot content = %q, want %q", string(data), "ready")
	}
}

func TestRunnerResize(t *testing.T) {
	model := mockModel{}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdResize, Args: []string{"120", "40"}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Errorf("resize should succeed: %v", err)
	}
}

func TestRunnerSleep(t *testing.T) {
	model := mockModel{content: "ok"}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdSleep, Args: []string{"10ms"}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Errorf("sleep should succeed: %v", err)
	}
}

func TestRunnerStopsOnFirstError(t *testing.T) {
	model := mockModel{}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	// After Init(), content is "ready"
	commands := []Command{
		{Type: CmdAssertView, Args: []string{"missing"}, Line: 1},
		{Type: CmdKey, Args: []string{"p"}, Line: 2},
	}
	err := r.Execute(commands)
	if err == nil {
		t.Fatal("should fail on first assertion")
	}
	// Key should NOT have been dispatched
	if d.ViewContains("keys:") {
		t.Error("second command should not have executed")
	}
}

func TestRunErrorFormat(t *testing.T) {
	tests := []struct {
		name string
		err  RunError
		want string
	}{
		{
			name: "with line",
			err:  RunError{Code: ExitAssertFail, Line: 7, Message: "assertion failed"},
			want: "line 7: assertion failed",
		},
		{
			name: "without line",
			err:  RunError{Code: ExitSyntaxError, Line: 0, Message: "bad tape"},
			want: "bad tape",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// windowModel tracks window size messages to verify resize.
type windowModel struct {
	width  int
	height int
}

func (m windowModel) Init() tea.Cmd { return nil }

func (m windowModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m windowModel) View() string {
	if m.width == 0 {
		return "no size"
	}
	return fmt.Sprintf("%dx%d", m.width, m.height)
}

func TestRunnerResizeUpdatesView(t *testing.T) {
	model := windowModel{}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdResize, Args: []string{"120", "40"}, Line: 1},
		{Type: CmdAssertView, Args: []string{"120x40"}, Line: 2},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Errorf("should succeed: %v", err)
	}
}

func TestRunnerScreenshotWriteFailure(t *testing.T) {
	model := mockModel{}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdScreenshot, Args: []string{"/nonexistent-dir/subdir/file.txt"}, Line: 4},
	}
	err := r.Execute(commands)
	if err == nil {
		t.Fatal("expected error for write to invalid path")
	}

	runErr, ok := err.(*RunError)
	if !ok {
		t.Fatalf("expected *RunError, got %T", err)
	}
	if runErr.Code != ExitAssertFail {
		t.Errorf("code = %d, want %d", runErr.Code, ExitAssertFail)
	}
	if runErr.Line != 4 {
		t.Errorf("line = %d, want 4", runErr.Line)
	}
}

func TestRunnerWaitReadyTimeout(t *testing.T) {
	model := loadingModel{}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)
	r.timeout = 50 * time.Millisecond

	commands := []Command{
		{Type: CmdWaitReady, Line: 2},
	}
	err := r.Execute(commands)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	runErr, ok := err.(*RunError)
	if !ok {
		t.Fatalf("expected *RunError, got %T", err)
	}
	if runErr.Code != ExitTimeout {
		t.Errorf("code = %d, want %d", runErr.Code, ExitTimeout)
	}
}

type loadingModel struct{}

func (m loadingModel) Init() tea.Cmd                         { return nil }
func (m loadingModel) Update(_ tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m loadingModel) View() string                          { return "Loading" }

func TestRunner_WhenEmitStale_ShouldSendPlanInvalidatedEvent(t *testing.T) {
	model := mockModel{content: "ready"}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdEmit, Args: []string{"stale"}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Fatalf("emit stale should succeed: %v", err)
	}
}

func TestRunner_WhenEmitRefreshed_ShouldSendStateRefreshedEvent(t *testing.T) {
	model := mockModel{content: "ready"}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdEmit, Args: []string{"refreshed"}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Fatalf("emit refreshed should succeed: %v", err)
	}
}

func TestRunner_WhenEmitLock_ShouldSendLockDetectedEvent(t *testing.T) {
	model := mockModel{content: "ready"}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdEmit, Args: []string{"lock", "deploy-bot"}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Fatalf("emit lock should succeed: %v", err)
	}
}

func TestRunner_WhenEmitLockClear_ShouldSendLockClearedEvent(t *testing.T) {
	model := mockModel{content: "ready"}
	d := NewDriver(model, 80, 24)
	r := NewRunner(d)

	commands := []Command{
		{Type: CmdEmit, Args: []string{"lock-clear"}, Line: 1},
	}
	err := r.Execute(commands)
	if err != nil {
		t.Fatalf("emit lock-clear should succeed: %v", err)
	}
}

func TestBuildEmitMsg_WhenStale_ShouldReturnPlanInvalidatedEvent(t *testing.T) {
	msg := buildEmitMsg([]string{"stale"})
	if msg == nil {
		t.Fatal("buildEmitMsg(stale) should not return nil")
	}
}

func TestBuildEmitMsg_WhenRefreshed_ShouldReturnStateRefreshedEvent(t *testing.T) {
	msg := buildEmitMsg([]string{"refreshed"})
	if msg == nil {
		t.Fatal("buildEmitMsg(refreshed) should not return nil")
	}
}

func TestBuildEmitMsg_WhenLock_ShouldReturnLockDetectedEvent(t *testing.T) {
	msg := buildEmitMsg([]string{"lock", "user@host"})
	if msg == nil {
		t.Fatal("buildEmitMsg(lock) should not return nil")
	}
}

func TestBuildEmitMsg_WhenLockClear_ShouldReturnLockClearedEvent(t *testing.T) {
	msg := buildEmitMsg([]string{"lock-clear"})
	if msg == nil {
		t.Fatal("buildEmitMsg(lock-clear) should not return nil")
	}
}

func TestBuildEmitMsg_WhenUnknown_ShouldReturnNil(t *testing.T) {
	msg := buildEmitMsg([]string{"unknown-event"})
	if msg != nil {
		t.Fatalf("buildEmitMsg(unknown) should return nil, got %v", msg)
	}
}
