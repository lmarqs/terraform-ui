package risk

import (
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestActivate_GivenSelectedSourceFailure_ShouldTerminateWithoutFallbackClassification(t *testing.T) {
	critical := sdk.RiskCritical
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete, Risk: sdk.RiskLow}}}
	classified := sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &critical, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_db_instance.database", Actions: []string{"delete"}, Status: sdk.ResourceClassified, Level: &critical}}}
	p, _, h := newAssessedPlugin(t, summary, classified)
	runLifecycleCmd(p, p.Activate(Input{}))
	if view := p.View(100, 30); !strings.Contains(view, "aws_db_instance.database") || !strings.Contains(view, "CRITICAL") {
		t.Fatalf("initial classification missing: %q", view)
	}
	h.AssessmentEvent.Assessment = sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorSourceUnreadable}
	runLifecycleCmd(p, p.Activate(Input{}))
	if p.Status() != sdk.StatusError {
		t.Fatalf("source failure status = %v; want terminal error", p.Status())
	}
	view := p.View(100, 30)
	if !strings.Contains(view, "risk unavailable: source_unreadable") {
		t.Fatalf("view = %q, want fixed source failure", view)
	}
	if strings.Contains(view, "CRITICAL") || len(p.groups) != 0 {
		t.Fatalf("source failure retained prior successful classification: %q", view)
	}
	h.AssessmentEvent.Assessment = classified
	runLifecycleCmd(p, p.Activate(Input{}))
	if view := p.View(100, 30); p.Status() != sdk.StatusDone || !strings.Contains(view, "aws_db_instance.database") || !strings.Contains(view, "CRITICAL") {
		t.Fatalf("restored source did not recover on next assessment: %q", view)
	}
}

func TestActivate_GivenSourceDeletedAfterAnalysis_ShouldReplacePreviousClassificationWithUnavailable(t *testing.T) {
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate, Risk: sdk.RiskLow}}}
	p, svc, h := newAssessedPlugin(t, summary, classifiedAs("aws_instance.web", sdk.RiskLow, "Web classification."))
	runLifecycleCmd(p, p.Activate(Input{}))
	if view := p.View(100, 30); !strings.Contains(view, "Web classification.") {
		t.Fatalf("initial classification missing: %q", view)
	}
	h.AssessmentEvent.Assessment = sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorSourceUnreadable}
	runLifecycleCmd(p, p.Activate(Input{}))
	if p.Status() != sdk.StatusError {
		t.Fatalf("activation status = %v; want terminal error", p.Status())
	}
	if got := p.View(100, 30); !strings.Contains(got, "risk unavailable: source_unreadable") || strings.Contains(got, "Low risk") || strings.Contains(got, "aws_instance.web") {
		t.Fatalf("reactivated Risk view = %q", got)
	}
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Risk ran Service.Plan %d times; planning belongs to the host", len(svc.PlanCalls))
	}
}
