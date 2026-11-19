package blastradius

import (
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

var (
	ctrlR  = tea.KeyMsg{Type: tea.KeyCtrlR}
	escKey = tea.KeyMsg{Type: tea.KeyEsc}
)

func blastSummary() *sdk.PlanSummary {
	return &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.a"}, Action: sdk.ActionUpdate}}}
}

// failedBlast returns a plugin in the named error state and the harness that
// answers its requests, with the setup request forgotten.
func failedBlast(t *testing.T, kind string) (*Plugin, *sdktest.PluginDepsHarness, *sdk.PlanSummary) {
	t.Helper()
	summary := blastSummary()
	assessment := sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorConfigurationInvalid}
	p, h, _ := newBlastHarness(t, summary, assessment)
	runBlastCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
	switch kind {
	case "cancelled":
		activation := p.Activate()
		p.Cancel()
		runBlastCmd(p, activation) // the runtime runs it all: the tick of a stopped timer ends at once, freeing the next Start
	case "unusable assessment":
		runBlastCmd(p, p.Activate())
	}
	if p.Status() == sdk.StatusLoading || p.Status() == sdk.StatusIdle || (p.assessment != nil && p.assessment.Usable()) {
		t.Fatalf("setup %s: status = %v, want a failed run", kind, p.Status())
	}
	h.AssessmentRequests = nil
	return p, h, summary
}

func TestUpdate_GivenCtrlRInErrorState_ShouldIssueOneRequestForTheSameSummaryAndLoad(t *testing.T) {
	for _, kind := range []string{"cancelled", "unusable assessment"} {
		t.Run(kind, func(t *testing.T) {
			p, h, summary := failedBlast(t, kind)

			_, cmd := p.Update(ctrlR)

			if p.Status() != sdk.StatusLoading {
				t.Fatalf("status after ctrl+r = %v, want loading", p.Status())
			}
			runBlastCmd(p, cmd)
			if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != summary {
				t.Fatalf("requests after ctrl+r = %v, want exactly one for %p", h.AssessmentRequests, summary)
			}
		})
	}
}

func TestUpdate_GivenCtrlRInErrorState_ShouldRestartTheElapsedTimer(t *testing.T) {
	p, _, _ := failedBlast(t, "cancelled")

	_, retry := p.Update(ctrlR)
	sdktest.LandTick(t, retry)

	if _, next := p.Update(ui.TimerTickMsg{}); next == nil {
		t.Fatal("tick after ctrl+r returned no next tick, so the retry's elapsed time would freeze")
	}
}

func TestUpdate_GivenRetryAnswered_ShouldLeaveTheErrorState(t *testing.T) {
	p, h, summary := failedBlast(t, "unusable assessment")
	low := sdk.RiskLow
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: summary, Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &low, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.a", Status: sdk.ResourceClassified, Level: &low},
	}}}

	_, cmd := p.Update(ctrlR)
	runBlastCmd(p, cmd)

	if p.Status() != sdk.StatusDone || !p.assessment.Usable() {
		t.Fatalf("after the retry was answered: status %v, assessment %+v; want a usable one", p.Status(), p.assessment)
	}
}

func TestUpdate_GivenCtrlROutsideErrorState_ShouldDoNothing(t *testing.T) {
	low := sdk.RiskLow
	complete := sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &low, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.a", Status: sdk.ResourceClassified, Level: &low},
	}}
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness)
	}{
		{"idle", func(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness) {
			p, h, _ := newBlastHarness(t, blastSummary(), complete)
			return p, h
		}},
		{"loading", func(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness) {
			p, h, _ := newBlastHarness(t, blastSummary(), complete)
			p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: blastSummary()})
			p.Activate()
			return p, h
		}},
		{"done", func(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness) {
			summary := blastSummary()
			p, h, _ := newBlastHarness(t, summary, complete)
			runBlastCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
			runBlastCmd(p, p.Activate())
			h.AssessmentRequests = nil
			return p, h
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, h := tc.setup(t)
			before := p.Status()
			requests := len(h.AssessmentRequests)

			_, cmd := p.Update(ctrlR)

			if cmd != nil || p.Status() != before || len(h.AssessmentRequests) != requests {
				t.Fatalf("ctrl+r moved status %v to %v, issued %d requests, cmd %v; want nothing",
					before, p.Status(), len(h.AssessmentRequests)-requests, cmd != nil)
			}
		})
	}
}

func TestHints_GivenErrorState_ShouldOfferRetryAndBackAndInspectWhenModulesShow(t *testing.T) {
	for _, tc := range []struct {
		kind string
		want []sdk.KeyHint
	}{
		{"cancelled", (sdk.HintSetRetry | sdk.HintSetBack).Hints()},
		{"unusable assessment", (sdk.HintSetInspect | sdk.HintSetRetry | sdk.HintSetBack).Hints()},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			p, _, _ := failedBlast(t, tc.kind)

			if got := p.Hints(); !slices.Equal(got, tc.want) {
				t.Fatalf("Hints() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUpdate_GivenEscInErrorState_ShouldRequestDeactivation(t *testing.T) {
	for _, kind := range []string{"cancelled", "unusable assessment"} {
		t.Run(kind, func(t *testing.T) {
			p, _, _ := failedBlast(t, kind)

			_, cmd := p.Update(escKey)

			if cmd == nil {
				t.Fatal("Esc returned no command, so the Esc back hint would do nothing")
			}
			if _, ok := cmd().(sdk.DeactivateMsg); !ok {
				t.Fatalf("Esc returned %T, want sdk.DeactivateMsg", cmd())
			}
		})
	}
}

func TestUpdate_GivenEscInDoneState_ShouldRequestDeactivation(t *testing.T) {
	low := sdk.RiskLow
	complete := sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &low, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.a", Status: sdk.ResourceClassified, Level: &low},
	}}
	summary := blastSummary()
	p, _, _ := newBlastHarness(t, summary, complete)
	runBlastCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
	runBlastCmd(p, p.Activate())
	if p.Status() != sdk.StatusDone {
		t.Fatalf("setup: status = %v, want done", p.Status())
	}

	_, cmd := p.Update(escKey)

	if cmd == nil {
		t.Fatal("Esc returned no command, so the result view would trap the user until q")
	}
	if _, ok := cmd().(sdk.DeactivateMsg); !ok {
		t.Fatalf("Esc returned %T, want sdk.DeactivateMsg", cmd())
	}
}
