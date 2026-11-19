package ui

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/riskassess"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestRequestAssessment_GivenInvalidationOrInvalidConfiguration_ShouldCancelOrPublishTerminalFailure(t *testing.T) {
	root := t.TempDir()
	app, _, recorder := newAssessmentRequestApp(t, root, false, newMockService("default", nil))
	cmd := cmdOf(app.risk.Request("riskprobe", app.holder.current, reviewedRiskSummary()))
	drainPolicyAppCommand(&app, func() tea.Msg { return sdk.PlanInvalidatedEvent{} })
	result := cmd().(riskassess.Result)
	if result.Event.Assessment.Status != "cancelled" || result.Event.Assessment.OverallLevel != nil {
		t.Fatalf("invalidated request assessed %+v, want cancelled", result.Event.Assessment)
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = `)
	got := requestRisk(t, &app, recorder, cmdOf(app.risk.Request("riskprobe", app.holder.current, reviewedRiskSummary()))).Assessment
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

	ev := cmd().(riskassess.Result).Event
	if !errors.Is(ev.Err, context.Canceled) || ev.Summary != nil || ev.Assessment.Status != "" {
		t.Fatalf("cancelled host plan = %+v, want cancellation without summary or assessment", ev)
	}
}
