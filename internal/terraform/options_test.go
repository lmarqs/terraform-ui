package terraform

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func newRecorder() *MacroService {
	return NewMacroService("terraform", nil)
}

func TestPlanOptions_WhenEmpty_ShouldProduceNoExtraFlags(t *testing.T) {
	svc := newRecorder()
	ctx := context.Background()

	_, _ = svc.Plan(ctx, sdk.PlanOptions{})
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform plan" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform plan")
	}
}

func TestPlanOptions_WhenTargetsProvided_ShouldEmitTargetFlags(t *testing.T) {
	tests := []struct {
		name     string
		opts     sdk.PlanOptions
		expected string
	}{
		{
			"ShouldEmitSingleTarget",
			sdk.PlanOptions{Targets: []string{"aws_instance.web"}},
			"terraform plan -target=aws_instance.web",
		},
		{
			"ShouldEmitMultipleTargets",
			sdk.PlanOptions{Targets: []string{"aws_instance.web", "aws_s3_bucket.data"}},
			"terraform plan -target=aws_instance.web -target=aws_s3_bucket.data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newRecorder()
			_, _ = svc.Plan(context.Background(), tt.opts)
			cmds := svc.Commands()
			if len(cmds) != 1 {
				t.Fatalf("expected 1 command, got %d", len(cmds))
			}
			if cmds[0].String() != tt.expected {
				t.Errorf("got %q, want %q", cmds[0].String(), tt.expected)
			}
		})
	}
}

func TestPlanOptions_WhenVarFilesProvided_ShouldEmitVarFileFlags(t *testing.T) {
	tests := []struct {
		name     string
		opts     sdk.PlanOptions
		expected string
	}{
		{
			"ShouldEmitSingleVarFile",
			sdk.PlanOptions{VarFiles: []string{"dev.tfvars"}},
			"terraform plan -var-file=dev.tfvars",
		},
		{
			"ShouldEmitMultipleVarFiles",
			sdk.PlanOptions{VarFiles: []string{"common.tfvars", "env.tfvars"}},
			"terraform plan -var-file=common.tfvars -var-file=env.tfvars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newRecorder()
			_, _ = svc.Plan(context.Background(), tt.opts)
			cmds := svc.Commands()
			if len(cmds) != 1 {
				t.Fatalf("expected 1 command, got %d", len(cmds))
			}
			if cmds[0].String() != tt.expected {
				t.Errorf("got %q, want %q", cmds[0].String(), tt.expected)
			}
		})
	}
}

func TestPlanOptions_WhenVarsProvided_ShouldEmitVarFlags(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{Vars: map[string]string{"region": "us-east-1"}}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	cmdStr := cmds[0].String()
	if !strings.Contains(cmdStr, "-var") {
		t.Errorf("expected -var flag in %q", cmdStr)
	}
	if !strings.Contains(cmdStr, "region=us-east-1") {
		t.Errorf("expected region=us-east-1 in %q", cmdStr)
	}
}

func TestPlanOptions_WhenReplaceProvided_ShouldEmitReplaceFlags(t *testing.T) {
	tests := []struct {
		name     string
		opts     sdk.PlanOptions
		expected string
	}{
		{
			"ShouldEmitSingleReplace",
			sdk.PlanOptions{Replace: []string{"aws_instance.web"}},
			"terraform plan -replace=aws_instance.web",
		},
		{
			"ShouldEmitMultipleReplace",
			sdk.PlanOptions{Replace: []string{"aws_instance.web", "aws_instance.api"}},
			"terraform plan -replace=aws_instance.web -replace=aws_instance.api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newRecorder()
			_, _ = svc.Plan(context.Background(), tt.opts)
			cmds := svc.Commands()
			if len(cmds) != 1 {
				t.Fatalf("expected 1 command, got %d", len(cmds))
			}
			if cmds[0].String() != tt.expected {
				t.Errorf("got %q, want %q", cmds[0].String(), tt.expected)
			}
		})
	}
}

func TestPlanOptions_WhenDestroyTrue_ShouldEmitDestroyFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{Destroy: true}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform plan -destroy" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform plan -destroy")
	}
}

func TestPlanOptions_WhenDestroyFalse_ShouldNotEmitDestroyFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{Destroy: false}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if strings.Contains(cmds[0].String(), "-destroy") {
		t.Errorf("should not contain -destroy, got %q", cmds[0].String())
	}
}

func TestPlanOptions_WhenRefreshOnlyTrue_ShouldEmitRefreshOnlyFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{RefreshOnly: true}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform plan -refresh-only" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform plan -refresh-only")
	}
}

func TestPlanOptions_WhenRefreshSetToFalse_ShouldEmitRefreshFalseFlag(t *testing.T) {
	svc := newRecorder()
	refreshFalse := false
	opts := sdk.PlanOptions{Refresh: &refreshFalse}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform plan -refresh=false" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform plan -refresh=false")
	}
}

func TestPlanOptions_WhenRefreshNil_ShouldNotEmitRefreshFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{Refresh: nil}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if strings.Contains(cmds[0].String(), "-refresh") {
		t.Errorf("should not contain -refresh, got %q", cmds[0].String())
	}
}

func TestPlanOptions_WhenRefreshSetToTrue_ShouldNotEmitRefreshFlag(t *testing.T) {
	svc := newRecorder()
	refreshTrue := true
	opts := sdk.PlanOptions{Refresh: &refreshTrue}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if strings.Contains(cmds[0].String(), "-refresh") {
		t.Errorf("refresh=true is the default, should not emit flag; got %q", cmds[0].String())
	}
}

func TestPlanOptions_WhenParallelismSet_ShouldEmitParallelismFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{Parallelism: 5}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform plan -parallelism=5" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform plan -parallelism=5")
	}
}

func TestPlanOptions_WhenParallelismZero_ShouldNotEmitParallelismFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{Parallelism: 0}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if strings.Contains(cmds[0].String(), "-parallelism") {
		t.Errorf("should not contain -parallelism for zero value, got %q", cmds[0].String())
	}
}

func TestPlanOptions_WhenLockFalse_ShouldEmitLockFalseFlag(t *testing.T) {
	svc := newRecorder()
	lockFalse := false
	opts := sdk.PlanOptions{Lock: &lockFalse}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform plan -lock=false" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform plan -lock=false")
	}
}

func TestPlanOptions_WhenLockNil_ShouldNotEmitLockFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{Lock: nil}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if strings.Contains(cmds[0].String(), "-lock") {
		t.Errorf("should not contain -lock for nil, got %q", cmds[0].String())
	}
}

func TestPlanOptions_WhenLockTimeoutSet_ShouldEmitLockTimeoutFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{LockTimeout: "30s"}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform plan -lock-timeout=30s" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform plan -lock-timeout=30s")
	}
}

func TestPlanOptions_WhenLockTimeoutEmpty_ShouldNotEmitLockTimeoutFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{LockTimeout: ""}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if strings.Contains(cmds[0].String(), "-lock-timeout") {
		t.Errorf("should not contain -lock-timeout for empty, got %q", cmds[0].String())
	}
}

func TestPlanOptions_WhenExtraArgsProvided_ShouldAppendRaw(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{ExtraArgs: []string{"-compact-warnings", "-no-color"}}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform plan -compact-warnings -no-color" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform plan -compact-warnings -no-color")
	}
}

func TestPlanOptions_WhenAllFieldsCombined_ShouldEmitCorrectOrder(t *testing.T) {
	svc := newRecorder()
	lockFalse := false
	opts := sdk.PlanOptions{
		Targets:     []string{"aws_instance.web"},
		VarFiles:    []string{"prod.tfvars"},
		Vars:        map[string]string{"env": "prod"},
		Replace:     []string{"aws_instance.old"},
		Destroy:     true,
		Parallelism: 2,
		Lock:        &lockFalse,
		LockTimeout: "10s",
		ExtraArgs:   []string{"-no-color"},
	}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	cmdStr := cmds[0].String()

	requiredParts := []string{
		"-target=aws_instance.web",
		"-var-file=prod.tfvars",
		"-var",
		"env=prod",
		"-replace=aws_instance.old",
		"-destroy",
		"-parallelism=2",
		"-lock=false",
		"-lock-timeout=10s",
		"-no-color",
	}
	for _, part := range requiredParts {
		if !strings.Contains(cmdStr, part) {
			t.Errorf("command %q missing expected part %q", cmdStr, part)
		}
	}
	if !strings.HasPrefix(cmdStr, "terraform plan") {
		t.Errorf("command should start with 'terraform plan', got %q", cmdStr)
	}
}

func TestApplyOptions_WhenEmpty_ShouldProduceNoExtraFlags(t *testing.T) {
	svc := newRecorder()
	_ = svc.Apply(context.Background(), sdk.ApplyOptions{})
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform apply" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform apply")
	}
}

func TestApplyOptions_WhenPlanFileProvided_ShouldEmitPlanFilePositional(t *testing.T) {
	svc := newRecorder()
	_ = svc.Apply(context.Background(), sdk.ApplyOptions{PlanFile: "/tmp/foo.tfplan"})
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform apply /tmp/foo.tfplan" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform apply /tmp/foo.tfplan")
	}
}

func TestApplyOptions_WhenVarFilesProvided_ShouldEmitVarFileFlags(t *testing.T) {
	svc := newRecorder()
	opts := sdk.ApplyOptions{VarFiles: []string{"prod.tfvars", "secrets.tfvars"}}
	_ = svc.Apply(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform apply -var-file=prod.tfvars -var-file=secrets.tfvars" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform apply -var-file=prod.tfvars -var-file=secrets.tfvars")
	}
}

func TestApplyOptions_WhenVarsProvided_ShouldEmitVarFlags(t *testing.T) {
	svc := newRecorder()
	opts := sdk.ApplyOptions{Vars: map[string]string{"name": "test"}}
	_ = svc.Apply(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	cmdStr := cmds[0].String()
	if !strings.Contains(cmdStr, "-var") {
		t.Errorf("expected -var in %q", cmdStr)
	}
	if !strings.Contains(cmdStr, "name=test") {
		t.Errorf("expected name=test in %q", cmdStr)
	}
}

func TestApplyOptions_WhenParallelismSet_ShouldEmitParallelismFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.ApplyOptions{Parallelism: 10}
	_ = svc.Apply(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform apply -parallelism=10" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform apply -parallelism=10")
	}
}

func TestApplyOptions_WhenLockFalse_ShouldEmitLockFalseFlag(t *testing.T) {
	svc := newRecorder()
	lockFalse := false
	opts := sdk.ApplyOptions{Lock: &lockFalse}
	_ = svc.Apply(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform apply -lock=false" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform apply -lock=false")
	}
}

func TestApplyOptions_WhenLockTimeoutSet_ShouldEmitLockTimeoutFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.ApplyOptions{LockTimeout: "1m"}
	_ = svc.Apply(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform apply -lock-timeout=1m" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform apply -lock-timeout=1m")
	}
}

func TestApplyOptions_WhenExtraArgsProvided_ShouldAppendRaw(t *testing.T) {
	svc := newRecorder()
	opts := sdk.ApplyOptions{ExtraArgs: []string{"-no-color", "-input=false"}}
	_ = svc.Apply(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform apply -no-color -input=false" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform apply -no-color -input=false")
	}
}

func TestApplyOptions_WhenTargetsProvided_ShouldEmitTargetFlags(t *testing.T) {
	svc := newRecorder()
	opts := sdk.ApplyOptions{Targets: []string{"aws_instance.web", "aws_s3_bucket.data"}}
	_ = svc.Apply(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	expected := "terraform apply -target=aws_instance.web -target=aws_s3_bucket.data"
	if cmds[0].String() != expected {
		t.Errorf("got %q, want %q", cmds[0].String(), expected)
	}
}

func TestApplyOptions_WhenAutoApprove_ShouldEmitFlag(t *testing.T) {
	svc := newRecorder()
	opts := sdk.ApplyOptions{
		Targets:     []string{"aws_instance.web"},
		AutoApprove: true,
	}
	_ = svc.Apply(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	expected := "terraform apply -target=aws_instance.web -auto-approve"
	if cmds[0].String() != expected {
		t.Errorf("got %q, want %q", cmds[0].String(), expected)
	}
}

func TestApplyOptions_WhenAllFieldsCombined_ShouldEmitCorrectOrder(t *testing.T) {
	svc := newRecorder()
	lockFalse := false
	opts := sdk.ApplyOptions{
		PlanFile:    "/tmp/foo.tfplan",
		VarFiles:    []string{"prod.tfvars"},
		Vars:        map[string]string{"env": "prod"},
		Parallelism: 3,
		Lock:        &lockFalse,
		LockTimeout: "5s",
		ExtraArgs:   []string{"-no-color"},
	}
	_ = svc.Apply(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	cmdStr := cmds[0].String()

	requiredParts := []string{
		"/tmp/foo.tfplan",
		"-var-file=prod.tfvars",
		"-var",
		"env=prod",
		"-parallelism=3",
		"-lock=false",
		"-lock-timeout=5s",
		"-no-color",
	}
	for _, part := range requiredParts {
		if !strings.Contains(cmdStr, part) {
			t.Errorf("command %q missing expected part %q", cmdStr, part)
		}
	}
	if !strings.HasPrefix(cmdStr, "terraform apply") {
		t.Errorf("command should start with 'terraform apply', got %q", cmdStr)
	}
}

func TestApplyOptions_ShouldRecordCommand(t *testing.T) {
	svc := newRecorder()
	err := svc.Apply(context.Background(), sdk.ApplyOptions{PlanFile: "/tmp/foo.tfplan"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].Binary != "terraform" {
		t.Errorf("binary = %q, want terraform", cmds[0].Binary)
	}
	if cmds[0].Verb != "apply" {
		t.Errorf("verb = %q, want apply", cmds[0].Verb)
	}
}

func TestPlanOptions_WhenCustomBinary_ShouldUseCorrectBinary(t *testing.T) {
	svc := NewMacroService("tofu", nil)
	opts := sdk.PlanOptions{Targets: []string{"aws_instance.web"}, Destroy: true}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	cmdStr := cmds[0].String()
	if !strings.HasPrefix(cmdStr, "tofu plan") {
		t.Errorf("expected tofu prefix, got %q", cmdStr)
	}
	if !strings.Contains(cmdStr, "-target=aws_instance.web") {
		t.Errorf("missing target flag in %q", cmdStr)
	}
	if !strings.Contains(cmdStr, "-destroy") {
		t.Errorf("missing -destroy flag in %q", cmdStr)
	}
}

func TestPlanOptions_WhenOnlyTargets_ShouldBeBackwardCompatible(t *testing.T) {
	svc := newRecorder()
	opts := sdk.PlanOptions{Targets: []string{"aws_instance.web", "aws_s3_bucket.logs"}}
	_, _ = svc.Plan(context.Background(), opts)
	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].String() != "terraform plan -target=aws_instance.web -target=aws_s3_bucket.logs" {
		t.Errorf("got %q, want %q", cmds[0].String(), "terraform plan -target=aws_instance.web -target=aws_s3_bucket.logs")
	}
}

func TestPlanOptions_WhenPreLoadedPlan_ShouldReturnPlanData(t *testing.T) {
	plan := &sdk.PlanSummary{
		Changes:  []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate}},
		ToCreate: 1,
	}
	cache := NewServiceCache()
	cache.SetPlan(plan)
	svc := NewMacroService("terraform", cache)
	got, err := svc.Plan(context.Background(), sdk.PlanOptions{Destroy: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.ToCreate != 1 {
		t.Errorf("ToCreate = %d, want 1", got.ToCreate)
	}
	if len(got.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(got.Changes))
	}
}

func TestPlanOptions_WhenPlanFileProvided_ShouldEmitOutFlag(t *testing.T) {
	planPath := filepath.Join(t.TempDir(), "tfui.tfplan")
	svc := newRecorder()
	_, _ = svc.Plan(context.Background(), sdk.PlanOptions{PlanFile: planPath})

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	want := "terraform plan -out=" + planPath
	if cmds[0].String() != want {
		t.Errorf("got %q, want %q", cmds[0].String(), want)
	}
}

func TestPlanOptions_WhenPlanFileEmpty_ShouldOmitOutFlag(t *testing.T) {
	svc := newRecorder()
	_, _ = svc.Plan(context.Background(), sdk.PlanOptions{})

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if strings.Contains(cmds[0].String(), "-out=") {
		t.Errorf("got %q, want no -out= flag when PlanFile is empty", cmds[0].String())
	}
}

func TestServiceInterface_WhenPlanOptionsSignature_ShouldCompile(t *testing.T) {
	var svc sdk.Service = newRecorder()
	_, _ = svc.Plan(context.Background(), sdk.PlanOptions{})
}

func TestServiceInterface_WhenApplyOptionsSignature_ShouldCompile(t *testing.T) {
	var svc sdk.Service = newRecorder()
	_ = svc.Apply(context.Background(), sdk.ApplyOptions{})
}
