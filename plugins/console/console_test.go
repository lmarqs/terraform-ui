package console

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newTestPlugin() *Plugin {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	h := sdktest.NewDeps(svc)
	h.Ctx.WorkingDir = "/tmp/test"
	p.Init(h.Deps)
	return p
}

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "console" {
		t.Errorf("ID() = %q, want %q", p.ID(), "console")
	}
	if p.Name() != "Console" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Console")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(map[string]interface{}{}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	h := sdktest.NewDeps(svc)
	h.Ctx.WorkingDir = "/tmp/test"
	if cmd := p.Init(h.Deps); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
	if p.Ready() {
		t.Error("Ready() should be false before activation")
	}
}

func TestPlugin_WhenActivated_ShouldSetDoneStatusAndBinaryPath(t *testing.T) {
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

func TestPlugin_WhenActivatedWithoutChdirGuard_ShouldProceedToDone(t *testing.T) {
	p := newTestPlugin()
	p.Activate()

	// Without ChdirGuard, Activate proceeds to sdk.StatusDone (no scope gating)
	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
}

func TestPlugin_WhenInitWithProjectDir_ShouldUseProjectDir(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	h := sdktest.NewDeps(svc)
	h.Ctx.WorkingDir = "/my/project"
	p.Init(h.Deps)

	p.Activate()

	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.dir != "/my/project" {
		t.Errorf("dir = %q, want %q", p.dir, "/my/project")
	}
}

func TestUpdate_WhenTypingCharacters_ShouldAppendToInput(t *testing.T) {
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

func TestUpdate_WhenBackspacePressed_ShouldRemoveLastCharacter(t *testing.T) {
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

func TestUpdate_WhenEnterWithInput_ShouldStartEvaluation(t *testing.T) {
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

func TestUpdate_WhenEnterWithEmptyInput_ShouldReturnNil(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = ""

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with empty input: cmd != nil, want nil")
	}
}

func TestUpdate_WhenEnterWithWhitespace_ShouldReturnNil(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "   "

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with whitespace input: cmd != nil, want nil")
	}
}

func TestUpdate_WhenEnterDuringEvaluating_ShouldReturnNil(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating
	p.input = "something"

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter during evaluating: cmd != nil, want nil")
	}
}

func TestUpdate_WhenCtrlUPressed_ShouldClearInputAndResetHistory(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "some expr"
	p.historyIdx = 2

	p.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if p.input != "" {
		t.Errorf("after ctrl+u: input = %q, want empty", p.input)
	}
	if p.historyIdx != -1 {
		t.Errorf("after ctrl+u: historyIdx = %d, want -1", p.historyIdx)
	}
}

func TestUpdate_WhenEscPressed_ShouldEmitDeactivateMsg(t *testing.T) {
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

func TestUpdate_WhenEscDuringEvaluating_ShouldReturnNil(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Error("esc during evaluating: cmd != nil, want nil")
	}
}

func TestUpdate_WhenReplResultSuccess_ShouldAddToHistory(t *testing.T) {
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

func TestUpdate_WhenReplResultError_ShouldAddErrorToHistory(t *testing.T) {
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

func TestUpdate_WhenUpDownKeys_ShouldNavigateHistory(t *testing.T) {
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

func TestUpdate_WhenUpDownWithEmptyHistory_ShouldKeepCurrentInput(t *testing.T) {
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

func TestUpdate_WhenMultipleResults_ShouldAccumulateHistory(t *testing.T) {
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

func TestView_WhenLongHistory_ShouldNotPanicWithSmallHeight(t *testing.T) {
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

func TestView_WhenIdle_ShouldReturnNonEmpty(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusIdle) returned empty string")
	}
}

func TestView_WhenError_ShouldReturnNonEmpty(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusError
	p.errMsg = "something went wrong"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusError) returned empty string")
	}
}

func TestView_WhenDone_ShouldReturnNonEmpty(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "local.env"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone) returned empty string")
	}
}

func TestView_WhenEvaluating_ShouldReturnNonEmpty(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusEvaluating) returned empty string")
	}
}

func TestView_WhenHistoryPresent_ShouldReturnNonEmpty(t *testing.T) {
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

func TestView_WhenUnknownStatus_ShouldReturnEmpty(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestUpdate_WhenUnknownMsg_ShouldReturnSelfAndNil(t *testing.T) {
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

func TestHistoryEntry_WhenOutOfBounds_ShouldReturnEmptyValues(t *testing.T) {
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

func TestFormatHistoryEntry_WhenCalled_ShouldFormatCorrectly(t *testing.T) {
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

func TestGetters_WhenFieldsSet_ShouldReturnCorrectValues(t *testing.T) {
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

func TestSetBinaryPath_WhenCalled_ShouldUpdatePath(t *testing.T) {
	p := newTestPlugin()
	p.SetBinaryPath("/custom/terraform")
	if p.binaryPath != "/custom/terraform" {
		t.Errorf("binaryPath = %q, want %q", p.binaryPath, "/custom/terraform")
	}
}

func TestHandleContextChanged_WhenCalled_ShouldResetStateAndUpdateDir(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	h := sdktest.NewDeps(svc)
	h.Ctx.WorkingDir = "/old/ctx"
	p.Init(h.Deps)

	// Simulate state accumulation
	p.history = []replEntry{{Expr: "old", Result: "stale"}}
	p.pastInputs = []string{"old"}

	// HandleContextChanged updates the working dir and resets history.
	p.HandleContextChanged(sdk.ContextChangedEvent{Next: &sdk.Context{
		WorkingDir: "/new/ctx",
		Service:    svc,
	}})

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

func TestHints_WhenStatusDone_ShouldReturnREPLHints(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone

	hints := p.Hints()
	if len(hints) != 4 {
		t.Fatalf("Hints() returned %d hints, want 4", len(hints))
	}
	if hints[0].Key != "Enter" || hints[0].Description != "evaluate" {
		t.Errorf("hints[0] = {%q, %q}, want {\"Enter\", \"evaluate\"}", hints[0].Key, hints[0].Description)
	}
	if hints[1].Key != "↑↓" || hints[1].Description != "history" {
		t.Errorf("hints[1] = {%q, %q}, want {\"↑↓\", \"history\"}", hints[1].Key, hints[1].Description)
	}
	if hints[2].Key != "^U" || hints[2].Description != "clear" {
		t.Errorf("hints[2] = {%q, %q}, want {\"^U\", \"clear\"}", hints[2].Key, hints[2].Description)
	}
	if hints[3].Key != "esc" || hints[3].Description != "back" {
		t.Errorf("hints[3] = {%q, %q}, want {\"esc\", \"back\"}", hints[3].Key, hints[3].Description)
	}
}

func TestHints_WhenStatusEvaluating_ShouldReturnREPLHints(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating

	hints := p.Hints()
	if len(hints) != 4 {
		t.Fatalf("Hints() returned %d hints, want 4", len(hints))
	}
	if hints[0].Key != "Enter" {
		t.Errorf("hints[0].Key = %q, want \"Enter\"", hints[0].Key)
	}
}

func TestHints_WhenStatusError_ShouldReturnBackHints(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusError

	hints := p.Hints()
	expected := (sdk.HintSetQuit).Hints()
	if len(hints) != len(expected) {
		t.Fatalf("Hints() returned %d hints, want %d", len(hints), len(expected))
	}
	if hints[0].Key != "q" || hints[0].Description != "quit" {
		t.Errorf("hints[0] = {%q, %q}, want {\"q\", \"back\"}", hints[0].Key, hints[0].Description)
	}
}

func TestHints_WhenStatusIdle_ShouldReturnBackHints(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusIdle

	hints := p.Hints()
	expected := (sdk.HintSetQuit).Hints()
	if len(hints) != len(expected) {
		t.Fatalf("Hints() returned %d hints, want %d", len(hints), len(expected))
	}
}

func TestEvaluate_WhenBinaryExists_ShouldReturnCmd(t *testing.T) {
	dir := t.TempDir()
	scriptPath := dir + "/faketerraform"
	script := "#!/bin/sh\ncat\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	p := newTestPlugin()
	p.binaryPath = scriptPath
	p.dir = dir

	cmd := p.Evaluate("hello world")
	if cmd == nil {
		t.Fatal("Evaluate() returned nil cmd")
	}

	msg := cmd()
	result, ok := msg.(ReplResultMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want ReplResultMsg", msg)
	}
	if result.Expr != "hello world" {
		t.Errorf("result.Expr = %q, want %q", result.Expr, "hello world")
	}
	if result.Err != nil {
		t.Errorf("result.Err = %v, want nil", result.Err)
	}
	if strings.TrimSpace(result.Output) != "hello world" {
		t.Errorf("result.Output = %q, want %q", strings.TrimSpace(result.Output), "hello world")
	}
}

func TestEvaluate_WhenBinaryNotFound_ShouldReturnError(t *testing.T) {
	p := newTestPlugin()
	p.binaryPath = "/nonexistent/binary/path"
	p.dir = t.TempDir()

	cmd := p.Evaluate("test")
	if cmd == nil {
		t.Fatal("Evaluate() returned nil cmd")
	}

	msg := cmd()
	result, ok := msg.(ReplResultMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want ReplResultMsg", msg)
	}
	if result.Expr != "test" {
		t.Errorf("result.Expr = %q, want %q", result.Expr, "test")
	}
	if result.Err == nil {
		t.Error("result.Err = nil, want error for nonexistent binary")
	}
}

func TestDetectBinary_WhenTofuOnPath_ShouldReturnTofu(t *testing.T) {
	dir := t.TempDir()
	tofuPath := dir + "/tofu"
	if err := os.WriteFile(tofuPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	result := detectBinary()
	if result != "tofu" {
		t.Errorf("detectBinary() = %q, want %q", result, "tofu")
	}
}

func TestDetectBinary_WhenTofuNotOnPath_ShouldReturnTerraform(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	result := detectBinary()
	if result != "terraform" {
		t.Errorf("detectBinary() = %q, want %q", result, "terraform")
	}
}

func TestHistory_ShouldReturnHistorySlice(t *testing.T) {
	p := newTestPlugin()
	p.history = []replEntry{
		{Expr: "a", Result: "1"},
		{Expr: "b", Error: "fail"},
	}

	h := p.History()
	if len(h) != 2 {
		t.Fatalf("History() returned %d entries, want 2", len(h))
	}
	if h[0].Expr != "a" || h[0].Result != "1" {
		t.Errorf("History()[0] = {%q, %q}, want {\"a\", \"1\"}", h[0].Expr, h[0].Result)
	}
	if h[1].Expr != "b" || h[1].Error != "fail" {
		t.Errorf("History()[1] = {%q, err=%q}, want {\"b\", \"fail\"}", h[1].Expr, h[1].Error)
	}
}

func TestRenderREPL_WhenHeightVerySmall_ShouldUseMinHistoryLines(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.history = []replEntry{
		{Expr: "a", Result: "1"},
		{Expr: "b", Result: "2"},
		{Expr: "c", Result: "3"},
	}

	view := p.View(80, 2)
	if view == "" {
		t.Error("View with height=2 returned empty string")
	}
}

func TestRenderREPL_WhenHeightExactlyAtThreshold_ShouldNotPanic(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.history = []replEntry{
		{Expr: "x", Result: "y"},
	}

	view := p.View(80, 5)
	if view == "" {
		t.Error("View with height=5 returned empty string")
	}
}

func TestRenderREPL_WhenEntryHasEmptyResult_ShouldRenderWithoutResultLine(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.history = []replEntry{
		{Expr: "null_expr", Result: "", Error: ""},
	}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with empty result entry returned empty string")
	}
}

func TestRenderREPL_WhenHistoryHasMultilineResult_ShouldSplitLines(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.history = []replEntry{
		{Expr: "multiline", Result: "line1\nline2\nline3"},
	}

	view := p.View(80, 24)
	if !strings.Contains(view, "line1") {
		t.Error("View should contain first line of multiline result")
	}
	if !strings.Contains(view, "line3") {
		t.Error("View should contain last line of multiline result")
	}
}

func TestPlugin_WhenCapturesKeysInDone_ShouldReturnTrue(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	if !p.CapturesKeys() {
		t.Error("CapturesKeys() in Done should be true")
	}
}

func TestPlugin_WhenCapturesKeysInIdle_ShouldReturnFalse(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusIdle
	if p.CapturesKeys() {
		t.Error("CapturesKeys() in Idle should be false")
	}
}

func TestPlugin_WhenQKeyWithNonEmptyInput_ShouldAppendQ(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "hello"

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Error("q with non-empty input: cmd != nil, want nil")
	}
	if p.input != "helloq" {
		t.Errorf("input = %q, want %q", p.input, "helloq")
	}
	if p.historyIdx != -1 {
		t.Errorf("historyIdx = %d, want -1", p.historyIdx)
	}
}

func TestPlugin_WhenQKeyWithEmptyInputDuringEvaluating_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin()
	p.status = StatusEvaluating
	p.input = ""

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Error("q during evaluating: cmd != nil, want nil")
	}
	if p.input != "q" {
		t.Errorf("input = %q, want %q (appended since evaluating captures keys)", p.input, "q")
	}
}

func TestPlugin_WhenNonPrintableKey_ShouldNotAppend(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "hello"

	p.Update(tea.KeyMsg{Type: tea.KeyTab})
	if p.input != "hello" {
		t.Errorf("input = %q after tab, want %q (unchanged)", p.input, "hello")
	}
}

func TestPlugin_WhenCtrlHKey_ShouldBackspace(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = "hello"

	p.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	if p.input != "hell" {
		t.Errorf("input after ctrl+h = %q, want %q", p.input, "hell")
	}
}

func TestPlugin_WhenQKeyWithEmptyInputNotEvaluating_ShouldDeactivate(t *testing.T) {
	p := newTestPlugin()
	p.status = sdk.StatusDone
	p.input = ""

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q with empty input in Done: cmd = nil, want DeactivateMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("q cmd returned %T, want DeactivateMsg", msg)
	}
}

func TestHandleContextChanged_WhenNextNil_ShouldBeNoOp(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.dir = "/keep"
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil})
	if cmd != nil {
		t.Error("HandleContextChanged with nil Next returned non-nil cmd")
	}
	if p.dir != "/keep" {
		t.Errorf("dir mutated on nil Next, got %q", p.dir)
	}
}
