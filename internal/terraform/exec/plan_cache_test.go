package exec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestPlan_WhenRereadAndSeededPlanFileChanged_ShouldReturnTheNewChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	writePlanFile(t, path, "null_resource.a")
	cache := terraform.NewServiceCache()
	if err := cache.SeedPlan(path, nil); err != nil {
		t.Fatal(err)
	}
	svc := NewExecService(t.TempDir(), "/nonexistent/terraform", cache)
	writePlanFile(t, path, "null_resource.a", "null_resource.b")

	got, err := svc.Plan(context.Background(), sdk.PlanOptions{Reread: true})

	if err != nil {
		t.Fatalf("Plan() error = %v, want the rewritten file served without terraform", err)
	}
	if len(got.Changes) != 2 {
		t.Errorf("Plan() returned %d changes, want the 2 in the rewritten file", len(got.Changes))
	}
}

func TestPlan_WhenRereadAndSeededPlanFileUnreadable_ShouldReturnTheError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	writePlanFile(t, path, "null_resource.a")
	cache := terraform.NewServiceCache()
	if err := cache.SeedPlan(path, nil); err != nil {
		t.Fatal(err)
	}
	svc := NewExecService(t.TempDir(), "/nonexistent/terraform", cache)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Plan(context.Background(), sdk.PlanOptions{Reread: true})

	if err == nil || !strings.Contains(err.Error(), "reading plan file") {
		t.Fatalf("Plan() error = %v, want the plan file read error, not a terraform run", err)
	}
}

func TestPlan_WhenNotRereadAndSeededPlanFileChanged_ShouldReturnTheCachedPlan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	writePlanFile(t, path, "null_resource.a")
	cache := terraform.NewServiceCache()
	if err := cache.SeedPlan(path, nil); err != nil {
		t.Fatal(err)
	}
	svc := NewExecService(t.TempDir(), "/nonexistent/terraform", cache)
	writePlanFile(t, path, "null_resource.a", "null_resource.b")

	got, err := svc.Plan(context.Background(), sdk.PlanOptions{})

	if err != nil || len(got.Changes) != 1 {
		t.Errorf("Plan() = %v changes, err %v; want the 1 cached change", got, err)
	}
}

func TestPlanSnapshot_ShouldBeTrueOnlyForAStdinPlan(t *testing.T) {
	fromStdin := terraform.NewServiceCache()
	if err := fromStdin.SeedPlan("", []byte(`{"format_version":"1.0","resource_changes":[]}`)); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		cache *terraform.ServiceCache
		want  bool
	}{
		{"Stdin", fromStdin, true},
		{"Empty", terraform.NewServiceCache(), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewExecService(t.TempDir(), "/nonexistent/terraform", tc.cache)
			if got := svc.PlanSnapshot(); got != tc.want {
				t.Errorf("PlanSnapshot() = %v, want %v", got, tc.want)
			}
		})
	}
}

func writePlanFile(t *testing.T, path string, addresses ...string) {
	t.Helper()
	var changes []string
	for _, addr := range addresses {
		changes = append(changes, fmt.Sprintf(`{"address": %q, "type": "null_resource", "name": "n",
		  "provider_name": "registry.terraform.io/hashicorp/null",
		  "change": {"actions": ["create"], "before": null, "after": {}, "after_unknown": {}}}`, addr))
	}
	doc := fmt.Sprintf(`{"format_version": "1.0", "terraform_version": "1.5.0", "planned_values": {}, "resource_changes": [%s]}`, strings.Join(changes, ","))
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}
