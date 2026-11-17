package plan

import (
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

// deliverPlanResult feeds a finished plan to the plugin as the plan run does and
// runs everything it asks for, including the host assessment.
func deliverPlanResult(t *testing.T, p *Plugin, summary *sdk.PlanSummary) {
	t.Helper()
	_, cmd := p.Update(PlanResultMsg{Summary: summary})
	drainAssessmentPlan(t, p, cmd, nil)
}

func TestPolicyFailure_GivenReviewNavigation_ShouldRemainVisibleAndSuppressClassifications(t *testing.T) {
	for _, mode := range []string{"empty", "filtered", "tree", "flat", "inspect"} {
		t.Run(mode, func(t *testing.T) {
			summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}
			if mode == "empty" {
				summary.Changes = nil
			}
			p, _, _ := newAssessedPlugin(t, summary, sourceInvalid())
			deliverPlanResult(t, p, summary)
			switch mode {
			case "filtered":
				p.SetFilter("no-match")
			case "tree":
				p.treeMode = true
				p.SetFilter("")
			case "flat":
				p.treeMode = false
				p.SetFilter("")
			case "inspect":
				p.inspectSelected()
			}
			view := sdktest.StripANSI(p.View(100, 30))
			if !strings.Contains(view, "risk unavailable: source_invalid") || strings.Contains(view, "CRITICAL") {
				t.Fatalf("review = %q", view)
			}
		})
	}
}

func TestActivate_GivenSourceChangedAfterPlan_ShouldReloadWithoutAcquiringAnotherPlan(t *testing.T) {
	low := sdk.RiskLow
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskCritical}}}
	classified := sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &low, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.web", Status: sdk.ResourceClassified, Level: &low}}}
	p, svc, h := newAssessedPlugin(t, summary, classified)
	deliverPlanResult(t, p, summary)

	h.AssessmentEvent.Assessment = sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorSourceUnreadable}
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	if !strings.Contains(p.View(100, 30), "risk unavailable: source_unreadable") {
		t.Fatalf("activation retained risk from deleted source: %q", p.View(100, 30))
	}

	h.AssessmentEvent.Assessment = classified
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	if view := p.View(100, 30); len(p.Stderr()) != 0 || strings.Contains(view, "risk unavailable") || !strings.Contains(view, "[low]") {
		t.Fatalf("next boundary retained resolved source failure: view=%q stderr=%q", view, p.Stderr())
	}
	if len(svc.PlanCalls) != 0 || len(h.AssessmentRequests) != 3 {
		t.Fatalf("plan calls=%d assessment requests=%d; want every boundary reassessed through the host without planning", len(svc.PlanCalls), len(h.AssessmentRequests))
	}
}
