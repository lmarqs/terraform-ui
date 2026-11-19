package riskassess

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/riskpolicy"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestRequestAssessment_GivenSourceChangedDuringEvaluation_ShouldPublishSourceChanged(t *testing.T) {
	for _, change := range []string{"bytes", "selector", "configuration", "deleted"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
			writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"low\",\"reasons\":[]}\n")
			h := newHarness(t, root, &sdktest.MockService{})
			ev := requestRisk(t, h, h.requestWith(reviewedRiskSummary(), func(ctx context.Context, policy riskpolicy.Policy, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
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
			}))
			got := ev.Assessment
			if got.Status != "unavailable" || got.ErrorCode != "source_changed" || got.OverallLevel != nil || len(got.Resources) != 0 {
				t.Fatalf("superseded source published: %+v", got)
			}
		})
	}
}

func TestRequestAssessment_GivenNoTempPlanDirectory_ShouldPublishErrorWithoutRunningPlan(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	svc := failingPlanService(t)
	h := newHarness(t, t.TempDir(), svc)

	ev := requestRisk(t, h, h.request(nil))
	if ev.Err == nil || ev.Assessment.Status != "" || len(svc.PlanCalls) != 0 {
		t.Fatalf("event = %+v with %d plan calls, want temp directory error and no plan", ev, len(svc.PlanCalls))
	}
}
