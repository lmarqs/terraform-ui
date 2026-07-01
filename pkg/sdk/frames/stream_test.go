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

// makeStreamWithElapsed builds a StreamFrame wired with an elapsed provider,
// mirroring how the plan/apply/init plugins construct it from their timer.
func makeStreamWithElapsed(lines []string, elapsed func() string) *StreamFrame {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("terraform plan", ch, nil).WithElapsed(elapsed)
	sf.lines = lines
	return sf
}

func TestStreamFrame_ViewShowsElapsedHeaderWhenNoLines(t *testing.T) {
	sf := makeStreamWithElapsed(nil, func() string { return "3s" })
	view := sf.View(80, 24)
	if !strings.Contains(view, "3s") {
		t.Errorf("view with an elapsed provider should show elapsed before any output, got %q", view)
	}
	if !strings.Contains(view, "terraform plan") {
		t.Errorf("elapsed header should carry the stream title, got %q", view)
	}
}

func TestStreamFrame_ViewElapsedHeaderIsFirstLine(t *testing.T) {
	sf := makeStreamWithElapsed([]string{"Refreshing state..."}, func() string { return "42s" })
	sf.done = true
	sf.autoScroll = false

	view := sf.View(80, 24)
	first := strings.SplitN(view, "\n", 2)[0]
	if !strings.Contains(first, "42s") {
		t.Errorf("first line should be the elapsed time, got %q", first)
	}
	if !strings.Contains(view, "Refreshing state...") {
		t.Errorf("log output should still render below the header, got %q", view)
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

func TestLineWriter_CloseIsIdempotent(t *testing.T) {
	lw, ch := NewLineWriter()
	lw.Close()
	lw.Close() // second close must not panic or re-close the channel

	var lines []string
	for line := range ch {
		lines = append(lines, line)
	}
	if len(lines) != 0 {
		t.Errorf("got lines %v after double-close, want []", lines)
	}
}

func TestLineWriter_DropsSilentlyWhenChannelFull(t *testing.T) {
	lw, _ := NewLineWriter() // intentionally do not drain
	// 256 fills the buffer; the 257th hits the default: branch
	for i := 0; i < 257; i++ {
		lw.Write([]byte("line\n")) //nolint
	}
	lw.Close()
}

func TestLineWriter_CloseDropsPartialLineWhenChannelFull(t *testing.T) {
	lw, _ := NewLineWriter() // intentionally do not drain
	// Fill all 256 channel slots with complete lines
	for i := 0; i < 256; i++ {
		lw.Write([]byte("line\n")) //nolint
	}
	lw.Write([]byte("partial")) // no newline → sits in buf
	lw.Close()                  // tries to flush "partial" into full channel → default:
}

func TestStreamFrame_DoneNonEscKeyCallsHandleScroll(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)
	sf.done = true
	sf.lines = []string{"a", "b", "c", "d", "e"}
	sf.scrollY = 4
	sf.autoScroll = false

	// "g" (lowercase) when done: hits lines 143-144 AND the g branch of handleScroll
	next, cmd := sf.Update(keyMsg("g"))
	if next == nil {
		t.Fatal("non-esc key when done should not pop frame")
	}
	if cmd != nil {
		t.Errorf("non-esc key when done should return nil cmd, got %T", cmd)
	}
	if sf.scrollY != 0 {
		t.Errorf("scrollY = %d, want 0 after g", sf.scrollY)
	}
	if sf.autoScroll {
		t.Fatal("g key should not enable autoScroll")
	}
}

func TestStreamFrame_DownKeyScrollsDown(t *testing.T) {
	_, ch := NewLineWriter()
	sf := NewStreamFrame("test", ch, nil)
	sf.lines = []string{"a", "b", "c"}
	sf.scrollY = 0
	sf.autoScroll = false

	sf.Update(keyMsg("down"))
	if sf.scrollY != 1 {
		t.Errorf("scrollY = %d, want 1 after down", sf.scrollY)
	}
	if sf.autoScroll {
		t.Fatal("down key should disable autoScroll")
	}
}

func TestStreamFrame_ViewWithZeroHeightFallsBackTo20(t *testing.T) {
	sf := makeStream([]string{"line1"})
	sf.done = true
	sf.autoScroll = false

	// height=0 triggers the height<=0 fallback; should render without panic
	view := sf.View(80, 0)
	if !strings.Contains(view, "line1") {
		t.Errorf("view with height=0 should fallback to height=20 and render lines, got %q", view)
	}
}

func TestStreamFrame_ViewNegativeScrollYClampsToZero(t *testing.T) {
	sf := makeStream([]string{"line1"})
	sf.done = true
	sf.autoScroll = false
	sf.scrollY = -1

	sf.View(80, 24)
	if sf.scrollY != 0 {
		t.Errorf("scrollY after View = %d, want 0 (clamped from -1)", sf.scrollY)
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
