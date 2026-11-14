package main

import (
	"fmt"
	"io"
	"os"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Effects owns the process I/O sinks. The struct is a test seam: prod code
// uses DefaultEffects(); tests construct it with custom writers.
type Effects struct {
	Stdout io.Writer
	Stderr io.Writer
}

// DefaultEffects returns the production Effects wired to os.Stdout/Stderr.
func DefaultEffects() Effects {
	return Effects{Stdout: os.Stdout, Stderr: os.Stderr}
}

// WriteStdout writes raw bytes to the stdout sink.
func (e Effects) WriteStdout(data []byte) {
	_, _ = e.Stdout.Write(data)
}

// WriteStderr writes raw bytes to the stderr sink.
func (e Effects) WriteStderr(data []byte) {
	_, _ = e.Stderr.Write(data)
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
