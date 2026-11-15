package risk

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newLifecyclePlugin(t *testing.T, svc sdk.Service) (*Plugin, *sdktest.PluginDepsHarness) {
	t.Helper()
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	runLifecycleCmd(p, p.Init(h.Deps))
	return p, h
}

func runLifecycleCmd(p *Plugin, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, child := range batch {
			runLifecycleCmd(p, child)
		}
		return
	}
	_, next := p.Update(msg)
	runLifecycleCmd(p, next)
}

func lifecycleSummary(address string) *sdk.PlanSummary {
	return &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: address}, Action: sdk.ActionCreate, Risk: sdk.RiskLow}}}
}

func TestActivate_GivenNoRetainedPlan_ShouldUseActiveOptionsAndRemoveTemporaryArtifact(t *testing.T) {
	for _, planErr := range []error{nil, errors.New("terraform planning failed")} {
		t.Run(map[bool]string{true: "failure", false: "success"}[planErr != nil], func(t *testing.T) {
			var artifact string
			scoped := &sdktest.MockService{PlanFn: func(_ context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
				artifact = opts.PlanFile
				if artifact == "" {
					t.Fatal("Risk must supply a temporary artifact instead of Terraform's tfplan.out default")
				}
				if err := os.WriteFile(artifact, []byte("saved plan"), 0600); err != nil {
					t.Fatal(err)
				}
				return lifecycleSummary("aws_instance.active"), planErr
			}}
			fallback := &sdktest.MockService{}
			p, h := newLifecyclePlugin(t, fallback)
			h.Ctx = &sdk.Context{Service: scoped, WorkingDir: t.TempDir(), Workspace: "production", Pins: sdk.Pins{"aws_instance.active"}, Vars: map[string]string{"size": "large"}, VarFiles: []string{"production.tfvars"}, ExtraArgs: []string{"-refresh=false"}, Parallelism: 3}
			runLifecycleCmd(p, p.Activate())
			if len(scoped.PlanCalls) != 1 {
				t.Fatalf("active Context plan calls = %d, want 1", len(scoped.PlanCalls))
			}
			got := scoped.PlanCalls[0]
			want := h.Ctx.PlanOptions()
			got.PlanFile = ""
			got.Writer = nil
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("plan options = %#v, want active Context %#v", got, want)
			}
			if len(fallback.PlanCalls) != 0 {
				t.Fatal("Risk used unscoped fallback service")
			}
			if _, err := os.Stat(artifact); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("temporary artifact remains: %v", err)
			}
			if !p.Ready() {
				t.Fatal("completed acquisition must terminate headless waiting, including errors")
			}
			if planErr != nil {
				if p.ExitCode() == 0 || !strings.Contains(string(p.Stderr()), planErr.Error()) || !strings.Contains(p.View(100, 30), planErr.Error()) {
					t.Fatalf("planning failure must be visible: exit %d, stderr %q", p.ExitCode(), p.Stderr())
				}
			} else if !strings.Contains(p.View(100, 30), "aws_instance.active") {
				t.Fatal("acquired plan not displayed")
			}
		})
	}
}

func TestPlanCompleted_GivenReviewedPlan_ShouldReuseEvidenceOnRiskActivation(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newLifecyclePlugin(t, svc)
	handler, ok := any(p).(sdk.PlanCompletedHandler)
	if !ok {
		t.Fatal("Risk must consume Plan completion rather than require a second plan")
	}
	runLifecycleCmd(p, handler.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: lifecycleSummary("aws_instance.reviewed"), PlanFile: "owned-by-plan"}))
	runLifecycleCmd(p, p.Activate())
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Risk acquired %d extra plans despite retained evidence", len(svc.PlanCalls))
	}
	if !p.Ready() || !strings.Contains(p.View(100, 30), "aws_instance.reviewed") {
		t.Fatal("retained reviewed plan was not assessed")
	}
}

func TestActivate_GivenPolicySourceFailure_ShouldTerminateHeadlessWaiting(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newLifecyclePlugin(t, svc)
	h.Deps.RiskPolicyDiagnostic = func() string { return "source_unreadable" }
	runLifecycleCmd(p, p.Init(h.Deps))
	runLifecycleCmd(p, p.Activate())
	if !p.Ready() {
		t.Fatal("policy source failure must be terminal for headless sessions")
	}
	if p.ExitCode() == 0 || !strings.Contains(string(p.Stderr()), "source_unreadable") {
		t.Fatal("policy source failure must remain unavailable and visible")
	}
	if len(svc.PlanCalls) != 0 {
		t.Fatal("unreadable policy should not acquire another plan")
	}
}

func TestAssessment_GivenInvalidationOrContextChange_ShouldRejectLateAcquisition(t *testing.T) {
	for _, event := range []string{"invalidation", "context", "cancel"} {
		t.Run(event, func(t *testing.T) {
			var operationCtx context.Context
			svc := &sdktest.MockService{PlanFn: func(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
				operationCtx = ctx
				return lifecycleSummary("aws_instance.stale"), nil
			}}
			p, h := newLifecyclePlugin(t, svc)
			cmd := p.Activate()
			if cmd == nil {
				t.Fatal("Risk activation must start acquiring its missing plan")
			}
			busy, ok := any(p).(sdk.Busy)
			if !ok || !busy.Busy() {
				t.Fatal("Risk must report its running operation")
			}
			late := cmd()
			switch event {
			case "invalidation":
				handler, ok := any(p).(sdk.PlanInvalidatedHandler)
				if !ok {
					t.Fatal("Risk must reset on plan invalidation")
				}
				runLifecycleCmd(p, handler.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{}))
			case "context":
				handler, ok := any(p).(sdk.ContextChangedHandler)
				if !ok {
					t.Fatal("Risk must reset on Context changes")
				}
				prev := h.Ctx
				h.Ctx = &sdk.Context{Service: &sdktest.MockService{}, WorkingDir: t.TempDir(), Workspace: "next"}
				runLifecycleCmd(p, handler.HandleContextChanged(sdk.ContextChangedEvent{Prev: prev, Next: h.Ctx}))
			case "cancel":
				cancel, ok := any(p).(sdk.Cancellable)
				if !ok {
					t.Fatal("Risk must support cancellation")
				}
				cancel.Cancel()
			}
			if operationCtx == nil || operationCtx.Err() != context.Canceled {
				t.Fatal("reset must cancel the old operation Context")
			}
			runLifecycleCmd(p, func() tea.Msg { return late })
			if strings.Contains(p.View(100, 30), "aws_instance.stale") {
				t.Fatal("late previous-generation result became visible")
			}
			if busy.Busy() {
				t.Fatal("reset left the operation busy")
			}
			if event == "cancel" && !p.Ready() {
				t.Fatal("cancelled Risk must terminate headless waiting")
			}
		})
	}
}

func TestPlanInvalidated_GivenCompletedAssessment_ShouldClearRowsAndAcquireFreshPlan(t *testing.T) {
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return lifecycleSummary("aws_instance.fresh"), nil
	}}
	p, _ := newLifecyclePlugin(t, svc)
	p.Analyze(lifecycleSummary("aws_instance.old"))
	handler, ok := any(p).(sdk.PlanInvalidatedHandler)
	if !ok {
		t.Fatal("Risk must discard derived assessment when the reviewed plan is invalidated")
	}
	runLifecycleCmd(p, handler.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{}))
	if strings.Contains(p.View(100, 30), "aws_instance.old") {
		t.Fatal("invalidated assessment rows remain visible")
	}
	runLifecycleCmd(p, p.Activate())
	if len(svc.PlanCalls) != 1 || !p.Ready() || !strings.Contains(p.View(100, 30), "aws_instance.fresh") {
		t.Fatalf("activation did not assess fresh evidence: plans %d, view %q", len(svc.PlanCalls), p.View(100, 30))
	}
}

func TestCancel_GivenRunningAcquisition_ShouldCancelServiceAndRejectItsLateResult(t *testing.T) {
	started := make(chan context.Context, 1)
	release := make(chan struct{})
	svc := &sdktest.MockService{PlanFn: func(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
		started <- ctx
		<-release
		return lifecycleSummary("aws_instance.cancelled"), nil
	}}
	p, _ := newLifecyclePlugin(t, svc)
	cmd := p.Activate()
	if cmd == nil {
		t.Fatal("Risk activation must acquire missing evidence")
	}
	cancellation, ok := any(p).(sdk.Cancellable)
	if !ok {
		t.Fatal("Risk must cancel running Terraform acquisition")
	}
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	ctx := <-started
	cancellation.Cancel()
	if ctx.Err() != context.Canceled {
		close(release)
		<-result
		t.Fatal("Cancel did not cancel the Terraform service Context")
	}
	close(release)
	late := <-result
	runLifecycleCmd(p, func() tea.Msg { return late })
	if !p.Ready() || strings.Contains(p.View(100, 30), "aws_instance.cancelled") {
		t.Fatalf("cancelled late acquisition was published or headless wait did not terminate: %q", p.View(100, 30))
	}
}

func TestAssessment_GivenPolicyTerminalStatuses_ShouldFinishWithoutSuccessfulFallback(t *testing.T) {
	for _, status := range []string{"complete", "empty", "partial", "unavailable", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			svc := &sdktest.MockService{}
			p, h := newLifecyclePlugin(t, svc)
			high := sdk.RiskHigh
			h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
				got := sdk.PolicyAssessment{Status: status, PlanID: "retained"}
				if status == "complete" {
					got.OverallLevel = &high
					got.Resources = []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high, Reasons: []sdk.PolicyReason{{Code: "authored", Message: "Project classification."}}}}
				}
				if status == "partial" || status == "unavailable" {
					got.ErrorCode = "evaluation_failed"
				}
				return got
			}
			runLifecycleCmd(p, p.Init(h.Deps))
			p.Analyze(lifecycleSummary("aws_instance.old"))
			handler, ok := any(p).(sdk.PlanCompletedHandler)
			if !ok {
				t.Fatal("Risk must receive plan completion")
			}
			runLifecycleCmd(p, handler.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: lifecycleSummary("aws_instance.reviewed")}))
			cmd := p.Activate()
			if strings.Contains(p.View(100, 30), "aws_instance.old") {
				t.Fatal("new assessment retained old classification while evaluating")
			}
			runLifecycleCmd(p, cmd)
			if !p.Ready() || p.Busy() {
				t.Fatalf("policy status %s left headless waiting", status)
			}
			if p.Assessment() == nil || p.Assessment().Status != status {
				t.Fatalf("policy terminal status %s was not retained", status)
			}
			switch status {
			case "complete":
				if p.ExitCode() != 0 || p.Overall() != high || !strings.Contains(p.View(100, 30), "aws_instance.reviewed") {
					t.Fatalf("selected classification missing: %q", p.View(100, 30))
				}
			case "empty":
				if p.ExitCode() != 0 || strings.Contains(p.View(100, 30), "aws_instance.reviewed") {
					t.Fatalf("empty eligibility rendered a classified resource: %q", p.View(100, 30))
				}
			default:
				if p.ExitCode() == 0 || len(p.Stderr()) == 0 || strings.Contains(p.View(100, 30), "Low risk") {
					t.Fatalf("policy %s became successful fallback: exit %d, view %q", status, p.ExitCode(), p.View(100, 30))
				}
			}
		})
	}
}

func TestAssessment_GivenLatePolicyResultAfterContextSwitch_ShouldKeepNewContextAssessment(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newLifecyclePlugin(t, svc)
	started := make(chan context.Context, 1)
	release := make(chan struct{})
	high, low := sdk.RiskHigh, sdk.RiskLow
	h.Deps.AssessRisk = func(ctx context.Context, current *sdk.Context, _ *sdk.PlanSummary) sdk.PolicyAssessment {
		level, address := &low, "aws_instance.new"
		if current.Workspace == "default" {
			started <- ctx
			<-release
			level, address = &high, "aws_instance.old"
		}
		return sdk.PolicyAssessment{Status: "complete", OverallLevel: level, Resources: []sdk.PolicyResourceAssessment{{Address: address, Status: "classified", Level: level}}}
	}
	runLifecycleCmd(p, p.Init(h.Deps))
	runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: lifecycleSummary("aws_instance.old")}))
	oldCmd := p.Activate()
	late := make(chan tea.Msg, 1)
	go func() { late <- oldCmd() }()
	oldCtx := <-started
	previous := h.Ctx
	h.Ctx = &sdk.Context{Service: svc, Workspace: "new", WorkingDir: t.TempDir()}
	runLifecycleCmd(p, p.HandleContextChanged(sdk.ContextChangedEvent{Prev: previous, Next: h.Ctx}))
	if oldCtx.Err() != context.Canceled {
		close(release)
		<-late
		t.Fatal("Context switch did not cancel previous policy evaluation")
	}
	if p.Assessment() != nil {
		close(release)
		<-late
		t.Fatal("Context switch retained previous assessment")
	}
	runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: lifecycleSummary("aws_instance.new")}))
	runLifecycleCmd(p, p.Activate())
	close(release)
	message := <-late
	runLifecycleCmd(p, func() tea.Msg { return message })
	if !p.Ready() || p.Overall() != low || !strings.Contains(p.View(100, 30), "aws_instance.new") || strings.Contains(p.View(100, 30), "aws_instance.old") {
		t.Fatalf("late policy result replaced current assessment: %q", p.View(100, 30))
	}
}

func TestAssessment_GivenCustomClassification_ShouldPreserveReviewedPlanSummary(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newLifecyclePlugin(t, svc)
	high := sdk.RiskHigh
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
		return sdk.PolicyAssessment{Status: "complete", OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high}}}
	}
	runLifecycleCmd(p, p.Init(h.Deps))
	summary := lifecycleSummary("aws_instance.reviewed")
	runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary, PlanFile: "plan-owned.tfplan"}))
	runLifecycleCmd(p, p.Activate())
	if p.Overall() != high {
		t.Fatalf("Risk did not use selected-policy classification: %s", p.Overall())
	}
	if len(summary.Changes) != 1 || summary.Changes[0].Risk != sdk.RiskLow {
		t.Fatalf("Risk changed Plan-owned summary: %+v", summary.Changes)
	}
	runLifecycleCmd(p, p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{}))
	if p.Assessment() != nil || len(p.groups) != 0 {
		t.Fatal("invalidation retained derived selected-policy assessment")
	}
}

func TestLifecycle_GivenShutdownOrNilContext_ShouldEndOrPreserveCurrentOperation(t *testing.T) {
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	cmd := p.Activate()
	p.HandleContextChanged(sdk.ContextChangedEvent{})
	if !p.Busy() {
		t.Fatal("nil Context event discarded current operation")
	}
	p.Shutdown()
	runLifecycleCmd(p, cmd)
	if !p.Ready() || p.Busy() || p.ExitCode() != 1 {
		t.Fatal("shutdown did not leave terminal cancellation")
	}
}

func TestPlanCompleted_GivenSourceFailure_ShouldClearPriorAssessmentWithoutAcquisition(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newLifecyclePlugin(t, svc)
	p.Analyze(lifecycleSummary("aws_instance.old"))
	calls := 0
	h.Deps.RiskPolicyDiagnostic = func() string { calls++; return "source_unreadable" }
	runLifecycleCmd(p, p.Init(h.Deps))
	summary := lifecycleSummary("aws_instance.reviewed")
	if cmd := p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}); cmd != nil {
		runLifecycleCmd(p, cmd)
		t.Fatal("Plan completion handler must only retain reviewed state")
	}
	if calls != 0 {
		t.Fatalf("Plan completion performed %d source diagnostic reads; filesystem work belongs to activation", calls)
	}
	if p.Status() != sdk.StatusIdle || p.Assessment() != nil || strings.Contains(p.View(100, 30), "aws_instance.old") {
		t.Fatalf("Plan completion retained previous assessment: status %v, view %q", p.Status(), p.View(100, 30))
	}
	runLifecycleCmd(p, p.Activate())
	if calls != 1 || !p.Ready() || p.ExitCode() != 1 || !strings.Contains(string(p.Stderr()), "source_unreadable") {
		t.Fatalf("activation did not report terminal source failure: calls %d, stderr %q", calls, p.Stderr())
	}
	if len(svc.PlanCalls) != 0 {
		t.Fatal("source failure acquired an unnecessary Terraform plan")
	}
	h.Deps.RiskPolicyDiagnostic = func() string { return "" }
	runLifecycleCmd(p, p.Init(h.Deps))
	runLifecycleCmd(p, p.Activate())
	if len(svc.PlanCalls) != 0 || !strings.Contains(p.View(100, 30), "aws_instance.reviewed") {
		t.Fatal("source recovery did not assess the retained reviewed summary")
	}
}

func TestActivate_GivenTemporaryDirectoryUnavailable_ShouldFailTerminally(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir()+"/missing")
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	runLifecycleCmd(p, p.Activate())
	if !p.Ready() || p.ExitCode() != 1 || len(p.Stderr()) == 0 {
		t.Fatal("temporary acquisition error did not terminate")
	}
}

func TestActivate_GivenAcquisitionAlreadyRunning_ShouldPreserveSingleOperation(t *testing.T) {
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return lifecycleSummary("aws_instance.reviewed"), nil
	}}
	p, _ := newLifecyclePlugin(t, svc)
	cmd := p.Activate()
	if cmd == nil {
		t.Fatal("initial activation did not acquire missing evidence")
	}
	if view := p.View(100, 30); !strings.Contains(view, "Loading") {
		t.Fatalf("running Risk acquisition must show the standard Loading marker, got %q", view)
	}
	if duplicate := p.Activate(); duplicate != nil {
		t.Fatal("repeated activation must not start a second in-flight operation")
	}
	runLifecycleCmd(p, cmd)
	if !p.Ready() || len(svc.PlanCalls) != 1 || !strings.Contains(p.View(100, 30), "aws_instance.reviewed") {
		t.Fatalf("repeated activation invalidated first operation: plans %d, view %q", len(svc.PlanCalls), p.View(100, 30))
	}
}
