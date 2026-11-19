package terraform

import (
	"encoding/json"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
)

func TestParsePlan_GivenDeposedResourceChange_ShouldSetResourceDeposedKey(t *testing.T) {
	plan := &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{
		{Address: "a.b", Type: "a", Name: "b", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionUpdate}}},
		{Address: "a.b", Type: "a", Name: "b", DeposedKey: "00000001", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionDelete}}},
	}}

	summary := ParsePlan(plan)

	if len(summary.Changes) != 2 {
		t.Fatalf("changes = %d, want 2", len(summary.Changes))
	}
	if got := summary.Changes[0].Resource.Deposed; got != "" {
		t.Errorf("current object Deposed = %q, want empty", got)
	}
	if got := summary.Changes[1].Resource.Deposed; got != "00000001" {
		t.Errorf("deposed object Deposed = %q, want 00000001", got)
	}
}

func TestLoadPlan_GivenCurrentAndDeposedObjectAtSameAddress_ShouldKeepBothDistinguishedByDeposedKey(t *testing.T) {
	object := func(deposed, change string) string {
		return `{"address":"a.b","mode":"managed","type":"a","name":"b","provider_name":"registry.terraform.io/hashicorp/a"` + deposed + `,"change":` + change + `}`
	}
	current := object("", `{"actions":["update"],"before":{"x":1},"after":{"x":2},"after_unknown":{}}`)
	leftover := object(`,"deposed":"00000001"`, `{"actions":["delete"],"before":{"x":1},"after":null,"after_unknown":{}}`)

	summary, err := LoadPlan([]byte(policyPlan(current + "," + leftover)))
	if err != nil {
		t.Fatal(err)
	}

	if len(summary.Changes) != 2 {
		t.Fatalf("display changes = %d, want 2 (one per object)", len(summary.Changes))
	}
	if got := summary.Changes[0].Resource.Deposed; got != "" {
		t.Errorf("display change 0 Deposed = %q, want empty", got)
	}
	if got := summary.Changes[1].Resource.Deposed; got != "00000001" {
		t.Errorf("display change 1 Deposed = %q, want 00000001", got)
	}

	rows := summary.PolicyEvidence.Resources
	if summary.PolicyEvidence.ErrorCode != "" || len(rows) != 2 {
		t.Fatalf("evidence = %+v, want 2 rows and no error", summary.PolicyEvidence)
	}
	if got := rows[0].Deposed; got != "" {
		t.Errorf("evidence row 0 Deposed = %q, want empty", got)
	}
	if got := rows[1].Deposed; got != "00000001" {
		t.Errorf("evidence row 1 Deposed = %q, want 00000001", got)
	}

	for i, want := range []any{nil, "00000001"} {
		data, err := json.Marshal(rows[i].Input)
		if err != nil {
			t.Fatal(err)
		}
		var input struct {
			Resource map[string]any `json:"resource"`
		}
		if err := json.Unmarshal(data, &input); err != nil {
			t.Fatal(err)
		}
		got, present := input.Resource["deposed"]
		if !present {
			t.Errorf("row %d policy input resource has no deposed key: %s", i, data)
			continue
		}
		if got != want {
			t.Errorf("row %d policy input resource.deposed = %#v, want %#v", i, got, want)
		}
	}
}
