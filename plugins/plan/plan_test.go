package plan

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// mockService implements sdk.Service for testing.
type mockService struct {
	planResult *sdk.PlanSummary
	planErr    error
}

func (m *mockService) Plan(_ context.Context, _ []string) (*sdk.PlanSummary, error) {
	return m.planResult, m.planErr
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

	if p.ID() != "plan" {
		t.Errorf("ID() = %q, want %q", p.ID(), "plan")
	}
	if p.Name() != "Plan" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Plan")
	}
	if p.Description() != "Review terraform plan changes" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Review terraform plan changes")
	}
	if p.KeyBinding() != "p" {
		t.Errorf("KeyBinding() = %q, want %q", p.KeyBinding(), "p")
	}
	if p.Ready() {
		t.Error("Ready() = true before data loads, want false")
	}
}

func TestCountable(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	var c sdk.Countable = p
	filtered, total := c.Count()
	if filtered != 0 || total != 0 {
		t.Errorf("Count() = (%d, %d), want (0, 0) when no summary", filtered, total)
	}

	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}}, {Resource: sdk.Resource{Address: "b"}}},
	}
	filtered, total = c.Count()
	if filtered != 2 || total != 2 {
		t.Errorf("Count() = (%d, %d), want (2, 2)", filtered, total)
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
		planResult: &sdk.PlanSummary{
			Changes:  []sdk.PlanChange{},
			ToCreate: 0,
		},
	}
	p := New(svc)
	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Workspace:  "default",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Session:    sdk.NewSession(),
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() should return nil cmd (no auto-plan)")
	}

	pp := p.(*Plugin)
	if pp.status != StatusIdle {
		t.Errorf("status = %v, want StatusIdle", pp.status)
	}
}

func TestActivate(t *testing.T) {
	svc := &mockService{
		planResult: &sdk.PlanSummary{Changes: []sdk.PlanChange{}},
	}
	p := New(svc)
	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Session:    sdk.NewSession(),
	}
	p.Init(ctx)

	pp := p.(*Plugin)
	cmd := pp.Activate()
	if cmd == nil {
		t.Error("Activate() returned nil cmd, want non-nil")
	}
	if pp.status != StatusLoading {
		t.Errorf("status = %v, want StatusLoading", pp.status)
	}
}

func TestActivateCmdReturnsPlanResultMsg(t *testing.T) {
	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource: sdk.Resource{Address: "aws_instance.web", Type: "aws_instance"},
				Action:   sdk.ActionCreate,
				Risk:     sdk.RiskLow,
			},
		},
		ToCreate: 1,
	}
	svc := &mockService{planResult: summary}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Session: sdk.NewSession()}
	p.Init(ctx)

	cmd := p.(*Plugin).Activate()
	msg := cmd()

	result, ok := msg.(PlanResultMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T, want PlanResultMsg", msg)
	}
	if result.Err != nil {
		t.Errorf("PlanResultMsg.Err = %v, want nil", result.Err)
	}
	if result.Summary == nil {
		t.Fatal("PlanResultMsg.Summary = nil, want non-nil")
	}
	if len(result.Summary.Changes) != 1 {
		t.Errorf("len(Summary.Changes) = %d, want 1", len(result.Summary.Changes))
	}
}

func TestActivateCmdReturnsError(t *testing.T) {
	svc := &mockService{planErr: errors.New("plan failed")}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Session: sdk.NewSession()}
	p.Init(ctx)

	cmd := p.(*Plugin).Activate()
	msg := cmd()

	result, ok := msg.(PlanResultMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T, want PlanResultMsg", msg)
	}
	if result.Err == nil {
		t.Error("PlanResultMsg.Err = nil, want error")
	}
}

func TestUpdatePlanResultSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_s3_bucket.test"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	result, cmd := p.Update(PlanResultMsg{Summary: summary, Err: nil})
	if cmd != nil {
		t.Errorf("Update(PlanResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusDone {
		t.Errorf("status = %v, want StatusDone", updated.status)
	}
	if updated.summary == nil {
		t.Error("summary = nil, want non-nil")
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdatePlanResultError(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	result, cmd := p.Update(PlanResultMsg{Summary: nil, Err: errors.New("terraform error")})
	if cmd != nil {
		t.Errorf("Update(PlanResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusError {
		t.Errorf("status = %v, want StatusError", updated.status)
	}
	if updated.errMsg != "terraform error" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "terraform error")
	}
}

func TestUpdateKeyMsgNavigation(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusDone
	pp.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
			{Resource: sdk.Resource{Address: "c"}},
		},
	}

	// Move down
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if pp.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", pp.selected)
	}

	// Move down again
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if pp.selected != 2 {
		t.Errorf("after j,j: selected = %d, want 2", pp.selected)
	}

	// Move down at boundary (should not go past last)
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if pp.selected != 2 {
		t.Errorf("after j,j,j: selected = %d, want 2 (boundary)", pp.selected)
	}

	// Move up
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if pp.selected != 1 {
		t.Errorf("after k: selected = %d, want 1", pp.selected)
	}

	// Move up to start
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if pp.selected != 0 {
		t.Errorf("after k,k: selected = %d, want 0", pp.selected)
	}

	// Move up at boundary
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if pp.selected != 0 {
		t.Errorf("after k,k,k: selected = %d, want 0 (boundary)", pp.selected)
	}
}

func TestUpdateKeyMsgMoveToEndAndStart(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusDone
	pp.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
			{Resource: sdk.Resource{Address: "c"}},
		},
	}

	// G moves to end
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if pp.selected != 2 {
		t.Errorf("after G: selected = %d, want 2", pp.selected)
	}

	// g moves to start
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if pp.selected != 0 {
		t.Errorf("after g: selected = %d, want 0", pp.selected)
	}
}

func TestUpdateKeyMsgToggleExpand(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusDone
	pp.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}

	// Toggle expand with enter
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !pp.expanded[0] {
		t.Error("after enter: expanded[0] = false, want true")
	}

	// Toggle again
	pp.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if pp.expanded[0] {
		t.Error("after enter,enter: expanded[0] = true, want false")
	}
}

func TestUpdateKeyMsgRefresh(t *testing.T) {
	svc := &mockService{planResult: &sdk.PlanSummary{}}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusDone

	// r triggers refresh when status is Done
	cmd := pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("after r in StatusDone: cmd = nil, want non-nil (refresh)")
	}

	// r triggers refresh when status is Error
	pp.status = StatusError
	cmd = pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("after r in StatusError: cmd = nil, want non-nil (refresh)")
	}

	// r does nothing when Loading
	pp.status = StatusLoading
	cmd = pp.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil {
		t.Error("after r in StatusLoading: cmd != nil, want nil")
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
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}

	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown: selected = %d, want 1", p.selected)
	}
	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown (boundary): selected = %d, want 1", p.selected)
	}
	p.MoveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp: selected = %d, want 0", p.selected)
	}
	p.MoveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp (boundary): selected = %d, want 0", p.selected)
	}
}

func TestMoveDownNilSummary(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.summary = nil
	p.MoveDown()
	if p.selected != 0 {
		t.Errorf("MoveDown nil summary: selected = %d, want 0", p.selected)
	}
}

func TestMoveToStartEnd(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
			{Resource: sdk.Resource{Address: "c"}},
		},
	}

	p.MoveToEnd()
	if p.selected != 2 {
		t.Errorf("MoveToEnd: selected = %d, want 2", p.selected)
	}
	p.MoveToStart()
	if p.selected != 0 {
		t.Errorf("MoveToStart: selected = %d, want 0", p.selected)
	}
}

func TestMoveToEndNilSummary(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.summary = nil
	p.MoveToEnd()
	if p.selected != 0 {
		t.Errorf("MoveToEnd nil summary: selected = %d, want 0", p.selected)
	}
}

func TestMoveToEndEmptyChanges(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}
	p.MoveToEnd()
	if p.selected != 0 {
		t.Errorf("MoveToEnd empty: selected = %d, want 0", p.selected)
	}
}

func TestToggleExpand(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.selected = 2

	p.ToggleExpand()
	if !p.IsExpanded(2) {
		t.Error("ToggleExpand: IsExpanded(2) = false, want true")
	}
	p.ToggleExpand()
	if p.IsExpanded(2) {
		t.Error("ToggleExpand: IsExpanded(2) = true, want false")
	}
}

func TestIsExpanded(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	if p.IsExpanded(0) {
		t.Error("IsExpanded(0) = true before toggle, want false")
	}
}

func TestSelectedChange(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	// nil summary
	if p.SelectedChange() != nil {
		t.Error("SelectedChange nil summary: want nil")
	}

	// valid selection
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.selected = 1
	sc := p.SelectedChange()
	if sc == nil {
		t.Fatal("SelectedChange: got nil")
	}
	if sc.Resource.Address != "b" {
		t.Errorf("SelectedChange.Resource.Address = %q, want %q", sc.Resource.Address, "b")
	}

	// out of bounds
	p.selected = 10
	if p.SelectedChange() != nil {
		t.Error("SelectedChange out of bounds: want nil")
	}
}

func TestSetTargets(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	targets := []string{"aws_instance.web", "aws_s3_bucket.data"}
	p.SetTargets(targets)
	if len(p.targets) != 2 {
		t.Errorf("SetTargets: len(targets) = %d, want 2", len(p.targets))
	}
}

func TestRefresh(t *testing.T) {
	svc := &mockService{planResult: &sdk.PlanSummary{}}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.selected = 5
	p.expanded[0] = true

	cmd := p.Refresh()
	if cmd == nil {
		t.Error("Refresh() returned nil cmd")
	}
	if p.status != StatusLoading {
		t.Errorf("after Refresh: status = %v, want StatusLoading", p.status)
	}
	if p.selected != 0 {
		t.Errorf("after Refresh: selected = %d, want 0", p.selected)
	}
}

func TestViewIdle(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusIdle) returned empty string")
	}
}

func TestViewLoading(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusLoading

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusLoading) returned empty string")
	}
}

func TestViewError(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusError
	p.errMsg = "some error"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusError) returned empty string")
	}
}

func TestViewDoneNoChanges(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, no changes) returned empty string")
	}
}

func TestViewDoneNilSummary(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.summary = nil

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, nil summary) returned empty string")
	}
}

func TestViewDoneWithChanges(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "aws_instance.web", Type: "aws_instance"},
				Action:         sdk.ActionCreate,
				Risk:           sdk.RiskLow,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "ami", OldValue: "", NewValue: "ami-123"}},
			},
			{
				Resource:  sdk.Resource{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket"},
				Action:    sdk.ActionDelete,
				Risk:      sdk.RiskCritical,
				IsPhantom: true,
			},
			{
				Resource: sdk.Resource{Address: "aws_vpc.main", Type: "aws_vpc"},
				Action:   sdk.ActionUpdate,
				Risk:     sdk.RiskMedium,
			},
			{
				Resource: sdk.Resource{Address: "aws_subnet.a", Type: "aws_subnet"},
				Action:   sdk.ActionDeleteThenCreate,
				Risk:     sdk.RiskHigh,
			},
			{
				Resource: sdk.Resource{Address: "aws_subnet.b", Type: "aws_subnet"},
				Action:   sdk.ActionCreateThenDelete,
				Risk:     sdk.RiskHigh,
			},
			{
				Resource: sdk.Resource{Address: "data.aws_ami.latest", Type: "aws_ami"},
				Action:   sdk.ActionRead,
				Risk:     sdk.RiskNone,
			},
		},
		ToCreate:  1,
		ToUpdate:  1,
		ToDelete:  1,
		ToReplace: 2,
	}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, with changes) returned empty string")
	}
}

func TestViewDoneWithExpandedAttributeDiffs(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource: sdk.Resource{Address: "aws_instance.web"},
				Action:   sdk.ActionUpdate,
				Risk:     sdk.RiskMedium,
				AttributeDiffs: []sdk.AttributeDiff{
					{Key: "ami", OldValue: "ami-old", NewValue: "ami-new"},
					{Key: "password", OldValue: "secret", NewValue: "newsecret", Sensitive: true},
				},
			},
		},
		ToUpdate: 1,
	}
	p.expanded[0] = true

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with expanded diffs returned empty string")
	}
}

func TestViewDoneScrolling(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone

	// Create many changes to trigger scrolling
	changes := make([]sdk.PlanChange, 50)
	for i := range changes {
		changes[i] = sdk.PlanChange{
			Resource: sdk.Resource{Address: "aws_instance.web_" + string(rune('a'+i%26))},
			Action:   sdk.ActionCreate,
			Risk:     sdk.RiskLow,
		}
	}
	p.summary = &sdk.PlanSummary{Changes: changes, ToCreate: 50}
	p.selected = 45

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestViewDefaultStatus(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = Status(99) // invalid status

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestRenderSummaryLineAllZero(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.summary = &sdk.PlanSummary{}

	result := p.renderSummaryLine()
	if result == "" {
		t.Error("renderSummaryLine with all zero returned empty string")
	}
}

func TestRenderOverallRisk(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	// nil summary
	p.summary = nil
	if p.renderOverallRisk() != "" {
		t.Error("renderOverallRisk nil summary: want empty")
	}

	// empty changes
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}
	if p.renderOverallRisk() != "" {
		t.Error("renderOverallRisk empty changes: want empty")
	}

	// various risk levels
	tests := []struct {
		risk    sdk.RiskLevel
		wantNon bool
	}{
		{sdk.RiskCritical, true},
		{sdk.RiskHigh, true},
		{sdk.RiskMedium, true},
		{sdk.RiskLow, true},
		{sdk.RiskNone, false},
	}

	for _, tt := range tests {
		p.summary = &sdk.PlanSummary{
			Changes: []sdk.PlanChange{
				{Risk: tt.risk},
			},
		}
		result := p.renderOverallRisk()
		if tt.wantNon && result == "" {
			t.Errorf("renderOverallRisk(risk=%v): got empty, want non-empty", tt.risk)
		}
		if !tt.wantNon && result != "" {
			t.Errorf("renderOverallRisk(risk=%v): got %q, want empty", tt.risk, result)
		}
	}
}

func TestTruncateValue(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 20, "short"},
		{"a long string that should be truncated", 15, "a long strin..."},
		{"test", 5, "te..."},
		{"hello", 10, "hello"},
		{"tiny", 3, "t..."}, // maxLen < 10 gets raised to 10
	}

	for _, tt := range tests {
		got := sdk.Truncate(tt.input, tt.maxLen)
		if tt.maxLen < 10 {
			// maxLen gets raised to 10
			if len(tt.input) <= 10 {
				if got != tt.input {
					t.Errorf("sdk.Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.input)
				}
			}
		} else if len(tt.input) > tt.maxLen {
			if len(got) != tt.maxLen {
				t.Errorf("sdk.Truncate(%q, %d): len = %d, want %d", tt.input, tt.maxLen, len(got), tt.maxLen)
			}
		} else {
			if got != tt.input {
				t.Errorf("sdk.Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.input)
			}
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
		sdk.ActionRead,
		sdk.ActionNoOp,
	}

	for _, action := range actions {
		result := sdk.ActionSymbol(action)
		if result == "" {
			// Only NoOp should return a space
			if action != sdk.ActionNoOp {
				t.Errorf("sdk.ActionSymbol(%q) returned empty, want non-empty", action)
			}
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

func TestStatus(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	if p.Status() != StatusIdle {
		t.Errorf("Status() = %v, want StatusIdle", p.Status())
	}
}

func TestSelected(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.selected = 5

	if p.Selected() != 5 {
		t.Errorf("Selected() = %d, want 5", p.Selected())
	}
}

func TestTargets(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.targets = []string{"a", "b"}

	targets := p.Targets()
	if len(targets) != 2 {
		t.Errorf("Targets() len = %d, want 2", len(targets))
	}
}

func TestSummary(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	if p.Summary() != nil {
		t.Error("Summary() = non-nil before plan, want nil")
	}

	p.summary = &sdk.PlanSummary{ToCreate: 3}
	if p.Summary().ToCreate != 3 {
		t.Errorf("Summary().ToCreate = %d, want 3", p.Summary().ToCreate)
	}
}

func TestViewSmallHeight(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	// Very small height
	view := p.View(80, 5)
	if view == "" {
		t.Error("View with small height returned empty string")
	}
}
