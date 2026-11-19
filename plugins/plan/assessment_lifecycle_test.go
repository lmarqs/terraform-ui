package plan

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestHandleRiskAssessed_GivenEventForAnotherPlan_ShouldNotClassifyCurrentReview(t *testing.T) {
	critical := sdk.RiskCritical
	for _, tc := range []struct {
		name string
		// arrange leaves the plugin with a review that the stale event does not belong to.
		arrange func(t *testing.T, p *Plugin, h *sdktest.PluginDepsHarness)
		settled func(p *Plugin) bool
		opts    []func(*sdktest.PluginDepsHarness)
	}{
		{"pending own request", func(t *testing.T, p *Plugin, _ *sdktest.PluginDepsHarness) {
			drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
		}, func(p *Plugin) bool { return p.Busy() && !p.Ready() }, []func(*sdktest.PluginDepsHarness){withholdAssessment(new(sdk.AssessmentID))}},
		{"context switch", func(t *testing.T, p *Plugin, h *sdktest.PluginDepsHarness) {
			drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
			h.Ctx = &sdk.Context{WorkingDir: "/different", Service: h.Deps.Service}
			drainAssessmentPlan(t, p, func() tea.Msg { return sdk.ContextChangedEvent{Next: h.Ctx} }, nil)
		}, func(p *Plugin) bool { return p.Status() == sdk.StatusIdle }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := &sdk.PlanSummary{ToDelete: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.reviewed"}, Action: sdk.ActionDelete}}}
			stale := sdk.RiskAssessedEvent{Summary: &sdk.PlanSummary{}, Assessment: sdk.PolicyAssessment{Status: "complete", OverallLevel: &critical, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &critical}}}}
			p, _, h := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: "complete"}, tc.opts...)
			tc.arrange(t, p, h)
			drainAssessmentPlan(t, p, func() tea.Msg { return stale }, nil)
			if view := p.View(100, 30); strings.Contains(view, "CRITICAL") {
				t.Fatalf("an assessment of another plan classified the current review: %q", view)
			}
			if !tc.settled(p) {
				t.Fatalf("status=%v busy=%v ready=%v; the event of another plan changed the review state", p.Status(), p.Busy(), p.Ready())
			}
		})
	}
}

func TestActivate_GivenRecoveredPolicy_ShouldClearFailureAndUpdateOpenInspection(t *testing.T) {
	high := sdk.RiskHigh
	summary := &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.reviewed"}, Action: sdk.ActionUpdate}}}
	p, svc, h := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: "unavailable", ErrorCode: "compilation_failed"})
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	p.inspectSelected()
	if view := p.View(120, 35); !strings.Contains(view, "compilation_failed") {
		t.Fatalf("initial inspection=%q; want compilation failure", view)
	}
	h.AssessmentEvent.Assessment = sdk.PolicyAssessment{Status: "complete", OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high, Reasons: []sdk.PolicyReason{{Code: "recovered", Message: "Current policy evidence"}}}}}
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	view := p.View(120, 35)
	if !strings.Contains(view, "HIGH") || !strings.Contains(view, "Current policy evidence") || strings.Contains(view, "unavailable") || len(p.Stderr()) != 0 {
		t.Fatalf("recovered inspection=%q stderr=%q; want current classification with cleared failure", view, p.Stderr())
	}
	if len(svc.PlanCalls) != 1 {
		t.Fatalf("source recovery planned %d times; want retained plan evidence", len(svc.PlanCalls))
	}
}

func TestActivate_GivenMissingAssessmentRow_ShouldShowInputIncompleteOnInspection(t *testing.T) {
	summary := &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.omitted"}, Action: sdk.ActionUpdate, Risk: sdk.RiskCritical}}}
	p, _, _ := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: "partial"})
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	p.inspectSelected()
	if view := p.View(100, 30); !strings.Contains(view, "input_incomplete") || strings.Contains(view, "CRITICAL") {
		t.Fatalf("omitted resource inspection=%q; want unavailable input", view)
	}
}

func TestCancel_GivenRunningPlan_ShouldTerminateAndRejectLateSuccess(t *testing.T) {
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.cancelled"}, Action: sdk.ActionUpdate}}}, nil
	}}
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	pending := p.Activate(Input{})
	p.Cancel()
	if p.Busy() {
		t.Fatal("cancelled Plan remains busy; headless session cannot terminate")
	}
	drainAssessmentPlan(t, p, pending, nil)
	if view := p.View(100, 30); strings.Contains(view, "aws_instance.cancelled") {
		t.Fatalf("cancelled Plan accepted late success: %q", view)
	}
	if p.Status() != sdk.StatusError || p.ExitCode() != 1 {
		t.Fatalf("cancelled Plan status=%v exit=%d; want terminal failure", p.Status(), p.ExitCode())
	}
}

func TestActivate_GivenCachedPlanReassessment_ShouldHideOldRiskUntilCurrentResult(t *testing.T) {
	high := sdk.RiskHigh
	summary := &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.reviewed"}, Action: sdk.ActionUpdate}}}
	p, _, _ := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: "complete", OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high}}})
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	if view := p.View(100, 30); !strings.Contains(view, "HIGH") {
		t.Fatalf("initial review=%q; want HIGH from the host assessment", view)
	}
	pending := p.Activate(Input{})
	if !p.Busy() || p.Ready() {
		t.Fatalf("reassessment status=%v ready=%v; must wait for current assessment", p.Status(), p.Ready())
	}
	if view := p.View(100, 30); strings.Contains(view, "HIGH") {
		t.Fatalf("reassessment exposes obsolete high classification: %q", view)
	}
	drainAssessmentPlan(t, p, pending, nil)
	if !p.Ready() || p.Busy() {
		t.Fatalf("completed reassessment status=%v; want ready", p.Status())
	}
}

func TestActivate_GivenPartialAssessment_ShouldReportFirstFailureByAddress(t *testing.T) {
	summary := &sdk.PlanSummary{ToUpdate: 2, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.z"}, Action: sdk.ActionUpdate}, {Resource: sdk.Resource{Address: "aws_instance.a"}, Action: sdk.ActionUpdate}}}
	p, _, _ := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: "partial", Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.z", Status: "unavailable", ErrorCode: "evaluation_failed"}, {Address: "aws_instance.a", Status: "unavailable", ErrorCode: "result_invalid"}}})
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	if got := string(p.Stderr()); got != "risk unavailable: result_invalid\n" {
		t.Fatalf("stderr=%q; want fixed diagnostic of first failed address", got)
	}
}

func TestActivate_GivenPinsBeforeFirstPlan_ShouldAcquireReviewedPlan(t *testing.T) {
	high := sdk.RiskHigh
	summary := &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.reviewed"}, Action: sdk.ActionUpdate}}}
	p, svc, h := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: "complete", OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high}}})
	p.HandleContextChanged(sdk.ContextChangedEvent{Next: h.Ctx, Reason: sdk.ContextPinsChanged})
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	if len(svc.PlanCalls) != 1 || !strings.Contains(p.View(100, 30), "aws_instance.reviewed") {
		t.Fatalf("initial selected plan calls=%d view=%q; want acquisition before assessment", len(svc.PlanCalls), p.View(100, 30))
	}
}

func TestStreamCancel_GivenRunningPlan_ShouldLeaveLoadingWithTheRunError(t *testing.T) {
	svc := &sdktest.MockService{PlanFn: func(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return nil, ctx.Err()
	}}
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	pending := p.Activate(Input{})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	drainAssessmentPlan(t, p, pending, nil)
	if p.Busy() || p.Status() != sdk.StatusError || p.ExitCode() != 1 || !strings.Contains(p.View(100, 30), context.Canceled.Error()) {
		t.Fatalf("plan cancelled from its stream frame: status=%v busy=%v view=%q; want the run's error", p.Status(), p.Busy(), p.View(100, 30))
	}
}
