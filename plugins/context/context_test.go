package context

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct{}

func (m *mockService) Plan(_ context.Context, _ []string) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (m *mockService) Apply(_ context.Context, _ []string) error           { return nil }
func (m *mockService) StateList(_ context.Context) ([]sdk.Resource, error) { return nil, nil }
func (m *mockService) Show(_ context.Context, _ string) (string, error)    { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)         { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"default"}, nil
}
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error            { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string) error               { return nil }
func (m *mockService) WorkspaceDelete(_ context.Context, _ string) error            { return nil }
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

	if p.ID() != "context" {
		t.Errorf("ID() = %q, want %q", p.ID(), "context")
	}
	if p.Name() != "Context" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Context")
	}
	if p.Description() != "Select terraform project scope" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Select terraform project scope")
	}
	if p.Ready() {
		t.Error("Ready() = true before discovery, want false")
	}
}

func TestNew_ActiveIsMinusOne(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	if p.active != -1 {
		t.Errorf("active = %d, want -1", p.active)
	}
	if p.ActiveScope() != nil {
		t.Error("ActiveScope() != nil for new plugin, want nil")
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

func TestSetConfig(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	cfg := config.Config{Dir: "/tmp/my-project"}
	p.SetConfig(cfg)
	if p.cfg.Dir != "/tmp/my-project" {
		t.Errorf("cfg.Dir = %q, want %q", p.cfg.Dir, "/tmp/my-project")
	}
}

func TestInit(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.cfg = config.Config{Dir: "/tmp"}

	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Workspace:  "default",
		Service:    svc,
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() returned non-nil cmd, should return nil (no auto-load)")
	}
	if p.status != StatusIdle {
		t.Errorf("status = %v, want StatusIdle", p.status)
	}
	if p.active != -1 {
		t.Errorf("after Init: active = %d, want -1", p.active)
	}
}

func TestInitCmdReturnsScopeDiscoveredMsg(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.cfg = config.Config{Dir: "."}

	ctx := &sdk.Context{Service: svc}
	p.Init(ctx)
	cmd := p.Activate()
	msg := cmd()

	result, ok := msg.(ScopeDiscoveredMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T, want ScopeDiscoveredMsg", msg)
	}
	if result.Err != nil {
		t.Errorf("ScopeDiscoveredMsg.Err = %v, want nil", result.Err)
	}
	if len(result.Scopes) == 0 {
		t.Error("len(Scopes) = 0, want at least 1 (the dir itself)")
	}
}

func TestUpdateScopeDiscoveredMsgSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	scopes := []Scope{
		{Path: "modules/vpc", Name: "vpc", AbsPath: "/tmp/modules/vpc"},
		{Path: "modules/rds", Name: "rds", AbsPath: "/tmp/modules/rds"},
	}

	result, cmd := p.Update(ScopeDiscoveredMsg{Scopes: scopes, Err: nil})
	if cmd != nil {
		t.Errorf("Update(ScopeDiscoveredMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusDone {
		t.Errorf("status = %v, want StatusDone", updated.status)
	}
	if len(updated.scopes) != 2 {
		t.Errorf("len(scopes) = %d, want 2", len(updated.scopes))
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdateScopeDiscoveredMsgError(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	result, cmd := p.Update(ScopeDiscoveredMsg{Err: errTest})
	if cmd != nil {
		t.Errorf("Update(ScopeDiscoveredMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusError {
		t.Errorf("status = %v, want StatusError", updated.status)
	}
	if updated.errMsg != "test error" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "test error")
	}
}

var errTest = testError("test error")

type testError string

func (e testError) Error() string { return string(e) }

func TestUpdateKeyMsgNavigation(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{
		{Path: "a", Name: "a"},
		{Path: "b", Name: "b"},
		{Path: "c", Name: "c"},
	}

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

	// Move up to start
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 0 {
		t.Errorf("after k,k: selected = %d, want 0", p.selected)
	}

	// Boundary
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 0 {
		t.Errorf("after k,k,k: selected = %d, want 0 (boundary)", p.selected)
	}
}

func TestUpdateKeyMsgEnter_SelectAndDeactivate(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{
		{Path: "a", Name: "a", AbsPath: "/tmp/a"},
		{Path: "b", Name: "b", AbsPath: "/tmp/b"},
	}
	p.selected = 1

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.active != 1 {
		t.Errorf("after enter: active = %d, want 1", p.active)
	}
	if cmd == nil {
		t.Fatal("after enter: cmd = nil, want DeactivateMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd returned %T, want sdk.DeactivateMsg", msg)
	}
}

func TestUpdateKeyMsgR_Refresh(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.cfg = config.Config{Dir: "."}

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
	p.scopes = []Scope{{Path: "a"}, {Path: "b"}, {Path: "c"}}

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

func TestMoveDownEmpty(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.scopes = []Scope{}
	p.MoveDown()
	if p.selected != 0 {
		t.Errorf("MoveDown empty: selected = %d, want 0", p.selected)
	}
}

func TestSelectCurrent(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.scopes = []Scope{
		{Path: "a"},
		{Path: "b"},
		{Path: "c"},
	}
	p.selected = 2

	cmd := p.SelectCurrent()
	if p.active != 2 {
		t.Errorf("SelectCurrent: active = %d, want 2", p.active)
	}
	if cmd == nil {
		t.Fatal("SelectCurrent: cmd = nil, want DeactivateMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("SelectCurrent cmd returned %T, want sdk.DeactivateMsg", msg)
	}
}

func TestSelectCurrentWithSession(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.session = sdk.NewSession()
	p.scopes = []Scope{
		{Path: "modules/vpc", AbsPath: "/tmp/modules/vpc"},
		{Path: "modules/rds", AbsPath: "/tmp/modules/rds"},
	}
	p.selected = 1

	p.SelectCurrent()

	ctx, ok := sdk.GetTyped[string](p.session, sdk.SessionKeyActiveScope)
	if !ok || ctx != "modules/rds" {
		t.Errorf("session context = %q, want %q", ctx, "modules/rds")
	}
	abs, ok := sdk.GetTyped[string](p.session, sdk.SessionKeyActiveScopeAbs)
	if !ok || abs != "/tmp/modules/rds" {
		t.Errorf("session context abs = %q, want %q", abs, "/tmp/modules/rds")
	}
}

func TestSelectCurrentOutOfBounds(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.scopes = []Scope{{Path: "a"}}
	p.selected = 5

	cmd := p.SelectCurrent()
	if cmd != nil {
		t.Error("SelectCurrent out of bounds: cmd != nil, want nil")
	}
}

func TestActiveScope(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.scopes = []Scope{{Path: "a"}, {Path: "b"}}
	p.active = 1

	ap := p.ActiveScope()
	if ap == nil {
		t.Fatal("ActiveScope() = nil, want non-nil")
	}
	if ap.Path != "b" {
		t.Errorf("ActiveScope().Path = %q, want %q", ap.Path, "b")
	}
}

func TestActiveScopeNoSelection(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.scopes = []Scope{{Path: "a"}, {Path: "b"}}

	if p.ActiveScope() != nil {
		t.Error("ActiveScope() != nil for new plugin (active=-1), want nil")
	}
}

func TestActiveScopeOutOfBounds(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.scopes = []Scope{}
	p.active = 5

	if p.ActiveScope() != nil {
		t.Error("ActiveScope() out of bounds: want nil")
	}
}

func TestSelectedScope(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.scopes = []Scope{{Path: "a"}, {Path: "b"}}
	p.selected = 1

	sp := p.SelectedScope()
	if sp == nil {
		t.Fatal("SelectedScope() = nil, want non-nil")
	}
	if sp.Path != "b" {
		t.Errorf("SelectedScope().Path = %q, want %q", sp.Path, "b")
	}
}

func TestSelectedScopeOutOfBounds(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.scopes = []Scope{}
	p.selected = 5

	if p.SelectedScope() != nil {
		t.Error("SelectedScope() out of bounds: want nil")
	}
}

func TestRefresh(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.cfg = config.Config{Dir: "."}

	cmd := p.Refresh()
	if cmd == nil {
		t.Error("Refresh() returned nil cmd")
	}
	if p.status != StatusLoading {
		t.Errorf("after Refresh: status = %v, want StatusLoading", p.status)
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
	p.errMsg = "discovery failed"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusError) returned empty string")
	}
}

func TestViewDone_NoScopes(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, no scopes) returned empty string")
	}
}

func TestViewDone_WithScopes(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{
		{Path: "modules/vpc", Name: "vpc", AbsPath: "/tmp/modules/vpc"},
		{Path: "modules/rds", Name: "rds", AbsPath: "/tmp/modules/rds"},
	}
	p.active = 0

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, with scopes) returned empty string")
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

	scopes := make([]Scope, 50)
	for i := range scopes {
		scopes[i] = Scope{Path: "module_" + string(rune('a'+i%26)), Name: "m" + string(rune('a'+i%26))}
	}
	p.scopes = scopes
	p.selected = 40

	view := p.View(80, 10)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestDeriveScopeName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"modules/vpc", "vpc"},
		{"envs/production/us-east-1", "us-east-1"},
		{"simple", "simple"},
		{".", "."},
		{"/", "/"},
	}

	for _, tt := range tests {
		got := deriveScopeName(tt.path)
		if got != tt.want {
			t.Errorf("deriveScopeName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestScopeCount(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.scopes = []Scope{{}, {}, {}}
	if p.ScopeCount() != 3 {
		t.Errorf("ScopeCount() = %d, want 3", p.ScopeCount())
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

func TestActiveGetter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.active = 2
	if p.Active() != 2 {
		t.Errorf("Active() = %d, want 2", p.Active())
	}
}

func TestUpdateKeyMsgDown(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{{Path: "a"}, {Path: "b"}}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down: selected = %d, want 1", p.selected)
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 0 {
		t.Errorf("after up: selected = %d, want 0", p.selected)
	}
}

func TestUpdateKeyMsgUnhandled(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{{Path: "a"}}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("after x: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgEsc_Deactivate(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{{Path: "a"}}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("after esc: cmd = nil, want DeactivateMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("cmd returned %T, want sdk.DeactivateMsg", msg)
	}
}

func TestStackHints_Done(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.scopes = []Scope{{Path: "a"}}

	hints := p.stack.Hints()
	if len(hints) != 4 {
		t.Fatalf("Hints() len = %d, want 4", len(hints))
	}
	if hints[0].Key != "↑↓" {
		t.Errorf("hints[0].Key = %q, want %q", hints[0].Key, "↑↓")
	}
	if hints[3].Key != "q" {
		t.Errorf("hints[3].Key = %q, want %q", hints[3].Key, "q")
	}
}

func TestStackHints_Error(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusError

	hints := p.stack.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() len = %d, want 2", len(hints))
	}
	if hints[0].Key != "r" {
		t.Errorf("hints[0].Key = %q, want %q", hints[0].Key, "r")
	}
}

func TestStackHints_Loading(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusLoading

	hints := p.stack.Hints()
	if hints != nil {
		t.Errorf("Hints() = %v, want nil during loading", hints)
	}
}

func TestActivate_OnlyWhenIdleOrError(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.cfg = config.Config{Dir: "."}

	// StatusDone should not re-activate
	p.status = StatusDone
	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() when Done: cmd != nil, want nil")
	}

	// StatusLoading should not re-activate
	p.status = StatusLoading
	cmd = p.Activate()
	if cmd != nil {
		t.Error("Activate() when Loading: cmd != nil, want nil")
	}

	// StatusIdle should activate
	p.status = StatusIdle
	cmd = p.Activate()
	if cmd == nil {
		t.Error("Activate() when Idle: cmd = nil, want non-nil")
	}

	// StatusError should activate
	p.status = StatusError
	cmd = p.Activate()
	if cmd == nil {
		t.Error("Activate() when Error: cmd = nil, want non-nil")
	}
}
