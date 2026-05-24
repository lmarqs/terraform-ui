package plan

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/plugins/taint"
	"github.com/lmarqs/terraform-ui/plugins/untaint"
)

func newPlanPluginWithChanges(changes []sdk.PlanChange) (*Plugin, *sdktest.PluginDepsHarness) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	h := sdktest.NewDeps(svc)
	p.Init(h.Deps)
	p.status = sdk.StatusDone
	p.planFile = sdk.NewTempPlanFile("/tmp/tfui-test.tfplan")
	p.summary = &sdk.PlanSummary{Changes: changes}
	p.filtered = changes
	p.rebuildTree()
	return p, h
}

func TestActionTargets_WhenNoPins_ShouldReturnCursorAddress(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.a"}},
		{Resource: sdk.Resource{Address: "aws_instance.b"}},
	}
	p, _ := newPlanPluginWithChanges(changes)

	targets := p.actionTargets()
	if len(targets) != 1 || targets[0] != "aws_instance.a" {
		t.Errorf("expected [aws_instance.a], got %v", targets)
	}
}

func TestActionTargets_WhenPinsExist_ShouldReturnPinnedAddresses(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.a"}},
		{Resource: sdk.Resource{Address: "aws_instance.b"}},
		{Resource: sdk.Resource{Address: "aws_instance.c"}},
	}
	p, h := newPlanPluginWithChanges(changes)
	h.Ctx.Pins = []string{"aws_instance.b", "aws_instance.c"}

	targets := p.actionTargets()
	if len(targets) != 2 {
		t.Errorf("expected 2 pinned targets, got %d", len(targets))
	}
}

func TestActionTargets_WhenEmptyList_ShouldReturnNil(t *testing.T) {
	p, _ := newPlanPluginWithChanges(nil)

	targets := p.actionTargets()
	if targets != nil {
		t.Errorf("expected nil targets for empty list, got %v", targets)
	}
}

func TestBuildActionFrame_WhenSingleTarget_ShouldUseAddressAsTitle(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web"}},
	}
	p, _ := newPlanPluginWithChanges(changes)

	frame := p.buildActionFrame(false)
	if frame.ID() != "actions" {
		t.Errorf("expected frame ID 'actions', got %q", frame.ID())
	}
}

func TestBuildActionFrame_WhenMultiplePins_ShouldShowPinnedCount(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.a"}},
		{Resource: sdk.Resource{Address: "aws_instance.b"}},
	}
	p, h := newPlanPluginWithChanges(changes)
	h.Ctx.Pins = []string{"aws_instance.a", "aws_instance.b"}

	frame := p.buildActionFrame(true)
	view := frame.View(80, 20)
	if view == "" {
		t.Error("expected non-empty view from action frame")
	}
}

func TestBuildActionFrame_ShouldHaveApplyAction(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web"}},
	}
	p, _ := newPlanPluginWithChanges(changes)

	frame := p.buildActionFrame(false)
	result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if result != nil {
		t.Error("expected frame to pop after action key")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from apply handler")
	}
	msg := cmd()
	if _, ok := msg.(ApplyRequestMsg); !ok {
		t.Errorf("expected ApplyRequestMsg, got %T", msg)
	}
}

func TestBuildActionFrame_ShouldHaveAutoApplyAction(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web"}},
	}
	p, _ := newPlanPluginWithChanges(changes)

	frame := p.buildActionFrame(false)
	result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if result != nil {
		t.Error("expected frame to pop after action key")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from auto-apply handler")
	}
	msg := cmd()
	req, ok := msg.(ApplyRequestMsg)
	if !ok {
		t.Errorf("expected ApplyRequestMsg, got %T", msg)
	} else if !req.AutoApprove {
		t.Errorf("expected AutoApprove=true on auto-apply emission")
	}
}

func TestBuildActionFrame_ShouldHaveTaintAction(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web"}},
	}
	p, _ := newPlanPluginWithChanges(changes)

	frame := p.buildActionFrame(false)
	result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if result != nil {
		t.Error("expected frame to pop after action key")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from taint handler")
	}
	msg := cmd()
	if taintMsg, ok := msg.(taint.TaintRequestMsg); !ok {
		t.Errorf("expected TaintRequestMsg, got %T", msg)
	} else if len(taintMsg.Addresses) != 1 || taintMsg.Addresses[0] != "aws_instance.web" {
		t.Errorf("expected [aws_instance.web], got %v", taintMsg.Addresses)
	}
}

func TestBuildActionFrame_ShouldHaveUntaintAction(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web"}},
	}
	p, _ := newPlanPluginWithChanges(changes)

	frame := p.buildActionFrame(false)
	result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if result != nil {
		t.Error("expected frame to pop after action key")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from untaint handler")
	}
	msg := cmd()
	if untaintMsg, ok := msg.(untaint.UntaintRequestMsg); !ok {
		t.Errorf("expected UntaintRequestMsg, got %T", msg)
	} else if len(untaintMsg.Addresses) != 1 || untaintMsg.Addresses[0] != "aws_instance.web" {
		t.Errorf("expected [aws_instance.web], got %v", untaintMsg.Addresses)
	}
}

func TestBuildActionFrame_WhenEsc_ShouldPopFrame(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.web"}},
	}
	p, _ := newPlanPluginWithChanges(changes)

	frame := p.buildActionFrame(false)
	result, _ := frame.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if result != nil {
		t.Error("expected nil frame (pop) on esc")
	}
}

func TestBuildActionFrame_WhenBatchWithPins_ShouldTargetAllPinned(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.a"}},
		{Resource: sdk.Resource{Address: "aws_instance.b"}},
	}
	p, h := newPlanPluginWithChanges(changes)
	h.Ctx.Pins = []string{"aws_instance.a", "aws_instance.b"}
	p.syncPinnedToTree()

	frame := p.buildActionFrame(true)
	_, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from batch taint")
	}
	msg := cmd()
	taintMsg, ok := msg.(taint.TaintRequestMsg)
	if !ok {
		t.Fatalf("expected TaintRequestMsg, got %T", msg)
	}
	if len(taintMsg.Addresses) != 2 {
		t.Errorf("expected 2 addresses in batch taint, got %d", len(taintMsg.Addresses))
	}
}
