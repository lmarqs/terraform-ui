package blastradius

import (
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func pendingBlast(t *testing.T) (*Plugin, *sdk.PlanSummary) {
	t.Helper()
	p, summary, _ := pendingBlastActivation(t)
	return p, summary
}

// pendingBlastActivation also returns the Cmd Activate answered with, for a
// test that needs the first tick to land.
func pendingBlastActivation(t *testing.T) (*Plugin, *sdk.PlanSummary, tea.Cmd) {
	t.Helper()
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.a"}, Action: sdk.ActionUpdate}}}
	p, _, _ := newBlastHarness(t, summary, sdk.PolicyAssessment{})
	p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary})
	cmd := p.Activate()
	if p.Status() != sdk.StatusLoading {
		t.Fatalf("setup: status = %v, want loading", p.Status())
	}
	return p, summary, cmd
}

func TestView_GivenPendingAssessment_ShouldShowElapsedTimeInFaintItalic(t *testing.T) {
	p, _ := pendingBlast(t)

	view := p.View(80, 20)

	want := sdk.StyleFaintItalic.Render("Loading risk assessment... 0s")
	if view != want || ansi.Strip(view) != "Loading risk assessment... 0s" {
		t.Fatalf("View() = %q, want %q", view, want)
	}
}

func TestHints_GivenPendingAssessment_ShouldShowOnlyEscBack(t *testing.T) {
	p, _ := pendingBlast(t)

	got, want := p.Hints(), []sdk.KeyHint{{Key: "Esc", Description: "back"}}

	if !slices.Equal(got, want) {
		t.Fatalf("Hints() = %v, want %v", got, want)
	}
}

func TestUpdate_GivenTimerTick_ShouldKeepTickingOnlyWhilePending(t *testing.T) {
	p, summary, activation := pendingBlastActivation(t)
	sdktest.LandTick(t, activation)
	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd == nil {
		t.Fatal("tick while pending returned no next tick, so the elapsed time would freeze")
	}

	deliverBlast(p, sdk.RiskAssessedEvent{Request: 1, Summary: summary, Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorCancelled}})

	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd != nil {
		t.Fatal("tick after the answer returned a next tick, so the timer would never stop")
	}
}

func TestUpdate_GivenEscWhilePending_ShouldRequestDeactivation(t *testing.T) {
	p, _ := pendingBlast(t)

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("Esc while loading returned no command, so it cannot go back")
	}
	if _, ok := cmd().(sdk.DeactivateMsg); !ok {
		t.Fatalf("Esc while loading returned %T, want sdk.DeactivateMsg", cmd())
	}
}
