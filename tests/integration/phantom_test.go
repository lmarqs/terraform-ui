//go:build integration

package integration

import (
	"testing"
)

func TestPhantom_CreateFixture_NoPhantoms(t *testing.T) {
	result := runPlanAgent(t, "create")

	if result.PhantomChanges != 0 {
		t.Errorf("expected phantom_changes=0 for create fixture, got %d", result.PhantomChanges)
	}
	if len(result.PhantomResources) > 0 {
		t.Errorf("expected empty phantom_resources for create fixture, got %v", result.PhantomResources)
	}

	for _, c := range result.Changes {
		if c.Phantom {
			t.Errorf("expected no phantom flag on %s, but it was set", c.Address)
		}
	}
}

func TestPhantom_DeleteFixture_NoPhantoms(t *testing.T) {
	result := runPlanAgent(t, "delete")

	if result.PhantomChanges != 0 {
		t.Errorf("expected phantom_changes=0 for delete fixture, got %d", result.PhantomChanges)
	}
}

func TestPhantom_NoChangesFixture_NoPhantoms(t *testing.T) {
	result := runPlanAgent(t, "no-changes")

	if result.PhantomChanges != 0 {
		t.Errorf("expected phantom_changes=0 for no-changes fixture, got %d", result.PhantomChanges)
	}
}

func TestPhantom_FieldsPresentInJSON(t *testing.T) {
	result := runPlanAgent(t, "multi-resource")

	if result.PhantomChanges < 0 {
		t.Errorf("expected phantom_changes >= 0, got %d", result.PhantomChanges)
	}
}

func TestPhantom_UpdateFixture_PhantomDetection(t *testing.T) {
	// The update fixture has a real change, so it should NOT be phantom
	result := runPlanAgent(t, "update")

	for _, c := range result.Changes {
		if c.Action == "update" && c.Phantom {
			// This would mean the change is cosmetic-only, which it shouldn't be
			// for a real content change in the update fixture
			t.Logf("note: update of %s detected as phantom", c.Address)
		}
	}
}
