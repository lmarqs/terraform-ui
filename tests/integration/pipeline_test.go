//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPipeline_Plan(t *testing.T) {
	binary := testBinary()
	dir := initFixtureWith(t, "create", binary)

	stdout, stderr, err := runTfui("plan", "--project", dir, "--terraform-bin", binary, "--output", "json")
	if err != nil {
		t.Fatalf("plan --output json with %s failed: %v\nstderr: %s", binary, err, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("expected valid JSON from %s, got: %v\noutput: %q", binary, err, stdout)
	}

	if _, ok := result["changes"]; !ok {
		t.Errorf("expected 'changes' field in JSON output from %s", binary)
	}
	if _, ok := result["summary"]; !ok {
		t.Errorf("expected 'summary' field in JSON output from %s", binary)
	}
}

func TestPipeline_PlanTreeView(t *testing.T) {
	binary := testBinary()
	dir := initFixtureWith(t, "create", binary)

	stdout, _, err := runTfui("plan", "--project", dir, "--terraform-bin", binary, "--ci")
	if err != nil {
		t.Fatalf("plan --ci with %s failed: %v", binary, err)
	}

	if !strings.Contains(stdout, "Plan:") {
		t.Errorf("expected 'Plan:' summary in tree output from %s, got: %q", binary, stdout)
	}
}
