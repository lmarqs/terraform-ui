package exec

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func planFixturePath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file location")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "tests", "fixtures", "plan.json")
}

func TestPlan_WhenCacheHoldsSeededPlan_ShouldReturnItWithoutTerraform(t *testing.T) {
	cache := terraform.NewServiceCache()
	if err := cache.SeedPlan(planFixturePath(t), nil); err != nil {
		t.Fatalf("seeding plan: %v", err)
	}
	seeded, ok := cache.GetPlan()
	if !ok || len(seeded.Changes) == 0 {
		t.Fatalf("fixture plan not cached: ok=%v", ok)
	}
	svc := NewExecService(t.TempDir(), "/nonexistent/terraform", cache)

	got, err := svc.Plan(context.Background(), sdk.PlanOptions{})

	if err != nil {
		t.Fatalf("Plan() error = %v, want the seeded plan served from the cache", err)
	}
	if got != seeded {
		t.Errorf("Plan() returned %p with %d changes, want the cached plan %p with %d changes",
			got, len(got.Changes), seeded, len(seeded.Changes))
	}
}

func TestPlan_WhenCacheHoldsNoPlan_ShouldRunTerraform(t *testing.T) {
	svc := NewExecService(t.TempDir(), "/nonexistent/terraform", terraform.NewServiceCache())

	_, err := svc.Plan(context.Background(), sdk.PlanOptions{})

	if err == nil {
		t.Fatal("Plan() error = nil, want the missing terraform binary to surface")
	}
}

func TestPlan_GivenNewContext_ShouldNotReuseSeededPlan(t *testing.T) {
	cache := terraform.NewServiceCache()
	if err := cache.SeedPlan(planFixturePath(t), nil); err != nil {
		t.Fatal(err)
	}
	svc := NewExecService(t.TempDir(), "/nonexistent/terraform", cache)
	for _, dir := range []string{t.TempDir(), svc.workingDir} {
		scoped := svc.WithDir(dir)
		if _, err := scoped.Plan(context.Background(), sdk.PlanOptions{}); err == nil {
			t.Fatalf("new Context at %q reused the previous Context's seeded plan", dir)
		}
	}
	if _, err := svc.Plan(context.Background(), sdk.PlanOptions{}); err != nil {
		t.Fatalf("scoping a service discarded the initial Context's seed: %v", err)
	}
}
