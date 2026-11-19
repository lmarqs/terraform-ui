package riskpolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/open-policy-agent/opa/v1/tester"
)

func TestLambdaTemplate_GivenIsolatedFixtures_ShouldExplainReviewedLevelsWithoutBuiltinFallback(t *testing.T) {
	dir := filepath.Join("..", "..", "examples", "risk-policy", "lambda")
	snapshot, err := LoadSource(config.RiskPolicySelection{Path: dir}, BuiltinModules())
	if err != nil {
		t.Fatalf("copyable complete Lambda policy is unavailable: %v", err)
	}
	if snapshot.Kind() != "custom" {
		t.Fatalf("copied policy ownership = %s, want custom", snapshot.Kind())
	}
	for _, tc := range []struct {
		name, code string
		level      sdk.RiskLevel
		actions    []string
		paths      [][]any
	}{
		{"filename-same-hash", "lambda-code-source-change", sdk.RiskHigh, []string{"update"}, [][]any{{"before", "filename"}, {"after", "filename"}}},
		{"filename-null-hash", "lambda-code-source-change", sdk.RiskHigh, []string{"update"}, [][]any{{"before", "filename"}, {"after", "filename"}}},
		{"filename-absent-hash", "lambda-code-source-change", sdk.RiskHigh, []string{"update"}, [][]any{{"before", "filename"}, {"after", "filename"}}},
		{"filename-unknown-hash", "lambda-code-source-change", sdk.RiskHigh, []string{"update"}, [][]any{{"before", "filename"}, {"after", "filename"}}},
		{"hash-unknown", "lambda-source-uncertain", sdk.RiskHigh, []string{"update"}, nil},
		{"memory-known-sources", "lambda-configuration-update", sdk.RiskHigh, []string{"update"}, [][]any{{"before", "memory_size"}, {"after", "memory_size"}}},
		{"memory-unknown-source", "lambda-source-uncertain", sdk.RiskHigh, []string{"update"}, nil},
		{"sensitive-source", "lambda-source-uncertain", sdk.RiskHigh, []string{"update"}, nil},
		{"function-create", "mutation-action", sdk.RiskMedium, []string{"create"}, [][]any{}},
		{"function-delete", "mutation-action", sdk.RiskCritical, []string{"delete"}, [][]any{}},
		{"function-delete-create", "mutation-action", sdk.RiskCritical, []string{"delete", "create"}, [][]any{}},
		{"function-create-delete", "mutation-action", sdk.RiskCritical, []string{"create", "delete"}, [][]any{}},
		{"layer-create", "mutation-action", sdk.RiskLow, []string{"create"}, [][]any{}},
		{"layer-update", "type-action-default", sdk.RiskLow, []string{"update"}, [][]any{}},
		{"layer-delete", "mutation-action", sdk.RiskMedium, []string{"delete"}, [][]any{}},
		{"layer-delete-create", "mutation-action", sdk.RiskMedium, []string{"delete", "create"}, [][]any{}},
		{"layer-create-delete", "mutation-action", sdk.RiskMedium, []string{"create", "delete"}, [][]any{}},
		{"database-delete", "project-default", sdk.RiskLow, []string{"delete"}, [][]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := templateInput(t, dir, tc.name)
			if !reflect.DeepEqual(input.Change.Actions, tc.actions) {
				t.Fatalf("fixture actions = %v, want %v", input.Change.Actions, tc.actions)
			}
			result := Compile(context.Background(), snapshot).Assess(context.Background(), sdk.PolicyEvidence{Resources: []sdk.PolicyResourceEvidence{{
				Address: input.Resource.Address, Actions: input.Change.Actions, Input: &input,
			}}})
			if result.Status != "complete" || len(result.Resources) != 1 {
				t.Fatalf("copied policy did not classify isolated fixture: %+v", result)
			}
			row := result.Resources[0]
			if row.Level == nil || *row.Level != tc.level || len(row.Reasons) != 1 {
				t.Fatalf("classification = %+v, want %s and one attributable reason", row, tc.level)
			}
			if !reflect.DeepEqual(row.Actions, tc.actions) {
				t.Errorf("reported action order = %v, want %v", row.Actions, tc.actions)
			}
			reason := row.Reasons[0]
			if reason.Code != tc.code {
				t.Errorf("reason code = %s, want %s", reason.Code, tc.code)
			}
			if tc.paths != nil && !reflect.DeepEqual(reason.Paths, tc.paths) {
				t.Errorf("evidence paths = %v, want %v", reason.Paths, tc.paths)
			}
			for _, claim := range []string{"placeholder", "externally managed", "memory-only", "look safe"} {
				if strings.Contains(strings.ToLower(reason.Message), claim) {
					t.Errorf("reason invents an unsupported claim %q: %s", claim, reason.Message)
				}
			}
		})
	}
}

func templateInput(t *testing.T, dir, name string) sdk.PolicyInput {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "fixtures.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures struct {
		Fixtures map[string]json.RawMessage `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	raw, ok := fixtures.Fixtures[name]
	if !ok {
		t.Fatalf("isolated fixture %q is missing", name)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var input sdk.PolicyInput
	if err := decoder.Decode(&input); err != nil {
		t.Fatalf("fixture is not a full v1 envelope: %v", err)
	}
	if input.SchemaVersion != "tfui.risk.input/v1" || input.Resource.Address == "" || input.Resource.Mode != "managed" || input.Resource.Type == "" || input.Resource.Name == "" || input.Resource.ProviderName == "" || input.Resource.Index.Kind != "absent" || input.Change.ReplacePaths == nil || input.Change.Before.Kind == "" || input.Change.After.Kind == "" {
		t.Fatalf("fixture lacks required independent input evidence: %+v", input)
	}
	return input
}

func TestLambdaTemplate_GivenOrdinaryRegoAuthorTests_ShouldPassWithIsolatedFixtures(t *testing.T) {
	dir := filepath.Join("..", "..", "examples", "risk-policy", "lambda")
	if _, err := os.Stat(filepath.Join(dir, "main_test.rego")); err != nil {
		t.Fatalf("copyable policy author tests are unavailable: %v", err)
	}
	results, err := tester.Run(context.Background(), dir)
	if err != nil {
		t.Fatalf("ordinary Rego author tests could not run: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("ordinary Rego author tests discovered no tests")
	}
	for _, result := range results {
		if !result.Pass() {
			t.Errorf("ordinary Rego author test failed: %s", result)
		}
	}
}
