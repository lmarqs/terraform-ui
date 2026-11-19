package blastradius

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Blast Radius always assesses a summary it holds, so a pending assessment is
// policy-only: no Terraform runs, but headless output still waits for the result.
func TestBusy_GivenPendingAssessment_ShouldNotBeBusyYetNotReady(t *testing.T) {
	summary := eventSummary()
	p, _, _ := newBlastHarness(t, summary, classifiedEvent(summary, sdk.RiskLow, "Host classification.").Assessment)
	p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary})

	p.Activate()

	if p.Status() != sdk.StatusLoading || p.Ready() {
		t.Fatalf("setup: status=%v ready=%v; want a pending assessment that is not ready", p.Status(), p.Ready())
	}
	if p.Busy() {
		t.Fatal("Busy() = true while only the policy assessment is pending, want false")
	}
}
