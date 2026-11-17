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

func TestAssessRisk_GivenSourceChangedDuringEvaluation_ShouldRejectSnapshot(t *testing.T) {
	for _, change := range []string{"bytes", "selector", "configuration", "deleted"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
			writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"low\",\"reasons\":[]}\n")
			app, deps := newPolicyDiagnosticApp(t, root, "", true)
			got := app.assessRiskWith(context.Background(), deps.Context(), reviewedRiskSummary(), func(ctx context.Context, snapshot riskpolicy.Snapshot, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
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
				return riskpolicy.Assess(ctx, snapshot, evidence)
			})
			if got.Status != "unavailable" || got.ErrorCode != "source_changed" || got.OverallLevel != nil || len(got.Resources) != 0 {
				t.Fatalf("superseded source published: %+v", got)
			}
		})
	}
}

func TestAssessRisk_GivenCancellationOrInvalidConfiguration_ShouldReturnTerminalFailure(t *testing.T) {
	root := t.TempDir()
	app, deps := newPolicyDiagnosticApp(t, root, "", false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := app.assessRisk(ctx, deps.Context(), reviewedRiskSummary())
	if got.Status != "cancelled" || got.OverallLevel != nil {
		t.Fatalf("cancelled assessment = %+v", got)
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = `)
	got = app.assessRisk(context.Background(), deps.Context(), reviewedRiskSummary())
	if got.Status != "unavailable" || got.ErrorCode != "configuration_invalid" {
		t.Fatalf("invalid configuration = %+v", got)
	}
}

func TestRequestAssessment_GivenSourceChangedDuringEvaluation_ShouldPublishSourceChanged(t *testing.T) {
	for _, change := range []string{"bytes", "selector", "configuration", "deleted"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
			writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"low\",\"reasons\":[]}\n")
			app, _, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
			ev := requestRisk(t, &app, recorder, app.requestAssessmentWith(reviewedRiskSummary(), func(ctx context.Context, snapshot riskpolicy.Snapshot, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
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
				return riskpolicy.Assess(ctx, snapshot, evidence)
			}))
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
	cmd := app.requestAssessmentWith(reviewedRiskSummary(), func(ctx context.Context, snapshot riskpolicy.Snapshot, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
		evaluated = ctx.Err()
		return riskpolicy.Assess(ctx, snapshot, evidence)
	})
	drainPolicyAppCommand(&app, func() tea.Msg { return sdk.PlanInvalidatedEvent{} })
	result := cmd().(riskAssessedMsg)
	if !errors.Is(evaluated, context.Canceled) || result.event.Assessment.Status != "cancelled" || result.event.Assessment.OverallLevel != nil {
		t.Fatalf("invalidated request ran with ctx err %v and assessed %+v, want cancelled", evaluated, result.event.Assessment)
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = `)
	got := requestRisk(t, &app, recorder, app.requestAssessment(reviewedRiskSummary())).Assessment
	if got.Status != "unavailable" || got.ErrorCode != "configuration_invalid" {
		t.Fatalf("invalid configuration = %+v", got)
	}
}

func TestRequestAssessment_GivenInvalidationDuringHostPlan_ShouldReportCancellationWithoutEvaluation(t *testing.T) {
	svc := newMockService("default", nil)
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return reviewedRiskSummary(), nil }
	app, deps, _ := newAssessmentRequestApp(t, t.TempDir(), false, svc)
	cmd := deps.RequestAssessment(nil)
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

	ev := requestRisk(t, &app, recorder, deps.RequestAssessment(nil))
	if ev.Err == nil || ev.Assessment.Status != "" || len(svc.PlanCalls) != 0 {
		t.Fatalf("event = %+v with %d plan calls, want temp directory error and no plan", ev, len(svc.PlanCalls))
	}
}
