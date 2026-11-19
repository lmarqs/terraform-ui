package riskassess

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestRequestAssessment_GivenBrokenPolicyAndNilSummary_ShouldPublishUnavailableWithoutRunningPlan(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "missing.rego" }`)
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		t.Error("Service.Plan ran although the selected policy cannot be loaded")
		return nil, errors.New("plan must not run")
	}
	h := newHarness(t, root, svc)

	h.deliver(h.request(nil))

	ev := onlyRiskEvent(t, h)
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Service.Plan called %d times, want 0", len(svc.PlanCalls))
	}
	if ev.Err != nil {
		t.Fatalf("policy load failure surfaced as plan error: %v", ev.Err)
	}
	if ev.Assessment.Status != sdk.AssessmentUnavailable || ev.Assessment.ErrorCode != sdk.ErrorSourceUnreadable || ev.Assessment.OverallLevel != nil {
		t.Fatalf("assessment = %+v, want unavailable with source_unreadable", ev.Assessment)
	}
}

func TestRequestAssessment_GivenNilSummaryAndUncompilablePolicy_ShouldReportCompilationFailedWithoutPlanning(t *testing.T) {
	tests := []struct {
		name   string
		policy string
	}{
		{"SyntaxError", "package tfui.risk.v1\nimport rego.v1\nclassification := {"},
		{"BuiltinOutsideCapabilities", "package tfui.risk.v1\nimport rego.v1\nclassification := http.send({\"method\": \"get\", \"url\": \"http://example.invalid\"})\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
			writePolicyFile(t, root, "policy.rego", tt.policy)
			svc := &sdktest.MockService{}
			svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
				return reviewedRiskSummary(), nil
			}
			h := newHarness(t, root, svc)

			h.deliver(h.request(nil))

			ev := onlyRiskEvent(t, h)
			if len(svc.PlanCalls) != 0 {
				t.Errorf("Service.Plan called %d times, want 0 for a policy that cannot compile", len(svc.PlanCalls))
			}
			if ev.Err != nil {
				t.Fatalf("compilation failure surfaced as plan error: %v", ev.Err)
			}
			if ev.Assessment.Status != sdk.AssessmentUnavailable || ev.Assessment.ErrorCode != sdk.ErrorCompilationFailed {
				t.Errorf("assessment = %+v, want unavailable with compilation_failed", ev.Assessment)
			}
			if ev.Assessment.Policy.Digest == "" {
				t.Error("assessment lost the policy identity, want a non-empty digest")
			}
		})
	}
}

func TestRequestAssessment_GivenValidPolicyAndNilSummary_ShouldPlanOnceAndPublishThatSummary(t *testing.T) {
	planned := reviewedRiskSummary()
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return planned, nil }
	h := newHarness(t, t.TempDir(), svc)

	h.deliver(h.request(nil))

	ev := onlyRiskEvent(t, h)
	if len(svc.PlanCalls) != 1 {
		t.Fatalf("Service.Plan called %d times, want 1", len(svc.PlanCalls))
	}
	if svc.PlanCalls[0].PlanFile == "" {
		t.Fatal("host plan fetch did not write to a temp plan file")
	}
	if ev.Err != nil || ev.Summary != planned {
		t.Fatalf("event summary = %p err = %v, want planned summary %p", ev.Summary, ev.Err, planned)
	}
	if !ev.Assessment.Usable() || ev.Assessment.PlanID != "reviewed-plan" {
		t.Fatalf("assessment of planned summary = %+v, want usable for reviewed-plan", ev.Assessment)
	}
}

func TestRequestAssessment_GivenSummary_ShouldEvaluateItWithoutRunningPlan(t *testing.T) {
	given := reviewedRiskSummary()
	svc := &sdktest.MockService{}
	h := newHarness(t, t.TempDir(), svc)

	h.deliver(h.request(given))

	ev := onlyRiskEvent(t, h)
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Service.Plan called %d times for a supplied summary, want 0", len(svc.PlanCalls))
	}
	if ev.Summary != given || ev.Err != nil || !ev.Assessment.Usable() {
		t.Fatalf("event = %+v, want usable assessment of the supplied summary", ev)
	}
}

func TestRequestAssessment_GivenPlanFailure_ShouldPublishErrorWithoutEvaluation(t *testing.T) {
	planErr := errors.New("terraform plan failed")
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return nil, planErr }
	h := newHarness(t, t.TempDir(), svc)

	h.deliver(h.request(nil))

	ev := onlyRiskEvent(t, h)
	if !errors.Is(ev.Err, planErr) {
		t.Fatalf("event err = %v, want %v", ev.Err, planErr)
	}
	if ev.Assessment.Status != "" || ev.Assessment.Usable() {
		t.Fatalf("plan failure was evaluated: %+v", ev.Assessment)
	}
}

func TestRequestAssessment_GivenNilSummary_ShouldPlanWithActiveContextOptionsAndRemoveTemporaryArtifact(t *testing.T) {
	var artifact string
	svc := &sdktest.MockService{}
	svc.PlanFn = func(_ context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
		artifact = opts.PlanFile
		if err := os.WriteFile(artifact, []byte("saved plan"), 0600); err != nil {
			t.Fatal(err)
		}
		return reviewedRiskSummary(), nil
	}
	h := newHarness(t, t.TempDir(), svc)

	requestRisk(t, h, h.request(nil))

	if len(svc.PlanCalls) != 1 {
		t.Fatalf("Service.Plan called %d times, want 1", len(svc.PlanCalls))
	}
	got, want := svc.PlanCalls[0], h.current.PlanOptions()
	got.PlanFile, got.Writer = "", nil
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("plan options = %#v, want active Context %#v", got, want)
	}
	if artifact == "" {
		t.Fatal("host plan must supply a temporary artifact instead of Terraform's tfplan.out default")
	}
	if _, err := os.Stat(artifact); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary artifact remains: %v", err)
	}
}
