package plan

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Plan is Busy only while its own terraform plan runs. Once the plan is done and
// only the policy assessment is pending, no Terraform runs, yet Ready stays false
// so headless output waits for the classification.
func TestBusy_GivenPlanRunOrPendingAssessment_ShouldBeBusyOnlyWhileTerraformRuns(t *testing.T) {
	for _, tc := range []struct {
		name      string
		runPlan   bool // let the plan finish, leaving only its assessment pending
		wantBusy  bool
		wantReady bool
	}{
		{"terraform plan running", false, true, false},
		{"plan done, assessment pending", true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := moduleSummary()
			var last sdk.AssessmentID
			p, _, _ := newAssessedPlugin(t, summary, allHigh(summary), withholdAssessment(&last))

			cmd := p.Activate(Input{})
			if tc.runPlan {
				drainAssessmentPlan(t, p, cmd, nil)
				if p.Status() != sdk.StatusDone || last == 0 {
					t.Fatalf("setup: status=%v request=%d; want a done plan with its assessment requested", p.Status(), last)
				}
			}

			if p.Busy() != tc.wantBusy || p.Ready() != tc.wantReady {
				t.Fatalf("Busy()=%v Ready()=%v, want Busy()=%v Ready()=%v", p.Busy(), p.Ready(), tc.wantBusy, tc.wantReady)
			}
		})
	}
}
