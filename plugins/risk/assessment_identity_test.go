package risk

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestHandleRiskAssessed_GivenRequestIdentity_ShouldAcceptOnlyThePendingRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		// reissue requests again, superseding the first request.
		reissue bool
		// result picks the Request of the delivered event from the issued IDs.
		result      func(issued []sdk.AssessmentID) sdk.AssessmentID
		wantApplied bool
	}{
		{"own ID", false, func(issued []sdk.AssessmentID) sdk.AssessmentID { return issued[0] }, true},
		{"different ID with the same summary", false, func(issued []sdk.AssessmentID) sdk.AssessmentID { return issued[0] + 100 }, false},
		{"zero ID", false, func([]sdk.AssessmentID) sdk.AssessmentID { return 0 }, false},
		{"older own ID after a newer request", true, func(issued []sdk.AssessmentID) sdk.AssessmentID { return issued[0] }, false},
		{"newer own ID after a newer request", true, func(issued []sdk.AssessmentID) sdk.AssessmentID { return issued[1] }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &sdktest.MockService{}
			h := sdktest.NewDeps(svc)
			var issued []sdk.AssessmentID
			h.Deps.RequestAssessment = func(*sdk.PlanSummary) (sdk.AssessmentID, tea.Cmd) {
				id := sdk.AssessmentID(len(issued) + 1)
				issued = append(issued, id)
				return id, nil
			}
			p := New(svc).(*Plugin)
			runLifecycleCmd(p, p.Init(h.Deps))
			summary := lifecycleSummary("aws_instance.retained")
			runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
			runLifecycleCmd(p, p.Activate(Input{}))
			if tc.reissue {
				p.Cancel()
				runLifecycleCmd(p, p.Activate(Input{}))
			}
			if want := map[bool]int{false: 1, true: 2}[tc.reissue]; len(issued) != want {
				t.Fatalf("issued %d requests; want %d", len(issued), want)
			}

			deliver(p, sdk.RiskAssessedEvent{Request: tc.result(issued), Summary: summary, Assessment: classifiedAs("aws_instance.retained", sdk.RiskLow, "Identified classification.")})

			if applied := !p.Busy() && p.Assessment() != nil; applied != tc.wantApplied {
				t.Fatalf("applied=%v busy=%v assessment=%v; want applied=%v", applied, p.Busy(), p.Assessment(), tc.wantApplied)
			}
		})
	}
}
