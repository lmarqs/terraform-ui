package terraform

import (
	"context"
	"errors"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestStaticServiceReadMethods(t *testing.T) {
	plan := &sdk.PlanSummary{
		Changes:  []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate}},
		ToCreate: 1,
	}
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Name: "web"},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket", Name: "data"},
	}
	state := &tfjson.State{
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Address:         "aws_instance.web",
						Type:            "aws_instance",
						Name:            "web",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"id": "i-123", "ami": "ami-abc"},
					},
					{
						Address:         "aws_s3_bucket.data",
						Type:            "aws_s3_bucket",
						Name:            "data",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"bucket": "my-bucket"},
					},
				},
			},
		},
	}

	svc := NewStaticService(plan, resources, state)
	ctx := context.Background()

	t.Run("Plan returns pre-loaded summary", func(t *testing.T) {
		got, err := svc.Plan(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.ToCreate != 1 {
			t.Errorf("ToCreate = %d, want 1", got.ToCreate)
		}
		if len(got.Changes) != 1 {
			t.Fatalf("len(Changes) = %d, want 1", len(got.Changes))
		}
		if got.Changes[0].Resource.Address != "aws_instance.web" {
			t.Errorf("Changes[0].Address = %q", got.Changes[0].Resource.Address)
		}
	})

	t.Run("Plan ignores targets", func(t *testing.T) {
		got, err := svc.Plan(ctx, []string{"aws_instance.web"})
		if err != nil {
			t.Fatal(err)
		}
		if got.ToCreate != 1 {
			t.Errorf("targets should be ignored, ToCreate = %d", got.ToCreate)
		}
	})

	t.Run("StateList returns pre-loaded resources", func(t *testing.T) {
		got, err := svc.StateList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].Address != "aws_instance.web" {
			t.Errorf("got[0].Address = %q", got[0].Address)
		}
	})

	t.Run("Show returns resource JSON", func(t *testing.T) {
		got, err := svc.Show(ctx, "aws_instance.web")
		if err != nil {
			t.Fatal(err)
		}
		if got == "" {
			t.Error("expected non-empty JSON")
		}
	})

	t.Run("Show error: not found", func(t *testing.T) {
		_, err := svc.Show(ctx, "aws_instance.nonexistent")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Workspace returns readonly", func(t *testing.T) {
		ws, err := svc.Workspace(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if ws != "readonly" {
			t.Errorf("workspace = %q, want 'readonly'", ws)
		}
	})

	t.Run("WorkspaceList returns readonly", func(t *testing.T) {
		list, err := svc.WorkspaceList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0] != "readonly" {
			t.Errorf("workspaces = %v, want [readonly]", list)
		}
	})

	t.Run("WithDir returns same service", func(t *testing.T) {
		got := svc.WithDir("/other/dir")
		if got != svc {
			t.Error("WithDir should return same instance")
		}
	})
}

func TestStaticServiceNilData(t *testing.T) {
	ctx := context.Background()

	t.Run("nil plan returns empty summary", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil)
		got, err := svc.Plan(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Changes) != 0 {
			t.Errorf("len(Changes) = %d, want 0", len(got.Changes))
		}
	})

	t.Run("nil resources returns empty slice", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil)
		got, err := svc.StateList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("nil state Show returns error", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil)
		_, err := svc.Show(ctx, "anything")
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestStaticServiceMutatingMethods(t *testing.T) {
	svc := NewStaticService(nil, nil, nil)
	ctx := context.Background()

	mutators := []struct {
		name string
		fn   func() error
	}{
		{"Apply", func() error { return svc.Apply(ctx, nil) }},
		{"WorkspaceSelect", func() error { return svc.WorkspaceSelect(ctx, "x") }},
		{"WorkspaceNew", func() error { return svc.WorkspaceNew(ctx, "x") }},
		{"WorkspaceDelete", func() error { return svc.WorkspaceDelete(ctx, "x") }},
		{"StateRm", func() error { return svc.StateRm(ctx, "x") }},
		{"StateMove", func() error { return svc.StateMove(ctx, "a", "b") }},
		{"Import", func() error { return svc.Import(ctx, "a", "b") }},
		{"Taint", func() error { return svc.Taint(ctx, "x") }},
		{"Untaint", func() error { return svc.Untaint(ctx, "x") }},
		{"Refresh", func() error { return svc.Refresh(ctx) }},
		{"Init", func() error { return svc.Init(ctx) }},
		{"ForceUnlock", func() error { return svc.ForceUnlock(ctx, "lock-id") }},
	}

	for _, m := range mutators {
		t.Run(m.name+" returns ErrReadOnly", func(t *testing.T) {
			err := m.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrReadOnly) {
				t.Errorf("error = %v, want ErrReadOnly", err)
			}
		})
	}

	t.Run("Validate returns ErrReadOnly", func(t *testing.T) {
		_, err := svc.Validate(ctx)
		if !errors.Is(err, ErrReadOnly) {
			t.Errorf("error = %v, want ErrReadOnly", err)
		}
	})

	t.Run("Output returns ErrReadOnly", func(t *testing.T) {
		_, err := svc.Output(ctx)
		if !errors.Is(err, ErrReadOnly) {
			t.Errorf("error = %v, want ErrReadOnly", err)
		}
	})
}

func TestStaticServiceImplementsInterface(t *testing.T) {
	var _ sdk.Service = (*StaticService)(nil)
}
