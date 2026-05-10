package workspaces

import (
	"context"
	"errors"
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

func (m *mockService) Plan(_ context.Context, _ []string) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (m *mockService) Apply(_ context.Context, _ []string) error           { return nil }
func (m *mockService) StateList(_ context.Context) ([]sdk.Resource, error) { return nil, nil }
func (m *mockService) Show(_ context.Context, _ string) (string, error)    { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error) {
	return m.workspace, m.workspaceErr
}
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return m.workspaceList, m.workspaceListErr
}
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error {
	return m.workspaceSelectErr
}
func (m *mockService) WorkspaceNew(_ context.Context, _ string) error {
	return m.workspaceNewErr
}
func (m *mockService) WorkspaceDelete(_ context.Context, _ string) error {
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
func (m *mockService) WithDir(_ string) sdk.Service                                 { return m }

func TestNew(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	if p.ID() != "workspaces" {
		t.Errorf("ID() = %q, want %q", p.ID(), "workspaces")
	}
	if p.Name() != "Workspaces" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Workspaces")
	}
	if p.Description() != "Manage terraform workspaces" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Manage terraform workspaces")
	}
	if p.KeyBinding() != "w" {
		t.Errorf("KeyBinding() = %q, want %q", p.KeyBinding(), "w")
	}
	if p.Ready() {
		t.Error("Ready() = true before load, want false")
	}
}

func TestConfigure(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	err := p.Configure(map[string]interface{}{"key": "value"})
	if err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestInit(t *testing.T) {
	svc := &mockService{
		workspaceList: []string{"default", "staging"},
		workspace:     "default",
	}
	p := New(svc)
	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Workspace:  "default",
		Service:    svc,
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() returned nil cmd, should return nil (no auto-load)")
	}

	pp := p.(*Plugin)
	if pp.status != StatusIdle {
		t.Errorf("status = %v, want StatusIdle", pp.status)
	}
}

func TestInitCmdReturnsWorkspaceListMsg(t *testing.T) {
	svc := &mockService{
		workspaceList: []string{"default", "staging", "production"},
		workspace:     "staging",
	}
	p := New(svc)
	ctx := &sdk.Context{Service: svc}

	p.Init(ctx)
	cmd := p.(*Plugin).Activate()
	msg := cmd()

	result, ok := msg.(WorkspaceListMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T, want WorkspaceListMsg", msg)
	}
	if result.Err != nil {
		t.Errorf("WorkspaceListMsg.Err = %v, want nil", result.Err)
	}
	if len(result.Workspaces) != 3 {
		t.Errorf("len(Workspaces) = %d, want 3", len(result.Workspaces))
	}
	if result.Current != "staging" {
		t.Errorf("Current = %q, want %q", result.Current, "staging")
	}
}

func TestActivateCmdWorkspaceListError(t *testing.T) {
	svc := &mockService{workspaceListErr: errors.New("list error")}
	p := New(svc)
	ctx := &sdk.Context{Service: svc}
	p.Init(ctx)

	cmd := p.(*Plugin).Activate()
	msg := cmd()

	result, ok := msg.(WorkspaceListMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T, want WorkspaceListMsg", msg)
	}
	if result.Err == nil {
		t.Error("WorkspaceListMsg.Err = nil, want error")
	}
}

func TestActivateCmdWorkspaceError(t *testing.T) {
	svc := &mockService{
		workspaceList: []string{"default"},
		workspaceErr:  errors.New("workspace error"),
	}
	p := New(svc)
	ctx := &sdk.Context{Service: svc}
	p.Init(ctx)

	cmd := p.(*Plugin).Activate()
	msg := cmd()

	result, ok := msg.(WorkspaceListMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T, want WorkspaceListMsg", msg)
	}
	if result.Err == nil {
		t.Error("WorkspaceListMsg.Err = nil, want error")
	}
}

func TestUpdateWorkspaceListMsgSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	result, cmd := p.Update(WorkspaceListMsg{
		Workspaces: []string{"default", "staging", "production"},
		Current:    "staging",
		Err:        nil,
	})
	if cmd != nil {
		t.Errorf("Update(WorkspaceListMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusDone {
		t.Errorf("status = %v, want StatusDone", updated.status)
	}
	if len(updated.workspaces) != 3 {
		t.Errorf("len(workspaces) = %d, want 3", len(updated.workspaces))
	}
	if updated.current != "staging" {
		t.Errorf("current = %q, want %q", updated.current, "staging")
	}
	// Should auto-select the current workspace
	if updated.selected != 1 {
		t.Errorf("selected = %d, want 1 (staging index)", updated.selected)
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdateWorkspaceListMsgError(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	result, cmd := p.Update(WorkspaceListMsg{Err: errors.New("error loading")})
	if cmd != nil {
		t.Errorf("Update(WorkspaceListMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusError {
		t.Errorf("status = %v, want StatusError", updated.status)
	}
	if updated.errMsg != "error loading" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "error loading")
	}
}

func TestUpdateWorkspaceSwitchMsgSuccess(t *testing.T) {
	svc := &mockService{
		workspaceList: []string{"default", "staging"},
		workspace:     "staging",
	}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusDone
	pp.svc = svc

	result, cmd := p.Update(WorkspaceSwitchMsg{Name: "staging", Err: nil})
	if cmd == nil {
		t.Error("Update(WorkspaceSwitchMsg) cmd = nil, want non-nil (refresh)")
	}

	updated := result.(*Plugin)
	if updated.current != "staging" {
		t.Errorf("current = %q, want %q", updated.current, "staging")
	}
}

func TestUpdateWorkspaceSwitchMsgError(t *testing.T) {
	svc := &mockService{
		workspaceList: []string{"default"},
		workspace:     "default",
	}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusDone
	pp.svc = svc

	result, cmd := p.Update(WorkspaceSwitchMsg{Name: "x", Err: errors.New("switch failed")})
	if cmd == nil {
		t.Error("Update(WorkspaceSwitchMsg error) cmd = nil, want non-nil (still refreshes)")
	}

	// After error, Refresh() is called which resets errMsg
	updated := result.(*Plugin)
	if updated.status != StatusLoading {
		t.Errorf("status = %v, want StatusLoading (refresh triggered)", updated.status)
	}
	_ = updated
}

func TestUpdateKeyMsgNavigation(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default", "staging", "production"}
	p.current = "default"

	// Move down with j
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", p.selected)
	}

	// Move down more
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 2 {
		t.Errorf("after j,j: selected = %d, want 2", p.selected)
	}

	// Boundary
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 2 {
		t.Errorf("after j,j,j: selected = %d, want 2 (boundary)", p.selected)
	}

	// Move up with k
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 1 {
		t.Errorf("after k: selected = %d, want 1", p.selected)
	}
}

func TestUpdateKeyMsgEnter_SwitchWorkspace(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 1

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("after enter (switch): cmd = nil, want non-nil")
	}
}

func TestUpdateKeyMsgEnter_SameWorkspace(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 0

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("after enter on current ws: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgN_CreateMode(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default"}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !p.creating {
		t.Error("after n: creating = false, want true")
	}
	if !p.IsCreating() {
		t.Error("IsCreating() = false, want true")
	}
}

func TestUpdateKeyMsgCreating_Enter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.creating = true
	p.newName = "my-workspace"

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("after enter in creating: cmd = nil, want non-nil")
	}
	if p.creating {
		t.Error("after enter in creating: creating = true, want false")
	}
}

func TestUpdateKeyMsgCreating_EnterEmpty(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.creating = true
	p.newName = ""

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("after enter with empty name: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgCreating_Esc(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.creating = true
	p.newName = "partial"

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.creating {
		t.Error("after esc in creating: creating = true, want false")
	}
	if p.newName != "" {
		t.Errorf("after esc in creating: newName = %q, want empty", p.newName)
	}
}

func TestUpdateKeyMsgCreating_Backspace(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.creating = true
	p.newName = "abc"

	p.stack.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.newName != "ab" {
		t.Errorf("after backspace: newName = %q, want %q", p.newName, "ab")
	}

	// Backspace on empty
	p.newName = ""
	p.stack.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.newName != "" {
		t.Errorf("after backspace on empty: newName = %q, want empty", p.newName)
	}
}

func TestUpdateKeyMsgCreating_TypeChar(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.creating = true
	p.newName = ""

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if p.newName != "t" {
		t.Errorf("after 't': newName = %q, want %q", p.newName, "t")
	}
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if p.newName != "te" {
		t.Errorf("after 'e': newName = %q, want %q", p.newName, "te")
	}
}

func TestUpdateKeyMsgD_DeleteSelected(t *testing.T) {
	svc := &mockService{
		workspaceList: []string{"default", "staging"},
		workspace:     "default",
	}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 1

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Error("after d on non-current: cmd = nil, want non-nil (refresh)")
	}
}

func TestUpdateKeyMsgD_DeleteCurrent(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 0 // trying to delete current

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Error("after d on current ws: cmd != nil, want nil (cannot delete current)")
	}
}

func TestUpdateKeyMsgD_DeleteDefault(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default", "staging"}
	p.current = "staging"
	p.selected = 0 // "default"

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Error("after d on 'default' ws: cmd != nil, want nil (cannot delete default)")
	}
}

func TestUpdateKeyMsgR_Refresh(t *testing.T) {
	svc := &mockService{
		workspaceList: []string{"default"},
		workspace:     "default",
	}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.svc = svc

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("after r: cmd = nil, want non-nil (refresh)")
	}
}

func TestUpdateUnknownMsg(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	type unknownMsg struct{}
	result, cmd := p.Update(unknownMsg{})
	if cmd != nil {
		t.Errorf("Update(unknownMsg) cmd = %v, want nil", cmd)
	}
	if result != p {
		t.Error("Update(unknownMsg) returned different plugin reference")
	}
}

func TestMoveUpDown(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.workspaces = []string{"a", "b", "c"}

	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown: selected = %d, want 1", p.selected)
	}
	p.MoveDown()
	if p.selected != 2 {
		t.Errorf("MoveDown: selected = %d, want 2", p.selected)
	}
	p.MoveDown()
	if p.selected != 2 {
		t.Errorf("MoveDown boundary: selected = %d, want 2", p.selected)
	}
	p.MoveUp()
	if p.selected != 1 {
		t.Errorf("MoveUp: selected = %d, want 1", p.selected)
	}
	p.selected = 0
	p.MoveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp boundary: selected = %d, want 0", p.selected)
	}
}

func TestSelectedWorkspace(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	// Empty
	p.workspaces = []string{}
	if p.SelectedWorkspace() != "" {
		t.Errorf("SelectedWorkspace empty: got %q, want empty", p.SelectedWorkspace())
	}

	// Valid selection
	p.workspaces = []string{"default", "staging"}
	p.selected = 1
	if p.SelectedWorkspace() != "staging" {
		t.Errorf("SelectedWorkspace: got %q, want %q", p.SelectedWorkspace(), "staging")
	}
}

func TestSwitchToSelected(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 1

	cmd := p.SwitchToSelected()
	if cmd == nil {
		t.Error("SwitchToSelected: cmd = nil, want non-nil")
	}

	// Execute and verify message
	msg := cmd()
	switchMsg, ok := msg.(WorkspaceSwitchMsg)
	if !ok {
		t.Fatalf("SwitchToSelected cmd returned %T, want WorkspaceSwitchMsg", msg)
	}
	if switchMsg.Name != "staging" {
		t.Errorf("switchMsg.Name = %q, want %q", switchMsg.Name, "staging")
	}
}

func TestSwitchToSelectedSameWorkspace(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 0

	cmd := p.SwitchToSelected()
	if cmd != nil {
		t.Error("SwitchToSelected same ws: cmd != nil, want nil")
	}
}

func TestSwitchToSelectedEmpty(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.workspaces = []string{}

	cmd := p.SwitchToSelected()
	if cmd != nil {
		t.Error("SwitchToSelected empty: cmd != nil, want nil")
	}
}

func TestDeleteSelected(t *testing.T) {
	svc := &mockService{
		workspaceList: []string{"default", "staging"},
		workspace:     "default",
	}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.workspaces = []string{"default", "staging"}
	p.current = "default"
	p.selected = 1

	cmd := p.DeleteSelected()
	if cmd == nil {
		t.Error("DeleteSelected non-current/non-default: cmd = nil, want non-nil")
	}
}

func TestDeleteSelectedCurrent(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.workspaces = []string{"default", "staging"}
	p.current = "staging"
	p.selected = 1

	cmd := p.DeleteSelected()
	if cmd != nil {
		t.Error("DeleteSelected current: cmd != nil, want nil")
	}
}

func TestDeleteSelectedDefault(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.workspaces = []string{"default", "staging"}
	p.current = "staging"
	p.selected = 0 // "default"

	cmd := p.DeleteSelected()
	if cmd != nil {
		t.Error("DeleteSelected default: cmd != nil, want nil")
	}
}

func TestDeleteSelectedEmpty(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.workspaces = []string{}

	cmd := p.DeleteSelected()
	if cmd != nil {
		t.Error("DeleteSelected empty: cmd != nil, want nil")
	}
}

func TestRefresh(t *testing.T) {
	svc := &mockService{
		workspaceList: []string{"default"},
		workspace:     "default",
	}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.selected = 5
	p.creating = true
	p.newName = "test"

	cmd := p.Refresh()
	if cmd == nil {
		t.Error("Refresh() returned nil cmd")
	}
	if p.status != StatusLoading {
		t.Errorf("after Refresh: status = %v, want StatusLoading", p.status)
	}
	if p.creating {
		t.Error("after Refresh: creating = true, want false")
	}
	if p.newName != "" {
		t.Errorf("after Refresh: newName = %q, want empty", p.newName)
	}
}

func TestViewIdleAndLoading(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	p.status = StatusIdle
	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusIdle) returned empty string")
	}

	p.status = StatusLoading
	view = p.View(80, 24)
	if view == "" {
		t.Error("View(StatusLoading) returned empty string")
	}
}

func TestViewError(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusError
	p.errMsg = "connection failed"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusError) returned empty string")
	}
}

func TestViewDone(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default", "staging", "production"}
	p.current = "staging"
	p.selected = 1

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone) returned empty string")
	}
}

func TestViewDoneCreating(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default"}
	p.current = "default"
	p.creating = true
	p.newName = "my-new-ws"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, creating) returned empty string")
	}
}

func TestViewDefaultStatus(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestViewScrolling(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.current = "ws-0"

	workspaces := make([]string, 50)
	for i := range workspaces {
		workspaces[i] = "ws-" + string(rune('0'+i%10))
	}
	p.workspaces = workspaces
	p.selected = 40

	view := p.View(80, 10)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestCurrent(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.current = "staging"
	if p.Current() != "staging" {
		t.Errorf("Current() = %q, want %q", p.Current(), "staging")
	}
}

func TestWorkspaces(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.workspaces = []string{"a", "b"}
	if len(p.Workspaces()) != 2 {
		t.Errorf("Workspaces() len = %d, want 2", len(p.Workspaces()))
	}
}

func TestUpdateKeyMsgDown(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"a", "b"}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down: selected = %d, want 1", p.selected)
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 0 {
		t.Errorf("after up: selected = %d, want 0", p.selected)
	}
}

func TestViewDoneWithDefaultWorkspace(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.workspaces = []string{"default", "other"}
	p.current = "other"
	p.selected = 0

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with default workspace badge returned empty string")
	}
}

func TestUpdateKeyMsgCreating_DeleteKey(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.creating = true
	p.newName = "abc"

	p.stack.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if p.newName != "ab" {
		t.Errorf("after delete key in creating: newName = %q, want %q", p.newName, "ab")
	}
}

func TestStatusGetter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	if p.Status() != StatusIdle {
		t.Errorf("Status() = %v, want StatusIdle", p.Status())
	}
}

func TestSelectedGetter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.selected = 3
	if p.Selected() != 3 {
		t.Errorf("Selected() = %d, want 3", p.Selected())
	}
}

func TestActivateWithSessionContextChange(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyActiveScopeAbs, "/new/ctx")
	ctx := &sdk.Context{Service: svc, Session: session}
	p.Init(ctx)
	p.status = StatusDone
	p.scopedContext = "/old/ctx"
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() context change: want non-nil cmd")
	}
}

func TestActivateWithSameContext(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyActiveScopeAbs, "/same")
	ctx := &sdk.Context{Service: svc, Session: session}
	p.Init(ctx)
	p.status = StatusDone
	p.scopedContext = "/same"
	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() same context done: want nil cmd")
	}
}

func TestActivateMultiContextNoSelection(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyScopeCount, 3)
	ctx := &sdk.Context{Service: svc, Session: session}
	p.Init(ctx)
	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() multi-context no selection: want nil")
	}
	if p.status != StatusError {
		t.Errorf("status = %v, want StatusError", p.status)
	}
}

func TestActivateWithContextDir(t *testing.T) {
	svc := &mockService{workspaceList: []string{"default"}, workspace: "default"}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyActiveScopeAbs, "/my/ctx")
	ctx := &sdk.Context{Service: svc, Session: session}
	p.Init(ctx)
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() with context dir: want non-nil cmd")
	}
}

func TestCreateWorkspaceCmd(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc
	cmd := p.createWorkspace("new-ws")
	msg := cmd()
	sm := msg.(WorkspaceSwitchMsg)
	if sm.Name != "new-ws" || sm.Err != nil {
		t.Errorf("createWorkspace: Name=%q Err=%v", sm.Name, sm.Err)
	}
}

func TestCreateWorkspaceCmdError(t *testing.T) {
	svc := &mockService{workspaceNewErr: errors.New("exists")}
	p := New(svc).(*Plugin)
	p.svc = svc
	cmd := p.createWorkspace("x")
	msg := cmd()
	sm := msg.(WorkspaceSwitchMsg)
	if sm.Err == nil {
		t.Error("createWorkspace error: want non-nil Err")
	}
}

func TestDeleteSelectedCmd(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc
	p.workspaces = []string{"default", "staging", "dev"}
	p.current = "default"
	p.selected = 2
	cmd := p.DeleteSelected()
	if cmd == nil {
		t.Fatal("DeleteSelected: want non-nil cmd")
	}
	msg := cmd()
	sm := msg.(WorkspaceSwitchMsg)
	if sm.Err != nil {
		t.Errorf("DeleteSelected: Err = %v", sm.Err)
	}
}

func TestSwitchWorkspaceCmd(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.svc = svc
	cmd := p.switchWorkspace("staging")
	msg := cmd()
	sm := msg.(WorkspaceSwitchMsg)
	if sm.Name != "staging" || sm.Err != nil {
		t.Errorf("switchWorkspace: Name=%q Err=%v", sm.Name, sm.Err)
	}
}

func TestSwitchWorkspaceCmdError(t *testing.T) {
	svc := &mockService{workspaceSelectErr: errors.New("fail")}
	p := New(svc).(*Plugin)
	p.svc = svc
	cmd := p.switchWorkspace("x")
	msg := cmd()
	sm := msg.(WorkspaceSwitchMsg)
	if sm.Err == nil {
		t.Error("switchWorkspace error: want non-nil Err")
	}
}
