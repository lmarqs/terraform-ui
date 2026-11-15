package terraform

import (
	"encoding/json"
	"strings"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestParsePlan_GivenUnknownAndSensitiveEvidence_ShouldPreserveReviewNotation(t *testing.T) {
	for _, tc := range []struct{ name, before, after, unknown, sensitive string }{
		{"top-level", `{"version":1,"password":"before-secret"}`, `{"version":null,"password":"after-secret"}`, `{"version":true}`, `{"password":true}`},
		{"unknown-only-key", `{}`, `{}`, `{"version":true}`, `{}`},
		{"nested-object", `{"settings":{"version":1},"credentials":{"password":"before-secret"}}`, `{"settings":{"version":null},"credentials":{"password":"after-secret"}}`, `{"settings":{"version":true}}`, `{"credentials":{"password":true}}`},
		{"nested-array", `{"settings":[{"version":1}],"credentials":[{"password":"before-secret"}]}`, `{"settings":[{"version":null}],"credentials":[{"password":"after-secret"}]}`, `{"settings":[{"version":true}]}`, `{"credentials":[{"password":true}]}`},
		{"null-becomes-unknown", `{"version":null}`, `{"version":null}`, `{"version":true}`, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `{"format_version":"1.2","resource_changes":[{"address":"aws_instance.web","mode":"managed","type":"aws_instance","name":"web","change":{"actions":["update"],"before":` + tc.before + `,"after":` + tc.after + `,"after_unknown":` + tc.unknown + `,"before_sensitive":` + tc.sensitive + `,"after_sensitive":` + tc.sensitive + `}}]}`
			var plan tfjson.Plan
			if err := json.Unmarshal([]byte(source), &plan); err != nil {
				t.Fatal(err)
			}
			summary := ParsePlan(&plan)
			if len(summary.Changes) != 1 {
				t.Fatalf("changes = %d, want 1", len(summary.Changes))
			}
			view := sdk.RenderAttributeDiffs(summary.Changes[0].AttributeDiffs)
			if !strings.Contains(view, "(known after apply)") {
				t.Errorf("review must distinguish unknown from null:\n%s", view)
			}
			if strings.Contains(tc.before, "secret") && !strings.Contains(view, "(sensitive)") {
				t.Errorf("review lost nested sensitivity:\n%s", view)
			}
			if strings.Contains(view, "before-secret") || strings.Contains(view, "after-secret") {
				t.Errorf("review exposed a sensitive nested value:\n%s", view)
			}
		})
	}
}

func TestParsePlan_GivenReplacementPaths_ShouldMarkAttributesForcingReplacement(t *testing.T) {
	var plan tfjson.Plan
	source := `{"format_version":"1.2","resource_changes":[{"address":"aws_instance.web","mode":"managed","type":"aws_instance","name":"web","change":{"actions":["create","delete"],"before":{"ami":"old","settings":{"size":1}},"after":{"ami":"new","settings":{"size":2}},"replace_paths":[["ami"],["settings","size"]]}}]}`
	if err := json.Unmarshal([]byte(source), &plan); err != nil {
		t.Fatal(err)
	}
	summary := ParsePlan(&plan)
	if len(summary.Changes) != 1 {
		t.Fatalf("changes = %d, want 1", len(summary.Changes))
	}
	if summary.Changes[0].Action != sdk.ActionCreateThenDelete {
		t.Errorf("action = %s, want create-then-delete", summary.Changes[0].Action)
	}
	for _, diff := range summary.Changes[0].AttributeDiffs {
		if !diff.ForcesNew {
			t.Errorf("replacement path %s must show forces new", diff.Key)
		}
	}
}

func TestParsePlan_GivenExplicitNullWithoutUnknownMask_ShouldRetainNull(t *testing.T) {
	var plan tfjson.Plan
	source := `{"format_version":"1.2","resource_changes":[{"address":"aws_instance.web","mode":"managed","type":"aws_instance","name":"web","change":{"actions":["update"],"before":{"version":1},"after":{"version":null},"after_unknown":{"version":false}}}]}`
	if err := json.Unmarshal([]byte(source), &plan); err != nil {
		t.Fatal(err)
	}
	summary := ParsePlan(&plan)
	if len(summary.Changes) != 1 || len(summary.Changes[0].AttributeDiffs) != 1 {
		t.Fatalf("expected one changed attribute, got %+v", summary.Changes)
	}
	diff := summary.Changes[0].AttributeDiffs[0]
	if diff.NewValue != "null" {
		t.Errorf("known null = %q, want null", diff.NewValue)
	}
}

func TestParsePlan_GivenReplacementMetadataWithoutAttributePath_ShouldKeepDiffWithoutReplacementAnnotation(t *testing.T) {
	var plan tfjson.Plan
	source := `{"format_version":"1.2","resource_changes":[{"address":"aws_instance.web","mode":"managed","type":"aws_instance","name":"web","change":{"actions":["update"],"before":{"ami":"old"},"after":{"ami":"new"},"replace_paths":[null,[],["other"]]}}]}`
	if err := json.Unmarshal([]byte(source), &plan); err != nil {
		t.Fatal(err)
	}
	summary := ParsePlan(&plan)
	if len(summary.Changes) != 1 || len(summary.Changes[0].AttributeDiffs) != 1 {
		t.Fatalf("expected unchanged diff availability, got %+v", summary.Changes)
	}
	diff := summary.Changes[0].AttributeDiffs[0]
	if diff.ForcesNew || diff.Key != "ami" || diff.NewValue != `"new"` {
		t.Errorf("nonmatching metadata changed visible diff: %+v", diff)
	}
}
