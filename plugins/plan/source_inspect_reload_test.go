package plan

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestActivate_GivenInspectedPlanAndDeletedPolicySource_ShouldReplaceCachedRiskBadge(t *testing.T) {
	critical := sdk.RiskCritical
	summary := &sdk.PlanSummary{ToDelete: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete, Risk: sdk.RiskLow}}}
	classified := sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &critical, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_db_instance.database", Status: sdk.ResourceClassified, Level: &critical}}}
	p, svc, h := newAssessedPlugin(t, summary, classified)
	deliverPlanResult(t, p, summary)
	drainAssessmentPlan(t, p, p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter}), nil)
	if p.stack.Peek().ID() != "inspect" {
		t.Fatalf("top frame = %q, want inspect", p.stack.Peek().ID())
	}
	if view := p.View(100, 30); !strings.Contains(view, "CRITICAL") {
		t.Fatalf("initial inspection = %q, want the host's CRITICAL badge", view)
	}

	h.AssessmentEvent.Assessment = sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorSourceUnreadable}
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)

	view := p.View(100, 30)
	if !strings.Contains(view, "risk unavailable: source_unreadable") || strings.Contains(view, "CRITICAL") {
		t.Fatalf("reactivated inspection = %q, want unavailable diagnostic without previous risk badge", view)
	}
	h.AssessmentEvent.Assessment = classified
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	recovered := p.View(100, 30)
	if !strings.Contains(recovered, "CRITICAL") || strings.Contains(recovered, "risk unavailable:") {
		t.Fatalf("inspection after source recovery = %q, want current risk badge", recovered)
	}
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("cached Plan reactivation acquired %d plans", len(svc.PlanCalls))
	}
}
