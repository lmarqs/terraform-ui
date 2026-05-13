package terraform

import (
	"context"
	"sync"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

var _ sdk.Service = (*RecordingService)(nil)

type stubService struct {
	plan      *sdk.PlanSummary
	resources []sdk.Resource
	dir       string
}

func (s *stubService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	if s.plan == nil {
		return &sdk.PlanSummary{Changes: []sdk.PlanChange{}}, nil
	}
	return s.plan, nil
}

func (s *stubService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (s *stubService) StateList(_ context.Context) ([]sdk.Resource, error) {
	return s.resources, nil
}
func (s *stubService) Show(_ context.Context, _ string) (string, error) { return "{}", nil }
func (s *stubService) Workspace(_ context.Context) (string, error)      { return "default", nil }
func (s *stubService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"default"}, nil
}
func (s *stubService) WorkspaceSelect(_ context.Context, _ string) error    { return nil }
func (s *stubService) WorkspaceNew(_ context.Context, _ string) error       { return nil }
func (s *stubService) WorkspaceDelete(_ context.Context, _ string) error    { return nil }
func (s *stubService) StateRm(_ context.Context, _ string) error            { return nil }
func (s *stubService) StateMove(_ context.Context, _, _ string) error       { return nil }
func (s *stubService) Import(_ context.Context, _, _ string) error          { return nil }
func (s *stubService) Taint(_ context.Context, _ string) error              { return nil }
func (s *stubService) Untaint(_ context.Context, _ string) error            { return nil }
func (s *stubService) Validate(_ context.Context) ([]sdk.Diagnostic, error) { return nil, nil }
func (s *stubService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}
func (s *stubService) Refresh(_ context.Context) error               { return nil }
func (s *stubService) Init(_ context.Context) error                  { return nil }
func (s *stubService) ForceUnlock(_ context.Context, _ string) error { return nil }
func (s *stubService) WithDir(dir string) sdk.Service {
	return &stubService{plan: s.plan, resources: s.resources, dir: dir}
}

func TestRecordingService_DelegatesToInner(t *testing.T) {
	plan := &sdk.PlanSummary{ToCreate: 3, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}}}}
	resources := []sdk.Resource{{Address: "r1"}}
	inner := &stubService{plan: plan, resources: resources}
	svc := NewRecordingService(inner, "terraform")
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

	t.Run("Show", func(t *testing.T) {
		got, err := svc.Show(ctx, "x")
		if err != nil {
			t.Fatal(err)
		}
		if got != "{}" {
			t.Errorf("got %q", got)
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
		if err := svc.WorkspaceNew(ctx, "x"); err != nil {
			t.Errorf("WorkspaceNew: %v", err)
		}
		if err := svc.WorkspaceDelete(ctx, "x"); err != nil {
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
		if err := svc.Init(ctx); err != nil {
			t.Errorf("Init: %v", err)
		}
		if err := svc.ForceUnlock(ctx, "id"); err != nil {
			t.Errorf("ForceUnlock: %v", err)
		}
	})

	t.Run("Validate", func(t *testing.T) {
		_, err := svc.Validate(ctx)
		if err != nil {
			t.Errorf("Validate: %v", err)
		}
	})

	t.Run("Output", func(t *testing.T) {
		_, err := svc.Output(ctx)
		if err != nil {
			t.Errorf("Output: %v", err)
		}
	})
}

func TestRecordingService_RecordsAllCommands(t *testing.T) {
	svc := NewRecordingService(&stubService{}, "terraform")
	ctx := context.Background()

	tests := []struct {
		name     string
		invoke   func()
		expected string
	}{
		{"Plan", func() { svc.Plan(ctx, sdk.PlanOptions{}) }, "terraform plan"},
		{"Apply", func() { svc.Apply(ctx, sdk.ApplyOptions{}) }, "terraform apply"},
		{"StateList", func() { svc.StateList(ctx) }, "terraform state list"},
		{"Show", func() { svc.Show(ctx, "aws_instance.web") }, "terraform state show aws_instance.web"},
		{"Workspace", func() { svc.Workspace(ctx) }, "terraform workspace show"},
		{"WorkspaceList", func() { svc.WorkspaceList(ctx) }, "terraform workspace list"},
		{"WorkspaceSelect", func() { svc.WorkspaceSelect(ctx, "prod") }, "terraform workspace select prod"},
		{"WorkspaceNew", func() { svc.WorkspaceNew(ctx, "staging") }, "terraform workspace new staging"},
		{"WorkspaceDelete", func() { svc.WorkspaceDelete(ctx, "old") }, "terraform workspace delete old"},
		{"StateRm", func() { svc.StateRm(ctx, "aws_instance.web") }, "terraform state rm aws_instance.web"},
		{"StateMove", func() { svc.StateMove(ctx, "old", "new") }, "terraform state mv old new"},
		{"Import", func() { svc.Import(ctx, "aws_instance.web", "i-123") }, "terraform import aws_instance.web i-123"},
		{"Taint", func() { svc.Taint(ctx, "aws_instance.web") }, "terraform taint aws_instance.web"},
		{"Untaint", func() { svc.Untaint(ctx, "aws_instance.web") }, "terraform untaint aws_instance.web"},
		{"Validate", func() { svc.Validate(ctx) }, "terraform validate"},
		{"Output", func() { svc.Output(ctx) }, "terraform output"},
		{"Refresh", func() { svc.Refresh(ctx) }, "terraform refresh"},
		{"Init", func() { svc.Init(ctx) }, "terraform init"},
		{"ForceUnlock", func() { svc.ForceUnlock(ctx, "abc-123") }, "terraform force-unlock -force abc-123"},
	}

	for i, tt := range tests {
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

func TestRecordingService_CommandsReturnsNilWhenEmpty(t *testing.T) {
	svc := NewRecordingService(&stubService{}, "terraform")
	cmds := svc.Commands()
	if cmds != nil {
		t.Errorf("expected nil, got %v", cmds)
	}
}

func TestRecordingService_WithDirSharesStore(t *testing.T) {
	svc := NewRecordingService(&stubService{}, "terraform")
	ctx := context.Background()

	child := svc.WithDir("/other/dir")
	child.Taint(ctx, "aws_instance.a")
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

func TestRecordingService_WithDirDelegatesToInner(t *testing.T) {
	inner := &stubService{dir: "/original"}
	svc := NewRecordingService(inner, "terraform")

	child := svc.WithDir("/new/dir")
	rec := child.(*RecordingService)
	stub := rec.inner.(*stubService)
	if stub.dir != "/new/dir" {
		t.Errorf("inner dir = %q, want /new/dir", stub.dir)
	}
}

func TestRecordingService_ConcurrentAccess(t *testing.T) {
	svc := NewRecordingService(&stubService{}, "terraform")
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

func TestRecordingService_PlanFlagsRecorded(t *testing.T) {
	svc := NewRecordingService(&stubService{}, "terraform")
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

func TestRecordingService_ApplyFlagsRecorded(t *testing.T) {
	svc := NewRecordingService(&stubService{}, "terraform")
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

func TestRecordingService_CustomBinary(t *testing.T) {
	svc := NewRecordingService(&stubService{}, "tofu")
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

func TestRecordingService_DefaultBinary(t *testing.T) {
	svc := NewRecordingService(&stubService{}, "")
	ctx := context.Background()

	svc.Refresh(ctx)
	cmds := svc.Commands()
	if cmds[0].Binary != "terraform" {
		t.Errorf("binary = %q, want terraform", cmds[0].Binary)
	}
}
