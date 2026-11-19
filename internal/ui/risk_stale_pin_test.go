package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/riskassess"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
)

// flatten runs cmd and the Cmds it batches, in order, and returns every message.
func flatten(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var msgs []tea.Msg
	for _, sub := range batch {
		msgs = append(msgs, flatten(sub)...)
	}
	return msgs
}

// A plan that no longer contains a pinned address asks the App to unpin it
// while the host still assesses that plan. The unpin must apply (#130: a
// pending assessment holds no lock), and the Context change it causes marks
// Plan stale and drops the assessment's late result.
func TestPinToggle_GivenPlanCompletedWithoutPinnedAddressWhileAssessmentPending_ShouldUnpinAndDropLateResult(t *testing.T) {
	const stale = "aws_db_instance.database"
	svc := newMockService("default", nil)
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", tfuiplan.New, plugin.PluginMeta{})
	registry.Build(svc, nil)
	app := NewApp(config.Config{Dir: t.TempDir()}, svc, registry, nil)
	app.Init()
	drainPolicyAppCommand(&app, func() tea.Msg { return sdk.PinToggleRequestMsg{Addresses: []string{stale}} })
	if !app.holder.current.Pins.HasAny() {
		t.Fatal("setup: pin was not applied")
	}
	model, _ := app.Update(sdk.NavigateMsg{PluginID: "plan"})
	app = model.(App)
	plan := app.ActivePlugin().(*tfuiplan.Plugin)
	if !plan.Busy() {
		t.Fatal("setup: Plan is not running its plan")
	}

	// The plan completes without the pinned address. Plan requests its
	// assessment during this Update; running the returned Cmds computes the
	// result, which the test holds undelivered so the assessment stays pending.
	reviewed := reviewedRiskSummary()
	reviewed.Changes = []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}}}
	model, cmd := app.Update(tfuiplan.PlanResultMsg{Summary: reviewed})
	app = model.(App)
	var late *riskassess.Result
	var prune *sdk.PinToggleRequestMsg
	for _, msg := range flatten(cmd) {
		switch msg := msg.(type) {
		case riskassess.Result:
			late = &msg
		case sdk.PinToggleRequestMsg:
			prune = &msg
		}
	}
	if late == nil || prune == nil {
		t.Fatalf("plan completion produced result=%v unpin request=%v, want both", late, prune)
	}
	if plan.Ready() {
		t.Fatal("setup: Plan is ready, want its assessment pending")
	}
	if !app.risk.Current(*late) {
		t.Fatal("setup: the assessment is already dropped")
	}

	model, next := app.Update(*prune)
	app = model.(App)
	drainPolicyAppCommand(&app, next)

	if app.commandError != "" {
		t.Fatalf("unpin was refused while the assessment was pending: %q", app.commandError)
	}
	if app.holder.current.Pins.HasAny() {
		t.Fatalf("stale pin survived the plan: %v", app.holder.current.Pins)
	}
	if plan.Busy() || !plan.Ready() {
		t.Fatalf("Plan Busy=%v Ready=%v after the pin change, want idle and ready", plan.Busy(), plan.Ready())
	}
	if app.risk.Current(*late) {
		t.Fatal("the pin change did not drop the pending assessment")
	}

	model, _ = app.Update(*late)
	app = model.(App)
	if plan.Assessment() != nil {
		t.Fatalf("Plan accepted the dropped assessment: %+v", plan.Assessment())
	}
	// A stale Plan replans on its next activation; a current one would reassess.
	plan.Activate(tfuiplan.Input{})
	if !plan.Busy() {
		t.Fatal("Plan is not stale after the pin change: activation did not replan")
	}
}
