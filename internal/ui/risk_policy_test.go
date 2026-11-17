package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestRiskPolicyDiagnostic_GivenChangedSourceAtSamePath_ShouldReloadWithoutFallback(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	writePolicyFile(t, root, "policy.rego", "package example\n")
	_, deps := newPolicyDiagnosticApp(t, root, "", true)
	for _, step := range []struct {
		name, source, want string
		deleted            bool
	}{
		{"initial source", "package example\n", "", false},
		{"same path emptied", "", "source_invalid", false},
		{"same path restored", "package changed\n", "", false},
		{"source deleted", "", "source_unreadable", true},
		{"source recreated", "package recreated\n", "", false},
	} {
		t.Run(step.name, func(t *testing.T) {
			if step.deleted {
				if err := os.Remove(filepath.Join(root, "policy.rego")); err != nil {
					t.Fatal(err)
				}
			} else {
				writePolicyFile(t, root, "policy.rego", step.source)
			}
			if got := deps.RiskPolicyDiagnostic(); got != step.want {
				t.Fatalf("policy diagnostic = %q, want %q", got, step.want)
			}
		})
	}
}

func TestRiskPolicyDiagnostic_GivenContextAndConfigurationChanges_ShouldReadCurrentSelection(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "root.rego" }`)
	writePolicyFile(t, root, "root.rego", "package root\n")
	writePolicyFile(t, root, "a/tfui.hcl", `risk_policy = "missing.rego"
workspace "dev" { risk_policy = "" }`)
	writePolicyFile(t, root, "b/tfui.hcl", `risk_policy = "b.rego"`)
	writePolicyFile(t, root, "b/b.rego", "package member\n")
	app, deps := newPolicyDiagnosticApp(t, root, "a", true)
	if got := deps.RiskPolicyDiagnostic(); got != "source_unreadable" {
		t.Fatalf("initial member diagnostic = %q", got)
	}
	for _, step := range []struct{ chdir, workspace, want string }{
		{"a", "dev", ""}, {"a", "prod", "source_unreadable"}, {"b", "prod", ""}, {"", "prod", ""},
	} {
		model, cmd := app.Update(sdk.ContextSwitchRequestMsg{Chdir: sdk.Chdir(step.chdir), Workspace: sdk.NewWorkspace(step.workspace)})
		app = model.(App)
		drainPolicyAppCommand(&app, cmd)
		if got := deps.RiskPolicyDiagnostic(); got != step.want {
			t.Fatalf("%s/%s diagnostic = %q, want %q", step.chdir, step.workspace, got, step.want)
		}
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = null }`)
	if got := deps.RiskPolicyDiagnostic(); got != "configuration_invalid" {
		t.Fatalf("edited root selector diagnostic = %q", got)
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "" }`)
	if got := deps.RiskPolicyDiagnostic(); got != "" {
		t.Fatalf("reset to builtin diagnostic = %q", got)
	}
}

func TestRiskPolicyDiagnostic_GivenNoSelectorOrExplicitReset_ShouldUseBuiltin(t *testing.T) {
	for _, tc := range []struct {
		name, root, member string
		loaded             bool
	}{
		{"zero config ignores member selector", "", `risk_policy = "missing.rego"`, false},
		{"absent selector", `member "member" {}`, "", true},
		{"explicit reset", `defaults { risk_policy = "missing.rego" }`, `risk_policy = ""`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.loaded {
				writePolicyFile(t, root, "tfui.hcl", tc.root)
			}
			writePolicyFile(t, root, "member/tfui.hcl", tc.member)
			_, deps := newPolicyDiagnosticApp(t, root, "member", tc.loaded)
			if got := deps.RiskPolicyDiagnostic(); got != "" {
				t.Fatalf("builtin selection diagnostic = %q", got)
			}
		})
	}
}

func newPolicyDiagnosticApp(t *testing.T, root, chdir string, loaded bool) (App, *sdk.PluginDeps) {
	t.Helper()
	var rootCfg *config.RootConfig
	if loaded {
		var err error
		rootCfg, err = config.LoadRoot(root)
		if err != nil {
			t.Fatal(err)
		}
	}
	var deps *sdk.PluginDeps
	registry := plugin.NewRegistry()
	registry.RegisterFactory("policyprobe", func(_ sdk.Service) sdk.Plugin {
		return &mockPluginWithInit{mockPlugin: mockPlugin{id: "policyprobe"}, initFn: func(d *sdk.PluginDeps) tea.Cmd { deps = d; return nil }}
	}, plugin.PluginMeta{})
	registry.Build(nil, nil)
	app := NewApp(config.Config{Dir: root, Chdir: chdir}, newMockService("default", nil), registry, rootCfg, &StandaloneConfig{PluginID: "policyprobe"})
	drainPolicyAppCommand(&app, app.Init())
	if deps == nil || deps.RiskPolicyDiagnostic == nil {
		t.Fatal("app did not provide policy diagnostic for current Context")
	}
	return app, deps
}

func drainPolicyAppCommand(app *App, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			drainPolicyAppCommand(app, sub)
		}
		return
	}
	model, next := app.Update(msg)
	*app = model.(App)
	drainPolicyAppCommand(app, next)
}

func writePolicyFile(t *testing.T, root, name, source string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRequestAssessment_GivenChangedSourceAtSamePath_ShouldReloadWithoutFallback(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	writePolicyFile(t, root, "policy.rego", "package example\n")
	app, deps, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
	for _, step := range []struct {
		name, source string
		want         sdk.ErrorCode
		deleted      bool
	}{
		{"initial source", "package example\n", "", false},
		{"same path emptied", "", sdk.ErrorSourceInvalid, false},
		{"same path restored", "package changed\n", "", false},
		{"source deleted", "", sdk.ErrorSourceUnreadable, true},
		{"source recreated", "package recreated\n", "", false},
	} {
		t.Run(step.name, func(t *testing.T) {
			if step.deleted {
				if err := os.Remove(filepath.Join(root, "policy.rego")); err != nil {
					t.Fatal(err)
				}
			} else {
				writePolicyFile(t, root, "policy.rego", step.source)
			}
			got := requestRisk(t, &app, recorder, deps.RequestAssessment(reviewedRiskSummary())).Assessment
			if got.ErrorCode != step.want || (step.want != "" && got.Status != sdk.AssessmentUnavailable) {
				t.Fatalf("assessment = %+v, want error code %q", got, step.want)
			}
		})
	}
}

func TestRequestAssessment_GivenContextAndConfigurationChanges_ShouldReadCurrentSelection(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "root.rego" }`)
	writePolicyFile(t, root, "root.rego", "package root\n")
	writePolicyFile(t, root, "a/tfui.hcl", `risk_policy = "missing.rego"
workspace "dev" { risk_policy = "" }`)
	writePolicyFile(t, root, "b/tfui.hcl", `risk_policy = "b.rego"`)
	writePolicyFile(t, root, "b/b.rego", "package member\n")
	app, deps, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
	request := func() sdk.ErrorCode {
		return requestRisk(t, &app, recorder, deps.RequestAssessment(reviewedRiskSummary())).Assessment.ErrorCode
	}
	for _, step := range []struct {
		chdir, workspace string
		want             sdk.ErrorCode
	}{
		{"a", "default", sdk.ErrorSourceUnreadable}, {"a", "dev", ""}, {"a", "prod", sdk.ErrorSourceUnreadable}, {"b", "prod", ""}, {"", "prod", ""},
	} {
		switchContext(&app, step.chdir, step.workspace)
		if got := request(); got != step.want {
			t.Fatalf("%s/%s error code = %q, want %q", step.chdir, step.workspace, got, step.want)
		}
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = null }`)
	if got := request(); got != sdk.ErrorConfigurationInvalid {
		t.Fatalf("edited root selector error code = %q", got)
	}
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "" }`)
	if got := request(); got != "" {
		t.Fatalf("reset to builtin error code = %q", got)
	}
}

func TestRequestAssessment_GivenNoSelectorOrExplicitReset_ShouldUseBuiltin(t *testing.T) {
	for _, tc := range []struct {
		name, root, member string
		loaded             bool
	}{
		{"zero config ignores member selector", "", `risk_policy = "missing.rego"`, false},
		{"absent selector", `member "member" {}`, "", true},
		{"explicit reset", `defaults { risk_policy = "missing.rego" }`, `risk_policy = ""`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.loaded {
				writePolicyFile(t, root, "tfui.hcl", tc.root)
			}
			writePolicyFile(t, root, "member/tfui.hcl", tc.member)
			app, deps, recorder := newAssessmentRequestApp(t, root, tc.loaded, newMockService("default", nil))
			switchContext(&app, "member", "default")
			got := requestRisk(t, &app, recorder, deps.RequestAssessment(reviewedRiskSummary())).Assessment
			if !got.Usable() || got.Policy.Kind != "builtin" {
				t.Fatalf("builtin selection assessment = %+v", got)
			}
		})
	}
}

func TestErrorCode_GivenErrorWithoutCode_ShouldUseFallback(t *testing.T) {
	if got := errorCode(errors.New("uncoded"), sdk.ErrorSourceUnreadable); got != sdk.ErrorSourceUnreadable {
		t.Fatalf("uncoded error = %q, want fallback", got)
	}
	if got := errorCode(fmt.Errorf("wrapped: %w", sdk.ErrorSourceInvalid), sdk.ErrorSourceUnreadable); got != sdk.ErrorSourceInvalid {
		t.Fatalf("coded error = %q, want its own code", got)
	}
}
