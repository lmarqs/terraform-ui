package workspace

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct {
	workspaceList      []string
	workspaceListErr   error
	workspace          string
	workspaceErr       error
	workspaceSelectErr error
	workspaceNewErr    error
	workspaceDeleteErr error
}

func (m *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (m *mockService) StateList(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
	return nil, nil
}
func (m *mockService) Show(_ context.Context, _ string) (string, error) { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error) {
	return m.workspace, m.workspaceErr
}
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return m.workspaceList, m.workspaceListErr
}
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error {
	return m.workspaceSelectErr
}
func (m *mockService) WorkspaceNew(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	return m.workspaceNewErr
}
func (m *mockService) WorkspaceDelete(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	return m.workspaceDeleteErr
}
func (m *mockService) StateRm(_ context.Context, _ string) error                    { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error               { return nil }
func (m *mockService) Import(_ context.Context, _, _ string) error                  { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error                      { return nil }
func (m *mockService) Untaint(_ context.Context, _ string) error                    { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error)         { return nil, nil }
func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) { return nil, nil }
func (m *mockService) Refresh(_ context.Context) error                              { return nil }
func (m *mockService) Init(_ context.Context) error                                 { return nil }
func (m *mockService) ForceUnlock(_ context.Context, _ string) error                { return nil }
func (m *mockService) Version(_ context.Context) (*sdk.VersionInfo, error)          { return nil, nil }
func (m *mockService) WithDir(_ string) sdk.Service                                 { return m }

// --- Plugin identity and lifecycle ---

func TestNew_ShouldReturn_PluginWithCorrectIdentity(t *testing.T) {
	p := New(&mockService{})
	if p.ID() != "workspace" {
		t.Errorf("ID() = %q, want %q", p.ID(), "workspace")
	}
	if p.Name() != "Workspace" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Workspace")
	}
	if p.Description() != "Manage terraform workspace" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Manage terraform workspace")
	}
}

func TestNew_ShouldNotBeReady_BeforeActivation(t *testing.T) {
	p := New(&mockService{})
	if p.(*Plugin).Ready() {
		t.Error("Ready() = true before activation, want false")
	}
}

func TestConfigure_ShouldAcceptAnyConfig(t *testing.T) {
	p := New(&mockService{})
	if err := p.(*Plugin).Configure(map[string]interface{}{"key": "value"}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestInit_ShouldSetStatusIdle_WithoutAutoLoading(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	cmd := p.Init(&sdk.Context{Service: svc, WorkingDir: "/tmp", Workspace: "default"})
	if cmd != nil {
		t.Error("Init() should not auto-load")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want Idle", p.status)
	}
}

func TestStack_ShouldReturn_NonNilStack(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	if p.Stack() == nil {
		t.Fatal("Stack() = nil")
	}
	if p.Stack().Depth() != 1 {
		t.Errorf("Stack().Depth() = %d, want 1", p.Stack().Depth())
	}
}

// --- Activation ---

func TestActivate_GivenIdle_ShouldStartLoading(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	p.Init(&sdk.Context{Service: svc})

	cmd := p.Activate()
	if cmd == nil {
		t.Fatal("Activate() from idle should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
}

func TestActivate_GivenError_ShouldRetryLoading(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	p.Init(&sdk.Context{Service: svc})
	p.status = sdk.StatusError

	cmd := p.Activate()
	if cmd == nil {
		t.Fatal("Activate() from error should retry")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
}

func TestActivate_GivenDone_ShouldNotReload(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	p.Init(&sdk.Context{Service: svc})
	p.status = sdk.StatusDone

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() from done should not reload")
	}
}

func TestActivate_ShouldFetchWorkspaceList(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default", "staging"}, workspace: "staging"}
	p := New(svc).(*Plugin)
	p.Init(&sdk.Context{Service: svc})

	cmd := p.Activate()
	msg := cmd()
	result, ok := msg.(WorkspaceListMsg)
	if !ok {
		t.Fatalf("Activate cmd returned %T, want WorkspaceListMsg", msg)
	}
	if len(result.Workspaces) != 2 {
		t.Errorf("len(Workspaces) = %d, want 2", len(result.Workspaces))
	}
	if result.Current != "staging" {
		t.Errorf("Current = %q, want %q", result.Current, "staging")
	}
}

func TestActivate_GivenListError_ShouldReturnErrorMsg(t *testing.T) {
	svc := &mockService{workspaceListErr: errors.New("network error")}
	p := New(svc).(*Plugin)
	p.Init(&sdk.Context{Service: svc})

	cmd := p.Activate()
	msg := cmd()
	result := msg.(WorkspaceListMsg)
	if result.Err == nil {
		t.Error("expected error in WorkspaceListMsg")
	}
}

func TestActivate_GivenWorkspaceError_ShouldReturnErrorMsg(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspaceErr: errors.New("fail")}
	p := New(svc).(*Plugin)
	p.Init(&sdk.Context{Service: svc})

	cmd := p.Activate()
	msg := cmd()
	result := msg.(WorkspaceListMsg)
	if result.Err == nil {
		t.Error("expected error")
	}
}

// --- Update: WorkspaceListMsg ---

func TestUpdate_GivenSuccessfulList_ShouldPopulateWorkspaces(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	result, cmd := p.Update(WorkspaceListMsg{
		Workspaces: []string{"default", "staging", "production"},
		Current:    "staging",
	})
	if cmd != nil {
		t.Error("cmd should be nil on list success")
	}
	updated := result.(*Plugin)
	if updated.status != sdk.StatusDone {
		t.Errorf("status = %v, want Done", updated.status)
	}
	if len(updated.workspaces) != 3 {
		t.Errorf("len(workspaces) = %d, want 3", len(updated.workspaces))
	}
	if updated.current != "staging" {
		t.Errorf("current = %q, want %q", updated.current, "staging")
	}
	if updated.selected != 1 {
		t.Errorf("selected = %d, want 1 (auto-select current)", updated.selected)
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdate_GivenListError_ShouldSetErrorStatus(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	result, _ := p.Update(WorkspaceListMsg{Err: errors.New("timeout")})
	updated := result.(*Plugin)
	if updated.status != sdk.StatusError {
		t.Errorf("status = %v, want Error", updated.status)
	}
	if updated.errMsg != "timeout" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "timeout")
	}
}

// --- Switch workspace flow ---

func TestSwitchToSelected_GivenDifferentWorkspace_ShouldSetLoadingAndDispatch(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 1

	cmd := p.SwitchToSelected()
	if cmd == nil {
		t.Fatal("SwitchToSelected should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
	if !strings.Contains(p.loadingMsg, "staging") {
		t.Errorf("loadingMsg = %q, should mention target workspace", p.loadingMsg)
	}
}

func TestSwitchToSelected_GivenSameWorkspace_ShouldDeactivate(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 0

	cmd := p.SwitchToSelected()
	if cmd == nil {
		t.Fatal("selecting current workspace should return deactivate command")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd returned %T, want sdk.DeactivateMsg", msg)
	}
}

func TestSwitchToSelected_GivenEmptyList_ShouldReturnNil(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.workspaces = []string{}

	cmd := p.SwitchToSelected()
	if cmd != nil {
		t.Error("switching with empty list should return nil")
	}
}

func TestUpdate_GivenSwitchSuccess_WithPopBack_ShouldEmitWorkspaceChangedEvent(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.workspaces = []string{"default", "staging"}
	p.current = "default"

	_, cmd := p.Update(WorkspaceSwitchMsg{Name: "staging", PopBack: true})
	if cmd == nil {
		t.Fatal("successful switch should emit event command")
	}
	msg := cmd()
	evt, ok := msg.(sdk.WorkspaceChangedEvent)
	if !ok {
		t.Fatalf("cmd returned %T, want WorkspaceChangedEvent", msg)
	}
	if evt.Name != "staging" {
		t.Errorf("event Name = %q, want %q", evt.Name, "staging")
	}
}

func TestUpdate_GivenSwitchError_ShouldSetErrorStatus(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	result, cmd := p.Update(WorkspaceSwitchMsg{Name: "x", Err: errors.New("fail"), PopBack: true})
	if cmd != nil {
		t.Error("error switch should return nil cmd")
	}
	updated := result.(*Plugin)
	if updated.status != sdk.StatusError {
		t.Errorf("status = %v, want Error", updated.status)
	}
}

func TestSelectWorkspaceCmd_PopBack_ShouldCallService(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc

	cmd := p.selectWorkspace("staging", true)
	msg := cmd()
	sm := msg.(WorkspaceSwitchMsg)
	if sm.Name != "staging" || sm.Err != nil {
		t.Errorf("selectWorkspace: Name=%q Err=%v", sm.Name, sm.Err)
	}
	if !sm.PopBack {
		t.Error("PopBack should be true")
	}
}

func TestSelectWorkspaceCmd_NoPopBack_ShouldCallService(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc

	cmd := p.selectWorkspace("staging", false)
	msg := cmd()
	sm := msg.(WorkspaceSwitchMsg)
	if sm.Name != "staging" || sm.Err != nil {
		t.Errorf("selectWorkspace: Name=%q Err=%v", sm.Name, sm.Err)
	}
	if sm.PopBack {
		t.Error("PopBack should be false")
	}
}

func TestSelectWorkspaceCmd_GivenServiceError_ShouldReturnError(t *testing.T) {
	svc := &mockService{workspaceSelectErr: errors.New("fail")}
	p := New(svc).(*Plugin)
	p.svc = svc

	cmd := p.selectWorkspace("x", true)
	msg := cmd()
	sm := msg.(WorkspaceSwitchMsg)
	if sm.Err == nil {
		t.Error("expected error")
	}
}

// --- Create workspace flow ---

func TestStartCreate_ShouldSetLoadingAndDispatch(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusDone

	cmd := p.startCreate("feature-x")
	if cmd == nil {
		t.Fatal("startCreate should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
	if !strings.Contains(p.loadingMsg, "feature-x") {
		t.Errorf("loadingMsg = %q, should mention workspace name", p.loadingMsg)
	}
}

func TestCreateWorkspaceCmd_ShouldReturnCreateMsg(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc

	cmd := p.createWorkspace("new-ws")
	msg := cmd()
	cm, ok := msg.(WorkspaceCreateMsg)
	if !ok {
		t.Fatalf("createWorkspace returned %T, want WorkspaceCreateMsg", msg)
	}
	if cm.Name != "new-ws" || cm.Err != nil {
		t.Errorf("Name=%q Err=%v", cm.Name, cm.Err)
	}
}

func TestCreateWorkspaceCmd_GivenServiceError_ShouldReturnError(t *testing.T) {
	svc := &mockService{workspaceNewErr: errors.New("exists")}
	p := New(svc).(*Plugin)
	p.svc = svc

	cmd := p.createWorkspace("x")
	msg := cmd()
	cm := msg.(WorkspaceCreateMsg)
	if cm.Err == nil {
		t.Error("expected error")
	}
}

func TestUpdate_GivenCreateSuccess_ShouldRefreshAndEmitCreatedEvent(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default", "feature-x"}, workspace: "feature-x"}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusLoading

	result, cmd := p.Update(WorkspaceCreateMsg{Name: "feature-x", Err: nil})
	updated := result.(*Plugin)

	if updated.current != "feature-x" {
		t.Errorf("current = %q, want %q", updated.current, "feature-x")
	}
	if updated.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading (refreshing)", updated.status)
	}
	if cmd == nil {
		t.Fatal("should return batch command")
	}
}

func TestUpdate_GivenCreateError_ShouldSetErrorStatus(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	result, cmd := p.Update(WorkspaceCreateMsg{Name: "x", Err: errors.New("already exists")})
	if cmd != nil {
		t.Error("error should return nil cmd")
	}
	updated := result.(*Plugin)
	if updated.status != sdk.StatusError {
		t.Errorf("status = %v, want Error", updated.status)
	}
	if updated.errMsg != "already exists" {
		t.Errorf("errMsg = %q", updated.errMsg)
	}
}

// --- Delete workspace flow ---

func TestDeleteWorkspace_ShouldSetLoadingAndDispatch(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "temp"}
	p.current = "default"

	cmd := p.deleteWorkspace("temp")
	if cmd == nil {
		t.Fatal("deleteWorkspace should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
	if !strings.Contains(p.loadingMsg, "temp") {
		t.Errorf("loadingMsg = %q, should mention workspace name", p.loadingMsg)
	}
}

func TestDeleteWorkspace_GivenServiceSuccess_ShouldRefresh(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusLoading
	p.workspaces = []string{"default", "temp"}
	p.current = "default"

	result, cmd := p.Update(WorkspaceDeleteMsg{Err: nil})
	updated := result.(*Plugin)
	if updated.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading (refreshing)", updated.status)
	}
	if cmd == nil {
		t.Fatal("should return refresh command")
	}
}

func TestDeleteWorkspace_GivenServiceError_ShouldSetErrorStatus(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	result, cmd := p.Update(WorkspaceDeleteMsg{Err: errors.New("locked")})
	if cmd != nil {
		t.Error("error should return nil cmd")
	}
	updated := result.(*Plugin)
	if updated.status != sdk.StatusError {
		t.Errorf("status = %v, want Error", updated.status)
	}
}

// --- Select workspace (stays in list) ---

func TestSelectCurrent_GivenDifferentWorkspace_ShouldSetLoadingAndDispatch(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 1

	cmd := p.SelectCurrent()
	if cmd == nil {
		t.Fatal("SelectCurrent should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
	if !strings.Contains(p.loadingMsg, "staging") {
		t.Errorf("loadingMsg = %q, should mention workspace", p.loadingMsg)
	}
}

func TestSelectCurrent_GivenSameWorkspace_ShouldReturnNil(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 0

	cmd := p.SelectCurrent()
	if cmd != nil {
		t.Error("selecting current workspace should return nil")
	}
}

func TestSelectCurrent_GivenEmptyList_ShouldReturnNil(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.workspaces = []string{}

	cmd := p.SelectCurrent()
	if cmd != nil {
		t.Error("selecting with empty list should return nil")
	}
}

func TestUpdate_GivenSelectSuccess_ShouldRefreshAndEmitCreatedEvent(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default", "staging"}, workspace: "staging"}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusLoading

	result, cmd := p.Update(WorkspaceSwitchMsg{Name: "staging", PopBack: false})
	updated := result.(*Plugin)

	if updated.current != "staging" {
		t.Errorf("current = %q, want %q", updated.current, "staging")
	}
	if updated.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading (refreshing)", updated.status)
	}
	if cmd == nil {
		t.Fatal("should return batch command (refresh + event)")
	}

	// Execute batch to verify event is emitted
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}
	foundEvent := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		subMsg := subCmd()
		if evt, ok := subMsg.(sdk.WorkspaceCreatedEvent); ok {
			foundEvent = true
			if evt.Name != "staging" {
				t.Errorf("event Name = %q, want %q", evt.Name, "staging")
			}
		}
	}
	if !foundEvent {
		t.Error("batch should contain WorkspaceCreatedEvent")
	}
}

func TestFrame_SKey_GivenDifferentWorkspace_ShouldSelect(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 1

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("s on different workspace should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
}

func TestFrame_SKey_GivenSameWorkspace_ShouldReturnNil(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 0

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd != nil {
		t.Error("s on current workspace should return nil")
	}
}

func TestFrame_SKey_GivenLoading_ShouldBeIgnored(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.workspaces = []string{"default", "staging"}
	p.selected = 1

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd != nil {
		t.Error("s during loading should be ignored")
	}
}

// --- Frame: Loading guards ---

func TestFrame_GivenLoading_EnterKey_ShouldBeIgnored(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.workspaces = []string{"default", "staging"}
	p.selected = 1

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Enter during loading should be ignored")
	}
}

func TestFrame_GivenLoading_DeleteKey_ShouldBeIgnored(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 1

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Error("d during loading should be ignored")
	}
}

func TestFrame_GivenLoading_RefreshKey_ShouldBeIgnored(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("ctrl+r during loading should be ignored")
	}
}

func TestFrame_GivenLoading_NewKey_ShouldBeIgnored(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd != nil {
		t.Error("n during loading should be ignored")
	}
}

func TestFrame_CtrlR_GivenError_ShouldRetry(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusError
	p.errMsg = "connection failed"

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("ctrl+r in error state should retry (refresh)")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("after retry: status = %v, want Loading", p.status)
	}
}

// --- Frame: Delete confirmation ---

func TestFrame_GivenDeletableWorkspace_DKey_ShouldPushConfirmFrame(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "temp"}
	p.current = "default"
	p.selected = 1

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if p.stack.Depth() != 2 {
		t.Errorf("stack depth = %d, want 2 (confirm frame pushed)", p.stack.Depth())
	}
	if p.stack.Peek().ID() != "confirm" {
		t.Errorf("top frame = %q, want %q", p.stack.Peek().ID(), "confirm")
	}
}

func TestFrame_GivenConfirmFrame_YKey_ShouldTriggerDelete(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "temp"}
	p.current = "default"
	p.selected = 1

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if cmd == nil {
		t.Fatal("confirming delete should return a command")
	}
	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1 (confirm popped)", p.stack.Depth())
	}
}

func TestFrame_GivenConfirmFrame_NKey_ShouldCancelDelete(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "temp"}
	p.current = "default"
	p.selected = 1

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1 (confirm dismissed)", p.stack.Depth())
	}
}

func TestFrame_GivenCurrentWorkspace_DKey_ShouldDoNothing(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 0

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Error("d on current workspace should do nothing")
	}
	if p.stack.Depth() != 1 {
		t.Error("no confirm frame should be pushed for current workspace")
	}
}

func TestFrame_GivenDefaultWorkspace_DKey_ShouldDoNothing(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "staging"
	p.selected = 0

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Error("d on 'default' workspace should do nothing")
	}
}

// --- Frame: Create via RequestInputMsg ---

func TestFrame_NKey_ShouldEmitRequestInputMsg(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default"}
	p.current = "default"

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("n key should return a command")
	}

	msg := cmd()
	reqMsg, ok := msg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("n cmd returned %T, want sdk.RequestInputMsg", msg)
	}
	if reqMsg.Request.Prompt != "New workspace:" {
		t.Errorf("prompt = %q, want %q", reqMsg.Request.Prompt, "New workspace:")
	}
}

func TestFrame_NKey_Callback_GivenValidName_ShouldStartCreate(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusDone
	p.workspaces = []string{"default"}
	p.current = "default"

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	result := reqMsg.Request.Callback("feature-x")
	if result == nil {
		t.Fatal("callback with valid name should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
}

func TestFrame_NKey_Callback_GivenEmptyName_ShouldCancel(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default"}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	result := reqMsg.Request.Callback("")
	if result != nil {
		t.Error("empty name should cancel (return nil)")
	}
}

func TestFrame_NKey_Callback_GivenInvalidName_ShouldReject(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default"}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	invalids := []string{"has space", "has/slash", "has@at"}
	for _, name := range invalids {
		result := reqMsg.Request.Callback(name)
		if result != nil {
			t.Errorf("invalid name %q should be rejected", name)
		}
	}
}

// --- Frame: Navigation ---

func TestFrame_EscKey_ShouldEmitDeactivateMsg(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default"}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should return a command")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("esc cmd returned %T, want sdk.DeactivateMsg", msg)
	}
}

func TestFrame_JKey_ShouldMoveDown(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"a", "b", "c"}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", p.selected)
	}
}

func TestFrame_KKey_ShouldMoveUp(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"a", "b", "c"}
	p.selected = 2

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 1 {
		t.Errorf("after k: selected = %d, want 1", p.selected)
	}
}

func TestFrame_DownKey_ShouldMoveDown(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"a", "b"}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down: selected = %d, want 1", p.selected)
	}
}

func TestFrame_UpKey_ShouldMoveUp(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"a", "b"}
	p.selected = 1

	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 0 {
		t.Errorf("after up: selected = %d, want 0", p.selected)
	}
}

func TestFrame_EnterKey_GivenSameWorkspace_ShouldDeactivate(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 0

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on current workspace should deactivate")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd returned %T, want sdk.DeactivateMsg", msg)
	}
}

func TestFrame_EnterKey_GivenDifferentWorkspace_ShouldSwitch(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.svc = &mockService{}
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 1

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on different workspace should return a command")
	}
}

func TestFrame_NonKeyMsg_ShouldBeIgnored(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone

	frame := p.stack.Peek()
	type customMsg struct{}
	result, cmd := frame.Update(customMsg{})
	if result != frame {
		t.Error("non-key msg should return same frame")
	}
	if cmd != nil {
		t.Error("non-key msg should return nil cmd")
	}
}

// --- Hints: context-sensitive ---

func TestHints_GivenLoading_ShouldShowOnlyBack(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	hints := p.stack.Peek().Hints()
	for _, h := range hints {
		if h.Key == "d" || h.Key == "n" || h.Key == "enter" {
			t.Errorf("hint %q should not appear during loading", h.Key)
		}
	}
}

func TestHints_GivenError_ShouldShowRetryAndBack(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusError

	hints := p.stack.Peek().Hints()
	if len(hints) == 0 {
		t.Fatal("error state should have hints")
	}
}

func TestHints_GivenDone_CursorOnDeletable_ShouldShowDeleteAndSelect(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging", "temp"}
	p.current = "default"
	p.selected = 2

	hints := p.stack.Peek().Hints()
	foundDelete := false
	foundSelect := false
	for _, h := range hints {
		if h.Key == "d" {
			foundDelete = true
		}
		if h.Key == "s" && h.Description == "select" {
			foundSelect = true
		}
	}
	if !foundDelete {
		t.Error("delete hint should appear when cursor is on deletable workspace")
	}
	if !foundSelect {
		t.Error("select hint should appear when cursor is on non-current workspace")
	}
}

func TestHints_GivenDone_CursorOnCurrent_ShouldHideDelete(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "staging"
	p.selected = 1

	hints := p.stack.Peek().Hints()
	for _, h := range hints {
		if h.Key == "d" {
			t.Error("delete hint should NOT appear when cursor is on current workspace")
		}
	}
}

func TestHints_GivenDone_CursorOnDefault_ShouldHideDelete(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "staging"
	p.selected = 0

	hints := p.stack.Peek().Hints()
	for _, h := range hints {
		if h.Key == "d" {
			t.Error("delete hint should NOT appear when cursor is on 'default'")
		}
	}
}

// --- View rendering ---

func TestView_GivenIdleStatus_ShouldShowLoadingMessage(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestView_GivenLoadingWithMessage_ShouldShowContextualMessage(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.loadingMsg = "Switching to staging..."

	view := p.View(80, 24)
	if !strings.Contains(view, "Switching to staging") {
		t.Errorf("view = %q, should contain contextual loading message", view)
	}
}

func TestView_GivenError_ShouldShowErrorMessage(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusError
	p.errMsg = "connection failed"

	view := p.View(80, 24)
	if !strings.Contains(view, "connection failed") {
		t.Errorf("view should contain error message")
	}
}

func TestView_GivenDone_ShouldShowWorkspaceList(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging", "production"}
	p.current = "staging"
	p.selected = 1

	view := p.View(80, 24)
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestView_GivenDone_ShouldScrollLargeList(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.current = "ws-0"

	workspaces := make([]string, 50)
	for i := range workspaces {
		workspaces[i] = "ws-" + string(rune('0'+i%10))
	}
	p.workspaces = workspaces
	p.selected = 40

	view := p.View(80, 10)
	if view == "" {
		t.Error("view with scrolling should not be empty")
	}
}

func TestView_GivenUnknownStatus_ShouldReturnEmpty(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("unknown status should return empty, got %q", view)
	}
}

// --- ChdirChanged handler ---

func TestHandleChdirChanged_ShouldResetAndUpdateService(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	p.Init(&sdk.Context{Service: svc})
	p.status = sdk.StatusDone
	p.scopedContext = "/old"

	p.HandleChdirChanged(sdk.ChdirChangedEvent{AbsPath: "/new", RelPath: "modules/new"})

	if p.scopedContext != "/new" {
		t.Errorf("scopedContext = %q, want %q", p.scopedContext, "/new")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want Idle (reset)", p.status)
	}
}

// --- Helpers ---

func TestMoveUp_AtBoundary_ShouldStay(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.workspaces = []string{"a", "b"}
	p.selected = 0

	p.MoveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp at 0: selected = %d, want 0", p.selected)
	}
}

func TestMoveDown_AtBoundary_ShouldStay(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.workspaces = []string{"a", "b"}
	p.selected = 1

	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown at end: selected = %d, want 1", p.selected)
	}
}

func TestSelectedWorkspace_GivenEmptyList_ShouldReturnEmpty(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.workspaces = []string{}
	if p.SelectedWorkspace() != "" {
		t.Error("SelectedWorkspace with empty list should return empty")
	}
}

func TestCurrent_ShouldReturnCurrentWorkspace(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.current = "staging"
	if p.Current() != "staging" {
		t.Errorf("Current() = %q, want %q", p.Current(), "staging")
	}
}

func TestWorkspaces_ShouldReturnList(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.workspaces = []string{"a", "b"}
	if len(p.Workspaces()) != 2 {
		t.Errorf("Workspaces() len = %d, want 2", len(p.Workspaces()))
	}
}

func TestStatus_ShouldReturnCurrentStatus(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	if p.Status() != sdk.StatusIdle {
		t.Errorf("Status() = %v, want Idle", p.Status())
	}
}

func TestSelected_ShouldReturnCurrentIndex(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.selected = 3
	if p.Selected() != 3 {
		t.Errorf("Selected() = %d, want 3", p.Selected())
	}
}

func TestRefresh_ShouldResetAndStartLoading(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusDone
	p.selected = 5

	cmd := p.Refresh()
	if cmd == nil {
		t.Fatal("Refresh() should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
}

func TestUpdate_UnknownMsg_ShouldPassThrough(t *testing.T) {
	p := New(&mockService{})
	type unknownMsg struct{}
	result, cmd := p.Update(unknownMsg{})
	if cmd != nil {
		t.Error("unknown msg should return nil cmd")
	}
	if result != p {
		t.Error("unknown msg should return same plugin")
	}
}

func TestListFrame_ID(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	frame := p.stack.Peek()
	if frame.ID() != "list" {
		t.Errorf("listFrame.ID() = %q, want %q", frame.ID(), "list")
	}
}

func TestListFrame_View_ShouldDelegateToPlugin(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"

	frame := p.stack.Peek()
	view := frame.View(80, 24)
	if view == "" {
		t.Error("frame View should not be empty")
	}
}

// --- Frame: ctrl+r refresh ---

func TestFrame_CtrlR_GivenDone_ShouldRefresh(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusDone
	p.workspaces = []string{"default"}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("ctrl+r when done should return a command")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("after ctrl+r: status = %v, want Loading", p.status)
	}
}

// --- Delete command execution ---

func TestDeleteWorkspaceCmd_ShouldCallServiceAndReturnMsg(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusDone

	cmd := p.deleteWorkspace("temp")
	msg := cmd()
	dm, ok := msg.(WorkspaceDeleteMsg)
	if !ok {
		t.Fatalf("deleteWorkspace cmd returned %T, want WorkspaceDeleteMsg", msg)
	}
	if dm.Err != nil {
		t.Errorf("Err = %v, want nil", dm.Err)
	}
}

// --- Create event emission ---

func TestUpdate_GivenCreateSuccess_BatchContainsCreatedEvent(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default", "new-ws"}, workspace: "new-ws"}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusLoading

	_, cmd := p.Update(WorkspaceCreateMsg{Name: "new-ws", Err: nil})
	if cmd == nil {
		t.Fatal("expected batch command")
	}

	// tea.Batch returns a BatchMsg containing sub-commands.
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}

	// Execute each sub-command to find the WorkspaceCreatedEvent
	foundEvent := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		subMsg := subCmd()
		if evt, ok := subMsg.(sdk.WorkspaceCreatedEvent); ok {
			foundEvent = true
			if evt.Name != "new-ws" {
				t.Errorf("event Name = %q, want %q", evt.Name, "new-ws")
			}
		}
	}
	if !foundEvent {
		t.Error("batch should contain WorkspaceCreatedEvent")
	}
}

// --- View edge case ---

func TestView_GivenVerySmallHeight_ShouldUseMinVisibleArea(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.workspaces = []string{"a", "b", "c", "d", "e"}
	p.current = "a"

	view := p.View(80, 5)
	if view == "" {
		t.Error("view with small height should not be empty")
	}
}

// --- Workspace name validation ---

func TestIsValidWorkspaceName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"default", true},
		{"my-workspace", true},
		{"my_workspace", true},
		{"my.workspace", true},
		{"MyWorkspace123", true},
		{"", false},
		{"has space", false},
		{"has/slash", false},
		{"has@at", false},
		{"has:colon", false},
		{"has!bang", false},
	}
	for _, tt := range tests {
		if got := isValidWorkspaceName(tt.name); got != tt.valid {
			t.Errorf("isValidWorkspaceName(%q) = %v, want %v", tt.name, got, tt.valid)
		}
	}
}
