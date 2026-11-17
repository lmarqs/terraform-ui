package ui

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/riskpolicy"
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

func TestRequestAssessment_GivenBrokenPolicyAndNilSummary_ShouldPublishUnavailableWithoutRunningPlan(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "missing.rego" }`)
	svc := newMockService("default", nil)
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		t.Error("Service.Plan ran although the selected policy cannot be loaded")
		return nil, errors.New("plan must not run")
	}
	app, deps, recorder := newAssessmentRequestApp(t, root, true, svc)

	deliverAssessment(&app, deps.RequestAssessment(nil))

	ev := onlyRiskEvent(t, recorder)
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Service.Plan called %d times, want 0", len(svc.PlanCalls))
	}
	if ev.Err != nil {
		t.Fatalf("policy load failure surfaced as plan error: %v", ev.Err)
	}
	if ev.Assessment.Status != sdk.AssessmentUnavailable || ev.Assessment.ErrorCode != sdk.ErrorSourceUnreadable || ev.Assessment.OverallLevel != nil {
		t.Fatalf("assessment = %+v, want unavailable with source_unreadable", ev.Assessment)
	}
}

func TestRequestAssessment_GivenValidPolicyAndNilSummary_ShouldPlanOnceAndPublishThatSummary(t *testing.T) {
	planned := reviewedRiskSummary()
	svc := newMockService("default", nil)
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return planned, nil }
	app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)

	deliverAssessment(&app, deps.RequestAssessment(nil))

	ev := onlyRiskEvent(t, recorder)
	if len(svc.PlanCalls) != 1 {
		t.Fatalf("Service.Plan called %d times, want 1", len(svc.PlanCalls))
	}
	if svc.PlanCalls[0].PlanFile == "" {
		t.Fatal("host plan fetch did not write to a temp plan file")
	}
	if ev.Err != nil || ev.Summary != planned {
		t.Fatalf("event summary = %p err = %v, want planned summary %p", ev.Summary, ev.Err, planned)
	}
	if !ev.Assessment.Usable() || ev.Assessment.PlanID != "reviewed-plan" {
		t.Fatalf("assessment of planned summary = %+v, want usable for reviewed-plan", ev.Assessment)
	}
}

func TestRequestAssessment_GivenSummary_ShouldEvaluateItWithoutRunningPlan(t *testing.T) {
	given := reviewedRiskSummary()
	svc := newMockService("default", nil)
	app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)

	deliverAssessment(&app, deps.RequestAssessment(given))

	ev := onlyRiskEvent(t, recorder)
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Service.Plan called %d times for a supplied summary, want 0", len(svc.PlanCalls))
	}
	if ev.Summary != given || ev.Err != nil || !ev.Assessment.Usable() {
		t.Fatalf("event = %+v, want usable assessment of the supplied summary", ev)
	}
}

func TestRequestAssessment_GivenPlanFailure_ShouldPublishErrorWithoutEvaluation(t *testing.T) {
	planErr := errors.New("terraform plan failed")
	svc := newMockService("default", nil)
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return nil, planErr }
	app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)

	deliverAssessment(&app, deps.RequestAssessment(nil))

	ev := onlyRiskEvent(t, recorder)
	if !errors.Is(ev.Err, planErr) {
		t.Fatalf("event err = %v, want %v", ev.Err, planErr)
	}
	if ev.Assessment.Status != "" || ev.Assessment.Usable() {
		t.Fatalf("plan failure was evaluated: %+v", ev.Assessment)
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
			cmd := deps.RequestAssessment(reviewedRiskSummary())

			drainPolicyAppCommand(&app, cmd) // positive control: a current result is dispatched
			onlyRiskEvent(t, recorder)
			recorder.events = nil

			cmd = deps.RequestAssessment(reviewedRiskSummary())
			drainPolicyAppCommand(&app, func() tea.Msg { return invalidate })
			deliverAssessment(&app, cmd)

			if len(recorder.events) != 0 {
				t.Fatalf("stale result dispatched after %s: %+v", name, recorder.events)
			}
		})
		t.Run(name+" before delivery", func(t *testing.T) {
			app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, newMockService("default", nil))
			result := deps.RequestAssessment(reviewedRiskSummary())()

			drainPolicyAppCommand(&app, func() tea.Msg { return invalidate })
			drainPolicyAppCommand(&app, func() tea.Msg { return result })

			if len(recorder.events) != 0 {
				t.Fatalf("stale result dispatched after %s: %+v", name, recorder.events)
			}
		})
	}
}

func TestRequestAssessment_GivenNilSummary_ShouldPlanWithActiveContextOptionsAndRemoveTemporaryArtifact(t *testing.T) {
	var artifact string
	svc := newMockService("default", nil)
	svc.PlanFn = func(_ context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
		artifact = opts.PlanFile
		if err := os.WriteFile(artifact, []byte("saved plan"), 0600); err != nil {
			t.Fatal(err)
		}
		return reviewedRiskSummary(), nil
	}
	app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)

	requestRisk(t, &app, recorder, deps.RequestAssessment(nil))

	if len(svc.PlanCalls) != 1 {
		t.Fatalf("Service.Plan called %d times, want 1", len(svc.PlanCalls))
	}
	got, want := svc.PlanCalls[0], app.holder.current.PlanOptions()
	got.PlanFile, got.Writer = "", nil
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("plan options = %#v, want active Context %#v", got, want)
	}
	if artifact == "" {
		t.Fatal("host plan must supply a temporary artifact instead of Terraform's tfplan.out default")
	}
	if _, err := os.Stat(artifact); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary artifact remains: %v", err)
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
			cmd := deps.RequestAssessment(nil)
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

func TestCancel_GivenSuppliedSummary_ShouldKeepEvaluatingIt(t *testing.T) {
	app, _, recorder := newAssessmentRequestApp(t, t.TempDir(), false, newMockService("default", nil))
	evaluating := make(chan context.Context, 1)
	release := make(chan struct{})
	cmd := app.requestAssessmentWith(reviewedRiskSummary(), func(ctx context.Context, _ riskpolicy.Snapshot, _ sdk.PolicyEvidence) sdk.PolicyAssessment {
		evaluating <- ctx
		<-release
		return sdk.PolicyAssessment{Status: sdk.AssessmentEmpty}
	})
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	ctx := <-evaluating

	app.cmdForceQuit()
	close(release)

	if ctx.Err() != nil {
		t.Fatalf("Cancel cancelled evaluation of a supplied summary: %v", ctx.Err())
	}
	drainPolicyAppCommand(&app, func() tea.Msg { return <-done })
	if event := onlyRiskEvent(t, recorder); event.Assessment.Status != sdk.AssessmentEmpty {
		t.Fatalf("supplied summary result = %+v, want the evaluated assessment", event.Assessment)
	}
}

func TestRequestAssessment_GivenContextChangeDuringSuppliedSummaryEvaluation_ShouldCancelEvaluationContext(t *testing.T) {
	for name, change := range map[string]tea.Msg{
		"context switch": sdk.ContextSwitchRequestMsg{Chdir: "", Workspace: sdk.NewWorkspace("other")},
		"pin change":     sdk.PinToggleRequestMsg{Addresses: []string{"aws_db_instance.database"}},
	} {
		t.Run(name, func(t *testing.T) {
			app, _, recorder := newAssessmentRequestApp(t, t.TempDir(), false, newMockService("default", nil))
			evaluating := make(chan context.Context, 1)
			cmd := app.requestAssessmentWith(reviewedRiskSummary(), func(ctx context.Context, _ riskpolicy.Snapshot, _ sdk.PolicyEvidence) sdk.PolicyAssessment {
				evaluating <- ctx
				<-ctx.Done()
				return sdk.PolicyAssessment{Status: sdk.AssessmentCancelled}
			})
			done := make(chan tea.Msg, 1)
			go func() { done <- cmd() }()
			ctx := <-evaluating

			drainPolicyAppCommand(&app, func() tea.Msg { return change })

			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
				t.Fatalf("%s did not cancel the evaluation context of a supplied summary", name)
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

func TestRequestAssessment_GivenNilSummaryRequestCancelledThenRequestedAgain_ShouldDispatchOnlyTheNewResult(t *testing.T) {
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
		cancelled func(*App) tea.Cmd
	}{
		{"cancelled during host plan", func(app *App) tea.Cmd {
			cmd := app.requestAssessment(nil)
			deactivateActivePlugin(app)
			return cmd
		}},
		{"cancelled during evaluation", func(app *App) tea.Cmd {
			return app.requestAssessmentWith(nil, func(ctx context.Context, _ riskpolicy.Snapshot, _ sdk.PolicyEvidence) sdk.PolicyAssessment {
				deactivateActivePlugin(app)
				<-ctx.Done()
				return sdk.PolicyAssessment{Status: sdk.AssessmentCancelled}
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, deps, recorder := newAssessmentRequestApp(t, t.TempDir(), false, svc)
			stale := tc.cancelled(&app)()
			current := deps.RequestAssessment(nil)

			drainPolicyAppCommand(&app, func() tea.Msg { return stale })
			if len(recorder.events) != 0 {
				t.Fatalf("cancelled request's result dispatched to the request issued after cancel: %+v", recorder.events)
			}
			deliverAssessment(&app, current)

			ev := onlyRiskEvent(t, recorder)
			if ev.Err != nil || ev.Summary != planned || !ev.Assessment.Usable() {
				t.Fatalf("dispatched event = %+v, want the new request's usable assessment of %p", ev, planned)
			}
		})
	}
}

// Blast Radius requests every assessment for the reviewed summary, so a request
// cancelled before an identical one is issued must not settle the new request
// with a result computed under the policy it replaced.
func TestRequestAssessment_GivenSuppliedSummaryRequestCancelledThenRequestedAgain_ShouldDispatchOnlyTheNewResult(t *testing.T) {
	root := t.TempDir()
	writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "policy.rego" }`)
	classifyAll := func(level string) string {
		return "package tfui.risk.v1\nimport rego.v1\ndefault classification := {\"level\":\"" + level + "\",\"reasons\":[{\"code\":\"edited\",\"message\":\"Edited policy.\",\"paths\":[]}]}\n"
	}
	writePolicyFile(t, root, "policy.rego", classifyAll("low"))
	app, deps, recorder := newAssessmentRequestApp(t, root, true, newMockService("default", nil))
	summary := reviewedRiskSummary()

	stale := deps.RequestAssessment(summary)()
	deactivateActivePlugin(&app)
	writePolicyFile(t, root, "policy.rego", classifyAll("high"))
	current := deps.RequestAssessment(summary)

	drainPolicyAppCommand(&app, func() tea.Msg { return stale })
	if len(recorder.events) != 0 {
		t.Fatalf("cancelled request's result dispatched to the request issued after cancel: %+v", recorder.events)
	}
	deliverAssessment(&app, current)

	ev := onlyRiskEvent(t, recorder)
	if ev.Assessment.OverallLevel == nil || *ev.Assessment.OverallLevel != sdk.RiskHigh {
		t.Fatalf("dispatched assessment = %+v, want the new request's high classification", ev.Assessment)
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
	cmd := deps.RequestAssessment(nil)
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
