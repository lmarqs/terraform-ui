//go:build integration

package integration

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// stateList runs terraform state list in the given directory.
func stateList(t *testing.T, dir string) []string {
	t.Helper()
	cmd := exec.Command("terraform", "state", "list")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("terraform state list failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func TestState_Rm_RemovesResource(t *testing.T) {
	dir := copyFixture(t, "state-ops")

	// Verify both resources exist initially
	resources := stateList(t, dir)
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources initially, got %d: %v", len(resources), resources)
	}

	// Remove one
	_, stderr, err := runTfui("state", "rm", "local_file.one", "-project", dir)
	if err != nil {
		t.Fatalf("state rm failed: %v\nstderr: %s", err, stderr)
	}

	// Verify only one remains
	resources = stateList(t, dir)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource after rm, got %d: %v", len(resources), resources)
	}
	if resources[0] != "local_file.two" {
		t.Errorf("expected remaining resource to be 'local_file.two', got %q", resources[0])
	}
}

func TestState_Mv_RenamesResource(t *testing.T) {
	dir := copyFixture(t, "state-ops")

	// Move/rename
	_, stderr, err := runTfui("state", "mv", "local_file.one", "local_file.renamed", "-project", dir)
	if err != nil {
		t.Fatalf("state mv failed: %v\nstderr: %s", err, stderr)
	}

	// Verify renamed
	resources := stateList(t, dir)
	found := false
	for _, r := range resources {
		if r == "local_file.renamed" {
			found = true
		}
		if r == "local_file.one" {
			t.Error("old address 'local_file.one' still exists after mv")
		}
	}
	if !found {
		t.Errorf("expected 'local_file.renamed' in state, got: %v", resources)
	}
}

func TestState_Taint_MarksForRecreation(t *testing.T) {
	dir := copyFixture(t, "state-ops")

	// Taint
	_, stderr, err := runTfui("state", "taint", "local_file.one", "-project", dir)
	if err != nil {
		t.Fatalf("state taint failed: %v\nstderr: %s", err, stderr)
	}

	// Verify: subsequent plan should show replace action
	result := runPlanAgentInDir(t, dir)
	foundReplace := false
	for _, rc := range result.ResourceChanges {
		if strings.Contains(rc.Address, "local_file.one") {
			actions := joinActions(rc.Change.Actions)
			if actions == "delete,create" || actions == "create,delete" {
				foundReplace = true
			}
		}
	}
	if !foundReplace {
		t.Errorf("expected tainted resource to show replace in plan, got resource_changes: %v", result.ResourceChanges)
	}
}

func TestState_Untaint_RemovesMark(t *testing.T) {
	dir := copyFixture(t, "state-ops")

	// Taint then untaint
	_, _, err := runTfui("state", "taint", "local_file.one", "-project", dir)
	if err != nil {
		t.Fatalf("taint failed: %v", err)
	}
	_, stderr, err := runTfui("state", "untaint", "local_file.one", "-project", dir)
	if err != nil {
		t.Fatalf("untaint failed: %v\nstderr: %s", err, stderr)
	}

	// Verify: plan should show no changes (resource is current)
	result := runPlanAgentInDir(t, dir)
	s := summarize(result)
	if s.add != 0 || s.change != 0 || s.destroy != 0 || s.replace != 0 {
		t.Errorf("expected no changes after untaint, got: add=%d change=%d destroy=%d replace=%d",
			s.add, s.change, s.destroy, s.replace)
	}
}

func TestState_Rm_InvalidAddress_Errors(t *testing.T) {
	dir := copyFixture(t, "state-ops")

	_, _, err := runTfui("state", "rm", "nonexistent.resource", "-project", dir)
	if err == nil {
		t.Error("expected error for state rm of nonexistent address, got nil")
	}
}

// runPlanAgentInDir runs plan in -json mode against the given directory.
func runPlanAgentInDir(t *testing.T, dir string) terraformPlan {
	t.Helper()
	stdout, stderr, err := runTfui("plan", "-project", dir, "-json")
	if err != nil && !isExitCode(err, 2) {
		t.Fatalf("plan -json failed: %v\nstderr: %s", err, stderr)
	}

	var result terraformPlan
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse terraform JSON: %v\noutput: %q", err, stdout)
	}
	return result
}
