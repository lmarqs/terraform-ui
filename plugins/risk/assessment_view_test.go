package risk

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestAssessment_GivenEmptyOrExplicitNone_ShouldExplainDistinctReviewStates(t *testing.T) {
	none := sdk.RiskNone
	for _, tt := range []struct {
		name       string
		assessment sdk.PolicyAssessment
		want       string
	}{
		{
			name: "empty",
			assessment: sdk.PolicyAssessment{Status: "empty", Resources: []sdk.PolicyResourceAssessment{
				{Address: "aws_instance.reviewed", Actions: []string{"no-op"}, Status: "not_assessed", Exclusion: "no_op"},
			}},
			want: "No eligible mutations (1 changes not assessed).",
		},
		{
			name: "explicit_none",
			assessment: sdk.PolicyAssessment{Status: "complete", OverallLevel: &none, Resources: []sdk.PolicyResourceAssessment{
				{Address: "aws_instance.reviewed", Actions: []string{"update"}, Status: "classified", Level: &none, Reasons: []sdk.PolicyReason{{Code: "explicit-default", Message: "Explicit project classification for this update."}}},
			}},
			want: "Risk: none — explicit policy classifications.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, h := newLifecyclePlugin(t, &sdktest.MockService{})
			h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment { return tt.assessment }
			runLifecycleCmd(p, p.Init(h.Deps))
			summary := lifecycleSummary("aws_instance.reviewed")
			if tt.name == "empty" {
				summary.Changes[0].Action = sdk.ActionNoOp
			} else {
				summary.Changes[0].Action = sdk.ActionUpdate
			}
			runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
			runLifecycleCmd(p, p.Activate())
			view := ansi.Strip(p.View(100, 24))
			if !strings.Contains(view, tt.want) {
				t.Errorf("review must distinguish %s assessment with %q:\n%s", tt.name, tt.want, view)
			}
			for _, claim := range []string{"No risk detected", "look safe", "No changes to analyze"} {
				if strings.Contains(view, claim) {
					t.Errorf("assessment must describe classification evidence without %q:\n%s", claim, view)
				}
			}
		})
	}
}

func TestAssessment_GivenResourceFailures_ShouldRetainReviewRowsWithoutAggregate(t *testing.T) {
	low := sdk.RiskLow
	for _, partial := range []bool{true, false} {
		name, want := "unavailable", "Risk unavailable: 3 of 3 eligible mutations failed."
		if partial {
			name, want = "partial", "Risk unavailable: 1 of 3 eligible mutations failed; 2 classified."
		}
		t.Run(name, func(t *testing.T) {
			rows := []sdk.PolicyResourceAssessment{
				{Address: "aws_instance.z_failed", Actions: []string{"update"}, Status: "unavailable", ErrorCode: "result_undefined"},
				{Address: "aws_instance.a_valid", Actions: []string{"update"}, Status: "unavailable", ErrorCode: "result_undefined"},
				{Address: "aws_instance.b_valid", Actions: []string{"update"}, Status: "unavailable", ErrorCode: "result_undefined"},
			}
			if partial {
				for i := 1; i < len(rows); i++ {
					rows[i].Status, rows[i].Level, rows[i].ErrorCode = "classified", &low, ""
					rows[i].Reasons = []sdk.PolicyReason{{Code: "observed", Message: "Memory size changed."}}
				}
			}
			p, h := newLifecyclePlugin(t, &sdktest.MockService{})
			h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
				return sdk.PolicyAssessment{Status: name, Resources: rows, ErrorCode: "result_undefined"}
			}
			runLifecycleCmd(p, p.Init(h.Deps))
			summary := &sdk.PlanSummary{}
			for _, row := range rows {
				summary.Changes = append(summary.Changes, sdk.PlanChange{Resource: sdk.Resource{Address: row.Address}, Action: sdk.ActionUpdate, AttributeDiffs: []sdk.AttributeDiff{{Key: "memory_size", OldValue: "128", NewValue: "256"}}})
			}
			runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
			runLifecycleCmd(p, p.Activate())
			view := ansi.Strip(p.View(110, 30))
			if !strings.Contains(view, want) {
				t.Errorf("incomplete assessment must explain its failed and classified counts with %q:\n%s", want, view)
			}
			for _, row := range rows {
				if !strings.Contains(view, row.Address) {
					t.Errorf("incomplete assessment must retain review row %q:\n%s", row.Address, view)
				}
			}
			if partial && !strings.Contains(view, "LOW (2)") {
				t.Errorf("partial assessment must retain classified rows under their level:\n%s", view)
			}
			if strings.Contains(view, "No risk detected") || strings.Contains(view, "Low risk") || strings.Contains(view, "NONE (") {
				t.Errorf("incomplete assessment must not publish a subset aggregate or a none group:\n%s", view)
			}
			_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
			runLifecycleCmd(p, cmd)
			inspection := ansi.Strip(p.View(110, 30))
			firstUnavailable := "aws_instance.a_valid"
			if partial {
				firstUnavailable = "aws_instance.z_failed"
			}
			for _, evidence := range []string{firstUnavailable, "unavailable", "result_undefined", "memory_size", "128", "256"} {
				if !strings.Contains(inspection, evidence) {
					t.Errorf("first unavailable mutation inspection must retain %q:\n%s", evidence, inspection)
				}
			}
			if strings.Contains(inspection, "none") {
				t.Errorf("unavailable mutation inspection must never show none:\n%s", inspection)
			}
		})
	}
}

func TestAssessment_GivenValidatedReasons_ShouldShowEvidenceWithoutInventedSafetyClaims(t *testing.T) {
	for _, tt := range []struct {
		name    string
		level   sdk.RiskLevel
		action  sdk.Action
		phantom bool
	}{
		{name: "low", level: sdk.RiskLow, action: sdk.ActionUpdate},
		{name: "high", level: sdk.RiskHigh, action: sdk.ActionUpdate},
		{name: "phantom", level: sdk.RiskMedium, action: sdk.ActionUpdate, phantom: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, h := newLifecyclePlugin(t, &sdktest.MockService{})
			h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
				return sdk.PolicyAssessment{Status: "complete", OverallLevel: &tt.level, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Actions: []string{"update"}, Status: "classified", Level: &tt.level, Reasons: []sdk.PolicyReason{{Code: "observed", Message: "Memory size changed."}, {Code: "source", Message: "Source hash is unchanged."}}}}}
			}
			runLifecycleCmd(p, p.Init(h.Deps))
			summary := lifecycleSummary("aws_instance.reviewed")
			summary.Changes[0].Action, summary.Changes[0].IsPhantom = tt.action, tt.phantom
			runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
			runLifecycleCmd(p, p.Activate())
			view := ansi.Strip(p.View(120, 24))
			if !strings.Contains(view, "Memory size changed.; Source hash is unchanged.") {
				t.Errorf("list must retain multiple validated explanations in order:\n%s", view)
			}
			for _, claim := range []string{"look safe", "cosmetic only", "modification of critical resource"} {
				if strings.Contains(view, claim) {
					t.Errorf("review must use selected-policy evidence instead of invented claim %q:\n%s", claim, view)
				}
			}
			_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
			runLifecycleCmd(p, cmd)
			if inspection := ansi.Strip(p.View(100, 24)); !strings.Contains(inspection, "Memory size changed.") {
				t.Errorf("inspection must retain validated policy explanation:\n%s", inspection)
			}
		})
	}
}
