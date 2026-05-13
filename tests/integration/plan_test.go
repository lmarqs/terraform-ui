//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

// agentJSON is the parsed structure of --output json output.
type agentJSON struct {
	Changes          []agentChange `json:"changes"`
	Summary          agentSummary  `json:"summary"`
	Risk             string        `json:"risk"`
	PhantomChanges   int           `json:"phantom_changes"`
	PhantomResources []string      `json:"phantom_resources"`
}

type agentChange struct {
	Address string `json:"address"`
	Action  string `json:"action"`
	Risk    string `json:"risk"`
	Phantom bool   `json:"phantom,omitempty"`
}

type agentSummary struct {
	Add     int `json:"add"`
	Change  int `json:"change"`
	Destroy int `json:"destroy"`
}

func runPlanAgent(t *testing.T, fixture string) agentJSON {
	t.Helper()
	initFixture(t, fixture)

	stdout, stderr, err := runTfui("plan", "--project", fixtureDir(fixture), "--output", "json")
	if err != nil {
		t.Fatalf("plan --output json failed for fixture %q: %v\nstderr: %s", fixture, err, stderr)
	}

	var result agentJSON
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse agent JSON for fixture %q: %v\noutput: %q", fixture, err, stdout)
	}
	return result
}

func runPlanSilent(t *testing.T, fixture string) string {
	t.Helper()
	initFixture(t, fixture)

	stdout, stderr, err := runTfui("plan", "--project", fixtureDir(fixture), "--ci")
	if err != nil {
		t.Fatalf("plan --ci failed for fixture %q: %v\nstderr: %s", fixture, err, stderr)
	}
	return stdout
}

// -- Create fixture tests --

func TestPlan_CreateFixture_AgentMode(t *testing.T) {
	result := runPlanAgent(t, "create")

	if result.Summary.Add != 2 {
		t.Errorf("expected summary.add=2, got %d", result.Summary.Add)
	}
	if result.Summary.Change != 0 {
		t.Errorf("expected summary.change=0, got %d", result.Summary.Change)
	}
	if result.Summary.Destroy != 0 {
		t.Errorf("expected summary.destroy=0, got %d", result.Summary.Destroy)
	}
	if len(result.Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(result.Changes))
	}

	for _, c := range result.Changes {
		if c.Action != "create" {
			t.Errorf("expected action 'create', got %q for %s", c.Action, c.Address)
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

	if result.Summary.Add != 0 {
		t.Errorf("expected summary.add=0, got %d", result.Summary.Add)
	}
	if result.Summary.Change != 0 {
		t.Errorf("expected summary.change=0, got %d", result.Summary.Change)
	}
	if result.Summary.Destroy != 0 {
		t.Errorf("expected summary.destroy=0, got %d", result.Summary.Destroy)
	}
	if len(result.Changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(result.Changes))
	}
}

func TestPlan_NoChangesFixture_SilentMode(t *testing.T) {
	output := runPlanSilent(t, "no-changes")

	// With no changes, the output should show 0 to add/change/destroy
	if !strings.Contains(output, "0 to add") {
		t.Errorf("expected '0 to add' in output for no-changes fixture, got: %q", output)
	}
}

// -- Delete fixture tests --

func TestPlan_DeleteFixture_AgentMode(t *testing.T) {
	result := runPlanAgent(t, "delete")

	if result.Summary.Destroy != 1 {
		t.Errorf("expected summary.destroy=1, got %d", result.Summary.Destroy)
	}
	if result.Summary.Add != 0 {
		t.Errorf("expected summary.add=0, got %d", result.Summary.Add)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	if result.Changes[0].Action != "delete" {
		t.Errorf("expected action 'delete', got %q", result.Changes[0].Action)
	}
	if !strings.Contains(result.Changes[0].Address, "local_file.to_remove") {
		t.Errorf("expected address to contain 'local_file.to_remove', got %q", result.Changes[0].Address)
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

	if result.Summary.Change != 1 {
		t.Errorf("expected summary.change=1, got %d", result.Summary.Change)
	}
	if result.Summary.Add != 0 {
		t.Errorf("expected summary.add=0, got %d", result.Summary.Add)
	}
	if result.Summary.Destroy != 0 {
		t.Errorf("expected summary.destroy=0, got %d", result.Summary.Destroy)
	}
	if len(result.Changes) < 1 {
		t.Fatalf("expected at least 1 change, got %d", len(result.Changes))
	}

	found := false
	for _, c := range result.Changes {
		if c.Action == "update" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find an 'update' action in changes")
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

	// Replace may be counted as change in summary since it's ToReplace added to Change
	if len(result.Changes) < 1 {
		t.Fatalf("expected at least 1 change, got %d", len(result.Changes))
	}

	foundReplace := false
	for _, c := range result.Changes {
		if c.Action == "delete-then-create" || c.Action == "create-then-delete" {
			foundReplace = true
			if !strings.Contains(c.Address, "local_file.moved") {
				t.Errorf("expected replace address to contain 'local_file.moved', got %q", c.Address)
			}
			break
		}
	}
	if !foundReplace {
		t.Errorf("expected a replace action in changes, got actions: %v", result.Changes)
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

	if result.Summary.Add != 5 {
		t.Errorf("expected summary.add=5, got %d", result.Summary.Add)
	}
	if len(result.Changes) != 5 {
		t.Errorf("expected 5 changes, got %d", len(result.Changes))
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

	stdout, _, err := runTfui("plan", "--project", fixtureDir("create"), "--output", "json")
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	// Verify it's valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify top-level fields
	expectedFields := map[string]string{
		"changes":           "array",
		"summary":           "object",
		"risk":              "string",
		"phantom_changes":   "number",
		"phantom_resources": "array",
	}

	for field, expectedType := range expectedFields {
		val, ok := raw[field]
		if !ok {
			t.Errorf("missing field %q in agent output", field)
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
		case "number":
			if _, ok := val.(float64); !ok {
				t.Errorf("field %q expected number, got %T", field, val)
			}
		}
	}

	// Verify summary sub-fields
	summary, _ := raw["summary"].(map[string]interface{})
	for _, field := range []string{"add", "change", "destroy"} {
		if _, ok := summary[field]; !ok {
			t.Errorf("missing field summary.%s in agent output", field)
		}
	}

	// Verify change entry structure
	changes, _ := raw["changes"].([]interface{})
	if len(changes) > 0 {
		change, ok := changes[0].(map[string]interface{})
		if !ok {
			t.Fatal("change entry is not an object")
		}
		for _, field := range []string{"address", "action", "risk"} {
			if _, ok := change[field]; !ok {
				t.Errorf("missing field changes[0].%s in agent output", field)
			}
		}
	}
}
