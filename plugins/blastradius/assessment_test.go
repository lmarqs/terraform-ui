package blastradius

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestActivate_GivenCustomAssessment_ShouldPreserveImpactArithmetic(t *testing.T) {
	for _, tc := range []struct {
		name  string
		level sdk.RiskLevel
		count int
		want  string
	}{
		{"one low database delete", sdk.RiskLow, 1, "Minimal blast radius"},
		{"three none including delete", sdk.RiskNone, 3, "High blast radius"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := &sdk.PlanSummary{}
			assessment := sdk.PolicyAssessment{Status: "complete", OverallLevel: &tc.level, Policy: sdk.PolicyIdentity{Kind: "custom", ID: "project-policy"}}
			for _, addr := range []string{"module.db.aws_db_instance.database", "module.db.aws_instance.a", "module.db.aws_instance.b"}[:tc.count] {
				summary.Changes = append(summary.Changes, sdk.PlanChange{Resource: sdk.Resource{Address: addr, Module: "module.db"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical})
				assessment.Resources = append(assessment.Resources, sdk.PolicyResourceAssessment{Address: addr, Status: "classified", Level: &tc.level, Reasons: []sdk.PolicyReason{{Code: "project", Message: "Project classification evidence"}}})
			}
			p := activateAssessmentBlast(t, summary, assessment)
			view := p.View(120, 40)
			if !strings.Contains(view, tc.want) || strings.Contains(view, "CRITICAL") {
				t.Fatalf("Blast Radius = %q; want %s from selected policy", view, tc.want)
			}
			p.ToggleExpand()
			if view = p.View(120, 40); !strings.Contains(view, "Project classification evidence") {
				t.Fatalf("expanded changes = %q; want available policy reasons", view)
			}
		})
	}
}

func TestActivate_GivenPartialAssessment_ShouldRetainCountsAndMarkOnlyIncompleteGroupsUnavailable(t *testing.T) {
	low, critical := sdk.RiskLow, sdk.RiskCritical
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "module.bad.aws_instance.a", Module: "module.bad"}, Action: sdk.ActionDelete, Risk: sdk.RiskLow},
		{Resource: sdk.Resource{Address: "module.bad.aws_instance.b", Module: "module.bad"}, Action: sdk.ActionUpdate, Risk: sdk.RiskLow},
		{Resource: sdk.Resource{Address: "module.bad.aws_instance.c", Module: "module.bad"}, Action: sdk.ActionUpdate, Risk: sdk.RiskLow},
		{Resource: sdk.Resource{Address: "module.good.aws_instance.a", Module: "module.good"}, Action: sdk.ActionUpdate, Risk: sdk.RiskLow},
	}}
	assessment := sdk.PolicyAssessment{Status: "partial", Resources: []sdk.PolicyResourceAssessment{
		{Address: summary.Changes[0].Resource.Address, Status: "unavailable", ErrorCode: "result_invalid"},
		{Address: summary.Changes[1].Resource.Address, Status: "classified", Level: &low, Reasons: []sdk.PolicyReason{{Code: "known", Message: "Available explanation"}}},
		{Address: summary.Changes[2].Resource.Address, Status: "classified", Level: &low},
		{Address: summary.Changes[3].Resource.Address, Status: "classified", Level: &critical},
	}}
	p := activateAssessmentBlast(t, summary, assessment)
	view := p.View(120, 40)
	if !strings.Contains(view, "Blast radius unavailable: incomplete risk assessment") || !strings.Contains(view, "4 total change(s)") {
		t.Fatalf("Blast Radius=%q; want unavailable overall banner with preserved counts", view)
	}
	bad, good := strings.Index(view, "module.bad"), strings.Index(view, "module.good")
	if bad < 0 || good < bad {
		t.Fatalf("group order=%q; unavailable group must come first", view)
	}
	badRow := strings.Split(view[bad:], "\n")[0]
	if !strings.Contains(strings.ToLower(badRow), "unavailable") || strings.Contains(badRow, "[HIGH]") {
		t.Fatalf("incomplete group=%q; want unavailable impact without numeric score", badRow)
	}
	if !strings.Contains(view[good:], "[CRITICAL]") {
		t.Fatalf("complete group=%q; want retained selected critical score", view[good:])
	}
	p.ToggleExpand()
	view = p.View(120, 40)
	if !strings.Contains(view, "Available explanation") || !strings.Contains(view, "result_invalid") {
		t.Fatalf("expanded group=%q; want retained reason and fixed failed classification diagnostic", view)
	}
}

func TestActivate_GivenGlobalAssessmentFailure_ShouldNotPresentMinimalImpact(t *testing.T) {
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.a"}, Action: sdk.ActionUpdate}}}
	p := activateAssessmentBlast(t, summary, sdk.PolicyAssessment{Status: "unavailable", ErrorCode: "compilation_failed"})
	view := p.View(120, 40)
	if !strings.Contains(view, "Blast radius unavailable: incomplete risk assessment") || strings.Contains(view, "[minimal]") {
		t.Fatalf("Blast Radius=%q; want explicit unavailable impact", view)
	}
}

func activateAssessmentBlast(t *testing.T, summary *sdk.PlanSummary, assessment sdk.PolicyAssessment) *Plugin {
	t.Helper()
	svc := &sdktest.MockService{}
	h := sdktest.NewDeps(svc)
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment { return assessment }
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	handler, ok := any(p).(interface {
		HandlePlanCompleted(sdk.PlanCompletedEvent) tea.Cmd
	})
	if !ok {
		t.Fatal("Blast Radius does not accept the completed reviewed plan for shared assessment")
	}
	var execute func(tea.Cmd)
	execute = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range batch {
				execute(sub)
			}
			return
		}
		_, next := p.Update(msg)
		execute(next)
	}
	execute(handler.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
	activator, ok := any(p).(interface{ Activate() tea.Cmd })
	if !ok {
		t.Fatal("Blast Radius cannot activate the shared selected-policy assessment")
	}
	execute(activator.Activate())
	return p
}
