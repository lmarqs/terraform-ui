package riskpolicy

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const projectDefault = `default classification := {"level":"low","reasons":[{"code":"project-default","message":"Explicit project default.","paths":[]}]}`

func evaluationSnapshot(t *testing.T, body string) Snapshot {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.rego")
	writePolicyModule(t, path, "package tfui.risk.v1\nimport rego.v1\n"+body+"\n")
	snapshot, err := LoadSource(config.RiskPolicySelection{Path: path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func evaluationResource(address, resourceType string) sdk.PolicyResourceEvidence {
	return sdk.PolicyResourceEvidence{
		Address: address, Actions: []string{"update"},
		Input: &sdk.PolicyInput{
			SchemaVersion: "tfui.risk.input/v1",
			Resource:      sdk.PolicyResource{Address: address, Mode: "managed", Type: resourceType, Name: "worker", ProviderName: "registry.terraform.io/hashicorp/aws", Index: sdk.PolicyValue{Kind: "absent"}},
			Change:        sdk.PolicyChange{Actions: []string{"update"}, ReplacePaths: [][]any{}, Before: sdk.PolicyValue{Kind: "null"}, After: sdk.PolicyValue{Kind: "null"}},
		},
	}
}

func TestAssess_GivenCompleteCustomPolicy_ShouldClassifyEachResourceWithoutHiddenDefaults(t *testing.T) {
	snapshot := evaluationSnapshot(t, projectDefault+`
classification := {"level":"high","reasons":[{"code":"lambda-update","message":"Lambda mutation.","paths":[]}]} if {
    input.resource.type == "aws_lambda_function"
    count(object.keys(input)) == 3
    input.schema_version == "tfui.risk.input/v1"
}
`)
	lambda := evaluationResource("aws_lambda_function.worker", "aws_lambda_function")
	database := evaluationResource("aws_db_instance.worker", "aws_db_instance")
	first := Assess(context.Background(), snapshot, sdk.PolicyEvidence{PlanID: "sha256:reviewed", Resources: []sdk.PolicyResourceEvidence{lambda}})
	combined := Assess(context.Background(), snapshot, sdk.PolicyEvidence{PlanID: "sha256:reviewed", Resources: []sdk.PolicyResourceEvidence{lambda, database}})
	if first.Status != "complete" || combined.Status != "complete" || len(first.Resources) != 1 || len(combined.Resources) != 2 {
		t.Fatalf("assessments = %+v / %+v; want complete independent classifications", first, combined)
	}
	if !reflect.DeepEqual(first.Resources[0], combined.Resources[0]) {
		t.Fatalf("neighbor changed Lambda classification: %+v / %+v", first.Resources[0], combined.Resources[0])
	}
	for _, row := range combined.Resources {
		want := sdk.RiskLow
		if row.Address == lambda.Address {
			want = sdk.RiskHigh
		}
		if row.Status != "classified" || row.Level == nil || *row.Level != want || len(row.Reasons) != 1 {
			t.Errorf("classification for %s = %+v, want selected-policy level %s and reason", row.Address, row, want)
		}
	}
	if combined.PlanID != "sha256:reviewed" || combined.Policy.Kind != "custom" || combined.Policy.ID != snapshot.ID() || combined.Policy.Digest != snapshot.Digest() {
		t.Fatalf("assessment lost reviewed plan/source provenance: %+v", combined)
	}
}

func TestAssess_GivenPureBuiltins_ShouldReturnAuthoredClassification(t *testing.T) {
	snapshot := evaluationSnapshot(t, `classification := {"level":"medium","reasons":[{"code":"pure-builtins","message":message,"paths":[]}]} if {
    startswith(input.resource.type, "aws_")
    count(object.keys(input.resource)) > 0
    message := concat(" ", [upper("pure"), lower("BUILTINS")])
}`)
	assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function")}})
	if assessment.Status != "complete" || len(assessment.Resources) != 1 || len(assessment.Resources[0].Reasons) != 1 || assessment.Resources[0].Reasons[0].Message != "PURE builtins" {
		t.Fatalf("allowed pure builtins did not classify: %+v", assessment)
	}
}

func TestAssess_GivenForbiddenCallsInUnqueriedModules_ShouldRejectWholeSource(t *testing.T) {
	for _, expression := range []string{
		`http.send({"method":"GET","url":"http://127.0.0.1"})`,
		`net.lookup_ip_addr("localhost")`,
		`time.now_ns()`,
		`rand.intn("seed", 10)`,
		`uuid.rfc4122("seed")`,
		`opa.runtime()`,
		`print("private source constant")`,
		`trace("private source constant")`,
		`1 + 2`,
	} {
		t.Run(expression, func(t *testing.T) {
			dir := t.TempDir()
			writePolicyModule(t, filepath.Join(dir, "main.rego"), "package tfui.risk.v1\nimport rego.v1\n"+projectDefault)
			writePolicyModule(t, filepath.Join(dir, "unused.rego"), "package unused\nimport rego.v1\nhidden if { "+expression+"; true }")
			snapshot, err := LoadSource(config.RiskPolicySelection{Path: dir}, nil)
			if err != nil {
				t.Fatal(err)
			}
			assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function")}})
			if assessment.Status != "unavailable" || assessment.ErrorCode != "compilation_failed" || assessment.OverallLevel != nil {
				t.Fatalf("forbidden unqueried call yielded %+v; want fixed compilation failure", assessment)
			}
		})
	}
}

func TestAssess_GivenExecutedBuiltinFailure_ShouldKeepOtherRowsAndUseFixedDiagnostic(t *testing.T) {
	snapshot := evaluationSnapshot(t, projectDefault+`
classification := {"level":"high","reasons":[{"code":"number","message":"Number.","paths":[]}]} if {
    input.resource.type == "aws_lambda_function"
    to_number("private source constant") == 7
}`)
	assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{
		evaluationResource("aws_lambda_function.worker", "aws_lambda_function"),
		evaluationResource("aws_db_instance.worker", "aws_db_instance"),
	}})
	if assessment.Status != "partial" || assessment.OverallLevel != nil || len(assessment.Resources) != 2 {
		t.Fatalf("strict executed builtin failure should preserve partial review: %+v", assessment)
	}
	for _, row := range assessment.Resources {
		if row.Address == "aws_lambda_function.worker" {
			if row.Status != "unavailable" || row.ErrorCode != "evaluation_failed" || row.Level != nil || len(row.Reasons) != 0 {
				t.Errorf("failed row leaked successful classification or diagnostic: %+v", row)
			}
		} else if row.Status != "classified" || row.Level == nil || *row.Level != sdk.RiskLow {
			t.Errorf("unreachable builtin error prevented independent default: %+v", row)
		}
	}
	if strings.Contains(fmt.Sprintf("%+v", assessment), "private source constant") {
		t.Fatal("raw evaluator diagnostic disclosed module contents")
	}
}

func TestAssess_GivenUndefinedOrMalformedClassification_ShouldNeverReturnSuccessfulNone(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"undefined", `other := true`, "result_undefined"},
		{"null", `classification := null`, "result_invalid"},
		{"empty object", `classification := {}`, "result_invalid"},
		{"unknown level", `classification := {"level":"safe","reasons":[]}`, "result_invalid"},
		{"empty reasons", `classification := {"level":"none","reasons":[]}`, "result_invalid"},
		{"conflict", `classification := {"level":"low","reasons":[]} if {input.resource.mode == "managed"}
classification := {"level":"high","reasons":[]} if {input.resource.name == "worker"}`, "evaluation_conflict"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := evaluationSnapshot(t, tc.body)
			assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function")}})
			if assessment.Status != "unavailable" || assessment.OverallLevel != nil || len(assessment.Resources) != 1 {
				t.Fatalf("invalid output yielded successful or missing assessment: %+v", assessment)
			}
			row := assessment.Resources[0]
			if row.Status != "unavailable" || row.ErrorCode != sdk.ErrorCode(tc.code) || row.Level != nil || len(row.Reasons) != 0 {
				t.Fatalf("resource=%+v; want unavailable %s without neutral classification", row, tc.code)
			}
		})
	}
}

func TestAssess_GivenCancelledContext_ShouldReturnNoPublishableClassifications(t *testing.T) {
	snapshot := evaluationSnapshot(t, projectDefault)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assessment := Assess(ctx, snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function")}})
	if assessment.Status != "cancelled" || assessment.OverallLevel != nil || len(assessment.Resources) != 0 {
		t.Fatalf("cancelled operation returned publishable assessment: %+v", assessment)
	}
}

func TestAssess_GivenExcludedChanges_ShouldKeepExclusionsSeparateFromExplicitNone(t *testing.T) {
	snapshot := evaluationSnapshot(t, `classification := {"level":"none","reasons":[{"code":"explicit-default","message":"Explicit none classification.","paths":[]}]}`)
	excluded := sdk.PolicyResourceEvidence{Address: "aws_db_instance.existing", Actions: []string{"no-op"}, Exclusion: "no_op"}
	empty := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{excluded}})
	if empty.Status != "empty" || empty.OverallLevel != nil || len(empty.Resources) != 1 || empty.Resources[0].Status != "not_assessed" || empty.Resources[0].Exclusion != "no_op" || empty.Resources[0].Level != nil {
		t.Fatalf("excluded-only evidence became a neutral classification: %+v", empty)
	}
	complete := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function"), excluded}})
	if complete.Status != "complete" || complete.OverallLevel == nil || *complete.OverallLevel != sdk.RiskNone {
		t.Fatalf("explicit none should be a complete classification: %+v", complete)
	}
}

func TestAssess_GivenInvalidEvidenceOrEmptyPlanWithBrokenPolicy_ShouldRemainUnavailable(t *testing.T) {
	valid := evaluationSnapshot(t, projectDefault)
	invalid := evaluationSnapshot(t, `classification :=`)
	for _, tc := range []struct {
		name     string
		snapshot Snapshot
		evidence sdk.PolicyEvidence
		code     string
	}{
		{"input incomplete", valid, sdk.PolicyEvidence{ErrorCode: "input_incomplete"}, "input_incomplete"},
		{"input invalid", valid, sdk.PolicyEvidence{ErrorCode: "input_invalid"}, "input_invalid"},
		{"compile empty", invalid, sdk.PolicyEvidence{}, "compilation_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assessment := Assess(context.Background(), tc.snapshot, tc.evidence)
			if assessment.Status != "unavailable" || assessment.ErrorCode != sdk.ErrorCode(tc.code) || assessment.OverallLevel != nil {
				t.Fatalf("invalid plan/source became an empty success: %+v", assessment)
			}
		})
	}
}

func TestAssess_GivenDeposedEvidence_ShouldCopyDeposedKeyToRows(t *testing.T) {
	snapshot := evaluationSnapshot(t, projectDefault)
	current := evaluationResource("aws_instance.web", "aws_instance")
	deposed := evaluationResource("aws_instance.web", "aws_instance")
	deposed.Deposed = "0a1b2c3d"
	excluded := sdk.PolicyResourceEvidence{Address: "aws_instance.web", Deposed: "ffffffff", Actions: []string{"forget"}, Exclusion: sdk.ExclusionForget}

	got := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{current, deposed, excluded}})

	var keys []string
	for _, row := range got.Resources {
		keys = append(keys, row.Deposed)
	}
	if want := []string{"", "0a1b2c3d", "ffffffff"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("row deposed keys = %q, want %q", keys, want)
	}
}
