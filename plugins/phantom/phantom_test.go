package phantom

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
func (m *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (m *mockService) StateList(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
	return nil, nil
}
func (m *mockService) Show(_ context.Context, _ string) (string, error) { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)      { return "default", nil }
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
func (m *mockService) Version(_ context.Context) (*sdk.VersionInfo, error)          { return nil, nil }
func (m *mockService) WithDir(_ string) sdk.Service                                 { return m }

func TestNew(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	if p.ID() != "phantom" {
		t.Errorf("ID() = %q, want %q", p.ID(), "phantom")
	}
	if p.Name() != "Phantom Changes" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Phantom Changes")
	}
	if p.Description() != "Detect and explain phantom (no-op) changes" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Detect and explain phantom (no-op) changes")
	}
	if p.Ready() {
		t.Error("Ready() = true before analysis, want false")
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
	if p.phantoms != nil {
		t.Error("phantoms = non-nil, want nil")
	}
	if p.total != 0 {
		t.Errorf("total = %d, want 0", p.total)
	}
	if p.real != 0 {
		t.Errorf("real = %d, want 0", p.real)
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
	if p.phantoms != nil {
		t.Error("phantoms = non-nil after empty changes, want nil")
	}
}

func TestAnalyzeWithPhantoms(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, IsPhantom: true, AttributeDiffs: []sdk.AttributeDiff{
				{Key: "tags.Name", OldValue: "old", NewValue: "old"},
			}},
			{Resource: sdk.Resource{Address: "b"}, IsPhantom: false},
			{Resource: sdk.Resource{Address: "c"}, IsPhantom: true, AttributeDiffs: []sdk.AttributeDiff{
				{Key: "policy_json", OldValue: "{}", NewValue: "{}"},
			}},
		},
	}

	p.Analyze(summary)

	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.PhantomCount() != 2 {
		t.Errorf("PhantomCount() = %d, want 2", p.PhantomCount())
	}
	if p.RealCount() != 1 {
		t.Errorf("RealCount() = %d, want 1", p.RealCount())
	}
	if p.TotalCount() != 3 {
		t.Errorf("TotalCount() = %d, want 3", p.TotalCount())
	}
	if p.selected != 0 {
		t.Errorf("selected = %d, want 0 after Analyze", p.selected)
	}
}

func TestAnalyzeNoPhantoms(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, IsPhantom: false},
			{Resource: sdk.Resource{Address: "b"}, IsPhantom: false},
		},
	}

	p.Analyze(summary)
	if p.PhantomCount() != 0 {
		t.Errorf("PhantomCount() = %d, want 0", p.PhantomCount())
	}
	if p.RealCount() != 2 {
		t.Errorf("RealCount() = %d, want 2", p.RealCount())
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
	p.selected = 2
	if p.Selected() != 2 {
		t.Errorf("Selected() = %d, want 2", p.Selected())
	}
}

func TestUpdateKeyMsgNavigation(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.phantoms = []PhantomChange{
		{Change: sdk.PlanChange{Resource: sdk.Resource{Address: "a"}}},
		{Change: sdk.PlanChange{Resource: sdk.Resource{Address: "b"}}},
		{Change: sdk.PlanChange{Resource: sdk.Resource{Address: "c"}}},
	}

	// Move down with j
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", p.selected)
	}

	// Move down again
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
	p.phantoms = []PhantomChange{
		{Change: sdk.PlanChange{Resource: sdk.Resource{Address: "a"}}},
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
	p.phantoms = []PhantomChange{{}, {}, {}}

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

func TestMoveDownEmptyPhantoms(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.phantoms = []PhantomChange{}

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

func TestViewIdle(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusIdle) returned empty string")
	}
}

func TestViewReady_NoPhantoms(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.phantoms = []PhantomChange{}
	p.total = 5

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, no phantoms) returned empty string")
	}
}

func TestViewReady_WithPhantoms(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.total = 3
	p.real = 1
	p.phantoms = []PhantomChange{
		{
			Change:      sdk.PlanChange{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}},
			Explanation: "JSON field reordering",
			Attributes: []sdk.AttributeDiff{
				{Key: "policy", OldValue: "{}", NewValue: "{}"},
				{Key: "secret", OldValue: "x", NewValue: "y", Sensitive: true},
			},
		},
		{
			Change:      sdk.PlanChange{Resource: sdk.Resource{Address: "aws_iam_role.admin"}},
			Explanation: "tag ordering",
			Attributes:  []sdk.AttributeDiff{},
		},
	}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, with phantoms) returned empty string")
	}
}

func TestViewReady_WithExpandedPhantom(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.total = 2
	p.real = 1
	p.phantoms = []PhantomChange{
		{
			Change:      sdk.PlanChange{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}},
			Explanation: "JSON field reordering",
			Attributes: []sdk.AttributeDiff{
				{Key: "policy", OldValue: "{\"a\":1}", NewValue: "{\"a\":1}"},
				{Key: "secret", OldValue: "x", NewValue: "y", Sensitive: true},
			},
		},
	}
	p.expanded = map[int]bool{0: true}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with expanded phantom returned empty string")
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
	p.phantoms = []PhantomChange{
		{Change: sdk.PlanChange{Resource: sdk.Resource{Address: "a"}}, Explanation: "test"},
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

	phantoms := make([]PhantomChange, 20)
	for i := range phantoms {
		phantoms[i] = PhantomChange{
			Change:      sdk.PlanChange{Resource: sdk.Resource{Address: "res_" + string(rune('a'+i%26))}},
			Explanation: "test",
		}
	}
	p.phantoms = phantoms
	p.selected = 15

	view := p.View(80, 10)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestExplainPhantom(t *testing.T) {
	tests := []struct {
		name   string
		change sdk.PlanChange
		want   string
	}{
		{
			name:   "empty diffs",
			change: sdk.PlanChange{AttributeDiffs: []sdk.AttributeDiff{}},
			want:   "empty diff detected",
		},
		{
			name: "json field",
			change: sdk.PlanChange{AttributeDiffs: []sdk.AttributeDiff{
				{Key: "policy_json", OldValue: "{}", NewValue: "{}"},
			}},
			want: "JSON/policy field reordering or whitespace difference",
		},
		{
			name: "policy field",
			change: sdk.PlanChange{AttributeDiffs: []sdk.AttributeDiff{
				{Key: "assume_role_policy", OldValue: "{}", NewValue: "{}"},
			}},
			want: "JSON/policy field reordering or whitespace difference",
		},
		{
			name: "tags ordering",
			change: sdk.PlanChange{AttributeDiffs: []sdk.AttributeDiff{
				{Key: "tags.Name", OldValue: "a", NewValue: "a"},
			}},
			want: "tag/label ordering difference (cosmetic)",
		},
		{
			name: "labels ordering",
			change: sdk.PlanChange{AttributeDiffs: []sdk.AttributeDiff{
				{Key: "labels.env", OldValue: "prod", NewValue: "prod"},
			}},
			want: "tag/label ordering difference (cosmetic)",
		},
		{
			name: "generic diff",
			change: sdk.PlanChange{AttributeDiffs: []sdk.AttributeDiff{
				{Key: "name", OldValue: "a", NewValue: "b"},
			}},
			want: "semantically equivalent values with different serialization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := explainPhantom(tt.change)
			if result != tt.want {
				t.Errorf("explainPhantom() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestHints_WhenStatusDoneWithPhantoms_ShouldIncludeInspectAndBack(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.phantoms = []PhantomChange{
		{Change: sdk.PlanChange{Resource: sdk.Resource{Address: "a"}}},
	}

	hints := p.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints() len = %d, want 2", len(hints))
	}
	if hints[0].Key != "Enter" || hints[0].Description != "inspect" {
		t.Errorf("Hints()[0] = {%q, %q}, want {Enter, inspect}", hints[0].Key, hints[0].Description)
	}
	if hints[1].Key != "q" || hints[1].Description != "back" {
		t.Errorf("Hints()[1] = {%q, %q}, want {q, back}", hints[1].Key, hints[1].Description)
	}
}

func TestHints_WhenStatusDoneNoPhantoms_ShouldReturnOnlyBack(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.phantoms = []PhantomChange{}

	hints := p.Hints()
	if len(hints) != 1 {
		t.Fatalf("Hints() len = %d, want 1", len(hints))
	}
	if hints[0].Key != "q" || hints[0].Description != "back" {
		t.Errorf("Hints()[0] = {%q, %q}, want {q, back}", hints[0].Key, hints[0].Description)
	}
}

func TestHints_WhenStatusIdle_ShouldReturnOnlyBack(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusIdle

	hints := p.Hints()
	if len(hints) != 1 {
		t.Fatalf("Hints() len = %d, want 1", len(hints))
	}
	if hints[0].Key != "q" || hints[0].Description != "back" {
		t.Errorf("Hints()[0] = {%q, %q}, want {q, back}", hints[0].Key, hints[0].Description)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 20, "short"},
		{"a long string that exceeds", 15, "a long strin..."},
		{"tiny", 3, "tiny"}, // max < 10 gets set to 10
	}

	for _, tt := range tests {
		got := sdk.Truncate(tt.input, tt.max)
		if tt.max < 10 {
			if len(tt.input) <= 10 && got != tt.input {
				t.Errorf("sdk.Truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.input)
			}
		} else if len(tt.input) > tt.max {
			if len(got) != tt.max {
				t.Errorf("sdk.Truncate(%q, %d): len = %d, want %d", tt.input, tt.max, len(got), tt.max)
			}
		} else {
			if got != tt.input {
				t.Errorf("sdk.Truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.input)
			}
		}
	}
}
