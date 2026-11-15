package ui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
