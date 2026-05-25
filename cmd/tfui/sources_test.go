package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/terraform"
)

func TestPreseedSources_Empty_WhenBothBlank_ShouldReturnTrue(t *testing.T) {
	s := PreseedSources{}
	if !s.Empty() {
		t.Error("expected Empty() = true for zero value")
	}
}

func TestPreseedSources_Empty_WhenPlanSet_ShouldReturnFalse(t *testing.T) {
	s := PreseedSources{Plan: "./plan.json"}
	if s.Empty() {
		t.Error("expected Empty() = false when Plan is set")
	}
}

func TestPreseedSources_Empty_WhenStateSet_ShouldReturnFalse(t *testing.T) {
	s := PreseedSources{State: "./state.json"}
	if s.Empty() {
		t.Error("expected Empty() = false when State is set")
	}
}

func TestPreseedSources_Seed_WhenBothStdin_ShouldReturnError(t *testing.T) {
	s := PreseedSources{Plan: "-", State: "-"}
	cache := terraform.NewServiceCache()
	err := s.Seed(cache)
	if err == nil {
		t.Fatal("expected error for dual stdin")
	}
	if got := err.Error(); got != "stdin (-) can only be used by one flag per invocation; use a file for the other" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestPreseedSources_Seed_WhenPlanFile_ShouldSeedCache(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(minimalPlanJSON), 0644); err != nil {
		t.Fatal(err)
	}

	s := PreseedSources{Plan: planFile}
	cache := terraform.NewServiceCache()
	if err := s.Seed(cache); err != nil {
		t.Fatalf("Seed error: %v", err)
	}
}

func TestPreseedSources_Seed_WhenStateFile_ShouldSeedCache(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0644); err != nil {
		t.Fatal(err)
	}

	s := PreseedSources{State: stateFile}
	cache := terraform.NewServiceCache()
	if err := s.Seed(cache); err != nil {
		t.Fatalf("Seed error: %v", err)
	}
}

func TestPreseedSources_Seed_WhenPlanFileMissing_ShouldReturnError(t *testing.T) {
	s := PreseedSources{Plan: "/nonexistent/plan.json"}
	cache := terraform.NewServiceCache()
	err := s.Seed(cache)
	if err == nil {
		t.Fatal("expected error for missing plan file")
	}
}

func TestResolveToAbsPath_WhenAbsolute_ShouldReturnUnchanged(t *testing.T) {
	got, err := resolveToAbsPath("/base", "/abs/path")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/abs/path" {
		t.Errorf("got %q, want /abs/path", got)
	}
}

func TestResolveToAbsPath_WhenRelative_ShouldJoinWithBase(t *testing.T) {
	got, err := resolveToAbsPath("/base", "./rel/path")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/base/rel/path" {
		t.Errorf("got %q, want /base/rel/path", got)
	}
}

func TestResolveToAbsPath_WhenFileSchemeAbsolute_ShouldReturnPath(t *testing.T) {
	got, err := resolveToAbsPath("/base", "file:///abs/path")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/abs/path" {
		t.Errorf("got %q, want /abs/path", got)
	}
}

func TestResolveToAbsPath_WhenFileSchemeRelative_ShouldJoinWithBase(t *testing.T) {
	got, err := resolveToAbsPath("/base", "file://rel/path")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/base/rel/path" {
		t.Errorf("got %q, want /base/rel/path", got)
	}
}

const minimalPlanJSON = `{
  "format_version": "1.0",
  "terraform_version": "1.5.0",
  "planned_values": {},
  "resource_changes": [
    {
      "address": "null_resource.test",
      "type": "null_resource",
      "name": "test",
      "provider_name": "registry.terraform.io/hashicorp/null",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {},
        "after_unknown": {},
        "before_sensitive": false,
        "after_sensitive": false
      }
    }
  ]
}`

const minimalStateJSON = `{
  "format_version": "1.0",
  "values": {
    "root_module": {
      "resources": [
        {
          "address": "null_resource.test",
          "type": "null_resource",
          "name": "test",
          "provider_name": "registry.terraform.io/hashicorp/null",
          "values": {}
        }
      ]
    }
  }
}`
