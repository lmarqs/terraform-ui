package frames

import (
	"bytes"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// StreamLineMsg carries a single log line from an in-progress terraform command.
type StreamLineMsg struct{ Line string }

// StreamDoneMsg signals that the command's output channel has been closed.
type StreamDoneMsg struct{}

// LineWriter implements io.Writer, splitting input into lines and
// forwarding each complete line to a channel for BubbleTea consumption.
type LineWriter struct {
	mu     sync.Mutex
	ch     chan<- string
	buf    []byte
	closed bool
}

// NewLineWriter creates a LineWriter. The returned receive-only channel carries
// one string per complete line. Call Close after the command finishes to flush
// any partial trailing line and signal channel closure to consumers.
func NewLineWriter() (*LineWriter, <-chan string) {
	ch := make(chan string, 256)
	return &LineWriter{ch: ch}, ch
}

// Write splits p on newlines and sends each complete line to the channel.
// Lines exceeding the buffer capacity are silently dropped to avoid blocking.
func (w *LineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]
		select {
		case w.ch <- line:
		default:
		}
	}
	return len(p), nil
}

// Close flushes any trailing partial line and closes the channel. Idempotent.
func (w *LineWriter) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.closed = true
	if len(w.buf) > 0 {
		select {
		case w.ch <- string(w.buf):
		default:
		}
	}
	close(w.ch)
}

// WaitForLine returns a Cmd that blocks until one line arrives on ch or the channel closes.
// On close it emits StreamDoneMsg; otherwise StreamLineMsg.
func WaitForLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return StreamDoneMsg{}
		}
		return StreamLineMsg{Line: line}
	}
}

// StreamFrame renders streaming terraform output as a scrollable log.
// It auto-scrolls to the bottom while the command is running; manual scroll
// disables auto-scroll until the user presses G.
//
// Cancellation: ^c (first press) calls cancelFn for a graceful SIGINT.
// A second ^c opens an internal ConfirmFrame to guard against accidental force-kill.
type StreamFrame struct {
	title      string
	lines      []string
	scrollY    int
	autoScroll bool
	done       bool
	ch         <-chan string
	cancelFn   func()
	sigintSent bool
	confirm    *ConfirmFrame
	panel      *ui.ContentPanel
}

// NewStreamFrame creates a StreamFrame.
// title is displayed as context (e.g. "terraform plan").
// ch receives log lines produced by a LineWriter.
// cancelFn is invoked on the first ^c for graceful cancellation.
func NewStreamFrame(title string, ch <-chan string, cancelFn func()) *StreamFrame {
	return &StreamFrame{
		title:      title,
		autoScroll: true,
		ch:         ch,
		cancelFn:   cancelFn,
		panel:      ui.NewContentPanel(),
	}
}

func (f *StreamFrame) ID() string { return "stream" }

func (f *StreamFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	if f.confirm != nil {
		next, cmd := f.confirm.Update(msg)
		if next == nil {
			f.confirm = nil
		}
		return f, cmd
	}

	switch msg := msg.(type) {
	case StreamLineMsg:
		f.lines = append(f.lines, msg.Line)
		return f, WaitForLine(f.ch)

	case StreamDoneMsg:
		f.done = true
		return f, nil

	case tea.KeyMsg:
		if f.done {
			if msg.String() == "esc" {
				return nil, nil
			}
			f.handleScroll(msg)
			return f, nil
		}
		switch msg.String() {
		case "ctrl+c":
			if !f.sigintSent {
				f.sigintSent = true
				if f.cancelFn != nil {
					f.cancelFn()
				}
				return f, nil
			}
			f.confirm = NewConfirmFrame(
				"Force cancel? Infrastructure may be left in a partial state.",
				func() tea.Cmd {
					if f.cancelFn != nil {
						f.cancelFn()
					}
					return nil
				},
				nil,
			)
			return f, nil
		default:
			f.handleScroll(msg)
			return f, nil
		}
	}
	return f, nil
}

func (f *StreamFrame) handleScroll(msg tea.KeyMsg) {
	switch msg.String() {
	case "up", "k":
		if f.scrollY > 0 {
			f.scrollY--
		}
		f.autoScroll = false
	case "down", "j":
		f.scrollY++
		f.autoScroll = false
	case "G":
		f.scrollY = len(f.lines)
		f.autoScroll = true
	case "g":
		f.scrollY = 0
		f.autoScroll = false
	}
}

func (f *StreamFrame) View(width, height int) string {
	if f.confirm != nil {
		return f.confirm.View(width, height)
	}
	if height <= 0 {
		height = 20
	}

	// Advance auto-scroll to the bottom before rendering.
	if f.autoScroll && len(f.lines) > 0 {
		f.scrollY = len(f.lines) - 1
	}

	maxScroll := len(f.lines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if f.scrollY > maxScroll {
		f.scrollY = maxScroll
	}
	if f.scrollY < 0 {
		f.scrollY = 0
	}

	end := f.scrollY + height
	if end > len(f.lines) {
		end = len(f.lines)
	}

	return f.panel.Render(ui.RenderParams{
		Rows:         f.lines[f.scrollY:end],
		Width:        width,
		Height:       height,
		TotalItems:   len(f.lines),
		Cursor:       -1,
		ScrollOffset: f.scrollY,
	})
}

func (f *StreamFrame) Hints() []sdk.KeyHint {
	if f.confirm != nil {
		return f.confirm.Hints()
	}
	if f.done {
		return []sdk.KeyHint{sdk.HintBack}
	}
	if f.sigintSent {
		return []sdk.KeyHint{{Key: "^c", Description: "force cancel"}}
	}
	return []sdk.KeyHint{{Key: "^c", Description: "cancel"}}
}

// Lines returns a snapshot of all accumulated log lines.
// Useful for re-displaying the log after the frame has been popped.
func (f *StreamFrame) Lines() []string {
	result := make([]string, len(f.lines))
	copy(result, f.lines)
	return result
}
