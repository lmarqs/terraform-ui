package editor

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeExecCommand struct {
	err            error
	heldDuringRun  bool
	stdin          io.Reader
	stdout, stderr io.Writer
}

func (f *fakeExecCommand) Run() error {
	f.heldDuringRun = HoldsTerminal()
	return f.err
}
func (f *fakeExecCommand) SetStdin(r io.Reader)  { f.stdin = r }
func (f *fakeExecCommand) SetStdout(w io.Writer) { f.stdout = w }
func (f *fakeExecCommand) SetStderr(w io.Writer) { f.stderr = w }

func TestTerminalHandoff_Run_ShouldHoldTerminalOnlyWhileCommandRuns(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"command succeeds", nil},
		{"command fails", errors.New("editor failed")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := &fakeExecCommand{err: tt.err}

			err := terminalHandoff{inner}.Run()

			if !errors.Is(err, tt.err) {
				t.Errorf("Run() error = %v, want %v", err, tt.err)
			}
			if !inner.heldDuringRun {
				t.Error("HoldsTerminal() = false while the command ran, want true")
			}
			if HoldsTerminal() {
				t.Error("HoldsTerminal() = true after the command returned, want false")
			}
		})
	}
}

func TestTerminalHandoff_ShouldPassStreamsToCommand(t *testing.T) {
	inner := &fakeExecCommand{}
	handoff := terminalHandoff{inner}
	in, out, errOut := strings.NewReader("in"), &bytes.Buffer{}, &bytes.Buffer{}

	handoff.SetStdin(in)
	handoff.SetStdout(out)
	handoff.SetStderr(errOut)

	if inner.stdin != in || inner.stdout != out || inner.stderr != errOut {
		t.Error("streams were not passed to the wrapped command")
	}
}

func TestOSCommand_SetStreams_ShouldFillOnlyUnsetStreams(t *testing.T) {
	preset, terminal := &bytes.Buffer{}, &bytes.Buffer{}
	presetIn, terminalIn := strings.NewReader("preset"), strings.NewReader("terminal")
	tests := []struct {
		name                   string
		cmd                    *exec.Cmd
		wantIn                 io.Reader
		wantOut, wantErrStream io.Writer
	}{
		{"unset streams take the terminal's", &exec.Cmd{}, terminalIn, terminal, terminal},
		{
			"preset streams are kept",
			&exec.Cmd{Stdin: presetIn, Stdout: preset, Stderr: preset},
			presetIn, preset, preset,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := osCommand{tt.cmd}

			c.SetStdin(terminalIn)
			c.SetStdout(terminal)
			c.SetStderr(terminal)

			if tt.cmd.Stdin != tt.wantIn || tt.cmd.Stdout != tt.wantOut || tt.cmd.Stderr != tt.wantErrStream {
				t.Errorf("streams = (%v, %v, %v), want (%v, %v, %v)",
					tt.cmd.Stdin, tt.cmd.Stdout, tt.cmd.Stderr, tt.wantIn, tt.wantOut, tt.wantErrStream)
			}
		})
	}
}

// execCommandOf returns the dynamic type of the ExecCommand that a tea.Exec
// message carries; bubbletea keeps the message type unexported.
func execCommandOf(t *testing.T, cmd tea.Cmd) reflect.Type {
	t.Helper()
	if cmd == nil {
		t.Fatal("cmd is nil")
	}
	field := reflect.ValueOf(cmd()).FieldByName("cmd")
	if !field.IsValid() || field.IsNil() {
		t.Fatal("message carries no ExecCommand")
	}
	return field.Elem().Type()
}

func TestOpen_ShouldHandTerminalThroughTerminalHandoff(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "code --wait")
	tests := []struct {
		name string
		cmd  tea.Cmd
	}{
		{"Open", Open(SourceLocation{File: "/tmp/a.tf", Line: 1})},
		{"OpenMultiple with code", OpenMultiple([]SourceLocation{{File: "/tmp/a.tf", Line: 1}, {File: "/tmp/b.tf"}})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := execCommandOf(t, tt.cmd), reflect.TypeOf(terminalHandoff{}); got != want {
				t.Errorf("ExecCommand type = %v, want %v", got, want)
			}
		})
	}
}
