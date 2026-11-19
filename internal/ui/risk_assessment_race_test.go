package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/riskpolicy"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestRequestAssessment_GivenSourceChangedDuringEvaluation_ShouldPublishSourceChanged(t *testing.T) {
	for _, change := range []string{"bytes", "selector", "configuration", "deleted"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
			writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"low\",\"reasons\":[]}\n")
			app, _, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
			ev := requestRisk(t, &app, recorder, cmdOf(app.requestAssessmentWith(reviewedRiskSummary(), func(ctx context.Context, policy riskpolicy.Policy, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
				switch change {
				case "bytes":
					writePolicyFile(t, root, "policy.rego", "package changed\n")
				case "selector":
					writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "" }`)
				case "configuration":
					writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = `)
				case "deleted":
					if err := os.Remove(filepath.Join(root, "policy.rego")); err != nil {
						t.Fatal(err)
					}
				}
				return policy.Assess(ctx, evidence)
			})))
			got := ev.Assessment
			if got.Status != "unavailable" || got.ErrorCode != "source_changed" || got.OverallLevel != nil || len(got.Resources) != 0 {
				t.Fatalf("superseded source published: %+v", got)
			}
		})
	}
}

func TestRequestAssessment_GivenInvalidationOrInvalidConfiguration_ShouldCancelOrPublishTerminalFailure(t *testing.T) {
	root := t.TempDir()
	app, _, recorder := newAssessmentRequestApp(t, root, false, newMockService("default", nil))
	var evaluated error
	cmd := cmdOf(app.requestAssessmentWith(reviewedRiskSummary(), func(ctx context.Context, policy riskpolicy.Policy, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
		evaluated = ctx.Err()
		return policy.Assess(ctx, evidence)
	}))
	drainPolicyAppCommand(&app, func() tea.Msg { return sdk.PlanInvalidatedEvent{} })
	result := cmd().(riskAssessedMsg)
	if !errors.Is(evaluated, context.Canceled) || result.event.Assessment.Status != "cancelled" || result.event.Assessment.OverallLevel != nil {
		t.Fatalf("invalidated request ran with ctx err %v and assessed %+v, want cancelled", evaluated, result.event.Assessment)
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = `)
	got := requestRisk(t, &app, recorder, cmdOf(app.requestAssessment(reviewedRiskSummary()))).Assessment
	if got.Status != "unavailable" || got.ErrorCode != "configuration_invalid" {
		t.Fatalf("invalid configuration = %+v", got)
	}
}

func TestRequestAssessment_GivenInvalidationDuringHostPlan_ShouldReportCancellationWithoutEvaluation(t *testing.T) {
	svc := newMockService("default", nil)
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return reviewedRiskSummary(), nil }
	app, deps, _ := newAssessmentRequestApp(t, t.TempDir(), false, svc)
	cmd := cmdOf(deps.RequestAssessment(nil))
	drainPolicyAppCommand(&app, func() tea.Msg { return sdk.PlanInvalidatedEvent{} })

	ev := cmd().(riskAssessedMsg).event
	if !errors.Is(ev.Err, context.Canceled) || ev.Summary != nil || ev.Assessment.Status != "" {
		t.Fatalf("cancelled host plan = %+v, want cancellation without summary or assessment", ev)
	}
}

func TestRequestAssessment_GivenNoTempPlanDirectory_ShouldPublishErrorWithoutRunningPlan(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	svc := failingPlanService(t)
	app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)

	ev := requestRisk(t, &app, recorder, cmdOf(deps.RequestAssessment(nil)))
	if ev.Err == nil || ev.Assessment.Status != "" || len(svc.PlanCalls) != 0 {
		t.Fatalf("event = %+v with %d plan calls, want temp directory error and no plan", ev, len(svc.PlanCalls))
	}
}
