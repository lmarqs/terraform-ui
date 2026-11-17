package blastradius

import (
	"errors"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func eventSummary() *sdk.PlanSummary {
	return &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "module.db.aws_db_instance.main", Module: "module.db"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical},
	}}
}

// classifiedEvent classifies every change of summary at level with reason, so a
// view showing reason came from the event, not from the plan's raw risk.
func classifiedEvent(summary *sdk.PlanSummary, level sdk.RiskLevel, reason string) sdk.RiskAssessedEvent {
	assessment := sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &level}
	for _, change := range summary.Changes {
		assessment.Resources = append(assessment.Resources, sdk.PolicyResourceAssessment{Address: change.Resource.Address, Status: sdk.ResourceClassified, Level: &level, Reasons: []sdk.PolicyReason{{Code: "authored", Message: reason}}})
	}
	return sdk.RiskAssessedEvent{Summary: summary, Assessment: assessment}
}

func TestActivate_GivenReviewedPlan_ShouldRequestOneHostAssessmentOfThatSummary(t *testing.T) {
	summary := eventSummary()
	p, h, svc := newBlastHarness(t, summary, classifiedEvent(summary, sdk.RiskLow, "Host classification.").Assessment)
	runBlastCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))

	runBlastCmd(p, p.Activate())

	if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != summary {
		t.Fatalf("assessment requests = %v; want exactly one request carrying the reviewed summary %p", h.AssessmentRequests, summary)
	}
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Blast Radius ran terraform plan %d times; want the reviewed summary assessed by the host", len(svc.PlanCalls))
	}
	view := p.View(120, 40)
	if p.Busy() || !p.Ready() || !strings.Contains(view, "Minimal blast radius") || strings.Contains(view, "CRITICAL") {
		t.Fatalf("Busy=%v Ready=%v view=%q; want the host's low classification, not the raw critical risk", p.Busy(), p.Ready(), view)
	}
	p.ToggleExpand()
	if view = p.View(120, 40); !strings.Contains(view, "Host classification.") {
		t.Fatalf("expanded view = %q; want the event's reason", view)
	}
}

func TestHandleRiskAssessed_GivenEventForAnotherSummary_ShouldIgnoreIt(t *testing.T) {
	summary, other := eventSummary(), eventSummary()
	p, h, _ := newBlastHarness(t, summary, classifiedEvent(summary, sdk.RiskLow, "Own classification.").Assessment)
	p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary})
	pending := p.Activate()

	deliverBlast(p, classifiedEvent(other, sdk.RiskCritical, "Other plan classification."))
	if !p.Busy() || p.Ready() || !strings.Contains(p.View(120, 40), "Loading risk assessment") {
		t.Fatalf("another plan's event settled the pending request: Busy=%v Ready=%v view=%q", p.Busy(), p.Ready(), p.View(120, 40))
	}

	runBlastCmd(p, pending)
	if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != summary {
		t.Fatalf("assessment requests = %v; want one request for %p", h.AssessmentRequests, summary)
	}
	if view := p.View(120, 40); !p.Ready() || !strings.Contains(view, "Minimal blast radius") {
		t.Fatalf("own event was not rendered after the stale one: Ready=%v view=%q", p.Ready(), view)
	}

	deliverBlast(p, classifiedEvent(other, sdk.RiskCritical, "Other plan classification."))
	p.ToggleExpand()
	if view := p.View(120, 40); strings.Contains(view, "CRITICAL") || strings.Contains(view, "Other plan classification.") || !strings.Contains(view, "Own classification.") {
		t.Fatalf("another plan's event replaced the result: %q", view)
	}
}

func TestHandleRiskAssessed_GivenFailedOrUnavailableEvent_ShouldShowUnavailableImpact(t *testing.T) {
	for _, tc := range []struct {
		name string
		ev   func(*sdk.PlanSummary) sdk.RiskAssessedEvent
	}{
		{"plan fetch error", func(summary *sdk.PlanSummary) sdk.RiskAssessedEvent {
			return sdk.RiskAssessedEvent{Summary: summary, Err: errors.New("terraform planning failed")}
		}},
		{"unavailable assessment", func(summary *sdk.PlanSummary) sdk.RiskAssessedEvent {
			return sdk.RiskAssessedEvent{Summary: summary, Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorSourceUnreadable}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := eventSummary()
			p, h, _ := newBlastHarness(t, summary, sdk.PolicyAssessment{})
			h.AssessmentEvent = tc.ev(summary)
			runBlastCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))

			runBlastCmd(p, p.Activate())

			if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != summary {
				t.Fatalf("assessment requests = %v; want one request for %p", h.AssessmentRequests, summary)
			}
			view := p.View(120, 40)
			if p.Busy() || !p.Ready() || !strings.Contains(view, "Blast radius unavailable: incomplete risk assessment") {
				t.Fatalf("Busy=%v Ready=%v view=%q; want the unavailable banner", p.Busy(), p.Ready(), view)
			}
			if strings.Contains(view, "CRITICAL") || strings.Contains(view, "[minimal]") {
				t.Fatalf("view = %q; want no impact from raw risk or a minimal fallback", view)
			}
			if !strings.Contains(view, "[unavailable]") {
				t.Fatalf("module row = %q; want [unavailable]", view)
			}
		})
	}
}

func TestActivate_GivenCancelledRequest_ShouldRequestAgainAndRenderTheNewResult(t *testing.T) {
	summary := eventSummary()
	p, h, _ := newBlastHarness(t, summary, classifiedEvent(summary, sdk.RiskLow, "Fresh classification.").Assessment)
	p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary})
	late := p.Activate()
	p.Cancel()
	runBlastCmd(p, late)
	if view := p.View(120, 40); !strings.Contains(view, "cancelled") {
		t.Fatalf("late result after Cancel = %q; want the cancellation kept", view)
	}

	runBlastCmd(p, p.Activate())

	if len(h.AssessmentRequests) != 2 || h.AssessmentRequests[0] != summary || h.AssessmentRequests[1] != summary {
		t.Fatalf("assessment requests = %v; want two requests for %p", h.AssessmentRequests, summary)
	}
	p.ToggleExpand()
	if view := p.View(120, 40); !p.Ready() || !strings.Contains(view, "Minimal blast radius") || !strings.Contains(view, "Fresh classification.") {
		t.Fatalf("re-activation after Cancel = %q; want the new host result", view)
	}
}
