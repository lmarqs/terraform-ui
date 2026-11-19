package risk

import (
	"slices"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func pendingPlugin(t *testing.T) *Plugin {
	t.Helper()
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	p.Activate(Input{})
	if p.Status() != sdk.StatusLoading {
		t.Fatalf("setup: status = %v, want loading", p.Status())
	}
	return p
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
	p := pendingPlugin(t)
	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd == nil {
		t.Fatal("tick while pending returned no next tick, so the elapsed time would freeze")
	}

	deliver(p, sdk.RiskAssessedEvent{Request: 1, Summary: lifecycleSummary("aws_instance.a"), Assessment: classifiedAs("aws_instance.a", sdk.RiskLow, "Answered.")})

	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd != nil {
		t.Fatal("tick after the answer returned a next tick, so the timer would never stop")
	}
}
