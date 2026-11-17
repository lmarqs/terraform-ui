package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runSeededRiskCommand(t *testing.T, plan, configuration, policy string, args ...string) (int, string, string) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{
		"main.tf": "# synthetic risk command fixture\n", "plan.json": plan,
		"tfui.hcl": configuration, "policy.rego": policy,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, session, _ := buildRoot()
	root.AddCommand(buildRiskCommand(session))
	var stdout, stderr bytes.Buffer
	session.effects = Effects{Stdout: &stdout, Stderr: &stderr}
	root.SetArgs(append([]string{"risk", "--project", dir, "--ci", "--plan", filepath.Join(dir, "plan.json"), "--terraform-bin", filepath.Join(dir, "nonexistent-terraform")}, args...))
	code, message := exitStatus(root.Execute())
	if message != "" {
		t.Fatalf("unexpected command error outside output ports: %q", message)
	}
	return code, stdout.String(), stderr.String()
}

func TestRiskCommand_GivenEveryClassificationAndThreshold_ShouldCompareInclusively(t *testing.T) {
	levels := []string{"none", "low", "medium", "high", "critical"}
	for rank, level := range levels {
		for thresholdRank, threshold := range levels {
			t.Run(level+"/"+threshold, func(t *testing.T) {
				policy := fmt.Sprintf("package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":%q,\"reasons\":[{\"code\":\"project-default\",\"message\":\"Explicit project classification.\",\"paths\":[]}]}\n", level)
				code, stdout, stderr := runSeededRiskCommand(t, sessionPlanJSON, `defaults { risk_policy = "policy.rego" }`, policy, "--json", "--risk-threshold", threshold)
				wantCode, wantResult, wantStderr := 0, "below", ""
				if rank >= thresholdRank {
					wantCode, wantResult = 3, "met"
					wantStderr = fmt.Sprintf("risk threshold met: %s >= %s\n", level, threshold)
				}
				var report struct {
					Status    string
					Overall   string `json:"overall_level"`
					Threshold struct{ Level, Result string }
					Resources []struct {
						Level   string
						Reasons []struct{ Code, Message string }
					}
				}
				if err := json.Unmarshal([]byte(stdout), &report); err != nil {
					t.Fatal(err)
				}
				if code != wantCode || stderr != wantStderr || report.Status != "complete" || report.Overall != level || report.Threshold.Level != threshold || report.Threshold.Result != wantResult {
					t.Fatalf("exit %d stderr %q report %+v, want exit %d, %s and stderr %q", code, stderr, report, wantCode, wantResult, wantStderr)
				}
				if len(report.Resources) != 1 || report.Resources[0].Level != level || len(report.Resources[0].Reasons) != 1 || report.Resources[0].Reasons[0].Code != "project-default" || report.Resources[0].Reasons[0].Message != "Explicit project classification." {
					t.Fatalf("explicit custom default or its basis was replaced: %+v", report.Resources)
				}
			})
		}
	}
}

func TestRiskCommand_GivenExplicitOffOverride_ShouldDisableConfiguredComparison(t *testing.T) {
	configuration := "defaults {\n risk_policy = \"policy.rego\"\n plugin \"risk\" { threshold = \"none\" }\n}"
	policy := "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"critical\",\"reasons\":[{\"code\":\"project-default\",\"message\":\"Explicit project classification.\",\"paths\":[]}]}\n"
	code, stdout, stderr := runSeededRiskCommand(t, sessionPlanJSON, configuration, policy, "--risk-threshold", "off", "--json")
	var report struct {
		Overall   string `json:"overall_level"`
		Threshold struct {
			Level  *string
			Result string
		}
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if code != 0 || stderr != "" || report.Overall != "critical" || report.Threshold.Level != nil || report.Threshold.Result != "disabled" {
		t.Fatalf("off override suppressed classification or retained comparison: exit %d stderr %q report %+v", code, stderr, report)
	}
}

func TestRiskCommand_GivenNoEligibleMutations_ShouldRetainExclusionsWithoutThresholdMatch(t *testing.T) {
	for _, tc := range []struct {
		name, plan string
		excluded   int
	}{
		{"no changes", `{"format_version":"1.2","resource_changes":[]}`, 0},
		{"excluded only", `{"format_version":"1.2","resource_changes":[{"address":"aws_s3_bucket.logs","mode":"managed","type":"aws_s3_bucket","name":"logs","change":{"actions":["no-op"],"before":{},"after":{}}},{"address":"data.aws_region.current","mode":"data","type":"aws_region","name":"current","change":{"actions":["read"],"before":null,"after":{}}}]}`, 2},
	} {
		for _, threshold := range []string{"off", "none", "critical"} {
			for _, format := range []string{"text", "json"} {
				t.Run(tc.name+"/"+threshold+"/"+format, func(t *testing.T) {
					args := []string{"--risk-threshold", threshold}
					if format == "json" {
						args = append(args, "--json")
					}
					code, stdout, stderr := runSeededRiskCommand(t, tc.plan, "", "", args...)
					if code != 0 || stderr != "" {
						t.Fatalf("empty assessment matched threshold or failed: exit %d stderr %q", code, stderr)
					}
					if format == "text" {
						if !strings.HasPrefix(stdout, fmt.Sprintf("No eligible mutations (%d changes not assessed).\n", tc.excluded)) || !strings.Contains(stdout, "Threshold: "+threshold+" (not_applicable)\n") {
							t.Fatalf("empty text = %q", stdout)
						}
						if tc.excluded > 0 && (!strings.Contains(stdout, "no-op aws_s3_bucket.logs: not assessed (no_op)\n") || !strings.Contains(stdout, "read data.aws_region.current: not assessed (read)\n")) {
							t.Fatalf("missing excluded text rows: %q", stdout)
						}
						return
					}
					var report struct {
						Status  string
						Overall *string `json:"overall_level"`
						Counts  struct {
							Eligible, Classified int
							Excluded             int `json:"not_assessed"`
						}
						Threshold struct {
							Level  *string
							Result string
						}
						Resources []map[string]any
					}
					if err := json.Unmarshal([]byte(stdout), &report); err != nil {
						t.Fatal(err)
					}
					if report.Status != "empty" || report.Overall != nil || report.Counts.Eligible != 0 || report.Counts.Classified != 0 || report.Counts.Excluded != tc.excluded || len(report.Resources) != tc.excluded || report.Threshold.Result != "not_applicable" {
						t.Fatalf("empty JSON = %s", stdout)
					}
					if threshold == "off" {
						if report.Threshold.Level != nil {
							t.Fatal("off threshold should be null")
						}
					} else if report.Threshold.Level == nil || *report.Threshold.Level != threshold {
						t.Fatalf("empty report lost configured threshold: %s", stdout)
					}
					for _, row := range report.Resources {
						if row["status"] != "not_assessed" || row["level"] != nil || row["reasons"] != nil || (row["exclusion"] != "no_op" && row["exclusion"] != "read") {
							t.Fatalf("excluded row gained classification: %+v", row)
						}
					}
				})
			}
		}
	}
}

func TestRiskCommand_GivenInvalidPolicy_ShouldEmitFixedDiagnosticWithoutSensitiveDetails(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"parse error", `classification := "synthetic-private-marker" if {`, "compilation_failed"},
		{"denied builtin", `classification := http.send({"method":"GET","url":"https://synthetic-private-marker.invalid"})`, "compilation_failed"},
		{"undefined", `other := "synthetic-private-marker"`, "result_undefined"},
		{"null", `classification := null`, "result_invalid"},
		{"empty object", `classification := {}`, "result_invalid"},
		{"empty array", `classification := []`, "result_invalid"},
		{"unknown level", `classification := {"level":"synthetic-private-marker","reasons":[]}`, "result_invalid"},
		{"empty reasons", `classification := {"level":"none","reasons":[]}`, "result_invalid"},
		{"invalid path", `classification := {"level":"none","reasons":[{"code":"test","message":"synthetic-private-marker","paths":[["after","missing"]]}]}`, "result_invalid"},
		{"conflicting values", "classification := {\"level\":\"low\",\"reasons\":[]} if {input.resource.mode == \"managed\"}\nclassification := {\"level\":\"high\",\"reasons\":[]} if {input.resource.name == \"worker\"}", "evaluation_conflict"},
		{"executed builtin error", `classification := {"level":"none","reasons":[]} if { to_number("synthetic-private-marker") == 7 }`, "evaluation_failed"},
	} {
		for _, format := range []string{"text", "json"} {
			t.Run(tc.name+"/"+format, func(t *testing.T) {
				args := []string{"--risk-threshold", "off"}
				if format == "json" {
					args = append(args, "--json")
				}
				code, stdout, stderr := runSeededRiskCommand(t, sessionPlanJSON, `defaults { risk_policy = "policy.rego" }`, "package tfui.risk.v1\nimport rego.v1\n"+tc.body+"\n", args...)
				want := "risk unavailable: " + tc.code
				if tc.code != "compilation_failed" {
					want += " for aws_lambda_function.worker"
				}
				want += "\n"
				if code != 1 || stdout != "" || stderr != want {
					t.Fatalf("failure exit %d stdout %q stderr %q, want exit 1, empty stdout, %q", code, stdout, stderr, want)
				}
			})
		}
	}
}

func TestRiskCommand_GivenSensitiveEvidence_ShouldClassifyRedactedNodeWithoutDisclosingValue(t *testing.T) {
	plan := strings.Replace(sessionPlanJSON, `"after":{"memory_size":256}`, `"after":{"memory_size":256,"token":"synthetic-sensitive-value"},"after_sensitive":{"token":true}`, 1)
	policy := `package tfui.risk.v1
import rego.v1
classification := {"level":"none","reasons":[{"code":"redacted","message":"Sensitive evidence is redacted.","paths":[]}]} if {
 input.change.after.entries.token.kind == "sensitive"
 not input.change.after.entries.token.value
}
`
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			args := []string{"--risk-threshold", "none"}
			if format == "json" {
				args = append(args, "--json")
			}
			code, stdout, stderr := runSeededRiskCommand(t, plan, `defaults { risk_policy = "policy.rego" }`, policy, args...)
			if code != 3 || !strings.Contains(stdout, "Sensitive evidence is redacted.") || stderr != "risk threshold met: none >= none\n" || strings.Contains(stdout+stderr, "synthetic-sensitive-value") {
				t.Fatalf("sensitive evidence was disclosed or prevented valid classification: exit %d stdout %q stderr %q", code, stdout, stderr)
			}
		})
	}
}

func TestRiskCommand_GivenPartialAssessment_ShouldSuppressReportAndThresholdComparison(t *testing.T) {
	worker := strings.TrimSuffix(strings.TrimPrefix(sessionPlanJSON, `{"format_version":"1.2","resource_changes":[`), "]}\n")
	neighbor := strings.ReplaceAll(worker, "worker", "neighbor")
	plan := `{"format_version":"1.2","resource_changes":[` + neighbor + "," + worker + "]}"
	policy := `package tfui.risk.v1
import rego.v1
classification := {"level":"critical","reasons":[{"code":"project-default","message":"Explicit project classification.","paths":[]}]} if { input.resource.name != "worker" }
`
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			args := []string{"--risk-threshold", "none"}
			if format == "json" {
				args = append(args, "--json")
			}
			code, stdout, stderr := runSeededRiskCommand(t, plan, `defaults { risk_policy = "policy.rego" }`, policy, args...)
			if code != 1 || stdout != "" || stderr != "risk unavailable: result_undefined for aws_lambda_function.worker\n" {
				t.Fatalf("partial assessment became a report or threshold success: exit %d stdout %q stderr %q", code, stdout, stderr)
			}
		})
	}
}

func TestRiskCommand_GivenUnavailableInputOrSource_ShouldEmitFixedGlobalDiagnostic(t *testing.T) {
	for _, tc := range []struct{ name, plan, configuration, policy, code string }{
		{"missing source", sessionPlanJSON, `defaults { risk_policy = "missing-private-marker.rego" }`, "", "source_unreadable"},
		{"empty source", sessionPlanJSON, `defaults { risk_policy = "policy.rego" }`, "", "source_invalid"},
		{"incomplete input", strings.Replace(sessionPlanJSON, `"before":{"memory_size":128},`, "", 1), "", "", "input_incomplete"},
		{"invalid input", strings.Replace(sessionPlanJSON, `"mode":"managed"`, `"mode":"synthetic-private-marker"`, 1), "", "", "input_invalid"},
	} {
		for _, format := range []string{"text", "json"} {
			t.Run(tc.name+"/"+format, func(t *testing.T) {
				args := []string{"--risk-threshold", "off"}
				if format == "json" {
					args = append(args, "--json")
				}
				code, stdout, stderr := runSeededRiskCommand(t, tc.plan, tc.configuration, tc.policy, args...)
				if code != 1 || stdout != "" || stderr != "risk unavailable: "+tc.code+"\n" {
					t.Fatalf("unavailable assessment output: exit %d stdout %q stderr %q", code, stdout, stderr)
				}
			})
		}
	}
}

func TestRiskCommand_GivenForgetChange_ShouldPrintNotAssessedForget(t *testing.T) {
	plan := `{"format_version":"1.2","resource_changes":[{"address":"aws_s3_bucket.forgotten","mode":"managed","type":"aws_s3_bucket","name":"forgotten","provider_name":"registry.terraform.io/hashicorp/aws","change":{"actions":["forget"],"before":{"bucket":"old"},"after":null,"after_unknown":{}}}]}`
	_, stdout, _ := runSeededRiskCommand(t, plan, "", "", "--risk-threshold", "off")
	if !strings.Contains(stdout, "forget aws_s3_bucket.forgotten: not assessed (forget)\n") {
		t.Fatalf("missing forget row in text output: %q", stdout)
	}
}
