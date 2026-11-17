package ui

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

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
