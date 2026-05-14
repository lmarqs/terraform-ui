package terraform

import (
	"context"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestTerraformService_WhenNewTerraformFails_ShouldWrapErrorWithOperationContext(t *testing.T) {
	svc := NewExecService(t.TempDir(), "", nil)
	ctx := context.Background()

	tests := []struct {
		name    string
		call    func() error
		wantMsg string
	}{
		{
			"ShouldWrapPlanError",
			func() error { _, err := svc.Plan(ctx, sdk.PlanOptions{}); return err },
			"planning",
		},
		{
			"ShouldWrapApplyError",
			func() error { return svc.Apply(ctx, sdk.ApplyOptions{}) },
			"applying",
		},
		{
			"ShouldWrapWorkspaceError",
			func() error { _, err := svc.Workspace(ctx); return err },
			"getting workspace",
		},
		{
			"ShouldWrapWorkspaceListError",
			func() error { _, err := svc.WorkspaceList(ctx); return err },
			"listing workspaces",
		},
		{
			"ShouldWrapWorkspaceSelectError",
			func() error { return svc.WorkspaceSelect(ctx, "dev") },
			"selecting workspace",
		},
		{
			"ShouldWrapWorkspaceNewError",
			func() error { return svc.WorkspaceNew(ctx, "new-ws") },
			"creating workspace",
		},
		{
			"ShouldWrapWorkspaceDeleteError",
			func() error { return svc.WorkspaceDelete(ctx, "old-ws") },
			"deleting workspace",
		},
		{
			"ShouldWrapStateRmError",
			func() error { return svc.StateRm(ctx, "aws_instance.web") },
			"removing from state",
		},
		{
			"ShouldWrapStateMoveError",
			func() error { return svc.StateMove(ctx, "aws_instance.old", "aws_instance.new") },
			"moving state",
		},
		{
			"ShouldWrapImportError",
			func() error { return svc.Import(ctx, "aws_instance.web", "i-12345") },
			"importing resource",
		},
		{
			"ShouldWrapTaintError",
			func() error { return svc.Taint(ctx, "aws_instance.web") },
			"tainting resource",
		},
		{
			"ShouldWrapUntaintError",
			func() error { return svc.Untaint(ctx, "aws_instance.web") },
			"untainting resource",
		},
		{
			"ShouldWrapValidateError",
			func() error { _, err := svc.Validate(ctx); return err },
			"validating",
		},
		{
			"ShouldWrapOutputError",
			func() error { _, err := svc.Output(ctx); return err },
			"getting output",
		},
		{
			"ShouldWrapRefreshError",
			func() error { return svc.Refresh(ctx) },
			"refreshing state",
		},
		{
			"ShouldWrapInitError",
			func() error { return svc.Init(ctx) },
			"initializing",
		},
		{
			"ShouldWrapForceUnlockError",
			func() error { return svc.ForceUnlock(ctx, "lock-id-123") },
			"force-unlocking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantMsg)
			}
			if !strings.Contains(err.Error(), "creating terraform instance") {
				t.Errorf("error %q does not preserve underlying cause", err.Error())
			}
		})
	}
}
