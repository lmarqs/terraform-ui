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
	if p.KeyBinding() != "c" {
		t.Errorf("KeyBinding() = %q, want %q", p.KeyBinding(), "c")
	}
	if p.Ready() {
		t.Error("Ready() = true before discovery, want false")
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
		Workspace: "default",
		Service:   svc,
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() returned nil cmd, should return nil (no auto-load)")
	}
	if p.status != StatusIdle {
		t.Errorf("status = %v, want StatusIdle", p.status)
	}
}

func TestInitCmdReturnsContextDiscoveredMsg(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	// Use a cfg with no patterns (returns just the Dir)
	p.cfg = config.Config{Dir: "."}

	ctx := &sdk.Context{Service: svc}
	p.Init(ctx)
	cmd := p.Activate()
	msg := cmd()

	result, ok := msg.(ContextDiscoveredMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T, want ContextDiscoveredMsg", msg)
	}
	if result.Err != nil {
		t.Errorf("ContextDiscoveredMsg.Err = %v, want nil", result.Err)
	}
	if len(result.Projects) == 0 {
		t.Error("len(Projects) = 0, want at least 1 (the dir itself)")
	}
}

func TestUpdateContextDiscoveredMsgSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	projects := []Project{
		{Path: "modules/vpc", Name: "vpc", AbsPath: "/tmp/modules/vpc"},
		{Path: "modules/rds", Name: "rds", AbsPath: "/tmp/modules/rds"},
	}

	result, cmd := p.Update(ContextDiscoveredMsg{Projects: projects, Err: nil})
	if cmd != nil {
		t.Errorf("Update(ContextDiscoveredMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusDone {
		t.Errorf("status = %v, want StatusDone", updated.status)
	}
	if len(updated.projects) != 2 {
		t.Errorf("len(projects) = %d, want 2", len(updated.projects))
	}
	if len(updated.filtered) != 2 {
		t.Errorf("len(filtered) = %d, want 2", len(updated.filtered))
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdateContextDiscoveredMsgError(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	result, cmd := p.Update(ContextDiscoveredMsg{Err: errTest})
	if cmd != nil {
		t.Errorf("Update(ContextDiscoveredMsg) cmd = %v, want nil", cmd)
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
	p.projects = []Project{
		{Path: "a", Name: "a"},
		{Path: "b", Name: "b"},
		{Path: "c", Name: "c"},
	}
	p.filtered = p.projects

	// Move down with j
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", p.selected)
	}

	// Move down more
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 2 {
		t.Errorf("after j,j: selected = %d, want 2", p.selected)
	}

	// Boundary
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 2 {
		t.Errorf("after j,j,j: selected = %d, want 2 (boundary)", p.selected)
	}

	// Move up with k
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 1 {
		t.Errorf("after k: selected = %d, want 1", p.selected)
	}

	// Move up to start
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 0 {
		t.Errorf("after k,k: selected = %d, want 0", p.selected)
	}

	// Boundary
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 0 {
		t.Errorf("after k,k,k: selected = %d, want 0 (boundary)", p.selected)
	}
}

func TestUpdateKeyMsgEnter_SelectCurrent(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{
		{Path: "a", Name: "a"},
		{Path: "b", Name: "b"},
	}
	p.filtered = p.projects
	p.selected = 1

	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.active != 1 {
		t.Errorf("after enter: active = %d, want 1", p.active)
	}
}

func TestUpdateKeyMsgBackspace(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{{Path: "abc"}}
	p.filtered = p.projects
	p.filter = "ab"

	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.filter != "a" {
		t.Errorf("after backspace: filter = %q, want %q", p.filter, "a")
	}
}

func TestUpdateKeyMsgR_Refresh(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.cfg = config.Config{Dir: "."}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
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
	p.filtered = []Project{{Path: "a"}, {Path: "b"}, {Path: "c"}}

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
	p.filtered = []Project{}
	p.MoveDown()
	if p.selected != 0 {
		t.Errorf("MoveDown empty: selected = %d, want 0", p.selected)
	}
}

func TestSelectCurrent(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.projects = []Project{
		{Path: "a"},
		{Path: "b"},
		{Path: "c"},
	}
	p.filtered = p.projects
	p.selected = 2

	p.SelectCurrent()
	if p.active != 2 {
		t.Errorf("SelectCurrent: active = %d, want 2", p.active)
	}
}

func TestSelectCurrentFiltered(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.projects = []Project{
		{Path: "a"},
		{Path: "b"},
		{Path: "c"},
	}
	// Filtered shows only "b" and "c"
	p.filtered = []Project{{Path: "b"}, {Path: "c"}}
	p.selected = 0

	p.SelectCurrent()
	if p.active != 1 {
		t.Errorf("SelectCurrent filtered: active = %d, want 1 (index of 'b' in projects)", p.active)
	}
}

func TestSelectCurrentOutOfBounds(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.projects = []Project{{Path: "a"}}
	p.filtered = []Project{}
	p.selected = 5

	// Should not panic
	p.SelectCurrent()
}

func TestActiveProject(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.projects = []Project{{Path: "a"}, {Path: "b"}}
	p.active = 1

	ap := p.ActiveProject()
	if ap == nil {
		t.Fatal("ActiveProject() = nil, want non-nil")
	}
	if ap.Path != "b" {
		t.Errorf("ActiveProject().Path = %q, want %q", ap.Path, "b")
	}
}

func TestActiveProjectOutOfBounds(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.projects = []Project{}
	p.active = 5

	if p.ActiveProject() != nil {
		t.Error("ActiveProject() out of bounds: want nil")
	}
}

func TestSelectedProject(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []Project{{Path: "a"}, {Path: "b"}}
	p.selected = 1

	sp := p.SelectedProject()
	if sp == nil {
		t.Fatal("SelectedProject() = nil, want non-nil")
	}
	if sp.Path != "b" {
		t.Errorf("SelectedProject().Path = %q, want %q", sp.Path, "b")
	}
}

func TestSelectedProjectOutOfBounds(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []Project{}
	p.selected = 5

	if p.SelectedProject() != nil {
		t.Error("SelectedProject() out of bounds: want nil")
	}
}

func TestSetFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.projects = []Project{
		{Path: "modules/vpc", Name: "vpc"},
		{Path: "modules/rds", Name: "rds"},
		{Path: "envs/prod", Name: "prod"},
	}
	p.filtered = p.projects

	// Filter by path
	p.SetFilter("vpc")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('vpc'): len(filtered) = %d, want 1", len(p.filtered))
	}
	if p.selected != 0 {
		t.Errorf("SetFilter resets selected: got %d, want 0", p.selected)
	}
	if p.filter != "vpc" {
		t.Errorf("filter = %q, want %q", p.filter, "vpc")
	}

	// Filter by name
	p.SetFilter("rds")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('rds'): len(filtered) = %d, want 1", len(p.filtered))
	}

	// Clear filter
	p.SetFilter("")
	if len(p.filtered) != 3 {
		t.Errorf("SetFilter(''): len(filtered) = %d, want 3", len(p.filtered))
	}

	// Case insensitive
	p.SetFilter("VPC")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('VPC'): len(filtered) = %d, want 1", len(p.filtered))
	}

	// No matches
	p.SetFilter("zzz")
	if len(p.filtered) != 0 {
		t.Errorf("SetFilter('zzz'): len(filtered) = %d, want 0", len(p.filtered))
	}
}

func TestAppendFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.projects = []Project{{Path: "abc"}}
	p.filtered = p.projects

	p.AppendFilter("a")
	if p.filter != "a" {
		t.Errorf("AppendFilter('a'): filter = %q, want %q", p.filter, "a")
	}
	p.AppendFilter("b")
	if p.filter != "ab" {
		t.Errorf("AppendFilter('b'): filter = %q, want %q", p.filter, "ab")
	}
}

func TestBackspaceFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.projects = []Project{{Path: "abc"}}
	p.filtered = p.projects
	p.filter = "abc"

	p.BackspaceFilter()
	if p.filter != "ab" {
		t.Errorf("BackspaceFilter: filter = %q, want %q", p.filter, "ab")
	}

	// Backspace on empty
	p.filter = ""
	p.BackspaceFilter()
	if p.filter != "" {
		t.Errorf("BackspaceFilter empty: filter = %q, want empty", p.filter)
	}
}

func TestRefresh(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.filter = "something"
	p.cfg = config.Config{Dir: "."}

	cmd := p.Refresh()
	if cmd == nil {
		t.Error("Refresh() returned nil cmd")
	}
	if p.status != StatusLoading {
		t.Errorf("after Refresh: status = %v, want StatusLoading", p.status)
	}
	if p.filter != "" {
		t.Errorf("after Refresh: filter = %q, want empty", p.filter)
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

func TestViewDone_NoProjects(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{}
	p.filtered = []Project{}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, no projects) returned empty string")
	}
}

func TestViewDone_WithProjects(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{
		{Path: "modules/vpc", Name: "vpc", AbsPath: "/tmp/modules/vpc"},
		{Path: "modules/rds", Name: "rds", AbsPath: "/tmp/modules/rds"},
	}
	p.filtered = p.projects
	p.active = 0

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, with projects) returned empty string")
	}
}

func TestViewDone_WithFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{{Path: "a"}, {Path: "b"}}
	p.filtered = p.projects[:1]
	p.filter = "a"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, with filter) returned empty string")
	}
}

func TestViewDone_FilterNoMatch(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{{Path: "a"}}
	p.filtered = []Project{}
	p.filter = "zzz"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, filter no match) returned empty string")
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

	projects := make([]Project, 50)
	for i := range projects {
		projects[i] = Project{Path: "module_" + string(rune('a'+i%26)), Name: "m" + string(rune('a'+i%26))}
	}
	p.projects = projects
	p.filtered = projects
	p.selected = 40

	view := p.View(80, 10)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestViewDone_ProjectWithDifferentName(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{
		{Path: "envs/production/us-east-1", Name: "us-east-1", AbsPath: "/tmp/envs/production/us-east-1"},
	}
	p.filtered = p.projects
	p.active = 0

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with project name != path returned empty string")
	}
}

func TestViewDone_ProjectSameNameAsPath(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{
		{Path: "vpc", Name: "vpc"},
	}
	p.filtered = p.projects
	p.active = 0

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with same name and path returned empty string")
	}
}

func TestDeriveProjectName(t *testing.T) {
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
		got := deriveProjectName(tt.path)
		if got != tt.want {
			t.Errorf("deriveProjectName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestContextCount(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.projects = []Project{{}, {}, {}}
	if p.ContextCount() != 3 {
		t.Errorf("ContextCount() = %d, want 3", p.ContextCount())
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

func TestFilterGetter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filter = "test"
	if p.Filter() != "test" {
		t.Errorf("Filter() = %q, want %q", p.Filter(), "test")
	}
}

func TestUpdateKeyMsgSlash(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{{Path: "a"}}
	p.filtered = p.projects

	// "/" key should not crash (handled but empty)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if cmd != nil {
		t.Error("after /: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgDown(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{{Path: "a"}, {Path: "b"}}
	p.filtered = p.projects

	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down: selected = %d, want 1", p.selected)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 0 {
		t.Errorf("after up: selected = %d, want 0", p.selected)
	}
}

func TestUpdateKeyMsgDelete(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{{Path: "abc"}}
	p.filtered = p.projects
	p.filter = "ab"

	p.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if p.filter != "a" {
		t.Errorf("after delete: filter = %q, want %q", p.filter, "a")
	}
}

func TestUpdateKeyMsgDefaultChar(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.projects = []Project{{Path: "a"}}
	p.filtered = p.projects

	// Default character handling (not j/k/r/enter/backspace/etc.)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("after x: cmd != nil, want nil")
	}
}
