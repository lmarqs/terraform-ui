package main

import (
	"fmt"
	"io"
	"os"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Effects owns process I/O sinks and exit. The struct is a test seam: prod
// code uses DefaultEffects(); tests construct with custom writers and exit
// recorder.
type Effects struct {
	Stdout io.Writer
	Stderr io.Writer
	Exit   func(int)
}

// DefaultEffects returns the production Effects wired to os.Stdout/Stderr/Exit.
func DefaultEffects() Effects {
	return Effects{Stdout: os.Stdout, Stderr: os.Stderr, Exit: os.Exit}
}

// WriteStdout writes raw bytes to the stdout sink.
func (e Effects) WriteStdout(data []byte) {
	_, _ = e.Stdout.Write(data)
}

// WriteStderr writes raw bytes to the stderr sink.
func (e Effects) WriteStderr(data []byte) {
	_, _ = e.Stderr.Write(data)
}

// ExitWithCode calls the exit function if code is non-zero.
func (e Effects) ExitWithCode(code int) {
	if code != 0 {
		e.Exit(code)
	}
}

// WriteRecordedCommands prints MacroService's recorded terraform calls in the
// requested format: JSON array when jsonStdout is true, one-per-line otherwise.
func (e Effects) WriteRecordedCommands(cmds []sdk.Command, jsonStdout bool) {
	if jsonStdout {
		strs := make([]string, len(cmds))
		for i, c := range cmds {
			strs[i] = c.String()
		}
		_, _ = e.Stdout.Write(sdk.MarshalJSON(strs))
		_, _ = e.Stdout.Write([]byte("\n"))
		return
	}
	for _, c := range cmds {
		_, _ = fmt.Fprintln(e.Stdout, c.String())
	}
}
