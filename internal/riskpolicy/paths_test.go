package riskpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func acquiredEvidence(t *testing.T, change string) *sdk.PlanSummary {
	t.Helper()
	source := `{"format_version":"1.2","resource_changes":[{"address":"aws_lambda_function.worker","mode":"managed","type":"aws_lambda_function","name":"worker","provider_name":"registry.terraform.io/hashicorp/aws","change":` + change + `}]}`
	summary, err := terraform.LoadPlan([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if summary.PolicyEvidence.ErrorCode != "" {
		t.Fatalf("evidence unavailable: %+v", summary.PolicyEvidence)
	}
	return summary
}

func TestAssess_GivenReasonPaths_ShouldResolveResourceNodesWithoutTraversingOpaqueValues(t *testing.T) {
	summary := acquiredEvidence(t, `{"actions":["update"],"before":{"nested":{"items":[null,{"name":"old"}]},"secret":{"token":"private"}},"after":{"nested":{"items":[null,{"name":"new"}]},"secret":{"token":"private"}},"after_unknown":{"future":true,"both":true},"after_sensitive":{"secret":true,"both":true}}`)
	for _, tc := range []struct {
		name, path string
		valid      bool
	}{
		{"before root", `["before"]`, true}, {"after root", `["after"]`, true},
		{"object", `["after","nested"]`, true}, {"list", `["after","nested","items"]`, true},
		{"null item", `["after","nested","items",0]`, true}, {"scientific integral index", `["after","nested","items",1e0]`, true}, {"index exceeding int64", `["after","nested","items",9223372036854775808]`, false}, {"list object child", `["after","nested","items",1,"name"]`, true},
		{"unknown node", `["after","future"]`, true}, {"sensitive node", `["after","secret"]`, true}, {"sensitive unknown node", `["after","both"]`, true},
		{"before independent mask", `["before","secret","token"]`, true},
		{"missing key", `["after","missing"]`, false}, {"unknown child", `["after","future","child"]`, false}, {"sensitive child", `["after","secret","token"]`, false}, {"sensitive unknown child", `["after","both","child"]`, false},
		{"null child", `["after","nested","items",0,"child"]`, false}, {"scalar child", `["after","nested","items",1,"name","child"]`, false},
		{"list out of bounds", `["after","nested","items",2]`, false}, {"negative index", `["after","nested","items",-1]`, false}, {"fractional index", `["after","nested","items",0.5]`, false}, {"boolean index", `["after","nested","items",true]`, false}, {"string index", `["after","nested","items","0"]`, false}, {"object index", `["after","nested",0]`, false},
		{"outside values", `["resource","type"]`, false}, {"empty path", `[]`, false}, {"nonarray path", `"after"`, false}, {"nonstring root", `[0]`, false}, {"huge index", `["after","nested","items",1e9999]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := evaluationSnapshot(t, fmt.Sprintf(`classification := {"level":"high","reasons":[{"code":"observed-change","message":"Observed mutation.","paths":[%s]}]}`, tc.path))
			assessment := Compile(context.Background(), snapshot).Assess(context.Background(), *summary.PolicyEvidence)
			row := assessment.Resources[0]
			if !tc.valid {
				if row.Status != "unavailable" || row.ErrorCode != "result_invalid" || row.Level != nil {
					t.Fatalf("invalid path returned classification: %+v", row)
				}
				return
			}
			if assessment.Status != "complete" || row.Status != "classified" || row.Level == nil || *row.Level != sdk.RiskHigh {
				t.Fatalf("valid resource path rejected: %+v", assessment)
			}
			got, err := json.Marshal(row.Reasons[0].Paths)
			if err != nil {
				t.Fatal(err)
			}
			var actual, expected any
			_ = json.Unmarshal(got, &actual)
			_ = json.Unmarshal([]byte("["+tc.path+"]"), &expected)
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("reason paths = %s, want [%s]", got, tc.path)
			}
		})
	}
}

func TestAssess_GivenAbsentSide_ShouldAllowItsRootButRejectChildren(t *testing.T) {
	for _, tc := range []struct{ name, change, root string }{
		{"create", `{"actions":["create"],"before":null,"after":{}}`, "before"},
		{"delete", `{"actions":["delete"],"before":{},"after":null}`, "after"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evidence := acquiredEvidence(t, tc.change).PolicyEvidence
			for _, child := range []bool{false, true} {
				path := fmt.Sprintf(`[%q]`, tc.root)
				if child {
					path = fmt.Sprintf(`[%q,"missing"]`, tc.root)
				}
				snapshot := evaluationSnapshot(t, fmt.Sprintf(`classification := {"level":"low","reasons":[{"code":"action","message":"Action evidence.","paths":[%s]}]}`, path))
				result := Compile(context.Background(), snapshot).Assess(context.Background(), *evidence)
				if !child && result.Status != "complete" {
					t.Fatalf("absent side root is a node: %+v", result)
				}
				if child && result.Resources[0].ErrorCode != "result_invalid" {
					t.Fatalf("absent node child accepted: %+v", result)
				}
			}
		})
	}
}

func TestAssess_GivenPreciseRedactedPhantomEvidence_ShouldEvaluateOriginalTaggedInput(t *testing.T) {
	summary := acquiredEvidence(t, `{"actions":["update"],"before":{"big":9007199254740993,"ordered":[null,2,1],"nullable":null,"secret":"private"},"after":{"big":9007199254740993,"ordered":[2,null,1],"secret":"private"},"before_sensitive":{"secret":true},"after_unknown":{"future":true,"both":true},"after_sensitive":{"secret":true,"both":true}}`)
	// Separate fixture establishes the phantom annotation without uncertainty changing its display diff.
	phantom := acquiredEvidence(t, `{"actions":["update"],"before":{"ordered":[1,2]},"after":{"ordered":[2,1]}}`)
	terraform.DetectPhantomChanges(phantom.Changes)
	if len(phantom.Changes) != 1 || !phantom.Changes[0].IsPhantom {
		t.Fatal("fixture must be an eligible phantom mutation")
	}
	snapshot := evaluationSnapshot(t, `classification := {"level":"high","reasons":[{"code":"precise-evidence","message":"Original tagged evidence retained.","paths":[]}]} if {
 input.change.before.entries.big.value == "9007199254740993"
 to_number(input.change.before.entries.big.value) > 9007199254740992
 input.change.before.entries.ordered.items[0].kind == "null"
 input.change.after.entries.ordered.items[0].value == "2"
 input.change.before.entries.nullable.kind == "null"
 not input.change.after.entries.nullable
 input.change.before.entries.secret == {"kind":"sensitive"}
 input.change.after.entries.future == {"kind":"unknown"}
 input.change.after.entries.both == {"kind":"sensitive_unknown"}
 count(object.keys(input)) == 3
 }`)
	result := Compile(context.Background(), snapshot).Assess(context.Background(), *summary.PolicyEvidence)
	if result.Status != "complete" || result.Resources[0].Level == nil || *result.Resources[0].Level != sdk.RiskHigh {
		t.Fatalf("precise input did not reach selected policy: %+v", result)
	}
	phantomPolicy := evaluationSnapshot(t, `classification := {"level":"high","reasons":[{"code":"ordered-change","message":"List order changed.","paths":[["before","ordered",0],["after","ordered",0]]}]} if {input.change.before.entries.ordered.items[0].value == "1"; input.change.after.entries.ordered.items[0].value == "2"}`)
	result = Compile(context.Background(), phantomPolicy).Assess(context.Background(), *phantom.PolicyEvidence)
	if result.Status != "complete" || result.Resources[0].Level == nil || *result.Resources[0].Level != sdk.RiskHigh {
		t.Fatalf("eligible phantom was exempted or normalized: %+v", result)
	}
}

func TestAssess_GivenCustomLambdaOnlyPolicy_ShouldLeaveDatabaseUndefined(t *testing.T) {
	snapshot := evaluationSnapshot(t, `classification := {"level":"low","reasons":[{"code":"lambda-only","message":"Selected Lambda rule.","paths":[]}]} if {input.resource.type == "aws_lambda_function"}`)
	result := Compile(context.Background(), snapshot).Assess(context.Background(), sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function"), evaluationResource("aws_db_instance.worker", "aws_db_instance")}})
	if result.Status != "partial" || result.OverallLevel != nil || result.Resources[1].ErrorCode != "result_undefined" || result.Resources[1].Level != nil {
		t.Fatalf("custom policy gained hidden database fallback: %+v", result)
	}
}
