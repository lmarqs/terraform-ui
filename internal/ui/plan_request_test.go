package ui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

// planEventRecorder is a plugin that receives host plan results only through
// the EventBus.
type planEventRecorder struct {
	mockPluginWithInit
	events []sdk.PlanFetchedEvent
}

func (r *planEventRecorder) HandlePlanFetched(ev sdk.PlanFetchedEvent) tea.Cmd {
	r.events = append(r.events, ev)
	return nil
}

// newPlanRequestApp returns an App whose project selects a risk policy that
// cannot be loaded, so a plan request that touched the policy would fail.
func newPlanRequestApp(t *testing.T, svc *sdktest.MockService) (App, *sdk.PluginDeps, *planEventRecorder) {
	t.Helper()
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "missing.rego" }`)
	rootCfg, err := config.LoadRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	var deps *sdk.PluginDeps
	recorder := &planEventRecorder{}
	recorder.mockPluginWithInit = mockPluginWithInit{
		mockPlugin: mockPlugin{id: "planprobe"},
		initFn:     func(d *sdk.PluginDeps) tea.Cmd { deps = d; return nil },
	}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("planprobe", func(_ sdk.Service) sdk.Plugin { return recorder }, plugin.PluginMeta{})
	registry.Build(nil, nil)
	app := NewApp(config.Config{Dir: root}, svc, registry, rootCfg, &StandaloneConfig{PluginID: "planprobe"})
	drainPolicyAppCommand(&app, app.Init())
	if deps == nil || deps.RequestPlan == nil {
		t.Fatal("app did not provide RequestPlan to plugins")
	}
	return app, deps, recorder
}

func plannedService(summary *sdk.PlanSummary) *sdktest.MockService {
	svc := newMockService("default", nil)
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return summary, nil }
	return svc
}

func TestRequestPlan_GivenUnloadableRiskPolicy_ShouldDispatchThePlanUnderTheReturnedID(t *testing.T) {
	planned := &sdk.PlanSummary{}
	app, deps, recorder := newPlanRequestApp(t, plannedService(planned))

	id, cmd := deps.RequestPlan()
	drainPolicyAppCommand(&app, cmd)

	if len(recorder.events) != 1 {
		t.Fatalf("dispatched %d PlanFetchedEvents, want 1: %+v", len(recorder.events), recorder.events)
	}
	if ev := recorder.events[0]; id == 0 || ev.Request != id || ev.Summary != planned || ev.Err != nil {
		t.Fatalf("event = %+v, want the plan under the nonzero ID %d RequestPlan returned", ev, id)
	}
}

func TestRequestPlan_GivenContextChangeOrPlanInvalidationBeforeDelivery_ShouldDropStaleResult(t *testing.T) {
	for name, invalidate := range map[string]tea.Msg{
		"context switch":    sdk.ContextSwitchRequestMsg{Chdir: "", Workspace: sdk.NewWorkspace("other")},
		"pin change":        sdk.PinToggleRequestMsg{Addresses: []string{"aws_db_instance.database"}},
		"plan invalidation": sdk.PlanInvalidatedEvent{},
	} {
		t.Run(name, func(t *testing.T) {
			app, deps, recorder := newPlanRequestApp(t, plannedService(&sdk.PlanSummary{}))
			_, cmd := deps.RequestPlan()
			result := cmd()

			drainPolicyAppCommand(&app, func() tea.Msg { return invalidate })
			drainPolicyAppCommand(&app, func() tea.Msg { return result })

			if len(recorder.events) != 0 {
				t.Fatalf("stale result dispatched after %s: %+v", name, recorder.events)
			}
		})
	}
}

func TestCancelPlugin_GivenPendingPlanRequest_ShouldCancelTheHostPlan(t *testing.T) {
	svc, started := blockingPlanService()
	app, deps, _ := newPlanRequestApp(t, svc)
	_, cmd := deps.RequestPlan()
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	planCtx := <-started

	deactivateActivePlugin(&app)

	select {
	case <-planCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling the plugin did not cancel the Terraform service Context of its host plan")
	}
	<-done
}
