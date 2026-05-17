package plan

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestPlugin_WhenOutputJsonWithNilSummary_ShouldReturnNil(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
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
	p := New(&mockService{}).(*Plugin)
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
	p := New(&mockService{}).(*Plugin)
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
	p := New(&mockService{}).(*Plugin)
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
	p := New(&mockService{}).(*Plugin)
	p.summary = nil
	if code := p.ExitCode(); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
}

func TestPlugin_WhenExitCodeWithChanges_ShouldReturnTwo(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate}},
	}
	if code := p.ExitCode(); code != 2 {
		t.Errorf("ExitCode() = %d, want 2", code)
	}
}

func TestPlugin_WhenExitCodeWithEmptyChanges_ShouldReturnZero(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}
	if code := p.ExitCode(); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
}

func TestPlugin_WhenHandleLockCleared_ShouldClearLockInfo(t *testing.T) {
	p := newTestPlugin(&mockService{})
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
	p := New(&mockService{}).(*Plugin)
	p.pins = nil
	if c := p.PinnedCount(); c != 0 {
		t.Errorf("PinnedCount() = %d, want 0", c)
	}
}

func TestPlugin_WhenPinnedCountWithPins_ShouldReturnCount(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.pins.Toggle("a")
	p.pins.Toggle("b")
	if c := p.PinnedCount(); c != 2 {
		t.Errorf("PinnedCount() = %d, want 2", c)
	}
}

func TestPlugin_WhenRequestAutoApply_ShouldEmitAutoApplyRequestMsg(t *testing.T) {
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
}

func TestListFrame_WhenBigTPressedDone_ShouldEmitUntaintRequest(t *testing.T) {
	p := newTestPlugin(&mockService{})
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
}

func TestListFrame_WhenBigAPressed_ShouldRequestAutoApply(t *testing.T) {
	p := newTestPlugin(&mockService{})
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
	p := newTestPlugin(&mockService{})
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
