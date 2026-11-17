package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/macro"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
)

const sessionPlanJSON = `{"format_version":"1.2","resource_changes":[{"address":"aws_lambda_function.worker","mode":"managed","type":"aws_lambda_function","name":"worker","provider_name":"registry.terraform.io/hashicorp/aws","change":{"actions":["update"],"before":{"memory_size":128},"after":{"memory_size":256}}}]}` + "\n"

func TestRunPlugin_GivenSeededPlan_ShouldAnalyzeWithoutTerraformInInitialContext(t *testing.T) {
	for _, chdir := range []string{"", "member"} {
		for _, id := range []string{"plan", "risk"} {
			t.Run(id+"/"+chdir, func(t *testing.T) {
				dir := t.TempDir()
				workDir := filepath.Join(dir, chdir)
				if err := os.MkdirAll(workDir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(workDir, "main.tf"), []byte("# seeded analysis\n"), 0600); err != nil {
					t.Fatal(err)
				}
				source := filepath.Join(dir, "plan.json")
				if err := os.WriteFile(source, []byte(sessionPlanJSON), 0600); err != nil {
					t.Fatal(err)
				}
				var stdout, stderr bytes.Buffer
				session := &Session{
					cfg:     config.Config{Dir: dir, Chdir: chdir, PreloadedData: true, Terraform: config.TerraformConfig{Bin: filepath.Join(dir, "nonexistent-terraform")}},
					sources: PreseedSources{Plan: source}, silentStderr: true,
					effects: Effects{Stdout: &stdout, Stderr: &stderr},
				}
				var plan *tfuiplan.Plugin
				var risk *tfuirisk.Plugin
				var evidence *sdk.PolicyEvidence
				err := session.RunPlugin(context.Background(), id, func(p sdk.Plugin) tea.Cmd {
					if id == "plan" {
						plan = p.(*tfuiplan.Plugin)
						return plan.Activate(tfuiplan.Input{})
					}
					risk = p.(*tfuirisk.Plugin)
					if got := risk.GetCtx().WorkingDir; got != workDir {
						t.Errorf("initial working directory = %q, want %q", got, workDir)
					}
					return risk.Activate()
				})
				var exitErr *exitError
				if err != nil && (id != "plan" || !errors.As(err, &exitErr) || exitErr.code != 2) {
					t.Fatalf("seeded %s RunPlugin error = %v; stderr = %s", id, err, stderr.String())
				}

				if id == "plan" {
					if !strings.Contains(stdout.String(), "aws_lambda_function.worker") {
						t.Fatalf("seeded Plan output = %q, want reviewed resource", stdout.String())
					}
					summary, err := plan.GetCtx().Service.Plan(context.Background(), sdk.PlanOptions{})
					if err != nil {
						t.Fatal(err)
					}
					evidence = summary.PolicyEvidence
				} else {
					if !strings.Contains(risk.View(100, 30), "aws_lambda_function.worker") {
						t.Fatalf("seeded Risk view = %q, want reviewed resource", risk.View(100, 30))
					}
					assessment := risk.Assessment()
					if assessment == nil || assessment.Status != "complete" || assessment.PlanID != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(sessionPlanJSON))) {
						t.Fatalf("seeded Risk assessment = %+v, want reviewed JSON identity", assessment)
					}
					summary, err := risk.GetCtx().Service.Plan(context.Background(), sdk.PlanOptions{})
					if err != nil {
						t.Fatal(err)
					}
					evidence = summary.PolicyEvidence
				}
				if evidence == nil || evidence.PlanID != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(sessionPlanJSON))) {
					t.Fatalf("seeded analysis evidence = %#v, want original reviewed JSON identity", evidence)
				}
				if len(evidence.Resources) != 1 || evidence.Resources[0].Input == nil || evidence.Resources[0].Input.Resource.Address != "aws_lambda_function.worker" {
					t.Fatalf("seeded policy resource evidence = %#v, want reviewed Lambda input", evidence.Resources)
				}
				if _, err := os.Stat(filepath.Join(workDir, "tfplan.out")); !os.IsNotExist(err) {
					t.Fatalf("seeded analysis created default plan artifact: %v", err)
				}
			})
		}
	}
}

func TestRunPlugin_GivenPlanJSON_ShouldPassThroughExactTerraformShowBytes(t *testing.T) {
	dir := t.TempDir()
	source := " \n" + sessionPlanJSON + "\t\n"
	if err := os.WriteFile(filepath.Join(dir, "show.json"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "terraform")
	script := `#!/bin/sh
case "$1" in
version) printf '%s\n' '{"terraform_version":"1.14.0","platform":"linux_amd64","provider_selections":{}}' ;;
plan) exit 2 ;;
show) cat show.json ;;
*) exit 1 ;;
esac
`
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	session := &Session{cfg: config.Config{Dir: dir, Terraform: config.TerraformConfig{Bin: binary}}, silentStderr: true, effects: Effects{Stdout: &stdout, Stderr: &stderr}}
	err := session.RunPlugin(context.Background(), "plan", func(p sdk.Plugin) tea.Cmd { return p.(*tfuiplan.Plugin).Activate(tfuiplan.Input{JSON: true}) })
	var exitErr *exitError
	if err != nil && (!errors.As(err, &exitErr) || exitErr.code != 2) {
		t.Fatalf("RunPlugin error = %v; stderr = %s", err, stderr.String())
	}
	if got := stdout.String(); got != source {
		t.Fatalf("Plan JSON output = %q, want exact Terraform bytes %q", got, source)
	}
}

func TestRunPlugin_GivenPolicySourceFailure_ShouldPreserveSeededPlanReview(t *testing.T) {
	for _, tc := range []struct {
		name, root, member, source, code string
	}{
		{"missing source", `defaults { risk_policy = "missing.rego" }`, "", "", "source_unreadable"},
		{"empty source", `defaults { risk_policy = "policy.rego" }`, "", "", "source_invalid"},
		{"malformed member selector", `defaults { risk_policy = "policy.rego" }`, `risk_policy = null`, "package example", "configuration_invalid"},
		{"malformed root selector", `defaults { risk_policy = null }`, "", "package example", "configuration_invalid"},
		{"malformed member HCL", `defaults { risk_policy = "policy.rego" }`, `risk_policy = "unterminated`, "package example", "configuration_invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			member := filepath.Join(dir, "member")
			if err := os.Mkdir(member, 0700); err != nil {
				t.Fatal(err)
			}
			for path, content := range map[string]string{
				filepath.Join(dir, "tfui.hcl"):    tc.root,
				filepath.Join(member, "tfui.hcl"): tc.member,
				filepath.Join(member, "main.tf"):  "# seeded review",
				filepath.Join(dir, "policy.rego"): tc.source,
				filepath.Join(dir, "plan.json"):   sessionPlanJSON,
			} {
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			root, err := config.LoadRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			session := &Session{
				cfg:     config.Config{Dir: dir, Chdir: "member", PreloadedData: true, Terraform: config.TerraformConfig{Bin: filepath.Join(dir, "nonexistent-terraform")}},
				rootCfg: root, sources: PreseedSources{Plan: filepath.Join(dir, "plan.json")}, silentStderr: true,
				effects: Effects{Stdout: &stdout, Stderr: &stderr},
			}
			var plan *tfuiplan.Plugin
			err = session.RunPlugin(context.Background(), "plan", func(p sdk.Plugin) tea.Cmd {
				plan = p.(*tfuiplan.Plugin)
				return plan.Activate(tfuiplan.Input{})
			})
			var exitErr *exitError
			if !errors.As(err, &exitErr) || exitErr.code != 2 {
				t.Fatalf("Plan exit = %v, want unchanged plan-with-changes exit 2; stderr = %q", err, stderr.String())
			}
			if plan.Status() != sdk.StatusDone || !strings.Contains(stdout.String(), "aws_lambda_function.worker") {
				t.Fatalf("Plan review lost: status = %v, stdout = %q", plan.Status(), stdout.String())
			}
			want := "risk unavailable: " + tc.code
			if !strings.Contains(stderr.String(), want) {
				t.Fatalf("stderr = %q, want explicit policy failure %q", stderr.String(), want)
			}
			if !strings.Contains(plan.View(100, 30), want) {
				t.Errorf("Plan view missing %q", want)
			}
			if strings.Contains(stdout.String(), "Risk:") {
				t.Errorf("Plan output retained successful risk badge: %q", stdout.String())
			}
		})
	}
}

func TestRunPlugin_GivenRiskSourceFailure_ShouldTerminateWithFixedDiagnosticWithoutTerraform(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tfui.hcl"), []byte(`defaults { risk_policy = "missing.rego" }`), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := config.LoadRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	session := &Session{
		cfg:     config.Config{Dir: dir, PreloadedData: true, Terraform: config.TerraformConfig{Bin: filepath.Join(dir, "nonexistent-terraform")}},
		rootCfg: root, silentStderr: true,
		effects: Effects{Stdout: &stdout, Stderr: &stderr},
	}
	done := make(chan error, 1)
	go func() {
		done <- session.RunPlugin(context.Background(), "risk", func(p sdk.Plugin) tea.Cmd { return p.(*tfuirisk.Plugin).Activate() })
	}()
	select {
	case err = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Risk source failure hung instead of reaching terminal error")
	}
	var exitErr *exitError
	if !errors.As(err, &exitErr) || exitErr.code != 1 {
		t.Fatalf("Risk exit = %v, want operational failure 1", err)
	}
	if got := stderr.String(); got != "risk unavailable: source_unreadable\n" {
		t.Fatalf("Risk stderr = %q", got)
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("Risk failure stdout = %q, want no successful report", got)
	}
}

// TestPlan_GivenSelectedPolicy_ShouldStayBusyUntilItsAssessmentArrives drives the
// App through the macro driver that runs headless `tfui plan`, which prints the
// output port at the first step where the plugin is Ready.
func TestPlan_GivenSelectedPolicy_ShouldStayBusyUntilItsAssessmentArrives(t *testing.T) {
	session := newRiskReviewSession(t, []byte(sessionPlanJSON))
	policy := strings.ReplaceAll(sessionRiskPolicy, `"low"`, `"critical"`)
	if err := os.WriteFile(filepath.Join(session.cfg.Dir, "policy.rego"), []byte(policy), 0600); err != nil {
		t.Fatal(err)
	}
	svc, _, err := session.buildService()
	if err != nil {
		t.Fatal(err)
	}
	registry := buildRegistry(svc, session.cfg, session.rootCfg)
	defer registry.Shutdown()
	p, _ := registry.ByID("plan")
	observer := &firstReadyOutput{Model: session.buildApp(svc, registry), plan: p.(*tfuiplan.Plugin)}
	driver := macro.NewDriver(observer, 120, 40)
	driver.Init()
	driver.SendMsg(sdk.NavigateMsg{PluginID: "plan"})
	if observer.stdout == nil {
		t.Fatalf("Plan never became Ready; view %q", driver.View())
	}
	if got := string(observer.stdout); !strings.Contains(got, "aws_lambda_function.worker") || !strings.Contains(got, "Risk: critical\n") {
		t.Fatalf("Plan stdout at first Ready = %q; want the selected policy's CRITICAL risk", got)
	}
}

// firstReadyOutput records the plan's Stdout at the first message after which
// the plan is Ready.
type firstReadyOutput struct {
	tea.Model
	plan   *tfuiplan.Plugin
	stdout []byte
}

func (m *firstReadyOutput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.Model.Update(msg)
	m.Model = model
	if m.stdout == nil && m.plan.Ready() {
		m.stdout, _ = m.plan.Stdout()
	}
	return m, cmd
}
