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

func TestRisk_DeleteFixture_HighRisk(t *testing.T) {
	result := runPlanAgent(t, "delete")

	// Deleting a local_file should be high risk (delete of non-critical resource)
	for _, c := range result.Changes {
		if c.Action == "delete" && c.Risk != "high" {
			t.Errorf("expected high risk for delete of %s, got %q", c.Address, c.Risk)
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

func TestRisk_ReplaceFixture_HighRisk(t *testing.T) {
	result := runPlanAgent(t, "replace")

	// Replace (delete+create) should be high risk for non-critical resources
	for _, c := range result.Changes {
		if c.Action == "delete-then-create" || c.Action == "create-then-delete" {
			if c.Risk != "high" {
				t.Errorf("expected high risk for replace of %s, got %q", c.Address, c.Risk)
			}
		}
	}
}

func TestRisk_UpdateFixture_MediumRisk(t *testing.T) {
	result := runPlanAgent(t, "update")

	// Update of a non-critical, non-high-risk resource should be medium
	for _, c := range result.Changes {
		if c.Action == "update" {
			if c.Risk != "medium" {
				t.Errorf("expected medium risk for update of %s, got %q", c.Address, c.Risk)
			}
		}
	}
}
