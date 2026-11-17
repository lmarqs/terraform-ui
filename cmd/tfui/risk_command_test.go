package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
)

func TestRunPlugin_GivenSeededRiskAssessment_ShouldWriteReportWithoutTerraform(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{"main.tf": "# seeded risk report\n", "plan.json": sessionPlanJSON} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	session := &Session{
		cfg:     config.Config{Dir: dir, PreloadedData: true, Terraform: config.TerraformConfig{Bin: filepath.Join(dir, "nonexistent-terraform")}},
		sources: PreseedSources{Plan: filepath.Join(dir, "plan.json")}, silentStderr: true,
		effects: Effects{Stdout: &stdout, Stderr: &stderr},
	}
	if err := session.RunPlugin(context.Background(), "risk", func(p sdk.Plugin) tea.Cmd { return p.(*tfuirisk.Plugin).Activate() }); err != nil {
		t.Fatalf("seeded Risk failed: %v; stderr %q", err, stderr.String())
	}
	if got := stdout.String(); !strings.HasPrefix(got, "Risk: ") || !strings.Contains(got, "update aws_lambda_function.worker:") || !strings.Contains(got, "Threshold: off (disabled)\n") {
		t.Fatalf("seeded Risk stdout = %q, want completed text report", got)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("successful disabled comparison stderr = %q", got)
	}
}

func TestRunPlugin_GivenLegacyRiskSetting_ShouldPreservePlanReviewWithUnavailableRisk(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"main.tf": "# retained plan review\n", "plan.json": sessionPlanJSON,
		"tfui.hcl": "defaults {\n plugin \"risk\" { level = \"high\" }\n}",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := config.LoadRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	resolved := config.Resolve(root, nil, "")
	var stdout, stderr bytes.Buffer
	session := &Session{
		cfg:     config.Config{Dir: dir, PreloadedData: true, Plugins: resolved.PluginConfigs(), Terraform: config.TerraformConfig{Bin: filepath.Join(dir, "nonexistent-terraform")}},
		rootCfg: root, sources: PreseedSources{Plan: filepath.Join(dir, "plan.json")}, silentStderr: true,
		effects: Effects{Stdout: &stdout, Stderr: &stderr},
	}
	var plan *tfuiplan.Plugin
	err = session.RunPlugin(context.Background(), "plan", func(p sdk.Plugin) tea.Cmd { plan = p.(*tfuiplan.Plugin); return plan.Activate(tfuiplan.Input{}) })
	if code, message := exitStatus(err); code != 2 || message != "" {
		t.Fatalf("legacy Risk setting changed Plan exit: code %d message %q stderr %q", code, message, stderr.String())
	}
	if !strings.Contains(stdout.String(), "aws_lambda_function.worker") || strings.Contains(stdout.String(), "Risk:") {
		t.Fatalf("Plan review lost or retained successful risk: %q", stdout.String())
	}
	want := "risk unavailable: configuration_invalid\n"
	if stderr.String() != want {
		t.Fatalf("legacy Risk setting Plan stderr = %q, want %q", stderr.String(), want)
	}
	if view := plan.View(100, 30); !strings.Contains(view, "risk unavailable: configuration_invalid") {
		t.Fatalf("Plan view retained available risk after invalid configuration: %q", view)
	}
}

func TestRoot_GivenThresholdFlag_ShouldExposeItForEveryCommand(t *testing.T) {
	root, _, _ := buildRoot()
	flag := root.PersistentFlags().Lookup("threshold")
	if flag == nil {
		t.Fatal("--threshold must be persistent so every tfui command accepts it")
	}
}

func TestRiskCommand_GivenSeededPlan_ShouldEmitRequestedFormatAndThresholdExit(t *testing.T) {
	for _, tc := range []struct {
		name, threshold string
		json            bool
		exit            int
	}{
		{"text disabled", "off", false, 0},
		{"text met", "none", false, 3},
		{"json disabled", "off", true, 0},
		{"json met", "none", true, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, body := range map[string]string{"main.tf": "# seeded command report\n", "plan.json": sessionPlanJSON} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			root, session, _ := buildRoot()
			root.AddCommand(buildRiskCommand(session))
			var stdout, stderr bytes.Buffer
			session.effects = Effects{Stdout: &stdout, Stderr: &stderr}
			args := []string{"--project", dir, "--ci", "--plan", filepath.Join(dir, "plan.json"), "--terraform-bin", filepath.Join(dir, "nonexistent-terraform"), "--threshold", tc.threshold, "risk"}
			if tc.json {
				args = append(args, "--json")
			}
			root.SetArgs(args)
			err := root.Execute()
			if code, message := exitStatus(err); code != tc.exit || message != "" {
				t.Fatalf("risk exit = %d, message %q, error %v; stderr %q", code, message, err, stderr.String())
			}
			if tc.json {
				var report struct {
					Schema  string `json:"schema_version"`
					PlanID  string `json:"plan_id"`
					Context struct {
						Project          *string `json:"project"`
						Chdir, Workspace string
					}
					Threshold struct {
						Level  *string
						Result string
					}
				}
				if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
					t.Fatalf("risk JSON stdout %q: %v", stdout.String(), err)
				}
				if report.Schema != "tfui.risk.report/v1" || report.PlanID != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(sessionPlanJSON))) {
					t.Fatalf("report lost seeded evidence identity: %+v", report)
				}
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatal(err)
				}
				chdir, err := filepath.Rel(cwd, dir)
				if err != nil {
					t.Fatal(err)
				}
				if report.Context.Project != nil || report.Context.Chdir != chdir || report.Context.Workspace != "default" {
					t.Fatalf("zero-config acquisition context = %+v", report.Context)
				}
				if tc.threshold == "off" && (report.Threshold.Level != nil || report.Threshold.Result != "disabled") {
					t.Fatalf("disabled threshold = %+v", report.Threshold)
				}
				if tc.threshold == "none" && (report.Threshold.Level == nil || *report.Threshold.Level != "none" || report.Threshold.Result != "met") {
					t.Fatalf("inclusive threshold = %+v", report.Threshold)
				}
			} else if got := stdout.String(); !strings.HasPrefix(got, "Risk: ") || !strings.Contains(got, "aws_lambda_function.worker:") {
				t.Fatalf("text report = %q", got)
			}
			if tc.exit == 3 && !strings.HasPrefix(stderr.String(), "risk threshold met: ") {
				t.Fatalf("met comparison diagnostic = %q", stderr.String())
			}
			if tc.exit == 0 && stderr.Len() != 0 {
				t.Fatalf("disabled threshold emitted stderr %q", stderr.String())
			}
		})
	}
}

func TestRiskCommand_GivenAssessmentFailure_ShouldLeaveStdoutEmpty(t *testing.T) {
	for _, tc := range []struct{ name, config, threshold, diagnostic string }{
		{"invalid CLI threshold", "", "urgent", "risk unavailable: configuration_invalid\n"},
		{"empty CLI threshold", "", "", "risk unavailable: configuration_invalid\n"},
		{"source failure", `defaults { risk_policy = "missing.rego" }`, "off", "risk unavailable: source_unreadable\n"},
		{"legacy level", "defaults {\n plugin \"risk\" { level = \"high\" }\n}", "off", "risk unavailable: configuration_invalid\n\nhint: risk setting level is unsupported; use threshold\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, body := range map[string]string{"main.tf": "# seeded failure\n", "plan.json": sessionPlanJSON, "tfui.hcl": tc.config} {
				if name == "tfui.hcl" && body == "" {
					continue
				}
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			root, session, _ := buildRoot()
			root.AddCommand(buildRiskCommand(session))
			var stdout, stderr bytes.Buffer
			session.effects = Effects{Stdout: &stdout, Stderr: &stderr}
			root.SetArgs([]string{"risk", "--project", dir, "--ci", "--json", "--plan", filepath.Join(dir, "plan.json"), "--threshold=" + tc.threshold, "--terraform-bin", filepath.Join(dir, "nonexistent-terraform")})
			code, message := exitStatus(root.Execute())
			if code != 1 || message != "" || stdout.Len() != 0 || stderr.String() != tc.diagnostic {
				t.Fatalf("failure exit %d message %q stdout %q stderr %q, want fixed diagnostic %q", code, message, stdout.String(), stderr.String(), tc.diagnostic)
			}
		})
	}
}

func TestRiskCommand_GivenConfiguredThreshold_ShouldUseSettingUnlessCLIOverrides(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(fmt.Sprint(override), func(t *testing.T) {
			dir := t.TempDir()
			for name, body := range map[string]string{
				"main.tf": "# seeded threshold\n", "plan.json": sessionPlanJSON,
				"tfui.hcl": "defaults {\n risk_policy = \"policy.rego\"\n plugin \"risk\" { threshold = \"low\" }\n}",
				"policy.rego": `package tfui.risk.v1
import rego.v1
default classification := {"level":"low","reasons":[{"code":"default","message":"Project default.","paths":[]}]}
`,
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			root, session, _ := buildRoot()
			root.AddCommand(buildRiskCommand(session))
			var stdout, stderr bytes.Buffer
			session.effects = Effects{Stdout: &stdout, Stderr: &stderr}
			args := []string{"risk", "--project", dir, "--ci", "--json", "--plan", filepath.Join(dir, "plan.json"), "--terraform-bin", filepath.Join(dir, "nonexistent-terraform")}
			wantExit, wantLevel, wantResult := 3, "low", "met"
			if override {
				args = append(args, "--threshold", "high")
				wantExit, wantLevel, wantResult = 0, "high", "below"
			}
			root.SetArgs(args)
			if code, message := exitStatus(root.Execute()); code != wantExit || message != "" {
				t.Fatalf("exit %d message %q stderr %q", code, message, stderr.String())
			}
			var report struct {
				Overall   string `json:"overall_level"`
				Threshold struct{ Level, Result string }
				Context   struct {
					Project          *string
					Chdir, Workspace string
				}
			}
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.Overall != "low" || report.Threshold.Level != wantLevel || report.Threshold.Result != wantResult {
				t.Fatalf("setting/override result = %+v", report)
			}
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			project, err := filepath.Rel(cwd, dir)
			if err != nil {
				t.Fatal(err)
			}
			if report.Context.Project == nil || *report.Context.Project != project || report.Context.Chdir != "." || report.Context.Workspace != "default" {
				t.Fatalf("project report context = %+v", report.Context)
			}
		})
	}
}
