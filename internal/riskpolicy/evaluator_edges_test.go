package riskpolicy

import (
	"context"
	"testing"
	"time"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/open-policy-agent/opa/v1/rego"
)

func TestAssess_GivenMissingSnapshot_ShouldNotReportEmptySuccess(t *testing.T) {
	result := Compile(context.Background(), Snapshot{}).Assess(context.Background(), sdk.PolicyEvidence{})
	if result.Status != "unavailable" || result.ErrorCode != "compilation_failed" || result.OverallLevel != nil {
		t.Fatalf("missing snapshot returned success: %+v", result)
	}
}

func TestClassifyResults_GivenAmbiguousRuntimeOutput_ShouldRejectCardinality(t *testing.T) {
	// Exercise the runtime-result boundary with real multi-result/multi-expression
	// queries; the production complete-document query must remain a singleton.
	for _, query := range []string{`[1, 2][_]`, `true; true`} {
		t.Run(query, func(t *testing.T) {
			results, err := rego.New(rego.Query(query)).Eval(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			_, reasons, code := classifyResults(results, nil)
			if code != "result_cardinality" || reasons != nil {
				t.Fatalf("ambiguous runtime output returned %q, %+v", code, reasons)
			}
		})
	}
}

func TestAssess_WhenDeadlineExpires_ShouldReturnNoPublishableRows(t *testing.T) {
	snapshot := evaluationSnapshot(t, projectDefault+`
classification := {"level":"high","reasons":[{"code":"large-input","message":"Large input.","paths":[]}]} if {
 input.resource.type == "aws_lambda_function"
 every x in input.change.before.items {
  every y in input.change.before.items { x == y }
 }
}`)
	slow := evaluationResource("aws_lambda_function.worker", "aws_lambda_function")
	items := make([]sdk.PolicyValue, 10000)
	for i := range items {
		items[i] = sdk.PolicyValue{Kind: "null"}
	}
	slow.Input.Change.Before = sdk.PolicyValue{Kind: "array", Items: &items}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	result := Compile(ctx, snapshot).Assess(ctx, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{
		evaluationResource("aws_db_instance.worker", "aws_db_instance"), slow,
	}})
	if ctx.Err() == nil || result.Status != "cancelled" || result.ErrorCode != "cancelled" || len(result.Resources) != 0 || result.OverallLevel != nil {
		t.Fatalf("cancelled assessment published rows: %+v; context=%v", result, ctx.Err())
	}
}
