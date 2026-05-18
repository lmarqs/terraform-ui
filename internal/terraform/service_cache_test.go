package terraform

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const minimalOutputsJSON = `[
  {"Name": "endpoint", "Value": "https://api.example.com", "Type": "string", "Sensitive": false},
  {"Name": "secret", "Value": "***", "Type": "string", "Sensitive": true}
]`

const minimalDiagnosticsJSON = `[
  {"Severity": "warning", "Summary": "Deprecated attribute", "Detail": "Use engine_type instead", "File": "main.tf", "Line": 42}
]`

const minimalWorkspacesJSON = `["default", "staging", "production"]`

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
          "values": {},
          "sensitive_values": {}
        }
      ]
    }
  }
}`

func TestNewServiceCache_WhenCreated_ShouldReturnEmptyCache(t *testing.T) {
	c := NewServiceCache()
	if c == nil {
		t.Fatal("NewServiceCache() returned nil")
	}

	t.Run("ShouldReturnNilForGetPlan", func(t *testing.T) {
		plan, ok := c.GetPlan()
		if ok {
			t.Error("GetPlan() ok = true, want false")
		}
		if plan != nil {
			t.Errorf("GetPlan() = %v, want nil", plan)
		}
	})

	t.Run("ShouldReturnNilForGetResources", func(t *testing.T) {
		resources, ok := c.GetResources()
		if ok {
			t.Error("GetResources() ok = true, want false")
		}
		if resources != nil {
			t.Errorf("GetResources() = %v, want nil", resources)
		}
	})

	t.Run("ShouldReturnNilForGetState", func(t *testing.T) {
		state, ok := c.GetState()
		if ok {
			t.Error("GetState() ok = true, want false")
		}
		if state != nil {
			t.Errorf("GetState() = %v, want nil", state)
		}
	})

	t.Run("ShouldReturnNilForGetOutputs", func(t *testing.T) {
		outputs, ok := c.GetOutputs()
		if ok {
			t.Error("GetOutputs() ok = true, want false")
		}
		if outputs != nil {
			t.Errorf("GetOutputs() = %v, want nil", outputs)
		}
	})

	t.Run("ShouldReturnNilForGetDiagnostics", func(t *testing.T) {
		diagnostics, ok := c.GetDiagnostics()
		if ok {
			t.Error("GetDiagnostics() ok = true, want false")
		}
		if diagnostics != nil {
			t.Errorf("GetDiagnostics() = %v, want nil", diagnostics)
		}
	})

	t.Run("ShouldReturnNilForGetWorkspaces", func(t *testing.T) {
		workspaces, ok := c.GetWorkspaces()
		if ok {
			t.Error("GetWorkspaces() ok = true, want false")
		}
		if workspaces != nil {
			t.Errorf("GetWorkspaces() = %v, want nil", workspaces)
		}
	})
}

func TestServiceCache_WhenSeedPlanFromFile_ShouldCachePlan(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(minimalPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedPlan(planFile, nil)
	if err != nil {
		t.Fatalf("SeedPlan() error = %v", err)
	}

	t.Run("ShouldReturnCachedPlan", func(t *testing.T) {
		plan, ok := c.GetPlan()
		if !ok {
			t.Fatal("GetPlan() ok = false, want true")
		}
		if plan == nil {
			t.Fatal("GetPlan() returned nil plan")
		}
		if plan.ToCreate != 1 {
			t.Errorf("plan.ToCreate = %d, want 1", plan.ToCreate)
		}
	})

	t.Run("ShouldSetSourceFile", func(t *testing.T) {
		if c.planSource.kind != sourceFile {
			t.Errorf("planSource.kind = %v, want sourceFile", c.planSource.kind)
		}
		if c.planSource.filePath != planFile {
			t.Errorf("planSource.filePath = %q, want %q", c.planSource.filePath, planFile)
		}
	})
}

func TestServiceCache_WhenSeedPlanFromBytes_ShouldCachePlan(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedPlan("", []byte(minimalPlanJSON))
	if err != nil {
		t.Fatalf("SeedPlan() error = %v", err)
	}

	t.Run("ShouldReturnCachedPlan", func(t *testing.T) {
		plan, ok := c.GetPlan()
		if !ok {
			t.Fatal("GetPlan() ok = false, want true")
		}
		if plan == nil {
			t.Fatal("GetPlan() returned nil plan")
		}
		if plan.ToCreate != 1 {
			t.Errorf("plan.ToCreate = %d, want 1", plan.ToCreate)
		}
	})

	t.Run("ShouldSetSourceStdin", func(t *testing.T) {
		if c.planSource.kind != sourceStdin {
			t.Errorf("planSource.kind = %v, want sourceStdin", c.planSource.kind)
		}
		if c.planSource.data == nil {
			t.Error("planSource.data = nil, want non-nil")
		}
	})
}

func TestServiceCache_WhenSeedPlanWithInvalidJSON_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedPlan("", []byte(`not json`))
	if err == nil {
		t.Error("SeedPlan() with invalid JSON: want error")
	}

	plan, ok := c.GetPlan()
	if ok {
		t.Error("GetPlan() after failed seed: ok = true, want false")
	}
	if plan != nil {
		t.Errorf("GetPlan() after failed seed = %v, want nil", plan)
	}
}

func TestServiceCache_WhenSeedPlanFromNonexistentFile_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedPlan("/nonexistent/path/plan.json", nil)
	if err == nil {
		t.Error("SeedPlan() with missing file: want error")
	}
}

func TestServiceCache_WhenSeedStateFromFile_ShouldCacheState(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedState(stateFile, nil)
	if err != nil {
		t.Fatalf("SeedState() error = %v", err)
	}

	t.Run("ShouldReturnCachedResources", func(t *testing.T) {
		resources, ok := c.GetResources()
		if !ok {
			t.Fatal("GetResources() ok = false, want true")
		}
		if len(resources) != 1 {
			t.Fatalf("len(resources) = %d, want 1", len(resources))
		}
		if resources[0].Address != "null_resource.test" {
			t.Errorf("resources[0].Address = %q, want %q", resources[0].Address, "null_resource.test")
		}
	})

	t.Run("ShouldReturnCachedState", func(t *testing.T) {
		state, ok := c.GetState()
		if !ok {
			t.Fatal("GetState() ok = false, want true")
		}
		if state == nil {
			t.Fatal("GetState() returned nil")
		}
		if state.Values == nil {
			t.Fatal("state.Values = nil, want non-nil")
		}
	})

	t.Run("ShouldSetSourceFile", func(t *testing.T) {
		if c.stateSource.kind != sourceFile {
			t.Errorf("stateSource.kind = %v, want sourceFile", c.stateSource.kind)
		}
		if c.stateSource.filePath != stateFile {
			t.Errorf("stateSource.filePath = %q, want %q", c.stateSource.filePath, stateFile)
		}
	})
}

func TestServiceCache_WhenSeedStateFromBytes_ShouldCacheState(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedState("", []byte(minimalStateJSON))
	if err != nil {
		t.Fatalf("SeedState() error = %v", err)
	}

	t.Run("ShouldReturnCachedResources", func(t *testing.T) {
		resources, ok := c.GetResources()
		if !ok {
			t.Fatal("GetResources() ok = false, want true")
		}
		if len(resources) != 1 {
			t.Fatalf("len(resources) = %d, want 1", len(resources))
		}
	})

	t.Run("ShouldReturnCachedState", func(t *testing.T) {
		state, ok := c.GetState()
		if !ok {
			t.Fatal("GetState() ok = false, want true")
		}
		if state == nil {
			t.Fatal("GetState() returned nil")
		}
	})

	t.Run("ShouldSetSourceStdin", func(t *testing.T) {
		if c.stateSource.kind != sourceStdin {
			t.Errorf("stateSource.kind = %v, want sourceStdin", c.stateSource.kind)
		}
		if c.stateSource.data == nil {
			t.Error("stateSource.data = nil, want non-nil")
		}
	})
}

func TestServiceCache_WhenSeedStateWithInvalidJSON_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedState("", []byte(`{broken`))
	if err == nil {
		t.Error("SeedState() with invalid JSON: want error")
	}

	resources, ok := c.GetResources()
	if ok {
		t.Error("GetResources() after failed seed: ok = true, want false")
	}
	if resources != nil {
		t.Errorf("GetResources() after failed seed = %v, want nil", resources)
	}
}

func TestServiceCache_WhenSeedStateFromNonexistentFile_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedState("/nonexistent/path/state.json", nil)
	if err == nil {
		t.Error("SeedState() with missing file: want error")
	}
}

func TestServiceCache_WhenSetPlan_ShouldStoreExecSourcedPlan(t *testing.T) {
	c := NewServiceCache()
	plan := &sdk.PlanSummary{
		ToCreate: 3,
		ToUpdate: 1,
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
	}

	c.SetPlan(plan)

	t.Run("ShouldReturnStoredPlan", func(t *testing.T) {
		got, ok := c.GetPlan()
		if !ok {
			t.Fatal("GetPlan() ok = false, want true")
		}
		if got.ToCreate != 3 {
			t.Errorf("plan.ToCreate = %d, want 3", got.ToCreate)
		}
		if got.ToUpdate != 1 {
			t.Errorf("plan.ToUpdate = %d, want 1", got.ToUpdate)
		}
	})

	t.Run("ShouldSetSourceExec", func(t *testing.T) {
		if c.planSource.kind != sourceExec {
			t.Errorf("planSource.kind = %v, want sourceExec", c.planSource.kind)
		}
	})
}

func TestServiceCache_WhenSetState_ShouldStoreExecSourcedState(t *testing.T) {
	c := NewServiceCache()
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Name: "web"},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket", Name: "data"},
	}
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values:        &tfjson.StateValues{},
	}

	c.SetState(resources, state)

	t.Run("ShouldReturnStoredResources", func(t *testing.T) {
		got, ok := c.GetResources()
		if !ok {
			t.Fatal("GetResources() ok = false, want true")
		}
		if len(got) != 2 {
			t.Fatalf("len(resources) = %d, want 2", len(got))
		}
		if got[0].Address != "aws_instance.web" {
			t.Errorf("resources[0].Address = %q, want %q", got[0].Address, "aws_instance.web")
		}
	})

	t.Run("ShouldReturnStoredState", func(t *testing.T) {
		got, ok := c.GetState()
		if !ok {
			t.Fatal("GetState() ok = false, want true")
		}
		if got == nil {
			t.Fatal("GetState() returned nil")
		}
	})

	t.Run("ShouldSetSourceExec", func(t *testing.T) {
		if c.stateSource.kind != sourceExec {
			t.Errorf("stateSource.kind = %v, want sourceExec", c.stateSource.kind)
		}
	})
}

func TestServiceCache_WhenInvalidateAll_ShouldClearExecSourcedData(t *testing.T) {
	c := NewServiceCache()
	c.SetPlan(&sdk.PlanSummary{ToCreate: 1})
	c.SetState([]sdk.Resource{{Address: "a"}}, &tfjson.State{})

	c.InvalidateAll()

	t.Run("ShouldClearExecPlan", func(t *testing.T) {
		plan, ok := c.GetPlan()
		if ok {
			t.Error("GetPlan() ok = true after invalidation of exec-sourced plan")
		}
		if plan != nil {
			t.Errorf("GetPlan() = %v, want nil", plan)
		}
	})

	t.Run("ShouldClearExecState", func(t *testing.T) {
		resources, ok := c.GetResources()
		if ok {
			t.Error("GetResources() ok = true after invalidation of exec-sourced state")
		}
		if resources != nil {
			t.Errorf("GetResources() = %v, want nil", resources)
		}
	})
}

func TestServiceCache_WhenInvalidateAll_ShouldPreserveStdinSourcedData(t *testing.T) {
	c := NewServiceCache()
	if err := c.SeedPlan("", []byte(minimalPlanJSON)); err != nil {
		t.Fatal(err)
	}
	if err := c.SeedState("", []byte(minimalStateJSON)); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	t.Run("ShouldPreserveStdinPlan", func(t *testing.T) {
		plan, ok := c.GetPlan()
		if !ok {
			t.Fatal("GetPlan() ok = false, want true (stdin data preserved)")
		}
		if plan == nil {
			t.Fatal("GetPlan() = nil, want non-nil")
		}
		if plan.ToCreate != 1 {
			t.Errorf("plan.ToCreate = %d, want 1", plan.ToCreate)
		}
	})

	t.Run("ShouldPreserveStdinState", func(t *testing.T) {
		resources, ok := c.GetResources()
		if !ok {
			t.Fatal("GetResources() ok = false, want true (stdin data preserved)")
		}
		if len(resources) != 1 {
			t.Errorf("len(resources) = %d, want 1", len(resources))
		}
	})
}

func TestServiceCache_WhenInvalidateAll_ShouldReReadFileSourcedData(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	stateFile := filepath.Join(dir, "state.json")

	if err := os.WriteFile(planFile, []byte(minimalPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedPlan(planFile, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.SeedState(stateFile, nil); err != nil {
		t.Fatal(err)
	}

	updatedPlanJSON := `{
  "format_version": "1.0",
  "terraform_version": "1.5.0",
  "planned_values": {},
  "resource_changes": [
    {
      "address": "null_resource.a",
      "type": "null_resource",
      "name": "a",
      "provider_name": "registry.terraform.io/hashicorp/null",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {},
        "after_unknown": {},
        "before_sensitive": false,
        "after_sensitive": false
      }
    },
    {
      "address": "null_resource.b",
      "type": "null_resource",
      "name": "b",
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

	updatedStateJSON := `{
  "format_version": "1.0",
  "values": {
    "root_module": {
      "resources": [
        {
          "address": "null_resource.a",
          "type": "null_resource",
          "name": "a",
          "provider_name": "registry.terraform.io/hashicorp/null",
          "values": {},
          "sensitive_values": {}
        },
        {
          "address": "null_resource.b",
          "type": "null_resource",
          "name": "b",
          "provider_name": "registry.terraform.io/hashicorp/null",
          "values": {},
          "sensitive_values": {}
        }
      ]
    }
  }
}`

	if err := os.WriteFile(planFile, []byte(updatedPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, []byte(updatedStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	t.Run("ShouldReReadPlanFromFile", func(t *testing.T) {
		plan, ok := c.GetPlan()
		if !ok {
			t.Fatal("GetPlan() ok = false after re-read")
		}
		if plan.ToCreate != 2 {
			t.Errorf("plan.ToCreate = %d, want 2 (re-read from updated file)", plan.ToCreate)
		}
	})

	t.Run("ShouldReReadStateFromFile", func(t *testing.T) {
		resources, ok := c.GetResources()
		if !ok {
			t.Fatal("GetResources() ok = false after re-read")
		}
		if len(resources) != 2 {
			t.Errorf("len(resources) = %d, want 2 (re-read from updated file)", len(resources))
		}
	})
}

func TestServiceCache_WhenInvalidateState_ShouldOnlyAffectState(t *testing.T) {
	c := NewServiceCache()
	c.SetPlan(&sdk.PlanSummary{ToCreate: 5})
	c.SetState([]sdk.Resource{{Address: "a"}}, &tfjson.State{})

	c.InvalidateState()

	t.Run("ShouldClearExecState", func(t *testing.T) {
		resources, ok := c.GetResources()
		if ok {
			t.Error("GetResources() ok = true after InvalidateState")
		}
		if resources != nil {
			t.Errorf("GetResources() = %v, want nil", resources)
		}
	})

	t.Run("ShouldPreservePlan", func(t *testing.T) {
		plan, ok := c.GetPlan()
		if !ok {
			t.Fatal("GetPlan() ok = false, want true (plan should be preserved)")
		}
		if plan.ToCreate != 5 {
			t.Errorf("plan.ToCreate = %d, want 5", plan.ToCreate)
		}
	})
}

func TestServiceCache_WhenInvalidateState_ShouldPreserveStdinState(t *testing.T) {
	c := NewServiceCache()
	if err := c.SeedState("", []byte(minimalStateJSON)); err != nil {
		t.Fatal(err)
	}

	c.InvalidateState()

	resources, ok := c.GetResources()
	if !ok {
		t.Fatal("GetResources() ok = false, want true (stdin-sourced state preserved)")
	}
	if len(resources) != 1 {
		t.Errorf("len(resources) = %d, want 1", len(resources))
	}
}

func TestServiceCache_WhenInvalidateState_ShouldReReadFileState(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedState(stateFile, nil); err != nil {
		t.Fatal(err)
	}

	updatedStateJSON := `{
  "format_version": "1.0",
  "values": {
    "root_module": {
      "resources": [
        {
          "address": "null_resource.a",
          "type": "null_resource",
          "name": "a",
          "provider_name": "registry.terraform.io/hashicorp/null",
          "values": {},
          "sensitive_values": {}
        },
        {
          "address": "null_resource.b",
          "type": "null_resource",
          "name": "b",
          "provider_name": "registry.terraform.io/hashicorp/null",
          "values": {},
          "sensitive_values": {}
        },
        {
          "address": "null_resource.c",
          "type": "null_resource",
          "name": "c",
          "provider_name": "registry.terraform.io/hashicorp/null",
          "values": {},
          "sensitive_values": {}
        }
      ]
    }
  }
}`
	if err := os.WriteFile(stateFile, []byte(updatedStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateState()

	resources, ok := c.GetResources()
	if !ok {
		t.Fatal("GetResources() ok = false after re-read")
	}
	if len(resources) != 3 {
		t.Errorf("len(resources) = %d, want 3 (re-read from updated file)", len(resources))
	}
}

func TestServiceCache_WhenClear_ShouldWipeEverything(t *testing.T) {
	c := NewServiceCache()

	if err := c.SeedPlan("", []byte(minimalPlanJSON)); err != nil {
		t.Fatal(err)
	}
	if err := c.SeedState("", []byte(minimalStateJSON)); err != nil {
		t.Fatal(err)
	}

	c.Clear()

	t.Run("ShouldClearStdinPlan", func(t *testing.T) {
		plan, ok := c.GetPlan()
		if ok {
			t.Error("GetPlan() ok = true after Clear")
		}
		if plan != nil {
			t.Errorf("GetPlan() = %v, want nil", plan)
		}
	})

	t.Run("ShouldClearStdinResources", func(t *testing.T) {
		resources, ok := c.GetResources()
		if ok {
			t.Error("GetResources() ok = true after Clear")
		}
		if resources != nil {
			t.Errorf("GetResources() = %v, want nil", resources)
		}
	})

	t.Run("ShouldClearStdinState", func(t *testing.T) {
		state, ok := c.GetState()
		if ok {
			t.Error("GetState() ok = true after Clear")
		}
		if state != nil {
			t.Errorf("GetState() = %v, want nil", state)
		}
	})
}

func TestServiceCache_WhenClear_ShouldWipeFileSourcedData(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(planFile, []byte(minimalPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedPlan(planFile, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.SeedState(stateFile, nil); err != nil {
		t.Fatal(err)
	}

	c.Clear()

	t.Run("ShouldClearFilePlan", func(t *testing.T) {
		plan, ok := c.GetPlan()
		if ok {
			t.Error("GetPlan() ok = true after Clear")
		}
		if plan != nil {
			t.Errorf("GetPlan() = %v, want nil", plan)
		}
	})

	t.Run("ShouldClearFileState", func(t *testing.T) {
		resources, ok := c.GetResources()
		if ok {
			t.Error("GetResources() ok = true after Clear")
		}
		if resources != nil {
			t.Errorf("GetResources() = %v, want nil", resources)
		}
	})
}

func TestServiceCache_WhenClear_ShouldWipeExecSourcedData(t *testing.T) {
	c := NewServiceCache()
	c.SetPlan(&sdk.PlanSummary{ToCreate: 1})
	c.SetState([]sdk.Resource{{Address: "a"}}, &tfjson.State{})

	c.Clear()

	plan, ok := c.GetPlan()
	if ok || plan != nil {
		t.Error("GetPlan() should be (nil, false) after Clear")
	}

	resources, rok := c.GetResources()
	if rok || resources != nil {
		t.Error("GetResources() should be (nil, false) after Clear")
	}
}

func TestServiceCache_WhenConcurrentAccess_ShouldNotPanic(t *testing.T) {
	c := NewServiceCache()
	if err := c.SeedPlan("", []byte(minimalPlanJSON)); err != nil {
		t.Fatal(err)
	}
	if err := c.SeedState("", []byte(minimalStateJSON)); err != nil {
		t.Fatal(err)
	}
	if err := c.SeedOutputs("", []byte(minimalOutputsJSON)); err != nil {
		t.Fatal(err)
	}
	if err := c.SeedDiagnostics("", []byte(minimalDiagnosticsJSON)); err != nil {
		t.Fatal(err)
	}
	if err := c.SeedWorkspaces("", []byte(minimalWorkspacesJSON)); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	const goroutines = 50

	wg.Add(goroutines * 12)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			c.GetPlan()
		}()
		go func() {
			defer wg.Done()
			c.GetResources()
		}()
		go func() {
			defer wg.Done()
			c.GetState()
		}()
		go func() {
			defer wg.Done()
			c.GetOutputs()
		}()
		go func() {
			defer wg.Done()
			c.GetDiagnostics()
		}()
		go func() {
			defer wg.Done()
			c.GetWorkspaces()
		}()
		go func() {
			defer wg.Done()
			c.SetPlan(&sdk.PlanSummary{ToCreate: 1})
		}()
		go func() {
			defer wg.Done()
			c.SetState([]sdk.Resource{{Address: "x"}}, &tfjson.State{})
		}()
		go func() {
			defer wg.Done()
			c.SetOutputs(map[string]sdk.OutputValue{"x": {Name: "x"}})
		}()
		go func() {
			defer wg.Done()
			c.SetDiagnostics([]sdk.Diagnostic{{Severity: "warning"}})
		}()
		go func() {
			defer wg.Done()
			c.SetWorkspaces([]string{"default"})
		}()
		go func() {
			defer wg.Done()
			c.InvalidateAll()
		}()
	}

	wg.Wait()
}

func TestServiceCache_WhenConcurrentInvalidateAndGet_ShouldNotPanic(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(planFile, []byte(minimalPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedPlan(planFile, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.SeedState(stateFile, nil); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	const goroutines = 50

	wg.Add(goroutines * 4)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			c.InvalidateAll()
		}()
		go func() {
			defer wg.Done()
			c.InvalidateState()
		}()
		go func() {
			defer wg.Done()
			c.GetPlan()
		}()
		go func() {
			defer wg.Done()
			c.GetResources()
		}()
	}

	wg.Wait()
}

func TestServiceCache_WhenInvalidateAllWithDeletedFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(minimalPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedPlan(planFile, nil); err != nil {
		t.Fatal(err)
	}

	plan, ok := c.GetPlan()
	if !ok || plan == nil {
		t.Fatal("plan should be cached before file removal")
	}

	if err := os.Remove(planFile); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	plan, ok = c.GetPlan()
	if ok {
		t.Error("GetPlan() ok = true after invalidation with deleted file, want false")
	}
	if plan != nil {
		t.Errorf("GetPlan() = %v, want nil after invalidation with deleted file", plan)
	}
}

func TestServiceCache_WhenSeedPlanWithBothFileAndData_ShouldPreferFile(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(minimalPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedPlan(planFile, []byte(`different data`))
	if err != nil {
		t.Fatalf("SeedPlan() error = %v", err)
	}

	if c.planSource.kind != sourceFile {
		t.Errorf("planSource.kind = %v, want sourceFile (file takes precedence)", c.planSource.kind)
	}
}

func TestServiceCache_WhenSeedStateWithBothFileAndData_ShouldPreferFile(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedState(stateFile, []byte(`different data`))
	if err != nil {
		t.Fatalf("SeedState() error = %v", err)
	}

	if c.stateSource.kind != sourceFile {
		t.Errorf("stateSource.kind = %v, want sourceFile (file takes precedence)", c.stateSource.kind)
	}
}

func TestServiceCache_WhenSeedWithNoFileNoData_ShouldNotError(t *testing.T) {
	c := NewServiceCache()

	t.Run("ShouldHandleEmptySeedPlan", func(t *testing.T) {
		err := c.SeedPlan("", nil)
		if err != nil {
			t.Errorf("SeedPlan('', nil) error = %v, want nil (no-op)", err)
		}
		plan, ok := c.GetPlan()
		if ok {
			t.Error("GetPlan() ok = true after no-op seed")
		}
		if plan != nil {
			t.Errorf("GetPlan() = %v, want nil", plan)
		}
	})

	t.Run("ShouldHandleEmptySeedState", func(t *testing.T) {
		err := c.SeedState("", nil)
		if err != nil {
			t.Errorf("SeedState('', nil) error = %v, want nil (no-op)", err)
		}
		resources, ok := c.GetResources()
		if ok {
			t.Error("GetResources() ok = true after no-op seed")
		}
		if resources != nil {
			t.Errorf("GetResources() = %v, want nil", resources)
		}
	})
}

func TestServiceCache_WhenSeedPlanFromFileWithInvalidJSON_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(`{not valid}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedPlan(planFile, nil)
	if err == nil {
		t.Error("SeedPlan() with invalid JSON file: want error")
	}
}

func TestServiceCache_WhenSeedStateFromFileWithInvalidJSON_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(`{not valid}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedState(stateFile, nil)
	if err == nil {
		t.Error("SeedState() with invalid JSON file: want error")
	}
}

func TestServiceCache_WhenInvalidatePlanWithCorruptedFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(minimalPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedPlan(planFile, nil); err != nil {
		t.Fatal(err)
	}

	plan, ok := c.GetPlan()
	if !ok || plan == nil {
		t.Fatal("plan should be cached before corruption")
	}

	if err := os.WriteFile(planFile, []byte(`{corrupt data`), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	plan, ok = c.GetPlan()
	if ok {
		t.Error("GetPlan() ok = true after invalidation with corrupted file, want false")
	}
	if plan != nil {
		t.Errorf("GetPlan() = %v, want nil", plan)
	}
}

func TestServiceCache_WhenInvalidatePlanWithSourceNone_ShouldDoNothing(t *testing.T) {
	c := NewServiceCache()
	c.InvalidateAll()

	plan, ok := c.GetPlan()
	if ok {
		t.Error("GetPlan() ok = true, want false")
	}
	if plan != nil {
		t.Errorf("GetPlan() = %v, want nil", plan)
	}
}

func TestServiceCache_WhenInvalidatePlanWithSourceStdin_ShouldPreserveData(t *testing.T) {
	c := NewServiceCache()
	if err := c.SeedPlan("", []byte(minimalPlanJSON)); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	plan, ok := c.GetPlan()
	if !ok {
		t.Fatal("GetPlan() ok = false, want true (stdin is immutable)")
	}
	if plan == nil {
		t.Fatal("GetPlan() = nil, want non-nil")
	}
}

func TestServiceCache_WhenInvalidateStateWithCorruptedFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedState(stateFile, nil); err != nil {
		t.Fatal(err)
	}

	resources, ok := c.GetResources()
	if !ok || resources == nil {
		t.Fatal("state should be cached before corruption")
	}

	if err := os.WriteFile(stateFile, []byte(`{corrupt data`), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateState()

	resources, ok = c.GetResources()
	if ok {
		t.Error("GetResources() ok = true after invalidation with corrupted file, want false")
	}
	if resources != nil {
		t.Errorf("GetResources() = %v, want nil", resources)
	}
}

func TestServiceCache_WhenInvalidateStateWithSourceNone_ShouldDoNothing(t *testing.T) {
	c := NewServiceCache()
	c.InvalidateState()

	resources, ok := c.GetResources()
	if ok {
		t.Error("GetResources() ok = true, want false")
	}
	if resources != nil {
		t.Errorf("GetResources() = %v, want nil", resources)
	}
}

func TestServiceCache_WhenInvalidateStateWithDeletedFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedState(stateFile, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(stateFile); err != nil {
		t.Fatal(err)
	}

	c.InvalidateState()

	resources, ok := c.GetResources()
	if ok {
		t.Error("GetResources() ok = true after invalidation with deleted file, want false")
	}
	if resources != nil {
		t.Errorf("GetResources() = %v, want nil", resources)
	}

	state, sok := c.GetState()
	if sok {
		t.Error("GetState() ok = true after invalidation with deleted file, want false")
	}
	if state != nil {
		t.Errorf("GetState() = %v, want nil", state)
	}
}

// --- Outputs ---

func TestServiceCache_WhenSeedOutputsFromFile_ShouldCacheOutputs(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "outputs.json")
	if err := os.WriteFile(file, []byte(minimalOutputsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedOutputs(file, nil)
	if err != nil {
		t.Fatalf("SeedOutputs() error = %v", err)
	}

	t.Run("ShouldReturnCachedOutputs", func(t *testing.T) {
		outputs, ok := c.GetOutputs()
		if !ok {
			t.Fatal("GetOutputs() ok = false, want true")
		}
		if len(outputs) != 2 {
			t.Fatalf("len(outputs) = %d, want 2", len(outputs))
		}
		ep, exists := outputs["endpoint"]
		if !exists {
			t.Fatal("outputs[\"endpoint\"] not found")
		}
		if ep.Value != "https://api.example.com" {
			t.Errorf("outputs[\"endpoint\"].Value = %v, want %q", ep.Value, "https://api.example.com")
		}
	})

	t.Run("ShouldSetSourceFile", func(t *testing.T) {
		if c.outputsSource.kind != sourceFile {
			t.Errorf("outputsSource.kind = %v, want sourceFile", c.outputsSource.kind)
		}
	})
}

func TestServiceCache_WhenSeedOutputsFromBytes_ShouldCacheOutputs(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedOutputs("", []byte(minimalOutputsJSON))
	if err != nil {
		t.Fatalf("SeedOutputs() error = %v", err)
	}

	t.Run("ShouldReturnCachedOutputs", func(t *testing.T) {
		outputs, ok := c.GetOutputs()
		if !ok {
			t.Fatal("GetOutputs() ok = false, want true")
		}
		if len(outputs) != 2 {
			t.Fatalf("len(outputs) = %d, want 2", len(outputs))
		}
		secret, exists := outputs["secret"]
		if !exists {
			t.Fatal("outputs[\"secret\"] not found")
		}
		if !secret.Sensitive {
			t.Error("outputs[\"secret\"].Sensitive = false, want true")
		}
	})

	t.Run("ShouldSetSourceStdin", func(t *testing.T) {
		if c.outputsSource.kind != sourceStdin {
			t.Errorf("outputsSource.kind = %v, want sourceStdin", c.outputsSource.kind)
		}
	})
}

func TestServiceCache_WhenSeedOutputsWithInvalidJSON_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedOutputs("", []byte(`not json`))
	if err == nil {
		t.Error("SeedOutputs() with invalid JSON: want error")
	}

	outputs, ok := c.GetOutputs()
	if ok {
		t.Error("GetOutputs() after failed seed: ok = true, want false")
	}
	if outputs != nil {
		t.Errorf("GetOutputs() after failed seed = %v, want nil", outputs)
	}
}

func TestServiceCache_WhenSeedOutputsFromNonexistentFile_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedOutputs("/nonexistent/path/outputs.json", nil)
	if err == nil {
		t.Error("SeedOutputs() with missing file: want error")
	}
}

func TestServiceCache_WhenSeedOutputsFromFileWithInvalidJSON_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "outputs.json")
	if err := os.WriteFile(file, []byte(`{not valid`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedOutputs(file, nil)
	if err == nil {
		t.Error("SeedOutputs() with invalid JSON file: want error")
	}
}

func TestServiceCache_WhenSeedOutputsWithNoFileNoData_ShouldNotError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedOutputs("", nil)
	if err != nil {
		t.Errorf("SeedOutputs('', nil) error = %v, want nil (no-op)", err)
	}
	outputs, ok := c.GetOutputs()
	if ok {
		t.Error("GetOutputs() ok = true after no-op seed")
	}
	if outputs != nil {
		t.Errorf("GetOutputs() = %v, want nil", outputs)
	}
}

func TestServiceCache_WhenSetOutputs_ShouldStoreExecSourcedOutputs(t *testing.T) {
	c := NewServiceCache()
	outputs := map[string]sdk.OutputValue{
		"url": {Name: "url", Value: "http://localhost", Type: "string"},
	}

	c.SetOutputs(outputs)

	t.Run("ShouldReturnStoredOutputs", func(t *testing.T) {
		got, ok := c.GetOutputs()
		if !ok {
			t.Fatal("GetOutputs() ok = false, want true")
		}
		if len(got) != 1 {
			t.Fatalf("len(outputs) = %d, want 1", len(got))
		}
	})

	t.Run("ShouldSetSourceExec", func(t *testing.T) {
		if c.outputsSource.kind != sourceExec {
			t.Errorf("outputsSource.kind = %v, want sourceExec", c.outputsSource.kind)
		}
	})
}

// --- Diagnostics ---

func TestServiceCache_WhenSeedDiagnosticsFromFile_ShouldCacheDiagnostics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "validate.json")
	if err := os.WriteFile(file, []byte(minimalDiagnosticsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedDiagnostics(file, nil)
	if err != nil {
		t.Fatalf("SeedDiagnostics() error = %v", err)
	}

	t.Run("ShouldReturnCachedDiagnostics", func(t *testing.T) {
		diags, ok := c.GetDiagnostics()
		if !ok {
			t.Fatal("GetDiagnostics() ok = false, want true")
		}
		if len(diags) != 1 {
			t.Fatalf("len(diagnostics) = %d, want 1", len(diags))
		}
		if diags[0].Severity != "warning" {
			t.Errorf("diagnostics[0].Severity = %q, want %q", diags[0].Severity, "warning")
		}
		if diags[0].Line != 42 {
			t.Errorf("diagnostics[0].Line = %d, want 42", diags[0].Line)
		}
	})

	t.Run("ShouldSetSourceFile", func(t *testing.T) {
		if c.diagnosticsSource.kind != sourceFile {
			t.Errorf("diagnosticsSource.kind = %v, want sourceFile", c.diagnosticsSource.kind)
		}
	})
}

func TestServiceCache_WhenSeedDiagnosticsFromBytes_ShouldCacheDiagnostics(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedDiagnostics("", []byte(minimalDiagnosticsJSON))
	if err != nil {
		t.Fatalf("SeedDiagnostics() error = %v", err)
	}

	t.Run("ShouldReturnCachedDiagnostics", func(t *testing.T) {
		diags, ok := c.GetDiagnostics()
		if !ok {
			t.Fatal("GetDiagnostics() ok = false, want true")
		}
		if len(diags) != 1 {
			t.Fatalf("len(diagnostics) = %d, want 1", len(diags))
		}
	})

	t.Run("ShouldSetSourceStdin", func(t *testing.T) {
		if c.diagnosticsSource.kind != sourceStdin {
			t.Errorf("diagnosticsSource.kind = %v, want sourceStdin", c.diagnosticsSource.kind)
		}
	})
}

func TestServiceCache_WhenSeedDiagnosticsWithInvalidJSON_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedDiagnostics("", []byte(`{broken`))
	if err == nil {
		t.Error("SeedDiagnostics() with invalid JSON: want error")
	}

	diags, ok := c.GetDiagnostics()
	if ok {
		t.Error("GetDiagnostics() after failed seed: ok = true, want false")
	}
	if diags != nil {
		t.Errorf("GetDiagnostics() after failed seed = %v, want nil", diags)
	}
}

func TestServiceCache_WhenSeedDiagnosticsFromNonexistentFile_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedDiagnostics("/nonexistent/path/validate.json", nil)
	if err == nil {
		t.Error("SeedDiagnostics() with missing file: want error")
	}
}

func TestServiceCache_WhenSeedDiagnosticsFromFileWithInvalidJSON_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "validate.json")
	if err := os.WriteFile(file, []byte(`{not valid`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedDiagnostics(file, nil)
	if err == nil {
		t.Error("SeedDiagnostics() with invalid JSON file: want error")
	}
}

func TestServiceCache_WhenSeedDiagnosticsWithNoFileNoData_ShouldNotError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedDiagnostics("", nil)
	if err != nil {
		t.Errorf("SeedDiagnostics('', nil) error = %v, want nil (no-op)", err)
	}
	diags, ok := c.GetDiagnostics()
	if ok {
		t.Error("GetDiagnostics() ok = true after no-op seed")
	}
	if diags != nil {
		t.Errorf("GetDiagnostics() = %v, want nil", diags)
	}
}

func TestServiceCache_WhenSetDiagnostics_ShouldStoreExecSourcedDiagnostics(t *testing.T) {
	c := NewServiceCache()
	diags := []sdk.Diagnostic{
		{Severity: "error", Summary: "Missing provider"},
	}

	c.SetDiagnostics(diags)

	t.Run("ShouldReturnStoredDiagnostics", func(t *testing.T) {
		got, ok := c.GetDiagnostics()
		if !ok {
			t.Fatal("GetDiagnostics() ok = false, want true")
		}
		if len(got) != 1 {
			t.Fatalf("len(diagnostics) = %d, want 1", len(got))
		}
		if got[0].Severity != "error" {
			t.Errorf("diagnostics[0].Severity = %q, want %q", got[0].Severity, "error")
		}
	})

	t.Run("ShouldSetSourceExec", func(t *testing.T) {
		if c.diagnosticsSource.kind != sourceExec {
			t.Errorf("diagnosticsSource.kind = %v, want sourceExec", c.diagnosticsSource.kind)
		}
	})
}

// --- Workspaces ---

func TestServiceCache_WhenSeedWorkspacesFromFile_ShouldCacheWorkspaces(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "workspaces.json")
	if err := os.WriteFile(file, []byte(minimalWorkspacesJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedWorkspaces(file, nil)
	if err != nil {
		t.Fatalf("SeedWorkspaces() error = %v", err)
	}

	t.Run("ShouldReturnCachedWorkspaces", func(t *testing.T) {
		ws, ok := c.GetWorkspaces()
		if !ok {
			t.Fatal("GetWorkspaces() ok = false, want true")
		}
		if len(ws) != 3 {
			t.Fatalf("len(workspaces) = %d, want 3", len(ws))
		}
		if ws[0] != "default" {
			t.Errorf("workspaces[0] = %q, want %q", ws[0], "default")
		}
		if ws[2] != "production" {
			t.Errorf("workspaces[2] = %q, want %q", ws[2], "production")
		}
	})

	t.Run("ShouldSetSourceFile", func(t *testing.T) {
		if c.workspacesSource.kind != sourceFile {
			t.Errorf("workspacesSource.kind = %v, want sourceFile", c.workspacesSource.kind)
		}
	})
}

func TestServiceCache_WhenSeedWorkspacesFromBytes_ShouldCacheWorkspaces(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedWorkspaces("", []byte(minimalWorkspacesJSON))
	if err != nil {
		t.Fatalf("SeedWorkspaces() error = %v", err)
	}

	t.Run("ShouldReturnCachedWorkspaces", func(t *testing.T) {
		ws, ok := c.GetWorkspaces()
		if !ok {
			t.Fatal("GetWorkspaces() ok = false, want true")
		}
		if len(ws) != 3 {
			t.Fatalf("len(workspaces) = %d, want 3", len(ws))
		}
	})

	t.Run("ShouldSetSourceStdin", func(t *testing.T) {
		if c.workspacesSource.kind != sourceStdin {
			t.Errorf("workspacesSource.kind = %v, want sourceStdin", c.workspacesSource.kind)
		}
	})
}

func TestServiceCache_WhenSeedWorkspacesWithInvalidJSON_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedWorkspaces("", []byte(`not an array`))
	if err == nil {
		t.Error("SeedWorkspaces() with invalid JSON: want error")
	}

	ws, ok := c.GetWorkspaces()
	if ok {
		t.Error("GetWorkspaces() after failed seed: ok = true, want false")
	}
	if ws != nil {
		t.Errorf("GetWorkspaces() after failed seed = %v, want nil", ws)
	}
}

func TestServiceCache_WhenSeedWorkspacesFromNonexistentFile_ShouldReturnError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedWorkspaces("/nonexistent/path/workspaces.json", nil)
	if err == nil {
		t.Error("SeedWorkspaces() with missing file: want error")
	}
}

func TestServiceCache_WhenSeedWorkspacesFromFileWithInvalidJSON_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "workspaces.json")
	if err := os.WriteFile(file, []byte(`{not valid`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedWorkspaces(file, nil)
	if err == nil {
		t.Error("SeedWorkspaces() with invalid JSON file: want error")
	}
}

func TestServiceCache_WhenSeedWorkspacesWithNoFileNoData_ShouldNotError(t *testing.T) {
	c := NewServiceCache()
	err := c.SeedWorkspaces("", nil)
	if err != nil {
		t.Errorf("SeedWorkspaces('', nil) error = %v, want nil (no-op)", err)
	}
	ws, ok := c.GetWorkspaces()
	if ok {
		t.Error("GetWorkspaces() ok = true after no-op seed")
	}
	if ws != nil {
		t.Errorf("GetWorkspaces() = %v, want nil", ws)
	}
}

func TestServiceCache_WhenSetWorkspaces_ShouldStoreExecSourcedWorkspaces(t *testing.T) {
	c := NewServiceCache()
	ws := []string{"default", "dev"}

	c.SetWorkspaces(ws)

	t.Run("ShouldReturnStoredWorkspaces", func(t *testing.T) {
		got, ok := c.GetWorkspaces()
		if !ok {
			t.Fatal("GetWorkspaces() ok = false, want true")
		}
		if len(got) != 2 {
			t.Fatalf("len(workspaces) = %d, want 2", len(got))
		}
		if got[1] != "dev" {
			t.Errorf("workspaces[1] = %q, want %q", got[1], "dev")
		}
	})

	t.Run("ShouldSetSourceExec", func(t *testing.T) {
		if c.workspacesSource.kind != sourceExec {
			t.Errorf("workspacesSource.kind = %v, want sourceExec", c.workspacesSource.kind)
		}
	})
}

// --- InvalidateAll for new data types ---

func TestServiceCache_WhenInvalidateAll_ShouldClearExecSourcedOutputs(t *testing.T) {
	c := NewServiceCache()
	c.SetOutputs(map[string]sdk.OutputValue{"x": {Name: "x"}})

	c.InvalidateAll()

	outputs, ok := c.GetOutputs()
	if ok {
		t.Error("GetOutputs() ok = true after invalidation of exec-sourced outputs")
	}
	if outputs != nil {
		t.Errorf("GetOutputs() = %v, want nil", outputs)
	}
}

func TestServiceCache_WhenInvalidateAll_ShouldPreserveStdinSourcedOutputs(t *testing.T) {
	c := NewServiceCache()
	if err := c.SeedOutputs("", []byte(minimalOutputsJSON)); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	outputs, ok := c.GetOutputs()
	if !ok {
		t.Fatal("GetOutputs() ok = false, want true (stdin data preserved)")
	}
	if len(outputs) != 2 {
		t.Errorf("len(outputs) = %d, want 2", len(outputs))
	}
}

func TestServiceCache_WhenInvalidateAll_ShouldReReadFileSourcedOutputs(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "outputs.json")
	if err := os.WriteFile(file, []byte(minimalOutputsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedOutputs(file, nil); err != nil {
		t.Fatal(err)
	}

	updatedJSON := `[{"Name": "new_output", "Value": "new", "Type": "string", "Sensitive": false}]`
	if err := os.WriteFile(file, []byte(updatedJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	outputs, ok := c.GetOutputs()
	if !ok {
		t.Fatal("GetOutputs() ok = false after re-read")
	}
	if len(outputs) != 1 {
		t.Fatalf("len(outputs) = %d, want 1 (re-read from updated file)", len(outputs))
	}
	if _, exists := outputs["new_output"]; !exists {
		t.Error("outputs[\"new_output\"] not found after re-read")
	}
}

func TestServiceCache_WhenInvalidateAll_ShouldClearExecSourcedDiagnostics(t *testing.T) {
	c := NewServiceCache()
	c.SetDiagnostics([]sdk.Diagnostic{{Severity: "error", Summary: "test"}})

	c.InvalidateAll()

	diags, ok := c.GetDiagnostics()
	if ok {
		t.Error("GetDiagnostics() ok = true after invalidation of exec-sourced diagnostics")
	}
	if diags != nil {
		t.Errorf("GetDiagnostics() = %v, want nil", diags)
	}
}

func TestServiceCache_WhenInvalidateAll_ShouldPreserveStdinSourcedDiagnostics(t *testing.T) {
	c := NewServiceCache()
	if err := c.SeedDiagnostics("", []byte(minimalDiagnosticsJSON)); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	diags, ok := c.GetDiagnostics()
	if !ok {
		t.Fatal("GetDiagnostics() ok = false, want true (stdin data preserved)")
	}
	if len(diags) != 1 {
		t.Errorf("len(diagnostics) = %d, want 1", len(diags))
	}
}

func TestServiceCache_WhenInvalidateAll_ShouldClearExecSourcedWorkspaces(t *testing.T) {
	c := NewServiceCache()
	c.SetWorkspaces([]string{"default", "dev"})

	c.InvalidateAll()

	ws, ok := c.GetWorkspaces()
	if ok {
		t.Error("GetWorkspaces() ok = true after invalidation of exec-sourced workspaces")
	}
	if ws != nil {
		t.Errorf("GetWorkspaces() = %v, want nil", ws)
	}
}

func TestServiceCache_WhenInvalidateAll_ShouldPreserveStdinSourcedWorkspaces(t *testing.T) {
	c := NewServiceCache()
	if err := c.SeedWorkspaces("", []byte(minimalWorkspacesJSON)); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	ws, ok := c.GetWorkspaces()
	if !ok {
		t.Fatal("GetWorkspaces() ok = false, want true (stdin data preserved)")
	}
	if len(ws) != 3 {
		t.Errorf("len(workspaces) = %d, want 3", len(ws))
	}
}

func TestServiceCache_WhenInvalidateAll_ShouldReReadFileSourcedDiagnostics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "validate.json")
	if err := os.WriteFile(file, []byte(minimalDiagnosticsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedDiagnostics(file, nil); err != nil {
		t.Fatal(err)
	}

	updatedJSON := `[{"Severity": "error", "Summary": "New error", "Detail": "details", "File": "x.tf", "Line": 1}]`
	if err := os.WriteFile(file, []byte(updatedJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	diags, ok := c.GetDiagnostics()
	if !ok {
		t.Fatal("GetDiagnostics() ok = false after re-read")
	}
	if len(diags) != 1 {
		t.Fatalf("len(diagnostics) = %d, want 1", len(diags))
	}
	if diags[0].Severity != "error" {
		t.Errorf("diagnostics[0].Severity = %q, want %q", diags[0].Severity, "error")
	}
}

func TestServiceCache_WhenInvalidateAll_ShouldReReadFileSourcedWorkspaces(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "workspaces.json")
	if err := os.WriteFile(file, []byte(minimalWorkspacesJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedWorkspaces(file, nil); err != nil {
		t.Fatal(err)
	}

	updatedJSON := `["default", "new-ws"]`
	if err := os.WriteFile(file, []byte(updatedJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	ws, ok := c.GetWorkspaces()
	if !ok {
		t.Fatal("GetWorkspaces() ok = false after re-read")
	}
	if len(ws) != 2 {
		t.Fatalf("len(workspaces) = %d, want 2", len(ws))
	}
	if ws[1] != "new-ws" {
		t.Errorf("workspaces[1] = %q, want %q", ws[1], "new-ws")
	}
}

func TestServiceCache_WhenInvalidateAllWithDeletedOutputsFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "outputs.json")
	if err := os.WriteFile(file, []byte(minimalOutputsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedOutputs(file, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	outputs, ok := c.GetOutputs()
	if ok {
		t.Error("GetOutputs() ok = true after invalidation with deleted file")
	}
	if outputs != nil {
		t.Errorf("GetOutputs() = %v, want nil", outputs)
	}
}

func TestServiceCache_WhenInvalidateAllWithDeletedDiagnosticsFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "validate.json")
	if err := os.WriteFile(file, []byte(minimalDiagnosticsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedDiagnostics(file, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	diags, ok := c.GetDiagnostics()
	if ok {
		t.Error("GetDiagnostics() ok = true after invalidation with deleted file")
	}
	if diags != nil {
		t.Errorf("GetDiagnostics() = %v, want nil", diags)
	}
}

func TestServiceCache_WhenInvalidateAllWithDeletedWorkspacesFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "workspaces.json")
	if err := os.WriteFile(file, []byte(minimalWorkspacesJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedWorkspaces(file, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	ws, ok := c.GetWorkspaces()
	if ok {
		t.Error("GetWorkspaces() ok = true after invalidation with deleted file")
	}
	if ws != nil {
		t.Errorf("GetWorkspaces() = %v, want nil", ws)
	}
}

func TestServiceCache_WhenInvalidateAllWithCorruptedOutputsFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "outputs.json")
	if err := os.WriteFile(file, []byte(minimalOutputsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedOutputs(file, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file, []byte(`{corrupt`), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	outputs, ok := c.GetOutputs()
	if ok {
		t.Error("GetOutputs() ok = true after invalidation with corrupted file")
	}
	if outputs != nil {
		t.Errorf("GetOutputs() = %v, want nil", outputs)
	}
}

func TestServiceCache_WhenInvalidateAllWithCorruptedDiagnosticsFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "validate.json")
	if err := os.WriteFile(file, []byte(minimalDiagnosticsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedDiagnostics(file, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file, []byte(`{corrupt`), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	diags, ok := c.GetDiagnostics()
	if ok {
		t.Error("GetDiagnostics() ok = true after invalidation with corrupted file")
	}
	if diags != nil {
		t.Errorf("GetDiagnostics() = %v, want nil", diags)
	}
}

func TestServiceCache_WhenInvalidateAllWithCorruptedWorkspacesFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "workspaces.json")
	if err := os.WriteFile(file, []byte(minimalWorkspacesJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedWorkspaces(file, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file, []byte(`{corrupt`), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	ws, ok := c.GetWorkspaces()
	if ok {
		t.Error("GetWorkspaces() ok = true after invalidation with corrupted file")
	}
	if ws != nil {
		t.Errorf("GetWorkspaces() = %v, want nil", ws)
	}
}
