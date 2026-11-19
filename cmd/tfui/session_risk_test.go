package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

const sessionRiskPolicy = `package tfui.risk.v1
import rego.v1
default classification := {"level":"low","reasons":[{"code":"selected-project-policy","message":"Reviewed with the selected project policy.","paths":[]}]}
`

func TestSession_GivenSeededRepositoryFixtures_ShouldShareExactSelectedPolicyClassificationsAndReasons(t *testing.T) {
	for _, fixture := range []string{"../../tests/fixtures/plan.json", "../../demo/fixtures/plan-large.json"} {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			data, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			session := newRiskReviewSession(t, data)
			var plan *tfuiplan.Plugin
			err = session.RunPlugin(context.Background(), "plan", func(p sdk.Plugin) tea.Cmd {
				plan = p.(*tfuiplan.Plugin)
				return plan.Activate(tfuiplan.Input{})
			})
			var exitErr *exitError
			if !errors.As(err, &exitErr) || exitErr.code != 2 {
				t.Fatalf("seeded Plan exit = %v, want changes exit 2", err)
			}
			planAssessment := plan.Assessment()
			var risk *tfuirisk.Plugin
			err = session.RunPlugin(context.Background(), "risk", func(p sdk.Plugin) tea.Cmd {
				risk = p.(*tfuirisk.Plugin)
				return risk.Activate(tfuirisk.Input{})
			})
			if err != nil {
				t.Fatalf("seeded Risk failed: %v; assessment %+v; stderr %q", err, risk.Assessment(), session.effects.Stderr)
			}
			assertSelectedSessionAssessment(t, planAssessment, risk.Assessment(), data)
			assertPlanSelectedReason(t, plan)
			if _, err := os.Stat(filepath.Join(session.cfg.Dir, "tfplan.out")); !os.IsNotExist(err) {
				t.Fatalf("seeded review created plan artifact: %v", err)
			}
		})
	}
}

func TestSession_GivenLivePlanThenRisk_ShouldReuseReviewedEvidenceWithoutAnotherTerraformPlan(t *testing.T) {
	session := newRiskReviewSession(t, []byte(sessionPlanJSON))
	session.sources = PreseedSources{}
	session.cfg.PreloadedData = false
	binary := filepath.Join(session.cfg.Dir, "terraform")
	script := `#!/bin/sh
printf '%s\n' "$1" >> calls.log
case "$1" in
version) printf '%s\n' '{"terraform_version":"1.14.0","platform":"linux_amd64","provider_selections":{}}' ;;
workspace) printf '%s\n' 'default' ;;
plan) exit 2 ;;
show) cat plan.json ;;
*) exit 1 ;;
esac
`
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	session.cfg.Terraform.Bin = binary
	svc, _, err := session.buildService()
	if err != nil {
		t.Fatal(err)
	}
	registry := buildRegistry(svc, session.cfg, session.rootCfg)
	defer registry.Shutdown()
	planPlugin, _ := registry.ByID("plan")
	plan := planPlugin.(*tfuiplan.Plugin)
	app := session.buildApp(svc, registry)
	driver := macro.NewDriver(app, 120, 40)
	driver.Init()
	driver.SendMsg(sdk.NavigateMsg{PluginID: "plan"})
	if err := driver.WaitUntil(func(string) bool { return plan.Assessment() != nil }, 5*time.Second); err != nil {
		t.Fatalf("live Plan did not finish: %v; view %q", err, driver.View())
	}
	planAssessment := plan.Assessment()
	assertPlanSelectedReason(t, plan)
	driver.SendMsg(sdk.NavigateMsg{PluginID: "risk"})
	p, _ := registry.ByID("risk")
	risk := p.(*tfuirisk.Plugin)
	if err := driver.WaitUntil(func(string) bool { return risk.Ready() }, 5*time.Second); err != nil {
		t.Fatalf("live Risk did not finish: %v; view %q", err, driver.View())
	}
	assertSelectedSessionAssessment(t, planAssessment, risk.Assessment(), []byte(sessionPlanJSON))
	initialDigest := risk.Assessment().Policy.Digest
	edited := strings.ReplaceAll(sessionRiskPolicy, `"low"`, `"high"`)
	edited = strings.ReplaceAll(edited, "Reviewed with the selected project policy.", "Reviewed with the edited project policy.")
	if err := os.WriteFile(filepath.Join(session.cfg.Dir, "policy.rego"), []byte(edited), 0600); err != nil {
		t.Fatal(err)
	}
	driver.SendMsg(sdk.NavigateMsg{PluginID: "plan"})
	if view := driver.View(); !strings.Contains(strings.ToLower(view), "overall risk: high") {
		t.Fatalf("Plan retained old classification after source edit: %q", view)
	}
	driver.SendMsg(sdk.NavigateMsg{PluginID: "risk"})
	current := risk.Assessment()
	if current == nil || current.Status != "complete" || current.Policy.Digest == initialDigest || current.PlanID != planAssessment.PlanID || current.OverallLevel == nil || *current.OverallLevel != sdk.RiskHigh || len(current.Resources) != 1 || len(current.Resources[0].Reasons) != 1 || current.Resources[0].Reasons[0].Message != "Reviewed with the edited project policy." {
		t.Fatalf("Risk did not reload edited selected policy against retained evidence: %+v", current)
	}
	if err := os.Remove(filepath.Join(session.cfg.Dir, "policy.rego")); err != nil {
		t.Fatal(err)
	}
	driver.SendMsg(sdk.NavigateMsg{PluginID: "plan"})
	if view := driver.View(); plan.Status() != sdk.StatusDone || !strings.Contains(view, "aws_lambda_function.worker") || !strings.Contains(view, "risk unavailable: source_unreadable") {
		t.Fatalf("Plan review was lost or retained old classification after source deletion: %q", view)
	}
	driver.SendMsg(sdk.NavigateMsg{PluginID: "risk"})
	if risk.Status() != sdk.StatusError || len(risk.Stderr()) == 0 || !strings.Contains(string(risk.Stderr()), "source_unreadable") {
		t.Fatalf("Risk source deletion status=%v stderr=%q, want terminal source error", risk.Status(), risk.Stderr())
	}
	calls, err := os.ReadFile(filepath.Join(session.cfg.Dir, "calls.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(calls), "plan\n") != 1 || strings.Count(string(calls), "show\n") != 1 {
		t.Fatalf("Plan → Risk Terraform commands = %q, want one plan and one show", calls)
	}
	member := filepath.Join(session.cfg.Dir, "member")
	if err := os.Mkdir(member, 0700); err != nil {
		t.Fatal(err)
	}
	workspacePolicy := strings.ReplaceAll(edited, "Reviewed with the edited project policy.", "Reviewed with the selected workspace policy.")
	for name, content := range map[string]string{
		"main.tf": "# workspace review\n", "plan.json": sessionPlanJSON,
		"tfui.hcl":       "risk_policy = \"missing.rego\"\nworkspace \"dev\" { risk_policy = \"workspace.rego\" }\n",
		"workspace.rego": workspacePolicy,
	} {
		if err := os.WriteFile(filepath.Join(member, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	driver.SendMsg(sdk.ContextSwitchRequestMsg{Chdir: "member", Workspace: "dev"})
	if risk.Assessment() != nil || risk.GetCtx().Chdir != "member" || risk.GetCtx().Workspace != "dev" || risk.GetCtx().WorkingDir != member {
		t.Fatalf("Context switch retained assessment or old Context: assessment=%+v Context=%+v", risk.Assessment(), risk.GetCtx())
	}
	driver.SendMsg(sdk.NavigateMsg{PluginID: "plan"})
	planAssessment = plan.Assessment()
	driver.SendMsg(sdk.NavigateMsg{PluginID: "risk"})
	current = risk.Assessment()
	if current == nil || !reflect.DeepEqual(planAssessment, current) || current.Status != "complete" || current.Policy.Digest == initialDigest || current.OverallLevel == nil || *current.OverallLevel != sdk.RiskHigh || len(current.Resources) != 1 || len(current.Resources[0].Reasons) != 1 || current.Resources[0].Reasons[0].Message != "Reviewed with the selected workspace policy." {
		t.Fatalf("new Context did not select workspace HCL policy for both views: Plan=%+v Risk=%+v", planAssessment, current)
	}
	memberCalls, err := os.ReadFile(filepath.Join(member, "calls.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(memberCalls), "plan\n") != 1 || strings.Count(string(memberCalls), "show\n") != 1 {
		t.Fatalf("new Context Plan → Risk commands = %q, want one plan and one show in member", memberCalls)
	}
}

func newRiskReviewSession(t *testing.T, plan []byte) *Session {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string][]byte{
		"main.tf": []byte("# selected policy review\n"), "plan.json": plan,
		"tfui.hcl":    []byte(`defaults { risk_policy = "policy.rego" }`),
		"policy.rego": []byte(sessionRiskPolicy),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := config.LoadRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &Session{
		cfg:     config.Config{Dir: dir, PreloadedData: true, Terraform: config.TerraformConfig{Bin: filepath.Join(dir, "nonexistent-terraform")}},
		rootCfg: root, sources: PreseedSources{Plan: filepath.Join(dir, "plan.json")}, silentStderr: true,
		effects: Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}},
	}
}

func assertSelectedSessionAssessment(t *testing.T, plan, risk *sdk.PolicyAssessment, data []byte) {
	t.Helper()
	if plan == nil || risk == nil || !reflect.DeepEqual(plan, risk) {
		t.Fatalf("Plan and Risk selected-policy assessments differ: Plan=%+v Risk=%+v", plan, risk)
	}
	if plan.Status != "complete" || plan.Policy.Kind != "custom" || plan.PlanID != fmt.Sprintf("sha256:%x", sha256.Sum256(data)) || plan.OverallLevel == nil || *plan.OverallLevel != sdk.RiskLow {
		t.Fatalf("selected assessment = %+v, want complete custom-policy LOW for exact reviewed bytes", plan)
	}
	var fixture struct {
		Resources []struct {
			Address string
			Change  struct{ Actions []string }
		} `json:"resource_changes"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(plan.Resources) != len(fixture.Resources) || len(plan.Resources) == 0 {
		t.Fatalf("assessed rows = %d, want all %d reviewed resource rows", len(plan.Resources), len(fixture.Resources))
	}
	for _, resource := range fixture.Resources {
		found := false
		for _, row := range plan.Resources {
			if row.Address == resource.Address && reflect.DeepEqual(row.Actions, resource.Change.Actions) {
				found = true
				if reflect.DeepEqual(resource.Change.Actions, []string{"no-op"}) && (row.Status != "not_assessed" || row.Exclusion != "no_op" || row.Level != nil || len(row.Reasons) != 0) {
					t.Fatalf("reviewed no-op %s must remain excluded without classification: %+v", resource.Address, row)
				}
				if !reflect.DeepEqual(resource.Change.Actions, []string{"no-op"}) && row.Exclusion != "" {
					t.Fatalf("reviewed mutation %s was excluded: %+v", resource.Address, row)
				}
			}
		}
		if !found {
			t.Fatalf("assessment omitted reviewed resource %s with actions %v", resource.Address, resource.Change.Actions)
		}
	}
	for _, row := range plan.Resources {
		if row.Exclusion != "" {
			if row.Exclusion != "no_op" || row.Status != "not_assessed" || row.Level != nil || len(row.Reasons) != 0 {
				t.Fatalf("excluded resource acquired a classification: %+v", row)
			}
			continue
		}
		wantReasons := []sdk.PolicyReason{{Code: "selected-project-policy", Message: "Reviewed with the selected project policy.", Paths: [][]any{}}}
		if row.Status != "classified" || row.Level == nil || *row.Level != sdk.RiskLow || !reflect.DeepEqual(row.Reasons, wantReasons) {
			t.Fatalf("resource %s = %+v, want exact selected LOW classification and reasons %+v", row.Address, row, wantReasons)
		}
	}
}

func assertPlanSelectedReason(t *testing.T, plan *tfuiplan.Plugin) {
	t.Helper()
	plan.View(120, 40)
	cmd := plan.Stack().Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		plan.Update(cmd())
	}
	if view := plan.View(120, 40); !strings.Contains(view, "Reviewed with the selected project policy.") || !strings.Contains(strings.ToLower(view), "low") {
		t.Fatalf("Plan inspection = %q, want selected LOW classification and author reason", view)
	}
	plan.Stack().Update(tea.KeyMsg{Type: tea.KeyEsc})
}
