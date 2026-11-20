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

func TestRequestAssessment_GivenCachedAssessmentAndSummary_ShouldReturnItWithoutCompilingPolicy(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	compiles := 0
	h.coord.compile = func(ctx context.Context, snapshot riskpolicy.Snapshot) riskpolicy.Policy {
		compiles++
		return riskpolicy.Compile(ctx, snapshot)
	}
	summary := reviewedRiskSummary()
	first := requestRisk(t, h, h.request(summary))
	if !first.Assessment.Usable() {
		t.Fatalf("first assessment %+v is unusable, so it is not cached", first.Assessment)
	}
	second := requestRisk(t, h, h.request(summary))
	if compiles != 1 {
		t.Fatalf("second identical request compiled the policy: %d compiles, want 1", compiles)
	}
	if second.Assessment.Status != first.Assessment.Status || second.Assessment.ErrorCode != first.Assessment.ErrorCode {
		t.Fatalf("cached assessment %+v differs from the first %+v", second.Assessment, first.Assessment)
	}
}

func TestRequestAssessment_GivenCachedAssessmentAndSourceEdit_ShouldCompileAgain(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\n")
	h := newHarness(t, root, &sdktest.MockService{})
	compiles := 0
	h.coord.compile = func(ctx context.Context, snapshot riskpolicy.Snapshot) riskpolicy.Policy {
		compiles++
		return riskpolicy.Compile(ctx, snapshot)
	}
	summary := reviewedRiskSummary()
	requestRisk(t, h, h.request(summary))
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\n# edited source\n")
	requestRisk(t, h, h.request(summary))
	if compiles != 2 {
		t.Fatalf("an edited source compiled %d times, want 2: the source is re-read at every boundary", compiles)
	}
}

func TestRequestAssessment_GivenCachedAssessmentAndNilSummary_ShouldCompileBeforePlanning(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\n")
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return reviewedRiskSummary(), nil }
	h := newHarness(t, root, svc)
	compiles := 0
	h.coord.compile = func(ctx context.Context, snapshot riskpolicy.Snapshot) riskpolicy.Policy {
		compiles++
		return riskpolicy.Compile(ctx, snapshot)
	}
	requestRisk(t, h, h.request(reviewedRiskSummary()))
	requestRisk(t, h, h.request(nil))
	if compiles != 2 {
		t.Fatalf("a nil-summary request compiled %d times in total, want 2: it compiles before planning", compiles)
	}
}

func TestRequestAssessment_GivenUnusableAssessment_ShouldEvaluateAgainOnTheNextRequest(t *testing.T) {
	for _, status := range []sdk.AssessmentStatus{sdk.AssessmentPartial, sdk.AssessmentUnavailable} {
		t.Run(string(status), func(t *testing.T) {
			h := newHarness(t, t.TempDir(), &sdktest.MockService{})
			calls := 0
			evaluator := func(context.Context, riskpolicy.Policy, sdk.PolicyEvidence) sdk.PolicyAssessment {
				calls++
				if calls == 1 {
					return sdk.PolicyAssessment{Status: status}
				}
				low := sdk.RiskLow
				return sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &low}
			}
			summary := reviewedRiskSummary()

			first := requestRisk(t, h, h.requestWith(summary, evaluator))
			second := requestRisk(t, h, h.request(summary))

			if calls != 2 {
				t.Fatalf("the %s assessment was served from the cache: %d evaluations, want 2", status, calls)
			}
			if first.Assessment.Usable() || !second.Assessment.Usable() {
				t.Fatalf("first = %+v, second = %+v, want an unusable then a usable assessment", first.Assessment, second.Assessment)
			}
		})
	}
}

func TestRequestAssessment_GivenCompleteAssessment_ShouldEvaluateOnceForRepeatedRequests(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	calls := 0
	evaluator := func(context.Context, riskpolicy.Policy, sdk.PolicyEvidence) sdk.PolicyAssessment {
		calls++
		low := sdk.RiskLow
		return sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &low}
	}
	summary := reviewedRiskSummary()

	requestRisk(t, h, h.requestWith(summary, evaluator))
	requestRisk(t, h, h.request(summary))
	requestRisk(t, h, h.request(summary))

	if calls != 1 {
		t.Fatalf("a complete assessment evaluated %d times for three requests, want 1", calls)
	}
}

func TestRequestAssessment_GivenRetryOfUnusableAssessment_ShouldEvaluateTheReviewedPlanWithoutPlanning(t *testing.T) {
	planned := reviewedRiskSummary()
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return planned, nil }
	h := newHarness(t, t.TempDir(), svc)
	var evidence []string
	evaluator := func(_ context.Context, _ riskpolicy.Policy, e sdk.PolicyEvidence) sdk.PolicyAssessment {
		evidence = append(evidence, e.PlanID)
		if len(evidence) == 1 {
			return sdk.PolicyAssessment{Status: sdk.AssessmentPartial}
		}
		low := sdk.RiskLow
		return sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &low}
	}

	first := requestRisk(t, h, h.requestWith(nil, evaluator))
	retry := requestRisk(t, h, h.request(first.Summary))

	if len(svc.PlanCalls) != 1 {
		t.Fatalf("Service.Plan ran %d times, want 1: the retry holds the reviewed plan", len(svc.PlanCalls))
	}
	if len(evidence) != 2 || evidence[0] != "reviewed-plan" || evidence[1] != "reviewed-plan" {
		t.Fatalf("evaluated plans = %v, want the reviewed plan twice", evidence)
	}
	if retry.Summary != planned || !retry.Assessment.Usable() {
		t.Fatalf("retry = %+v, want a usable assessment of the reviewed plan", retry)
	}
}
