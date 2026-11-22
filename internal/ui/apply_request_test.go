package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiapply "github.com/lmarqs/terraform-ui/plugins/apply"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
)

// newReviewedPlanApp pins pins, runs Plan to a plan of summary and returns the
// App and the messages Plan produced when the plan finished.
func newReviewedPlanApp(t *testing.T, pins []string, summary *sdk.PlanSummary) (App, []tea.Msg) {
	t.Helper()
	svc := newMockService("default", nil)
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", tfuiplan.New, plugin.PluginMeta{})
	registry.RegisterFactory("apply", tfuiapply.New, plugin.PluginMeta{})
	registry.Build(svc, nil)
	app := NewApp(config.Config{Dir: t.TempDir()}, svc, registry, nil)
	app.Init()
	if len(pins) > 0 {
		deliver(&app, sdk.PinToggleRequestMsg{Addresses: pins})
	}
	model, _ := app.Update(sdk.NavigateMsg{PluginID: "plan"})
	app = model.(App)
	model, cmd := app.Update(tfuiplan.PlanResultMsg{Summary: summary})
	return model.(App), flatten(cmd)
}

// press sends key to the App and returns the ApplyRequestMsg it produced.
func press(t *testing.T, app *App, key rune) tfuiplan.ApplyRequestMsg {
	t.Helper()
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
	*app = model.(App)
	for _, msg := range flatten(cmd) {
		if req, ok := msg.(tfuiplan.ApplyRequestMsg); ok {
			return req
		}
	}
	t.Fatalf("setup: %q produced no apply request", key)
	return tfuiplan.ApplyRequestMsg{}
}

func applyStarted(app *App) bool {
	return app.ActivePlugin() != nil && app.ActivePlugin().ID() == "apply"
}

// A plan reviewed with A stops being current when a pin toggle or an
// invalidation lands before the App handles the apply request: Apply must not
// run it, and the user stays on the stale plan, where A replans.
func TestApplyRequestMsg_GivenPlanMadeStaleAfterA_ShouldNotStartApply(t *testing.T) {
	cases := []struct {
		name      string
		between   []tea.Msg
		wantApply bool
	}{
		{"nothing in between", nil, true},
		{"pin toggle", []tea.Msg{sdk.PinToggleRequestMsg{Addresses: []string{"aws_s3_bucket.logs"}}}, false},
		{"plan invalidation", []tea.Msg{sdk.PlanInvalidatedEvent{}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}, Action: sdk.ActionUpdate}}}
			app, msgs := newReviewedPlanApp(t, nil, summary)
			deliver(&app, msgs...)
			req := press(t, &app, 'A')
			deliver(&app, tc.between...)

			deliver(&app, req)

			if got := applyStarted(&app); got != tc.wantApply {
				t.Fatalf("Apply started = %v, want %v", got, tc.wantApply)
			}
			if !tc.wantApply && app.ActivePlugin().ID() != "plan" {
				t.Fatalf("active plugin = %q, want plan", app.ActivePlugin().ID())
			}
		})
	}
}

// The apply deferred by a on a stale plan fires with the replan's completion.
// The App's unpin of an address the replan no longer changes is the plan's own
// Context change, so Apply still starts, whichever message lands first.
func TestApplyRequestMsg_GivenDeferredApplyOfAPlanThatPrunesAPin_ShouldStartApply(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		name := map[bool]string{false: "completion first", true: "apply request first"}[reversed]
		t.Run(name, func(t *testing.T) {
			summary := &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}, Action: sdk.ActionUpdate}}}
			app, msgs := newReviewedPlanApp(t, []string{"aws_s3_bucket.logs", prunedPin}, summary)
			// Hold the completion so the plan keeps both pins until the replan.
			_, rest := splitCompletion(t, msgs)
			deliver(&app, rest...)
			deliver(&app, sdk.PlanInvalidatedEvent{})
			model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
			app = model.(App)
			if !app.ActivePlugin().(*tfuiplan.Plugin).Busy() {
				t.Fatalf("setup: a did not replan the stale plan (%d messages)", len(flatten(cmd)))
			}
			model, cmd = app.Update(tfuiplan.PlanResultMsg{Summary: summary})
			app = model.(App)
			replanned := flatten(cmd)
			if reversed {
				for i, j := 0, len(replanned)-1; i < j; i, j = i+1, j-1 {
					replanned[i], replanned[j] = replanned[j], replanned[i]
				}
			}

			deliver(&app, replanned...)

			if app.holder.current.Pins.Contains(prunedPin) {
				t.Fatalf("setup: the replan kept the pin it no longer changes: %v", app.holder.current.Pins)
			}
			if !applyStarted(&app) {
				t.Fatal("the deferred apply did not start Apply")
			}
		})
	}
}
