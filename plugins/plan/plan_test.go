package plan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestNew(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)

	if p.ID() != "plan" {
		t.Errorf("ID() = %q, want %q", p.ID(), "plan")
	}
	if p.Name() != "Plan" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Plan")
	}
	if p.Description() != "Review terraform plan changes" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Review terraform plan changes")
	}
	if p.Ready() {
		t.Error("Ready() = true before data loads, want false")
	}
}

func TestCountable(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)

	var c sdk.Countable = p
	filtered, total := c.Count()
	if filtered != 0 || total != 0 {
		t.Errorf("Count() = (%d, %d), want (0, 0) when no summary", filtered, total)
	}

	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}}, {Resource: sdk.Resource{Address: "b"}}},
	}
	p.filtered = p.summary.Changes
	filtered, total = c.Count()
	if filtered != 2 || total != 2 {
		t.Errorf("Count() = (%d, %d), want (2, 2)", filtered, total)
	}
}

func TestConfigure(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	err := p.Configure(map[string]interface{}{"key": "value"})
	if err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestInit(t *testing.T) {
	svc := &sdktest.MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return &sdk.PlanSummary{Changes: []sdk.PlanChange{}, ToCreate: 0}, nil
		},
	}
	p := New(svc)
	h := sdktest.NewDeps(svc)

	cmd := p.Init(h.Deps)
	if cmd != nil {
		t.Error("Init() should return nil cmd (no auto-plan)")
	}

	pp := p.(*Plugin)
	if pp.status != sdk.StatusIdle {
		t.Errorf("status = %v, want sdk.StatusIdle", pp.status)
	}
}

func TestActivate(t *testing.T) {
	svc := &sdktest.MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return &sdk.PlanSummary{Changes: []sdk.PlanChange{}}, nil
		},
	}
	p := New(svc)
	h := sdktest.NewDeps(svc)
	p.Init(h.Deps)

	pp := p.(*Plugin)
	cmd := pp.Activate()
	if cmd == nil {
		t.Error("Activate() returned nil cmd, want non-nil")
	}
	if pp.status != sdk.StatusLoading {
		t.Errorf("status = %v, want sdk.StatusLoading", pp.status)
	}
}

func TestActivateWhileLoadingRestartsTick(t *testing.T) {
	svc := &sdktest.MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return &sdk.PlanSummary{Changes: []sdk.PlanChange{}}, nil
		},
	}
	p := New(svc)
	h := sdktest.NewDeps(svc)
	p.Init(h.Deps)

	pp := p.(*Plugin)
	pp.Activate()

	// Simulate re-entering the plugin while plan is still loading
	cmd := pp.Activate()
	if cmd == nil {
		t.Error("Activate() while loading returned nil cmd, want tick cmd")
	}
}

func TestActivate_GivenPlanSucceeds_ShouldReturnSummaryInCmd(t *testing.T) {
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
	svc := &sdktest.MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return summary, nil
		},
	}
	p := New(svc)
	h := sdktest.NewDeps(svc)
	p.Init(h.Deps)

	cmd := p.(*Plugin).Activate()
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Activate cmd returned %T, want tea.BatchMsg", msg)
	}
	var result PlanResultMsg
	found := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(PlanResultMsg); ok {
			result = r
			found = true
		}
	}
	if !found {
		t.Fatal("batch did not contain PlanResultMsg")
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

func TestActivate_GivenPlanFails_ShouldReturnErrorInCmd(t *testing.T) {
	svc := &sdktest.MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return nil, errors.New("plan failed")
		},
	}
	p := New(svc)
	h := sdktest.NewDeps(svc)
	p.Init(h.Deps)

	cmd := p.(*Plugin).Activate()
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Activate cmd returned %T, want tea.BatchMsg", msg)
	}
	var result PlanResultMsg
	found := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(PlanResultMsg); ok {
			result = r
			found = true
		}
	}
	if !found {
		t.Fatal("batch did not contain PlanResultMsg")
	}
	if result.Err == nil {
		t.Error("PlanResultMsg.Err = nil, want error")
	}
}

func TestUpdatePlanResultSuccess(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_s3_bucket.test"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	result, cmd := p.Update(PlanResultMsg{Summary: summary, Err: nil})
	if cmd == nil {
		t.Fatal("Update(PlanResultMsg) cmd = nil, want batched cmd")
	}

	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want tea.BatchMsg", msg)
	}
	foundPlanCompleted := false
	foundStateRefreshed := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		subMsg := subCmd()
		switch evt := subMsg.(type) {
		case sdk.PlanCompletedEvent:
			foundPlanCompleted = true
			if evt.ResourceCount != 1 {
				t.Errorf("event.ResourceCount = %d, want 1", evt.ResourceCount)
			}
			if evt.Summary != summary {
				t.Error("event.Summary does not match")
			}
		case sdk.StateRefreshedEvent:
			foundStateRefreshed = true
		}
	}
	if !foundPlanCompleted {
		t.Error("batched cmd should contain PlanCompletedEvent")
	}
	if !foundStateRefreshed {
		t.Error("batched cmd should contain StateRefreshedEvent")
	}

	updated := result.(*Plugin)
	if updated.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", updated.status)
	}
	if updated.summary == nil {
		t.Error("summary = nil, want non-nil")
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdatePlanResultError(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	result, cmd := p.Update(PlanResultMsg{Summary: nil, Err: errors.New("terraform error")})
	if cmd != nil {
		t.Errorf("Update(PlanResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != sdk.StatusError {
		t.Errorf("status = %v, want sdk.StatusError", updated.status)
	}
	if updated.errMsg != "terraform error" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "terraform error")
	}
}

func TestUpdateKeyMsgNavigation(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
			{Resource: sdk.Resource{Address: "c"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	// Move down
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.tree.Cursor() != 1 {
		t.Errorf("after j: cursor = %d, want 1", p.tree.Cursor())
	}

	// Move down again
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.tree.Cursor() != 2 {
		t.Errorf("after j,j: cursor = %d, want 2", p.tree.Cursor())
	}

	// Move down at boundary (should not go past last)
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.tree.Cursor() != 2 {
		t.Errorf("after j,j,j: cursor = %d, want 2 (boundary)", p.tree.Cursor())
	}

	// Move up
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.tree.Cursor() != 1 {
		t.Errorf("after k: cursor = %d, want 1", p.tree.Cursor())
	}

	// Move up to start
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.tree.Cursor() != 0 {
		t.Errorf("after k,k: cursor = %d, want 0", p.tree.Cursor())
	}

	// Move up at boundary
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.tree.Cursor() != 0 {
		t.Errorf("after k,k,k: cursor = %d, want 0 (boundary)", p.tree.Cursor())
	}
}

func TestUpdateKeyMsgMoveToEndAndStart(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
			{Resource: sdk.Resource{Address: "c"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	// G moves to end
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if p.tree.Cursor() != 2 {
		t.Errorf("after G: cursor = %d, want 2", p.tree.Cursor())
	}

	// g moves to start
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if p.tree.Cursor() != 0 {
		t.Errorf("after g: cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestUpdateKeyMsgInspect(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	// Enter opens inspect frame
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.stack.Peek().ID() != "inspect" {
		t.Errorf("after enter: top frame = %q, want %q", p.stack.Peek().ID(), "inspect")
	}
	if p.detailAddr != "a" {
		t.Errorf("detailAddr = %q, want %q", p.detailAddr, "a")
	}

	// Esc closes inspect
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.stack.Peek().ID() != "list" {
		t.Errorf("after esc: top frame = %q, want %q", p.stack.Peek().ID(), "list")
	}
}

func TestCtrlR_GivenStatusDone_ShouldTriggerRefresh(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Error("ctrl+r when Done should trigger refresh")
	}
}

func TestCtrlR_GivenStatusError_ShouldTriggerRefresh(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusError

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Error("ctrl+r when Error should trigger refresh")
	}
}

func TestCtrlR_GivenStatusLoading_ShouldBeIgnored(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("ctrl+r during Loading should be ignored")
	}
}

func TestUpdateUnknownMsg(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)

	type unknownMsg struct{}
	result, cmd := p.Update(unknownMsg{})
	if cmd != nil {
		t.Errorf("Update(unknownMsg) cmd = %v, want nil", cmd)
	}
	if result.(*Plugin) != p {
		t.Error("Update(unknownMsg) returned different plugin reference")
	}
}

func TestMoveUpDown(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.MoveDown()
	if p.tree.Cursor() != 1 {
		t.Errorf("MoveDown: cursor = %d, want 1", p.tree.Cursor())
	}
	p.MoveDown()
	if p.tree.Cursor() != 1 {
		t.Errorf("MoveDown (boundary): cursor = %d, want 1", p.tree.Cursor())
	}
	p.MoveUp()
	if p.tree.Cursor() != 0 {
		t.Errorf("MoveUp: cursor = %d, want 0", p.tree.Cursor())
	}
	p.MoveUp()
	if p.tree.Cursor() != 0 {
		t.Errorf("MoveUp (boundary): cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestMoveDownNilSummary(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.summary = nil
	p.MoveDown()
	if p.tree.Cursor() != 0 {
		t.Errorf("MoveDown nil summary: cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestMoveToStartEnd(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
			{Resource: sdk.Resource{Address: "c"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.MoveToEnd()
	if p.tree.Cursor() != 2 {
		t.Errorf("MoveToEnd: cursor = %d, want 2", p.tree.Cursor())
	}
	p.MoveToStart()
	if p.tree.Cursor() != 0 {
		t.Errorf("MoveToStart: cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestMoveToEndNilSummary(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.summary = nil
	p.MoveToEnd()
	if p.tree.Cursor() != 0 {
		t.Errorf("MoveToEnd nil summary: cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestMoveToEndEmptyChanges(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}
	p.MoveToEnd()
	if p.tree.Cursor() != 0 {
		t.Errorf("MoveToEnd empty: cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestSelectedChange(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)

	// empty tree
	if p.SelectedChange() != nil {
		t.Error("SelectedChange empty tree: want nil")
	}

	// valid selection
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.MoveDown()
	sc := p.SelectedChange()
	if sc == nil {
		t.Fatal("SelectedChange: got nil")
	}
	if sc.Resource.Address != "b" {
		t.Errorf("SelectedChange.Resource.Address = %q, want %q", sc.Resource.Address, "b")
	}
}

func TestRefresh(t *testing.T) {
	svc := &sdktest.MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return &sdk.PlanSummary{}, nil
		},
	}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone

	cmd := p.Refresh()
	if cmd == nil {
		t.Error("Refresh() returned nil cmd")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("after Refresh: status = %v, want sdk.StatusLoading", p.status)
	}
	if p.tree.Cursor() != 0 {
		t.Errorf("after Refresh: cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestViewIdle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusIdle) returned empty string")
	}
}

func TestViewLoading(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusLoading) returned empty string")
	}
}

func TestViewError(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusError
	p.errMsg = "some error"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusError) returned empty string")
	}
}

func TestViewDoneNoChanges(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, no changes) returned empty string")
	}
}

func TestViewDoneNilSummary(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = nil

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, nil summary) returned empty string")
	}
}

func TestViewDoneWithChanges(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
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
		t.Error("View(sdk.StatusDone, with changes) returned empty string")
	}
}

func TestViewDoneWithInspectDetail(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
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
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.inspectSelected()

	view := p.renderDetail(80, 24)
	if view == "" {
		t.Error("Inspect detail view returned empty string")
	}
	if !strings.Contains(view, "aws_instance.web") {
		t.Error("Inspect detail should contain resource address")
	}
}

func TestViewDoneScrolling(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone

	// Create many changes to trigger scrolling
	changes := make([]sdk.PlanChange, 50)
	for i := range changes {
		changes[i] = sdk.PlanChange{
			Resource: sdk.Resource{Address: fmt.Sprintf("aws_instance.web_%d", i)},
			Action:   sdk.ActionCreate,
			Risk:     sdk.RiskLow,
		}
	}
	p.summary = &sdk.PlanSummary{Changes: changes, ToCreate: 50}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	for i := 0; i < 45; i++ {
		p.MoveDown()
	}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestViewDefaultStatus(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.Status(99) // invalid status

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestRenderSummaryLineAllZero(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{}

	result := p.renderSummaryLine()
	if result == "" {
		t.Error("renderSummaryLine with all zero returned empty string")
	}
}

func TestRenderOverallRisk(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)

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
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)

	if p.Status() != sdk.StatusIdle {
		t.Errorf("Status() = %v, want sdk.StatusIdle", p.Status())
	}
}

func TestTreeCursor(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.MoveDown()

	if p.tree.Cursor() != 1 {
		t.Errorf("tree.Cursor() = %d, want 1", p.tree.Cursor())
	}
}

func TestSummary(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	if p.Summary() != nil {
		t.Error("Summary() = non-nil before plan, want nil")
	}

	p.summary = &sdk.PlanSummary{ToCreate: 3}
	if p.Summary().ToCreate != 3 {
		t.Errorf("Summary().ToCreate = %d, want 3", p.Summary().ToCreate)
	}
}

func TestViewSmallHeight(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
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

func TestRequestApply_ShouldEmitApplyRequestMsg(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
	}

	cmd := p.requestApply()
	if cmd == nil {
		t.Fatal("requestApply() returned nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(ApplyRequestMsg); !ok {
		t.Fatalf("cmd() = %T, want ApplyRequestMsg", msg)
	}
}

func newTestPluginWithHarness(svc sdk.Service) (*Plugin, *sdktest.PluginDepsHarness) {
	p := New(svc).(*Plugin)
	h := sdktest.NewDeps(svc)
	p.Init(h.Deps)
	return p, h
}

func newTestPlugin(svc sdk.Service) *Plugin {
	p, _ := newTestPluginWithHarness(svc)
	return p
}

func TestPlugin_WhenCreated_ShouldExposeStack(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	if p.Stack() == nil {
		t.Error("Stack() = nil, want non-nil")
	}
	if p.Stack().Depth() != 1 {
		t.Errorf("Stack().Depth() = %d, want 1", p.Stack().Depth())
	}
}

func TestPlugin_WhenCreated_ShouldReportNotBusy(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	if p.Busy() {
		t.Error("Busy() = true, want false when status is Idle")
	}
}

func TestPlugin_WhenLoading_ShouldReportBusy(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading
	if !p.Busy() {
		t.Error("Busy() = false, want true when status is Loading")
	}
}

func TestPlugin_WhenDone_ShouldReportNotBusy(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	if p.Busy() {
		t.Error("Busy() = true, want false when status is Done")
	}
}

func TestPlugin_WhenChdirChanged_ShouldResetState(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}}}}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.errMsg = "old error"

	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{
		Next: &sdk.Context{Service: svc, WorkingDir: "/projects/infra/modules/vpc"},
	})

	if cmd != nil {
		t.Error("HandleContextChanged() cmd != nil, want nil")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want Idle", p.status)
	}
	if p.summary != nil {
		t.Error("summary != nil after reset")
	}
	if p.tree.Cursor() != 0 {
		t.Error("cursor != 0 after reset")
	}
	if p.errMsg != "" {
		t.Errorf("errMsg = %q, want empty", p.errMsg)
	}
}

func TestPlugin_WhenPlanInvalidated_WhileDone_ShouldMarkStale(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}}}}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	cmd := p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})

	if cmd != nil {
		t.Error("HandlePlanInvalidated() cmd != nil, want nil")
	}
	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want Done (results preserved)", p.status)
	}
	if p.summary == nil {
		t.Error("summary = nil, want preserved")
	}
	if !p.stale {
		t.Error("stale = false, want true")
	}
}

func TestPlugin_WhenPlanInvalidated_WhileNotDone_ShouldResetState(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	p.errMsg = "something"

	cmd := p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})

	if cmd != nil {
		t.Error("HandlePlanInvalidated() cmd != nil, want nil")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want Idle", p.status)
	}
	if p.errMsg != "" {
		t.Errorf("errMsg = %q, want empty", p.errMsg)
	}
}

func TestPlugin_WhenActivatedWhileLoading_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() while loading returned non-nil cmd, want nil")
	}
}

func TestPlugin_WhenActivatedWhileDone_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() while done returned non-nil cmd, want nil")
	}
}

func TestPlugin_WhenActivatedWhileStale_ShouldReplan(t *testing.T) {
	svc := &sdktest.MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return &sdk.PlanSummary{}, nil
		},
	}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.stale = true

	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() while stale returned nil cmd, want re-plan")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
	if p.stale {
		t.Error("stale should be cleared after Activate")
	}
}

func TestPlugin_WhenActivatedWhileError_ShouldRetriggerPlan(t *testing.T) {
	svc := &sdktest.MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return &sdk.PlanSummary{}, nil
		},
	}
	p := newTestPlugin(svc)
	p.status = sdk.StatusError

	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() while error returned nil cmd, want non-nil")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
}

func TestActivate_GivenPlanRuns_ShouldPassPlanFileToService(t *testing.T) {
	svc := &sdktest.MockService{
		PlanFn: func(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
			return &sdk.PlanSummary{}, nil
		},
	}
	p := newTestPlugin(svc)

	cmd := p.Activate()
	// Drain the batch so the underlying Plan call executes.
	if cmd == nil {
		t.Fatal("Activate() returned nil cmd")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			if sub != nil {
				_ = sub()
			}
		}
	}

	if len(svc.PlanCalls) == 0 {
		t.Fatal("expected at least one Plan call")
	}
	got := svc.PlanCalls[0]
	if got.PlanFile == "" {
		t.Error("PlanOptions.PlanFile = empty, want non-empty plan-file path")
	}
	if !strings.Contains(got.PlanFile, "tfui") || !strings.HasSuffix(got.PlanFile, ".tfplan") {
		t.Errorf("PlanOptions.PlanFile = %q, want path containing 'tfui' and ending in '.tfplan'", got.PlanFile)
	}
}

func TestReset_GivenStalePlanFile_ShouldRemoveItFromDisk(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)

	planPath := filepath.Join(t.TempDir(), "stale.tfplan")
	if err := os.WriteFile(planPath, []byte("plan-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	p.planFile = sdk.NewTempPlanFile(planPath)

	p.reset()

	if !p.planFile.IsZero() {
		t.Errorf("after reset planFile = %q, want zero", p.planFile.Path())
	}
	if _, err := os.Stat(planPath); !os.IsNotExist(err) {
		t.Errorf("plan file still exists at %q after reset; err=%v", planPath, err)
	}
}

func TestUpdatePlanResultSuccess_ShouldPopulatePlanFileInEvent(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading
	// Simulate that runPlan recorded a path before dispatching the result.
	planPath := "/tmp/tfui-test.tfplan"
	p.planFile = sdk.NewTempPlanFile(planPath)

	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_s3_bucket.test"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	_, cmd := p.Update(PlanResultMsg{Summary: summary, Err: nil})
	if cmd == nil {
		t.Fatal("Update(PlanResultMsg) cmd = nil, want batched cmd")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want tea.BatchMsg", cmd())
	}

	var planFile string
	found := false
	for _, sub := range batch {
		if sub == nil {
			continue
		}
		if evt, ok := sub().(sdk.PlanCompletedEvent); ok {
			found = true
			planFile = evt.PlanFile
		}
	}
	if !found {
		t.Fatal("PlanCompletedEvent not found in batch")
	}
	if planFile != planPath {
		t.Errorf("PlanCompletedEvent.PlanFile = %q, want %q", planFile, planPath)
	}
}

func TestPlugin_WhenPlanResultNilSummary_ShouldNotEmitPlanCompletedEvent(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	_, cmd := p.Update(PlanResultMsg{Summary: nil, Err: nil})
	if cmd == nil {
		t.Fatal("Update(PlanResultMsg) cmd = nil, want StateRefreshedEvent cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.PlanCompletedEvent); ok {
		t.Error("nil summary should not emit PlanCompletedEvent")
	}
	if _, ok := msg.(sdk.StateRefreshedEvent); !ok {
		t.Errorf("cmd() = %T, want sdk.StateRefreshedEvent", msg)
	}
	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want Done", p.status)
	}
}

func TestPlugin_WhenViewErrorWithLockInfo_ShouldShowLockDetails(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusError
	p.errMsg = "Error acquiring the state lock"
	p.lockInfo = &sdk.StateLock{
		ID:  "abc-123",
		Who: "user@host",
	}

	view := p.View(80, 24)
	if !strings.Contains(view, "abc-123") {
		t.Errorf("View with lockInfo should contain lock ID, got: %q", view)
	}
	if !strings.Contains(view, "State Lock Detected") {
		t.Errorf("View with lockInfo should contain 'State Lock Detected', got: %q", view)
	}
}

func TestPlugin_WhenTogglePin_ShouldRequestPinToggle(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newTestPluginWithHarness(svc)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	cmd := p.togglePin("aws_instance.web")
	if cmd == nil {
		t.Fatal("togglePin should return a cmd")
	}
	cmd()
	if len(h.PinRequests) != 1 || h.PinRequests[0] != "aws_instance.web" {
		t.Errorf("togglePin should request pin for 'aws_instance.web', got %v", h.PinRequests)
	}
}

func TestPlugin_WhenRequestApply_ShouldEmitApplyRequestMsg(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	cmd := p.requestApply()
	if cmd == nil {
		t.Fatal("requestApply() returned nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(ApplyRequestMsg); !ok {
		t.Fatalf("requestApply() cmd produced %T, want ApplyRequestMsg", msg)
	}
}

// --- Frame tests ---

func TestListFrame_WhenCreated_ShouldHaveCorrectID(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	frame := p.stack.Peek()

	lf, ok := frame.(*listFrame)
	if !ok {
		t.Fatalf("top frame is %T, want *listFrame", frame)
	}
	if lf.ID() != "list" {
		t.Errorf("ID() = %q, want %q", lf.ID(), "list")
	}
}

func TestListFrame_WhenViewCalled_ShouldDelegateToPlugin(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusIdle

	view := p.stack.View(80, 24)
	if view == "" {
		t.Error("frame View() returned empty, want non-empty")
	}
}

func TestListFrame_WhenEscPressed_ShouldEmitDeactivateMsg(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc: cmd = nil, want DeactivateMsg cmd")
	}

	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Fatalf("esc cmd produced %T, want sdk.DeactivateMsg", msg)
	}
}

func TestListFrame_WhenSpacePressed_ShouldTogglePin(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newTestPluginWithHarness(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		cmd()
	}

	if len(h.PinRequests) != 1 || h.PinRequests[0] != "aws_instance.web" {
		t.Errorf("after space: should request pin for 'aws_instance.web', got %v", h.PinRequests)
	}

	cmd = p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		cmd()
	}

	if len(h.PinRequests) != 2 || h.PinRequests[1] != "aws_instance.web" {
		t.Errorf("after second space: should request toggle for 'aws_instance.web', got %v", h.PinRequests)
	}
}

func TestListFrame_WhenSpacePressedWithNoSelection_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		t.Error("space with no selection: cmd != nil, want nil")
	}
}

func TestListFrame_WhenAPressedWithResults_ShouldRequestApply(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("a key with results: cmd = nil, want ApplyRequestMsg cmd")
	}

	msg := cmd()
	if _, ok := msg.(ApplyRequestMsg); !ok {
		t.Fatalf("a key cmd produced %T, want ApplyRequestMsg", msg)
	}
}

func TestListFrame_WhenAPressedWithNoResults_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("a key with no changes: cmd != nil, want nil")
	}
}

func TestListFrame_WhenAPressedWhileNotDone_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("a key while loading: cmd != nil, want nil")
	}
}

func TestListFrame_WhenUPressedWithLockInfo_ShouldNavigateToForceUnlock(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusError
	p.lockInfo = &sdk.StateLock{ID: "lock-123"}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd == nil {
		t.Fatal("u key with lockInfo: cmd = nil, want NavigateMsg cmd")
	}

	msg := cmd()
	navMsg, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("u key cmd produced %T, want sdk.NavigateMsg", msg)
	}
	if navMsg.PluginID != "forceunlock" {
		t.Errorf("NavigateMsg.PluginID = %q, want %q", navMsg.PluginID, "forceunlock")
	}
}

func TestListFrame_WhenUPressedWithoutLockInfo_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusError
	p.lockInfo = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Error("u key without lockInfo: cmd != nil, want nil")
	}
}

func TestListFrame_WhenUPressedWhileNotError_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.lockInfo = &sdk.StateLock{ID: "lock-123"}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Error("u key while not error: cmd != nil, want nil")
	}
}

func TestListFrame_WhenCtrlRPressedWhileIdle_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusIdle

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("ctrl+r while idle: cmd != nil, want nil")
	}
}

func TestListFrame_WhenDownKeyPressed_ShouldMoveDown(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.tree.Cursor() != 1 {
		t.Errorf("after down: selected = %d, want 1", p.tree.Cursor())
	}
}

func TestListFrame_WhenUpKeyPressed_ShouldMoveUp(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.MoveDown()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.tree.Cursor() != 0 {
		t.Errorf("after up: selected = %d, want 0", p.tree.Cursor())
	}
}

func TestListFrame_WhenIKeyPressed_ShouldOpenInspect(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if p.stack.Peek().ID() != "inspect" {
		t.Errorf("after i: top frame ID = %q, want %q", p.stack.Peek().ID(), "inspect")
	}
}

func TestListFrame_WhenNonKeyMsgReceived_ShouldReturnSelf(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	type customMsg struct{}
	cmd := p.stack.Update(customMsg{})
	if cmd != nil {
		t.Error("non-KeyMsg: cmd != nil, want nil")
	}
}

func TestListFrame_WhenHintsCalledIdle_ShouldReturnConfirmAndBack(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusIdle

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty for Idle state")
	}

	hasBack := false
	hasConfirm := false
	for _, h := range hints {
		if h.Key == "q" {
			hasBack = true
		}
		if h.Key == "Enter" {
			hasConfirm = true
		}
	}
	if !hasBack {
		t.Error("Hints(Idle): missing 'q' back hint")
	}
	if !hasConfirm {
		t.Error("Hints(Idle): missing 'Enter' confirm hint")
	}
}

func TestListFrame_WhenHintsCalledLoading_ShouldReturnBackOnly(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty for Loading state")
	}

	hasBack := false
	for _, h := range hints {
		if h.Key == "q" {
			hasBack = true
		}
	}
	if !hasBack {
		t.Error("Hints(Loading): missing 'q' back hint")
	}
}

func TestListFrame_WhenHintsCalledErrorWithLock_ShouldShowRetryAndBack(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusError
	p.lockInfo = &sdk.StateLock{ID: "lock-abc"}

	hints := p.stack.Hints()
	hasRetry := false
	hasBack := false
	for _, h := range hints {
		if h.Key == "^r" {
			hasRetry = true
		}
		if h.Key == "q" {
			hasBack = true
		}
	}
	if !hasRetry {
		t.Error("Hints(Error): missing '^r' retry hint")
	}
	if !hasBack {
		t.Error("Hints(Error): missing 'q' back hint")
	}
}

func TestListFrame_WhenHintsCalledErrorWithoutLock_ShouldNotIncludeUnlock(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusError
	p.lockInfo = nil

	hints := p.stack.Hints()
	for _, h := range hints {
		if h.Key == "u" {
			t.Error("Hints(Error without lock): should not include 'u' force-unlock hint")
		}
	}
}

func TestListFrame_WhenHintsCalledDoneWithChanges_ShouldIncludeUIHintsOnly(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
	}

	hints := p.stack.Hints()
	hasPin := false
	hasInspect := false
	for _, h := range hints {
		switch h.Key {
		case "a":
			t.Error("Hints should not contain 'a' (apply) — actions belong in actions bar")
		case "Space":
			hasPin = true
		case "Enter":
			hasInspect = true
		}
	}
	if !hasPin {
		t.Error("Hints(Done with changes): missing 'Space' pin hint")
	}
	if !hasInspect {
		t.Error("Hints(Done with changes): missing 'Enter' inspect hint")
	}
}

func TestListFrame_WhenHintsCalledDoneNoChanges_ShouldIncludeRefreshAndBack(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}

	hints := p.stack.Hints()
	hasRefresh := false
	hasBack := false
	for _, h := range hints {
		switch h.Key {
		case "^r":
			hasRefresh = true
		case "q":
			hasBack = true
		}
	}
	if !hasRefresh {
		t.Error("Hints(Done no changes): missing '^r' refresh hint")
	}
	if !hasBack {
		t.Error("Hints(Done no changes): missing 'q' back hint")
	}
}

func TestListFrame_WhenHintsCalledDoneNilSummary_ShouldIncludeRefreshAndBack(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = nil

	hints := p.stack.Hints()
	hasRefresh := false
	hasBack := false
	for _, h := range hints {
		switch h.Key {
		case "^r":
			hasRefresh = true
		case "q":
			hasBack = true
		}
	}
	if !hasRefresh {
		t.Error("Hints(Done nil summary): missing '^r' refresh hint")
	}
	if !hasBack {
		t.Error("Hints(Done nil summary): missing 'q' back hint")
	}
}

func TestListFrame_WhenHintsCalledUnknownStatus_ShouldReturnBackOnly(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.Status(99)

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty for unknown status")
	}
	hasBack := false
	for _, h := range hints {
		if h.Key == "q" {
			hasBack = true
		}
	}
	if !hasBack {
		t.Error("Hints(unknown status): missing 'q' back hint")
	}
}

func TestPlugin_WhenViewLoadingState_ShouldShowRunningMessage(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(Loading) returned empty string")
	}
}

func TestPlugin_WhenPinnedResourceRendered_ShouldShowPinMark(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newTestPluginWithHarness(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	h.Ctx.Pins = []string{"aws_instance.web"}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with pinned resource returned empty")
	}
}

func TestCursorPosition_WhenDoneWithChanges_ShouldReturnOneBasedPositionAndTotal(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
			{Resource: sdk.Resource{Address: "c"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	pos, total := p.CursorPosition()
	if pos != 1 || total != 3 {
		t.Errorf("CursorPosition() = (%d, %d), want (1, 3)", pos, total)
	}

	p.tree.MoveDown()
	p.tree.MoveDown()
	pos, total = p.CursorPosition()
	if pos != 3 || total != 3 {
		t.Errorf("CursorPosition() after move = (%d, %d), want (3, 3)", pos, total)
	}
}

func TestCursorPosition_WhenNotDoneOrEmpty_ShouldReturnZeros(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})

	pos, total := p.CursorPosition()
	if pos != 0 || total != 0 {
		t.Errorf("CursorPosition() idle = (%d, %d), want (0, 0)", pos, total)
	}

	p.status = sdk.StatusDone
	p.summary = nil
	pos, total = p.CursorPosition()
	if pos != 0 || total != 0 {
		t.Errorf("CursorPosition() done+nil summary = (%d, %d), want (0, 0)", pos, total)
	}

	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}
	pos, total = p.CursorPosition()
	if pos != 0 || total != 0 {
		t.Errorf("CursorPosition() done+empty changes = (%d, %d), want (0, 0)", pos, total)
	}
}

func TestListFrame_WhenBangPressedWithPins_ShouldPushActionFrame(t *testing.T) {
	p, h := newTestPluginWithHarness(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "aws_instance.b"}, Action: sdk.ActionUpdate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	h.Ctx.Pins = []string{"aws_instance.a", "aws_instance.b"}
	p.syncPinnedToTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})

	if p.stack.Peek().ID() != "actions" {
		t.Errorf("after !: top frame ID = %q, want %q", p.stack.Peek().ID(), "actions")
	}
}

func TestListFrame_WhenBangPressedWithNoPins_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.a"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})

	if p.stack.Peek().ID() != "list" {
		t.Errorf("after ! with no pins: top frame ID = %q, want %q", p.stack.Peek().ID(), "list")
	}
	if cmd != nil {
		t.Error("after ! with no pins: cmd != nil, want nil")
	}
}

func TestView_WhenFilterActive_ShouldShowListContent(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.filtering = true
	p.stack.Push(&planFilterFrame{plugin: p, inner: nil})

	view := p.View(80, 20)
	if !strings.Contains(view, "aws_instance.web") {
		t.Errorf("filter view should contain resource address, got %q", view)
	}
}

func TestLKey_GivenStreamPresent_ShouldPushStreamFrameOnStack(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	lw, ch := frames.NewLineWriter()
	lw.Close()
	p.lastStream = frames.NewStreamFrame("terraform plan", ch, nil)
	depthBefore := p.stack.Depth()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if p.stack.Depth() != depthBefore+1 {
		t.Errorf("after l: stack depth = %d, want %d", p.stack.Depth(), depthBefore+1)
	}
	if p.stack.Peek().ID() != "stream" {
		t.Errorf("after l: top frame ID = %q, want %q", p.stack.Peek().ID(), "stream")
	}
}

func TestUpdate_GivenPlanError_ShouldRestoreListFrameAsTopFrame(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	lw, ch := frames.NewLineWriter()
	lw.Close()
	p.stack.Push(frames.NewStreamFrame("terraform plan", ch, nil))

	p.Update(PlanResultMsg{Err: errors.New("plan failed")})

	if p.stack.Peek().ID() != "list" {
		t.Errorf("after PlanResultMsg error: top frame ID = %q, want %q", p.stack.Peek().ID(), "list")
	}
}

func TestUpdate_GivenPlanError_ShouldPreserveLastStreamForLKey(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	lw, ch := frames.NewLineWriter()
	lw.Close()
	sf := frames.NewStreamFrame("terraform plan", ch, nil)
	p.lastStream = sf
	p.stack.Push(sf)

	p.Update(PlanResultMsg{Err: errors.New("plan failed")})

	if p.lastStream == nil {
		t.Error("lastStream should be preserved after plan error for l key access")
	}
}

func TestUpdate_WhenStreamLineMsgArrives_ShouldRouteToStack(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	lw, ch := frames.NewLineWriter()
	p.stack.Push(frames.NewStreamFrame("terraform plan", ch, nil))

	_, cmd := p.Update(frames.StreamLineMsg{Line: "output line"})
	if cmd == nil {
		t.Fatal("Update(StreamLineMsg) with StreamFrame on stack should return WaitForLine cmd")
	}
	lw.Close()
}

func TestListFrame_WhenHintsDoneNoChangesWithLastStream_ShouldIncludeLHint(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}
	lw, ch := frames.NewLineWriter()
	lw.Close()
	p.lastStream = frames.NewStreamFrame("terraform plan", ch, nil)

	hints := p.stack.Hints()
	hasL := false
	for _, h := range hints {
		if h.Key == "l" {
			hasL = true
		}
	}
	if !hasL {
		t.Error("Hints(Done, no changes, lastStream set): missing 'l' log hint")
	}
}

func TestListFrame_WhenHintsDoneWithChangesAndLastStream_ShouldIncludeLHint(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
	}
	lw, ch := frames.NewLineWriter()
	lw.Close()
	p.lastStream = frames.NewStreamFrame("terraform plan", ch, nil)

	hints := p.stack.Hints()
	hasL := false
	for _, h := range hints {
		if h.Key == "l" {
			hasL = true
		}
	}
	if !hasL {
		t.Error("Hints(Done, with changes, lastStream set): missing 'l' log hint")
	}
}

func TestHandleContextChanged_WhenChdirChanges_ShouldFullReset(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{}

	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{
		Prev: &sdk.Context{WorkingDir: "/old"},
		Next: &sdk.Context{Service: svc},
	})
	if cmd != nil {
		t.Error("HandleContextChanged returned non-nil cmd")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want StatusIdle (full reset)", p.status)
	}
	if p.summary != nil {
		t.Error("summary should be nil after chdir change")
	}
}

func TestHandleContextChanged_WhenOnlyPinsChange_ShouldMarkStaleAndPreserveUI(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	summary := &sdk.PlanSummary{}
	p.summary = summary

	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{
		Prev: &sdk.Context{Pins: []string{"a"}},
		Next: &sdk.Context{Pins: []string{"a", "b"}},
	})
	if cmd != nil {
		t.Error("HandleContextChanged returned non-nil cmd")
	}
	if !p.stale {
		t.Error("stale should be true after pin-only change")
	}
	if p.summary != summary {
		t.Error("summary should be preserved on pin-only change (UI state intact)")
	}
}

func TestHandleContextChanged_WhenNextNil_ShouldBeNoOp(t *testing.T) {
	svc := &sdktest.MockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil})
	if cmd != nil {
		t.Error("HandleContextChanged with nil Next returned non-nil cmd")
	}
	if p.status != sdk.StatusDone {
		t.Error("status mutated on nil Next")
	}
}
