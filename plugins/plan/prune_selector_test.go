package plan

import (
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

// A -target pin can be a terraform selector (a module path, or a resource
// without its instance key) rather than one changed address.
func TestPlugin_WhenPruneStalePinsWithSelectorPins_ShouldKeepThoseThatCoverAChange(t *testing.T) {
	changes := []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "module.networking.aws_vpc.main"}},
		{Resource: sdk.Resource{Address: `module.app["blue"].aws_instance.api`}},
		{Resource: sdk.Resource{Address: "aws_instance.web[0]"}},
		{Resource: sdk.Resource{Address: "aws_instance.webserver"}},
	}
	tests := []struct {
		name      string
		pin       string
		wantStale bool
	}{
		{name: "module path", pin: "module.networking"},
		{name: "module path over keyed instances", pin: "module.app"},
		{name: "resource over its indexed instances", pin: "aws_instance.web"},
		{name: "exact address", pin: "aws_instance.webserver"},
		{name: "prefix without a boundary", pin: "module.net", wantStale: true},
		{name: "resource name prefix", pin: "aws_instance.webs", wantStale: true},
		{name: "no change under it", pin: "module.storage", wantStale: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, h := newTestPluginWithHarness(&sdktest.MockService{})
			h.Ctx.Pins = []string{tt.pin}
			cmd := p.pruneStalePins(changes)
			if cmd != nil {
				cmd()
			}
			if got := slices.Contains(h.PinRequests, tt.pin); got != tt.wantStale {
				t.Errorf("pin %q pruned = %v, want %v", tt.pin, got, tt.wantStale)
			}
		})
	}
}

// Pruning a module pin marks the plan stale, so `A` would replan untargeted
// and auto-apply the wider plan.
func TestPlugin_WhenModulePinPlanCompletes_ShouldAutoApplyThatPlanWithoutReplanning(t *testing.T) {
	p, h := newTestPluginWithHarness(&sdktest.MockService{})
	h.Ctx.Pins = []string{"module.networking"}
	p.status = sdk.StatusLoading
	p.planFile = sdk.NewTempPlanFile("/tmp/tfui-module-pin.tfplan")

	_, cmd := p.Update(PlanResultMsg{Summary: &sdk.PlanSummary{
		Changes:  []sdk.PlanChange{{Resource: sdk.Resource{Address: "module.networking.aws_vpc.main"}, Action: sdk.ActionCreate}},
		ToCreate: 1,
	}})
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, sub := range batch {
			if sub != nil {
				_ = sub()
			}
		}
	}
	if len(h.PinRequests) != 0 {
		t.Fatalf("plan completion toggled pins %q, want the module pin kept", h.PinRequests)
	}

	req, ok := p.requestAutoApply()().(ApplyRequestMsg)
	if !ok || req.PlanFile != "/tmp/tfui-module-pin.tfplan" || !req.AutoApprove {
		t.Errorf("A emitted %#v, want an auto-approved apply of the reviewed plan file", req)
	}
	if p.pendingApply {
		t.Error("A queued a replan, want the reviewed plan applied")
	}
}
