package sdk

import (
	"strings"
	"testing"
	"time"
)

func TestNewStalenessGuard_DefaultThreshold(t *testing.T) {
	g := NewStalenessGuard(0)
	if g.Threshold != 5*time.Minute {
		t.Fatalf("expected 5m default threshold, got %v", g.Threshold)
	}
}

func TestNewStalenessGuard_CustomThreshold(t *testing.T) {
	g := NewStalenessGuard(10 * time.Minute)
	if g.Threshold != 10*time.Minute {
		t.Fatalf("expected 10m threshold, got %v", g.Threshold)
	}
}

func TestCheckState_NilState(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	req := g.CheckState(nil)

	if req == nil {
		t.Fatal("expected non-nil InputRequest for nil state")
	}
	if req.Mode != InputRequestBool {
		t.Fatalf("expected InputRequestBool, got %d", req.Mode)
	}
	if !strings.Contains(req.Prompt, "No state loaded") {
		t.Fatalf("expected prompt to mention no state loaded, got %q", req.Prompt)
	}
}

func TestCheckState_FreshState(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	state := &TerraformState{
		LastRefreshed: time.Now().Add(-1 * time.Minute),
	}

	req := g.CheckState(state)
	if req != nil {
		t.Fatalf("expected nil for fresh state, got %+v", req)
	}
}

func TestCheckState_StaleState(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	state := &TerraformState{
		LastRefreshed: time.Now().Add(-10 * time.Minute),
	}

	req := g.CheckState(state)
	if req == nil {
		t.Fatal("expected non-nil InputRequest for stale state")
	}
	if req.Mode != InputRequestBool {
		t.Fatalf("expected InputRequestBool, got %d", req.Mode)
	}
	if !strings.Contains(req.Prompt, "State is") {
		t.Fatalf("expected prompt to mention 'State is', got %q", req.Prompt)
	}
	if !strings.Contains(req.Prompt, "old") {
		t.Fatalf("expected prompt to mention 'old', got %q", req.Prompt)
	}
	if !strings.Contains(req.Prompt, "threshold") {
		t.Fatalf("expected prompt to mention 'threshold', got %q", req.Prompt)
	}
}

func TestCheckPlan_NilPlan(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	req := g.CheckPlan(nil)

	if req == nil {
		t.Fatal("expected non-nil InputRequest for nil plan")
	}
	if req.Mode != InputRequestBool {
		t.Fatalf("expected InputRequestBool, got %d", req.Mode)
	}
	if !strings.Contains(req.Prompt, "No plan cached") {
		t.Fatalf("expected prompt to mention no plan cached, got %q", req.Prompt)
	}
}

func TestCheckPlan_FreshPlan(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	plan := &TerraformPlan{
		LastRefreshed: time.Now().Add(-2 * time.Minute),
	}

	req := g.CheckPlan(plan)
	if req != nil {
		t.Fatalf("expected nil for fresh plan, got %+v", req)
	}
}

func TestCheckPlan_StalePlan(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	plan := &TerraformPlan{
		LastRefreshed: time.Now().Add(-15 * time.Minute),
	}

	req := g.CheckPlan(plan)
	if req == nil {
		t.Fatal("expected non-nil InputRequest for stale plan")
	}
	if req.Mode != InputRequestBool {
		t.Fatalf("expected InputRequestBool, got %d", req.Mode)
	}
	if !strings.Contains(req.Prompt, "Plan is") {
		t.Fatalf("expected prompt to mention 'Plan is', got %q", req.Prompt)
	}
	if !strings.Contains(req.Prompt, "Re-run plan") {
		t.Fatalf("expected prompt to mention 'Re-run plan', got %q", req.Prompt)
	}
}

func TestIsStale_ZeroTime(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	if !g.IsStale(time.Time{}) {
		t.Fatal("expected zero time to be stale")
	}
}

func TestIsStale_FreshTime(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	if g.IsStale(time.Now().Add(-1 * time.Minute)) {
		t.Fatal("expected 1 minute ago to NOT be stale with 5m threshold")
	}
}

func TestIsStale_StaleTime(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	if !g.IsStale(time.Now().Add(-10 * time.Minute)) {
		t.Fatal("expected 10 minutes ago to be stale with 5m threshold")
	}
}

func TestIsStale_ExactBoundary(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	// At exactly the threshold boundary, time.Since > threshold should be false
	// (give a small buffer for test execution)
	justFresh := time.Now().Add(-4*time.Minute - 59*time.Second)
	if g.IsStale(justFresh) {
		t.Fatal("expected just-under-threshold to NOT be stale")
	}
}

func TestFormatDuration_ViaPrompt_Seconds(t *testing.T) {
	g := NewStalenessGuard(30 * time.Second)
	state := &TerraformState{
		LastRefreshed: time.Now().Add(-45 * time.Second),
	}

	req := g.CheckState(state)
	if req == nil {
		t.Fatal("expected stale state")
	}
	// The prompt should contain "45s" for the age and "30s" for the threshold
	if !strings.Contains(req.Prompt, "30s") {
		t.Fatalf("expected prompt to contain threshold '30s', got %q", req.Prompt)
	}
}

func TestFormatDuration_ViaPrompt_Minutes(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	state := &TerraformState{
		LastRefreshed: time.Now().Add(-7 * time.Minute),
	}

	req := g.CheckState(state)
	if req == nil {
		t.Fatal("expected stale state")
	}
	if !strings.Contains(req.Prompt, "7m") {
		t.Fatalf("expected prompt to contain '7m', got %q", req.Prompt)
	}
	if !strings.Contains(req.Prompt, "5m") {
		t.Fatalf("expected prompt to contain threshold '5m', got %q", req.Prompt)
	}
}

func TestFormatDuration_ViaPrompt_Hours(t *testing.T) {
	g := NewStalenessGuard(30 * time.Minute)
	state := &TerraformState{
		LastRefreshed: time.Now().Add(-2*time.Hour - 15*time.Minute),
	}

	req := g.CheckState(state)
	if req == nil {
		t.Fatal("expected stale state")
	}
	if !strings.Contains(req.Prompt, "2h15m") {
		t.Fatalf("expected prompt to contain '2h15m', got %q", req.Prompt)
	}
}

func TestCheckState_AtThresholdBoundary(t *testing.T) {
	g := NewStalenessGuard(5 * time.Minute)
	// Just under the threshold - should be fresh
	state := &TerraformState{
		LastRefreshed: time.Now().Add(-4*time.Minute - 58*time.Second),
	}
	req := g.CheckState(state)
	if req != nil {
		t.Fatalf("expected nil for state just under threshold, got %+v", req)
	}
}

func TestCheckPlan_CustomThreshold(t *testing.T) {
	g := NewStalenessGuard(1 * time.Minute)
	plan := &TerraformPlan{
		LastRefreshed: time.Now().Add(-2 * time.Minute),
	}

	req := g.CheckPlan(plan)
	if req == nil {
		t.Fatal("expected non-nil for plan older than 1m threshold")
	}
}
