package sdktest

import (
	"context"
	"errors"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

var _ sdk.Service = (*MockService)(nil)

func TestMockService_WhenNoFnSet_ShouldReturnDefaults(t *testing.T) {
	m := &MockService{}
	ctx := context.Background()

	summary, err := m.Plan(ctx, sdk.PlanOptions{})
	if summary != nil || err != nil {
		t.Errorf("Plan() = (%v, %v), want (nil, nil)", summary, err)
	}

	if err := m.Apply(ctx, sdk.ApplyOptions{}); err != nil {
		t.Errorf("Apply() = %v, want nil", err)
	}

	resources, err := m.StateList(ctx)
	if resources != nil || err != nil {
		t.Errorf("StateList() = (%v, %v), want (nil, nil)", resources, err)
	}

	show, err := m.Show(ctx, "addr")
	if show != "" || err != nil {
		t.Errorf("Show() = (%q, %v), want (\"\", nil)", show, err)
	}

	ws, err := m.Workspace(ctx)
	if ws != "default" || err != nil {
		t.Errorf("Workspace() = (%q, %v), want (\"default\", nil)", ws, err)
	}

	list, err := m.WorkspaceList(ctx)
	if list != nil || err != nil {
		t.Errorf("WorkspaceList() = (%v, %v), want (nil, nil)", list, err)
	}

	if err := m.WorkspaceSelect(ctx, "prod"); err != nil {
		t.Errorf("WorkspaceSelect() = %v, want nil", err)
	}

	if err := m.WorkspaceNew(ctx, "dev", sdk.WorkspaceNewOptions{}); err != nil {
		t.Errorf("WorkspaceNew() = %v, want nil", err)
	}

	if err := m.WorkspaceDelete(ctx, "old", sdk.WorkspaceDeleteOptions{}); err != nil {
		t.Errorf("WorkspaceDelete() = %v, want nil", err)
	}

	if err := m.StateRm(ctx, "addr"); err != nil {
		t.Errorf("StateRm() = %v, want nil", err)
	}

	if err := m.StateMove(ctx, "a", "b"); err != nil {
		t.Errorf("StateMove() = %v, want nil", err)
	}

	if err := m.Import(ctx, "addr", "id"); err != nil {
		t.Errorf("Import() = %v, want nil", err)
	}

	if err := m.Taint(ctx, "addr"); err != nil {
		t.Errorf("Taint() = %v, want nil", err)
	}

	if err := m.Untaint(ctx, "addr"); err != nil {
		t.Errorf("Untaint() = %v, want nil", err)
	}

	diags, err := m.Validate(ctx)
	if diags != nil || err != nil {
		t.Errorf("Validate() = (%v, %v), want (nil, nil)", diags, err)
	}

	outputs, err := m.Output(ctx)
	if outputs != nil || err != nil {
		t.Errorf("Output() = (%v, %v), want (nil, nil)", outputs, err)
	}

	if err := m.Refresh(ctx); err != nil {
		t.Errorf("Refresh() = %v, want nil", err)
	}

	if err := m.Init(ctx, sdk.InitOptions{}); err != nil {
		t.Errorf("Init() = %v, want nil", err)
	}

	if err := m.ForceUnlock(ctx, "lock-id"); err != nil {
		t.Errorf("ForceUnlock() = %v, want nil", err)
	}

	info, err := m.Version(ctx)
	if info != nil || err != nil {
		t.Errorf("Version() = (%v, %v), want (nil, nil)", info, err)
	}

	svc := m.WithDir("/tmp")
	if svc != m {
		t.Error("WithDir() should return self by default")
	}
}

func TestMockService_WhenFnSet_ShouldDelegateToFn(t *testing.T) {
	expectedErr := errors.New("planned")
	m := &MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return &sdk.PlanSummary{ToCreate: 1}, expectedErr
		},
		TaintFn: func(_ context.Context, addr string) error {
			if addr == "x" {
				return errors.New("no")
			}
			return nil
		},
		WithDirFn: func(dir string) sdk.Service {
			return &MockService{}
		},
	}
	ctx := context.Background()

	summary, err := m.Plan(ctx, sdk.PlanOptions{})
	if summary == nil || summary.ToCreate != 1 {
		t.Errorf("Plan() summary = %v, want ToCreate=1", summary)
	}
	if err != expectedErr {
		t.Errorf("Plan() err = %v, want %v", err, expectedErr)
	}

	if err := m.Taint(ctx, "x"); err == nil {
		t.Error("Taint(x) should return error")
	}
	if err := m.Taint(ctx, "y"); err != nil {
		t.Errorf("Taint(y) = %v, want nil", err)
	}

	svc := m.WithDir("/other")
	if svc == m {
		t.Error("WithDir with custom fn should return different service")
	}
}

func TestMockService_WhenAllFnsSet_ShouldDelegateEveryMethod(t *testing.T) {
	e := errors.New("err")
	m := &MockService{
		ApplyFn:           func(_ context.Context, _ sdk.ApplyOptions) error { return e },
		StateListFn:       func(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) { return nil, e },
		ShowFn:            func(_ context.Context, _ string) (string, error) { return "x", e },
		WorkspaceFn:       func(_ context.Context) (string, error) { return "ws", e },
		WorkspaceListFn:   func(_ context.Context) ([]string, error) { return []string{"a"}, e },
		WorkspaceSelectFn: func(_ context.Context, _ string) error { return e },
		WorkspaceNewFn:    func(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error { return e },
		WorkspaceDeleteFn: func(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error { return e },
		StateRmFn:         func(_ context.Context, _ string) error { return e },
		StateMoveFn:       func(_ context.Context, _, _ string) error { return e },
		ImportFn:          func(_ context.Context, _, _ string) error { return e },
		UntaintFn:         func(_ context.Context, _ string) error { return e },
		ValidateFn:        func(_ context.Context) ([]sdk.Diagnostic, error) { return nil, e },
		OutputFn:          func(_ context.Context) (map[string]sdk.OutputValue, error) { return nil, e },
		RefreshFn:         func(_ context.Context) error { return e },
		InitFn:            func(_ context.Context, _ sdk.InitOptions) error { return e },
		ForceUnlockFn:     func(_ context.Context, _ string) error { return e },
		VersionFn:         func(_ context.Context) (*sdk.VersionInfo, error) { return nil, e },
	}
	ctx := context.Background()

	if err := m.Apply(ctx, sdk.ApplyOptions{}); err != e {
		t.Errorf("Apply() = %v, want %v", err, e)
	}
	if _, err := m.StateList(ctx); err != e {
		t.Errorf("StateList() err = %v, want %v", err, e)
	}
	if s, _ := m.Show(ctx, "a"); s != "x" {
		t.Errorf("Show() = %q, want %q", s, "x")
	}
	if ws, _ := m.Workspace(ctx); ws != "ws" {
		t.Errorf("Workspace() = %q, want %q", ws, "ws")
	}
	if list, _ := m.WorkspaceList(ctx); len(list) != 1 {
		t.Errorf("WorkspaceList() len = %d, want 1", len(list))
	}
	if err := m.WorkspaceSelect(ctx, "x"); err != e {
		t.Errorf("WorkspaceSelect() = %v, want %v", err, e)
	}
	if err := m.WorkspaceNew(ctx, "x", sdk.WorkspaceNewOptions{}); err != e {
		t.Errorf("WorkspaceNew() = %v, want %v", err, e)
	}
	if err := m.WorkspaceDelete(ctx, "x", sdk.WorkspaceDeleteOptions{}); err != e {
		t.Errorf("WorkspaceDelete() = %v, want %v", err, e)
	}
	if err := m.StateRm(ctx, "a"); err != e {
		t.Errorf("StateRm() = %v, want %v", err, e)
	}
	if err := m.StateMove(ctx, "a", "b"); err != e {
		t.Errorf("StateMove() = %v, want %v", err, e)
	}
	if err := m.Import(ctx, "a", "b"); err != e {
		t.Errorf("Import() = %v, want %v", err, e)
	}
	if err := m.Untaint(ctx, "a"); err != e {
		t.Errorf("Untaint() = %v, want %v", err, e)
	}
	if _, err := m.Validate(ctx); err != e {
		t.Errorf("Validate() err = %v, want %v", err, e)
	}
	if _, err := m.Output(ctx); err != e {
		t.Errorf("Output() err = %v, want %v", err, e)
	}
	if err := m.Refresh(ctx); err != e {
		t.Errorf("Refresh() = %v, want %v", err, e)
	}
	if err := m.Init(ctx, sdk.InitOptions{}); err != e {
		t.Errorf("Init() = %v, want %v", err, e)
	}
	if err := m.ForceUnlock(ctx, "x"); err != e {
		t.Errorf("ForceUnlock() = %v, want %v", err, e)
	}
	if _, err := m.Version(ctx); err != e {
		t.Errorf("Version() err = %v, want %v", err, e)
	}
}

func TestMockService_ShouldTrackCalls(t *testing.T) {
	m := &MockService{}
	ctx := context.Background()

	m.Plan(ctx, sdk.PlanOptions{Targets: []string{"a"}})
	m.Plan(ctx, sdk.PlanOptions{Targets: []string{"b"}})
	if len(m.PlanCalls) != 2 {
		t.Errorf("PlanCalls = %d, want 2", len(m.PlanCalls))
	}

	m.Taint(ctx, "res1")
	m.Taint(ctx, "res2")
	if len(m.TaintCalls) != 2 || m.TaintCalls[0] != "res1" || m.TaintCalls[1] != "res2" {
		t.Errorf("TaintCalls = %v, want [res1, res2]", m.TaintCalls)
	}

	m.StateMove(ctx, "a", "b")
	if len(m.StateMoveCalls) != 1 || m.StateMoveCalls[0] != [2]string{"a", "b"} {
		t.Errorf("StateMoveCalls = %v, want [[a b]]", m.StateMoveCalls)
	}

	m.Import(ctx, "addr", "id")
	if len(m.ImportCalls) != 1 || m.ImportCalls[0] != [2]string{"addr", "id"} {
		t.Errorf("ImportCalls = %v, want [[addr id]]", m.ImportCalls)
	}

	m.WithDir("/path")
	if len(m.WithDirCalls) != 1 || m.WithDirCalls[0] != "/path" {
		t.Errorf("WithDirCalls = %v, want [/path]", m.WithDirCalls)
	}

	m.Workspace(ctx)
	m.Workspace(ctx)
	if m.WorkspaceCalls != 2 {
		t.Errorf("WorkspaceCalls = %d, want 2", m.WorkspaceCalls)
	}

	m.ForceUnlock(ctx, "lock-1")
	if len(m.ForceUnlockCalls) != 1 || m.ForceUnlockCalls[0] != "lock-1" {
		t.Errorf("ForceUnlockCalls = %v, want [lock-1]", m.ForceUnlockCalls)
	}
}
