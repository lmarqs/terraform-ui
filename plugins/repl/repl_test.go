package repl

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct{}

func (m *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error            { return nil }
func (m *mockService) StateList(_ context.Context) ([]sdk.Resource, error)          { return nil, nil }
func (m *mockService) Show(_ context.Context, _ string) (string, error)             { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)                  { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error)            { return nil, nil }
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error            { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string) error               { return nil }
func (m *mockService) WorkspaceDelete(_ context.Context, _ string) error            { return nil }
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
func (m *mockService) WithDir(_ string) sdk.Service                                 { return m }

func newTestPlugin() *Plugin {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{
		WorkingDir: "/tmp/test",
		Workspace:  "default",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.Init(ctx)
	return p
}

func TestNew(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	if p.ID() != "repl" {
		t.Errorf("ID() = %q, want %q", p.ID(), "repl")
	}
	if p.Name() != "Console" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Console")
	}
	if p.Description() != "Terraform console (REPL)" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Terraform console (REPL)")
	}
	if p.Ready() {
		t.Error("Ready() = true before activation, want false")
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
	p := newTestPlugin()

	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want sdk.StatusIdle", p.status)
	}
	if p.dir != "/tmp/test" {
		t.Errorf("dir = %q, want %q", p.dir, "/tmp/test")
	}
	if p.input != "" {
		t.Errorf("input = %q, want empty", p.input)
	}
	if p.historyIdx != -1 {
		t.Errorf("historyIdx = %d, want -1", p.historyIdx)
	}
	if len(p.history) != 0 {
		t.Errorf("len(history) = %d, want 0", len(p.history))
	}
}

func TestActivate(t *testing.T) {
	p := newTestPlugin()
	p.Activate()

	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.binaryPath == "" {
		t.Error("binaryPath should be set after Activate")
	}
	if !p.Ready() {
		t.Error("Ready() = false after Activate, want true")
	}
}

func TestActivateMultiContextNoSelection(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{
		WorkingDir: "/tmp/test",
		Workspace:  "default",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.Init(ctx)

	p.Activate()

	// Without ChdirGuard, Activate proceeds to sdk.StatusDone (no scope gating)
	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
}

func TestActivateWithScopeDir(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{
		WorkingDir: "/my/project",
		Workspace:  "default",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.Init(ctx)

	p.Activate()

	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.dir != "/my/project" {
		t.Errorf("dir = %q, want %q", p.dir, "/my/project")
	}
}

func TestInputTyping(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.binaryPath = "terraform"

	// Type characters
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if p.input != "l" {
		t.Errorf("input = %q, want %q", p.input, "l")
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if p.input != "lo" {
		t.Errorf("input = %q, want %q", p.input, "lo")
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if p.input != "loc" {
		t.Errorf("input = %q, want %q", p.input, "loc")
	}
}

func TestInputBackspace(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "hello"

	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.input != "hell" {
		t.Errorf("after backspace: input = %q, want %q", p.input, "hell")
	}

	// Backspace on empty does nothing
	p.input = ""
	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.input != "" {
		t.Errorf("backspace on empty: input = %q, want empty", p.input)
	}
}

func TestInputEnter(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.binaryPath = "terraform"
	p.input = "local.env"

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("enter with input: cmd = nil, want non-nil")
	}
	if p.status != StatusEvaluating {
		t.Errorf("status = %v, want StatusEvaluating", p.status)
	}
	if p.input != "" {
		t.Errorf("input = %q, want empty (cleared after enter)", p.input)
	}
	if len(p.pastInputs) != 1 {
		t.Errorf("len(pastInputs) = %d, want 1", len(p.pastInputs))
	}
	if p.pastInputs[0] != "local.env" {
		t.Errorf("pastInputs[0] = %q, want %q", p.pastInputs[0], "local.env")
	}
}

func TestInputEnterEmpty(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = ""

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with empty input: cmd != nil, want nil")
	}
}

func TestInputEnterWhitespace(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "   "

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with whitespace input: cmd != nil, want nil")
	}
}

func TestInputEnterDuringEvaluating(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating
	p.input = "something"

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter during evaluating: cmd != nil, want nil")
	}
}

func TestCtrlCClearsInput(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "some expr"
	p.historyIdx = 2

	p.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if p.input != "" {
		t.Errorf("after ctrl+c: input = %q, want empty", p.input)
	}
	if p.historyIdx != -1 {
		t.Errorf("after ctrl+c: historyIdx = %d, want -1", p.historyIdx)
	}
}

func TestEscDeactivates(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Error("esc: cmd = nil, want DeactivateMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("esc cmd returned %T, want DeactivateMsg", msg)
	}
}

func TestEscDuringEvaluatingDoesNothing(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Error("esc during evaluating: cmd != nil, want nil")
	}
}

func TestReplResultMsgSuccess(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating

	result, cmd := p.Update(ReplResultMsg{Expr: "local.env", Output: "\"production\"\n", Err: nil})
	if cmd != nil {
		t.Errorf("ReplResultMsg success: cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", updated.status)
	}
	if updated.HistoryLen() != 1 {
		t.Errorf("len(history) = %d, want 1", updated.HistoryLen())
	}

	expr, res, errStr := updated.HistoryEntry(0)
	if expr != "local.env" {
		t.Errorf("history[0].Expr = %q, want %q", expr, "local.env")
	}
	if res != "\"production\"" {
		t.Errorf("history[0].Result = %q, want %q", res, "\"production\"")
	}
	if errStr != "" {
		t.Errorf("history[0].Error = %q, want empty", errStr)
	}
}

func TestReplResultMsgError(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating

	result, cmd := p.Update(ReplResultMsg{Expr: "bad_expr", Output: "", Err: errors.New("eval failed")})
	if cmd != nil {
		t.Errorf("ReplResultMsg error: cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", updated.status)
	}
	if updated.HistoryLen() != 1 {
		t.Errorf("len(history) = %d, want 1", updated.HistoryLen())
	}

	expr, res, errStr := updated.HistoryEntry(0)
	if expr != "bad_expr" {
		t.Errorf("history[0].Expr = %q, want %q", expr, "bad_expr")
	}
	if res != "" {
		t.Errorf("history[0].Result = %q, want empty", res)
	}
	if errStr != "eval failed" {
		t.Errorf("history[0].Error = %q, want %q", errStr, "eval failed")
	}
}

func TestHistoryNavigation(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.binaryPath = "terraform"
	p.pastInputs = []string{"expr1", "expr2", "expr3"}
	p.input = "current"

	// Up arrow: recall last expression
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.input != "expr3" {
		t.Errorf("after first up: input = %q, want %q", p.input, "expr3")
	}
	if p.historyIdx != 2 {
		t.Errorf("after first up: historyIdx = %d, want 2", p.historyIdx)
	}

	// Up again
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.input != "expr2" {
		t.Errorf("after second up: input = %q, want %q", p.input, "expr2")
	}
	if p.historyIdx != 1 {
		t.Errorf("after second up: historyIdx = %d, want 1", p.historyIdx)
	}

	// Up again
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.input != "expr1" {
		t.Errorf("after third up: input = %q, want %q", p.input, "expr1")
	}
	if p.historyIdx != 0 {
		t.Errorf("after third up: historyIdx = %d, want 0", p.historyIdx)
	}

	// Up at boundary
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.input != "expr1" {
		t.Errorf("up at boundary: input = %q, want %q", p.input, "expr1")
	}

	// Down to go forward
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.input != "expr2" {
		t.Errorf("after down: input = %q, want %q", p.input, "expr2")
	}

	// Down more
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.input != "expr3" {
		t.Errorf("after second down: input = %q, want %q", p.input, "expr3")
	}

	// Down past end restores saved input
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.input != "current" {
		t.Errorf("down past end: input = %q, want %q", p.input, "current")
	}
	if p.historyIdx != -1 {
		t.Errorf("down past end: historyIdx = %d, want -1", p.historyIdx)
	}
}

func TestHistoryNavigationEmpty(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.pastInputs = nil
	p.input = "current"

	// Up with no history does nothing
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.input != "current" {
		t.Errorf("up with no history: input = %q, want %q", p.input, "current")
	}

	// Down with no history idx does nothing
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.input != "current" {
		t.Errorf("down with no history: input = %q, want %q", p.input, "current")
	}
}

func TestMultipleResults(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone

	// Simulate multiple evaluations
	p.Update(ReplResultMsg{Expr: "1+1", Output: "2\n", Err: nil})
	p.Update(ReplResultMsg{Expr: "2+2", Output: "4\n", Err: nil})
	p.Update(ReplResultMsg{Expr: "bad", Output: "", Err: errors.New("error")})

	if p.HistoryLen() != 3 {
		t.Errorf("len(history) = %d, want 3", p.HistoryLen())
	}

	expr, res, _ := p.HistoryEntry(0)
	if expr != "1+1" || res != "2" {
		t.Errorf("history[0] = (%q, %q), want (\"1+1\", \"2\")", expr, res)
	}

	expr, res, _ = p.HistoryEntry(1)
	if expr != "2+2" || res != "4" {
		t.Errorf("history[1] = (%q, %q), want (\"2+2\", \"4\")", expr, res)
	}

	expr, _, errStr := p.HistoryEntry(2)
	if expr != "bad" || errStr != "error" {
		t.Errorf("history[2] = (%q, err=%q), want (\"bad\", \"error\")", expr, errStr)
	}
}

func TestScrollingWithLongHistory(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone

	// Add many entries to trigger scrolling
	for i := 0; i < 50; i++ {
		p.Update(ReplResultMsg{
			Expr:   "expr_" + string(rune('a'+i%26)),
			Output: "result_" + string(rune('a'+i%26)) + "\n",
			Err:    nil,
		})
	}

	if p.HistoryLen() != 50 {
		t.Errorf("len(history) = %d, want 50", p.HistoryLen())
	}

	// View should not panic with small height
	view := p.View(80, 10)
	if view == "" {
		t.Error("View with long history returned empty string")
	}
}

func TestViewIdle(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusIdle) returned empty string")
	}
}

func TestViewError(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusError
	p.errMsg = "something went wrong"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusError) returned empty string")
	}
}

func TestViewReady(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "local.env"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone) returned empty string")
	}
}

func TestViewEvaluating(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusEvaluating) returned empty string")
	}
}

func TestViewWithHistory(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.history = []replEntry{
		{Expr: "local.env", Result: "\"production\""},
		{Expr: "bad", Error: "some error"},
	}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with history returned empty string")
	}
}

func TestViewDefaultStatus(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestUnknownMsg(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone

	type unknownMsg struct{}
	result, cmd := p.Update(unknownMsg{})
	if cmd != nil {
		t.Errorf("Update(unknownMsg) cmd = %v, want nil", cmd)
	}
	if result.(*Plugin) != p {
		t.Error("Update(unknownMsg) returned different plugin reference")
	}
}

func TestHistoryEntryOutOfBounds(t *testing.T) {
	p := newTestPlugin()

	expr, res, errStr := p.HistoryEntry(-1)
	if expr != "" || res != "" || errStr != "" {
		t.Error("HistoryEntry(-1) should return empty values")
	}

	expr, res, errStr = p.HistoryEntry(100)
	if expr != "" || res != "" || errStr != "" {
		t.Error("HistoryEntry(100) should return empty values")
	}
}

func TestFormatHistoryEntry(t *testing.T) {
	// Success entry
	entry := replEntry{Expr: "local.x", Result: "\"hello\""}
	formatted := FormatHistoryEntry(entry)
	if formatted != "> local.x\n\"hello\"" {
		t.Errorf("FormatHistoryEntry success = %q, want %q", formatted, "> local.x\n\"hello\"")
	}

	// Error entry
	entry = replEntry{Expr: "bad", Error: "not found"}
	formatted = FormatHistoryEntry(entry)
	if formatted != "> bad\nError: not found" {
		t.Errorf("FormatHistoryEntry error = %q, want %q", formatted, "> bad\nError: not found")
	}

	// Empty result
	entry = replEntry{Expr: "null_expr"}
	formatted = FormatHistoryEntry(entry)
	if formatted != "> null_expr" {
		t.Errorf("FormatHistoryEntry empty = %q, want %q", formatted, "> null_expr")
	}
}

func TestExportedGetters(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "test"
	p.historyIdx = 2
	p.dir = "/a/b"
	p.binaryPath = "/usr/bin/terraform"
	p.errMsg = "oops"
	p.pastInputs = []string{"x", "y"}

	if p.Status() != sdk.StatusDone {
		t.Errorf("Status() = %v, want sdk.StatusDone", p.Status())
	}
	if p.Input() != "test" {
		t.Errorf("Input() = %q, want %q", p.Input(), "test")
	}
	if p.HistoryIdx() != 2 {
		t.Errorf("HistoryIdx() = %d, want 2", p.HistoryIdx())
	}
	if p.Dir() != "/a/b" {
		t.Errorf("Dir() = %q, want %q", p.Dir(), "/a/b")
	}
	if p.BinaryPath() != "/usr/bin/terraform" {
		t.Errorf("BinaryPath() = %q, want %q", p.BinaryPath(), "/usr/bin/terraform")
	}
	if p.ErrMsg() != "oops" {
		t.Errorf("ErrMsg() = %q, want %q", p.ErrMsg(), "oops")
	}
	if len(p.PastInputs()) != 2 {
		t.Errorf("len(PastInputs()) = %d, want 2", len(p.PastInputs()))
	}
}

func TestSetBinaryPath(t *testing.T) {
	p := newTestPlugin()
	p.SetBinaryPath("/custom/terraform")
	if p.binaryPath != "/custom/terraform" {
		t.Errorf("binaryPath = %q, want %q", p.binaryPath, "/custom/terraform")
	}
}

func TestHandleChdirChanged(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{
		WorkingDir: "/old/ctx",
		Workspace:  "default",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.Init(ctx)

	// First activation sets the scope
	p.Activate()

	// Simulate state accumulation
	p.history = []replEntry{{Expr: "old", Result: "stale"}}
	p.pastInputs = []string{"old"}

	// HandleChdirChanged resets state
	p.HandleChdirChanged(sdk.ChdirChangedEvent{AbsPath: "/new/ctx"})

	if p.dir != "/new/ctx" {
		t.Errorf("dir = %q, want %q", p.dir, "/new/ctx")
	}
	if len(p.history) != 0 {
		t.Errorf("history should be reset on context change, got %d entries", len(p.history))
	}
	if len(p.pastInputs) != 0 {
		t.Errorf("pastInputs should be reset on context change, got %d entries", len(p.pastInputs))
	}
}
