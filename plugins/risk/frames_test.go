package risk

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestListFrame_GivenNoReviewedMutation_ShouldKeepListOnEnter(t *testing.T) {
	for _, status := range []sdk.Status{sdk.StatusIdle, sdk.StatusLoading, sdk.StatusDone, sdk.StatusError} {
		t.Run(map[sdk.Status]string{sdk.StatusIdle: "idle", sdk.StatusLoading: "loading", sdk.StatusDone: "done", sdk.StatusError: "error"}[status], func(t *testing.T) {
			p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
			p.status = status
			_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
			runLifecycleCmd(p, cmd)
			if p.Stack().Peek().ID() != "list" {
				t.Error("enter without a reviewed mutation must stay on the list")
			}
			if got, want := p.Stack().View(80, 18), p.View(80, 18); got != want {
				t.Errorf("root frame view = %q, want plugin view %q", got, want)
			}
			for _, hint := range p.Hints() {
				if hint.Description == "inspect" {
					t.Errorf("empty/status %v offers inspect", status)
				}
			}
		})
	}
}

func TestListFrame_WhenEscapePressed_ShouldRequestDeactivation(t *testing.T) {
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("escape should request deactivation")
	}
	if msg := cmd(); msg != (sdk.DeactivateMsg{}) {
		t.Errorf("escape emitted %T, want DeactivateMsg", msg)
	}
	if p.Stack().Peek().ID() != "list" {
		t.Error("root escape must preserve list for next activation")
	}
}

func TestFrames_GivenNonKeyMessage_ShouldPreserveCurrentView(t *testing.T) {
	p := newAnalyzedPlugin(t, lifecycleSummary("aws_instance.web"))
	for _, inspect := range []bool{false, true} {
		if inspect {
			_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
			runLifecycleCmd(p, cmd)
		}
		before := p.View(80, 18)
		cmd := p.Stack().Update(tea.WindowSizeMsg{Width: 80, Height: 18})
		runLifecycleCmd(p, cmd)
		if got := p.View(80, 18); got != before {
			t.Errorf("non-key message changed view:\n%s", got)
		}
	}
}

func TestInspect_GivenSelectedResourceInSecondGroup_ShouldShowOnlyItsValidatedReasons(t *testing.T) {
	low, high := sdk.RiskLow, sdk.RiskHigh
	p, _, _ := newAssessedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_db_instance.main"}, Action: sdk.ActionDelete, Risk: sdk.RiskHigh},
		{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionUpdate, Risk: sdk.RiskLow, AttributeDiffs: []sdk.AttributeDiff{{Key: "size", OldValue: "small", NewValue: "large"}}},
	}}, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{
		{Address: "aws_db_instance.main", Status: sdk.ResourceClassified, Level: &high, Reasons: []sdk.PolicyReason{{Message: "Database deletion"}}},
		{Address: "aws_instance.web", Status: sdk.ResourceClassified, Level: &low, Reasons: []sdk.PolicyReason{{Message: "Memory size changed"}}},
	}})
	runLifecycleCmd(p, p.Activate(Input{}))
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyDown})
	runLifecycleCmd(p, cmd)
	_, cmd = p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	runLifecycleCmd(p, cmd)
	view := ansi.Strip(p.View(80, 24))
	for _, want := range []string{"aws_instance.web", "size", "Memory size changed"} {
		if !strings.Contains(view, want) {
			t.Errorf("second resource inspection missing %q:\n%s", want, view)
		}
	}
	for _, unwanted := range []string{"Database deletion", "aws_db_instance.main"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("second resource inspection includes unrelated %q:\n%s", unwanted, view)
		}
	}
	hints := p.Hints()
	want := (sdk.HintSetWrap | sdk.HintSetBack).Hints()
	if len(hints) != len(want) {
		t.Fatalf("inspect hints = %v, want %v", hints, want)
	}
	for i := range want {
		if hints[i] != want[i] {
			t.Errorf("inspect hint = %v, want %v", hints[i], want[i])
		}
	}
}

func TestInspect_GivenPlanInvalidatedAndLateResult_ShouldCloseOldDetails(t *testing.T) {
	summary := lifecycleSummary("aws_instance.stale")
	p, _, h := newAssessedPlugin(t, summary, classifiedAs("aws_instance.stale", sdk.RiskLow, "Stale classification."))
	runLifecycleCmd(p, p.Activate(Input{}))
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	runLifecycleCmd(p, cmd)
	if p.Stack().Peek().ID() != "inspect" {
		t.Fatal("expected mutation inspection before invalidation")
	}
	runLifecycleCmd(p, p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{}))
	deliver(p, h.AssessmentEvent)
	if p.Stack().Peek().ID() != "list" {
		t.Error("invalidation must close old mutation inspection")
	}
	if view := p.View(80, 18); strings.Contains(view, "aws_instance.stale") || !strings.Contains(view, "Run a plan") {
		t.Errorf("late result restored stale details:\n%s", view)
	}
}

func TestInspect_WhenWrapToggled_ShouldShowCurrentWrapMode(t *testing.T) {
	p := newAnalyzedPlugin(t, lifecycleSummary("aws_instance.web"))
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	runLifecycleCmd(p, cmd)
	for _, want := range []string{"wrap(on)", "wrap(off)"} {
		_, cmd = p.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
		runLifecycleCmd(p, cmd)
		found := false
		for _, hint := range p.Hints() {
			if hint.Description == want {
				found = true
			}
		}
		if !found {
			t.Errorf("after wrap toggle hints = %v, want %s", p.Hints(), want)
		}
	}
}
