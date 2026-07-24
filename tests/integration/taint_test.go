//go:build integration

package integration

import (
	"strings"
	"testing"
)

// Headless action verbs skip the interactive confirmation (ADR-0022): CLI
// arguments are declared intent, mirroring terraform's own prompt-free
// taint/untaint/import.

func TestTaint_SilentMode_RunsWithoutPromptAndReportsOnStderr(t *testing.T) {
	dir := copyFixture(t, "state-ops")
	applyInDir(t, dir)

	stdout, stderr, err := runTfui("taint", "-project", dir, "-ci", "local_file.one")
	if err != nil {
		t.Fatalf("taint failed: %v\nstdout: %q\nstderr: %q", err, stdout, stderr)
	}
	if !strings.Contains(stderr, "Tainted local_file.one") {
		t.Errorf("expected taint confirmation on stderr, got: %q", stderr)
	}

	stateOut, _, err := runTfui("state", "-project", dir, "-ci", "-json")
	if err != nil {
		t.Fatalf("state failed: %v", err)
	}
	if !strings.Contains(stateOut, "tainted") {
		t.Errorf("expected tainted resource in state, got: %q", stateOut)
	}
}

func TestTaint_SilentMode_BogusAddressFailsWithError(t *testing.T) {
	dir := copyFixture(t, "state-ops")
	applyInDir(t, dir)

	stdout, stderr, err := runTfui("taint", "-project", dir, "-ci", "local_file.nope")
	if !isExitCode(err, 1) {
		t.Fatalf("expected exit code 1 for bogus address, got err=%v\nstdout: %q\nstderr: %q", err, stdout, stderr)
	}
	if stderr == "" {
		t.Error("expected terraform's error on stderr, got empty")
	}
}
