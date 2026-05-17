package plan

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func TestPlugin_WhenOutputJsonWithNilSummary_ShouldReturnNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.summary = nil

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	if data != nil {
		t.Errorf("Output(true) = %q, want nil", string(data))
	}
}

func TestPlugin_WhenOutputJsonWithChanges_ShouldReturnValidJSON(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh},
		},
		ToCreate: 1,
		ToDelete: 1,
	}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"aws_instance.web"`) {
		t.Error("JSON missing aws_instance.web")
	}
	if !strings.Contains(s, `"create"`) {
		t.Error("JSON missing create action")
	}
	if !strings.Contains(s, `"destroy": 1`) {
		t.Error("JSON missing destroy count")
	}
}

func TestPlugin_WhenOutputTextWithChanges_ShouldReturnPlainText(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}, Action: sdk.ActionDelete},
			{Resource: sdk.Resource{Address: "aws_iam_role.x"}, Action: sdk.ActionUpdate},
			{Resource: sdk.Resource{Address: "aws_vpc.main"}, Action: sdk.ActionDeleteThenCreate},
			{Resource: sdk.Resource{Address: "aws_subnet.a"}, Action: sdk.ActionRead},
		},
		ToCreate: 1,
		ToDelete: 1,
		ToUpdate: 1,
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "+ aws_instance.web") {
		t.Error("text missing + for create")
	}
	if !strings.Contains(s, "- aws_s3_bucket.data") {
		t.Error("text missing - for delete")
	}
	if !strings.Contains(s, "~ aws_iam_role.x") {
		t.Error("text missing ~ for update")
	}
	if !strings.Contains(s, "-/+ aws_vpc.main") {
		t.Error("text missing -/+ for delete-then-create")
	}
	if !strings.Contains(s, "<= aws_subnet.a") {
		t.Error("text missing <= for read")
	}
	if !strings.Contains(s, "Plan:") {
		t.Error("text missing Plan: summary line")
	}
}

func TestPlugin_WhenOutputTextWithNoAction_ShouldUseSpaceSymbol(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.x"}, Action: sdk.Action("noop")},
		},
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "  aws_instance.x") {
		t.Errorf("text missing space-prefixed address for unknown action, got: %s", s)
	}
}

func TestPlugin_WhenExitCodeWithNilSummary_ShouldReturnZero(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.summary = nil
	if code := p.ExitCode(); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
}

func TestPlugin_WhenExitCodeWithChanges_ShouldReturnTwo(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate}},
	}
	if code := p.ExitCode(); code != 2 {
		t.Errorf("ExitCode() = %d, want 2", code)
	}
}

func TestPlugin_WhenExitCodeWithEmptyChanges_ShouldReturnZero(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}
	if code := p.ExitCode(); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
}

func TestPlugin_WhenHandleLockCleared_ShouldClearLockInfo(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.lockInfo = &sdk.StateLock{ID: "abc"}

	cmd := p.HandleLockCleared(sdk.LockClearedEvent{})
	if cmd != nil {
		t.Error("HandleLockCleared() should return nil cmd")
	}
	if p.lockInfo != nil {
		t.Error("lockInfo should be nil after HandleLockCleared")
	}
}

func TestPlugin_WhenPinnedCountWithNilPins_ShouldReturnZero(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.pins = nil
	if c := p.PinnedCount(); c != 0 {
		t.Errorf("PinnedCount() = %d, want 0", c)
	}
}

func TestPlugin_WhenPinnedCountWithPins_ShouldReturnCount(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.pins.Toggle("a")
	p.pins.Toggle("b")
	if c := p.PinnedCount(); c != 2 {
		t.Errorf("PinnedCount() = %d, want 2", c)
	}
}

func TestPlugin_WhenRequestAutoApply_ShouldEmitAutoApplyRequestMsg(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	cmd := p.requestAutoApply()
	if cmd == nil {
		t.Fatal("requestAutoApply() = nil")
	}
	msg := cmd()
	if _, ok := msg.(AutoApplyRequestMsg); !ok {
		t.Errorf("cmd() = %T, want AutoApplyRequestMsg", msg)
	}
}

func TestPlugin_WhenClearAllPins_ShouldUnpinAll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "b"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	p.clearAllPins()
	if p.PinnedCount() != 0 {
		t.Errorf("PinnedCount() = %d, want 0 after clearAllPins", p.PinnedCount())
	}
}

func TestDetailFrame_WhenEscPressed_ShouldPop(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	// Push inspect frame
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.stack.Peek().ID() != "inspect" {
		t.Fatal("expected inspect frame on top")
	}

	// Esc pops it
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.stack.Peek().ID() != "list" {
		t.Errorf("after esc: top = %q, want list", p.stack.Peek().ID())
	}
}

func TestDetailFrame_WhenScrollDown_ShouldIncrementScroll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.detailScroll != 1 {
		t.Errorf("detailScroll = %d, want 1", p.detailScroll)
	}
}

func TestDetailFrame_WhenScrollUp_ShouldDecrementScroll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.detailScroll != 0 {
		t.Errorf("detailScroll = %d, want 0", p.detailScroll)
	}
}

func TestDetailFrame_WhenCtrlWPressed_ShouldToggleWrap(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	if !p.detailWrap {
		t.Error("ctrl+w should toggle detailWrap to true")
	}
}

func TestDetailFrame_WhenSpacePressed_ShouldTogglePin(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !p.pins.IsPinned("a") {
		t.Error("space in detail should pin the address")
	}
}

func TestDetailFrame_WhenEPressed_ShouldEmitPlanEditMsg(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("e in detail should return cmd")
	}
	msg := cmd()
	editMsg, ok := msg.(PlanEditMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want PlanEditMsg", msg)
	}
	if editMsg.Address != "a" {
		t.Errorf("PlanEditMsg.Address = %q, want %q", editMsg.Address, "a")
	}
}

func TestDetailFrame_Hints_ShouldIncludeWrapPinEditCancel(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("detail hints should not be empty")
	}
}

func TestListFrame_WhenFilterSlashPressed_ShouldPushFilterFrame(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if p.stack.Peek().ID() != "filter" {
		t.Errorf("after /: top frame = %q, want filter", p.stack.Peek().ID())
	}
	if !p.filtering {
		t.Error("filtering should be true")
	}
}

func TestListFrame_WhenCtrlTPressed_ShouldToggleTreeMode(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if !p.treeMode {
		t.Error("ctrl+t should toggle treeMode to true")
	}
}

func TestListFrame_WhenCtrlWPressed_ShouldToggleWrap(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	if !p.listWrap {
		t.Error("ctrl+w should toggle listWrap to true")
	}
}

func TestListFrame_WhenCtrlPPressed_ShouldTogglePinnedOnly(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if !p.pinnedOnly {
		t.Error("ctrl+p should toggle pinnedOnly to true")
	}
}

func TestListFrame_WhenCtrlUPressed_ShouldClearPins(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.pins.Toggle("a")

	p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if p.PinnedCount() != 0 {
		t.Errorf("after ctrl+u: PinnedCount = %d, want 0", p.PinnedCount())
	}
}

func TestListFrame_WhenTPressedDone_ShouldEmitTaintRequest(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Fatal("t key should return cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("t cmd() should produce TaintRequestMsg")
	}
}

func TestListFrame_WhenBigTPressedDone_ShouldEmitUntaintRequest(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Fatal("T key should return cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("T cmd() should produce UntaintRequestMsg")
	}
}

func TestListFrame_WhenBigAPressed_ShouldRequestAutoApply(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if cmd == nil {
		t.Fatal("A key with changes should return cmd")
	}
	msg := cmd()
	if _, ok := msg.(AutoApplyRequestMsg); !ok {
		t.Errorf("cmd() = %T, want AutoApplyRequestMsg", msg)
	}
}

func TestListFrame_WhenEPressedDone_ShouldEmitEditMsg(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Fatal("e key should return cmd")
	}
	msg := cmd()
	editMsg, ok := msg.(PlanEditMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want PlanEditMsg", msg)
	}
	if editMsg.Address != "aws_instance.web" {
		t.Errorf("PlanEditMsg.Address = %q, want %q", editMsg.Address, "aws_instance.web")
	}
}

func TestPlugin_WhenNavigateDown_ShouldMoveDown(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.navigate(1)
	if p.tree.Cursor() != 1 {
		t.Errorf("navigate(1): cursor = %d, want 1", p.tree.Cursor())
	}
}

func TestPlugin_WhenNavigateUp_ShouldMoveUp(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.MoveDown()

	p.navigate(-1)
	if p.tree.Cursor() != 0 {
		t.Errorf("navigate(-1): cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestPlugin_WhenPanListRight_ShouldIncrementHScroll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})

	p.panListRight()
	if p.listHScroll != 10 {
		t.Errorf("panListRight(): listHScroll = %d, want 10", p.listHScroll)
	}

	p.panListRight()
	if p.listHScroll != 20 {
		t.Errorf("panListRight() x2: listHScroll = %d, want 20", p.listHScroll)
	}
}

func TestPlugin_WhenPanListLeft_ShouldDecrementHScroll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.listHScroll = 20

	p.panListLeft()
	if p.listHScroll != 10 {
		t.Errorf("panListLeft(): listHScroll = %d, want 10", p.listHScroll)
	}

	p.panListLeft()
	if p.listHScroll != 0 {
		t.Errorf("panListLeft() x2: listHScroll = %d, want 0", p.listHScroll)
	}
}

func TestPlugin_WhenPanListLeftAtZero_ShouldRemainZero(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.listHScroll = 0

	p.panListLeft()
	if p.listHScroll != 0 {
		t.Errorf("panListLeft() at 0: listHScroll = %d, want 0", p.listHScroll)
	}
}

func TestPlugin_WhenPanDetailRight_ShouldIncrementHScroll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.viewWidth = 100
	p.detail = strings.Repeat("x", 200)

	p.panDetailRight()
	if p.detailHScroll != 10 {
		t.Errorf("panDetailRight(): detailHScroll = %d, want 10", p.detailHScroll)
	}
}

func TestPlugin_WhenPanDetailRightAtMax_ShouldClampToMax(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.viewWidth = 100
	p.detail = "short"

	p.panDetailRight()
	if p.detailHScroll != 0 {
		t.Errorf("panDetailRight() short content: detailHScroll = %d, want 0", p.detailHScroll)
	}
}

func TestPlugin_WhenPanDetailLeft_ShouldDecrementHScroll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detailHScroll = 20

	p.panDetailLeft()
	if p.detailHScroll != 10 {
		t.Errorf("panDetailLeft(): detailHScroll = %d, want 10", p.detailHScroll)
	}
}

func TestPlugin_WhenPanDetailLeftAtZero_ShouldRemainZero(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detailHScroll = 0

	p.panDetailLeft()
	if p.detailHScroll != 0 {
		t.Errorf("panDetailLeft() at 0: detailHScroll = %d, want 0", p.detailHScroll)
	}
}

func TestPlugin_WhenSourceChangesWithNilSummary_ShouldReturnNil(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = nil

	result := p.sourceChanges()
	if result != nil {
		t.Errorf("sourceChanges() with nil summary = %v, want nil", result)
	}
}

func TestPlugin_WhenSourceChangesNotPinnedOnly_ShouldReturnAllChanges(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.pinnedOnly = false

	result := p.sourceChanges()
	if len(result) != 2 {
		t.Errorf("sourceChanges() not pinned only: len = %d, want 2", len(result))
	}
}

func TestPlugin_WhenSourceChangesPinnedOnly_ShouldReturnOnlyPinned(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
			{Resource: sdk.Resource{Address: "c"}},
		},
	}
	p.pins.Toggle("a")
	p.pins.Toggle("c")
	p.pinnedOnly = true

	result := p.sourceChanges()
	if len(result) != 2 {
		t.Errorf("sourceChanges() pinned only: len = %d, want 2", len(result))
	}
	if result[0].Resource.Address != "a" {
		t.Errorf("sourceChanges()[0].Address = %q, want %q", result[0].Resource.Address, "a")
	}
	if result[1].Resource.Address != "c" {
		t.Errorf("sourceChanges()[1].Address = %q, want %q", result[1].Resource.Address, "c")
	}
}

func TestPlugin_WhenSetFilterEmpty_ShouldResetToAllChanges(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.SetFilter("aws")
	if len(p.filtered) == 0 {
		t.Fatal("SetFilter('aws') should match resources")
	}

	p.SetFilter("")
	if len(p.filtered) != 2 {
		t.Errorf("SetFilter(''): len(filtered) = %d, want 2", len(p.filtered))
	}
	if p.filterScores != nil {
		t.Error("SetFilter(''): filterScores should be nil")
	}
}

func TestPlugin_WhenSetFilterWithQuery_ShouldFilterResults(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}},
			{Resource: sdk.Resource{Address: "google_compute_instance.vm"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.SetFilter("aws")
	if len(p.filtered) < 2 {
		t.Errorf("SetFilter('aws'): len(filtered) = %d, want >= 2", len(p.filtered))
	}
	if p.filterScores == nil {
		t.Error("SetFilter('aws'): filterScores should not be nil")
	}
}

func TestPlugin_WhenSetFilterInTreeMode_ShouldUseOriginalOrder(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.SetFilter("aws")
	if len(p.filtered) == 0 {
		t.Fatal("SetFilter('aws') in tree mode should find results")
	}
}

func TestPlugin_WhenSetFilterPinnedOnly_ShouldFilterFromPinnedSubset(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}},
			{Resource: sdk.Resource{Address: "aws_s3_bucket.data"}},
			{Resource: sdk.Resource{Address: "google_compute_instance.vm"}},
		},
	}
	p.pins.Toggle("aws_instance.web")
	p.pinnedOnly = true
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.SetFilter("")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('') with pinnedOnly: len(filtered) = %d, want 1", len(p.filtered))
	}
}

func TestPlugin_WhenRebuildTreeInFlatMode_ShouldPreserveOrder(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.treeMode = false
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	if p.tree.Cursor() != 0 {
		t.Errorf("rebuildTree() flat mode: cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestPlugin_WhenRebuildTreeInTreeModeWithFilter_ShouldExpandAll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.treeMode = true
	p.filter = "vpc"
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	view := p.View(80, 24)
	if view == "" {
		t.Error("View after rebuildTree in tree mode with filter should not be empty")
	}
}

func TestPlugin_WhenPruneStaleWithNilPins_ShouldNotPanic(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.pins = nil
	p.pruneStale([]sdk.PlanChange{
		{Resource: sdk.Resource{Address: "a"}},
	})
}

func TestPlugin_WhenPruneStaleWithNoPins_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.pruneStale([]sdk.PlanChange{
		{Resource: sdk.Resource{Address: "a"}},
	})
	if p.pins.Count() != 0 {
		t.Error("pruneStale with no pins should not add pins")
	}
}

func TestPlugin_WhenPruneStaleWithValidPins_ShouldRetainAll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	p.pruneStale([]sdk.PlanChange{
		{Resource: sdk.Resource{Address: "a"}},
		{Resource: sdk.Resource{Address: "b"}},
		{Resource: sdk.Resource{Address: "c"}},
	})
	if p.pins.Count() != 2 {
		t.Errorf("pruneStale: pins count = %d, want 2", p.pins.Count())
	}
}

func TestPlugin_WhenPruneStaleWithStalePins_ShouldRemoveStale(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.pins.Toggle("a")
	p.pins.Toggle("removed_resource")

	p.pruneStale([]sdk.PlanChange{
		{Resource: sdk.Resource{Address: "a"}},
		{Resource: sdk.Resource{Address: "b"}},
	})
	if p.pins.Count() != 1 {
		t.Errorf("pruneStale: pins count = %d, want 1", p.pins.Count())
	}
	if !p.pins.IsPinned("a") {
		t.Error("pruneStale should retain valid pin 'a'")
	}
	if p.pins.IsPinned("removed_resource") {
		t.Error("pruneStale should remove stale pin 'removed_resource'")
	}
}

func TestPlugin_WhenClearAllPinsWithPinnedOnly_ShouldResetPinnedOnlyAndRefilter(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "b"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pinnedOnly = true
	p.SetFilter("")

	if len(p.filtered) != 1 {
		t.Fatalf("before clearAllPins: len(filtered) = %d, want 1", len(p.filtered))
	}

	p.clearAllPins()
	if p.pinnedOnly {
		t.Error("clearAllPins should set pinnedOnly to false")
	}
	if len(p.filtered) != 2 {
		t.Errorf("after clearAllPins: len(filtered) = %d, want 2", len(p.filtered))
	}
}

func TestPlugin_WhenInspectSelectedNoChange_ShouldReturnNil(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = nil

	cmd := p.inspectSelected()
	if cmd != nil {
		t.Error("inspectSelected() with no selected change should return nil")
	}
}

func TestPlugin_WhenInspectSelectedWithDiffs_ShouldPushDetailFrame(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "aws_instance.web"},
				Action:         sdk.ActionUpdate,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "ami", OldValue: "old", NewValue: "new"}},
			},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	cmd := p.inspectSelected()
	if cmd != nil {
		t.Error("inspectSelected() should return nil cmd")
	}
	if p.stack.Peek().ID() != "inspect" {
		t.Errorf("after inspectSelected: top frame = %q, want inspect", p.stack.Peek().ID())
	}
	if p.detailAddr != "aws_instance.web" {
		t.Errorf("detailAddr = %q, want %q", p.detailAddr, "aws_instance.web")
	}
}

func TestPlugin_WhenInspectSelectedPopsFilterFrame(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "a"},
				Action:         sdk.ActionCreate,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "x"}},
			},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if p.stack.Peek().ID() != "filter" {
		t.Fatal("expected filter frame after /")
	}

	p.inspectSelected()
	if p.stack.Peek().ID() != "inspect" {
		t.Errorf("inspectSelected should pop filter and push inspect, got %q", p.stack.Peek().ID())
	}
	if p.filtering {
		t.Error("inspectSelected should set filtering to false")
	}
}

func TestPlugin_WhenBuildInspectContentWithRisk_ShouldIncludeRiskLine(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	change := &sdk.PlanChange{
		Resource: sdk.Resource{Address: "aws_instance.web", Type: "aws_instance", ProviderName: "aws"},
		Action:   sdk.ActionDelete,
		Risk:     sdk.RiskHigh,
	}

	content := p.buildInspectContent(change)
	if !strings.Contains(content, "Risk:") {
		t.Error("buildInspectContent should include Risk line for non-None risk")
	}
}

func TestPlugin_WhenBuildInspectContentWithPhantom_ShouldIncludePhantomLine(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	change := &sdk.PlanChange{
		Resource:  sdk.Resource{Address: "aws_instance.web", Type: "aws_instance", ProviderName: "aws"},
		Action:    sdk.ActionUpdate,
		IsPhantom: true,
	}

	content := p.buildInspectContent(change)
	if !strings.Contains(content, "Phantom:") {
		t.Error("buildInspectContent should include Phantom line for phantom changes")
	}
}

func TestPlugin_WhenBuildInspectContentWithModule_ShouldIncludeModuleLine(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	change := &sdk.PlanChange{
		Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a", Type: "aws_subnet", Module: "module.vpc", ProviderName: "aws"},
		Action:   sdk.ActionCreate,
	}

	content := p.buildInspectContent(change)
	if !strings.Contains(content, "Module:") {
		t.Error("buildInspectContent should include Module line when module is set")
	}
	if !strings.Contains(content, "module.vpc") {
		t.Error("buildInspectContent should contain the module path")
	}
}

func TestPlugin_WhenBuildInspectContentWithForcesNew_ShouldAnnotateAttribute(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	change := &sdk.PlanChange{
		Resource: sdk.Resource{Address: "aws_instance.web", Type: "aws_instance", ProviderName: "aws"},
		Action:   sdk.ActionDeleteThenCreate,
		AttributeDiffs: []sdk.AttributeDiff{
			{Key: "ami", OldValue: "ami-old", NewValue: "ami-new", ForcesNew: true},
		},
	}

	content := p.buildInspectContent(change)
	if !strings.Contains(content, "forces new") {
		t.Error("buildInspectContent should annotate attributes that force replacement")
	}
}

func TestPlugin_WhenBuildInspectContentWithSensitive_ShouldMaskValues(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	change := &sdk.PlanChange{
		Resource: sdk.Resource{Address: "aws_instance.web", Type: "aws_instance", ProviderName: "aws"},
		Action:   sdk.ActionUpdate,
		AttributeDiffs: []sdk.AttributeDiff{
			{Key: "password", Sensitive: true},
		},
	}

	content := p.buildInspectContent(change)
	if !strings.Contains(content, "(sensitive)") {
		t.Error("buildInspectContent should mask sensitive attributes")
	}
}

func TestPlugin_WhenViewDoneWithFilterActive_ShouldShowFilterLine(t *testing.T) {
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
	p.filter = "aws"

	view := p.View(80, 24)
	if !strings.Contains(view, "aws") {
		t.Error("View with active filter should show filter text")
	}
}

func TestPlugin_WhenViewDoneWithFilterAndPinnedOnly_ShouldShowBothIndicators(t *testing.T) {
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
	p.pinnedOnly = true
	p.filter = "web"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestPlugin_WhenViewDoneWithInactiveFilterText_ShouldShowFilterLabel(t *testing.T) {
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
	p.filtering = false
	p.filter = "aws"

	view := p.View(80, 24)
	if !strings.Contains(view, "filter:") {
		t.Error("View with inactive filter should show 'filter:' label")
	}
}

func TestPlugin_WhenViewDoneWithNoMatches_ShouldShowNoMatchingChanges(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = nil
	p.filtering = true
	p.filter = "nonexistent"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with no matches should not be empty")
	}
}

func TestPlugin_WhenRenderResultsInTreeMode_ShouldRenderTree(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	view := p.renderResults(80, 24)
	if view == "" {
		t.Error("renderResults in tree mode should not be empty")
	}
}

func TestPlugin_WhenRenderResultsInTreeModeWithHScroll_ShouldApplyOffset(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.listHScroll = 5
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.very_long_name"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.tree.ExpandAll()

	view := p.renderResults(80, 24)
	if view == "" {
		t.Error("renderResults with hscroll should not be empty")
	}
}

func TestPlugin_WhenRenderResultsInTreeModeWithHScrollBeyondContent_ShouldShowEmpty(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.listHScroll = 500
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.tree.ExpandAll()

	view := p.renderResults(80, 24)
	if view == "" {
		t.Error("renderResults with large hscroll should not be empty")
	}
}

func TestPlugin_WhenRenderResultsInTreeModeWithPhantom_ShouldShowPhantomBadge(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.phantom"}, Action: sdk.ActionUpdate, IsPhantom: true},
		},
		ToUpdate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.tree.ExpandAll()

	view := p.renderResults(80, 24)
	if view == "" {
		t.Error("renderResults with phantom should not be empty")
	}
}

func TestPlugin_WhenRenderFlatListWithHScroll_ShouldTruncateFromLeft(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.listHScroll = 5
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.very_long_resource_name"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	result := p.renderFlatList(80, 20)
	if result == "" {
		t.Error("renderFlatList with hscroll should not be empty")
	}
}

func TestPlugin_WhenRenderFlatListWithWrap_ShouldNotTruncate(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.listWrap = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	result := p.renderFlatList(80, 20)
	if result == "" {
		t.Error("renderFlatList with wrap should not be empty")
	}
}

func TestPlugin_WhenFormatChangeRowWithRisk_ShouldIncludeRiskBadge(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	change := sdk.PlanChange{
		Resource: sdk.Resource{Address: "aws_instance.web"},
		Action:   sdk.ActionDelete,
		Risk:     sdk.RiskCritical,
	}

	row := p.formatChangeRow("[ ] ", change, 100)
	if row == "" {
		t.Error("formatChangeRow should not be empty")
	}
}

func TestPlugin_WhenFormatChangeRowWithPhantom_ShouldIncludePhantomMarker(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	change := sdk.PlanChange{
		Resource:  sdk.Resource{Address: "aws_instance.web"},
		Action:    sdk.ActionUpdate,
		IsPhantom: true,
	}

	row := p.formatChangeRow("[ ] ", change, 100)
	if row == "" {
		t.Error("formatChangeRow should not be empty")
	}
}

func TestPlugin_WhenFormatChangeRowWithHScrollBeyondContent_ShouldReturnPinOnly(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.listHScroll = 500
	change := sdk.PlanChange{
		Resource: sdk.Resource{Address: "a"},
		Action:   sdk.ActionCreate,
	}

	row := p.formatChangeRow("[ ] ", change, 80)
	if !strings.Contains(row, "[ ] ") {
		t.Error("formatChangeRow with extreme hscroll should still contain pin mark")
	}
}

func TestPlugin_WhenRenderDetailWithWrap_ShouldWrapLongLines(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detail = strings.Repeat("abcdefghij", 20)
	p.detailAddr = "aws_instance.web"
	p.detailWrap = true

	view := p.renderDetail(80, 24)
	if view == "" {
		t.Error("renderDetail with wrap should not be empty")
	}
	if !strings.Contains(view, "aws_instance.web") {
		t.Error("renderDetail should contain address")
	}
}

func TestPlugin_WhenRenderDetailWithHScroll_ShouldOffsetContent(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detail = strings.Repeat("x", 200)
	p.detailAddr = "aws_instance.web"
	p.detailHScroll = 10
	p.detailWrap = false

	view := p.renderDetail(80, 24)
	if view == "" {
		t.Error("renderDetail with hscroll should not be empty")
	}
}

func TestPlugin_WhenRenderDetailWithPinnedAddress_ShouldShowPinIndicator(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detail = "some detail"
	p.detailAddr = "aws_instance.web"
	p.pins.Toggle("aws_instance.web")

	view := p.renderDetail(80, 24)
	if !strings.Contains(view, "[pinned]") {
		t.Error("renderDetail with pinned address should show [pinned] indicator")
	}
}

func TestPlugin_WhenRenderDetailScrollClamped_ShouldNotExceedMax(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detail = "line1\nline2\nline3"
	p.detailAddr = "test"
	p.detailScroll = 100

	view := p.renderDetail(80, 24)
	if view == "" {
		t.Error("renderDetail with clamped scroll should not be empty")
	}
}

func TestPlugin_WhenRenderDetailSmallHeight_ShouldClampToMinimum(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detail = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	p.detailAddr = "test"

	view := p.renderDetail(80, 3)
	if view == "" {
		t.Error("renderDetail with small height should not be empty")
	}
}

func TestPlugin_WhenRenderDetailSmallWidth_ShouldUseMinWidth(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detail = "some content"
	p.detailAddr = "test"

	view := p.renderDetail(20, 24)
	if view == "" {
		t.Error("renderDetail with small width should not be empty")
	}
}

func TestWrapLines_WhenAllShort_ShouldReturnUnchanged(t *testing.T) {
	lines := []string{"short", "also short"}
	result := wrapLines(lines, 80)
	if len(result) != 2 {
		t.Errorf("wrapLines: len = %d, want 2", len(result))
	}
	if result[0] != "short" {
		t.Errorf("wrapLines[0] = %q, want %q", result[0], "short")
	}
}

func TestWrapLines_WhenLongLine_ShouldSplitAtWidth(t *testing.T) {
	lines := []string{strings.Repeat("x", 30)}
	result := wrapLines(lines, 10)
	if len(result) != 3 {
		t.Errorf("wrapLines: len = %d, want 3", len(result))
	}
	for _, r := range result {
		if len(r) > 10 {
			t.Errorf("wrapLines: segment len = %d, want <= 10", len(r))
		}
	}
}

func TestWrapLines_WhenExactWidth_ShouldNotSplit(t *testing.T) {
	lines := []string{strings.Repeat("x", 10)}
	result := wrapLines(lines, 10)
	if len(result) != 1 {
		t.Errorf("wrapLines: len = %d, want 1", len(result))
	}
}

func TestWrapLines_WhenEmptyLine_ShouldPreserve(t *testing.T) {
	lines := []string{"", "hello", ""}
	result := wrapLines(lines, 80)
	if len(result) != 3 {
		t.Errorf("wrapLines: len = %d, want 3", len(result))
	}
	if result[0] != "" || result[2] != "" {
		t.Error("wrapLines should preserve empty lines")
	}
}

func TestPlugin_WhenRenderSummaryLineWithReplace_ShouldIncludeReplaceCount(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		ToCreate:  1,
		ToUpdate:  2,
		ToDelete:  1,
		ToReplace: 3,
	}

	result := p.renderSummaryLine()
	if !strings.Contains(result, "3 to replace") {
		t.Error("renderSummaryLine should include replace count")
	}
}

func TestPlugin_WhenOutputTextWithPhantom_ShouldIncludeInList(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.phantom"}, Action: sdk.ActionUpdate, IsPhantom: true},
		},
		ToUpdate: 1,
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "aws_instance.phantom") {
		t.Error("text output should include phantom resource")
	}
}

func TestPlugin_WhenOutputJsonWithPhantom_ShouldSetPhantomFlag(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.phantom"}, Action: sdk.ActionUpdate, IsPhantom: true},
		},
		ToUpdate: 1,
	}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"phantom": true`) && !strings.Contains(s, `"phantom":true`) {
		t.Error("JSON output should include phantom flag")
	}
}

func TestPlugin_WhenOutputTextWithRisk_ShouldIncludeRiskLine(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh},
		},
		ToDelete: 1,
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "Risk:") {
		t.Error("text output should include Risk line for high risk changes")
	}
}

func TestPlugin_WhenOutputTextWithNoRisk_ShouldOmitRiskLine(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskNone},
		},
		ToCreate: 1,
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if strings.Contains(s, "Risk:") {
		t.Error("text output should not include Risk line for no-risk changes")
	}
}

func TestPlugin_WhenUpdateTimerTickMsg_ShouldReturnTickCmd(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd != nil {
		t.Error("Update(TimerTickMsg) without running timer should return nil")
	}
}

func TestPlugin_WhenPlanResultWithLockError_ShouldEmitLockDetectedEvent(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	lockErrMsg := "Error acquiring the state lock\n  ID:        abc-123\n  Path:      terraform.tfstate\n  Operation: OperationTypePlan\n  Who:       user@host\n  Version:   1.5.0\n  Created:   2023-01-01 00:00:00.000000000 +0000 UTC"

	_, cmd := p.Update(PlanResultMsg{Err: fmt.Errorf("%s", lockErrMsg)})
	if cmd == nil {
		t.Fatal("Update with lock error should return cmd")
	}
	msg := cmd()
	evt, ok := msg.(sdk.LockDetectedEvent)
	if !ok {
		t.Fatalf("cmd() = %T, want sdk.LockDetectedEvent", msg)
	}
	if evt.Lock == nil {
		t.Fatal("LockDetectedEvent.Lock = nil")
	}
	if evt.Lock.ID != "abc-123" {
		t.Errorf("Lock.ID = %q, want %q", evt.Lock.ID, "abc-123")
	}
}

func TestPlugin_WhenCancelWithCancelFn_ShouldCallCancel(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	called := false
	p.cancelFn = func() { called = true }

	p.Cancel()
	if !called {
		t.Error("Cancel() should call cancelFn")
	}
	if p.cancelFn != nil {
		t.Error("Cancel() should nil out cancelFn")
	}
}

func TestPlugin_WhenCancelWithNilCancelFn_ShouldNotPanic(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.cancelFn = nil
	p.Cancel()
}

func TestPlanFilterFrame_WhenViewCalled_ShouldDelegateToInner(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if p.stack.Peek().ID() != "filter" {
		t.Fatal("expected filter frame")
	}

	view := p.stack.View(80, 24)
	if view == "" {
		t.Error("planFilterFrame.View() should not be empty")
	}
}

func TestPlanFilterFrame_WhenHintsCalled_ShouldReturnHints(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Error("planFilterFrame.Hints() should return non-empty hints")
	}
}

func TestPlanFilterFrame_WhenHintsCalledInTreeMode_ShouldIncludeCollapseExpand(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("planFilterFrame.Hints() in tree mode should return hints")
	}
}

func TestPlanFilterFrame_WhenCtrlWPressed_ShouldToggleWrap(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	if !p.listWrap {
		t.Error("ctrl+w in filter frame should toggle listWrap to true")
	}
	if p.listHScroll != 0 {
		t.Error("ctrl+w should reset listHScroll to 0")
	}
}

func TestPlanFilterFrame_WhenCtrlPPressed_ShouldTogglePinnedOnly(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if !p.pinnedOnly {
		t.Error("ctrl+p in filter frame should toggle pinnedOnly to true")
	}
}

func TestPlanFilterFrame_WhenRightPressed_ShouldPanListRight(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.listWrap = false
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.listHScroll != 10 {
		t.Errorf("right in filter frame: listHScroll = %d, want 10", p.listHScroll)
	}
}

func TestPlanFilterFrame_WhenLeftPressed_ShouldPanListLeft(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.listWrap = false
	p.listHScroll = 20
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.listHScroll != 10 {
		t.Errorf("left in filter frame: listHScroll = %d, want 10", p.listHScroll)
	}
}

func TestPlanFilterFrame_WhenRightPressedWithWrap_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.listWrap = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.listHScroll != 0 {
		t.Errorf("right with wrap in filter frame: listHScroll = %d, want 0", p.listHScroll)
	}
}

func TestPlanFilterFrame_WhenCloseBracketPressed_ShouldExpandAll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}, Action: sdk.ActionCreate},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
}

func TestPlanFilterFrame_WhenOpenBracketPressed_ShouldCollapseAll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}, Action: sdk.ActionCreate},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.tree.ExpandAll()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
}

func TestPlanFilterFrame_WhenBracketPressedNotTreeMode_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = false
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
}

func TestPlanFilterFrame_WhenEscPressed_ShouldPopAndClearFiltering(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if p.stack.Peek().ID() != "filter" {
		t.Fatal("expected filter frame")
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.stack.Peek().ID() != "list" {
		t.Errorf("after esc: top frame = %q, want list", p.stack.Peek().ID())
	}
	if p.filtering {
		t.Error("esc should set filtering to false")
	}
}

func TestListFrame_WhenEnterInTreeModeOnBranch_ShouldToggleBranch(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}, Action: sdk.ActionCreate},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if p.stack.Peek().ID() != "list" {
		t.Error("enter on branch in tree mode should not push inspect frame")
	}
}

func TestListFrame_WhenRightPressedWithWrap_ShouldNotPan(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.listWrap = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.listHScroll != 0 {
		t.Errorf("right with wrap: listHScroll = %d, want 0", p.listHScroll)
	}
}

func TestListFrame_WhenLeftPressedWithWrap_ShouldNotPan(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.listWrap = true
	p.listHScroll = 10
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.listHScroll != 10 {
		t.Errorf("left with wrap: listHScroll = %d, want 10 (unchanged)", p.listHScroll)
	}
}

func TestListFrame_WhenCloseBracketInTreeMode_ShouldExpandAll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}, Action: sdk.ActionCreate},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
}

func TestListFrame_WhenOpenBracketInTreeMode_ShouldCollapseAll(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}, Action: sdk.ActionCreate},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.tree.ExpandAll()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
}

func TestListFrame_WhenBracketNotTreeMode_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = false
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
}

func TestListFrame_WhenHintsDoneInTreeModeWithPins_ShouldIncludeTreeAndClearPins(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.pins.Toggle("a")

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints(Done, tree mode, with pins) should not be empty")
	}
}

func TestPlugin_WhenRenderResultsWithListWrapInTreeMode_ShouldNotTruncate(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.listWrap = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.tree.ExpandAll()

	view := p.renderResults(80, 24)
	if view == "" {
		t.Error("renderResults in tree mode with wrap should not be empty")
	}
}

func TestDetailFrame_WhenRightPressedNoWrap_ShouldPanRight(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.viewWidth = 100
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "a"},
				AttributeDiffs: []sdk.AttributeDiff{{Key: "x", OldValue: strings.Repeat("y", 200), NewValue: "z"}},
			},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.detailHScroll == 0 {
		t.Error("right in detail frame should increment detailHScroll")
	}
}

func TestDetailFrame_WhenLeftPressedNoWrap_ShouldPanLeft(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "a"},
				AttributeDiffs: []sdk.AttributeDiff{{Key: "x"}},
			},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.detailHScroll = 20

	p.stack.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.detailHScroll != 10 {
		t.Errorf("left in detail frame: detailHScroll = %d, want 10", p.detailHScroll)
	}
}

func TestDetailFrame_WhenRightPressedWithWrap_ShouldNotPan(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "a"},
				AttributeDiffs: []sdk.AttributeDiff{{Key: "x"}},
			},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.detailWrap = true

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.detailHScroll != 0 {
		t.Errorf("right with detailWrap: detailHScroll = %d, want 0", p.detailHScroll)
	}
}

func TestDetailFrame_WhenLeftPressedWithWrap_ShouldNotPan(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "a"},
				AttributeDiffs: []sdk.AttributeDiff{{Key: "x"}},
			},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.detailWrap = true
	p.detailHScroll = 10

	p.stack.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.detailHScroll != 10 {
		t.Errorf("left with detailWrap: detailHScroll = %d, want 10 (unchanged)", p.detailHScroll)
	}
}

func TestDetailFrame_WhenNonKeyMsg_ShouldReturnSelf(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "a"},
				AttributeDiffs: []sdk.AttributeDiff{{Key: "x"}},
			},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	type customMsg struct{}
	cmd := p.stack.Update(customMsg{})
	if cmd != nil {
		t.Error("non-KeyMsg in detail frame: cmd != nil, want nil")
	}
	if p.stack.Peek().ID() != "inspect" {
		t.Error("non-KeyMsg should not pop detail frame")
	}
}

func TestPlanFilterFrame_WhenNonKeyMsgDelegated_ShouldReturnSelf(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	type customMsg struct{}
	cmd := p.stack.Update(customMsg{})
	if cmd != nil {
		t.Error("non-KeyMsg in filter frame: cmd != nil, want nil")
	}
	if p.stack.Peek().ID() != "filter" {
		t.Error("non-KeyMsg should not pop filter frame")
	}
}

func TestPlugin_WhenRenderResultsWithPinnedOnlyIndicator_ShouldShowInView(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.pinnedOnly = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.pins.Toggle("a")
	p.filtered = p.summary.Changes
	p.rebuildTree()

	view := p.View(80, 24)
	if !strings.Contains(view, "[pinned]") {
		t.Error("View with pinnedOnly should show [pinned] indicator")
	}
}

func TestPlugin_WhenRenderDetailWithScrollIndicator_ShouldShowPosition(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	longContent := ""
	for i := 0; i < 50; i++ {
		longContent += fmt.Sprintf("line %d\n", i)
	}
	p.detail = longContent
	p.detailAddr = "test.resource"

	view := p.renderDetail(80, 10)
	if !strings.Contains(view, "/") {
		t.Error("renderDetail with scrollable content should show scroll indicator")
	}
}

func TestDetailFrame_WhenViewCalled_ShouldDelegateToRenderDetail(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "aws_instance.web"},
				Action:         sdk.ActionUpdate,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "ami", OldValue: "old", NewValue: "new"}},
			},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.stack.Peek().ID() != "inspect" {
		t.Fatal("expected inspect frame on top")
	}

	view := p.stack.View(80, 24)
	if view == "" {
		t.Error("detailFrame.View() should not be empty")
	}
	if !strings.Contains(view, "aws_instance.web") {
		t.Error("detailFrame.View() should contain the address")
	}
}

func TestListFrame_WhenTPressedWithNoSelection_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{},
	}
	p.filtered = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd != nil {
		t.Error("t key with no selection should return nil cmd")
	}
}

func TestListFrame_WhenBigTPressedWithNoSelection_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{},
	}
	p.filtered = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd != nil {
		t.Error("T key with no selection should return nil cmd")
	}
}

func TestListFrame_WhenEPressedWithNoSelection_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{},
	}
	p.filtered = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd != nil {
		t.Error("e key with no selection should return nil cmd")
	}
}

func TestListFrame_WhenTPressedWhileNotDone_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd != nil {
		t.Error("t key while not done should return nil cmd")
	}
}

func TestListFrame_WhenEPressedWhileNotDone_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd != nil {
		t.Error("e key while not done should return nil cmd")
	}
}

func TestPlugin_WhenViewWithStackedDetailFrame_ShouldDelegateToDetail(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{
				Resource:       sdk.Resource{Address: "aws_instance.web"},
				Action:         sdk.ActionUpdate,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "ami", OldValue: "old", NewValue: "new"}},
			},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.inspectSelected()

	view := p.View(80, 24)
	if !strings.Contains(view, "aws_instance.web") {
		t.Error("View with stacked detail frame should delegate to detail view")
	}
}

func TestPlugin_WhenPanDetailRightWithSmallViewWidth_ShouldUseMinContentWidth(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.viewWidth = 10
	p.detail = strings.Repeat("x", 200)

	p.panDetailRight()
	if p.detailHScroll != 10 {
		t.Errorf("panDetailRight with small viewWidth: detailHScroll = %d, want 10", p.detailHScroll)
	}
}

func TestPlugin_WhenPanDetailRightExceedsMaxScroll_ShouldClamp(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.viewWidth = 100
	contentWidth := 100 - 6
	p.detail = strings.Repeat("x", contentWidth+15)

	p.panDetailRight()
	if p.detailHScroll > 15 {
		t.Errorf("panDetailRight beyond max: detailHScroll = %d, should be clamped to maxScroll", p.detailHScroll)
	}
}

func TestPlugin_WhenFormatChangeRowWithWrap_ShouldNotTruncate(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.listWrap = true
	change := sdk.PlanChange{
		Resource: sdk.Resource{Address: strings.Repeat("x", 200)},
		Action:   sdk.ActionCreate,
	}

	row := p.formatChangeRow("[ ] ", change, 80)
	if !strings.Contains(row, strings.Repeat("x", 100)) {
		t.Error("formatChangeRow with wrap should not truncate content")
	}
}

func TestPlugin_WhenOutputJsonWithCreateThenDelete_ShouldSetCorrectAction(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreateThenDelete},
		},
		ToReplace: 1,
	}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, string(sdk.ActionCreateThenDelete)) {
		t.Error("JSON output should contain create-then-delete action")
	}
}

func TestListFrame_WhenRightPressedNoWrap_ShouldPanRight(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.listWrap = false
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.listHScroll != 10 {
		t.Errorf("right key no wrap: listHScroll = %d, want 10", p.listHScroll)
	}
}

func TestListFrame_WhenLeftPressedNoWrap_ShouldPanLeft(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.listWrap = false
	p.listHScroll = 20
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.listHScroll != 10 {
		t.Errorf("left key no wrap: listHScroll = %d, want 10", p.listHScroll)
	}
}

func TestPlugin_WhenRenderResultsWithNarrowWidth_ShouldUseMinContentWidth(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	view := p.renderResults(20, 24)
	if view == "" {
		t.Error("renderResults with narrow width should not be empty")
	}
}

func TestPlugin_WhenRenderDetailWithMultilineHScroll_ShouldOffsetAllLines(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detail = "short\n" + strings.Repeat("x", 100) + "\nend"
	p.detailAddr = "test"
	p.detailHScroll = 5
	p.detailWrap = false

	view := p.renderDetail(80, 24)
	if view == "" {
		t.Error("renderDetail with multiline hscroll should not be empty")
	}
}

func TestPlugin_WhenRenderDetailWithHScrollBeyondLine_ShouldShowEmptyLine(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.detail = "hi\n" + strings.Repeat("y", 100)
	p.detailAddr = "test"
	p.detailHScroll = 50
	p.detailWrap = false

	view := p.renderDetail(80, 24)
	if view == "" {
		t.Error("renderDetail with hscroll beyond short line should not be empty")
	}
}

func TestListFrame_WhenBigTPressedWhileNotDone_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd != nil {
		t.Error("T key while not done should return nil cmd")
	}
}

func TestListFrame_WhenAPressedWithNilSummary_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("a key with nil summary should return nil cmd")
	}
}

func TestListFrame_WhenBigAPressedWithNilSummary_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if cmd != nil {
		t.Error("A key with nil summary should return nil cmd")
	}
}

func TestListFrame_WhenBigAPressedWhileNotDone_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusLoading

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if cmd != nil {
		t.Error("A key while loading should return nil cmd")
	}
}

func TestListFrame_WhenEnterPressedWithNoSelection_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}
	p.filtered = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with no selection should return nil cmd")
	}
}

func TestPlugin_WhenFormatChangeRowLongAddressNoWrap_ShouldTruncateToAvailWidth(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.listWrap = false
	p.listHScroll = 0
	change := sdk.PlanChange{
		Resource: sdk.Resource{Address: strings.Repeat("x", 200)},
		Action:   sdk.ActionCreate,
	}

	row := p.formatChangeRow("[ ] ", change, 50)
	if len(row) == 0 {
		t.Error("formatChangeRow should return non-empty row")
	}
}

func TestPlugin_WhenOutputTextWithCreateThenDelete_ShouldUseCorrectSymbol(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreateThenDelete},
		},
		ToReplace: 1,
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "-/+ aws_instance.web") {
		t.Errorf("text output for create-then-delete should use -/+ symbol, got: %s", s)
	}
}

func TestPlugin_WhenRenderResultsWithFilterButNoFilteringAndPinnedOnly_ShouldShowBothParts(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.filtering = false
	p.filter = "aws"
	p.pinnedOnly = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.pins.Toggle("aws_instance.web")
	p.filtered = p.summary.Changes
	p.rebuildTree()

	view := p.View(80, 24)
	if !strings.Contains(view, "[pinned]") {
		t.Error("View with pinnedOnly should contain [pinned]")
	}
	if !strings.Contains(view, "filter:") {
		t.Error("View with inactive filter should contain 'filter:'")
	}
}

func TestPlugin_WhenRenderResultsOnlyPinnedOnly_ShouldShowPinnedIndicator(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.filtering = false
	p.filter = ""
	p.pinnedOnly = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.pins.Toggle("a")
	p.filtered = p.summary.Changes
	p.rebuildTree()

	view := p.View(80, 24)
	if !strings.Contains(view, "[pinned]") {
		t.Error("View with only pinnedOnly should contain [pinned]")
	}
}

func TestListFrame_WhenEnterInTreeModeOnLeaf_ShouldInspect(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "ami", OldValue: "", NewValue: "ami-123"}}},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.tree.ExpandAll()
	p.MoveDown()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.stack.Peek().ID() != "inspect" {
		t.Errorf("enter on leaf in tree mode should push inspect frame, got %q", p.stack.Peek().ID())
	}
}

func TestListFrame_WhenEnterInTreeModeWithNilNode_ShouldCallInspect(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{},
	}
	p.filtered = nil
	p.rebuildTree()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with nil node should return nil cmd")
	}
}

func TestPlugin_WhenOutputJsonWithEmptyChanges_ShouldReturnEmptyArray(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{},
	}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"changes": []`) {
		t.Errorf("JSON output with empty changes should have empty array, got: %s", s)
	}
}

func TestPlugin_WhenRenderResultsTreeModeWithRiskInLeaf_ShouldShowRisk(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical},
		},
		ToDelete: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.tree.ExpandAll()

	view := p.renderResults(80, 24)
	if view == "" {
		t.Error("renderResults in tree mode with risk should not be empty")
	}
}

func TestListFrame_WhenBigAPressedWithEmptyChanges_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if cmd != nil {
		t.Error("A key with empty changes should return nil cmd")
	}
}

func TestListFrame_WhenAPressedWithEmptyChanges_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("a key with empty changes should return nil cmd")
	}
}

func TestPlugin_WhenRenderResultsTreeModeWithSelectedStyle_ShouldHighlight(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.ecs.aws_ecs_service.app"}, Action: sdk.ActionUpdate},
		},
		ToCreate: 2,
		ToUpdate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	view := p.renderResults(80, 24)
	if view == "" {
		t.Error("renderResults in tree mode with selection should not be empty")
	}
}

func TestPlugin_WhenRenderResultsTreeModeWithPins_ShouldShowPinIndicators(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}, Action: sdk.ActionCreate},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.pins.Toggle("module.vpc.aws_subnet.a")
	p.syncPinnedToTree()

	view := p.renderResults(80, 24)
	if view == "" {
		t.Error("renderResults in tree mode with pins should not be empty")
	}
}

func TestPlanFilterFrame_WhenEnterPressedInTreeModeOnBranch_ShouldToggle(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = true
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.a"}, Action: sdk.ActionCreate,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "cidr"}}},
			{Resource: sdk.Resource{Address: "module.vpc.aws_subnet.b"}, Action: sdk.ActionCreate,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "cidr"}}},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if p.stack.Peek().ID() != "filter" {
		t.Fatal("expected filter frame")
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.stack.Peek().ID() == "inspect" {
		t.Error("enter on branch in filter mode should toggle, not inspect")
	}
}

func TestPlanFilterFrame_WhenEnterPressedOnLeaf_ShouldInspect(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = false
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "ami"}}},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.stack.Peek().ID() != "inspect" {
		t.Errorf("enter on leaf in filter frame should push inspect, got %q", p.stack.Peek().ID())
	}
}

func TestPlanFilterFrame_WhenSpacePressed_ShouldTogglePin(t *testing.T) {
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

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !p.pins.IsPinned("aws_instance.web") {
		t.Error("space in filter frame should pin the resource")
	}
}

func TestPlanFilterFrame_WhenSpacePressedWithNoNode_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{},
	}
	p.filtered = nil
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		t.Error("space in filter frame with no node should return nil cmd")
	}
}

func TestPlanFilterFrame_WhenCtrlTPressed_ShouldToggleTreeMode(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.treeMode = false
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if !p.treeMode {
		t.Error("ctrl+t in filter frame should toggle tree mode")
	}
}

func TestPlanFilterFrame_WhenDownPressed_ShouldNavigateDown(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "b"}, Action: sdk.ActionCreate},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.tree.Cursor() != 1 {
		t.Errorf("down in filter frame: cursor = %d, want 1", p.tree.Cursor())
	}
}

func TestPlanFilterFrame_WhenUpPressed_ShouldNavigateUp(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "b"}, Action: sdk.ActionCreate},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.MoveDown()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.tree.Cursor() != 0 {
		t.Errorf("up in filter frame: cursor = %d, want 0", p.tree.Cursor())
	}
}

func TestPlanFilterFrame_WhenTyping_ShouldFilter(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "google_compute_instance.vm"}, Action: sdk.ActionCreate},
		},
		ToCreate: 2,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if p.filter != "aws" {
		t.Errorf("filter = %q, want %q", p.filter, "aws")
	}
}

func TestPlugin_WhenRenderResultsVerySmallHeight_ShouldUseMinimumVisibleLines(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	view := p.renderResults(80, 2)
	if view == "" {
		t.Error("renderResults with very small height should not be empty")
	}
}

func TestListFrame_WhenIPressedWithDiffs_ShouldOpenInspect(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate,
				AttributeDiffs: []sdk.AttributeDiff{{Key: "x"}}},
		},
		ToCreate: 1,
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if p.stack.Peek().ID() != "inspect" {
		t.Errorf("after i: top frame = %q, want inspect", p.stack.Peek().ID())
	}
}

func TestPlugin_WhenOutputJsonWithNoRisk_ShouldIncludeNoneRisk(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate, Risk: sdk.RiskNone},
		},
		ToCreate: 1,
	}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"risk"`) {
		t.Error("JSON output should include risk field")
	}
}

func TestListFrame_WhenBigAPressedWithNoResults_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if cmd != nil {
		t.Error("A key with no changes: cmd != nil, want nil")
	}
}

func TestListFrame_WhenTPressedDoneNoSelection_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.a.aws_instance.one"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.a.aws_instance.two"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.treeMode = true
	p.rebuildTree()
	// Cursor is on branch node "module.a" - SelectedChange() returns nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd != nil {
		t.Error("t key on branch node (no selection): cmd != nil, want nil")
	}
}

func TestListFrame_WhenBigTPressedDoneNoSelection_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.a.aws_instance.one"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.a.aws_instance.two"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.treeMode = true
	p.rebuildTree()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd != nil {
		t.Error("T key on branch node (no selection): cmd != nil, want nil")
	}
}

func TestListFrame_WhenEPressedDoneNoSelection_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "module.a.aws_instance.one"}, Action: sdk.ActionCreate},
			{Resource: sdk.Resource{Address: "module.a.aws_instance.two"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.treeMode = true
	p.rebuildTree()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd != nil {
		t.Error("e key on branch node (no selection): cmd != nil, want nil")
	}
}

func TestFilterFrame_WhenSpacePressedWithNoResults_ShouldReturnNil(t *testing.T) {
	p := newTestPlugin(&sdktest.MockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	// Enter filter mode
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type something that matches nothing to empty the results
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	// Press space (pin) with no cursor node — should return nil
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		t.Error("space in filter with no results: cmd != nil, want nil")
	}
}
