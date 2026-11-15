package plan

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func TestActivate_GivenSelectedPolicy_ShouldReviewCustomClassification(t *testing.T) {
	low := sdk.RiskLow
	summary := &sdk.PlanSummary{ToDelete: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return summary, nil }}
	h := sdktest.NewDeps(svc)
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
		return sdk.PolicyAssessment{Status: "complete", OverallLevel: &low, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_db_instance.database", Status: "classified", Level: &low, Reasons: []sdk.PolicyReason{{Code: "custom", Message: "Project deletion classification"}}}}}
	}
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	output, err := p.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(output)), "critical") || !strings.Contains(string(output), "Risk: low") {
		t.Fatalf("Plan output = %q; want selected custom LOW classification", output)
	}
	p.inspectSelected()
	if view := p.View(120, 40); !strings.Contains(view, "Project deletion classification") {
		t.Fatalf("inspection = %q; want selected policy explanation", view)
	}
}

func TestActivate_GivenAssessmentFailure_ShouldKeepReviewAndPlanArtifact(t *testing.T) {
	low := sdk.RiskLow
	for _, tc := range []struct {
		name       string
		assessment sdk.PolicyAssessment
	}{
		{"global", sdk.PolicyAssessment{Status: "unavailable", ErrorCode: "compilation_failed"}},
		{"partial", sdk.PolicyAssessment{Status: "partial", Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.good", Status: "classified", Level: &low}, {Address: "aws_db_instance.failed", Status: "unavailable", ErrorCode: "result_invalid"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := &sdk.PlanSummary{ToDelete: 1, ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.good"}, Action: sdk.ActionUpdate, Risk: sdk.RiskCritical}, {Resource: sdk.Resource{Address: "aws_db_instance.failed"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}
			svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return summary, nil }}
			h := sdktest.NewDeps(svc)
			h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment { return tc.assessment }
			p := New(svc).(*Plugin)
			p.Init(h.Deps)
			defer p.Shutdown()
			var completed []sdk.PlanCompletedEvent
			drainAssessmentPlan(t, p, p.Activate(Input{}), &completed)
			if p.Status() != sdk.StatusDone || p.ExitCode() != 2 {
				t.Fatalf("status=%v exit=%d; policy failure must preserve successful Plan review", p.Status(), p.ExitCode())
			}
			view := p.View(120, 40)
			if !strings.Contains(strings.ToLower(view), "risk unavailable") || !strings.Contains(view, "aws_db_instance.failed") || strings.Contains(view, "CRITICAL") {
				t.Fatalf("Plan review = %q; want unavailable assessment and retained rows without stale classifications", view)
			}
			if !strings.Contains(string(p.Stderr()), "risk unavailable:") {
				t.Fatalf("stderr=%q; want fixed assessment diagnostic", p.Stderr())
			}
			if len(completed) != 1 || completed[0].PlanFile == "" || completed[0].PlanFile != svc.PlanCalls[0].PlanFile {
				t.Fatalf("completed=%+v; want original Plan artifact preserved", completed)
			}
			req, ok := p.requestApply()().(ApplyRequestMsg)
			if !ok || req.PlanFile != completed[0].PlanFile {
				t.Fatalf("apply request=%+v; want unchanged Plan artifact handoff", req)
			}
		})
	}
}

func drainAssessmentPlan(t *testing.T, p *Plugin, cmd tea.Cmd, completed *[]sdk.PlanCompletedEvent) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			drainAssessmentPlan(t, p, sub, completed)
		}
		return
	}
	if ev, ok := msg.(sdk.PlanCompletedEvent); ok {
		if completed != nil {
			*completed = append(*completed, ev)
		}
		return
	}
	if _, ok := msg.(ui.TimerTickMsg); ok {
		return
	}
	_, next := p.Update(msg)
	drainAssessmentPlan(t, p, next, completed)
}

func TestActivate_GivenCachedPlanAndChangedPolicy_ShouldReassessWithoutPlanningAgain(t *testing.T) {
	low, critical := sdk.RiskLow, sdk.RiskCritical
	level := &low
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return &sdk.PlanSummary{ToDelete: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete}}}, nil
	}}
	h := sdktest.NewDeps(svc)
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
		return sdk.PolicyAssessment{Status: "complete", OverallLevel: level, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_db_instance.database", Status: "classified", Level: level}}}
	}
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	output, _ := p.Stdout()
	if !strings.Contains(string(output), "Risk: low") {
		t.Fatalf("initial output=%q; want initial selected-policy low", output)
	}
	level = &critical
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	output, _ = p.Stdout()
	if !strings.Contains(string(output), "Risk: critical") || len(svc.PlanCalls) != 1 {
		t.Fatalf("reactivated output=%q, Plan calls=%d; want refreshed classification over same reviewed plan", output, len(svc.PlanCalls))
	}
}
