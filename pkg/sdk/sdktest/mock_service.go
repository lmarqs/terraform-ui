package sdktest

import (
	"context"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// MockService is a shared test double for sdk.Service.
// Set function fields to override behavior; unset methods return zero values.
// Call-tracking fields are populated automatically by each method.
type MockService struct {
	PlanFn            func(ctx context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error)
	ApplyFn           func(ctx context.Context, opts sdk.ApplyOptions) error
	StateListFn       func(ctx context.Context, opts ...sdk.StateListOption) ([]sdk.Resource, error)
	ShowFn            func(ctx context.Context, address string) (string, error)
	WorkspaceFn       func(ctx context.Context) (string, error)
	WorkspaceListFn   func(ctx context.Context) ([]string, error)
	WorkspaceSelectFn func(ctx context.Context, name string) error
	WorkspaceNewFn    func(ctx context.Context, name string, opts sdk.WorkspaceNewOptions) error
	WorkspaceDeleteFn func(ctx context.Context, name string, opts sdk.WorkspaceDeleteOptions) error
	StateRmFn         func(ctx context.Context, address string) error
	StateMoveFn       func(ctx context.Context, source, dest string) error
	ImportFn          func(ctx context.Context, address, id string) error
	TaintFn           func(ctx context.Context, address string) error
	UntaintFn         func(ctx context.Context, address string) error
	ValidateFn        func(ctx context.Context) ([]sdk.Diagnostic, error)
	OutputFn          func(ctx context.Context) (map[string]sdk.OutputValue, error)
	RefreshFn         func(ctx context.Context) error
	InitFn            func(ctx context.Context, opts sdk.InitOptions) error
	ForceUnlockFn     func(ctx context.Context, lockID string) error
	VersionFn         func(ctx context.Context) (*sdk.VersionInfo, error)
	WithDirFn         func(dir string) sdk.Service

	PlanCalls            []sdk.PlanOptions
	ApplyCalls           []sdk.ApplyOptions
	StateListCalls       int
	ShowCalls            []string
	WorkspaceCalls       int
	WorkspaceListCalls   int
	WorkspaceSelectCalls []string
	WorkspaceNewCalls    []string
	WorkspaceDeleteCalls []string
	StateRmCalls         []string
	StateMoveCalls       [][2]string
	ImportCalls          [][2]string
	TaintCalls           []string
	UntaintCalls         []string
	ValidateCalls        int
	OutputCalls          int
	RefreshCalls         int
	InitCalls            []sdk.InitOptions
	ForceUnlockCalls     []string
	VersionCalls         int
	WithDirCalls         []string
}

func (m *MockService) Plan(ctx context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
	m.PlanCalls = append(m.PlanCalls, opts)
	if m.PlanFn != nil {
		return m.PlanFn(ctx, opts)
	}
	return nil, nil
}

func (m *MockService) Apply(ctx context.Context, opts sdk.ApplyOptions) error {
	m.ApplyCalls = append(m.ApplyCalls, opts)
	if m.ApplyFn != nil {
		return m.ApplyFn(ctx, opts)
	}
	return nil
}

func (m *MockService) StateList(ctx context.Context, opts ...sdk.StateListOption) ([]sdk.Resource, error) {
	m.StateListCalls++
	if m.StateListFn != nil {
		return m.StateListFn(ctx, opts...)
	}
	return nil, nil
}

func (m *MockService) Show(ctx context.Context, address string) (string, error) {
	m.ShowCalls = append(m.ShowCalls, address)
	if m.ShowFn != nil {
		return m.ShowFn(ctx, address)
	}
	return "", nil
}

func (m *MockService) Workspace(ctx context.Context) (string, error) {
	m.WorkspaceCalls++
	if m.WorkspaceFn != nil {
		return m.WorkspaceFn(ctx)
	}
	return "default", nil
}

func (m *MockService) WorkspaceList(ctx context.Context) ([]string, error) {
	m.WorkspaceListCalls++
	if m.WorkspaceListFn != nil {
		return m.WorkspaceListFn(ctx)
	}
	return nil, nil
}

func (m *MockService) WorkspaceSelect(ctx context.Context, name string) error {
	m.WorkspaceSelectCalls = append(m.WorkspaceSelectCalls, name)
	if m.WorkspaceSelectFn != nil {
		return m.WorkspaceSelectFn(ctx, name)
	}
	return nil
}

func (m *MockService) WorkspaceNew(ctx context.Context, name string, opts sdk.WorkspaceNewOptions) error {
	m.WorkspaceNewCalls = append(m.WorkspaceNewCalls, name)
	if m.WorkspaceNewFn != nil {
		return m.WorkspaceNewFn(ctx, name, opts)
	}
	return nil
}

func (m *MockService) WorkspaceDelete(ctx context.Context, name string, opts sdk.WorkspaceDeleteOptions) error {
	m.WorkspaceDeleteCalls = append(m.WorkspaceDeleteCalls, name)
	if m.WorkspaceDeleteFn != nil {
		return m.WorkspaceDeleteFn(ctx, name, opts)
	}
	return nil
}

func (m *MockService) StateRm(ctx context.Context, address string) error {
	m.StateRmCalls = append(m.StateRmCalls, address)
	if m.StateRmFn != nil {
		return m.StateRmFn(ctx, address)
	}
	return nil
}

func (m *MockService) StateMove(ctx context.Context, source, dest string) error {
	m.StateMoveCalls = append(m.StateMoveCalls, [2]string{source, dest})
	if m.StateMoveFn != nil {
		return m.StateMoveFn(ctx, source, dest)
	}
	return nil
}

func (m *MockService) Import(ctx context.Context, address, id string) error {
	m.ImportCalls = append(m.ImportCalls, [2]string{address, id})
	if m.ImportFn != nil {
		return m.ImportFn(ctx, address, id)
	}
	return nil
}

func (m *MockService) Taint(ctx context.Context, address string) error {
	m.TaintCalls = append(m.TaintCalls, address)
	if m.TaintFn != nil {
		return m.TaintFn(ctx, address)
	}
	return nil
}

func (m *MockService) Untaint(ctx context.Context, address string) error {
	m.UntaintCalls = append(m.UntaintCalls, address)
	if m.UntaintFn != nil {
		return m.UntaintFn(ctx, address)
	}
	return nil
}

func (m *MockService) Validate(ctx context.Context) ([]sdk.Diagnostic, error) {
	m.ValidateCalls++
	if m.ValidateFn != nil {
		return m.ValidateFn(ctx)
	}
	return nil, nil
}

func (m *MockService) Output(ctx context.Context) (map[string]sdk.OutputValue, error) {
	m.OutputCalls++
	if m.OutputFn != nil {
		return m.OutputFn(ctx)
	}
	return nil, nil
}

func (m *MockService) Refresh(ctx context.Context) error {
	m.RefreshCalls++
	if m.RefreshFn != nil {
		return m.RefreshFn(ctx)
	}
	return nil
}

func (m *MockService) Init(ctx context.Context, opts sdk.InitOptions) error {
	m.InitCalls = append(m.InitCalls, opts)
	if m.InitFn != nil {
		return m.InitFn(ctx, opts)
	}
	return nil
}

func (m *MockService) ForceUnlock(ctx context.Context, lockID string) error {
	m.ForceUnlockCalls = append(m.ForceUnlockCalls, lockID)
	if m.ForceUnlockFn != nil {
		return m.ForceUnlockFn(ctx, lockID)
	}
	return nil
}

func (m *MockService) Version(ctx context.Context) (*sdk.VersionInfo, error) {
	m.VersionCalls++
	if m.VersionFn != nil {
		return m.VersionFn(ctx)
	}
	return nil, nil
}

func (m *MockService) WithDir(dir string) sdk.Service {
	m.WithDirCalls = append(m.WithDirCalls, dir)
	if m.WithDirFn != nil {
		return m.WithDirFn(dir)
	}
	return m
}
