package risk

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestActivate_GivenSourceRestored_ShouldClearFailureWithoutRevivingOldClassification(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newLifecyclePlugin(t, svc)
	h.AssessmentEvent = sdk.RiskAssessedEvent{Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorSourceUnreadable}}
	runLifecycleCmd(p, p.Activate(Input{}))
	if p.ExitCode() != 1 || string(p.Stderr()) != "risk unavailable: source_unreadable\n" {
		t.Fatalf("source failure: exit %d, stderr %q", p.ExitCode(), p.Stderr())
	}
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: &sdk.PlanSummary{}, Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentEmpty}}
	runLifecycleCmd(p, p.Activate(Input{}))
	if p.Status() != sdk.StatusDone || p.ExitCode() != 0 || len(p.Stderr()) != 0 || len(p.groups) != 0 {
		t.Fatalf("recovery revived assessment: status %v, stderr %q", p.Status(), p.Stderr())
	}
	runLifecycleCmd(p, p.Activate(Input{}))
	if p.Status() != sdk.StatusDone {
		t.Fatalf("unchanged activation: status %v", p.Status())
	}
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Risk ran Service.Plan %d times; planning belongs to the host", len(svc.PlanCalls))
	}
	if len(h.AssessmentRequests) < 2 || h.AssessmentRequests[0] != nil || h.AssessmentRequests[1] != nil {
		t.Fatalf("assessment requests = %v; want nil-summary requests while Risk retains no plan", h.AssessmentRequests)
	}
}
