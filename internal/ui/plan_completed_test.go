package ui

import (
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiblastradius "github.com/lmarqs/terraform-ui/plugins/blastradius"
	tfuiphantom "github.com/lmarqs/terraform-ui/plugins/phantom"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
)

const prunedPin = "aws_db_instance.gone"

// newPlanCompletionApp runs Plan with prunedPin pinned and returns the App and
// the messages Plan produced once its plan finished without that address.
func newPlanCompletionApp(t *testing.T) (App, []tea.Msg) {
	t.Helper()
	svc := newMockService("default", nil)
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", tfuiplan.New, plugin.PluginMeta{})
	registry.RegisterFactory("risk", tfuirisk.New, plugin.PluginMeta{})
	registry.RegisterFactory("blastradius", tfuiblastradius.New, plugin.PluginMeta{})
	registry.RegisterFactory("phantom", tfuiphantom.New, plugin.PluginMeta{})
	registry.Build(svc, nil)
	app := NewApp(config.Config{Dir: t.TempDir()}, svc, registry, nil)
	app.Init()
	drainPolicyAppCommand(&app, func() tea.Msg { return sdk.PinToggleRequestMsg{Addresses: []string{prunedPin}} })
	model, _ := app.Update(sdk.NavigateMsg{PluginID: "plan"})
	app = model.(App)
	if !app.ActivePlugin().(*tfuiplan.Plugin).Busy() {
		t.Fatal("setup: Plan is not running its plan")
	}
	reviewed := reviewedRiskSummary()
	reviewed.Changes = []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}, Action: sdk.ActionUpdate}}
	model, cmd := app.Update(tfuiplan.PlanResultMsg{Summary: reviewed})
	return model.(App), flatten(cmd)
}

// planResults reports which of Risk, Blast Radius and Phantom hold a plan:
// activating each one shows whether it retained the completed plan.
func planResults(t *testing.T, app *App) (risk, blastRadius, phantom bool) {
	t.Helper()
	plugin := func(id string) sdk.Plugin {
		p, ok := app.registry.ByID(id)
		if !ok {
			t.Fatalf("setup: %s not registered", id)
		}
		return p
	}
	r := plugin("risk").(*tfuirisk.Plugin)
	r.Activate(tfuirisk.Input{})
	b := plugin("blastradius").(*tfuiblastradius.Plugin)
	p := plugin("phantom").(*tfuiphantom.Plugin)
	return !r.Busy(), b.Activate() != nil, p.Status() == sdk.StatusDone
}

func deliver(app *App, msgs ...tea.Msg) {
	for _, msg := range msgs {
		drainPolicyAppCommand(app, func() tea.Msg { return msg })
	}
}

// splitCompletion separates Plan's completion from the other messages it produced.
func splitCompletion(t *testing.T, msgs []tea.Msg) (sdk.PlanCompletedEvent, []tea.Msg) {
	t.Helper()
	var completion *sdk.PlanCompletedEvent
	var rest []tea.Msg
	for _, msg := range msgs {
		if ev, ok := msg.(sdk.PlanCompletedEvent); ok {
			completion = &ev
			continue
		}
		rest = append(rest, msg)
	}
	if completion == nil {
		t.Fatal("setup: Plan published no completion")
	}
	return *completion, rest
}

// replan runs Plan again to a plan of summary and delivers everything it produced.
func replan(t *testing.T, app *App, summary *sdk.PlanSummary) {
	t.Helper()
	plan := app.ActivePlugin().(*tfuiplan.Plugin)
	plan.Refresh()
	if !plan.Busy() {
		t.Fatal("setup: Plan is not running its second plan")
	}
	model, cmd := app.Update(tfuiplan.PlanResultMsg{Summary: summary})
	*app = model.(App)
	drainPolicyAppCommand(app, cmd)
}

// A completion still queued when its plan stops being current must not reach
// Risk, Blast Radius and Phantom, whether an invalidation or a newer plan's
// completion came first in the same Context.
func TestPlanCompletedEvent_GivenInvalidationOrNewerPlanBeforeIt_ShouldBeDropped(t *testing.T) {
	// The newer plan still changes the pinned address, so its completion keeps
	// the Context the earlier completion ran against.
	newer := reviewedRiskSummary()
	newer.Changes = []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}, Action: sdk.ActionUpdate},
		{Resource: sdk.Resource{Address: prunedPin}, Action: sdk.ActionDelete},
	}
	cases := []struct {
		name      string
		supersede func(t *testing.T, app *App)
		check     func(t *testing.T, app *App)
	}{
		{
			name:      "plan invalidation",
			supersede: func(_ *testing.T, app *App) { deliver(app, sdk.PlanInvalidatedEvent{}) },
			check: func(t *testing.T, app *App) {
				risk, blastRadius, phantom := planResults(t, app)
				if risk || blastRadius || phantom {
					t.Fatalf("late completion restored results: risk=%v blast radius=%v phantom=%v, want none", risk, blastRadius, phantom)
				}
			},
		},
		{
			name: "newer plan completion",
			supersede: func(t *testing.T, app *App) {
				replan(t, app, newer)
				for _, id := range []string{"risk", "blastradius"} {
					p, _ := app.registry.ByID(id)
					switch p := p.(type) {
					case *tfuirisk.Plugin:
						drainPolicyAppCommand(app, p.Activate(tfuirisk.Input{}))
					case *tfuiblastradius.Plugin:
						drainPolicyAppCommand(app, p.Activate())
					}
				}
			},
			check: func(t *testing.T, app *App) {
				r, _ := app.registry.ByID("risk")
				b, _ := app.registry.ByID("blastradius")
				p, _ := app.registry.ByID("phantom")
				risk := r.(*tfuirisk.Plugin).Ready()
				blastRadius := b.(*tfuiblastradius.Plugin).Status() == sdk.StatusDone
				phantom := p.(*tfuiphantom.Plugin).TotalCount()
				if !risk || !blastRadius || phantom != len(newer.Changes) {
					t.Fatalf("late completion replaced the newer plan: risk ready=%v blast radius done=%v phantom changes=%d, want true, true, %d", risk, blastRadius, phantom, len(newer.Changes))
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, msgs := newPlanCompletionApp(t)
			completion, rest := splitCompletion(t, msgs)
			deliver(&app, rest...)
			tc.supersede(t, &app)

			deliver(&app, completion)

			tc.check(t, &app)
		})
	}
}

func TestPlanCompletedEvent_GivenContextChangeAfterThePlan_ShouldBeDropped(t *testing.T) {
	changes := []struct {
		name   string
		change func(app *App)
	}{
		{"workspace switch", func(app *App) { switchContext(app, "", "staging") }},
		{"chdir switch", func(app *App) { switchContext(app, "member", "default") }},
		{"pin toggle", func(app *App) { deliver(app, sdk.PinToggleRequestMsg{Addresses: []string{"aws_s3_bucket.logs"}}) }},
	}
	for _, tc := range changes {
		t.Run(tc.name, func(t *testing.T) {
			app, msgs := newPlanCompletionApp(t)
			completion, rest := splitCompletion(t, msgs)
			deliver(&app, rest...)
			tc.change(&app)

			deliver(&app, completion)

			risk, blastRadius, phantom := planResults(t, &app)
			if risk || blastRadius || phantom {
				t.Fatalf("late completion restored results: risk=%v blast radius=%v phantom=%v, want none", risk, blastRadius, phantom)
			}
		})
	}
}

// Plan's result unpins the address it no longer changes; that pin change is the
// plan's own, so it must not cost the plan its completion, whatever order the
// messages arrive in.
func TestPlanCompletedEvent_GivenPlanThatPrunesAPin_ShouldReachRiskBlastRadiusAndPhantom(t *testing.T) {
	app, msgs := newPlanCompletionApp(t)
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	deliver(&app, msgs...)

	if app.holder.current.Pins.Contains(prunedPin) {
		t.Fatalf("the plan kept the pin it no longer changes: %v", app.holder.current.Pins)
	}
	risk, blastRadius, phantom := planResults(t, &app)
	if !risk || !blastRadius || !phantom {
		t.Fatalf("completion lost: risk=%v blast radius=%v phantom=%v, want all", risk, blastRadius, phantom)
	}
}

func TestPlanCompletedEvent_GivenPins_ShouldUnpinOnlyAddressesThePlanNoLongerChanges(t *testing.T) {
	summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "kept"}}}}
	cases := []struct {
		name    string
		pins    sdk.Pins
		busy    bool
		want    sdk.Pins
		refused bool
	}{
		{"every pin still planned", sdk.Pins{"kept"}, false, sdk.Pins{"kept"}, false},
		{"a pin no longer planned", sdk.Pins{"kept", "gone"}, false, sdk.Pins{"kept"}, false},
		{"a pin no longer planned while terraform runs", sdk.Pins{"kept", "gone"}, true, sdk.Pins{"kept", "gone"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := setupTestAppWithBusyPlugin(tc.busy)
			app.Init()
			app.holder.current = app.holder.current.WithPins(tc.pins)

			model, _ := app.Update(sdk.PlanCompletedEvent{Context: app.holder.current, Summary: summary})
			app = model.(App)

			if got := app.holder.current.Pins; !slices.Equal(got, tc.want) {
				t.Errorf("pins = %v, want %v", got, tc.want)
			}
			if refused := app.commandError != ""; refused != tc.refused {
				t.Errorf("unpin refused = %v (%q), want %v", refused, app.commandError, tc.refused)
			}
		})
	}
}
