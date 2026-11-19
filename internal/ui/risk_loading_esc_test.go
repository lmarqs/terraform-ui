package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Esc while a plugin loads its assessment goes home and cancels that plugin's
// own request: its Cmd, run afterwards, finds its context done and reports a
// cancellation, whether the host would plan (Risk) or only evaluate policy.
func TestEsc_GivenPendingAssessment_ShouldGoHomeAndCancelTheRequest(t *testing.T) {
	for _, tc := range []struct {
		pluginID string
		retained bool
	}{
		{"risk", false},
		{"risk", true},
		{"blastradius", true},
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
			cancelled := event.Err != nil || event.Assessment.ErrorCode == sdk.ErrorCancelled
			if !cancelled {
				t.Fatalf("request ran after Esc and produced %+v, want a cancelled request", event)
			}
		})
	}
}
