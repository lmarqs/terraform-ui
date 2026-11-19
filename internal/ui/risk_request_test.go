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

// riskEventRecorder is a plugin that receives host assessment results only
// through the EventBus, as every risk consumer must.
type riskEventRecorder struct {
	mockPluginWithInit
	events []sdk.RiskAssessedEvent
}

func (r *riskEventRecorder) HandleRiskAssessed(ev sdk.RiskAssessedEvent) tea.Cmd {
	r.events = append(r.events, ev)
	return nil
}

func newAssessmentRequestApp(t *testing.T, root string, loaded bool, svc *sdktest.MockService) (App, *sdk.PluginDeps, *riskEventRecorder) {
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
	recorder := &riskEventRecorder{}
	recorder.mockPluginWithInit = mockPluginWithInit{
		mockPlugin: mockPlugin{id: "riskprobe"},
		initFn:     func(d *sdk.PluginDeps) tea.Cmd { deps = d; return nil },
	}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("riskprobe", func(_ sdk.Service) sdk.Plugin { return recorder }, plugin.PluginMeta{})
	registry.Build(nil, nil)
	app := NewApp(config.Config{Dir: root}, svc, registry, rootCfg, &StandaloneConfig{PluginID: "riskprobe"})
	drainPolicyAppCommand(&app, app.Init())
	if deps == nil || deps.RequestAssessment == nil {
		t.Fatal("app did not provide RequestAssessment to plugins")
	}
	return app, deps, recorder
}

// deliverAssessment runs the request Cmd off the Update loop, as bubbletea does,
// then hands its message to the App for dispatch.
func deliverAssessment(app *App, cmd tea.Cmd) {
	drainPolicyAppCommand(app, cmd)
}

func onlyRiskEvent(t *testing.T, recorder *riskEventRecorder) sdk.RiskAssessedEvent {
	t.Helper()
	if len(recorder.events) != 1 {
		t.Fatalf("dispatched %d RiskAssessedEvents, want 1: %+v", len(recorder.events), recorder.events)
	}
	return recorder.events[0]
}

func TestRequestAssessment_ShouldDispatchTheEventCarryingTheReturnedID(t *testing.T) {
	app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, newMockService("default", nil))
	for range 2 {
		recorder.events = nil
		id, cmd := deps.RequestAssessment(reviewedRiskSummary())
		deliverAssessment(&app, cmd)
		if got := onlyRiskEvent(t, recorder).Request; id == 0 || got != id {
			t.Fatalf("dispatched event Request = %d, want the nonzero ID %d RequestAssessment returned", got, id)
		}
	}
}

func TestRequestAssessment_GivenContextChangeOrPlanInvalidationAfterRequest_ShouldDropStaleResult(t *testing.T) {
	invalidations := map[string]tea.Msg{
		"context switch":    sdk.ContextSwitchRequestMsg{Chdir: "", Workspace: sdk.NewWorkspace("other")},
		"pin change":        sdk.PinToggleRequestMsg{Addresses: []string{"aws_db_instance.database"}},
		"plan invalidation": sdk.PlanInvalidatedEvent{},
	}
	for name, invalidate := range invalidations {
		t.Run(name+" while request runs", func(t *testing.T) {
			app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, newMockService("default", nil))
			cmd := cmdOf(deps.RequestAssessment(reviewedRiskSummary()))

			drainPolicyAppCommand(&app, cmd) // positive control: a current result is dispatched
			onlyRiskEvent(t, recorder)
			recorder.events = nil

			cmd = cmdOf(deps.RequestAssessment(reviewedRiskSummary()))
			drainPolicyAppCommand(&app, func() tea.Msg { return invalidate })
			deliverAssessment(&app, cmd)

			if len(recorder.events) != 0 {
				t.Fatalf("stale result dispatched after %s: %+v", name, recorder.events)
			}
		})
		t.Run(name+" before delivery", func(t *testing.T) {
			app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, newMockService("default", nil))
			result := cmdOf(deps.RequestAssessment(reviewedRiskSummary()))()

			drainPolicyAppCommand(&app, func() tea.Msg { return invalidate })
			drainPolicyAppCommand(&app, func() tea.Msg { return result })

			if len(recorder.events) != 0 {
				t.Fatalf("stale result dispatched after %s: %+v", name, recorder.events)
			}
		})
	}
}

// blockingPlanService runs a Terraform plan that lasts until its context ends,
// as exec.CommandContext does, and reports that context once the plan starts.
func blockingPlanService() (*sdktest.MockService, <-chan context.Context) {
	started := make(chan context.Context, 1)
	svc := newMockService("default", nil)
	svc.PlanFn = func(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
		started <- ctx
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return svc, started
}

func TestCancel_GivenHostPlanForNilSummary_ShouldCancelTerraformContext(t *testing.T) {
	for name, cancel := range map[string]func(*App){
		"deactivate": func(app *App) {
			model, _ := app.Update(sdk.DeactivateMsg{})
			*app = model.(App)
		},
		"navigate":   func(app *App) { app.navigateTo(app.activePlugin) },
		"force quit": func(app *App) { app.cmdForceQuit() },
	} {
		t.Run(name, func(t *testing.T) {
			svc, started := blockingPlanService()
			app, deps, _ := newAssessmentRequestApp(t, t.TempDir(), false, svc)
			done := make(chan tea.Msg, 1)
			cmd := cmdOf(deps.RequestAssessment(nil))
			go func() { done <- cmd() }()
			planCtx := <-started

			cancel(&app)

			select {
			case <-planCtx.Done():
			case <-time.After(5 * time.Second):
				t.Fatal("Cancel did not cancel the Terraform service Context of the host plan")
			}
			<-done
		})
	}
}

func TestRequestAssessment_GivenContextChangeDuringHostPlan_ShouldCancelTerraformContextAndDropResult(t *testing.T) {
	for name, change := range map[string]tea.Msg{
		"context switch": sdk.ContextSwitchRequestMsg{Chdir: "", Workspace: sdk.NewWorkspace("other")},
		"pin change":     sdk.PinToggleRequestMsg{Addresses: []string{"aws_db_instance.database"}},
	} {
		t.Run(name, func(t *testing.T) {
			svc, started := blockingPlanService()
			app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)
			cmd := cmdOf(deps.RequestAssessment(nil))
			done := make(chan tea.Msg, 1)
			go func() { done <- cmd() }()
			planCtx := <-started

			drainPolicyAppCommand(&app, func() tea.Msg { return change })

			select {
			case <-planCtx.Done():
			case <-time.After(5 * time.Second):
				t.Fatalf("%s did not cancel the Terraform service Context of the host plan", name)
			}
			drainPolicyAppCommand(&app, func() tea.Msg { return <-done })
			if len(recorder.events) != 0 {
				t.Fatalf("result superseded by %s was dispatched: %+v", name, recorder.events)
			}
		})
	}
}

func deactivateActivePlugin(app *App) {
	model, _ := app.Update(sdk.DeactivateMsg{})
	*app = model.(App)
}

// assessmentFor returns the one dispatched event carrying id.
func assessmentFor(t *testing.T, recorder *riskEventRecorder, id sdk.AssessmentID) sdk.RiskAssessedEvent {
	t.Helper()
	var found []sdk.RiskAssessedEvent
	for _, ev := range recorder.events {
		if ev.Request == id {
			found = append(found, ev)
		}
	}
	if len(found) != 1 {
		t.Fatalf("dispatched %d events for request %d, want 1: %+v", len(found), id, recorder.events)
	}
	return found[0]
}

// A cancelled request's result is dispatched under its own ID; a requester
// that requested again keeps only the event carrying the new ID.
func TestRequestAssessment_GivenNilSummaryRequestCancelledThenRequestedAgain_ShouldDispatchEachResultUnderItsOwnID(t *testing.T) {
	planned := reviewedRiskSummary()
	svc := newMockService("default", nil)
	svc.PlanFn = func(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return planned, nil
	}
	for _, tc := range []struct {
		name      string
		cancelled func(*App) (sdk.AssessmentID, tea.Cmd)
	}{
		{"cancelled during host plan", func(app *App) (sdk.AssessmentID, tea.Cmd) {
			id, cmd := app.risk.Request("riskprobe", app.holder.current, nil)
			deactivateActivePlugin(app)
			return id, cmd
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)
			staleID, staleCmd := tc.cancelled(&app)
			stale := staleCmd()
			currentID, current := deps.RequestAssessment(nil)

			drainPolicyAppCommand(&app, func() tea.Msg { return stale })
			deliverAssessment(&app, current)

			if ev := assessmentFor(t, recorder, staleID); ev.Err == nil && ev.Assessment.Status != sdk.AssessmentCancelled {
				t.Fatalf("cancelled request's result = %+v, want a cancellation", ev)
			}
			if ev := assessmentFor(t, recorder, currentID); ev.Err != nil || ev.Summary != planned || !ev.Assessment.Usable() {
				t.Fatalf("new request's result = %+v, want its usable assessment of %p", ev, planned)
			}
		})
	}
}

// A request cancelled before an identical one is issued for the same summary
// still delivers its result, under the cancelled ID, never the new one. That a
// requester ignores it by ID is tested per plugin, e.g. Blast Radius's "older own
// ID after a newer request" case.
func TestRequestAssessment_GivenSuppliedSummaryRequestCancelledThenRequestedAgain_ShouldDispatchEachResultUnderItsOwnID(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	classifyAll := func(level string) string {
		return "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"" + level + "\",\"reasons\":[{\"code\":\"edited\",\"message\":\"Edited policy.\",\"paths\":[]}]}\n"
	}
	writePolicyFile(t, root, "policy.rego", classifyAll("low"))
	app, deps, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
	summary := reviewedRiskSummary()

	staleID, staleCmd := deps.RequestAssessment(summary)
	stale := staleCmd()
	deactivateActivePlugin(&app)
	writePolicyFile(t, root, "policy.rego", classifyAll("high"))
	currentID, current := deps.RequestAssessment(summary)

	drainPolicyAppCommand(&app, func() tea.Msg { return stale })
	deliverAssessment(&app, current)

	if ev := assessmentFor(t, recorder, staleID); ev.Assessment.OverallLevel == nil || *ev.Assessment.OverallLevel != sdk.RiskLow {
		t.Fatalf("cancelled request's assessment = %+v, want the low classification it computed", ev.Assessment)
	}
	if ev := assessmentFor(t, recorder, currentID); ev.Assessment.OverallLevel == nil || *ev.Assessment.OverallLevel != sdk.RiskHigh {
		t.Fatalf("new request's assessment = %+v, want the high classification", ev.Assessment)
	}
}

func TestQuitKey_GivenIdlePluginWithHostPlan_ShouldCancelTerraformContext(t *testing.T) {
	svc, started := blockingPlanService()
	var deps *sdk.PluginDeps
	probe := &mockPluginWithInit{
		mockPlugin: mockPlugin{id: "riskprobe"},
		initFn:     func(d *sdk.PluginDeps) tea.Cmd { deps = d; return nil },
	}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("riskprobe", func(_ sdk.Service) sdk.Plugin { return probe }, plugin.PluginMeta{})
	registry.Build(nil, nil)
	app := NewApp(config.Config{Dir: t.TempDir()}, svc, registry, nil)
	drainPolicyAppCommand(&app, app.Init())
	model, _ := app.Update(sdk.NavigateMsg{PluginID: "riskprobe"})
	app = model.(App)
	done := make(chan tea.Msg, 1)
	cmd := cmdOf(deps.RequestAssessment(nil))
	go func() { done <- cmd() }()
	planCtx := <-started

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	app = model.(App)

	select {
	case <-planCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("leaving an idle plugin with q left the host plan running")
	}
	<-done
}

// cmdOf drops the request ID for callers that only run the request.
func cmdOf(_ sdk.AssessmentID, cmd tea.Cmd) tea.Cmd { return cmd }

func TestCancelPlugin_GivenHostPlanForNilSummary_ShouldCancelOnlyThatPluginsRequest(t *testing.T) {
	svc, started := blockingPlanService()
	deps := map[string]*sdk.PluginDeps{}
	registry := plugin.NewRegistry()
	probes := map[string]sdk.Plugin{}
	for _, id := range []string{"requester", "other"} {
		probe := &mockPluginWithInit{
			mockPlugin: mockPlugin{id: id},
			initFn:     func(d *sdk.PluginDeps) tea.Cmd { deps[id] = d; return nil },
		}
		probes[id] = probe
		registry.RegisterFactory(id, func(_ sdk.Service) sdk.Plugin { return probe }, plugin.PluginMeta{})
	}
	registry.Build(nil, nil)
	app := NewApp(config.Config{Dir: t.TempDir()}, svc, registry, nil)
	drainPolicyAppCommand(&app, app.Init())
	done := make(chan tea.Msg, 1)
	cmd := cmdOf(deps["requester"].RequestAssessment(nil))
	go func() { done <- cmd() }()
	planCtx := <-started

	app.cancelPlugin(probes["other"])
	if planCtx.Err() != nil {
		t.Fatalf("cancelling another plugin cancelled the requester's host plan: %v", planCtx.Err())
	}
	app.cancelPlugin(probes["requester"])

	select {
	case <-planCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling the requester did not cancel its host plan")
	}
	<-done
}
