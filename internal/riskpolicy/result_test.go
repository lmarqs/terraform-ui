package riskpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestAssess_GivenMalformedAuthoredResult_ShouldRejectInsteadOfDisplayingClassification(t *testing.T) {
	reason := func(code, message, paths any) map[string]any {
		return map[string]any{"code": code, "message": message, "paths": paths}
	}
	validReason := reason("project-default", "Project default.", []any{})
	result := func(reasons any) map[string]any {
		return map[string]any{"level": "low", "reasons": reasons}
	}
	tooManyReasons := make([]any, 17)
	for i := range tooManyReasons {
		tooManyReasons[i] = validReason
	}
	tooManyPaths := make([]any, 17)
	for i := range tooManyPaths {
		tooManyPaths[i] = []any{"after"}
	}
	for _, tc := range []struct {
		name   string
		output any
	}{
		{"nonstring level", map[string]any{"level": 3, "reasons": []any{validReason}}},
		{"extra result field", map[string]any{"level": "none", "reasons": []any{validReason}, "threshold": "off"}},
		{"nonarray reasons", result("project-default")},
		{"too many reasons", result(tooManyReasons)},
		{"nonobject reason", result([]any{"project-default"})},
		{"missing reason field", result([]any{map[string]any{"code": "project-default", "message": "Default."}})},
		{"extra reason field", result([]any{map[string]any{"code": "project-default", "message": "Default.", "paths": []any{}, "trusted": true}})},
		{"nonstring code", result([]any{reason(42, "Default.", []any{})})},
		{"uppercase code", result([]any{reason("Project-default", "Default.", []any{})})},
		{"long code", result([]any{reason(strings.Repeat("a", 65), "Default.", []any{})})},
		{"nonstring message", result([]any{reason("project-default", false, []any{})})},
		{"blank message", result([]any{reason("project-default", " \t ", []any{})})},
		{"long unicode message", result([]any{reason("project-default", strings.Repeat("界", 513), []any{})})},
		{"terminal control message", result([]any{reason("project-default", "Default.\x1b[31m", []any{})})},
		{"nonarray paths", result([]any{reason("project-default", "Default.", "after")})},
		{"too many paths", result([]any{reason("project-default", "Default.", tooManyPaths)})},
		{"unvalidated resource path", result([]any{reason("project-default", "Default.", []any{[]any{"after", "missing"}})})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(tc.output)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := evaluationSnapshot(t, "classification := "+string(encoded))
			assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function")}})
			if assessment.Status != "unavailable" || assessment.OverallLevel != nil || len(assessment.Resources) != 1 {
				t.Fatalf("malformed authored output should remain unavailable: %+v", assessment)
			}
			row := assessment.Resources[0]
			if row.Status != "unavailable" || row.ErrorCode != "result_invalid" || row.Level != nil || len(row.Reasons) != 0 {
				t.Fatalf("malformed authored output leaked a classification: %+v", row)
			}
		})
	}
}

func TestAssess_GivenCriticalClassificationWithMaximumUnicodeMessage_ShouldPreserveReason(t *testing.T) {
	message := strings.Repeat("界", 512)
	snapshot := evaluationSnapshot(t, fmt.Sprintf(`classification := {"level":"critical","reasons":[{"code":"critical-mutation","message":%q,"paths":[]}]}`, message))
	assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function")}})
	if assessment.Status != "complete" || assessment.OverallLevel == nil || *assessment.OverallLevel != sdk.RiskCritical || len(assessment.Resources) != 1 {
		t.Fatalf("valid critical output should classify: %+v", assessment)
	}
	row := assessment.Resources[0]
	if row.Level == nil || *row.Level != sdk.RiskCritical || len(row.Reasons) != 1 || row.Reasons[0].Message != message {
		t.Fatalf("critical classification lost accepted Unicode explanation: %+v", row)
	}
}

func TestAssess_GivenEligibleMutationWithoutEvidence_ShouldInvalidateWholePlan(t *testing.T) {
	snapshot := evaluationSnapshot(t, projectDefault)
	assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{
		evaluationResource("aws_lambda_function.worker", "aws_lambda_function"),
		{Address: "aws_db_instance.worker", Actions: []string{"update"}},
	}})
	if assessment.Status != "unavailable" || assessment.ErrorCode != "input_incomplete" || assessment.OverallLevel != nil || len(assessment.Resources) != 0 {
		t.Fatalf("missing eligible evidence should invalidate all classifications: %+v", assessment)
	}
}

func TestAssess_GivenNestedOutputCallInUnusedRule_ShouldRejectWholeSource(t *testing.T) {
	for _, call := range []string{`print("private constant")`, `trace("private constant")`} {
		t.Run(call, func(t *testing.T) {
			snapshot := evaluationSnapshot(t, projectDefault+"\nunused := {\"nested\": ["+call+"]}")
			assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function")}})
			if assessment.Status != "unavailable" || assessment.ErrorCode != "compilation_failed" || assessment.OverallLevel != nil {
				t.Fatalf("nested output builtin should invalidate unused code too: %+v", assessment)
			}
		})
	}
}
