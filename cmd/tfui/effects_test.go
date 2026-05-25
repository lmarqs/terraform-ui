package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestEffects_WriteStdout_ShouldWriteToSink(t *testing.T) {
	var buf bytes.Buffer
	e := Effects{Stdout: &buf, Stderr: &bytes.Buffer{}, Exit: func(int) {}}
	e.WriteStdout([]byte("hello"))
	if got := buf.String(); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestEffects_WriteStderr_ShouldWriteToSink(t *testing.T) {
	var buf bytes.Buffer
	e := Effects{Stdout: &bytes.Buffer{}, Stderr: &buf, Exit: func(int) {}}
	e.WriteStderr([]byte("warning"))
	if got := buf.String(); got != "warning" {
		t.Errorf("got %q, want %q", got, "warning")
	}
}

func TestEffects_ExitWithCode_WhenZero_ShouldNotCallExit(t *testing.T) {
	called := false
	e := Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Exit: func(int) { called = true }}
	e.ExitWithCode(0)
	if called {
		t.Error("Exit called for code 0")
	}
}

func TestEffects_ExitWithCode_WhenNonZero_ShouldCallExit(t *testing.T) {
	var exitCode int
	e := Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Exit: func(c int) { exitCode = c }}
	e.ExitWithCode(2)
	if exitCode != 2 {
		t.Errorf("Exit called with %d, want 2", exitCode)
	}
}

func TestEffects_WriteRecordedCommands_WhenHuman_ShouldWriteOnePerLine(t *testing.T) {
	var buf bytes.Buffer
	e := Effects{Stdout: &buf, Stderr: &bytes.Buffer{}, Exit: func(int) {}}
	cmds := []sdk.Command{
		{Binary: "terraform", Verb: "plan"},
		{Binary: "terraform", Verb: "apply"},
	}
	e.WriteRecordedCommands(cmds, false)
	got := buf.String()
	if got != "terraform plan\nterraform apply\n" {
		t.Errorf("got %q, want human-readable one-per-line", got)
	}
}

func TestEffects_WriteRecordedCommands_WhenJSON_ShouldWriteJSONArray(t *testing.T) {
	var buf bytes.Buffer
	e := Effects{Stdout: &buf, Stderr: &bytes.Buffer{}, Exit: func(int) {}}
	cmds := []sdk.Command{
		{Binary: "terraform", Verb: "plan"},
	}
	e.WriteRecordedCommands(cmds, true)
	got := buf.String()
	if !strings.Contains(got, "terraform plan") {
		t.Errorf("got %q, want JSON containing command string", got)
	}
	if got[0] != '[' {
		t.Errorf("expected JSON array, got %q", got)
	}
}

func TestDefaultEffects_ShouldReturnNonNilFields(t *testing.T) {
	e := DefaultEffects()
	if e.Stdout == nil {
		t.Error("Stdout is nil")
	}
	if e.Stderr == nil {
		t.Error("Stderr is nil")
	}
	if e.Exit == nil {
		t.Error("Exit is nil")
	}
}
