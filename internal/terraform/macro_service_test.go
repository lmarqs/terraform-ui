package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
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

	t.Run("WorkspaceList without cache", func(t *testing.T) {
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

func TestMacroService_WhenOutputsCached_ShouldReturnFromCache(t *testing.T) {
	cache := NewServiceCache()
	cache.SetOutputs(map[string]sdk.OutputValue{
		"url": {Name: "url", Value: "http://localhost", Type: "string"},
	})

	svc := NewMacroService("terraform", cache)
	got, err := svc.Output(context.Background())
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(outputs) = %d, want 1", len(got))
	}
	if got["url"].Value != "http://localhost" {
		t.Errorf("outputs[\"url\"].Value = %v, want %q", got["url"].Value, "http://localhost")
	}
}

func TestMacroService_WhenDiagnosticsCached_ShouldReturnFromCache(t *testing.T) {
	cache := NewServiceCache()
	cache.SetDiagnostics([]sdk.Diagnostic{
		{Severity: "warning", Summary: "Deprecated"},
	})

	svc := NewMacroService("terraform", cache)
	got, err := svc.Validate(context.Background())
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(diagnostics) = %d, want 1", len(got))
	}
	if got[0].Summary != "Deprecated" {
		t.Errorf("diagnostics[0].Summary = %q, want %q", got[0].Summary, "Deprecated")
	}
}

func TestMacroService_WhenWorkspacesCached_ShouldReturnFromCache(t *testing.T) {
	cache := NewServiceCache()
	cache.SetWorkspaces([]string{"default", "staging", "production"})

	svc := NewMacroService("terraform", cache)
	got, err := svc.WorkspaceList(context.Background())
	if err != nil {
		t.Fatalf("WorkspaceList() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(workspaces) = %d, want 3", len(got))
	}
	if got[1] != "staging" {
		t.Errorf("workspaces[1] = %q, want %q", got[1], "staging")
	}
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

	opts := sdk.PlanOptions{
		Targets:     []string{"aws_instance.web"},
		VarFiles:    []string{"prod.tfvars"},
		Destroy:     true,
		Parallelism: 5,
		Lock:        sdk.LockDisabled,
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

func TestMacroService_PlanFlagsAllBranches(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	opts := sdk.PlanOptions{
		PlanFile:    "/tmp/plan.out",
		Targets:     []string{"aws_instance.web"},
		VarFiles:    []string{"prod.tfvars"},
		Vars:        map[string]string{"region": "us-east-1"},
		Replace:     []string{"aws_instance.old"},
		Destroy:     true,
		Refresh:     sdk.RefreshEnabled,
		Parallelism: 5,
		Lock:        sdk.LockEnabled,
		LockTimeout: sdk.LockTimeout("10s"),
		ExtraArgs:   []string{"-input=false"},
	}
	svc.Plan(ctx, opts)

	cmds := svc.Commands()
	cmd := cmds[0].String()
	for _, want := range []string{
		"-out=/tmp/plan.out",
		"-target=aws_instance.web",
		"-var-file=prod.tfvars",
		"-var", "region=us-east-1",
		"-replace=aws_instance.old",
		"-destroy",
		"-refresh=true",
		"-parallelism=5",
		"-lock=true",
		"-lock-timeout=10s",
		"-input=false",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("plan flags missing %q in %q", want, cmd)
		}
	}
}

func TestMacroService_PlanFlagsRefreshModes(t *testing.T) {
	tests := []struct {
		name    string
		refresh sdk.RefreshMode
		want    string
	}{
		{"RefreshOnly", sdk.RefreshOnly, "-refresh-only"},
		{"RefreshDisabled", sdk.RefreshDisabled, "-refresh=false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMacroService("terraform", nil)
			svc.Plan(context.Background(), sdk.PlanOptions{Refresh: tt.refresh})
			cmd := svc.Commands()[0].String()
			if !strings.Contains(cmd, tt.want) {
				t.Errorf("plan flags missing %q in %q", tt.want, cmd)
			}
		})
	}
}

func TestMacroService_ApplyFlagsRecorded(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	opts := sdk.ApplyOptions{
		PlanFile: "/tmp/foo.tfplan",
		VarFiles: []string{"prod.tfvars"},
	}
	svc.Apply(ctx, opts)

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1, got %d", len(cmds))
	}
	expected := "terraform apply /tmp/foo.tfplan -var-file=prod.tfvars"
	if cmds[0].String() != expected {
		t.Errorf("got %q, want %q", cmds[0].String(), expected)
	}
}

func TestMacroService_ApplyFlagsAllBranches(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	opts := sdk.ApplyOptions{
		Targets:     []string{"aws_instance.web"},
		AutoApprove: true,
		VarFiles:    []string{"prod.tfvars"},
		Vars:        map[string]string{"env": "prod"},
		Parallelism: 8,
		Lock:        sdk.LockEnabled,
		LockTimeout: sdk.LockTimeout("30s"),
		ExtraArgs:   []string{"-input=false"},
	}
	svc.Apply(ctx, opts)

	cmd := svc.Commands()[0].String()
	for _, want := range []string{
		"-target=aws_instance.web",
		"-auto-approve",
		"-var-file=prod.tfvars",
		"-var", "env=prod",
		"-parallelism=8",
		"-lock=true",
		"-lock-timeout=30s",
		"-input=false",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("apply flags missing %q in %q", want, cmd)
		}
	}
}

func TestMacroService_ApplyWithTargets_ShouldEmitTargetFlags(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	opts := sdk.ApplyOptions{
		Targets: []string{"aws_instance.web", "aws_s3_bucket.data"},
	}
	err := svc.Apply(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1, got %d", len(cmds))
	}
	expected := "terraform apply -target=aws_instance.web -target=aws_s3_bucket.data"
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

func TestMacroService_WhenSetApplyError_ShouldReturnErrorOnApply(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	expectedErr := errors.New("apply failed")
	svc.SetApplyError(expectedErr)

	err := svc.Apply(context.Background(), sdk.ApplyOptions{})
	if err == nil {
		t.Fatal("expected error from Apply")
	}
	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
}

func TestMacroService_WhenSetApplyErrorNil_ShouldReturnNilOnApply(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	svc.SetApplyError(errors.New("initial error"))
	svc.SetApplyError(nil)

	err := svc.Apply(context.Background(), sdk.ApplyOptions{})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMacroService_WhenShowWithCachedState_ShouldReturnResourceJSON(t *testing.T) {
	cache := NewServiceCache()
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Address:         "aws_instance.web",
						Type:            "aws_instance",
						Name:            "web",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"id": "i-123", "ami": "ami-abc"},
						SensitiveValues: json.RawMessage(`{}`),
					},
				},
			},
		},
	}
	cache.SetState([]sdk.Resource{{Address: "aws_instance.web"}}, state)

	svc := NewMacroService("terraform", cache)
	result, err := svc.Show(context.Background(), "aws_instance.web")
	if err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if result == "" || result == "{}" {
		t.Error("Show() returned empty/default when resource should exist")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if parsed["address"] != "aws_instance.web" {
		t.Errorf("address = %v, want aws_instance.web", parsed["address"])
	}
}

func TestMacroService_WhenShowWithCachedStateAndMissingResource_ShouldReturnError(t *testing.T) {
	cache := NewServiceCache()
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Address:         "aws_instance.web",
						Type:            "aws_instance",
						Name:            "web",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"id": "i-123"},
						SensitiveValues: json.RawMessage(`{}`),
					},
				},
			},
		},
	}
	cache.SetState([]sdk.Resource{{Address: "aws_instance.web"}}, state)

	svc := NewMacroService("terraform", cache)
	_, err := svc.Show(context.Background(), "aws_instance.nonexistent")
	if err == nil {
		t.Fatal("expected error for missing resource")
	}
}

func TestBuildInitFlags_WhenAllOptionsSet_ShouldProduceCorrectFlags(t *testing.T) {
	opts := sdk.InitOptions{
		Upgrade:       true,
		Reconfigure:   true,
		Backend:       sdk.BackendDisabled,
		BackendConfig: []string{"key=value", "region=us-east-1"},
		ExtraArgs:     []string{"-input=false"},
	}

	flags := buildInitFlags(opts)

	expected := []string{
		"-upgrade",
		"-reconfigure",
		"-backend=false",
		"-backend-config=key=value",
		"-backend-config=region=us-east-1",
		"-input=false",
	}

	if len(flags) != len(expected) {
		t.Fatalf("len(flags) = %d, want %d; flags = %v", len(flags), len(expected), flags)
	}
	for i, want := range expected {
		if flags[i] != want {
			t.Errorf("flags[%d] = %q, want %q", i, flags[i], want)
		}
	}
}

func TestBuildInitFlags_WhenBackendTrue_ShouldNotIncludeBackendFlag(t *testing.T) {
	opts := sdk.InitOptions{
		Backend: sdk.BackendEnabled,
	}

	flags := buildInitFlags(opts)
	for _, f := range flags {
		if f == "-backend=false" {
			t.Error("flags should not include -backend=false when backend is true")
		}
	}
}

func TestBuildInitFlags_WhenBackendDefault_ShouldNotIncludeBackendFlag(t *testing.T) {
	opts := sdk.InitOptions{
		Backend: sdk.BackendDefault,
	}

	flags := buildInitFlags(opts)
	for _, f := range flags {
		if f == "-backend=false" {
			t.Error("flags should not include -backend=false when backend is default")
		}
	}
}

func TestBuildInitFlags_WhenEmpty_ShouldReturnNil(t *testing.T) {
	flags := buildInitFlags(sdk.InitOptions{})
	if flags != nil {
		t.Errorf("expected nil, got %v", flags)
	}
}

func TestMacroService_WhenInitWithFlags_ShouldRecordCommand(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	opts := sdk.InitOptions{
		Upgrade:       true,
		Reconfigure:   true,
		Backend:       sdk.BackendDisabled,
		BackendConfig: []string{"key=val"},
		ExtraArgs:     []string{"-input=false"},
	}
	svc.Init(context.Background(), opts)

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	expected := "terraform init -upgrade -reconfigure -backend=false -backend-config=key=val -input=false"
	if cmds[0].String() != expected {
		t.Errorf("got %q, want %q", cmds[0].String(), expected)
	}
}

func TestMacroService_WhenVersion_ShouldReturnDefaultVersion(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	info, err := svc.Version(context.Background())
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if info == nil {
		t.Fatal("Version() returned nil")
	}
	if info.TerraformVersion != "0.0.0" {
		t.Errorf("TerraformVersion = %q, want %q", info.TerraformVersion, "0.0.0")
	}
}

func TestShowFromState_WhenNilState_ShouldReturnError(t *testing.T) {
	_, err := showFromState(nil, "aws_instance.web")
	if err == nil {
		t.Fatal("expected error for nil state")
	}
}

func TestShowFromState_WhenNilValues_ShouldReturnError(t *testing.T) {
	state := &tfjson.State{FormatVersion: "1.0", Values: nil}
	_, err := showFromState(state, "aws_instance.web")
	if err == nil {
		t.Fatal("expected error for nil Values")
	}
}

func TestShowFromState_WhenResourceNotFound_ShouldReturnError(t *testing.T) {
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{Address: "aws_instance.web", Type: "aws_instance", Name: "web"},
				},
			},
		},
	}
	_, err := showFromState(state, "aws_instance.nonexistent")
	if err == nil {
		t.Fatal("expected error for missing resource")
	}
}

func TestShowFromState_WhenResourceFound_ShouldReturnJSON(t *testing.T) {
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Address:         "aws_instance.web",
						Type:            "aws_instance",
						Name:            "web",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"id": "i-123"},
						SensitiveValues: json.RawMessage(`{}`),
					},
				},
			},
		},
	}

	result, err := showFromState(state, "aws_instance.web")
	if err != nil {
		t.Fatalf("showFromState() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed["address"] != "aws_instance.web" {
		t.Errorf("address = %v, want aws_instance.web", parsed["address"])
	}
	if parsed["type"] != "aws_instance" {
		t.Errorf("type = %v, want aws_instance", parsed["type"])
	}
	if parsed["name"] != "web" {
		t.Errorf("name = %v, want web", parsed["name"])
	}
}

func TestShowFromState_WhenResourceHasSensitiveValues_ShouldProduceValidJSON(t *testing.T) {
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Address:         "aws_instance.web",
						Type:            "aws_instance",
						Name:            "web",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"id": "i-123", "password": "secret"},
						SensitiveValues: json.RawMessage(`{"password": true}`),
					},
				},
			},
		},
	}

	result, err := showFromState(state, "aws_instance.web")
	if err != nil {
		t.Fatalf("showFromState() error = %v", err)
	}

	var parsed struct {
		Values map[string]interface{} `json:"values"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed.Values["id"] != "i-123" {
		t.Errorf("id = %v, want i-123", parsed.Values["id"])
	}
}

func TestMacroService_PlanJSON_RecordsCommand(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	ctx := context.Background()

	data, err := svc.PlanJSON(ctx, sdk.PlanOptions{Targets: []string{"aws_instance.web"}})
	if err != nil {
		t.Fatalf("PlanJSON() error = %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("PlanJSON() = %q, want %q (placeholder)", data, "{}")
	}

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].Verb != "plan" {
		t.Errorf("verb = %q, want plan", cmds[0].Verb)
	}
	got := cmds[0].String()
	if !strings.Contains(got, "-json") {
		t.Errorf("PlanJSON command missing -json: %q", got)
	}
	if !strings.Contains(got, "-target=aws_instance.web") {
		t.Errorf("PlanJSON command missing target: %q", got)
	}
}

func TestMacroService_ValidateJSON_RecordsCommand(t *testing.T) {
	svc := NewMacroService("terraform", nil)

	data, err := svc.ValidateJSON(context.Background())
	if err != nil {
		t.Fatalf("ValidateJSON() error = %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("ValidateJSON() = %q, want %q", data, "{}")
	}

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform validate -json" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform validate -json")
	}
}

func TestMacroService_OutputJSON_RecordsCommand(t *testing.T) {
	svc := NewMacroService("terraform", nil)

	data, err := svc.OutputJSON(context.Background())
	if err != nil {
		t.Fatalf("OutputJSON() error = %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("OutputJSON() = %q, want %q", data, "{}")
	}

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform output -json" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform output -json")
	}
}

func TestMacroService_VersionJSON_RecordsCommand(t *testing.T) {
	svc := NewMacroService("terraform", nil)

	data, err := svc.VersionJSON(context.Background())
	if err != nil {
		t.Fatalf("VersionJSON() error = %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("VersionJSON() = %q, want %q", data, "{}")
	}

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform version -json" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform version -json")
	}
}
