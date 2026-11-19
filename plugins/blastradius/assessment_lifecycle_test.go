package blastradius

import (
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestAssessment_GivenCancelledOrInvalidatedOperation_ShouldRejectLateResult(t *testing.T) {
	for _, tc := range []string{"cancel", "invalidate", "context", "new plan", "shutdown"} {
		t.Run(tc, func(t *testing.T) {
			high := sdk.RiskHigh
			summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.old"}, Action: sdk.ActionUpdate}}}
			p, h, svc := newBlastHarness(t, summary, sdk.PolicyAssessment{Status: "complete", Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.old", Status: "classified", Level: &high}}})
			p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary})
			cmd := p.Activate()
			if _, busy := any(p).(sdk.Busy); busy {
				t.Fatal("Blast Radius implements sdk.Busy; a policy-only assessment runs no Terraform")
			}
			if p.Status() != sdk.StatusLoading || p.Ready() || !strings.Contains(p.View(80, 20), "Loading risk assessment") {
				t.Fatalf("activation should show assessment loading and pending: status=%v ready=%v", p.Status(), p.Ready())
			}
			if p.Activate() != nil {
				t.Fatal("activation while loading should not launch duplicate assessment")
			}
			switch tc {
			case "cancel":
				p.Cancel()
			case "invalidate":
				p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
			case "context":
				h.Ctx = &sdk.Context{WorkingDir: "/different", Service: svc}
				p.HandleContextChanged(sdk.ContextChangedEvent{Next: h.Ctx})
			case "new plan":
				p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: &sdk.PlanSummary{}})
			case "shutdown":
				p.Shutdown()
			}
			runBlastCmd(p, cmd)
			if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != summary {
				t.Fatalf("assessment requests = %v; want one request for the reviewed summary %p", h.AssessmentRequests, summary)
			}
			if strings.Contains(p.View(80, 20), "aws_instance.old") {
				t.Fatalf("late result restored obsolete assessment: status=%v view=%q", p.Status(), p.View(80, 20))
			}
			if tc == "cancel" || tc == "shutdown" {
				if !strings.Contains(p.View(80, 20), "cancelled") {
					t.Fatalf("cancellation should be visible: %q", p.View(80, 20))
				}
			}
		})
	}
}

func TestActivate_GivenNoReviewedPlan_ShouldWaitForPlan(t *testing.T) {
	svc := &sdktest.MockService{}
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	if p.Activate() != nil || !strings.Contains(p.View(80, 20), "Run a plan first") {
		t.Fatal("without reviewed plan Blast Radius should remain idle")
	}
	p.HandleContextChanged(sdk.ContextChangedEvent{})
	if p.Status() != sdk.StatusIdle {
		t.Fatalf("empty context event changed status=%v", p.Status())
	}
}

func TestActivate_GivenRawPlanRisk_ShouldRenderHostClassificationInstead(t *testing.T) {
	low := sdk.RiskLow
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.legacy"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh}}}
	p := activateAssessmentBlast(t, summary, sdk.PolicyAssessment{Status: "complete", OverallLevel: &low, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.legacy", Status: "classified", Level: &low},
	}})
	if view := p.View(100, 20); !p.Ready() || !strings.Contains(view, "Minimal blast radius") || strings.Contains(view, "High blast radius") {
		t.Fatalf("Blast Radius = %q; want the host's low classification, not the plan's raw high risk", view)
	}
}

func TestActivate_GivenExcludedChanges_ShouldKeepActionCountImpactAndNotAssessedLabel(t *testing.T) {
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "module.z.data.a", Module: "module.z"}, Action: sdk.ActionRead, Risk: sdk.RiskCritical},
		{Resource: sdk.Resource{Address: "module.a.data.a", Module: "module.a"}, Action: sdk.ActionRead, Risk: sdk.RiskCritical},
		{Resource: sdk.Resource{Address: "module.a.data.b", Module: "module.a"}, Action: sdk.ActionRead, Risk: sdk.RiskCritical},
		{Resource: sdk.Resource{Address: "module.a.data.c", Module: "module.a"}, Action: sdk.ActionRead, Risk: sdk.RiskCritical},
		{Resource: sdk.Resource{Address: "module.b.data.a", Module: "module.b"}, Action: sdk.ActionRead, Risk: sdk.RiskCritical},
	}}
	assessment := sdk.PolicyAssessment{Status: "empty"}
	for _, change := range summary.Changes {
		assessment.Resources = append(assessment.Resources, sdk.PolicyResourceAssessment{Address: change.Resource.Address, Status: "not_assessed", Exclusion: "read"})
	}
	p := activateAssessmentBlast(t, summary, assessment)
	view := p.View(140, 40)
	if !strings.Contains(view, "Moderate blast radius") || !strings.Contains(view, "5 total change(s)") || strings.Contains(view, "CRITICAL") {
		t.Fatalf("excluded changes impact=%q; want count-only impact without synthetic risk", view)
	}
	if strings.Index(view, "module.b") > strings.Index(view, "module.z") {
		t.Fatalf("equal impact groups should sort by address: %q", view)
	}
	p.ToggleExpand()
	view = p.View(140, 40)
	if !strings.Contains(view, "Risk: not assessed") || strings.Contains(view, "unavailable") {
		t.Fatalf("excluded changes=%q; want distinct not assessed label", view)
	}
}

func TestActivate_GivenEmptyPlanAndGlobalFailure_ShouldShowUnavailableRatherThanZero(t *testing.T) {
	p := activateAssessmentBlast(t, &sdk.PlanSummary{}, sdk.PolicyAssessment{Status: "unavailable", ErrorCode: "compilation_failed"})
	if view := p.View(100, 20); !strings.Contains(view, "Blast radius unavailable") || strings.Contains(view, "zero") {
		t.Fatalf("empty failed assessment=%q; want unavailable", view)
	}
	p = activateAssessmentBlast(t, &sdk.PlanSummary{}, sdk.PolicyAssessment{Status: "empty"})
	if view := p.View(100, 20); !strings.Contains(view, "Blast radius is zero") {
		t.Fatalf("valid empty assessment=%q", view)
	}
}

func TestActivate_GivenMissingClassification_ShouldMarkGroupUnavailable(t *testing.T) {
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.omitted"}, Action: sdk.ActionUpdate}}}
	p := activateAssessmentBlast(t, summary, sdk.PolicyAssessment{Status: "partial"})
	p.ToggleExpand()
	if view := p.View(100, 25); !strings.Contains(view, "[unavailable]") || !strings.Contains(view, "input_incomplete") {
		t.Fatalf("missing classification=%q; want explicit incomplete input", view)
	}
}
