package risk

import (
	"errors"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func pendingPlugin(t *testing.T) *Plugin {
	t.Helper()
	p, _ := pendingPluginActivation(t)
	return p
}

// pendingPluginActivation also returns the Cmd Activate answered with, for a
// test that needs the first tick to land.
func pendingPluginActivation(t *testing.T) (*Plugin, tea.Cmd) {
	t.Helper()
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	cmd := p.Activate(Input{})
	if p.Status() != sdk.StatusLoading {
		t.Fatalf("setup: status = %v, want loading", p.Status())
	}
	return p, cmd
}

func TestView_GivenPendingAssessment_ShouldShowElapsedTimeInFaintItalic(t *testing.T) {
	p := pendingPlugin(t)

	view := p.View(80, 20)

	want := sdk.StyleFaintItalic.Render("Loading risk assessment... 0s")
	if view != want || ansi.Strip(view) != "Loading risk assessment... 0s" {
		t.Fatalf("View() = %q, want %q", view, want)
	}
}

func TestHints_GivenPendingAssessment_ShouldShowOnlyEscBack(t *testing.T) {
	p := pendingPlugin(t)

	got, want := p.Hints(), []sdk.KeyHint{{Key: "Esc", Description: "back"}}

	if !slices.Equal(got, want) {
		t.Fatalf("Hints() = %v, want %v", got, want)
	}
}

func TestUpdate_GivenTimerTick_ShouldKeepTickingOnlyWhilePending(t *testing.T) {
	p, activation := pendingPluginActivation(t)
	sdktest.LandTick(t, activation)
	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd == nil {
		t.Fatal("tick while pending returned no next tick, so the elapsed time would freeze")
	}

	deliver(p, sdk.RiskAssessedEvent{Request: 1, Summary: lifecycleSummary("aws_instance.a"), Assessment: classifiedAs("aws_instance.a", sdk.RiskLow, "Answered.")})

	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd != nil {
		t.Fatal("tick after the answer returned a next tick, so the timer would never stop")
	}
}

// Plan clears its busy state before it publishes PlanCompletedEvent, so Risk can
// be opened cold in that gap and receive the event while its request runs.
func TestPlanCompleted_GivenColdRequestPending_ShouldRequestCompletedPlanAndRenderIt(t *testing.T) {
	summary := lifecycleSummary("aws_instance.reviewed")
	p, h := newLifecyclePlugin(t, &sdktest.MockService{})
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: summary, Assessment: classifiedAs("aws_instance.reviewed", sdk.RiskMedium, "Completed plan classification.")}
	cold := p.Activate(Input{})

	runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))

	if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != summary {
		t.Fatalf("assessment requests = %v; want one request for the completed plan %p", h.AssessmentRequests, summary)
	}
	if p.Status() != sdk.StatusDone || !strings.Contains(ansi.Strip(p.View(100, 30)), "Completed plan classification.") {
		t.Fatalf("status %v, view %q; want the completed plan's result rendered", p.Status(), ansi.Strip(p.View(100, 30)))
	}

	h.AssessmentEvent = sdk.RiskAssessedEvent{Err: errors.New("late cold failure")}
	runLifecycleCmd(p, cold)
	if p.Status() != sdk.StatusDone || strings.Contains(ansi.Strip(p.View(100, 30)), "late cold failure") {
		t.Fatalf("the replaced cold request's late result changed the view: %q", ansi.Strip(p.View(100, 30)))
	}
}

func TestPlanCompleted_GivenColdRequestPending_ShouldStayLoadingUntilAnswered(t *testing.T) {
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	p.Activate(Input{})

	cmd := p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: lifecycleSummary("aws_instance.reviewed")})

	if cmd == nil || p.Status() != sdk.StatusLoading || p.Busy() {
		t.Fatalf("cmd nil=%v status %v busy=%v; want a request in flight, loading, and no host plan to wait for", cmd == nil, p.Status(), p.Busy())
	}
}
