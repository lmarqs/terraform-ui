//go:build integration

package integration

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestPipeline_Plan(t *testing.T) {
	dir := initFixture(t, "create")

	stdout, stderr, err := runTfui("plan", "-project", dir, "-json")
	if err != nil {
		if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 2 {
			t.Fatalf("plan -json failed: %v\nstderr: %s", err, stderr)
		}
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("expected valid JSON, got: %v\noutput: %q", err, stdout)
	}

	if _, ok := result["changes"]; !ok {
		t.Errorf("expected 'changes' field in JSON output")
	}
	if _, ok := result["summary"]; !ok {
		t.Errorf("expected 'summary' field in JSON output")
	}
}

func TestPipeline_PlanTreeView(t *testing.T) {
	dir := initFixture(t, "create")

	stdout, _, err := runTfui("plan", "-project", dir, "-ci")
	if err != nil {
		if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 2 {
			t.Fatalf("plan -ci failed: %v", err)
		}
	}

	if !strings.Contains(stdout, "Plan:") {
		t.Errorf("expected 'Plan:' summary in tree output, got: %q", stdout)
	}
}
