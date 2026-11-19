package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/riskassess"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func reviewedRiskSummary() *sdk.PlanSummary {
	resource := sdk.PolicyResource{Address: "aws_db_instance.database", Mode: "managed", Type: "aws_db_instance", Name: "database", ProviderName: "registry.terraform.io/hashicorp/aws", Index: sdk.PolicyValue{Kind: "absent"}}
	return &sdk.PlanSummary{PolicyEvidence: &sdk.PolicyEvidence{PlanID: "reviewed-plan", Resources: []sdk.PolicyResourceEvidence{{Address: resource.Address, Actions: []string{"delete"}, Input: &sdk.PolicyInput{SchemaVersion: "tfui.risk.input/v1", Resource: resource, Change: sdk.PolicyChange{Actions: []string{"delete"}, Before: sdk.PolicyValue{Kind: "null"}, After: sdk.PolicyValue{Kind: "null"}, ReplacePaths: [][]any{}}}}}}}
}

// requestRisk runs one host assessment request through the App and returns the
// event plugins receive.
func requestRisk(t *testing.T, app *App, recorder *riskEventRecorder, cmd tea.Cmd) sdk.RiskAssessedEvent {
	t.Helper()
	recorder.events = nil
	deliverAssessment(app, cmd)
	return onlyRiskEvent(t, recorder)
}

func switchContext(app *App, chdir, workspace string) {
	drainPolicyAppCommand(app, func() tea.Msg {
		return sdk.ContextSwitchRequestMsg{Chdir: sdk.Chdir(chdir), Workspace: sdk.NewWorkspace(workspace)}
	})
}

func TestRequestAssessment_GivenContextSwitchAfterRequest_ShouldEvaluateTheRequestContext(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "missing.rego" }`)
	writePolicyFile(t, root, "member/tfui.hcl", `risk_policy = ""`)
	app, deps, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
	switchContext(&app, "member", "default")
	cmd := cmdOf(deps.RequestAssessment(reviewedRiskSummary()))
	switchContext(&app, "", "default")

	result, ok := cmd().(riskassess.Result)
	if !ok {
		t.Fatalf("request produced %T, want riskassess.Result", result)
	}
	// The switch cancels superseded work; the policy it loaded is still the member's.
	if got := result.Event.Assessment; got.Policy.Kind != "builtin" || got.Status != sdk.AssessmentCancelled {
		t.Fatalf("superseded request did not keep the member Context it captured: %+v", got)
	}
	got := requestRisk(t, &app, recorder, cmdOf(deps.RequestAssessment(reviewedRiskSummary()))).Assessment
	if got.Status == "complete" || got.ErrorCode != "source_unreadable" {
		t.Fatalf("root Context missing policy should remain unavailable: %+v", got)
	}
}
