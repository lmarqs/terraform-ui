package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	tfexec "github.com/lmarqs/terraform-ui/internal/terraform/exec"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
)

const rereadPlanJSON = `{"format_version": "1.0", "terraform_version": "1.5.0", "planned_values": {}, "resource_changes": [
  {"address": "null_resource.a", "type": "null_resource", "name": "a", "provider_name": "registry.terraform.io/hashicorp/null",
   "change": {"actions": ["create"], "before": null, "after": {}, "after_unknown": {}}}]}`

// A failed ^r on a -plan file leaves no plan for the plugins that ask the host
// for one (Risk, Phantom): they get the read error until the file is fixed, and
// terraform never runs.
func TestCtrlR_GivenPlanFileThatFailsToReread_ShouldLeaveNoPlanUntilTheFileIsFixed(t *testing.T) {
	planPath := filepath.Join(t.TempDir(), "plan.json")
	writeRereadPlan := func() {
		t.Helper()
		if err := os.WriteFile(planPath, []byte(rereadPlanJSON), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeRereadPlan()
	cache := terraform.NewServiceCache()
	if err := cache.SeedPlan(planPath, nil); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	// The binary does not exist: any real `terraform plan` fails.
	svc := tfexec.NewExecService(dir, "/nonexistent/terraform", cache)

	var deps *sdk.PluginDeps
	recorder := &planEventRecorder{}
	recorder.mockPluginWithInit = mockPluginWithInit{
		mockPlugin: mockPlugin{id: "planprobe"},
		initFn:     func(d *sdk.PluginDeps) tea.Cmd { deps = d; return nil },
	}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", tfuiplan.New, plugin.PluginMeta{})
	registry.RegisterFactory("planprobe", func(_ sdk.Service) sdk.Plugin { return recorder }, plugin.PluginMeta{})
	registry.Build(svc, nil)
	app := NewApp(config.Config{Dir: dir}, svc, registry, nil)
	drainPolicyAppCommand(&app, app.Init())
	if deps == nil || deps.RequestPlan == nil {
		t.Fatal("app did not provide RequestPlan to plugins")
	}
	model, cmd := app.Update(sdk.NavigateMsg{PluginID: "plan"})
	app = model.(App)
	drainPolicyAppCommand(&app, cmd)

	// fetched asks the host for a plan, as Risk and Phantom do.
	fetched := func() sdk.PlanFetchedEvent {
		t.Helper()
		recorder.events = nil
		_, cmd := deps.RequestPlan()
		drainPolicyAppCommand(&app, cmd)
		if len(recorder.events) != 1 {
			t.Fatalf("got %d PlanFetchedEvents, want 1: %+v", len(recorder.events), recorder.events)
		}
		return recorder.events[0]
	}
	reread := func() {
		t.Helper()
		model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
		app = model.(App)
		drainPolicyAppCommand(&app, cmd)
	}

	if ev := fetched(); ev.Err != nil || ev.Summary == nil || len(ev.Summary.Changes) != 1 {
		t.Fatalf("before the failure: event = %+v, want the seeded plan", ev)
	}

	if err := os.Remove(planPath); err != nil {
		t.Fatal(err)
	}
	reread()

	ev := fetched()
	if ev.Summary != nil {
		t.Fatalf("after the failed ^r: Risk/Phantom received the old plan %+v, want none", ev.Summary)
	}
	if ev.Err == nil || !strings.Contains(ev.Err.Error(), "reading plan file") {
		t.Fatalf("after the failed ^r: Err = %v, want the plan file read error and no terraform run", ev.Err)
	}

	writeRereadPlan()

	if ev := fetched(); ev.Err != nil || ev.Summary == nil || len(ev.Summary.Changes) != 1 {
		t.Fatalf("after the file is fixed: event = %+v, want the plan restored without a terraform run", ev)
	}

	reread()

	if ev := fetched(); ev.Err != nil || ev.Summary == nil || len(ev.Summary.Changes) != 1 {
		t.Fatalf("after the fixed ^r: event = %+v, want the plan still served without a terraform run", ev)
	}
}
