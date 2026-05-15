package terraform

import (
	"context"
	"sync"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

var _ sdk.Service = (*MacroService)(nil)

func TestMacroService_ReadsFromCache(t *testing.T) {
	cache := NewServiceCache()
	cache.SetPlan(&sdk.PlanSummary{ToCreate: 3, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}}}})
	cache.SetState([]sdk.Resource{{Address: "r1"}}, nil)

	svc := NewMacroService("terraform", cache)
	ctx := context.Background()

	t.Run("Plan", func(t *testing.T) {
		got, err := svc.Plan(ctx, sdk.PlanOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if got.ToCreate != 3 {
			t.Errorf("ToCreate = %d, want 3", got.ToCreate)
		}
	})

	t.Run("StateList", func(t *testing.T) {
		got, err := svc.StateList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Address != "r1" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("Show returns empty when no tfjson state", func(t *testing.T) {
		got, err := svc.Show(ctx, "x")
		if err != nil {
			t.Errorf("Show: %v", err)
		}
		if got != "{}" {
			t.Errorf("got %q, want {}", got)
		}
	})

	t.Run("Workspace", func(t *testing.T) {
		got, err := svc.Workspace(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got != "default" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("WorkspaceList", func(t *testing.T) {
		got, err := svc.WorkspaceList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != "default" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("mutators return nil", func(t *testing.T) {
		if err := svc.Apply(ctx, sdk.ApplyOptions{}); err != nil {
			t.Errorf("Apply: %v", err)
		}
		if err := svc.WorkspaceSelect(ctx, "x"); err != nil {
			t.Errorf("WorkspaceSelect: %v", err)
		}
		if err := svc.WorkspaceNew(ctx, "x", sdk.WorkspaceNewOptions{}); err != nil {
			t.Errorf("WorkspaceNew: %v", err)
		}
		if err := svc.WorkspaceDelete(ctx, "x", sdk.WorkspaceDeleteOptions{}); err != nil {
			t.Errorf("WorkspaceDelete: %v", err)
		}
		if err := svc.StateRm(ctx, "x"); err != nil {
			t.Errorf("StateRm: %v", err)
		}
		if err := svc.StateMove(ctx, "a", "b"); err != nil {
			t.Errorf("StateMove: %v", err)
		}
		if err := svc.Import(ctx, "a", "b"); err != nil {
			t.Errorf("Import: %v", err)
		}
		if err := svc.Taint(ctx, "x"); err != nil {
			t.Errorf("Taint: %v", err)
		}
		if err := svc.Untaint(ctx, "x"); err != nil {
			t.Errorf("Untaint: %v", err)
		}
		if err := svc.Refresh(ctx); err != nil {
			t.Errorf("Refresh: %v", err)
		}
		if err := svc.Init(ctx, sdk.InitOptions{}); err != nil {
			t.Errorf("Init: %v", err)
		}
		if err := svc.ForceUnlock(ctx, "id"); err != nil {
			t.Errorf("ForceUnlock: %v", err)
		}
	})

	t.Run("Validate", func(t *testing.T) {
		got, err := svc.Validate(ctx)
		if err != nil {
			t.Errorf("Validate: %v", err)
		}
		if got == nil {
			t.Error("Validate returned nil")
		}
	})

	t.Run("Output", func(t *testing.T) {
		got, err := svc.Output(ctx)
		if err != nil {
			t.Errorf("Output: %v", err)
		}
		if got == nil {
			t.Error("Output returned nil")
		}
	})
}

func TestMacroService_ReturnsEmptyWhenCacheEmpty(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	t.Run("Plan", func(t *testing.T) {
		got, err := svc.Plan(ctx, sdk.PlanOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if got == nil {
			t.Fatal("Plan returned nil")
		}
		if got.ToCreate != 0 {
			t.Errorf("ToCreate = %d, want 0", got.ToCreate)
		}
	})

	t.Run("StateList", func(t *testing.T) {
		got, err := svc.StateList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got == nil {
			t.Fatal("StateList returned nil")
		}
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("Show", func(t *testing.T) {
		got, err := svc.Show(ctx, "x")
		if err != nil {
			t.Fatal(err)
		}
		if got != "{}" {
			t.Errorf("got %q, want {}", got)
		}
	})
}

func TestMacroService_RecordsOperationsOnly(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	svc.StateList(ctx)
	svc.Show(ctx, "aws_instance.web")
	svc.Workspace(ctx)
	svc.WorkspaceList(ctx)
	svc.Validate(ctx)
	svc.Output(ctx)

	if cmds := svc.Commands(); len(cmds) != 0 {
		t.Fatalf("data fetches produced %d commands, want 0: %v", len(cmds), cmds)
	}

	operations := []struct {
		name     string
		invoke   func()
		expected string
	}{
		{"Plan", func() { svc.Plan(ctx, sdk.PlanOptions{}) }, "terraform plan"},
		{"Apply", func() { svc.Apply(ctx, sdk.ApplyOptions{}) }, "terraform apply"},
		{"WorkspaceSelect", func() { svc.WorkspaceSelect(ctx, "prod") }, "terraform workspace select prod"},
		{"WorkspaceNew", func() { svc.WorkspaceNew(ctx, "staging", sdk.WorkspaceNewOptions{}) }, "terraform workspace new staging"},
		{"WorkspaceDelete", func() { svc.WorkspaceDelete(ctx, "old", sdk.WorkspaceDeleteOptions{}) }, "terraform workspace delete old"},
		{"StateRm", func() { svc.StateRm(ctx, "aws_instance.web") }, "terraform state rm aws_instance.web"},
		{"StateMove", func() { svc.StateMove(ctx, "old", "new") }, "terraform state mv old new"},
		{"Import", func() { svc.Import(ctx, "aws_instance.web", "i-123") }, "terraform import aws_instance.web i-123"},
		{"Taint", func() { svc.Taint(ctx, "aws_instance.web") }, "terraform taint aws_instance.web"},
		{"Untaint", func() { svc.Untaint(ctx, "aws_instance.web") }, "terraform untaint aws_instance.web"},
		{"Refresh", func() { svc.Refresh(ctx) }, "terraform refresh"},
		{"Init", func() { svc.Init(ctx, sdk.InitOptions{}) }, "terraform init"},
		{"ForceUnlock", func() { svc.ForceUnlock(ctx, "abc-123") }, "terraform force-unlock -force abc-123"},
	}

	for i, tt := range operations {
		t.Run(tt.name, func(t *testing.T) {
			tt.invoke()
			cmds := svc.Commands()
			if len(cmds) != i+1 {
				t.Fatalf("expected %d commands, got %d", i+1, len(cmds))
			}
			if cmds[i].String() != tt.expected {
				t.Errorf("got %q, want %q", cmds[i].String(), tt.expected)
			}
		})
	}
}

func TestMacroService_CommandsReturnsNilWhenEmpty(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	cmds := svc.Commands()
	if cmds != nil {
		t.Errorf("expected nil, got %v", cmds)
	}
}

func TestMacroService_WithDirSharesStore(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	child := svc.WithDir("/other/dir")
	child.(*MacroService).Taint(ctx, "aws_instance.a")
	svc.Untaint(ctx, "aws_instance.b")

	cmds := svc.Commands()
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0].Verb != "taint" {
		t.Errorf("first verb = %q, want taint", cmds[0].Verb)
	}
	if cmds[1].Verb != "untaint" {
		t.Errorf("second verb = %q, want untaint", cmds[1].Verb)
	}
}

func TestMacroService_WithDirClearsCache(t *testing.T) {
	cache := NewServiceCache()
	cache.SetState([]sdk.Resource{{Address: "r1"}}, nil)

	svc := NewMacroService("terraform", cache)
	child := svc.WithDir("/other/dir")

	resources, err := child.(*MacroService).StateList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 0 {
		t.Errorf("child should have empty cache, got %d resources", len(resources))
	}
}

func TestMacroService_ConcurrentAccess(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.Taint(ctx, "res")
		}()
	}
	wg.Wait()

	cmds := svc.Commands()
	if len(cmds) != 100 {
		t.Fatalf("expected 100 commands, got %d", len(cmds))
	}
}

func TestMacroService_PlanFlagsRecorded(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	lockFalse := false
	opts := sdk.PlanOptions{
		Targets:     []string{"aws_instance.web"},
		VarFiles:    []string{"prod.tfvars"},
		Destroy:     true,
		Parallelism: 5,
		Lock:        &lockFalse,
	}
	svc.Plan(ctx, opts)

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1, got %d", len(cmds))
	}
	expected := "terraform plan -target=aws_instance.web -var-file=prod.tfvars -destroy -parallelism=5 -lock=false"
	if cmds[0].String() != expected {
		t.Errorf("got %q, want %q", cmds[0].String(), expected)
	}
}

func TestMacroService_ApplyFlagsRecorded(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	opts := sdk.ApplyOptions{
		Targets:  []string{"aws_instance.web"},
		VarFiles: []string{"prod.tfvars"},
	}
	svc.Apply(ctx, opts)

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1, got %d", len(cmds))
	}
	expected := "terraform apply -target=aws_instance.web -var-file=prod.tfvars"
	if cmds[0].String() != expected {
		t.Errorf("got %q, want %q", cmds[0].String(), expected)
	}
}

func TestMacroService_CustomBinary(t *testing.T) {
	svc := NewMacroService("tofu", nil)
	ctx := context.Background()

	svc.Apply(ctx, sdk.ApplyOptions{})
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1, got %d", len(cmds))
	}
	if cmds[0].Binary != "tofu" {
		t.Errorf("binary = %q, want tofu", cmds[0].Binary)
	}
	if cmds[0].String() != "tofu apply" {
		t.Errorf("got %q", cmds[0].String())
	}
}

func TestMacroService_DefaultBinary(t *testing.T) {
	svc := NewMacroService("", nil)
	ctx := context.Background()

	svc.Refresh(ctx)
	cmds := svc.Commands()
	if cmds[0].Binary != "terraform" {
		t.Errorf("binary = %q, want terraform", cmds[0].Binary)
	}
}

func TestMacroService_MutationsNeverExecute(t *testing.T) {
	cache := NewServiceCache()
	cache.SetState([]sdk.Resource{{Address: "before"}}, nil)

	svc := NewMacroService("terraform", cache)
	ctx := context.Background()

	svc.Apply(ctx, sdk.ApplyOptions{})
	svc.StateRm(ctx, "aws_instance.web")
	svc.StateMove(ctx, "old", "new")
	svc.Import(ctx, "aws_instance.web", "i-123")
	svc.Taint(ctx, "aws_instance.web")
	svc.Untaint(ctx, "aws_instance.web")
	svc.WorkspaceSelect(ctx, "prod")
	svc.WorkspaceNew(ctx, "staging", sdk.WorkspaceNewOptions{})
	svc.WorkspaceDelete(ctx, "old", sdk.WorkspaceDeleteOptions{})
	svc.Init(ctx, sdk.InitOptions{})
	svc.ForceUnlock(ctx, "lock-id")

	cmds := svc.Commands()
	if len(cmds) != 11 {
		t.Fatalf("expected 11 recorded commands, got %d", len(cmds))
	}

	// State should still be readable (mutations don't modify cache)
	resources, err := svc.StateList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].Address != "before" {
		t.Errorf("state changed after mutations: %v", resources)
	}
}
