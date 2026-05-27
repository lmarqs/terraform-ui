//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

// terraformPlan is the parsed `terraform show -json <planfile>` output that
// `tfui plan -json` passes through verbatim (ADR-0006).
type terraformPlan struct {
	FormatVersion   string                  `json:"format_version"`
	ResourceChanges []terraformResourceChange `json:"resource_changes"`
}

type terraformResourceChange struct {
	Address string          `json:"address"`
	Change  terraformChange `json:"change"`
}

type terraformChange struct {
	Actions []string `json:"actions"`
}

// summary tallies create/update/delete from terraform's resource_changes.
type summary struct {
	add     int
	change  int
	destroy int
	replace int
}

func summarize(p terraformPlan) summary {
	var s summary
	for _, rc := range p.ResourceChanges {
		switch joinActions(rc.Change.Actions) {
		case "create":
			s.add++
		case "update":
			s.change++
		case "delete":
			s.destroy++
		case "delete,create", "create,delete":
			s.replace++
		}
	}
	return s
}

func joinActions(a []string) string { return strings.Join(a, ",") }

func runPlanAgent(t *testing.T, fixture string) terraformPlan {
	t.Helper()
	initFixture(t, fixture)

	stdout, stderr, err := runTfui("plan", "-project", fixtureDir(fixture), "-json")
	if err != nil && !isExitCode(err, 2) {
		t.Fatalf("plan -json failed for fixture %q: %v\nstderr: %s", fixture, err, stderr)
	}

	var result terraformPlan
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse terraform JSON for fixture %q: %v\noutput: %q", fixture, err, stdout)
	}
	return result
}

func runPlanSilent(t *testing.T, fixture string) string {
	t.Helper()
	initFixture(t, fixture)

	stdout, stderr, err := runTfui("plan", "-project", fixtureDir(fixture), "-ci")
	if err != nil && !isExitCode(err, 2) {
		t.Fatalf("plan -ci failed for fixture %q: %v\nstderr: %s", fixture, err, stderr)
	}
	return stdout
}

// -- Create fixture tests --

func TestPlan_CreateFixture_AgentMode(t *testing.T) {
	result := runPlanAgent(t, "create")
	s := summarize(result)

	if s.add != 2 {
		t.Errorf("expected add=2, got %d", s.add)
	}
	if s.change != 0 {
		t.Errorf("expected change=0, got %d", s.change)
	}
	if s.destroy != 0 {
		t.Errorf("expected destroy=0, got %d", s.destroy)
	}
	if len(result.ResourceChanges) != 2 {
		t.Errorf("expected 2 resource_changes, got %d", len(result.ResourceChanges))
	}

	for _, rc := range result.ResourceChanges {
		if joinActions(rc.Change.Actions) != "create" {
			t.Errorf("expected actions=[create], got %v for %s", rc.Change.Actions, rc.Address)
		}
	}
}

func TestPlan_CreateFixture_SilentMode(t *testing.T) {
	output := runPlanSilent(t, "create")

	if !strings.Contains(output, "+ local_file.alpha") {
		t.Errorf("expected '+ local_file.alpha' in output, got: %q", output)
	}
	if !strings.Contains(output, "+ local_file.beta") {
		t.Errorf("expected '+ local_file.beta' in output, got: %q", output)
	}
	if !strings.Contains(output, "2 to add") {
		t.Errorf("expected '2 to add' in plan summary, got: %q", output)
	}
	if !strings.Contains(output, "0 to change") {
		t.Errorf("expected '0 to change' in plan summary, got: %q", output)
	}
	if !strings.Contains(output, "0 to destroy") {
		t.Errorf("expected '0 to destroy' in plan summary, got: %q", output)
	}
}

// -- No-changes fixture tests --

func TestPlan_NoChangesFixture_AgentMode(t *testing.T) {
	result := runPlanAgent(t, "no-changes")
	s := summarize(result)

	if s.add != 0 || s.change != 0 || s.destroy != 0 {
		t.Errorf("expected all zero, got add=%d change=%d destroy=%d", s.add, s.change, s.destroy)
	}
}

func TestPlan_NoChangesFixture_SilentMode(t *testing.T) {
	output := runPlanSilent(t, "no-changes")

	if !strings.Contains(output, "0 to add") {
		t.Errorf("expected '0 to add' in output for no-changes fixture, got: %q", output)
	}
}

// -- Delete fixture tests --

func TestPlan_DeleteFixture_AgentMode(t *testing.T) {
	result := runPlanAgent(t, "delete")
	s := summarize(result)

	if s.destroy != 1 {
		t.Errorf("expected destroy=1, got %d", s.destroy)
	}
	if s.add != 0 {
		t.Errorf("expected add=0, got %d", s.add)
	}

	var deletes []terraformResourceChange
	for _, rc := range result.ResourceChanges {
		if joinActions(rc.Change.Actions) == "delete" {
			deletes = append(deletes, rc)
		}
	}
	if len(deletes) != 1 {
		t.Fatalf("expected 1 delete, got %d", len(deletes))
	}
	if !strings.Contains(deletes[0].Address, "local_file.to_remove") {
		t.Errorf("expected address to contain 'local_file.to_remove', got %q", deletes[0].Address)
	}
}

func TestPlan_DeleteFixture_SilentMode(t *testing.T) {
	output := runPlanSilent(t, "delete")

	if !strings.Contains(output, "- local_file.to_remove") {
		t.Errorf("expected '- local_file.to_remove' in output, got: %q", output)
	}
	if !strings.Contains(output, "1 to destroy") {
		t.Errorf("expected '1 to destroy' in plan summary, got: %q", output)
	}
}

// -- Update fixture tests --

func TestPlan_UpdateFixture_AgentMode(t *testing.T) {
	result := runPlanAgent(t, "update")
	s := summarize(result)

	if s.change != 1 {
		t.Errorf("expected change=1, got %d", s.change)
	}
	if s.add != 0 {
		t.Errorf("expected add=0, got %d", s.add)
	}
	if s.destroy != 0 {
		t.Errorf("expected destroy=0, got %d", s.destroy)
	}

	found := false
	for _, rc := range result.ResourceChanges {
		if joinActions(rc.Change.Actions) == "update" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find an 'update' action in resource_changes")
	}
}

func TestPlan_UpdateFixture_SilentMode(t *testing.T) {
	output := runPlanSilent(t, "update")

	if !strings.Contains(output, "~ terraform_data.doc") {
		t.Errorf("expected '~ terraform_data.doc' in output, got: %q", output)
	}
	if !strings.Contains(output, "1 to change") {
		t.Errorf("expected '1 to change' in plan summary, got: %q", output)
	}
}

// -- Replace fixture tests --

func TestPlan_ReplaceFixture_AgentMode(t *testing.T) {
	result := runPlanAgent(t, "replace")

	if len(result.ResourceChanges) < 1 {
		t.Fatalf("expected at least 1 resource_change, got %d", len(result.ResourceChanges))
	}

	foundReplace := false
	for _, rc := range result.ResourceChanges {
		actions := joinActions(rc.Change.Actions)
		if actions == "delete,create" || actions == "create,delete" {
			foundReplace = true
			if !strings.Contains(rc.Address, "local_file.moved") {
				t.Errorf("expected replace address to contain 'local_file.moved', got %q", rc.Address)
			}
			break
		}
	}
	if !foundReplace {
		t.Errorf("expected a replace action in resource_changes, got: %v", result.ResourceChanges)
	}
}

func TestPlan_ReplaceFixture_SilentMode(t *testing.T) {
	output := runPlanSilent(t, "replace")

	if !strings.Contains(output, "-/+ local_file.moved") {
		t.Errorf("expected '-/+ local_file.moved' in output, got: %q", output)
	}
}

// -- Multi-resource fixture tests --

func TestPlan_MultiResourceFixture_AgentMode(t *testing.T) {
	result := runPlanAgent(t, "multi-resource")
	s := summarize(result)

	if s.add != 5 {
		t.Errorf("expected add=5, got %d", s.add)
	}
	if len(result.ResourceChanges) != 5 {
		t.Errorf("expected 5 resource_changes, got %d", len(result.ResourceChanges))
	}
}

func TestPlan_MultiResourceFixture_SilentMode(t *testing.T) {
	output := runPlanSilent(t, "multi-resource")

	if !strings.Contains(output, "5 to add") {
		t.Errorf("expected '5 to add' in plan summary, got: %q", output)
	}
	if !strings.Contains(output, "0 to change") {
		t.Errorf("expected '0 to change' in plan summary, got: %q", output)
	}
	if !strings.Contains(output, "0 to destroy") {
		t.Errorf("expected '0 to destroy' in plan summary, got: %q", output)
	}
}

// -- Output JSON structure tests --

func TestPlan_AgentMode_JSONStructure(t *testing.T) {
	initFixture(t, "create")

	stdout, _, err := runTfui("plan", "-project", fixtureDir("create"), "-json")
	if err != nil && !isExitCode(err, 2) {
		t.Fatalf("plan failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// terraform show -json top-level keys (ADR-0006 passthrough)
	expectedFields := map[string]string{
		"format_version":   "string",
		"resource_changes": "array",
		"planned_values":   "object",
	}

	for field, expectedType := range expectedFields {
		val, ok := raw[field]
		if !ok {
			t.Errorf("missing field %q in terraform JSON output", field)
			continue
		}
		switch expectedType {
		case "array":
			if _, ok := val.([]interface{}); !ok {
				t.Errorf("field %q expected array, got %T", field, val)
			}
		case "object":
			if _, ok := val.(map[string]interface{}); !ok {
				t.Errorf("field %q expected object, got %T", field, val)
			}
		case "string":
			if _, ok := val.(string); !ok {
				t.Errorf("field %q expected string, got %T", field, val)
			}
		}
	}

	// resource_changes entries must have address + change.actions
	rcs, _ := raw["resource_changes"].([]interface{})
	if len(rcs) > 0 {
		rc, ok := rcs[0].(map[string]interface{})
		if !ok {
			t.Fatal("resource_changes[0] is not an object")
		}
		if _, ok := rc["address"]; !ok {
			t.Error("missing resource_changes[0].address")
		}
		change, ok := rc["change"].(map[string]interface{})
		if !ok {
			t.Fatal("resource_changes[0].change is not an object")
		}
		if _, ok := change["actions"]; !ok {
			t.Error("missing resource_changes[0].change.actions")
		}
	}
}
