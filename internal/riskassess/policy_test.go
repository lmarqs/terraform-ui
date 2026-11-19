package riskassess

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestRequestAssessment_GivenChangedSourceAtSamePath_ShouldReloadWithoutFallback(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	writePolicyFile(t, root, "policy.rego", "package example\n")
	h := newHarness(t, root, &sdktest.MockService{})
	for _, step := range []struct {
		name, source string
		want         sdk.ErrorCode
		deleted      bool
	}{
		{"initial source", "package example\n", "", false},
		{"same path emptied", "", sdk.ErrorSourceInvalid, false},
		{"same path restored", "package changed\n", "", false},
		{"source deleted", "", sdk.ErrorSourceUnreadable, true},
		{"source recreated", "package recreated\n", "", false},
	} {
		t.Run(step.name, func(t *testing.T) {
			if step.deleted {
				if err := os.Remove(filepath.Join(root, "policy.rego")); err != nil {
					t.Fatal(err)
				}
			} else {
				writePolicyFile(t, root, "policy.rego", step.source)
			}
			got := requestRisk(t, h, h.request(reviewedRiskSummary())).Assessment
			if got.ErrorCode != step.want || (step.want != "" && got.Status != sdk.AssessmentUnavailable) {
				t.Fatalf("assessment = %+v, want error code %q", got, step.want)
			}
		})
	}
}

func TestRequestAssessment_GivenContextAndConfigurationChanges_ShouldReadCurrentSelection(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "root.rego" }`)
	writePolicyFile(t, root, "root.rego", "package root\n")
	writePolicyFile(t, root, "a/tfui.hcl", `risk_policy = "missing.rego"
workspace "dev" { risk_policy = "" }`)
	writePolicyFile(t, root, "b/tfui.hcl", `risk_policy = "b.rego"`)
	writePolicyFile(t, root, "b/b.rego", "package member\n")
	h := newHarness(t, root, &sdktest.MockService{})
	request := func() sdk.ErrorCode {
		return requestRisk(t, h, h.request(reviewedRiskSummary())).Assessment.ErrorCode
	}
	for _, step := range []struct {
		chdir, workspace string
		want             sdk.ErrorCode
	}{
		{"a", "default", sdk.ErrorSourceUnreadable}, {"a", "dev", ""}, {"a", "prod", sdk.ErrorSourceUnreadable}, {"b", "prod", ""}, {"", "prod", ""},
	} {
		h.switchContext(step.chdir, step.workspace)
		if got := request(); got != step.want {
			t.Fatalf("%s/%s error code = %q, want %q", step.chdir, step.workspace, got, step.want)
		}
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = null }`)
	if got := request(); got != sdk.ErrorConfigurationInvalid {
		t.Fatalf("edited root selector error code = %q", got)
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "" }`)
	if got := request(); got != "" {
		t.Fatalf("reset to builtin error code = %q", got)
	}
}

func TestRequestAssessment_GivenNoSelectorOrExplicitReset_ShouldUseBuiltin(t *testing.T) {
	for _, tc := range []struct {
		name, root, member string
		loaded             bool
	}{
		{"zero config ignores member selector", "", `risk_policy = "missing.rego"`, false},
		{"absent selector", `member "member" {}`, "", true},
		{"explicit reset", `defaults { risk_policy = "missing.rego" }`, `risk_policy = ""`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.loaded {
				writePolicyFile(t, root, "tfui.hcl", tc.root)
			}
			writePolicyFile(t, root, "member/tfui.hcl", tc.member)
			h := newHarness(t, root, &sdktest.MockService{})
			h.switchContext("member", "default")
			got := requestRisk(t, h, h.request(reviewedRiskSummary())).Assessment
			if !got.Usable() || got.Policy.Kind != "builtin" {
				t.Fatalf("builtin selection assessment = %+v", got)
			}
		})
	}
}

func TestErrorCode_GivenErrorWithoutCode_ShouldUseFallback(t *testing.T) {
	if got := errorCode(errors.New("uncoded"), sdk.ErrorSourceUnreadable); got != sdk.ErrorSourceUnreadable {
		t.Fatalf("uncoded error = %q, want fallback", got)
	}
	if got := errorCode(fmt.Errorf("wrapped: %w", sdk.ErrorSourceInvalid), sdk.ErrorSourceUnreadable); got != sdk.ErrorSourceInvalid {
		t.Fatalf("coded error = %q, want its own code", got)
	}
}
