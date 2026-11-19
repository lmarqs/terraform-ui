package risk

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestView_GivenMutationWithThreeDiffs_ShouldShowTwoPathsAndOverflow(t *testing.T) {
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{{
		Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionUpdate, Risk: sdk.RiskLow,
		AttributeDiffs: []sdk.AttributeDiff{
			{Key: "instance_type", OldValue: "t3.small", NewValue: "t3.large"},
			{Key: "tags.Owner", OldValue: "ops", NewValue: "platform"},
			{Key: "tags.Hidden", OldValue: "before", NewValue: "after"},
		},
	}}})
	view := ansi.Strip(p.View(90, 24))
	for _, want := range []string{"~ aws_instance.web", "instance_type", "t3.small", "t3.large", "tags.Owner", "+1 more"} {
		if !strings.Contains(view, want) {
			t.Errorf("compact mutation view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "tags.Hidden") {
		t.Errorf("third path must be available through inspection, not repeated inline:\n%s", view)
	}
	if strings.Contains(view, "Change:") {
		t.Errorf("resource rows must not repeat a Change label:\n%s", view)
	}
}

func TestNavigation_GivenLongPlan_WhenMovingAndResizing_ShouldKeepSelectedResourceVisible(t *testing.T) {
	changes := make([]sdk.PlanChange, 30)
	for i := range changes {
		changes[i] = sdk.PlanChange{Resource: sdk.Resource{Address: fmt.Sprintf("aws_instance.node_%02d", i)}, Action: sdk.ActionUpdate, Risk: sdk.RiskLow}
	}
	p := newAnalyzedPlugin(t, &sdk.PlanSummary{Changes: changes})
	p.View(80, 14)
	for range 24 {
		_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyDown})
		runLifecycleCmd(p, cmd)
	}
	for _, size := range [][2]int{{80, 14}, {42, 8}, {110, 22}} {
		view := ansi.Strip(p.View(size[0], size[1]))
		if !strings.Contains(view, "aws_instance.node_24") {
			t.Errorf("selected resource lost at %dx%d:\n%s", size[0], size[1], view)
		}
		if !strings.ContainsAny(view, "▲▼┃│") {
			t.Errorf("long plan must show standard scroll gutter at %dx%d:\n%s", size[0], size[1], view)
		}
		if len(strings.Split(strings.TrimSuffix(view, "\n"), "\n")) > size[1] {
			t.Errorf("view exceeds height %d:\n%s", size[1], view)
		}
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Errorf("line exceeds width %d: %q", size[0], line)
			}
		}
	}
	pos, total := p.CursorPosition()
	if pos != 25 || total != 30 {
		t.Errorf("resource cursor = (%d,%d), want (25,30)", pos, total)
	}
	for range 24 {
		_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyUp})
		runLifecycleCmd(p, cmd)
	}
	if view := ansi.Strip(p.View(42, 8)); !strings.Contains(view, "aws_instance.node_00") {
		t.Errorf("scrolling back must restore first selected resource:\n%s", view)
	}
}

func TestInspect_GivenReplacementWithSensitiveAndUnknownDiffs_ShouldShowEveryPathAndValidatedReason(t *testing.T) {
	for _, action := range []sdk.Action{sdk.ActionDeleteThenCreate, sdk.ActionCreateThenDelete} {
		t.Run(string(action), func(t *testing.T) {
			high := sdk.RiskHigh
			p, _, _ := newAssessedPlugin(t, &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: action, Risk: sdk.RiskHigh, AttributeDiffs: []sdk.AttributeDiff{
				{Key: "ami", OldValue: "old", NewValue: "(known after apply)", ForcesNew: true},
				{Key: "password", OldValue: "secret-before", NewValue: "secret-after", Sensitive: true},
				{Key: "tags.Owner", OldValue: "ops", NewValue: "platform"},
			}}}}, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.web", Status: sdk.ResourceClassified, Level: &high, Reasons: []sdk.PolicyReason{{Code: "replacement", Message: "The image change requires replacement."}}}}})
			runLifecycleCmd(p, p.Activate())
			compact := ansi.Strip(p.View(42, 18))
			symbol := "-/+"
			if action == sdk.ActionCreateThenDelete {
				symbol = "+/-"
			}
			if !strings.Contains(compact, symbol+" aws_instance.web") {
				t.Errorf("narrow list must preserve replacement order:\n%s", compact)
			}
			if strings.Contains(compact, "secret-before") || strings.Contains(compact, "secret-after") {
				t.Errorf("narrow list leaks sensitive value:\n%s", compact)
			}
			_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
			runLifecycleCmd(p, cmd)
			view := ansi.Strip(p.View(90, 30))
			for _, want := range []string{string(action), "ami", "(forces new)", "(known after apply)", "password", "(sensitive)", "tags.Owner", "The image change requires replacement."} {
				if !strings.Contains(view, want) {
					t.Errorf("inspection missing %q:\n%s", want, view)
				}
			}
			if strings.Contains(view, "secret-before") || strings.Contains(view, "secret-after") {
				t.Errorf("inspection leaks sensitive value:\n%s", view)
			}
			_, cmd = p.Update(tea.KeyMsg{Type: tea.KeyEsc})
			runLifecycleCmd(p, cmd)
			if strings.Contains(ansi.Strip(p.View(90, 30)), "tags.Owner") {
				t.Error("escape should return to compact resource list")
			}
		})
	}
}
