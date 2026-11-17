package ui

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/riskpolicy"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func riskAssessor(t *testing.T, deps *sdk.PluginDeps) func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
	t.Helper()
	if deps.AssessRisk == nil {
		t.Fatal("app must provide its selected-policy assessor to plugins")
	}
	return deps.AssessRisk
}

func TestAssessRisk_GivenInvalidComparisonSettings_ShouldMakeRiskUnavailableWithoutEvaluation(t *testing.T) {
	for _, settings := range []map[string]interface{}{{"level": "high"}, {"threshold": "urgent"}} {
		app, deps := newPolicyDiagnosticApp(t, t.TempDir(), "", false)
		app.cfg.Plugins = map[string]config.PluginConfig{"risk": {Options: settings}}
		got := app.assessRisk(context.Background(), deps.Context(), reviewedRiskSummary())
		if got.Status != "unavailable" || got.ErrorCode != "configuration_invalid" || got.OverallLevel != nil || len(got.Resources) != 0 {
			t.Fatalf("invalid settings %v retained publishable classification or noncanonical code: %+v", settings, got)
		}
		if want := sdk.RiskSettingsError(settings).Error(); app.riskPolicyDiagnostic() != want {
			t.Fatalf("settings diagnostic = %q, want actionable diagnostic %q", app.riskPolicyDiagnostic(), want)
		}
	}
}

func reviewedRiskSummary() *sdk.PlanSummary {
	resource := sdk.PolicyResource{Address: "aws_db_instance.database", Mode: "managed", Type: "aws_db_instance", Name: "database", ProviderName: "registry.terraform.io/hashicorp/aws", Index: sdk.PolicyValue{Kind: "absent"}}
	return &sdk.PlanSummary{PolicyEvidence: &sdk.PolicyEvidence{PlanID: "reviewed-plan", Resources: []sdk.PolicyResourceEvidence{{Address: resource.Address, Actions: []string{"delete"}, Input: &sdk.PolicyInput{SchemaVersion: "tfui.risk.input/v1", Resource: resource, Change: sdk.PolicyChange{Actions: []string{"delete"}, Before: sdk.PolicyValue{Kind: "null"}, After: sdk.PolicyValue{Kind: "null"}, ReplacePaths: [][]any{}}}}}}}
}

func TestAssessRisk_GivenSelectedSourceEdits_ShouldReloadClassificationAndProvenance(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	_, deps := newPolicyDiagnosticApp(t, root, "", true)
	assess := riskAssessor(t, deps)
	var previousDigest string
	for _, step := range []struct {
		level string
		want  sdk.RiskLevel
	}{{"low", sdk.RiskLow}, {"high", sdk.RiskHigh}} {
		writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\""+step.level+"\",\"reasons\":[{\"code\":\"project-default\",\"message\":\"Explicit project default.\",\"paths\":[]}]}\n")
		got := assess(context.Background(), deps.Context(), reviewedRiskSummary())
		if got.Status != "complete" || got.OverallLevel == nil || *got.OverallLevel != step.want {
			t.Fatalf("selected %s source assessment = %+v", step.level, got)
		}
		if got.PlanID != "reviewed-plan" || got.Policy.Kind != "custom" || got.Policy.Digest == "" || got.Policy.Digest == previousDigest {
			t.Fatalf("assessment did not retain fresh source provenance: %+v", got)
		}
		if len(got.Resources) != 1 || len(got.Resources[0].Reasons) != 1 || got.Resources[0].Reasons[0].Code != "project-default" {
			t.Fatalf("custom database classification gained hidden built-in behavior: %+v", got)
		}
		previousDigest = got.Policy.Digest
	}
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"not-a-level\",\"reasons\":[]}\n")
	got := assess(context.Background(), deps.Context(), reviewedRiskSummary())
	if got.Status == "complete" || got.OverallLevel != nil || (got.ErrorCode == "" && (len(got.Resources) != 1 || got.Resources[0].ErrorCode == "")) {
		t.Fatalf("invalid policy silently retained a successful classification: %+v", got)
	}
}

func TestAssessRisk_GivenBuiltinSelectionOrMissingEvidence_ShouldDistinguishClassificationFromUnavailable(t *testing.T) {
	_, deps := newPolicyDiagnosticApp(t, t.TempDir(), "", false)
	assess := riskAssessor(t, deps)
	got := assess(context.Background(), deps.Context(), reviewedRiskSummary())
	if got.Status != "complete" || got.Policy.Kind != "builtin" || got.OverallLevel == nil || *got.OverallLevel != sdk.RiskCritical {
		t.Fatalf("built-in database deletion = %+v", got)
	}
	for _, summary := range []*sdk.PlanSummary{nil, {Changes: []sdk.PlanChange{{Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}} {
		got = assess(context.Background(), deps.Context(), summary)
		if got.Status == "complete" || got.OverallLevel != nil || got.ErrorCode == "" {
			t.Fatalf("missing evidence must be unavailable rather than neutral or display-risk fallback: %+v", got)
		}
	}
}

func TestAssessRisk_GivenCapturedContext_ShouldUseItsPolicySelection(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "missing.rego" }`)
	writePolicyFile(t, root, "member/tfui.hcl", `risk_policy = ""`)
	_, deps := newPolicyDiagnosticApp(t, root, "", true)
	assess := riskAssessor(t, deps)
	captured := *deps.Context()
	captured.Chdir = "member"
	captured.WorkingDir = filepath.Join(root, "member")
	captured.Workspace = "default"
	got := assess(context.Background(), &captured, reviewedRiskSummary())
	if got.Status != "complete" || got.Policy.Kind != "builtin" {
		t.Fatalf("captured member Context did not select its reset policy: %+v", got)
	}
	got = assess(context.Background(), deps.Context(), reviewedRiskSummary())
	if got.Status == "complete" || got.ErrorCode != "source_unreadable" {
		t.Fatalf("root Context missing policy should remain unavailable: %+v", got)
	}
}

func TestAssessRisk_GivenUnchangedEvidenceAndSource_ShouldReuseAssessmentUntilContextOrSourceChanges(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\n")
	app, deps := newPolicyDiagnosticApp(t, root, "", true)
	calls := 0
	evaluator := func(context.Context, riskpolicy.Snapshot, sdk.PolicyEvidence) sdk.PolicyAssessment {
		calls++
		low := sdk.RiskLow
		return sdk.PolicyAssessment{Status: "complete", OverallLevel: &low}
	}
	current, summary := deps.Context(), reviewedRiskSummary()
	for i := 0; i < 2; i++ {
		app.assessRiskWith(context.Background(), current, summary, evaluator)
	}
	if calls != 1 {
		t.Fatalf("unchanged reviewed evidence evaluated %d times, want one shared assessment", calls)
	}
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\n# edited source\n")
	app.assessRiskWith(context.Background(), current, summary, evaluator)
	if calls != 2 {
		t.Fatalf("same-path source edit did not reevaluate: %d evaluations", calls)
	}
	next := *current
	app.assessRiskWith(context.Background(), &next, summary, evaluator)
	if calls != 3 {
		t.Fatalf("replaced Context reused previous Context assessment: %d evaluations", calls)
	}
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

func failingPlanService(t *testing.T) *sdktest.MockService {
	t.Helper()
	svc := newMockService("default", nil)
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		t.Error("Service.Plan ran although risk is unavailable")
		return nil, errors.New("plan must not run")
	}
	return svc
}

func TestRequestAssessment_GivenInvalidComparisonSettings_ShouldPublishUnavailableWithoutPlanOrEvaluation(t *testing.T) {
	for _, settings := range []map[string]interface{}{{"level": "high"}, {"threshold": "urgent"}} {
		svc := failingPlanService(t)
		app, _, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)
		app.cfg.Plugins = map[string]config.PluginConfig{"risk": {Options: settings}}
		ev := requestRisk(t, &app, recorder, app.requestAssessment(nil))
		got := ev.Assessment
		if got.Status != "unavailable" || got.ErrorCode != "configuration_invalid" || got.OverallLevel != nil || len(got.Resources) != 0 || ev.Err != nil {
			t.Fatalf("invalid settings %v retained publishable classification or noncanonical code: %+v", settings, ev)
		}
	}
}

func TestRequestAssessment_GivenSelectedSourceEdits_ShouldReloadClassificationAndProvenance(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	app, deps, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
	var previousDigest string
	for _, step := range []struct {
		level string
		want  sdk.RiskLevel
	}{{"low", sdk.RiskLow}, {"high", sdk.RiskHigh}} {
		writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\""+step.level+"\",\"reasons\":[{\"code\":\"project-default\",\"message\":\"Explicit project default.\",\"paths\":[]}]}\n")
		got := requestRisk(t, &app, recorder, deps.RequestAssessment(reviewedRiskSummary())).Assessment
		if got.Status != "complete" || got.OverallLevel == nil || *got.OverallLevel != step.want {
			t.Fatalf("selected %s source assessment = %+v", step.level, got)
		}
		if got.PlanID != "reviewed-plan" || got.Policy.Kind != "custom" || got.Policy.Digest == "" || got.Policy.Digest == previousDigest {
			t.Fatalf("assessment did not retain fresh source provenance: %+v", got)
		}
		if len(got.Resources) != 1 || len(got.Resources[0].Reasons) != 1 || got.Resources[0].Reasons[0].Code != "project-default" {
			t.Fatalf("custom database classification gained hidden built-in behavior: %+v", got)
		}
		previousDigest = got.Policy.Digest
	}
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"not-a-level\",\"reasons\":[]}\n")
	got := requestRisk(t, &app, recorder, deps.RequestAssessment(reviewedRiskSummary())).Assessment
	if got.Status == "complete" || got.OverallLevel != nil || (got.ErrorCode == "" && (len(got.Resources) != 1 || got.Resources[0].ErrorCode == "")) {
		t.Fatalf("invalid policy silently retained a successful classification: %+v", got)
	}
}

func TestRequestAssessment_GivenBuiltinSelectionOrMissingEvidence_ShouldDistinguishClassificationFromUnavailable(t *testing.T) {
	svc := newMockService("default", nil)
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return &sdk.PlanSummary{}, nil }
	app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)
	got := requestRisk(t, &app, recorder, deps.RequestAssessment(reviewedRiskSummary())).Assessment
	if got.Status != "complete" || got.Policy.Kind != "builtin" || got.OverallLevel == nil || *got.OverallLevel != sdk.RiskCritical {
		t.Fatalf("built-in database deletion = %+v", got)
	}
	for _, summary := range []*sdk.PlanSummary{nil, {Changes: []sdk.PlanChange{{Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}} {
		got = requestRisk(t, &app, recorder, deps.RequestAssessment(summary)).Assessment
		if got.Status == "complete" || got.OverallLevel != nil || got.ErrorCode == "" {
			t.Fatalf("missing evidence must be unavailable rather than neutral or display-risk fallback: %+v", got)
		}
	}
}

func TestRequestAssessment_GivenContextSwitchAfterRequest_ShouldEvaluateTheRequestContext(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "missing.rego" }`)
	writePolicyFile(t, root, "member/tfui.hcl", `risk_policy = ""`)
	app, deps, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
	switchContext(&app, "member", "default")
	cmd := deps.RequestAssessment(reviewedRiskSummary())
	switchContext(&app, "", "default")

	result, ok := cmd().(riskAssessedMsg)
	if !ok {
		t.Fatalf("request produced %T, want riskAssessedMsg", result)
	}
	// The switch cancels superseded work; the policy it loaded is still the member's.
	if got := result.event.Assessment; got.Policy.Kind != "builtin" || got.Status != sdk.AssessmentCancelled {
		t.Fatalf("superseded request did not keep the member Context it captured: %+v", got)
	}
	got := requestRisk(t, &app, recorder, deps.RequestAssessment(reviewedRiskSummary())).Assessment
	if got.Status == "complete" || got.ErrorCode != "source_unreadable" {
		t.Fatalf("root Context missing policy should remain unavailable: %+v", got)
	}
}

func TestRequestAssessment_GivenUnchangedEvidenceAndSource_ShouldReuseAssessmentUntilContextOrSourceChanges(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\n")
	app, _, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
	calls := 0
	evaluator := func(context.Context, riskpolicy.Snapshot, sdk.PolicyEvidence) sdk.PolicyAssessment {
		calls++
		low := sdk.RiskLow
		return sdk.PolicyAssessment{Status: "complete", OverallLevel: &low}
	}
	summary := reviewedRiskSummary()
	for i := 0; i < 2; i++ {
		requestRisk(t, &app, recorder, app.requestAssessmentWith(summary, evaluator))
	}
	if calls != 1 {
		t.Fatalf("unchanged reviewed evidence evaluated %d times, want one shared assessment", calls)
	}
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\n# edited source\n")
	requestRisk(t, &app, recorder, app.requestAssessmentWith(summary, evaluator))
	if calls != 2 {
		t.Fatalf("same-path source edit did not reevaluate: %d evaluations", calls)
	}
	switchContext(&app, "", "default")
	requestRisk(t, &app, recorder, app.requestAssessmentWith(summary, evaluator))
	if calls != 3 {
		t.Fatalf("replaced Context reused previous Context assessment: %d evaluations", calls)
	}
}
