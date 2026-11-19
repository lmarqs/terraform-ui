package riskassess

import (
	"context"
	"errors"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/riskpolicy"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func failingPlanService(t *testing.T) *sdktest.MockService {
	t.Helper()
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		t.Error("Service.Plan ran although risk is unavailable")
		return nil, errors.New("plan must not run")
	}
	return svc
}

func TestRequestAssessment_GivenInvalidComparisonSettings_ShouldPublishUnavailableWithoutPlanOrEvaluation(t *testing.T) {
	for _, settings := range []map[string]interface{}{{"level": "high"}, {"threshold": "urgent"}} {
		svc := failingPlanService(t)
		h := newHarness(t, t.TempDir(), svc)
		h.coord.cfg.Plugins = map[string]config.PluginConfig{"risk": {Options: settings}}
		ev := requestRisk(t, h, h.request(nil))
		got := ev.Assessment
		if got.Status != "unavailable" || got.ErrorCode != "configuration_invalid" || got.OverallLevel != nil || len(got.Resources) != 0 || ev.Err != nil {
			t.Fatalf("invalid settings %v retained publishable classification or noncanonical code: %+v", settings, ev)
		}
		if want := sdk.DiagnosticHint(sdk.RiskSettingsError(settings)); ev.Hint != want {
			t.Fatalf("invalid settings %v hint = %q, want %q", settings, ev.Hint, want)
		}
		if _, legacy := settings["level"]; legacy && (sdk.DiagnosticError{Code: got.ErrorCode, Hint: ev.Hint}).Error() != sdk.RiskSettingsError(settings).Error() {
			t.Fatalf("legacy level event %+v does not carry the settings hint", ev)
		}
	}
}

func TestRequestAssessment_GivenSelectedSourceEdits_ShouldReloadClassificationAndProvenance(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	h := newHarness(t, root, &sdktest.MockService{})
	var previousDigest string
	for _, step := range []struct {
		level string
		want  sdk.RiskLevel
	}{{"low", sdk.RiskLow}, {"high", sdk.RiskHigh}} {
		writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\""+step.level+"\",\"reasons\":[{\"code\":\"project-default\",\"message\":\"Explicit project default.\",\"paths\":[]}]}\n")
		got := requestRisk(t, h, h.request(reviewedRiskSummary())).Assessment
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
	got := requestRisk(t, h, h.request(reviewedRiskSummary())).Assessment
	if got.Status == "complete" || got.OverallLevel != nil || (got.ErrorCode == "" && (len(got.Resources) != 1 || got.Resources[0].ErrorCode == "")) {
		t.Fatalf("invalid policy silently retained a successful classification: %+v", got)
	}
}

func TestRequestAssessment_GivenBuiltinSelectionOrMissingEvidence_ShouldDistinguishClassificationFromUnavailable(t *testing.T) {
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return &sdk.PlanSummary{}, nil }
	h := newHarness(t, t.TempDir(), svc)
	got := requestRisk(t, h, h.request(reviewedRiskSummary())).Assessment
	if got.Status != "complete" || got.Policy.Kind != "builtin" || got.OverallLevel == nil || *got.OverallLevel != sdk.RiskCritical {
		t.Fatalf("built-in database deletion = %+v", got)
	}
	for _, summary := range []*sdk.PlanSummary{nil, {Changes: []sdk.PlanChange{{Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}} {
		got = requestRisk(t, h, h.request(summary)).Assessment
		if got.Status == "complete" || got.OverallLevel != nil || got.ErrorCode == "" {
			t.Fatalf("missing evidence must be unavailable rather than neutral or display-risk fallback: %+v", got)
		}
	}
}

func TestRequestAssessment_GivenUnchangedEvidenceAndSource_ShouldReuseAssessmentUntilContextOrSourceChanges(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\n")
	h := newHarness(t, root, &sdktest.MockService{})
	calls := 0
	evaluator := func(context.Context, riskpolicy.Policy, sdk.PolicyEvidence) sdk.PolicyAssessment {
		calls++
		low := sdk.RiskLow
		return sdk.PolicyAssessment{Status: "complete", OverallLevel: &low}
	}
	summary := reviewedRiskSummary()
	for i := 0; i < 2; i++ {
		requestRisk(t, h, h.requestWith(summary, evaluator))
	}
	if calls != 1 {
		t.Fatalf("unchanged reviewed evidence evaluated %d times, want one shared assessment", calls)
	}
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\n# edited source\n")
	requestRisk(t, h, h.requestWith(summary, evaluator))
	if calls != 2 {
		t.Fatalf("same-path source edit did not reevaluate: %d evaluations", calls)
	}
	h.switchContext("", "default")
	requestRisk(t, h, h.requestWith(summary, evaluator))
	if calls != 3 {
		t.Fatalf("replaced Context reused previous Context assessment: %d evaluations", calls)
	}
}
