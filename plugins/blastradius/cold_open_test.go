package blastradius

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

// coldBlast is a plugin that has seen no plan; the host answers its requests with ev.
func coldBlast(t *testing.T, ev sdk.RiskAssessedEvent) (*Plugin, *sdktest.PluginDepsHarness, *sdktest.MockService) {
	t.Helper()
	p, h, svc := newBlastHarness(t, nil, sdk.PolicyAssessment{})
	h.AssessmentEvent = ev
	return p, h, svc
}

func TestActivate_GivenNoReviewedPlan_ShouldRequestHostPlanAndShowItsAssessment(t *testing.T) {
	planned := eventSummary()
	p, h, svc := coldBlast(t, classifiedEvent(planned, sdk.RiskLow, "Host planned classification."))

	cmd := p.Activate()

	if cmd == nil || p.Status() != sdk.StatusLoading || p.Ready() || !p.Busy() {
		t.Fatalf("cold Activate: cmd=%v status=%v ready=%v busy=%v; want a pending host plan that holds the directory", cmd != nil, p.Status(), p.Ready(), p.Busy())
	}
	runBlastCmd(p, cmd)
	if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != nil {
		t.Fatalf("assessment requests = %v; want one nil-summary request so the host plans", h.AssessmentRequests)
	}
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Blast Radius ran terraform plan %d times; planning belongs to the host", len(svc.PlanCalls))
	}
	p.ToggleExpand()
	if view := p.View(120, 40); !p.Ready() || p.Busy() || !strings.Contains(view, "Minimal blast radius") || !strings.Contains(view, "Host planned classification.") {
		t.Fatalf("Ready=%v Busy=%v view=%q; want the host-planned summary classified by the event", p.Ready(), p.Busy(), view)
	}
}

func TestHandleRiskAssessed_GivenHostPlannedSummary_ShouldRetainItForTheNextRequest(t *testing.T) {
	planned := eventSummary()
	p, h, _ := coldBlast(t, classifiedEvent(planned, sdk.RiskLow, "Host planned classification."))
	runBlastCmd(p, p.Activate())

	again := p.Activate()

	if p.Busy() {
		t.Fatal("Busy() with a retained summary; a policy-only request runs no terraform")
	}
	runBlastCmd(p, again)
	if len(h.AssessmentRequests) != 2 || h.AssessmentRequests[1] != planned {
		t.Fatalf("assessment requests = %v; want the second request to carry the summary the host planned (%p)", h.AssessmentRequests, planned)
	}
}

func TestHandleRiskAssessed_GivenHostFailureWithoutPlan_ShouldShowItWithRetryAndBack(t *testing.T) {
	for _, tc := range []struct {
		name string
		ev   sdk.RiskAssessedEvent
		want []string // lines the view shows, in order
	}{
		{"plan error", sdk.RiskAssessedEvent{Err: errors.New("terraform planning failed")}, []string{"plan unavailable: terraform planning failed"}},
		{"policy failure", sdk.RiskAssessedEvent{
			Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorConfigurationInvalid},
			Hint:       "fix tfui.hcl",
		}, []string{"risk unavailable: configuration_invalid", "hint: fix tfui.hcl"}},
		{"no reason", sdk.RiskAssessedEvent{}, []string{"risk unavailable"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, h, _ := coldBlast(t, tc.ev)
			runBlastCmd(p, p.Activate())

			var lines []string
			for _, line := range strings.Split(ansi.Strip(p.View(120, 40)), "\n") {
				lines = append(lines, strings.TrimSpace(line))
			}
			if shown := slices.DeleteFunc(slices.Clone(lines), func(l string) bool { return l == "" }); !p.Ready() || p.Busy() || !slices.Equal(shown, tc.want) {
				t.Fatalf("Ready=%v Busy=%v lines=%q; want %q", p.Ready(), p.Busy(), shown, tc.want)
			}
			wantHints := (sdk.HintSetRetry | sdk.HintSetBack).Hints()
			if got := p.Hints(); !slices.Equal(got, wantHints) {
				t.Fatalf("Hints() = %v, want %v", got, wantHints)
			}

			h.AssessmentRequests = nil
			runBlastCmd(p, p.Activate())
			if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != nil {
				t.Fatalf("retry requests = %v; want one nil-summary request, since no plan was retained", h.AssessmentRequests)
			}
		})
	}
}

func TestUpdate_GivenEscOrCancelWhileHostPlans_ShouldLeaveAndIgnoreTheLateResult(t *testing.T) {
	planned := eventSummary()
	p, _, _ := coldBlast(t, classifiedEvent(planned, sdk.RiskLow, "Late classification."))
	late := p.Activate()

	_, cmd := p.Update(escKey)
	if _, ok := cmd().(sdk.DeactivateMsg); !ok {
		t.Fatal("Esc while the host plans did not request deactivation")
	}
	p.Cancel()
	runBlastCmd(p, late)

	if view := ansi.Strip(p.View(120, 40)); p.Status() != sdk.StatusError || view != "risk unavailable: cancelled" {
		t.Fatalf("status=%v view=%q; want the cancellation kept after a late host result", p.Status(), view)
	}
}
