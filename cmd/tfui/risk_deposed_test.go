package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readDeposedFixture(t *testing.T) (plan, configuration, policy string) {
	t.Helper()
	read := func(name string) string {
		body, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "risk-deposed", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	return read("plan.json"), read("tfui.hcl"), read("policy.rego")
}

func TestRiskCommand_GivenUpdateAndDeposedDeleteAtSameAddress_ShouldEmitTwoRowsDistinguishedByDeposedKey(t *testing.T) {
	plan, configuration, policy := readDeposedFixture(t)

	code, stdout, stderr := runSeededRiskCommand(t, plan, configuration, policy, "--json")

	var report struct {
		Status    string
		Overall   string `json:"overall_level"`
		Resources []map[string]any
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("report is not JSON: %v\n%s", err, stdout)
	}
	if code != 0 || stderr != "" || report.Status != "complete" || report.Overall != "high" {
		t.Fatalf("exit %d stderr %q report %s, want exit 0, complete, overall high", code, stderr, stdout)
	}
	if len(report.Resources) != 2 {
		t.Fatalf("resources = %d, want 2 (one per object at the address): %s", len(report.Resources), stdout)
	}
	current, leftover := report.Resources[0], report.Resources[1]
	for i, row := range report.Resources {
		if _, present := row["deposed"]; !present {
			t.Fatalf("row %d has no deposed field: %v", i, row)
		}
		if row["address"] != "aws_instance.web" {
			t.Fatalf("row %d address = %v, want aws_instance.web", i, row["address"])
		}
	}
	if current["deposed"] != nil || current["level"] != "low" || current["actions"].([]any)[0] != "update" {
		t.Errorf("first row = %v, want the current object (deposed null) classified low", current)
	}
	if leftover["deposed"] != "00000001" || leftover["level"] != "high" || leftover["actions"].([]any)[0] != "delete" {
		t.Errorf("second row = %v, want the deposed object 00000001 classified high", leftover)
	}
	if reasons, _ := leftover["reasons"].([]any); len(reasons) != 1 || reasons[0].(map[string]any)["code"] != "deposed-object" {
		t.Errorf("deposed row reasons = %v, want its own deposed-object reason", leftover["reasons"])
	}
	if reasons, _ := current["reasons"].([]any); len(reasons) != 1 || reasons[0].(map[string]any)["code"] != "current-object" {
		t.Errorf("current row reasons = %v, want its own current-object reason", current["reasons"])
	}
}

func TestRiskCommand_GivenUpdateAndDeposedDeleteAtSameAddress_ShouldShowDeposedKeyNextToAddressInText(t *testing.T) {
	plan, configuration, policy := readDeposedFixture(t)

	code, stdout, stderr := runSeededRiskCommand(t, plan, configuration, policy)

	want := "Risk: high\n" +
		"Policy: " + policyID(t, stdout) + "\n" +
		"Mutations: 2 eligible, 2 classified; 0 changes not assessed\n" +
		"Threshold: off (disabled)\n" +
		"update aws_instance.web: low\n" +
		"  current-object: Current object update.\n" +
		"delete aws_instance.web (deposed 00000001): high\n" +
		"  deposed-object: Deposed object cleanup.\n"
	if code != 0 || stderr != "" || stdout != want {
		t.Fatalf("exit %d stderr %q\ngot:\n%s\nwant:\n%s", code, stderr, stdout, want)
	}
}

func policyID(t *testing.T, text string) string {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		if id, ok := strings.CutPrefix(line, "Policy: "); ok {
			return id
		}
	}
	t.Fatalf("no policy line in %q", text)
	return ""
}
