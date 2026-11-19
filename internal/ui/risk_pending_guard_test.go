package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/riskassess"

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
	app, _ := newPendingAssessmentAppWithRequest(t, pluginID, retained)
	return app
}

// newPendingAssessmentAppWithRequest also returns the Cmd that opening pluginID
// produced, which carries its unanswered assessment request.
func newPendingAssessmentAppWithRequest(t *testing.T, pluginID string, retained bool) (App, tea.Cmd) {
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
	model, request := app.Update(sdk.NavigateMsg{PluginID: pluginID})
	app = model.(App)
	if app.ActivePlugin().ID() != pluginID {
		t.Fatalf("setup: active plugin %s, want %s", app.ActivePlugin().ID(), pluginID)
	}
	if ready := app.ActivePlugin().Ready(); ready {
		t.Fatalf("setup: %s is ready, want its assessment request pending", pluginID)
	}
	return app, request
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

// riskResult runs cmd and the Cmds it batches, and returns the assessment
// result among their messages.
func riskResult(t *testing.T, cmd tea.Cmd) riskassess.Result {
	t.Helper()
	if cmd == nil {
		t.Fatal("no assessment request Cmd")
	}
	switch msg := cmd().(type) {
	case riskassess.Result:
		return msg
	case tea.BatchMsg:
		for _, sub := range msg {
			if sub == nil {
				continue
			}
			if result, ok := sub().(riskassess.Result); ok {
				return result
			}
		}
	}
	t.Fatal("Cmd produced no assessment result")
	return riskassess.Result{}
}

// Navigation cancels the plugin's policy-only assessment without advancing the
// generation, and the plugin it opens issues its own request. The cancelled
// request's late result still reaches every plugin and its requester ignores it
// by AssessmentID; the new request's result is neither dropped nor ignored.
func TestNavigate_GivenPendingPolicyOnlyAssessment_ShouldDeliverLateResultThatThePluginIgnores(t *testing.T) {
	for _, tc := range []struct{ from, to string }{
		{"blastradius", "risk"},
		{"risk", "blastradius"},
	} {
		t.Run(tc.from+" to "+tc.to, func(t *testing.T) {
			app, request := newPendingAssessmentAppWithRequest(t, tc.from, true)
			requester := app.ActivePlugin()
			model, next := app.Update(sdk.NavigateMsg{PluginID: tc.to})
			app = model.(App)
			if app.ActivePlugin().ID() != tc.to {
				t.Fatalf("navigation refused: %q", app.commandError)
			}
			target := app.ActivePlugin()
			if target.Ready() {
				t.Fatalf("setup: %s is ready, want its own assessment request pending", tc.to)
			}
			before := requester.View(120, 40)

			late := riskResult(t, request)
			current := riskResult(t, next)
			if !app.risk.Current(late) {
				t.Fatalf("late result of %s's cancelled request is dropped after %s requested", tc.from, tc.to)
			}
			if !app.risk.Current(current) {
				t.Fatalf("%s's pending result is dropped", tc.to)
			}
			model, _ = app.Update(late)
			app = model.(App)

			if after := requester.View(120, 40); after != before {
				t.Fatalf("%s accepted the late result of its cancelled request:\nbefore:\n%s\nafter:\n%s", tc.from, before, after)
			}
			// Risk renders its own cancellation like a cancelled result, so its
			// retained assessment tells whether it accepted the result.
			if assessed, ok := requester.(interface{ Assessment() *sdk.PolicyAssessment }); ok && assessed.Assessment() != nil {
				t.Fatalf("%s retained the late result's assessment %+v", tc.from, assessed.Assessment())
			}
			if target.Ready() {
				t.Fatalf("%s accepted the result of %s's cancelled request", tc.to, tc.from)
			}

			model, _ = app.Update(current)
			app = model.(App)
			if !target.Ready() {
				t.Fatalf("%s ignored the result of its own pending request", tc.to)
			}
		})
	}
}
