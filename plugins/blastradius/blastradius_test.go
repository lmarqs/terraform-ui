package blastradius

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct{}

func (m *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error   { return nil }
func (m *mockService) StateList(_ context.Context) ([]sdk.Resource, error) { return nil, nil }
func (m *mockService) Show(_ context.Context, _ string) (string, error)    { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)         { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"default"}, nil
}
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	return nil
}
func (m *mockService) WorkspaceDelete(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	return nil
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

	if p.ID() != "blastradius" {
		t.Errorf("ID() = %q, want %q", p.ID(), "blastradius")
	}
	if p.Name() != "Blast Radius" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Blast Radius")
	}
	if p.Description() != "Visualize module-grouped changes with impact scores" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Visualize module-grouped changes with impact scores")
	}
	if p.Ready() {
		t.Error("Ready() = true before analysis, want false")
	}
}

func TestCountable(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	var c sdk.Countable = p
	filtered, total := c.Count()
	if filtered != 0 || total != 0 {
		t.Errorf("Count() = (%d, %d), want (0, 0) when empty", filtered, total)
	}

	p.modules = []ModuleImpact{{Group: sdk.ModuleGroup{Module: "mod_a"}}, {Group: sdk.ModuleGroup{Module: "mod_b"}}}
	p.total = 5
	filtered, total = c.Count()
	if filtered != 2 || total != 5 {
		t.Errorf("Count() = (%d, %d), want (2, 5)", filtered, total)
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
	svc := &mockService{}
	p := New(svc)
	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Workspace:  "default",
		Service:    svc,
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() returned non-nil cmd, want nil")
	}
}

func TestAnalyzeNilSummary(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	p.Analyze(nil)
	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.modules != nil {
		t.Error("modules = non-nil, want nil")
	}
	if p.total != 0 {
		t.Errorf("total = %d, want 0", p.total)
	}
	if !p.Ready() {
		t.Error("Ready() = false after Analyze(nil), want true")
	}
}

func TestAnalyzeEmptyChanges(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	p.Analyze(&sdk.PlanSummary{Changes: []sdk.PlanChange{}})
	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.modules != nil {
		t.Error("modules = non-nil after empty changes, want nil")
	}
}

func TestAnalyzeWithChanges(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web", Module: ""}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a", Module: "module.vpc"}, Action: sdk.ActionUpdate, Risk: sdk.RiskMedium},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b", Module: "module.vpc"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh},
			{Resource: sdk.Resource{Address: "module.rds.aws_db_instance.main", Module: "module.rds"}, Action: sdk.ActionDeleteThenCreate, Risk: sdk.RiskCritical},
		},
	}

	p.Analyze(summary)

	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.TotalChanges() != 4 {
		t.Errorf("TotalChanges() = %d, want 4", p.TotalChanges())
	}
	if p.ModuleCount() == 0 {
		t.Error("ModuleCount() = 0, want > 0")
	}
	if p.selected != 0 {
		t.Errorf("selected = %d, want 0 after Analyze", p.selected)
	}

	// Highest impact should be first (sorted by impact score)
	if p.modules[0].Score < p.modules[len(p.modules)-1].Score {
		t.Error("modules not sorted by impact score descending")
	}
}

func TestAnalyzeSingleModule(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web", Module: ""}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
		},
	}

	p.Analyze(summary)
	if p.ModuleCount() != 1 {
		t.Errorf("ModuleCount() = %d, want 1", p.ModuleCount())
	}
}

func TestStatus(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	if p.Status() != sdk.StatusIdle {
		t.Errorf("Status() = %v, want sdk.StatusIdle", p.Status())
	}
}

func TestSelected(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.selected = 3
	if p.Selected() != 3 {
		t.Errorf("Selected() = %d, want 3", p.Selected())
	}
}

func TestUpdateKeyMsgNavigation(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.modules = []ModuleImpact{
		{Group: sdk.ModuleGroup{Module: "root"}, Score: ImpactMinimal},
		{Group: sdk.ModuleGroup{Module: "module.vpc"}, Score: ImpactHigh},
		{Group: sdk.ModuleGroup{Module: "module.rds"}, Score: ImpactCritical},
	}

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

func TestUpdateKeyMsgToggleExpand(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.modules = []ModuleImpact{
		{Group: sdk.ModuleGroup{Module: "root", Changes: []sdk.PlanChange{{}}}},
	}

	// Toggle with enter
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !p.expanded[0] {
		t.Error("after enter: expanded[0] = false, want true")
	}

	// Toggle again
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.expanded[0] {
		t.Error("after enter,enter: expanded[0] = true, want false")
	}

	// Toggle with i
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if !p.expanded[0] {
		t.Error("after i: expanded[0] = false, want true")
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
	p.modules = []ModuleImpact{{}, {}, {}}

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
	p.modules = []ModuleImpact{}
	p.MoveDown()
	if p.selected != 0 {
		t.Errorf("MoveDown empty: selected = %d, want 0", p.selected)
	}
}

func TestToggleExpand(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.selected = 1

	p.ToggleExpand()
	if !p.expanded[1] {
		t.Error("ToggleExpand: expanded[1] = false, want true")
	}
	p.ToggleExpand()
	if p.expanded[1] {
		t.Error("ToggleExpand: expanded[1] = true, want false")
	}
}

func TestSelectedModule(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	// Empty modules
	p.modules = []ModuleImpact{}
	if p.SelectedModule() != nil {
		t.Error("SelectedModule empty: want nil")
	}

	// Valid selection
	p.modules = []ModuleImpact{
		{Group: sdk.ModuleGroup{Module: "root"}, Score: ImpactMinimal},
		{Group: sdk.ModuleGroup{Module: "module.vpc"}, Score: ImpactHigh},
	}
	p.selected = 1
	sm := p.SelectedModule()
	if sm == nil {
		t.Fatal("SelectedModule: got nil")
	}
	if sm.Group.Module != "module.vpc" {
		t.Errorf("SelectedModule.Group.Module = %q, want %q", sm.Group.Module, "module.vpc")
	}
}

func TestSelectedModuleOutOfBounds(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.modules = []ModuleImpact{{}}
	p.selected = 5

	if p.SelectedModule() != nil {
		t.Error("SelectedModule out of bounds: want nil")
	}
}

func TestViewIdle(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusIdle) returned empty string")
	}
}

func TestViewReady_NoModules(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.modules = nil

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, no modules) returned empty string")
	}
}

func TestViewReady_EmptyModules(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.modules = []ModuleImpact{}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, empty modules) returned empty string")
	}
}

func TestViewReady_WithModules(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.total = 4
	p.modules = []ModuleImpact{
		{
			Group: sdk.ModuleGroup{
				Module: "module.rds",
				Changes: []sdk.PlanChange{
					{Resource: sdk.Resource{Address: "module.rds.aws_db_instance.main", Module: "module.rds"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical},
				},
				Summary: sdk.ActionSummary{Destroy: 1},
			},
			Score: ImpactCritical,
		},
		{
			Group: sdk.ModuleGroup{
				Module: "module.vpc",
				Changes: []sdk.PlanChange{
					{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a", Module: "module.vpc"}, Action: sdk.ActionUpdate, Risk: sdk.RiskMedium},
					{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b", Module: "module.vpc"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow, IsPhantom: true},
				},
				Summary: sdk.ActionSummary{Change: 1, Add: 1},
			},
			Score: ImpactModerate,
		},
		{
			Group: sdk.ModuleGroup{
				Module: "root",
				Changes: []sdk.PlanChange{
					{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
				},
				Summary: sdk.ActionSummary{Add: 1},
			},
			Score: ImpactMinimal,
		},
	}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, with modules) returned empty string")
	}
}

func TestViewReady_WithExpanded(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.total = 2
	p.modules = []ModuleImpact{
		{
			Group: sdk.ModuleGroup{
				Module: "module.vpc",
				Changes: []sdk.PlanChange{
					{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a", Module: "module.vpc"}, Action: sdk.ActionUpdate, Risk: sdk.RiskMedium},
					{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b", Module: "module.vpc"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh, IsPhantom: true},
				},
				Summary: sdk.ActionSummary{Change: 1, Destroy: 1},
			},
			Score: ImpactHigh,
		},
	}
	p.expanded = map[int]bool{0: true}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with expanded module returned empty string")
	}
}

func TestViewReady_WithReplaceAction(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.total = 1
	p.modules = []ModuleImpact{
		{
			Group: sdk.ModuleGroup{
				Module: "root",
				Changes: []sdk.PlanChange{
					{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreateThenDelete, Risk: sdk.RiskHigh},
				},
				Summary: sdk.ActionSummary{Replace: 1},
			},
			Score: ImpactHigh,
		},
	}
	p.expanded = map[int]bool{0: true}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with replace action returned empty string")
	}
}

func TestViewDefaultStatus(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestViewSmallHeight(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.total = 1
	p.modules = []ModuleImpact{
		{Group: sdk.ModuleGroup{Module: "root", Changes: []sdk.PlanChange{{}}}, Score: ImpactMinimal},
	}

	view := p.View(80, 5)
	if view == "" {
		t.Error("View with small height returned empty string")
	}
}

func TestViewScrolling(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.total = 20

	modules := make([]ModuleImpact, 20)
	for i := range modules {
		modules[i] = ModuleImpact{
			Group: sdk.ModuleGroup{Module: "module_" + string(rune('a'+i%26)), Changes: []sdk.PlanChange{{}}},
			Score: ImpactMinimal,
		}
	}
	p.modules = modules
	p.selected = 15

	view := p.View(80, 10)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestCalculateImpact(t *testing.T) {
	tests := []struct {
		name  string
		group sdk.ModuleGroup
		want  ImpactScore
	}{
		{
			name: "critical risk resource",
			group: sdk.ModuleGroup{
				Changes: []sdk.PlanChange{
					{Risk: sdk.RiskCritical, Action: sdk.ActionUpdate},
				},
			},
			want: ImpactCritical,
		},
		{
			name: "high risk resource",
			group: sdk.ModuleGroup{
				Changes: []sdk.PlanChange{
					{Risk: sdk.RiskHigh, Action: sdk.ActionUpdate},
				},
			},
			want: ImpactHigh,
		},
		{
			name: "destructive with 3+ changes",
			group: sdk.ModuleGroup{
				Changes: []sdk.PlanChange{
					{Risk: sdk.RiskMedium, Action: sdk.ActionDelete},
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
				},
			},
			want: ImpactHigh,
		},
		{
			name: "medium risk",
			group: sdk.ModuleGroup{
				Changes: []sdk.PlanChange{
					{Risk: sdk.RiskMedium, Action: sdk.ActionUpdate},
				},
			},
			want: ImpactModerate,
		},
		{
			name: "3+ changes low risk",
			group: sdk.ModuleGroup{
				Changes: []sdk.PlanChange{
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
				},
			},
			want: ImpactModerate,
		},
		{
			name: "minimal - 1 low risk create",
			group: sdk.ModuleGroup{
				Changes: []sdk.PlanChange{
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
				},
			},
			want: ImpactMinimal,
		},
		{
			name: "minimal - 2 low risk",
			group: sdk.ModuleGroup{
				Changes: []sdk.PlanChange{
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
					{Risk: sdk.RiskNone, Action: sdk.ActionCreate},
				},
			},
			want: ImpactMinimal,
		},
		{
			name: "delete then create counts as destructive",
			group: sdk.ModuleGroup{
				Changes: []sdk.PlanChange{
					{Risk: sdk.RiskMedium, Action: sdk.ActionDeleteThenCreate},
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
				},
			},
			want: ImpactHigh,
		},
		{
			name: "create then delete counts as destructive",
			group: sdk.ModuleGroup{
				Changes: []sdk.PlanChange{
					{Risk: sdk.RiskMedium, Action: sdk.ActionCreateThenDelete},
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
					{Risk: sdk.RiskLow, Action: sdk.ActionCreate},
				},
			},
			want: ImpactHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateImpact(tt.group)
			if got != tt.want {
				t.Errorf("calculateImpact() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortByImpact(t *testing.T) {
	modules := []ModuleImpact{
		{Score: ImpactMinimal},
		{Score: ImpactCritical},
		{Score: ImpactModerate},
		{Score: ImpactHigh},
	}

	sortByImpact(modules)

	for i := 1; i < len(modules); i++ {
		if modules[i].Score > modules[i-1].Score {
			t.Errorf("sortByImpact: modules[%d].Score (%v) > modules[%d].Score (%v)", i, modules[i].Score, i-1, modules[i-1].Score)
		}
	}
}

func TestSortByImpactEmpty(t *testing.T) {
	modules := []ModuleImpact{}
	sortByImpact(modules) // Should not panic
}

func TestSortByImpactSingle(t *testing.T) {
	modules := []ModuleImpact{{Score: ImpactHigh}}
	sortByImpact(modules)
	if modules[0].Score != ImpactHigh {
		t.Error("sortByImpact single: unexpected change")
	}
}

func TestImpactScoreString(t *testing.T) {
	tests := []struct {
		score ImpactScore
		want  string
	}{
		{ImpactMinimal, "minimal"},
		{ImpactModerate, "moderate"},
		{ImpactHigh, "high"},
		{ImpactCritical, "critical"},
		{ImpactScore(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.score.String()
		if got != tt.want {
			t.Errorf("ImpactScore(%d).String() = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestHints_WhenDoneWithModules_ShouldIncludeInspectAndBack(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.modules = []ModuleImpact{
		{Group: sdk.ModuleGroup{Module: "root"}, Score: ImpactMinimal},
	}

	hints := p.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() returned %d hints, want 2", len(hints))
	}
	if hints[0].Key != "Enter" || hints[0].Description != "inspect" {
		t.Errorf("Hints()[0] = {%q, %q}, want {Enter, inspect}", hints[0].Key, hints[0].Description)
	}
	if hints[1].Key != "q" || hints[1].Description != "back" {
		t.Errorf("Hints()[1] = {%q, %q}, want {q, back}", hints[1].Key, hints[1].Description)
	}
}

func TestHints_WhenNotReady_ShouldReturnOnlyBack(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	hints := p.Hints()
	if len(hints) != 1 {
		t.Fatalf("Hints() returned %d hints, want 1", len(hints))
	}
	if hints[0].Key != "q" || hints[0].Description != "back" {
		t.Errorf("Hints()[0] = {%q, %q}, want {q, back}", hints[0].Key, hints[0].Description)
	}
}

func TestHints_WhenDoneWithNoModules_ShouldReturnOnlyBack(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.modules = []ModuleImpact{}

	hints := p.Hints()
	if len(hints) != 1 {
		t.Fatalf("Hints() returned %d hints, want 1", len(hints))
	}
	if hints[0].Key != "q" || hints[0].Description != "back" {
		t.Errorf("Hints()[0] = {%q, %q}, want {q, back}", hints[0].Key, hints[0].Description)
	}
}

func TestRenderImpactBadge(t *testing.T) {
	scores := []ImpactScore{ImpactMinimal, ImpactModerate, ImpactHigh, ImpactCritical, ImpactScore(99)}
	for _, score := range scores {
		result := renderImpactBadge(score)
		if score != ImpactScore(99) && result == "" {
			t.Errorf("renderImpactBadge(%v): got empty, want non-empty", score)
		}
	}
}

func TestRenderActionBar(t *testing.T) {
	tests := []struct {
		summary sdk.ActionSummary
		wantNon bool
	}{
		{sdk.ActionSummary{Add: 1}, true},
		{sdk.ActionSummary{Change: 2}, true},
		{sdk.ActionSummary{Destroy: 1}, true},
		{sdk.ActionSummary{Replace: 1}, true},
		{sdk.ActionSummary{Add: 1, Change: 2, Destroy: 1, Replace: 1}, true},
		{sdk.ActionSummary{}, false},
	}

	for _, tt := range tests {
		result := renderActionBar(tt.summary)
		if tt.wantNon && result == "" {
			t.Errorf("renderActionBar(%v): got empty, want non-empty", tt.summary)
		}
		if !tt.wantNon && result != "" {
			t.Errorf("renderActionBar(%v): got %q, want empty", tt.summary, result)
		}
	}
}

func TestActionSymbol(t *testing.T) {
	actions := []sdk.Action{
		sdk.ActionCreate,
		sdk.ActionUpdate,
		sdk.ActionDelete,
		sdk.ActionDeleteThenCreate,
		sdk.ActionCreateThenDelete,
		sdk.ActionNoOp,
	}

	for _, action := range actions {
		result := sdk.ActionSymbol(action)
		if result == "" && action != sdk.ActionNoOp {
			t.Errorf("sdk.ActionSymbol(%q) returned empty", action)
		}
	}
}

func TestRiskBadge(t *testing.T) {
	tests := []struct {
		risk    sdk.RiskLevel
		wantNon bool
	}{
		{sdk.RiskLow, true},
		{sdk.RiskMedium, true},
		{sdk.RiskHigh, true},
		{sdk.RiskCritical, true},
		{sdk.RiskNone, false},
	}

	for _, tt := range tests {
		result := sdk.RiskBadge(tt.risk)
		if tt.wantNon && result == "" {
			t.Errorf("sdk.RiskBadge(%v): got empty, want non-empty", tt.risk)
		}
		if !tt.wantNon && result != "" {
			t.Errorf("sdk.RiskBadge(%v): got %q, want empty", tt.risk, result)
		}
	}
}

func TestRenderOverallSummary(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	tests := []struct {
		name    string
		modules []ModuleImpact
		total   int
	}{
		{
			name:    "critical",
			modules: []ModuleImpact{{Score: ImpactCritical}},
			total:   5,
		},
		{
			name:    "high",
			modules: []ModuleImpact{{Score: ImpactHigh}},
			total:   3,
		},
		{
			name:    "moderate",
			modules: []ModuleImpact{{Score: ImpactModerate}},
			total:   2,
		},
		{
			name:    "minimal",
			modules: []ModuleImpact{{Score: ImpactMinimal}},
			total:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.modules = tt.modules
			p.total = tt.total
			result := p.renderOverallSummary()
			if result == "" {
				t.Errorf("renderOverallSummary(%s): got empty", tt.name)
			}
		})
	}
}

func TestModuleCount(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.modules = []ModuleImpact{{}, {}, {}}
	if p.ModuleCount() != 3 {
		t.Errorf("ModuleCount() = %d, want 3", p.ModuleCount())
	}
}

func TestTotalChanges(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.total = 7
	if p.TotalChanges() != 7 {
		t.Errorf("TotalChanges() = %d, want 7", p.TotalChanges())
	}
}
