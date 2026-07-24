//go:build integration

package integration

import (
	"strings"
	"testing"
)

// Agent environments resolve to headless mode via env vars (ADR-0022). The
// TTY-override branch itself is unit-tested on resolveSilentStderrFrom
// (stderr is never a TTY under go test); these tests pin the end-to-end
// output contract of an agent-shaped invocation: no -ci flag, env var only.

func TestPlan_ClaudeCodeEnv_RunsHeadlessWithCorrectOutput(t *testing.T) {
	initFixture(t, "create")

	stdout, stderr, err := runTfuiEnv(
		[]string{"CLAUDECODE=1"},
		"plan", "-project", fixtureDir("create"),
	)
	if !isExitCode(err, 2) {
		t.Fatalf("expected exit code 2 (changes), got err=%v\nstderr: %q", err, stderr)
	}
	if !strings.Contains(stdout, "2 to add") {
		t.Errorf("expected plan summary on stdout, got: %q", stdout)
	}
	if strings.Contains(stderr, "\x1b[") {
		t.Errorf("expected no ANSI escapes on stderr for agent invocation, got: %q", stderr)
	}
}

func TestPlan_ClaudeCodeEnv_BrokenFixtureFailsLoudly(t *testing.T) {
	initFixture(t, "broken")

	stdout, stderr, err := runTfuiEnv(
		[]string{"CLAUDECODE=1"},
		"plan", "-project", fixtureDir("broken"),
	)
	if !isExitCode(err, 1) {
		t.Fatalf("expected exit code 1, got err=%v\nstdout: %q\nstderr: %q", err, stdout, stderr)
	}
	if !strings.Contains(stderr, "not been declared") {
		t.Errorf("expected terraform's error on stderr, got: %q", stderr)
	}
}
