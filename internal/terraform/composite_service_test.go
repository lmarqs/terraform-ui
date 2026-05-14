package terraform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const testPlanJSON = `{"format_version":"1.2","terraform_version":"1.5.0","planned_values":{},"resource_changes":[{"address":"local_file.test","type":"local_file","name":"test","provider_name":"registry.terraform.io/hashicorp/local","change":{"actions":["create"],"before":null,"after":{"content":"hello","filename":"test.txt"},"after_unknown":{}}}]}`

const testStateJSON = `{"format_version":"1.0","terraform_version":"1.5.0","values":{"root_module":{"resources":[{"address":"local_file.foo","type":"local_file","name":"foo","provider_name":"registry.terraform.io/hashicorp/local","values":{"content":"bar","filename":"foo.txt"},"sensitive_values":{}}]}}}`

type mockLiveService struct {
	planResult        *sdk.PlanSummary
	planErr           error
	planCalled        bool
	planOpts          sdk.PlanOptions
	applyErr          error
	applyCalled       bool
	applyOpts         sdk.ApplyOptions
	stateListResult   []sdk.Resource
	stateListErr      error
	stateListCalled   bool
	showResult        string
	showErr           error
	showCalled        bool
	showAddress       string
	refreshErr        error
	refreshCalled     bool
	workspaceResult   string
	workspaceErr      error
	workspaceCalled   bool
	wsListResult      []string
	wsListErr         error
	wsListCalled      bool
	wsSelectCalled    bool
	wsSelectName      string
	wsSelectErr       error
	wsNewCalled       bool
	wsNewName         string
	wsNewErr          error
	wsDeleteCalled    bool
	wsDeleteName      string
	wsDeleteErr       error
	stateRmCalled     bool
	stateRmAddress    string
	stateRmErr        error
	stateMoveCalled   bool
	stateMoveSource   string
	stateMoveDest     string
	stateMoveErr      error
	importCalled      bool
	importAddress     string
	importID          string
	importErr         error
	taintCalled       bool
	taintAddress      string
	taintErr          error
	untaintCalled     bool
	untaintAddress    string
	untaintErr        error
	validateResult    []sdk.Diagnostic
	validateErr       error
	validateCalled    bool
	outputResult      map[string]sdk.OutputValue
	outputErr         error
	outputCalled      bool
	initCalled        bool
	initErr           error
	forceUnlockID     string
	forceUnlockErr    error
	forceUnlockCalled bool
	withDirPath       string
	withDirResult     sdk.Service
}

func (m *mockLiveService) Plan(_ context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
	m.planCalled = true
	m.planOpts = opts
	if m.planResult == nil && m.planErr == nil {
		return &sdk.PlanSummary{Changes: []sdk.PlanChange{}}, nil
	}
	return m.planResult, m.planErr
}

func (m *mockLiveService) Apply(_ context.Context, opts sdk.ApplyOptions) error {
	m.applyCalled = true
	m.applyOpts = opts
	return m.applyErr
}

func (m *mockLiveService) StateList(_ context.Context) ([]sdk.Resource, error) {
	m.stateListCalled = true
	return m.stateListResult, m.stateListErr
}

func (m *mockLiveService) Show(_ context.Context, address string) (string, error) {
	m.showCalled = true
	m.showAddress = address
	return m.showResult, m.showErr
}

func (m *mockLiveService) Workspace(_ context.Context) (string, error) {
	m.workspaceCalled = true
	return m.workspaceResult, m.workspaceErr
}

func (m *mockLiveService) WorkspaceList(_ context.Context) ([]string, error) {
	m.wsListCalled = true
	return m.wsListResult, m.wsListErr
}

func (m *mockLiveService) WorkspaceSelect(_ context.Context, name string) error {
	m.wsSelectCalled = true
	m.wsSelectName = name
	return m.wsSelectErr
}

func (m *mockLiveService) WorkspaceNew(_ context.Context, name string) error {
	m.wsNewCalled = true
	m.wsNewName = name
	return m.wsNewErr
}

func (m *mockLiveService) WorkspaceDelete(_ context.Context, name string) error {
	m.wsDeleteCalled = true
	m.wsDeleteName = name
	return m.wsDeleteErr
}

func (m *mockLiveService) StateRm(_ context.Context, address string) error {
	m.stateRmCalled = true
	m.stateRmAddress = address
	return m.stateRmErr
}

func (m *mockLiveService) StateMove(_ context.Context, source, dest string) error {
	m.stateMoveCalled = true
	m.stateMoveSource = source
	m.stateMoveDest = dest
	return m.stateMoveErr
}

func (m *mockLiveService) Import(_ context.Context, address, id string) error {
	m.importCalled = true
	m.importAddress = address
	m.importID = id
	return m.importErr
}

func (m *mockLiveService) Taint(_ context.Context, address string) error {
	m.taintCalled = true
	m.taintAddress = address
	return m.taintErr
}

func (m *mockLiveService) Untaint(_ context.Context, address string) error {
	m.untaintCalled = true
	m.untaintAddress = address
	return m.untaintErr
}

func (m *mockLiveService) Validate(_ context.Context) ([]sdk.Diagnostic, error) {
	m.validateCalled = true
	return m.validateResult, m.validateErr
}

func (m *mockLiveService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	m.outputCalled = true
	return m.outputResult, m.outputErr
}

func (m *mockLiveService) Refresh(_ context.Context) error {
	m.refreshCalled = true
	return m.refreshErr
}

func (m *mockLiveService) Init(_ context.Context) error {
	m.initCalled = true
	return m.initErr
}

func (m *mockLiveService) ForceUnlock(_ context.Context, lockID string) error {
	m.forceUnlockCalled = true
	m.forceUnlockID = lockID
	return m.forceUnlockErr
}

func (m *mockLiveService) WithDir(dir string) sdk.Service {
	m.withDirPath = dir
	if m.withDirResult != nil {
		return m.withDirResult
	}
	return &mockLiveService{
		planResult:      m.planResult,
		stateListResult: m.stateListResult,
		showResult:      m.showResult,
		workspaceResult: m.workspaceResult,
		wsListResult:    m.wsListResult,
	}
}

func TestCompositeService_ImplementsInterface(t *testing.T) {
	var _ sdk.Service = (*CompositeService)(nil)
}

func TestComposite_Plan_FromFile(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planPath, []byte(testPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, planFile: planPath}

	summary, err := svc.Plan(context.Background(), sdk.PlanOptions{})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if summary.ToCreate != 1 {
		t.Errorf("ToCreate = %d, want 1", summary.ToCreate)
	}
	if summary.Changes[0].Resource.Address != "local_file.test" {
		t.Errorf("address = %q, want %q", summary.Changes[0].Resource.Address, "local_file.test")
	}
	if mock.planCalled {
		t.Error("live.Plan() should not be called when planFile is set")
	}
}

func TestComposite_Plan_FromStdin(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stdinPlan: []byte(testPlanJSON)}

	summary, err := svc.Plan(context.Background(), sdk.PlanOptions{})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if summary.ToCreate != 1 {
		t.Errorf("ToCreate = %d, want 1", summary.ToCreate)
	}
	if mock.planCalled {
		t.Error("live.Plan() should not be called when stdinPlan is set")
	}
}

func TestComposite_Plan_LiveFallback(t *testing.T) {
	expected := &sdk.PlanSummary{
		Changes:  []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.live"}}},
		ToCreate: 1,
	}
	mock := &mockLiveService{planResult: expected}
	svc := &CompositeService{live: mock}

	opts := sdk.PlanOptions{Targets: []string{"aws_instance.live"}}
	summary, err := svc.Plan(context.Background(), opts)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if !mock.planCalled {
		t.Fatal("live.Plan() should be called")
	}
	if mock.planOpts.Targets[0] != "aws_instance.live" {
		t.Errorf("opts not passed through")
	}
	if summary != expected {
		t.Error("should return live result directly")
	}
}

func TestComposite_Plan_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "bad.json")
	os.WriteFile(planPath, []byte("not json"), 0o644)

	svc := &CompositeService{live: &mockLiveService{}, planFile: planPath}
	_, err := svc.Plan(context.Background(), sdk.PlanOptions{})
	if err == nil {
		t.Fatal("should return error for invalid JSON")
	}
}

func TestComposite_Plan_LiveError(t *testing.T) {
	expected := errors.New("plan failed")
	mock := &mockLiveService{planErr: expected}
	svc := &CompositeService{live: mock}

	_, err := svc.Plan(context.Background(), sdk.PlanOptions{})
	if !errors.Is(err, expected) {
		t.Errorf("error = %v, want %v", err, expected)
	}
}

func TestComposite_StateList_FromFile(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	os.WriteFile(statePath, []byte(testStateJSON), 0o644)

	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stateFile: statePath}

	resources, err := svc.StateList(context.Background())
	if err != nil {
		t.Fatalf("StateList() error = %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("len = %d, want 1", len(resources))
	}
	if resources[0].Address != "local_file.foo" {
		t.Errorf("address = %q, want %q", resources[0].Address, "local_file.foo")
	}
	if mock.stateListCalled {
		t.Error("live.StateList() should not be called")
	}
}

func TestComposite_StateList_FromStdin(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stdinState: []byte(testStateJSON)}

	resources, err := svc.StateList(context.Background())
	if err != nil {
		t.Fatalf("StateList() error = %v", err)
	}
	if len(resources) != 1 || resources[0].Address != "local_file.foo" {
		t.Errorf("unexpected resources = %v", resources)
	}
	if mock.stateListCalled {
		t.Error("live.StateList() should not be called")
	}
}

func TestComposite_StateList_LiveFallback(t *testing.T) {
	expected := []sdk.Resource{{Address: "aws_instance.live"}}
	mock := &mockLiveService{stateListResult: expected}
	svc := &CompositeService{live: mock}

	resources, err := svc.StateList(context.Background())
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !mock.stateListCalled {
		t.Fatal("live.StateList() should be called")
	}
	if resources[0].Address != "aws_instance.live" {
		t.Errorf("address = %q", resources[0].Address)
	}
}

func TestComposite_Show_FromFile(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	os.WriteFile(statePath, []byte(testStateJSON), 0o644)

	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stateFile: statePath}

	result, err := svc.Show(context.Background(), "local_file.foo")
	if err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if result == "" {
		t.Fatal("Show() returned empty")
	}
	if mock.showCalled {
		t.Error("live.Show() should not be called")
	}
}

func TestComposite_Show_NotFound(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	os.WriteFile(statePath, []byte(testStateJSON), 0o644)

	svc := &CompositeService{live: &mockLiveService{}, stateFile: statePath}
	_, err := svc.Show(context.Background(), "nonexistent.addr")
	if err == nil {
		t.Fatal("should error for missing address")
	}
}

func TestComposite_Show_LiveFallback(t *testing.T) {
	mock := &mockLiveService{showResult: `{"address":"aws_instance.web"}`}
	svc := &CompositeService{live: mock}

	result, err := svc.Show(context.Background(), "aws_instance.web")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !mock.showCalled {
		t.Fatal("live.Show() should be called")
	}
	if result != `{"address":"aws_instance.web"}` {
		t.Errorf("result = %q", result)
	}
}

func TestComposite_Apply_AlwaysLive(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, planFile: "/some/plan.out"}

	err := svc.Apply(context.Background(), sdk.ApplyOptions{Targets: []string{"x"}})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !mock.applyCalled {
		t.Fatal("live.Apply() should always be called")
	}
	if mock.applyOpts.Targets[0] != "x" {
		t.Error("opts not passed through")
	}
}

func TestComposite_StateRm_DelegatesToLive(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stateFile: "/state.json"}

	svc.StateRm(context.Background(), "local_file.foo")
	if !mock.stateRmCalled || mock.stateRmAddress != "local_file.foo" {
		t.Error("should delegate with correct address")
	}
}

func TestComposite_StateMove_DelegatesToLive(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stateFile: "/state.json"}

	svc.StateMove(context.Background(), "old", "new")
	if !mock.stateMoveCalled || mock.stateMoveSource != "old" || mock.stateMoveDest != "new" {
		t.Error("should delegate with correct args")
	}
}

func TestComposite_Import_DelegatesToLive(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stateFile: "/state.json"}

	svc.Import(context.Background(), "addr", "id-123")
	if !mock.importCalled || mock.importAddress != "addr" || mock.importID != "id-123" {
		t.Error("should delegate with correct args")
	}
}

func TestComposite_Taint_DelegatesToLive(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stateFile: "/state.json"}

	svc.Taint(context.Background(), "aws_instance.web")
	if !mock.taintCalled || mock.taintAddress != "aws_instance.web" {
		t.Error("should delegate")
	}
}

func TestComposite_Untaint_DelegatesToLive(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stateFile: "/state.json"}

	svc.Untaint(context.Background(), "aws_instance.web")
	if !mock.untaintCalled || mock.untaintAddress != "aws_instance.web" {
		t.Error("should delegate")
	}
}

func TestComposite_Output_DelegatesToLive(t *testing.T) {
	expected := map[string]sdk.OutputValue{"vpc_id": {Name: "vpc_id", Value: "vpc-123"}}
	mock := &mockLiveService{outputResult: expected}
	svc := &CompositeService{live: mock, stateFile: "/state.json"}

	result, err := svc.Output(context.Background())
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !mock.outputCalled || result["vpc_id"].Name != "vpc_id" {
		t.Error("should delegate")
	}
}

func TestComposite_Refresh_ReReadsStateFile(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	os.WriteFile(statePath, []byte(testStateJSON), 0o644)

	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stateFile: statePath}

	resources, _ := svc.StateList(context.Background())
	if len(resources) != 1 {
		t.Fatalf("initial len = %d", len(resources))
	}

	updatedState := `{"format_version":"1.0","terraform_version":"1.5.0","values":{"root_module":{"resources":[{"address":"local_file.foo","type":"local_file","name":"foo","provider_name":"registry.terraform.io/hashicorp/local","values":{"content":"bar","filename":"foo.txt"},"sensitive_values":{}},{"address":"local_file.bar","type":"local_file","name":"bar","provider_name":"registry.terraform.io/hashicorp/local","values":{"content":"baz","filename":"bar.txt"},"sensitive_values":{}}]}}}`
	os.WriteFile(statePath, []byte(updatedState), 0o644)

	svc.Refresh(context.Background())
	resources, _ = svc.StateList(context.Background())
	if len(resources) != 2 {
		t.Fatalf("after refresh len = %d, want 2", len(resources))
	}
	if mock.refreshCalled {
		t.Error("live.Refresh() should not be called")
	}
}

func TestComposite_Refresh_ReReadsPlanFile(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.json")
	os.WriteFile(planPath, []byte(testPlanJSON), 0o644)

	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, planFile: planPath}

	summary, _ := svc.Plan(context.Background(), sdk.PlanOptions{})
	if summary.ToCreate != 1 {
		t.Fatalf("initial ToCreate = %d", summary.ToCreate)
	}

	updatedPlan := `{"format_version":"1.2","terraform_version":"1.5.0","planned_values":{},"resource_changes":[{"address":"local_file.a","type":"local_file","name":"a","provider_name":"registry.terraform.io/hashicorp/local","change":{"actions":["create"],"before":null,"after":{},"after_unknown":{}}},{"address":"local_file.b","type":"local_file","name":"b","provider_name":"registry.terraform.io/hashicorp/local","change":{"actions":["create"],"before":null,"after":{},"after_unknown":{}}}]}`
	os.WriteFile(planPath, []byte(updatedPlan), 0o644)

	svc.Refresh(context.Background())
	summary, _ = svc.Plan(context.Background(), sdk.PlanOptions{})
	if summary.ToCreate != 2 {
		t.Errorf("after refresh ToCreate = %d, want 2", summary.ToCreate)
	}
	if mock.refreshCalled {
		t.Error("live.Refresh() should not be called")
	}
}

func TestComposite_Refresh_StdinCached(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, stdinPlan: []byte(testPlanJSON), stdinState: []byte(testStateJSON)}

	svc.Refresh(context.Background())
	summary, _ := svc.Plan(context.Background(), sdk.PlanOptions{})
	if summary.ToCreate != 1 {
		t.Errorf("ToCreate = %d, want 1", summary.ToCreate)
	}
	resources, _ := svc.StateList(context.Background())
	if len(resources) != 1 {
		t.Errorf("len = %d, want 1", len(resources))
	}
	if mock.refreshCalled {
		t.Error("live.Refresh() should not be called")
	}
}

func TestComposite_Refresh_LiveFallback(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock}

	svc.Refresh(context.Background())
	if !mock.refreshCalled {
		t.Fatal("live.Refresh() should be called")
	}
}

func TestComposite_WithDir_PropagatesLive(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.json")
	statePath := filepath.Join(dir, "state.json")
	os.WriteFile(planPath, []byte(testPlanJSON), 0o644)
	os.WriteFile(statePath, []byte(testStateJSON), 0o644)

	newLive := &mockLiveService{}
	mock := &mockLiveService{withDirResult: newLive}
	svc := &CompositeService{
		live:       mock,
		planFile:   planPath,
		stateFile:  statePath,
		stdinPlan:  []byte("p"),
		stdinState: []byte("s"),
	}

	result := svc.WithDir("/new/dir")
	composite := result.(*CompositeService)
	if mock.withDirPath != "/new/dir" {
		t.Errorf("WithDir path = %q", mock.withDirPath)
	}
	if composite.planFile != planPath {
		t.Error("planFile not preserved")
	}
	if composite.stateFile != statePath {
		t.Error("stateFile not preserved")
	}
	if composite.live != newLive {
		t.Error("live not updated")
	}
}

func TestComposite_Workspace_AlwaysDelegatesToLive(t *testing.T) {
	mock := &mockLiveService{workspaceResult: "production"}
	svc := &CompositeService{live: mock, stateFile: "/s", planFile: "/p"}

	result, err := svc.Workspace(context.Background())
	if err != nil || result != "production" || !mock.workspaceCalled {
		t.Errorf("Workspace() = %q, err = %v, called = %v", result, err, mock.workspaceCalled)
	}
}

func TestComposite_Validate_DelegatesToLive(t *testing.T) {
	expected := []sdk.Diagnostic{{Severity: "error", Summary: "bad"}}
	mock := &mockLiveService{validateResult: expected}
	svc := &CompositeService{live: mock, planFile: "/p"}

	result, _ := svc.Validate(context.Background())
	if !mock.validateCalled || len(result) != 1 {
		t.Error("should delegate")
	}
}

func TestComposite_Init_DelegatesToLive(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock, planFile: "/p"}

	svc.Init(context.Background())
	if !mock.initCalled {
		t.Error("should delegate")
	}
}

func TestComposite_ForceUnlock_DelegatesToLive(t *testing.T) {
	mock := &mockLiveService{}
	svc := &CompositeService{live: mock}

	svc.ForceUnlock(context.Background(), "lock-abc")
	if !mock.forceUnlockCalled || mock.forceUnlockID != "lock-abc" {
		t.Error("should delegate")
	}
}

func TestComposite_FileNotFound_Plan(t *testing.T) {
	svc := &CompositeService{live: &mockLiveService{}, planFile: "/nonexistent/plan.json"}
	_, err := svc.Plan(context.Background(), sdk.PlanOptions{})
	if err == nil {
		t.Fatal("should error when file not found")
	}
}

func TestComposite_FileNotFound_StateList(t *testing.T) {
	svc := &CompositeService{live: &mockLiveService{}, stateFile: "/nonexistent/state.json"}
	_, err := svc.StateList(context.Background())
	if err == nil {
		t.Fatal("should error when file not found")
	}
}

func TestComposite_FileNotFound_Show(t *testing.T) {
	svc := &CompositeService{live: &mockLiveService{}, stateFile: "/nonexistent/state.json"}
	_, err := svc.Show(context.Background(), "some.addr")
	if err == nil {
		t.Fatal("should error when file not found")
	}
}
