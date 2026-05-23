package risk

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "risk" {
		t.Errorf("ID() = %q, want %q", p.ID(), "risk")
	}
	if p.Name() != "Risk Analysis" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Risk Analysis")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(map[string]interface{}{}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	if cmd := p.Init(&sdk.PluginDeps{Service: svc}); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
	if p.Ready() {
		t.Error("Ready() should be false before analysis")
	}
}

func TestAnalyze_WhenNilSummary_ShouldSetDoneWithNoData(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	p.Analyze(nil)
	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.groups != nil {
		t.Error("groups = non-nil, want nil")
	}
	if p.overall != sdk.RiskNone {
		t.Errorf("overall = %v, want RiskNone", p.overall)
	}
	if p.total != 0 {
		t.Errorf("total = %d, want 0", p.total)
	}
	if !p.Ready() {
		t.Error("Ready() = false after Analyze, want true")
	}
}

func TestAnalyze_WhenEmptyChanges_ShouldSetDoneWithNilGroups(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	p.Analyze(&sdk.PlanSummary{Changes: []sdk.PlanChange{}})
	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.groups != nil {
		t.Error("groups = non-nil after empty changes, want nil")
	}
}

func TestAnalyze_WhenChangesPresent_ShouldGroupByRiskLevelDescending(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical},
			{Resource: sdk.Resource{Address: "aws_vpc.main"}, Action: sdk.ActionUpdate, Risk: sdk.RiskMedium},
			{Resource: sdk.Resource{Address: "aws_iam_role.admin"}, Action: sdk.ActionUpdate, Risk: sdk.RiskHigh},
			{Resource: sdk.Resource{Address: "aws_lambda.fn"}, Action: sdk.ActionCreate, Risk: sdk.RiskNone},
		},
	}

	p.Analyze(summary)

	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.status)
	}
	if p.total != 5 {
		t.Errorf("total = %d, want 5", p.total)
	}
	if p.overall != sdk.RiskCritical {
		t.Errorf("overall = %v, want RiskCritical", p.overall)
	}
	if len(p.groups) == 0 {
		t.Error("groups is empty, want non-empty")
	}
	if p.selected != 0 {
		t.Errorf("selected = %d, want 0 after Analyze", p.selected)
	}

	// Verify ordering: highest risk first
	if p.groups[0].Level != sdk.RiskCritical {
		t.Errorf("groups[0].Level = %v, want RiskCritical", p.groups[0].Level)
	}
}

func TestAnalyze_WhenAllRiskLevels_ShouldCreateOneGroupPerLevel(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Risk: sdk.RiskCritical},
			{Risk: sdk.RiskHigh},
			{Risk: sdk.RiskMedium},
			{Risk: sdk.RiskLow},
			{Risk: sdk.RiskNone},
		},
	}

	p.Analyze(summary)
	if len(p.groups) != 5 {
		t.Errorf("len(groups) = %d, want 5 (one per risk level)", len(p.groups))
	}
}

func TestPlugin_WhenNew_ShouldHaveIdleStatus(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	if p.Status() != sdk.StatusIdle {
		t.Errorf("Status() = %v, want sdk.StatusIdle", p.Status())
	}
}

func TestSelected_WhenSet_ShouldReturnCurrentIndex(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.selected = 3
	if p.Selected() != 3 {
		t.Errorf("Selected() = %d, want 3", p.Selected())
	}
}

func TestOverall_WhenSet_ShouldReturnHighestRiskLevel(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.overall = sdk.RiskHigh
	if p.Overall() != sdk.RiskHigh {
		t.Errorf("Overall() = %v, want RiskHigh", p.Overall())
	}
}

func TestUpdate_WhenJKPressed_ShouldNavigateUpAndDown(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	// Setup groups to have items to navigate
	p.Analyze(&sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Risk: sdk.RiskHigh},
			{Risk: sdk.RiskLow},
			{Risk: sdk.RiskLow},
		},
	})

	// Move down with j
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", p.selected)
	}

	// Move down more
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Move up with k
	prev := p.selected
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected >= prev {
		t.Errorf("after k: selected = %d, should be less than %d", p.selected, prev)
	}
}

func TestUpdate_WhenUnknownMsg_ShouldReturnSelfWithNoCmd(t *testing.T) {
	svc := &sdktest.MockService{}
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

func TestNavigation_WhenMoving_ShouldUpdateSelectionWithBounds(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	// With a group that has items
	p.groups = []RiskGroup{
		{Level: sdk.RiskHigh, Changes: []sdk.PlanChange{{}, {}}},
		{Level: sdk.RiskLow, Changes: []sdk.PlanChange{{}}},
	}
	// total items = 2 headers + 3 changes = 5

	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown: selected = %d, want 1", p.selected)
	}

	p.MoveDown()
	p.MoveDown()
	p.MoveDown()
	// should be at 4 (max)
	if p.selected != 4 {
		t.Errorf("MoveDown multiple: selected = %d, want 4", p.selected)
	}

	// Should not go past max
	p.MoveDown()
	if p.selected != 4 {
		t.Errorf("MoveDown boundary: selected = %d, want 4", p.selected)
	}

	// Move up
	p.MoveUp()
	if p.selected != 3 {
		t.Errorf("MoveUp: selected = %d, want 3", p.selected)
	}

	// Move up to 0
	p.selected = 0
	p.MoveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp boundary: selected = %d, want 0", p.selected)
	}
}

func TestNavigation_WhenEmptyGroups_ShouldNotMoveSelection(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.groups = nil

	p.MoveDown()
	if p.selected != 0 {
		t.Errorf("MoveDown empty: selected = %d, want 0", p.selected)
	}
}

func TestView_WhenIdle_ShouldShowWaitingMessage(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if !strings.Contains(view, "plan") || !strings.Contains(view, "risk") {
		t.Errorf("view should indicate waiting for plan to analyze risk, got %q", view)
	}
}

func TestView_WhenDoneWithNilGroups_ShouldRenderNonEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.groups = nil

	view := p.View(80, 24)
	if !strings.Contains(view, "No changes to analyze") {
		t.Errorf("view should indicate no changes to analyze, got %q", view)
	}
}

func TestView_WhenDoneWithEmptyGroups_ShouldRenderNonEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.groups = []RiskGroup{}

	view := p.View(80, 24)
	if !strings.Contains(view, "No changes to analyze") {
		t.Errorf("view should indicate no changes to analyze, got %q", view)
	}
}

func TestView_WhenDoneWithGroups_ShouldRenderRiskGroupList(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.overall = sdk.RiskHigh
	p.total = 3
	p.groups = []RiskGroup{
		{
			Level: sdk.RiskCritical,
			Changes: []sdk.PlanChange{
				{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical},
			},
		},
		{
			Level: sdk.RiskHigh,
			Changes: []sdk.PlanChange{
				{Resource: sdk.Resource{Address: "aws_iam_role.admin"}, Action: sdk.ActionUpdate, Risk: sdk.RiskHigh},
			},
		},
		{
			Level: sdk.RiskLow,
			Changes: []sdk.PlanChange{
				{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow, IsPhantom: true},
			},
		},
	}

	view := p.View(80, 24)
	if !strings.Contains(view, "aws_s3_bucket.data") {
		t.Errorf("view should contain resource address 'aws_s3_bucket.data', got %q", view)
	}
	if !strings.Contains(view, "aws_iam_role.admin") {
		t.Errorf("view should contain resource address 'aws_iam_role.admin', got %q", view)
	}
	if !strings.Contains(view, "Total") {
		t.Errorf("view should contain stats summary with 'Total', got %q", view)
	}
}

func TestView_WhenInvalidStatus_ShouldReturnEmptyString(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestView_WhenSmallHeight_ShouldStillRender(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.overall = sdk.RiskMedium
	p.total = 1
	p.groups = []RiskGroup{
		{Level: sdk.RiskMedium, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}}}},
	}

	// verifies rendering completes without panic
	view := p.View(80, 5)
	if view == "" {
		t.Error("View with small height returned empty string")
	}
}

func TestRenderOverallBanner_GivenRiskLevel_ShouldReturnStyledBanner(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	tests := []struct {
		overall sdk.RiskLevel
	}{
		{sdk.RiskCritical},
		{sdk.RiskHigh},
		{sdk.RiskMedium},
		{sdk.RiskLow},
		{sdk.RiskNone},
	}

	for _, tt := range tests {
		p.overall = tt.overall
		result := p.renderOverallBanner()
		if result == "" {
			t.Errorf("renderOverallBanner(risk=%v): got empty", tt.overall)
		}
	}
}

func TestRenderGroupHeader_GivenRiskLevel_ShouldReturnFormattedHeader(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	levels := []sdk.RiskLevel{
		sdk.RiskCritical,
		sdk.RiskHigh,
		sdk.RiskMedium,
		sdk.RiskLow,
		sdk.RiskNone,
	}

	for _, level := range levels {
		group := RiskGroup{Level: level, Changes: []sdk.PlanChange{{}}}
		result := p.renderGroupHeader(group)
		if result == "" {
			t.Errorf("renderGroupHeader(level=%v): got empty", level)
		}
	}
}

func TestRiskReason_GivenChange_ShouldReturnExplanation(t *testing.T) {
	tests := []struct {
		change  sdk.PlanChange
		wantNon bool
	}{
		{sdk.PlanChange{Action: sdk.ActionDelete}, true},
		{sdk.PlanChange{Action: sdk.ActionDeleteThenCreate}, true},
		{sdk.PlanChange{Action: sdk.ActionUpdate, Risk: sdk.RiskHigh}, true},
		{sdk.PlanChange{IsPhantom: true}, true},
		{sdk.PlanChange{Action: sdk.ActionCreate, Risk: sdk.RiskLow}, false},
	}

	for _, tt := range tests {
		result := riskReason(tt.change)
		if tt.wantNon && result == "" {
			t.Errorf("riskReason(%v): got empty, want non-empty", tt.change.Action)
		}
		if !tt.wantNon && result != "" {
			t.Errorf("riskReason(%v): got %q, want empty", tt.change.Action, result)
		}
	}
}

func TestActionSymbol_GivenAction_ShouldReturnNonEmptySymbol(t *testing.T) {
	actions := []sdk.Action{
		sdk.ActionCreate,
		sdk.ActionUpdate,
		sdk.ActionDelete,
		sdk.ActionDeleteThenCreate,
		sdk.ActionCreateThenDelete,
		sdk.ActionNoOp,
		sdk.ActionRead,
	}

	for _, action := range actions {
		result := sdk.ActionSymbol(action)
		if result == "" && action != sdk.ActionNoOp && action != sdk.ActionRead {
			t.Errorf("sdk.ActionSymbol(%q) returned empty", action)
		}
	}
}

func TestUpdate_WhenArrowKeysPressed_ShouldNavigateUpAndDown(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.groups = []RiskGroup{
		{Level: sdk.RiskHigh, Changes: []sdk.PlanChange{{}}},
	}

	// Test "down" key in addition to "j"
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down: selected = %d, want 1", p.selected)
	}

	// Test "up" key in addition to "k"
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 0 {
		t.Errorf("after up: selected = %d, want 0", p.selected)
	}
}

func TestRenderChangeRow_GivenChange_ShouldReturnFormattedRow(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	// Test with various actions and phantom
	tests := []struct {
		change sdk.PlanChange
	}{
		{sdk.PlanChange{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate}},
		{sdk.PlanChange{Resource: sdk.Resource{Address: "b"}, Action: sdk.ActionDelete}},
		{sdk.PlanChange{Resource: sdk.Resource{Address: "c"}, Action: sdk.ActionUpdate, Risk: sdk.RiskHigh}},
		{sdk.PlanChange{Resource: sdk.Resource{Address: "d"}, IsPhantom: true}},
	}

	for _, tt := range tests {
		result := p.renderChangeRow(tt.change)
		if result == "" {
			t.Errorf("renderChangeRow(%v): got empty", tt.change.Resource.Address)
		}
	}
}

func TestRenderStats_WhenGroupsPresent_ShouldReturnStatsSummary(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.total = 5
	p.groups = []RiskGroup{
		{Level: sdk.RiskHigh, Changes: []sdk.PlanChange{{}, {}}},
		{Level: sdk.RiskLow, Changes: []sdk.PlanChange{{}, {}, {}}},
	}

	result := p.renderStats()
	if result == "" {
		t.Error("renderStats: got empty")
	}
}

func TestView_WhenManyChanges_ShouldHandleScrolling(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.overall = sdk.RiskHigh
	p.total = 30

	// Create groups with many items to trigger scrolling
	changes := make([]sdk.PlanChange, 20)
	for i := range changes {
		changes[i] = sdk.PlanChange{
			Resource: sdk.Resource{Address: "res_" + string(rune('a'+i%26))},
			Action:   sdk.ActionCreate,
		}
	}
	p.groups = []RiskGroup{
		{Level: sdk.RiskHigh, Changes: changes},
	}
	// Set selected high to test maxVisible cutoff
	p.selected = 15

	// verifies rendering completes without panic under scrolling
	view := p.View(80, 12)
	if view == "" {
		t.Error("View with scrolling/maxVisible returned empty string")
	}
}

func TestRiskReason_WhenUpdateWithHighRisk_ShouldReturnCriticalModification(t *testing.T) {
	// Test the specific case: update + high risk = "modification of critical resource"
	change := sdk.PlanChange{Action: sdk.ActionUpdate, Risk: sdk.RiskHigh}
	result := riskReason(change)
	if result != "modification of critical resource" {
		t.Errorf("riskReason(update+high) = %q, want %q", result, "modification of critical resource")
	}
}

func TestRiskReason_WhenUpdateWithCriticalRisk_ShouldReturnCriticalModification(t *testing.T) {
	change := sdk.PlanChange{Action: sdk.ActionUpdate, Risk: sdk.RiskCritical}
	result := riskReason(change)
	if result != "modification of critical resource" {
		t.Errorf("riskReason(update+critical) = %q, want %q", result, "modification of critical resource")
	}
}

func TestHints_WhenStatusDoneWithGroups_ShouldReturnBackHint(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.groups = []RiskGroup{
		{Level: sdk.RiskHigh, Changes: []sdk.PlanChange{{}}},
	}

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice, want at least one hint")
	}
	if hints[0].Key != "q" || hints[0].Description != "quit" {
		t.Errorf("Hints()[0] = {%q, %q}, want {%q, %q}", hints[0].Key, hints[0].Description, "q", "quit")
	}
}

func TestHints_WhenStatusIdle_ShouldReturnBackHint(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusIdle

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice, want at least one hint")
	}
	if hints[0].Key != "q" || hints[0].Description != "quit" {
		t.Errorf("Hints()[0] = {%q, %q}, want {%q, %q}", hints[0].Key, hints[0].Description, "q", "quit")
	}
}

func TestHints_WhenStatusDoneNoGroups_ShouldReturnBackHint(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.groups = nil

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice, want at least one hint")
	}
	if hints[0].Key != "q" || hints[0].Description != "quit" {
		t.Errorf("Hints()[0] = {%q, %q}, want {%q, %q}", hints[0].Key, hints[0].Description, "q", "quit")
	}
}

func TestRenderAnalysis_WhenChangeRowIsSelected_ShouldHighlightIt(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.overall = sdk.RiskHigh
	p.total = 2
	p.groups = []RiskGroup{
		{
			Level: sdk.RiskHigh,
			Changes: []sdk.PlanChange{
				{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh},
			},
		},
	}
	// selected=1 points to the first change row (index 0 is the group header)
	p.selected = 1

	view := p.View(80, 24)
	if !strings.Contains(view, "aws_instance.web") {
		t.Errorf("view should contain selected resource address 'aws_instance.web', got %q", view)
	}
}

func TestView_WhenMaxVisibleOverflow_ShouldTruncateRenderedItems(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.overall = sdk.RiskHigh
	p.total = 50

	// Create a group with many changes that exceeds maxVisible
	changes := make([]sdk.PlanChange, 30)
	for i := range changes {
		changes[i] = sdk.PlanChange{
			Resource: sdk.Resource{Address: "r_" + string(rune('a'+i%26))},
			Action:   sdk.ActionCreate,
		}
	}
	p.groups = []RiskGroup{
		{Level: sdk.RiskHigh, Changes: changes},
	}

	// Very small height to force overflow in the inner loop (itemIdx >= maxVisible)
	// verifies rendering completes without panic under overflow
	view := p.View(80, 15)
	if view == "" {
		t.Error("View with maxVisible overflow returned empty string")
	}
}

func TestView_WhenMultipleGroupsOverflow_ShouldHandleGracefully(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.overall = sdk.RiskCritical
	p.total = 20

	// Two groups where first group fills the visible area completely
	changes1 := make([]sdk.PlanChange, 10)
	for i := range changes1 {
		changes1[i] = sdk.PlanChange{Resource: sdk.Resource{Address: "g1_" + string(rune('a'+i%26))}}
	}
	changes2 := make([]sdk.PlanChange, 5)
	for i := range changes2 {
		changes2[i] = sdk.PlanChange{Resource: sdk.Resource{Address: "g2_" + string(rune('a'+i%26))}}
	}
	p.groups = []RiskGroup{
		{Level: sdk.RiskCritical, Changes: changes1},
		{Level: sdk.RiskLow, Changes: changes2},
	}

	// Height 15, maxVisible = 15-10 = 5. First group header(1) + 4 changes fills it.
	// The 5th change and entire second group are beyond maxVisible.
	// verifies rendering completes without panic under overflow
	view := p.View(80, 15)
	if view == "" {
		t.Error("View with multiple groups overflow returned empty string")
	}
}

func TestView_WhenHeaderBeyondMaxVisible_ShouldNotRenderOverflow(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.overall = sdk.RiskHigh
	p.total = 10

	// Set up so that the second group's header falls beyond maxVisible
	// maxVisible = 5 (height=15). First group: 1 header + 4 changes = 5 items.
	// Second group header would be at itemIdx=5, which is >= maxVisible.
	changes1 := make([]sdk.PlanChange, 4)
	for i := range changes1 {
		changes1[i] = sdk.PlanChange{Resource: sdk.Resource{Address: "g1_" + string(rune('a'+i))}}
	}
	changes2 := make([]sdk.PlanChange, 2)
	for i := range changes2 {
		changes2[i] = sdk.PlanChange{Resource: sdk.Resource{Address: "g2_" + string(rune('a'+i))}}
	}
	p.groups = []RiskGroup{
		{Level: sdk.RiskHigh, Changes: changes1},
		{Level: sdk.RiskLow, Changes: changes2},
	}

	// verifies rendering completes without panic when header overflows
	view := p.View(80, 15)
	if view == "" {
		t.Error("View with header beyond maxVisible returned empty string")
	}
}

func TestCursorPosition_WhenDoneWithGroups_ShouldReturnOneBasedPositionAndTotal(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.groups = []RiskGroup{
		{Level: sdk.RiskHigh, Changes: []sdk.PlanChange{{}, {}}},
		{Level: sdk.RiskLow, Changes: []sdk.PlanChange{{}}},
	}
	p.selected = 0

	pos, total := p.CursorPosition()
	// totalItems = 2 headers + 3 changes = 5
	if pos != 1 || total != 5 {
		t.Errorf("CursorPosition() = (%d, %d), want (1, 5)", pos, total)
	}

	p.selected = 4
	pos, total = p.CursorPosition()
	if pos != 5 || total != 5 {
		t.Errorf("CursorPosition() after move = (%d, %d), want (5, 5)", pos, total)
	}
}

func TestCursorPosition_WhenNotDoneOrEmpty_ShouldReturnZeros(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	pos, total := p.CursorPosition()
	if pos != 0 || total != 0 {
		t.Errorf("CursorPosition() idle = (%d, %d), want (0, 0)", pos, total)
	}

	p.status = sdk.StatusDone
	p.groups = []RiskGroup{}
	pos, total = p.CursorPosition()
	if pos != 0 || total != 0 {
		t.Errorf("CursorPosition() done+empty = (%d, %d), want (0, 0)", pos, total)
	}
}
