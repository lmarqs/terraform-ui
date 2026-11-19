package ui

import (
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiblastradius "github.com/lmarqs/terraform-ui/plugins/blastradius"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
	tfuistate "github.com/lmarqs/terraform-ui/plugins/state"
)

// newPendingAssessmentApp opens pluginID with its assessment request issued but
// never answered. retained gives the plugin a reviewed plan first, so the host
// only evaluates policy; without it Risk's request makes the host run terraform plan.
func newPendingAssessmentApp(t *testing.T, pluginID string, retained bool) App {
	t.Helper()
	svc := newMockService("default", nil)
	registry := plugin.NewRegistry()
	registry.RegisterFactory("risk", tfuirisk.New, plugin.PluginMeta{})
	registry.RegisterFactory("blastradius", tfuiblastradius.New, plugin.PluginMeta{})
	registry.RegisterFactory("state", tfuistate.New, plugin.PluginMeta{})
	registry.Build(svc, nil)
	app := NewApp(config.Config{Dir: t.TempDir()}, svc, registry, nil)
	app.Init()
	if retained {
		model, cmd := app.Update(sdk.PlanCompletedEvent{Summary: reviewedRiskSummary()})
		app = model.(App)
		drainPolicyAppCommand(&app, cmd)
	}
	model, _ := app.Update(sdk.NavigateMsg{PluginID: pluginID})
	app = model.(App)
	if app.ActivePlugin().ID() != pluginID {
		t.Fatalf("setup: active plugin %s, want %s", app.ActivePlugin().ID(), pluginID)
	}
	if ready := app.ActivePlugin().Ready(); ready {
		t.Fatalf("setup: %s is ready, want its assessment request pending", pluginID)
	}
	return app
}

// A pending policy evaluation holds no directory lock: navigation, pin changes
// and context switches proceed and cancel it. While the host runs terraform plan
// for Risk the lock is held, so the same actions stay refused.
func TestRequireIdle_GivenPendingAssessment_ShouldRefuseOnlyWhileTerraformRuns(t *testing.T) {
	actions := []struct {
		name string
		do   func(t *testing.T, app *App) (applied bool)
	}{
		{"navigation", func(t *testing.T, app *App) bool {
			model, _ := app.Update(sdk.NavigateMsg{PluginID: "state"})
			*app = model.(App)
			return app.ActivePlugin().ID() == "state"
		}},
		{"pin toggle", func(t *testing.T, app *App) bool {
			model, cmd := app.Update(sdk.PinToggleRequestMsg{Addresses: []string{"aws_db_instance.database"}})
			*app = model.(App)
			drainPolicyAppCommand(app, cmd)
			return app.holder.current.Pins.HasAny()
		}},
		{"context switch", func(t *testing.T, app *App) bool {
			model, cmd := app.Update(sdk.ContextSwitchRequestMsg{Workspace: sdk.NewWorkspace("staging")})
			*app = model.(App)
			drainPolicyAppCommand(app, cmd)
			return app.activeWorkspace.String() == "staging"
		}},
	}
	for _, tc := range []struct {
		name        string
		pluginID    string
		retained    bool
		wantApplied bool
	}{
		{"blast radius awaiting policy-only result", "blastradius", true, true},
		{"risk awaiting policy-only result", "risk", true, true},
		{"risk awaiting host-run plan", "risk", false, false},
	} {
		for _, action := range actions {
			t.Run(tc.name+"/"+action.name, func(t *testing.T) {
				app := newPendingAssessmentApp(t, tc.pluginID, tc.retained)

				applied := action.do(t, &app)

				if applied != tc.wantApplied {
					t.Fatalf("%s applied=%v, want %v (commandError=%q)", action.name, applied, tc.wantApplied, app.commandError)
				}
				refused := strings.Contains(app.commandError, tc.pluginID+" is running")
				if refused == tc.wantApplied {
					t.Fatalf("commandError=%q; want a %q refusal exactly when the action is not applied", app.commandError, tc.pluginID+" is running")
				}
			})
		}
	}
}
