//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCLI_Version(t *testing.T) {
	stdout, _, err := runTfui("version")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.HasPrefix(stdout, "tfui ") {
		t.Errorf("expected output to start with 'tfui ', got: %q", stdout)
	}
	if !strings.Contains(stdout, "0.0.0-test") {
		t.Errorf("expected version '0.0.0-test' in output, got: %q", stdout)
	}
}

func TestCLI_Help(t *testing.T) {
	stdout, _, err := runTfui("--help")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(stdout, "terraform-ui") {
		t.Errorf("expected help output to mention terraform-ui, got: %q", stdout)
	}
	if !strings.Contains(stdout, "plan") {
		t.Errorf("expected help output to mention plan command, got: %q", stdout)
	}
	if !strings.Contains(stdout, "apply") {
		t.Errorf("expected help output to mention apply command, got: %q", stdout)
	}
}

func TestCLI_HelpSubcommand(t *testing.T) {
	stdout, _, err := runTfui("help")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(stdout, "Usage") || !strings.Contains(stdout, "plan") {
		t.Errorf("expected usage information, got: %q", stdout)
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	_, stderr, err := runTfui("bogus")
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	combined := stderr
	if !strings.Contains(combined, "unknown command") {
		t.Errorf("expected 'unknown command' in error output, got: %q", combined)
	}
}

func TestCLI_PlanInvalidMode(t *testing.T) {
	_, stderr, err := runTfui("plan", "--mode", "bogus", "--dir", fixtureDir("create"))
	if err == nil {
		t.Fatal("expected error for invalid mode, got nil")
	}
	combined := stderr
	if !strings.Contains(combined, "unknown mode") {
		t.Errorf("expected 'unknown mode' in error output, got: %q", combined)
	}
}

func TestCLI_PlanNonexistentDir(t *testing.T) {
	_, _, err := runTfui("plan", "--dir", "/nonexistent/path/does/not/exist", "--mode", "silent")
	if err == nil {
		t.Fatal("expected error for nonexistent directory, got nil")
	}
}

func TestCLI_PlanAgentModeOutputsValidJSON(t *testing.T) {
	initFixture(t, "create")

	stdout, _, err := runTfui("plan", "--dir", fixtureDir("create"), "--mode", "agent")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %q", err, stdout)
	}

	// Verify required fields exist
	requiredFields := []string{"changes", "summary", "risk", "phantom_changes", "phantom_resources"}
	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("expected field %q in agent JSON output", field)
		}
	}
}

func TestCLI_PlanSilentModeOutputsTreeView(t *testing.T) {
	initFixture(t, "create")

	stdout, _, err := runTfui("plan", "--dir", fixtureDir("create"), "--mode", "silent")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(stdout, "+ ") {
		t.Errorf("expected tree view with '+' symbols for creates, got: %q", stdout)
	}
	if !strings.Contains(stdout, "local_file.") {
		t.Errorf("expected resource addresses in output, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Plan:") {
		t.Errorf("expected 'Plan:' summary line, got: %q", stdout)
	}
}
