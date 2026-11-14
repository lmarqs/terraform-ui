package editor

import (
	"io"
	"os/exec"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

var holdingTerminal atomic.Bool

// HoldsTerminal reports whether an editor this package started has the
// terminal. In cooked mode a Ctrl+C meant for the editor also signals tfui.
func HoldsTerminal() bool {
	return holdingTerminal.Load()
}

// terminalHandoff marks the terminal as held by the editor while the wrapped
// command runs.
type terminalHandoff struct {
	tea.ExecCommand
}

func (h terminalHandoff) Run() error {
	holdingTerminal.Store(true)
	defer holdingTerminal.Store(false)
	return h.ExecCommand.Run()
}

// osCommand is an *exec.Cmd as a tea.ExecCommand: a stream the command
// already sets is kept, an unset one takes the terminal's.
type osCommand struct {
	*exec.Cmd
}

func (c osCommand) SetStdin(r io.Reader) {
	if c.Stdin == nil {
		c.Stdin = r
	}
}

func (c osCommand) SetStdout(w io.Writer) {
	if c.Stdout == nil {
		c.Stdout = w
	}
}

func (c osCommand) SetStderr(w io.Writer) {
	if c.Stderr == nil {
		c.Stderr = w
	}
}

// execEditor hands the terminal to the editor command, as tea.ExecProcess
// does, and marks it held while the editor runs.
func execEditor(c *exec.Cmd, fn tea.ExecCallback) tea.Cmd {
	return tea.Exec(terminalHandoff{osCommand{c}}, fn)
}
