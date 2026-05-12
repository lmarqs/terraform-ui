package terraform

import (
	"context"
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

	svc := NewStaticService(plan, resources, state, "")
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
		svc := NewStaticService(nil, nil, nil, "")
		got, err := svc.Plan(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Changes) != 0 {
			t.Errorf("len(Changes) = %d, want 0", len(got.Changes))
		}
	})

	t.Run("nil resources returns empty slice", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil, "")
		got, err := svc.StateList(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("nil state Show returns error", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil, "")
		_, err := svc.Show(ctx, "anything")
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestStaticServiceMutatingMethods(t *testing.T) {
	svc := NewStaticService(nil, nil, nil, "")
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
		t.Run(m.name+" returns CommandErr", func(t *testing.T) {
			err := m.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			cmd, ok := sdk.IsCommandErr(err)
			if !ok {
				t.Fatalf("expected CommandErr, got %T: %v", err, err)
			}
			if cmd.Binary != "terraform" {
				t.Errorf("binary = %q, want terraform", cmd.Binary)
			}
			if cmd.Verb == "" {
				t.Error("expected non-empty verb")
			}
		})
	}

	t.Run("Validate returns CommandErr", func(t *testing.T) {
		_, err := svc.Validate(ctx)
		if _, ok := sdk.IsCommandErr(err); !ok {
			t.Errorf("expected CommandErr, got: %v", err)
		}
	})

	t.Run("Output returns CommandErr", func(t *testing.T) {
		_, err := svc.Output(ctx)
		if _, ok := sdk.IsCommandErr(err); !ok {
			t.Errorf("expected CommandErr, got: %v", err)
		}
	})
}

func TestStaticServiceImplementsInterface(t *testing.T) {
	var _ sdk.Service = (*StaticService)(nil)
}

func TestStaticServiceCommands(t *testing.T) {
	ctx := context.Background()

	t.Run("custom binary", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil, "tofu")
		_ = svc.Apply(ctx, []string{"aws_instance.web"})
		cmds := svc.Commands()
		if len(cmds) != 1 {
			t.Fatalf("expected 1 command, got %d", len(cmds))
		}
		if cmds[0].Binary != "tofu" {
			t.Errorf("binary = %q, want tofu", cmds[0].Binary)
		}
		if cmds[0].String() != "tofu apply -target=aws_instance.web" {
			t.Errorf("string = %q", cmds[0].String())
		}
	})

	t.Run("execution order", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil, "terraform")
		_ = svc.Taint(ctx, "aws_instance.a")
		_ = svc.Apply(ctx, nil)
		_ = svc.StateRm(ctx, "aws_instance.b")
		cmds := svc.Commands()
		if len(cmds) != 3 {
			t.Fatalf("expected 3 commands, got %d", len(cmds))
		}
		if cmds[0].Verb != "taint" {
			t.Errorf("first verb = %q, want taint", cmds[0].Verb)
		}
		if cmds[1].Verb != "apply" {
			t.Errorf("second verb = %q, want apply", cmds[1].Verb)
		}
		if cmds[2].Verb != "state rm" {
			t.Errorf("third verb = %q, want state rm", cmds[2].Verb)
		}
	})

	t.Run("read methods do not collect", func(t *testing.T) {
		plan := &sdk.PlanSummary{Changes: []sdk.PlanChange{}}
		resources := []sdk.Resource{{Address: "a"}}
		svc := NewStaticService(plan, resources, nil, "")
		_, _ = svc.Plan(ctx, nil)
		_, _ = svc.StateList(ctx)
		_, _ = svc.Workspace(ctx)
		_, _ = svc.WorkspaceList(ctx)
		if len(svc.Commands()) != 0 {
			t.Errorf("expected 0 commands, got %d", len(svc.Commands()))
		}
	})

	t.Run("default binary when empty", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil, "")
		_ = svc.Refresh(ctx)
		cmds := svc.Commands()
		if len(cmds) != 1 {
			t.Fatalf("expected 1 command, got %d", len(cmds))
		}
		if cmds[0].Binary != "terraform" {
			t.Errorf("binary = %q, want terraform", cmds[0].Binary)
		}
	})

	t.Run("Commands returns nil when nothing triggered", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil, "terraform")
		cmds := svc.Commands()
		if cmds != nil {
			t.Errorf("expected nil, got %v", cmds)
		}
	})

	t.Run("multiple calls accumulate", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil, "terraform")
		_ = svc.Apply(ctx, nil)
		_ = svc.Apply(ctx, []string{"aws_instance.a"})
		cmds := svc.Commands()
		if len(cmds) != 2 {
			t.Fatalf("expected 2 commands, got %d", len(cmds))
		}
		if cmds[0].String() != "terraform apply" {
			t.Errorf("first = %q, want %q", cmds[0].String(), "terraform apply")
		}
		if cmds[1].String() != "terraform apply -target=aws_instance.a" {
			t.Errorf("second = %q, want %q", cmds[1].String(), "terraform apply -target=aws_instance.a")
		}
	})

	t.Run("WithDir does not affect command collection", func(t *testing.T) {
		svc := NewStaticService(nil, nil, nil, "terraform")
		sub := svc.WithDir("/some/other/dir")
		subSvc := sub.(*StaticService)
		_ = subSvc.Taint(ctx, "aws_instance.x")
		_ = svc.Untaint(ctx, "aws_instance.y")
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
	})

	t.Run("all mutating methods produce correct command strings", func(t *testing.T) {
		tests := []struct {
			name     string
			invoke   func(svc *StaticService) error
			expected string
		}{
			{
				"Apply without targets",
				func(svc *StaticService) error { return svc.Apply(ctx, nil) },
				"terraform apply",
			},
			{
				"Apply with single target",
				func(svc *StaticService) error { return svc.Apply(ctx, []string{"aws_instance.web"}) },
				"terraform apply -target=aws_instance.web",
			},
			{
				"Apply with multiple targets",
				func(svc *StaticService) error {
					return svc.Apply(ctx, []string{"aws_instance.web", "aws_s3_bucket.data"})
				},
				"terraform apply -target=aws_instance.web -target=aws_s3_bucket.data",
			},
			{
				"StateRm",
				func(svc *StaticService) error { return svc.StateRm(ctx, "aws_instance.web") },
				"terraform state rm aws_instance.web",
			},
			{
				"StateMove",
				func(svc *StaticService) error { return svc.StateMove(ctx, "aws_instance.old", "aws_instance.new") },
				"terraform state mv aws_instance.old aws_instance.new",
			},
			{
				"Import",
				func(svc *StaticService) error { return svc.Import(ctx, "aws_instance.web", "i-1234567890") },
				"terraform import aws_instance.web i-1234567890",
			},
			{
				"Taint",
				func(svc *StaticService) error { return svc.Taint(ctx, "aws_instance.web") },
				"terraform taint aws_instance.web",
			},
			{
				"Untaint",
				func(svc *StaticService) error { return svc.Untaint(ctx, "aws_instance.web") },
				"terraform untaint aws_instance.web",
			},
			{
				"Validate",
				func(svc *StaticService) error { _, err := svc.Validate(ctx); return err },
				"terraform validate",
			},
			{
				"Output",
				func(svc *StaticService) error { _, err := svc.Output(ctx); return err },
				"terraform output",
			},
			{
				"Refresh",
				func(svc *StaticService) error { return svc.Refresh(ctx) },
				"terraform refresh",
			},
			{
				"Init",
				func(svc *StaticService) error { return svc.Init(ctx) },
				"terraform init",
			},
			{
				"ForceUnlock",
				func(svc *StaticService) error { return svc.ForceUnlock(ctx, "abc-123") },
				"terraform force-unlock -force abc-123",
			},
			{
				"WorkspaceSelect",
				func(svc *StaticService) error { return svc.WorkspaceSelect(ctx, "production") },
				"terraform workspace select production",
			},
			{
				"WorkspaceNew",
				func(svc *StaticService) error { return svc.WorkspaceNew(ctx, "staging") },
				"terraform workspace new staging",
			},
			{
				"WorkspaceDelete",
				func(svc *StaticService) error { return svc.WorkspaceDelete(ctx, "old-env") },
				"terraform workspace delete old-env",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc := NewStaticService(nil, nil, nil, "")
				err := tt.invoke(svc)
				if err == nil {
					t.Fatal("expected error")
				}
				cmds := svc.Commands()
				if len(cmds) != 1 {
					t.Fatalf("expected 1 command, got %d", len(cmds))
				}
				if cmds[0].String() != tt.expected {
					t.Errorf("got %q, want %q", cmds[0].String(), tt.expected)
				}
			})
		}
	})
}
