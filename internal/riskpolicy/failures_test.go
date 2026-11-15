package riskpolicy

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestAssess_GivenResourceFailuresAmongExplicitNoneResults_ShouldPreserveFailuresWithoutAggregate(t *testing.T) {
	const explicitNone = `{"level":"none","reasons":[{"code":"project-default","message":"Explicit project classification.","paths":[]}]}`
	for _, tc := range []struct{ name, rules, code string }{
		{"undefined", "", "result_undefined"},
		{"null", `classification := null if { input.resource.name == "worker" }`, "result_invalid"},
		{"invalid reason", `classification := {"level":"none","reasons":[{"code":"invalid","message":"private-source-marker","paths":[["after","missing"]]}]} if { input.resource.name == "worker" }`, "result_invalid"},
		{"conflicting values", `classification := {"level":"low","reasons":[]} if { input.resource.name == "worker" }
classification := {"level":"high","reasons":[]} if { input.resource.name == "worker" }`, "evaluation_conflict"},
		{"executed builtin failure", `classification := {"level":"high","reasons":[]} if { input.resource.name == "worker"; to_number("private-source-marker") == 7 }`, "evaluation_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := evaluationSnapshot(t, `classification := `+explicitNone+` if { input.resource.name != "worker" }
`+tc.rules)
			failed := evaluationResource("aws_lambda_function.worker", "aws_lambda_function")
			before := evaluationResource("aws_lambda_function.before", "aws_lambda_function")
			before.Input.Resource.Name = "before"
			after := evaluationResource("aws_lambda_function.after", "aws_lambda_function")
			after.Input.Resource.Name = "after"
			excluded := sdk.PolicyResourceEvidence{Address: "aws_db_instance.existing", Actions: []string{"no-op"}, Exclusion: "no_op"}
			for _, scenario := range []struct {
				name      string
				resources []sdk.PolicyResourceEvidence
				status    string
			}{
				{"independent neighbors", []sdk.PolicyResourceEvidence{before, failed, after, excluded}, "partial"},
				{"all eligible fail", []sdk.PolicyResourceEvidence{failed, excluded}, "unavailable"},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{PlanID: "sha256:reviewed-plan", Resources: scenario.resources})
					if assessment.Status != scenario.status || assessment.OverallLevel != nil || len(assessment.Resources) != len(scenario.resources) {
						t.Fatalf("resource failure became successful, lost review rows, or produced aggregate: %+v", assessment)
					}
					if assessment.PlanID != "sha256:reviewed-plan" || assessment.Policy.ID != snapshot.ID() || assessment.Policy.Digest != snapshot.Digest() || assessment.Policy.Kind != "custom" || assessment.Policy.EvaluatorVersion == "" || assessment.Policy.CapabilityDigest == "" {
						t.Fatalf("failed assessment lost evaluated plan/source identity: %+v", assessment)
					}
					for i, row := range assessment.Resources {
						resource := scenario.resources[i]
						if row.Address != resource.Address || !reflect.DeepEqual(row.Actions, resource.Actions) {
							t.Fatalf("review row lost mutation identity: %+v", row)
						}
						switch row.Address {
						case failed.Address:
							if row.Status != "unavailable" || row.ErrorCode != tc.code || row.Level != nil || len(row.Reasons) != 0 {
								t.Errorf("failed resource returned a classification instead of %s: %+v", tc.code, row)
							}
						case excluded.Address:
							if row.Status != "not_assessed" || row.Exclusion != "no_op" || row.Level != nil || len(row.Reasons) != 0 || row.ErrorCode != "" {
								t.Errorf("excluded resource became a classification or failure: %+v", row)
							}
						default:
							if row.Status != "classified" || row.Level == nil || *row.Level != sdk.RiskNone || row.ErrorCode != "" || len(row.Reasons) != 1 || row.Reasons[0].Code != "project-default" || row.Reasons[0].Message != "Explicit project classification." {
								t.Errorf("failure erased independent explicit-none classification: %+v", row)
							}
						}
					}
					encoded, err := json.Marshal(assessment)
					if err != nil {
						t.Fatal(err)
					}
					if strings.Contains(string(encoded), "private-source-marker") {
						t.Fatalf("failed assessment disclosed authored value or raw evaluator diagnostic: %s", encoded)
					}
				})
			}
		})
	}
}

func TestAssess_GivenCompilationFailureRegardlessOfEligibility_ShouldRemainUnavailableWithoutClassifications(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"parse failure", `classification := "private-source-marker" if {`},
		{"unsafe variable", `classification := private_source_marker`},
		{"builtin type error", `classification := startswith(42, "private-source-marker")`},
		{"conflicting definitions", `classification := "private-source-marker"
classification(value) := value`},
		{"denied capability", `classification := http.send({"method":"GET","url":"https://private-source-marker.invalid"})`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := evaluationSnapshot(t, tc.body)
			excluded := sdk.PolicyResourceEvidence{Address: "aws_db_instance.existing", Actions: []string{"no-op"}, Exclusion: "no_op"}
			for _, scenario := range []struct {
				name      string
				resources []sdk.PolicyResourceEvidence
			}{
				{"eligible mutation", []sdk.PolicyResourceEvidence{evaluationResource("aws_lambda_function.worker", "aws_lambda_function")}},
				{"excluded only", []sdk.PolicyResourceEvidence{excluded}},
				{"no changes", nil},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					assessment := Assess(context.Background(), snapshot, sdk.PolicyEvidence{PlanID: "sha256:reviewed-plan", Resources: scenario.resources})
					if assessment.Status != "unavailable" || assessment.ErrorCode != "compilation_failed" || assessment.OverallLevel != nil || len(assessment.Resources) != 0 {
						t.Fatalf("invalid source became neutral, empty success, or resource classifications: %+v", assessment)
					}
					if assessment.PlanID != "sha256:reviewed-plan" || assessment.Policy.ID != snapshot.ID() || assessment.Policy.Digest != snapshot.Digest() {
						t.Fatalf("compilation failure lost selected-source provenance: %+v", assessment)
					}
					encoded, err := json.Marshal(assessment)
					if err != nil {
						t.Fatal(err)
					}
					if strings.Contains(string(encoded), "private-source-marker") || strings.Contains(string(encoded), "private_source_marker") {
						t.Fatalf("compilation failure disclosed raw source diagnostics: %s", encoded)
					}
				})
			}
		})
	}
}
