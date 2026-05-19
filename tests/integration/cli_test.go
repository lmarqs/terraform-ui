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
	stdout, _, err := runTfui("-help")
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

func TestCLI_PlanUnknownFlag(t *testing.T) {
	_, stderr, err := runTfui("plan", "-bogus", "-project", fixtureDir("create"))
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
	if !strings.Contains(stderr, "unknown flag") {
		t.Errorf("expected 'unknown flag' in error output, got: %q", stderr)
	}
}

func TestCLI_PlanNonexistentDir(t *testing.T) {
	_, _, err := runTfui("plan", "-project", "/nonexistent/path/does/not/exist", "-ci")
	if err == nil {
		t.Fatal("expected error for nonexistent directory, got nil")
	}
}

func TestCLI_PlanJSONValid(t *testing.T) {
	initFixture(t, "create")

	stdout, _, err := runTfui("plan", "-project", fixtureDir("create"), "-json")
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

func TestCLI_PlanCIModeOutputsTreeView(t *testing.T) {
	initFixture(t, "create")

	stdout, _, err := runTfui("plan", "-project", fixtureDir("create"), "-ci")
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
