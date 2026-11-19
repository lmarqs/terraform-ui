package risk

import (
	"errors"
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

// partialAssessment classifies aws_instance.a and fails aws_instance.b, so the
// run is unusable yet still lists the classified change.
func partialAssessment() sdk.PolicyAssessment {
	low := sdk.RiskLow
	return sdk.PolicyAssessment{Status: sdk.AssessmentPartial, ErrorCode: sdk.ErrorResultInvalid, OverallLevel: &low, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_instance.a", Status: sdk.ResourceClassified, Level: &low},
		{Address: "aws_instance.b", Status: sdk.ResourceUnavailable, ErrorCode: sdk.ErrorResultInvalid},
	}}
}

func twoChangeSummary() *sdk.PlanSummary {
	return &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_instance.a"}, Action: sdk.ActionCreate},
		{Resource: sdk.Resource{Address: "aws_instance.b"}, Action: sdk.ActionCreate},
	}}
}

// failedRisk returns a plugin in the named error outcome and the harness that
// answers its requests, with the setup request forgotten.
func failedRisk(t *testing.T, kind string) (*Plugin, *sdktest.PluginDepsHarness, *sdk.PlanSummary) {
	t.Helper()
	p, h := newLifecyclePlugin(t, &sdktest.MockService{})
	var retained *sdk.PlanSummary
	switch kind {
	case "plan failed":
		h.AssessmentEvent = sdk.RiskAssessedEvent{Err: errors.New("terraform planning failed")}
	case "plan failed with retained plan":
		retained = lifecycleSummary("aws_instance.a")
		runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: retained}))
		h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: retained, Err: errors.New("policy engine crashed")}
	case "unusable":
		h.AssessmentEvent = sdk.RiskAssessedEvent{Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorConfigurationInvalid}}
	case "unusable with items":
		retained = twoChangeSummary()
		runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: retained}))
		h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: retained, Assessment: partialAssessment()}
	}
	runLifecycleCmd(p, p.Activate(Input{}))
	if p.Status() != sdk.StatusError {
		t.Fatalf("setup %s: status = %v, want error", kind, p.Status())
	}
	h.AssessmentRequests = nil
	return p, h, retained
}

func TestUpdate_GivenCtrlRInErrorState_ShouldIssueOneRequestForTheSameSummaryAndLoad(t *testing.T) {
	for _, kind := range []string{"plan failed", "plan failed with retained plan", "unusable", "unusable with items"} {
		t.Run(kind, func(t *testing.T) {
			p, h, retained := failedRisk(t, kind)

			_, cmd := p.Update(ctrlR)

			if p.Status() != sdk.StatusLoading {
				t.Fatalf("status after ctrl+r = %v, want loading", p.Status())
			}
			runLifecycleCmd(p, cmd)
			if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != retained {
				t.Fatalf("requests after ctrl+r = %v, want exactly one for %p", h.AssessmentRequests, retained)
			}
		})
	}
}

func TestUpdate_GivenCtrlRInErrorState_ShouldRestartTheElapsedTimer(t *testing.T) {
	p, _, _ := failedRisk(t, "plan failed")

	_, cmd := p.Update(ctrlR)

	if got := p.View(80, 20); got == "" || p.Status() != sdk.StatusLoading {
		t.Fatalf("View() = %q while status %v, want the loading view", got, p.Status())
	}
	if cmd == nil {
		t.Fatal("ctrl+r returned no command")
	}
	if _, next := p.Update(ui.TimerTickMsg{}); next == nil {
		t.Fatal("tick after ctrl+r returned no next tick, so the retry's elapsed time would freeze")
	}
}

func TestUpdate_GivenRetryAnswered_ShouldLeaveTheErrorState(t *testing.T) {
	p, h, _ := failedRisk(t, "plan failed")
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: lifecycleSummary("aws_instance.a"), Assessment: classifiedAs("aws_instance.a", sdk.RiskLow, "Answered.")}

	_, cmd := p.Update(ctrlR)
	runLifecycleCmd(p, cmd)

	if p.Status() != sdk.StatusDone {
		t.Fatalf("status after the retry was answered = %v, want done", p.Status())
	}
}

func TestCancel_GivenRetryOfPartialAssessment_ShouldNotShowTheOldAnalysis(t *testing.T) {
	p, _, _ := failedRisk(t, "unusable with items")

	p.Update(ctrlR)
	p.Cancel()

	if got := p.View(80, 20); len(p.items) != 0 || got == "" {
		t.Fatalf("cancelled retry kept %d items from the previous run; view = %q", len(p.items), got)
	}
}

func TestUpdate_GivenCtrlROutsideErrorState_ShouldDoNothing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness)
	}{
		{"idle", func(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness) {
			return newLifecyclePlugin(t, &sdktest.MockService{})
		}},
		{"loading", func(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness) {
			p, h := newLifecyclePlugin(t, &sdktest.MockService{})
			p.Activate(Input{})
			return p, h
		}},
		{"done", func(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness) {
			p, h := newLifecyclePlugin(t, &sdktest.MockService{})
			assessAtPlanRisk(p, h, lifecycleSummary("aws_instance.a"))
			return p, h
		}},
		{"invalid threshold", func(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness) {
			p, h := newLifecyclePlugin(t, &sdktest.MockService{})
			bogus := "bogus"
			p.Activate(Input{Threshold: &bogus})
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

func TestHints_GivenErrorState_ShouldOfferOnlyKeysThatWork(t *testing.T) {
	retryBack := (sdk.HintSetRetry | sdk.HintSetBack).Hints()
	for _, tc := range []struct {
		name string
		kind string
		want []sdk.KeyHint
	}{
		{"plan failed", "plan failed", retryBack},
		{"unusable", "unusable", retryBack},
		{"unusable with items", "unusable with items", (sdk.HintSetInspect | sdk.HintSetWrap | sdk.HintSetRetry | sdk.HintSetBack).Hints()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, _, _ := failedRisk(t, tc.kind)

			if got := p.Hints(); !slices.Equal(got, tc.want) {
				t.Fatalf("Hints() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHints_GivenInvalidThreshold_ShouldOfferOnlyBack(t *testing.T) {
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	bogus := "bogus"
	p.Activate(Input{Threshold: &bogus})

	if got, want := p.Hints(), sdk.HintSetBack.Hints(); !slices.Equal(got, want) {
		t.Fatalf("Hints() = %v, want %v: the threshold is fixed by the CLI, so retry would fail the same way", got, want)
	}
}

func TestUpdate_GivenEscInEachErrorState_ShouldRequestDeactivation(t *testing.T) {
	for _, kind := range []string{"plan failed", "unusable", "unusable with items"} {
		t.Run(kind, func(t *testing.T) {
			p, _, _ := failedRisk(t, kind)

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

func TestUpdate_GivenEscWithInvalidThreshold_ShouldRequestDeactivation(t *testing.T) {
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	bogus := "bogus"
	p.Activate(Input{Threshold: &bogus})

	_, cmd := p.Update(escKey)

	if cmd == nil {
		t.Fatal("Esc returned no command, so the Esc back hint would do nothing")
	}
	if _, ok := cmd().(sdk.DeactivateMsg); !ok {
		t.Fatalf("Esc returned %T, want sdk.DeactivateMsg", cmd())
	}
}
