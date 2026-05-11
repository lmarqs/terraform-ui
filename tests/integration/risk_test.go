//go:build integration

package integration

import (
	"testing"
)

func TestRisk_CreateFixture_LowRisk(t *testing.T) {
	result := runPlanAgent(t, "create")

	// local_file resources should be low risk for create
	for _, c := range result.Changes {
		if c.Risk != "low" {
			t.Errorf("expected low risk for create of %s, got %q", c.Address, c.Risk)
		}
	}

	if result.Risk != "low" {
		t.Errorf("expected overall risk 'low' for create fixture, got %q", result.Risk)
	}
}

func TestRisk_DeleteFixture_MediumRisk(t *testing.T) {
	result := runPlanAgent(t, "delete")

	// Deleting an unmapped resource type (local_file) is medium risk
	for _, c := range result.Changes {
		if c.Action == "delete" && c.Risk != "medium" {
			t.Errorf("expected medium risk for delete of %s, got %q", c.Address, c.Risk)
		}
	}
}

func TestRisk_OverallReflectsHighest(t *testing.T) {
	// The delete fixture has a delete action, which should elevate overall risk
	result := runPlanAgent(t, "delete")

	if result.Risk == "low" || result.Risk == "none" {
		t.Errorf("expected overall risk > low for delete fixture, got %q", result.Risk)
	}
}

func TestRisk_ReplaceFixture_MediumRisk(t *testing.T) {
	result := runPlanAgent(t, "replace")

	// Replace (delete+create) of an unmapped resource type (local_file) is medium risk
	for _, c := range result.Changes {
		if c.Action == "delete-then-create" || c.Action == "create-then-delete" {
			if c.Risk != "medium" {
				t.Errorf("expected medium risk for replace of %s, got %q", c.Address, c.Risk)
			}
		}
	}
}

func TestRisk_UpdateFixture_LowRisk(t *testing.T) {
	result := runPlanAgent(t, "update")

	// Update of an unmapped resource type (terraform_data) is low risk
	for _, c := range result.Changes {
		if c.Action == "update" {
			if c.Risk != "low" {
				t.Errorf("expected low risk for update of %s, got %q", c.Address, c.Risk)
			}
		}
	}
}
