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
	p, _, _ := newAssessedPlugin(t, nil, sdk.PolicyAssessment{Status: sdk.AssessmentEmpty})
	runLifecycleCmd(p, p.Activate(Input{}))
	if p.Status() != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.Status())
	}
	if p.groups != nil {
		t.Error("groups = non-nil, want nil")
	}
	if p.overall() != sdk.RiskNone {
		t.Errorf("overall = %v, want RiskNone", p.overall())
	}
	if len(p.items) != 0 {
		t.Errorf("total = %d, want 0", len(p.items))
	}
	if !p.Ready() {
		t.Error("Ready() = false after Analyze, want true")
	}
}

func TestAnalyze_WhenEmptyChanges_ShouldSetDoneWithNilGroups(t *testing.T) {
	p, _, _ := newAssessedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{}}, sdk.PolicyAssessment{Status: sdk.AssessmentEmpty})
	runLifecycleCmd(p, p.Activate(Input{}))
	if p.Status() != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.Status())
	}
	if p.groups != nil {
		t.Error("groups = non-nil after empty changes, want nil")
	}
}

func TestAnalyze_WhenChangesPresent_ShouldGroupByRiskLevelDescending(t *testing.T) {
	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical},
			{Resource: sdk.Resource{Address: "aws_vpc.main"}, Action: sdk.ActionUpdate, Risk: sdk.RiskMedium},
			{Resource: sdk.Resource{Address: "aws_iam_role.admin"}, Action: sdk.ActionUpdate, Risk: sdk.RiskHigh},
			{Resource: sdk.Resource{Address: "aws_lambda.fn"}, Action: sdk.ActionCreate, Risk: sdk.RiskNone},
		},
	}

	p := newAnalyzedPlugin(t, summary)

	if p.Status() != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", p.Status())
	}
	if len(p.items) != 5 {
		t.Errorf("total = %d, want 5", len(p.items))
	}
	if p.overall() != sdk.RiskCritical {
		t.Errorf("overall = %v, want RiskCritical", p.overall())
	}
	if len(p.groups) == 0 {
		t.Error("groups is empty, want non-empty")
	}
	if p.selected != 0 {
		t.Errorf("selected = %d, want 0 after Analyze", p.selected)
	}

	// Verify ordering: highest risk first
	if p.groups[0].level != sdk.RiskCritical {
		t.Errorf("groups[0].level = %v, want RiskCritical", p.groups[0].level)
	}
}

func TestAnalyze_WhenAllRiskLevels_ShouldCreateOneGroupPerLevel(t *testing.T) {
	summary := &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Risk: sdk.RiskCritical},
			{Resource: sdk.Resource{Address: "b"}, Risk: sdk.RiskHigh},
			{Resource: sdk.Resource{Address: "c"}, Risk: sdk.RiskMedium},
			{Resource: sdk.Resource{Address: "d"}, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "e"}, Risk: sdk.RiskNone},
		},
	}

	p := newAnalyzedPlugin(t, summary)
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

func TestOverall_GivenAssessment_ShouldReturnItsOverallLevel(t *testing.T) {
	high := sdk.RiskHigh
	for _, tt := range []struct {
		name       string
		assessment sdk.PolicyAssessment
		want       sdk.RiskLevel
	}{
		{"complete", sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high}, sdk.RiskHigh},
		{"without overall level", sdk.PolicyAssessment{Status: sdk.AssessmentPartial}, sdk.RiskNone},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := New(&sdktest.MockService{}).(*Plugin)
			p.assessment = &tt.assessment
			if got := p.overall(); got != tt.want {
				t.Errorf("overall() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdate_WhenJKPressed_ShouldNavigateUpAndDown(t *testing.T) {
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Risk: sdk.RiskHigh},
			{Resource: sdk.Resource{Address: "b"}, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "c"}, Risk: sdk.RiskLow},
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

	p.items = make([]riskItem, 3)

	p.moveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown: selected = %d, want 1", p.selected)
	}

	p.moveDown()
	p.moveDown()
	p.moveDown()
	// should be at 2 (max)
	if p.selected != 2 {
		t.Errorf("MoveDown multiple: selected = %d, want 2", p.selected)
	}

	// Should not go past max
	p.moveDown()
	if p.selected != 2 {
		t.Errorf("MoveDown boundary: selected = %d, want 2", p.selected)
	}

	// Move up
	p.moveUp()
	if p.selected != 1 {
		t.Errorf("MoveUp: selected = %d, want 1", p.selected)
	}

	// Move up to 0
	p.selected = 0
	p.moveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp boundary: selected = %d, want 0", p.selected)
	}
}

func TestNavigation_WhenEmptyGroups_ShouldNotMoveSelection(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.items = nil

	p.moveDown()
	if p.selected != 0 {
		t.Errorf("MoveDown empty: selected = %d, want 0", p.selected)
	}
}

func TestView_WhenIdle_ShouldShowWaitingMessage(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.outcome.kind = outcomeIdle

	view := p.View(80, 24)
	if !strings.Contains(view, "plan") || !strings.Contains(view, "risk") {
		t.Errorf("view should indicate waiting for plan to analyze risk, got %q", view)
	}
}

func TestView_WhenDoneWithNilGroups_ShouldRenderNonEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.outcome.kind = outcomeDone
	p.groups = nil

	view := p.View(80, 24)
	if !strings.Contains(view, "No eligible mutations") {
		t.Errorf("view should indicate no changes to analyze, got %q", view)
	}
}

func TestView_WhenDoneWithEmptyGroups_ShouldRenderNonEmpty(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.outcome.kind = outcomeDone
	p.groups = []riskGroup{}

	view := p.View(80, 24)
	if !strings.Contains(view, "No eligible mutations") {
		t.Errorf("view should indicate no changes to analyze, got %q", view)
	}
}

func TestView_WhenDoneWithGroups_ShouldRenderRiskGroupList(t *testing.T) {
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical},
		{Resource: sdk.Resource{Address: "aws_iam_role.admin"}, Action: sdk.ActionUpdate, Risk: sdk.RiskHigh},
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow, IsPhantom: true},
	}})

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
	p.outcome.kind = outcomeKind(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestView_WhenSmallHeight_ShouldStillRender(t *testing.T) {
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}, Risk: sdk.RiskMedium}}})

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
		p.assessment = &sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &tt.overall}
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
		group := riskGroup{level: level, items: []riskItem{{}}}
		result := p.renderGroupHeader(group)
		if result == "" {
			t.Errorf("renderGroupHeader(level=%v): got empty", level)
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
	p.items = make([]riskItem, 2)

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
		result := p.renderChangeRow(riskItem{change: tt.change})
		if result == "" {
			t.Errorf("renderChangeRow(%v): got empty", tt.change.Resource.Address)
		}
	}
}

func TestRenderStats_WhenGroupsPresent_ShouldReturnStatsSummary(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.groups = []riskGroup{
		{level: sdk.RiskHigh, items: make([]riskItem, 2)},
		{level: sdk.RiskLow, items: make([]riskItem, 3)},
	}

	result := p.renderStats()
	if result == "" {
		t.Error("renderStats: got empty")
	}
}

func TestView_WhenManyChanges_ShouldHandleScrolling(t *testing.T) {
	// Create groups with many items to trigger scrolling
	changes := make([]sdk.PlanChange, 20)
	for i := range changes {
		changes[i] = sdk.PlanChange{
			Resource: sdk.Resource{Address: "res_" + string(rune('a'+i%26))},
			Action:   sdk.ActionCreate,
			Risk:     sdk.RiskHigh,
		}
	}
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: changes})
	p.selected = 15

	// verifies rendering completes without panic under scrolling
	view := p.View(80, 12)
	if view == "" {
		t.Error("View with scrolling/maxVisible returned empty string")
	}
}

func TestHints_WhenStatusDoneWithGroups_ShouldOfferInspectAndQuit(t *testing.T) {
	p := newAnalyzedPlugin(t, lifecycleSummary("aws_instance.web"))
	hints := p.Hints()
	want := (sdk.HintSetInspect | sdk.HintSetQuit).Hints()
	if len(hints) != len(want) {
		t.Fatalf("Hints() = %v, want %v", hints, want)
	}
	for i := range want {
		if hints[i] != want[i] {
			t.Errorf("Hints()[%d] = %v, want %v", i, hints[i], want[i])
		}
	}
}

func TestHints_WhenStatusIdle_ShouldReturnBackHint(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.outcome.kind = outcomeIdle

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
	p.outcome.kind = outcomeDone
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
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh},
	}})
	p.selected = 0

	view := p.View(80, 24)
	if !strings.Contains(view, "aws_instance.web") {
		t.Errorf("view should contain selected resource address 'aws_instance.web', got %q", view)
	}
}

func TestView_WhenMaxVisibleOverflow_ShouldTruncateRenderedItems(t *testing.T) {
	changes := make([]sdk.PlanChange, 30)
	for i := range changes {
		changes[i] = sdk.PlanChange{
			Resource: sdk.Resource{Address: "r_" + string(rune('a'+i%26))},
			Action:   sdk.ActionCreate,
			Risk:     sdk.RiskHigh,
		}
	}
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: changes})

	// verifies rendering completes without panic under overflow
	view := p.View(80, 15)
	if view == "" {
		t.Error("View with maxVisible overflow returned empty string")
	}
}

func TestView_WhenMultipleGroupsOverflow_ShouldHandleGracefully(t *testing.T) {
	// Two groups where first group fills the visible area completely
	changes1 := make([]sdk.PlanChange, 10)
	for i := range changes1 {
		changes1[i] = sdk.PlanChange{Resource: sdk.Resource{Address: "g1_" + string(rune('a'+i%26))}, Risk: sdk.RiskCritical}
	}
	changes2 := make([]sdk.PlanChange, 5)
	for i := range changes2 {
		changes2[i] = sdk.PlanChange{Resource: sdk.Resource{Address: "g2_" + string(rune('a'+i%26))}, Risk: sdk.RiskLow}
	}
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: append(changes1, changes2...)})

	// verifies rendering completes without panic under overflow
	view := p.View(80, 15)
	if view == "" {
		t.Error("View with multiple groups overflow returned empty string")
	}
}

func TestView_WhenHeaderBeyondMaxVisible_ShouldNotRenderOverflow(t *testing.T) {
	changes1 := make([]sdk.PlanChange, 4)
	for i := range changes1 {
		changes1[i] = sdk.PlanChange{Resource: sdk.Resource{Address: "g1_" + string(rune('a'+i))}, Risk: sdk.RiskHigh}
	}
	changes2 := make([]sdk.PlanChange, 2)
	for i := range changes2 {
		changes2[i] = sdk.PlanChange{Resource: sdk.Resource{Address: "g2_" + string(rune('a'+i))}, Risk: sdk.RiskLow}
	}
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: append(changes1, changes2...)})

	// verifies rendering completes without panic when header overflows
	view := p.View(80, 15)
	if view == "" {
		t.Error("View with header beyond maxVisible returned empty string")
	}
}

func TestCursorPosition_WhenDoneWithGroups_ShouldReturnOneBasedPositionAndTotal(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.outcome.kind = outcomeDone
	p.items = make([]riskItem, 3)
	p.selected = 0

	pos, total := p.CursorPosition()
	if pos != 1 || total != 3 {
		t.Errorf("CursorPosition() = (%d, %d), want (1, 3)", pos, total)
	}

	p.selected = 2
	pos, total = p.CursorPosition()
	if pos != 3 || total != 3 {
		t.Errorf("CursorPosition() after move = (%d, %d), want (3, 3)", pos, total)
	}
}

func TestCursorPosition_WhenNotDoneOrEmpty_ShouldReturnZeros(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	pos, total := p.CursorPosition()
	if pos != 0 || total != 0 {
		t.Errorf("CursorPosition() idle = (%d, %d), want (0, 0)", pos, total)
	}

	p.outcome.kind = outcomeDone
	p.groups = []riskGroup{}
	pos, total = p.CursorPosition()
	if pos != 0 || total != 0 {
		t.Errorf("CursorPosition() done+empty = (%d, %d), want (0, 0)", pos, total)
	}
}

func TestAnalyze_WhenCalledAgain_ShouldRebuildRatherThanAppend(t *testing.T) {
	p := newAnalyzedPlugin(t, lifecycleSummary("aws_instance.web"))

	p.analyze(p.summary, p.assessment.Rows())

	if len(p.groups) != 1 || len(p.items) != 1 {
		t.Errorf("after a second analyze: %d groups, %d items; want 1 group, 1 item", len(p.groups), len(p.items))
	}
}
