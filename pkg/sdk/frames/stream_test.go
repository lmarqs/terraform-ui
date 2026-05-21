package frames

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// makeStream creates a pre-seeded StreamFrame useful for rendering tests.
func makeStream(lines []string) *StreamFrame {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("terraform test", ch, nil)
	sf.lines = lines
	return sf
}

func TestStreamFrame_ID(t *testing.T) {
	sf := makeStream(nil)
	if sf.ID() != "stream" {
		t.Errorf("ID() = %q, want %q", sf.ID(), "stream")
	}
}

func TestStreamFrame_InterfaceCompliance(t *testing.T) {
	var _ sdk.Frame = (*StreamFrame)(nil)
}

func TestLineWriter_SendsLines(t *testing.T) {
	lw, ch := NewLineWriter()
	lw.Write([]byte("hello\nworld\n")) //nolint
	lw.Close()

	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}
	if len(lines) != 2 || lines[0] != "hello" || lines[1] != "world" {
		t.Errorf("got lines %v, want [hello world]", lines)
	}
}

func TestLineWriter_PartialLineFlushOnClose(t *testing.T) {
	lw, ch := NewLineWriter()
	lw.Write([]byte("no newline")) //nolint
	lw.Close()

	var got []string
	for line := range ch {
		got = append(got, line)
	}
	if len(got) != 1 || got[0] != "no newline" {
		t.Errorf("got %v, want [no newline]", got)
	}
}

func TestWaitForLine_ReceivesLine(t *testing.T) {
	lw, ch := NewLineWriter()
	lw.Write([]byte("line1\n")) //nolint

	cmd := WaitForLine(ch)
	msg := cmd()

	line, ok := msg.(StreamLineMsg)
	if !ok {
		t.Fatalf("got %T, want StreamLineMsg", msg)
	}
	if line.Line != "line1" {
		t.Errorf("Line = %q, want %q", line.Line, "line1")
	}
	lw.Close()
}

func TestWaitForLine_EmitsStreamDoneMsgOnClose(t *testing.T) {
	lw, ch := NewLineWriter()
	lw.Close()

	cmd := WaitForLine(ch)
	msg := cmd()

	if _, ok := msg.(StreamDoneMsg); !ok {
		t.Fatalf("got %T, want StreamDoneMsg", msg)
	}
}

func TestStreamFrame_AccumulatesLines(t *testing.T) {
	lw, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)

	sf.Update(StreamLineMsg{Line: "line1"})
	sf.Update(StreamLineMsg{Line: "line2"})

	if len(sf.lines) != 2 || sf.lines[0] != "line1" || sf.lines[1] != "line2" {
		t.Errorf("lines = %v, want [line1 line2]", sf.lines)
	}
	lw.Close()
}

func TestStreamFrame_StreamLineMsgSchedulesNextRead(t *testing.T) {
	lw, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)

	_, cmd := sf.Update(StreamLineMsg{Line: "x"})
	if cmd == nil {
		t.Fatal("StreamLineMsg should schedule next WaitForLine cmd")
	}
	lw.Close()
}

func TestStreamFrame_StreamDoneMsgMarksDone(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)
	sf.Update(StreamDoneMsg{})

	if !sf.done {
		t.Fatal("StreamDoneMsg should mark frame as done")
	}
}

func TestStreamFrame_EscPopsWhenDone(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)
	sf.done = true

	next, _ := sf.Update(keyMsg("esc"))
	if next != nil {
		t.Fatal("esc when done should pop frame (return nil)")
	}
}

func TestStreamFrame_EscIgnoredWhenRunning(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)

	next, _ := sf.Update(keyMsg("esc"))
	if next == nil {
		t.Fatal("esc while running should NOT pop frame")
	}
}

func TestStreamFrame_CtrlCSendsGracefulCancel(t *testing.T) {
	called := false
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, func() { called = true })

	sf.Update(keyMsg("ctrl+c"))
	if !called {
		t.Fatal("first ^c should call cancelFn")
	}
	if !sf.sigintSent {
		t.Fatal("sigintSent should be true after first ^c")
	}
}

func TestStreamFrame_CtrlCTwiceOpensConfirm(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, func() {})
	sf.sigintSent = true

	sf.Update(keyMsg("ctrl+c"))
	if sf.confirm == nil {
		t.Fatal("second ^c should open confirm overlay")
	}
}

func TestStreamFrame_ForceConfirmYesCallsCancelFnAgain(t *testing.T) {
	count := 0
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, func() { count++ })
	sf.sigintSent = true

	sf.Update(keyMsg("ctrl+c")) // opens confirm
	sf.Update(keyMsg("y"))      // confirms force cancel

	if count != 1 {
		t.Errorf("cancelFn call count = %d, want 1", count)
	}
	if sf.confirm != nil {
		t.Fatal("confirm should be dismissed after y")
	}
}

func TestStreamFrame_ForceConfirmNoDismissesOverlay(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, func() {})
	sf.sigintSent = true

	sf.Update(keyMsg("ctrl+c")) // opens confirm
	sf.Update(keyMsg("n"))      // cancels force

	if sf.confirm != nil {
		t.Fatal("confirm should be dismissed after n")
	}
}

func TestStreamFrame_HintsWhileRunning(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)

	hints := sf.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints while running should not be empty")
	}
	if hints[0].Key != "^c" || hints[0].Description != "cancel" {
		t.Errorf("hints[0] = %v, want {^c cancel}", hints[0])
	}
}

func TestStreamFrame_HintsAfterFirstCtrlC(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)
	sf.sigintSent = true

	hints := sf.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints should not be empty after first ^c")
	}
	if hints[0].Key != "^c" || hints[0].Description != "force cancel" {
		t.Errorf("hints[0] = %v, want {^c force cancel}", hints[0])
	}
}

func TestStreamFrame_HintsWhenDone(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)
	sf.done = true

	hints := sf.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints when done should not be empty")
	}
	if hints[0] != sdk.HintBack {
		t.Errorf("hints[0] = %v, want HintBack", hints[0])
	}
}

func TestStreamFrame_HintsDelegatesToConfirm(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, func() {})
	sf.sigintSent = true
	sf.Update(keyMsg("ctrl+c")) // opens confirm overlay

	hints := sf.Hints()
	if len(hints) != 2 || hints[0].Key != "y" || hints[1].Key != "n" {
		t.Errorf("hints = %v, want confirm hints", hints)
	}
}

func TestStreamFrame_ViewRendersLines(t *testing.T) {
	sf := makeStream([]string{"alpha", "beta"})
	sf.done = true
	sf.autoScroll = false

	view := sf.View(80, 24)
	if !strings.Contains(view, "alpha") || !strings.Contains(view, "beta") {
		t.Errorf("view %q should contain 'alpha' and 'beta'", view)
	}
}

func TestStreamFrame_ViewEmptyWhenNoLines(t *testing.T) {
	sf := makeStream(nil)
	view := sf.View(80, 24)
	if view != "" {
		t.Errorf("view with no lines should be empty, got %q", view)
	}
}

func TestStreamFrame_AutoScrollAdvancesToBottom(t *testing.T) {
	sf := makeStream([]string{"a", "b", "c", "d", "e"})
	sf.autoScroll = true

	sf.View(80, 3) // 3-line viewport

	if sf.scrollY != 2 { // len(lines)-1 = 4, but clamped to len-height=5-3=2
		t.Errorf("scrollY = %d, want 2 (bottom of 5 lines with height 3)", sf.scrollY)
	}
}

func TestStreamFrame_ManualScrollDisablesAutoScroll(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)
	sf.lines = []string{"a", "b", "c"}
	sf.scrollY = 2

	sf.Update(keyMsg("up"))
	if sf.autoScroll {
		t.Fatal("scrolling up should disable autoScroll")
	}
}

func TestStreamFrame_GKeyReenablesAutoScroll(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)
	sf.autoScroll = false

	sf.Update(keyMsg("G"))
	if !sf.autoScroll {
		t.Fatal("G key should re-enable autoScroll")
	}
}

func TestStreamFrame_Lines(t *testing.T) {
	sf := makeStream([]string{"x", "y"})
	got := sf.Lines()
	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Errorf("Lines() = %v, want [x y]", got)
	}
}

func TestStreamFrame_LinesMutationDoesNotAffectInternal(t *testing.T) {
	sf := makeStream([]string{"original"})
	got := sf.Lines()
	got[0] = "mutated"
	if sf.lines[0] != "original" {
		t.Fatal("Lines() should return a copy, not a reference")
	}
}

func TestStreamFrame_ViewDelegatesToConfirmWhenActive(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, func() {})
	sf.sigintSent = true
	sf.Update(keyMsg("ctrl+c")) // opens confirm overlay

	view := sf.View(80, 24)
	if !strings.Contains(view, "Force cancel") {
		t.Errorf("view should show confirm prompt, got %q", view)
	}
}

func TestStreamFrame_ConfirmUpdateIsRouted(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, func() {})
	sf.sigintSent = true
	sf.Update(keyMsg("ctrl+c")) // opens confirm

	// Route n to dismiss it
	sf.Update(keyMsg("n"))
	if sf.confirm != nil {
		t.Fatal("n should dismiss the confirm overlay")
	}
}

func TestStreamFrame_NonKeyMsgPassthrough(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)

	next, cmd := sf.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if next != sf {
		t.Fatal("non-key, non-stream msg should return same frame")
	}
	if cmd != nil {
		t.Fatal("non-key, non-stream msg should not produce cmd")
	}
}
