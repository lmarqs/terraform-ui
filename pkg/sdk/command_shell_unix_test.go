//go:build unix

package sdk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommand_String_WhenRunByAShell_ShouldDeliverTheExactArguments runs the
// serialized command against a stub that prints one argument per line.
func TestCommand_String_WhenRunByAShell_ShouldDeliverTheExactArguments(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "terraform")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := Command{
		Binary: stub,
		Verb:   "state rm",
		Flags:  []string{`-var=a "b" 'c'`},
		Args:   []string{`aws_instance.web["a"]`, "it's $HOME", ""},
		Dir:    "/work/my modules/vpc",
	}

	out, err := exec.Command("sh", "-c", cmd.String()).Output()
	if err != nil {
		t.Fatalf("sh -c %q: %v", cmd.String(), err)
	}

	want := []string{"-chdir=/work/my modules/vpc", "state", "rm", `-var=a "b" 'c'`, `aws_instance.web["a"]`, "it's $HOME", ""}
	got := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("stub received %q, want %q", got, want)
	}
}
