package ui

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

// Esc while a plugin loads its assessment goes home and cancels that plugin's
// own request: its Cmd, run afterwards, finds its context done and reports a
// cancellation, whether the host would plan (Risk) or only evaluate policy. A
// request that would plan reports context.Canceled and never calls Plan.
func TestEsc_GivenPendingAssessment_ShouldGoHomeAndCancelTheRequest(t *testing.T) {
	for _, tc := range []struct {
		pluginID string
		retained bool
		hostPlan bool
	}{
		{"risk", false, true},
		{"risk", true, false},
		{"blastradius", true, false},
	} {
		name := tc.pluginID + " retained=" + map[bool]string{true: "yes", false: "no"}[tc.retained]
		t.Run(name, func(t *testing.T) {
			app, request := newPendingAssessmentAppWithRequest(t, tc.pluginID, tc.retained)

			model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
			app = model.(App)
			drainPolicyAppCommand(&app, cmd)

			if app.ActivePlugin() != nil {
				t.Fatalf("Esc while loading left %s active, want home", app.ActivePlugin().ID())
			}
			event := riskResult(t, request).Event
			if tc.hostPlan {
				if !errors.Is(event.Err, context.Canceled) {
					t.Fatalf("request ran after Esc with Err %v, want context.Canceled", event.Err)
				}
				if calls := app.svc.(*sdktest.MockService).PlanCalls; len(calls) != 0 {
					t.Fatalf("Esc left %d Plan calls, want none", len(calls))
				}
				return
			}
			if event.Assessment.Status != sdk.AssessmentCancelled || event.Assessment.ErrorCode != sdk.ErrorCancelled {
				t.Fatalf("request ran after Esc and produced %+v, want a cancelled assessment", event)
			}
		})
	}
}
