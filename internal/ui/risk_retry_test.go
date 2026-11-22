package ui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiblastradius "github.com/lmarqs/terraform-ui/plugins/blastradius"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
)

// newFailedAssessmentApp opens pluginID on a reviewed plan whose policy file is
// missing, so the assessment fails and the plugin shows its error state.
func newFailedAssessmentApp(t *testing.T, pluginID string) (App, string) {
	t.Helper()
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	rootCfg, err := config.LoadRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	svc := newMockService("default", nil)
	registry := plugin.NewRegistry()
	registry.RegisterFactory("risk", tfuirisk.New, plugin.PluginMeta{})
	registry.RegisterFactory("blastradius", tfuiblastradius.New, plugin.PluginMeta{})
	registry.Build(svc, nil)
	app := NewApp(config.Config{Dir: root}, svc, registry, rootCfg)
	app.Init()
	model, cmd := app.Update(sdk.PlanCompletedEvent{Context: app.holder.current, Summary: reviewedRiskSummary()})
	app = model.(App)
	drainPolicyAppCommand(&app, cmd)
	model, cmd = app.Update(sdk.NavigateMsg{PluginID: pluginID})
	app = model.(App)
	drainPolicyAppCommand(&app, cmd)
	return app, root
}

func statusOf(t *testing.T, app App) sdk.Status {
	t.Helper()
	statusable, ok := app.ActivePlugin().(sdk.Statusable)
	if !ok {
		t.Fatalf("%s has no status", app.ActivePlugin().ID())
	}
	return statusable.Status()
}

// ctrl+r in the error state asks the host again. The policy is read afresh at
// each assessment, so a fixed policy file turns the retry into a result.
func TestCtrlR_GivenFailedAssessment_ShouldLoadAgainAndShowTheFixedResult(t *testing.T) {
	for _, pluginID := range []string{"risk", "blastradius"} {
		t.Run(pluginID, func(t *testing.T) {
			app, root := newFailedAssessmentApp(t, pluginID)
			want := (sdk.HintSetRetry | sdk.HintSetBack).Hints()
			if got := app.ActivePlugin().(sdk.Hintable).Hints(); !slices.Equal(got, want) {
				t.Fatalf("error-state hints = %v, want %v", got, want)
			}
			writePolicyFile(t, root, "policy.rego", "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"low\",\"reasons\":[{\"code\":\"fixed\",\"message\":\"Fixed policy.\",\"paths\":[]}]}\n")

			model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
			app = model.(App)

			if statusOf(t, app) != sdk.StatusLoading {
				t.Fatalf("status after ctrl+r = %v, want loading", statusOf(t, app))
			}
			drainPolicyAppCommand(&app, cmd)
			if statusOf(t, app) != sdk.StatusDone || strings.Contains(app.ActivePlugin().View(100, 30), "unavailable") {
				t.Fatalf("after the retry: status %v, view %q; want the fixed result", statusOf(t, app), app.ActivePlugin().View(100, 30))
			}
		})
	}
}

func TestEsc_GivenFailedAssessment_ShouldGoHome(t *testing.T) {
	for _, pluginID := range []string{"risk", "blastradius"} {
		t.Run(pluginID, func(t *testing.T) {
			app, _ := newFailedAssessmentApp(t, pluginID)

			model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
			app = model.(App)
			drainPolicyAppCommand(&app, cmd)

			if app.ActivePlugin() != nil {
				t.Fatalf("Esc in the error state left %s active, want home", app.ActivePlugin().ID())
			}
		})
	}
}
