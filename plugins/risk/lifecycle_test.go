package risk

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

// Risk recognises the answer to its own nil-summary request by its pending
// state, not by the Summary pointer: the host plans that summary, so Risk has
// never seen it. While a nil request is pending, Risk accepts the next
// RiskAssessedEvent whatever its Summary (a host-planned summary, or nil when
// the policy failed before planning). The host drops results requested before
// a Context change or plan invalidation (FR-9), and Risk leaves the pending
// state on those, on Cancel and on PlanCompletedEvent, so the next event while
// pending answers this request. A request for a retained summary accepts only
// the event whose Summary is that pointer.

func newLifecyclePlugin(t *testing.T, svc sdk.Service) (*Plugin, *sdktest.PluginDepsHarness) {
	t.Helper()
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	runLifecycleCmd(p, p.Init(h.Deps))
	return p, h
}

// newAssessedPlugin returns a Risk plugin that retains summary from Plan and
// whose host answers every assessment request with assessment for summary.
func newAssessedPlugin(t *testing.T, summary *sdk.PlanSummary, assessment sdk.PolicyAssessment) (*Plugin, *sdktest.MockService, *sdktest.PluginDepsHarness) {
	t.Helper()
	svc := &sdktest.MockService{}
	p, h := newLifecyclePlugin(t, svc)
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: summary, Assessment: assessment}
	runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}))
	return p, svc, h
}

// runLifecycleCmd runs cmd depth-first like the macro driver. Plugin messages
// go to Update; events go through an EventBus, as the App dispatches them.
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
	if _, ok := msg.(sdk.Event); ok {
		runLifecycleCmd(p, sdk.NewEventBus([]sdk.Plugin{p}).Dispatch(msg))
		return
	}
	_, next := p.Update(msg)
	runLifecycleCmd(p, next)
}

// deliver hands ev to the plugin as the App does when the host publishes it.
func deliver(p *Plugin, ev sdk.RiskAssessedEvent) {
	runLifecycleCmd(p, func() tea.Msg { return ev })
}

func lifecycleSummary(address string) *sdk.PlanSummary {
	return &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: address}, Action: sdk.ActionCreate, Risk: sdk.RiskLow}}}
}

// classifiedAs is a complete assessment that classifies address at level with
// reason, so a view showing reason came from the event, not the raw plan risk.
func classifiedAs(address string, level sdk.RiskLevel, reason string) sdk.PolicyAssessment {
	return sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &level, Resources: []sdk.PolicyResourceAssessment{
		{Address: address, Actions: []string{"create"}, Status: sdk.ResourceClassified, Level: &level, Reasons: []sdk.PolicyReason{{Code: "authored", Message: reason}}},
	}}
}

func TestActivate_GivenNoRetainedPlan_ShouldRequestHostPlanWithoutRunningTerraform(t *testing.T) {
	for _, tc := range []struct {
		name string
		ev   func(planned *sdk.PlanSummary) sdk.RiskAssessedEvent
	}{
		{"success", func(planned *sdk.PlanSummary) sdk.RiskAssessedEvent {
			return sdk.RiskAssessedEvent{Summary: planned, Assessment: classifiedAs("aws_instance.active", sdk.RiskHigh, "Host planned classification.")}
		}},
		{"failure", func(*sdk.PlanSummary) sdk.RiskAssessedEvent {
			return sdk.RiskAssessedEvent{Err: errors.New("terraform planning failed")}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scoped := &sdktest.MockService{}
			fallback := &sdktest.MockService{}
			p, h := newLifecyclePlugin(t, fallback)
			h.Ctx = &sdk.Context{Service: scoped, WorkingDir: t.TempDir(), Workspace: "production", Pins: sdk.Pins{"aws_instance.active"}}
			h.AssessmentEvent = tc.ev(lifecycleSummary("aws_instance.active"))
			cmd := p.Activate()
			if !p.Busy() || p.Ready() {
				t.Fatalf("before the host answers: busy=%v ready=%v; want busy and not ready for headless waiting", p.Busy(), p.Ready())
			}
			runLifecycleCmd(p, cmd)
			if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != nil {
				t.Fatalf("assessment requests = %v; want one nil-summary request so the host plans", h.AssessmentRequests)
			}
			if len(scoped.PlanCalls)+len(fallback.PlanCalls) != 0 {
				t.Fatalf("Risk ran Service.Plan itself: scoped %d, fallback %d; planning belongs to the host", len(scoped.PlanCalls), len(fallback.PlanCalls))
			}
			if !p.Ready() || p.Busy() {
				t.Fatalf("after the host answered: busy=%v ready=%v; the answer must terminate headless waiting, including errors", p.Busy(), p.Ready())
			}
			view := ansi.Strip(p.View(100, 30))
			if tc.name == "failure" {
				if p.ExitCode() != 1 || string(p.Stderr()) != "terraform planning failed\n" || view != "terraform planning failed" {
					t.Fatalf("planning failure: exit %d, stderr %q, view %q; want the plan error as before", p.ExitCode(), p.Stderr(), view)
				}
				return
			}
			if !strings.Contains(view, "aws_instance.active") || !strings.Contains(view, "Host planned classification.") || p.Overall() != sdk.RiskHigh {
				t.Fatalf("view = %q overall = %s; want the host-planned summary classified by the event", view, p.Overall())
			}
		})
	}
}

func TestPlanCompleted_GivenReviewedPlan_ShouldRequestAssessmentOfRetainedSummary(t *testing.T) {
	summary := lifecycleSummary("aws_instance.reviewed")
	p, svc, h := newAssessedPlugin(t, summary, classifiedAs("aws_instance.reviewed", sdk.RiskMedium, "Reviewed plan classification."))
	if _, ok := any(p).(sdk.PlanCompletedHandler); !ok {
		t.Fatal("Risk must consume Plan completion rather than require a second plan")
	}
	runLifecycleCmd(p, p.Activate())
	if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != summary {
		t.Fatalf("assessment requests = %v; want one request for the retained summary %p", h.AssessmentRequests, summary)
	}
	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Risk acquired %d extra plans despite retained evidence", len(svc.PlanCalls))
	}
	if view := p.View(100, 30); !p.Ready() || !strings.Contains(view, "aws_instance.reviewed") || !strings.Contains(view, "Reviewed plan classification.") {
		t.Fatalf("retained reviewed plan was not rendered from the event: %q", view)
	}
}

func TestActivate_GivenPolicyFailureWithoutPlan_ShouldShowUnavailableCode(t *testing.T) {
	const hint = "risk setting level is unsupported; use threshold"
	for _, tc := range []struct {
		name       string
		code       sdk.ErrorCode
		hint       string
		wantStderr string
		wantView   []string
	}{
		{"source", sdk.ErrorSourceUnreadable, "", "risk unavailable: source_unreadable\n", []string{"risk unavailable: source_unreadable"}},
		{"hint", sdk.ErrorConfigurationInvalid, hint, "risk unavailable: configuration_invalid\n\nhint: " + hint + "\n", []string{"risk unavailable: configuration_invalid", "hint: " + hint}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &sdktest.MockService{}
			p, h := newLifecyclePlugin(t, svc)
			h.AssessmentEvent = sdk.RiskAssessedEvent{Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: tc.code}, Hint: tc.hint}
			runLifecycleCmd(p, p.Activate())
			if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != nil {
				t.Fatalf("assessment requests = %v; want one nil-summary request", h.AssessmentRequests)
			}
			if !p.Ready() || p.Busy() {
				t.Fatal("policy failure must be terminal for headless sessions")
			}
			if p.ExitCode() != 1 || string(p.Stderr()) != tc.wantStderr {
				t.Fatalf("exit %d stderr %q; want exit 1 stderr %q", p.ExitCode(), p.Stderr(), tc.wantStderr)
			}
			view := ansi.Strip(p.View(100, 30))
			for _, want := range tc.wantView {
				if !strings.Contains(view, want) {
					t.Errorf("view = %q; want %q", view, want)
				}
			}
			if len(svc.PlanCalls) != 0 {
				t.Fatal("unavailable policy must not run Terraform")
			}
		})
	}
}

func TestHandleRiskAssessed_GivenStaleEvent_ShouldIgnoreIt(t *testing.T) {
	retained := lifecycleSummary("aws_instance.retained")
	other := lifecycleSummary("aws_instance.other")
	otherEvent := sdk.RiskAssessedEvent{Summary: other, Assessment: classifiedAs("aws_instance.other", sdk.RiskCritical, "Other plan classification.")}

	t.Run("another summary while own request pending", func(t *testing.T) {
		p, _, _ := newAssessedPlugin(t, retained, classifiedAs("aws_instance.retained", sdk.RiskLow, "Retained plan classification."))
		cmd := p.Activate()
		deliver(p, otherEvent)
		if !p.Busy() || p.Ready() || strings.Contains(p.View(100, 30), "aws_instance.other") {
			t.Fatalf("event for another plan settled the request: busy=%v ready=%v view %q", p.Busy(), p.Ready(), p.View(100, 30))
		}
		runLifecycleCmd(p, cmd)
		if view := p.View(100, 30); !p.Ready() || !strings.Contains(view, "Retained plan classification.") {
			t.Fatalf("own event was not rendered after the stale one: %q", view)
		}
	})

	t.Run("another summary after own result", func(t *testing.T) {
		p, _, _ := newAssessedPlugin(t, retained, classifiedAs("aws_instance.retained", sdk.RiskLow, "Retained plan classification."))
		runLifecycleCmd(p, p.Activate())
		deliver(p, otherEvent)
		view := p.View(100, 30)
		if !strings.Contains(view, "Retained plan classification.") || strings.Contains(view, "aws_instance.other") || p.Overall() != sdk.RiskLow {
			t.Fatalf("late event for another plan replaced the result: overall %s, view %q", p.Overall(), view)
		}
	})

	t.Run("no request", func(t *testing.T) {
		p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
		deliver(p, otherEvent)
		if p.Status() != sdk.StatusIdle || strings.Contains(p.View(100, 30), "aws_instance.other") {
			t.Fatalf("unrequested event changed Risk: status %v view %q", p.Status(), p.View(100, 30))
		}
	})
}

func TestAssessment_GivenInvalidationOrContextChange_ShouldRejectLateResult(t *testing.T) {
	for _, event := range []string{"invalidation", "context", "cancel"} {
		t.Run(event, func(t *testing.T) {
			p, h := newLifecyclePlugin(t, &sdktest.MockService{})
			late := sdk.RiskAssessedEvent{Summary: lifecycleSummary("aws_instance.stale"), Assessment: classifiedAs("aws_instance.stale", sdk.RiskHigh, "Stale classification.")}
			if cmd := p.Activate(); cmd == nil {
				t.Fatal("Risk activation must request an assessment")
			}
			busy, ok := any(p).(sdk.Busy)
			if !ok || !busy.Busy() {
				t.Fatal("Risk must report its pending request")
			}
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
			deliver(p, late)
			if strings.Contains(p.View(100, 30), "aws_instance.stale") {
				t.Fatal("late previous-request result became visible")
			}
			if busy.Busy() {
				t.Fatal("reset left the request busy")
			}
			if event == "cancel" && !p.Ready() {
				t.Fatal("cancelled Risk must terminate headless waiting")
			}
		})
	}
}

func TestPlanInvalidated_GivenCompletedAssessment_ShouldClearRowsAndRequestFreshPlan(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newLifecyclePlugin(t, svc)
	p.Analyze(lifecycleSummary("aws_instance.old"))
	handler, ok := any(p).(sdk.PlanInvalidatedHandler)
	if !ok {
		t.Fatal("Risk must discard derived assessment when the reviewed plan is invalidated")
	}
	runLifecycleCmd(p, handler.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{}))
	if strings.Contains(p.View(100, 30), "aws_instance.old") {
		t.Fatal("invalidated assessment rows remain visible")
	}
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: lifecycleSummary("aws_instance.fresh"), Assessment: classifiedAs("aws_instance.fresh", sdk.RiskLow, "Fresh classification.")}
	runLifecycleCmd(p, p.Activate())
	if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != nil || len(svc.PlanCalls) != 0 {
		t.Fatalf("requests %v plans %d; want one nil-summary request and no plan by Risk", h.AssessmentRequests, len(svc.PlanCalls))
	}
	if view := p.View(100, 30); !p.Ready() || !strings.Contains(view, "Fresh classification.") {
		t.Fatalf("activation did not render the fresh host assessment: %q", view)
	}
}

func TestCancel_GivenPendingRequest_ShouldTerminateAndRejectItsLateResult(t *testing.T) {
	p, h := newLifecyclePlugin(t, &sdktest.MockService{})
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: lifecycleSummary("aws_instance.cancelled"), Assessment: classifiedAs("aws_instance.cancelled", sdk.RiskHigh, "Cancelled classification.")}
	cmd := p.Activate()
	if cmd == nil {
		t.Fatal("Risk activation must request missing evidence")
	}
	cancellation, ok := any(p).(sdk.Cancellable)
	if !ok {
		t.Fatal("Risk must support cancellation")
	}
	cancellation.Cancel()
	runLifecycleCmd(p, cmd)
	if view := p.View(100, 30); !p.Ready() || p.Busy() || strings.Contains(view, "aws_instance.cancelled") {
		t.Fatalf("cancelled late result was published or headless wait did not terminate: %q", view)
	}
	if p.ExitCode() != 1 || string(p.Stderr()) != "risk unavailable: cancelled\n" {
		t.Fatalf("cancelled Risk: exit %d stderr %q; want exit 1 and the cancelled diagnostic", p.ExitCode(), p.Stderr())
	}
}

func TestAssessment_GivenPolicyTerminalStatuses_ShouldFinishWithoutSuccessfulFallback(t *testing.T) {
	for _, status := range []string{"complete", "empty", "partial", "unavailable", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			high := sdk.RiskHigh
			got := sdk.PolicyAssessment{Status: sdk.AssessmentStatus(status), PlanID: "retained"}
			if status == "complete" {
				got.OverallLevel = &high
				got.Resources = []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high, Reasons: []sdk.PolicyReason{{Code: "authored", Message: "Project classification."}}}}
			}
			if status == "partial" || status == "unavailable" {
				got.ErrorCode = "evaluation_failed"
			}
			p, h := newLifecyclePlugin(t, &sdktest.MockService{})
			p.Analyze(lifecycleSummary("aws_instance.old"))
			reviewed := lifecycleSummary("aws_instance.reviewed")
			h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: reviewed, Assessment: got}
			runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: reviewed}))
			cmd := p.Activate()
			if strings.Contains(p.View(100, 30), "aws_instance.old") {
				t.Fatal("new assessment retained old classification while evaluating")
			}
			runLifecycleCmd(p, cmd)
			if !p.Ready() || p.Busy() {
				t.Fatalf("policy status %s left headless waiting", status)
			}
			if p.Assessment() == nil || p.Assessment().Status != sdk.AssessmentStatus(status) {
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
	old := lifecycleSummary("aws_instance.old")
	p, svc, h := newAssessedPlugin(t, old, classifiedAs("aws_instance.old", sdk.RiskHigh, "Old classification."))
	late := h.AssessmentEvent
	if cmd := p.Activate(); cmd == nil {
		t.Fatal("Risk activation must request an assessment")
	}
	previous := h.Ctx
	h.Ctx = &sdk.Context{Service: svc, Workspace: "new", WorkingDir: t.TempDir()}
	runLifecycleCmd(p, p.HandleContextChanged(sdk.ContextChangedEvent{Prev: previous, Next: h.Ctx}))
	if p.Assessment() != nil {
		t.Fatal("Context switch retained previous assessment")
	}
	current := lifecycleSummary("aws_instance.new")
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: current, Assessment: classifiedAs("aws_instance.new", sdk.RiskLow, "New classification.")}
	runLifecycleCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: current}))
	runLifecycleCmd(p, p.Activate())
	deliver(p, late)
	if view := p.View(100, 30); !p.Ready() || p.Overall() != sdk.RiskLow || !strings.Contains(view, "New classification.") || strings.Contains(view, "aws_instance.old") {
		t.Fatalf("late policy result replaced current assessment: %q", view)
	}
}

func TestAssessment_GivenCustomClassification_ShouldPreserveReviewedPlanSummary(t *testing.T) {
	high := sdk.RiskHigh
	summary := lifecycleSummary("aws_instance.reviewed")
	p, _, _ := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: "complete", OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high}}})
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
	p, h := newLifecyclePlugin(t, &sdktest.MockService{})
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: lifecycleSummary("aws_instance.late"), Assessment: classifiedAs("aws_instance.late", sdk.RiskLow, "Late classification.")}
	cmd := p.Activate()
	p.HandleContextChanged(sdk.ContextChangedEvent{})
	if !p.Busy() {
		t.Fatal("nil Context event discarded current operation")
	}
	p.Shutdown()
	runLifecycleCmd(p, cmd)
	if !p.Ready() || p.Busy() || p.ExitCode() != 1 || strings.Contains(p.View(100, 30), "aws_instance.late") {
		t.Fatal("shutdown did not leave terminal cancellation")
	}
}

func TestPlanCompleted_GivenSourceFailure_ShouldClearPriorAssessmentWithoutRequest(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newLifecyclePlugin(t, svc)
	p.Analyze(lifecycleSummary("aws_instance.old"))
	summary := lifecycleSummary("aws_instance.reviewed")
	if cmd := p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: summary}); cmd != nil {
		runLifecycleCmd(p, cmd)
		t.Fatal("Plan completion handler must only retain reviewed state")
	}
	if len(h.AssessmentRequests) != 0 {
		t.Fatalf("Plan completion made %d assessment requests; assessment belongs to activation", len(h.AssessmentRequests))
	}
	if p.Status() != sdk.StatusIdle || p.Assessment() != nil || strings.Contains(p.View(100, 30), "aws_instance.old") {
		t.Fatalf("Plan completion retained previous assessment: status %v, view %q", p.Status(), p.View(100, 30))
	}
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: summary, Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorSourceUnreadable}}
	runLifecycleCmd(p, p.Activate())
	if !p.Ready() || p.ExitCode() != 1 || string(p.Stderr()) != "risk unavailable: source_unreadable\n" {
		t.Fatalf("activation did not report terminal source failure: exit %d, stderr %q", p.ExitCode(), p.Stderr())
	}
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: summary, Assessment: classifiedAs("aws_instance.reviewed", sdk.RiskLow, "Recovered classification.")}
	runLifecycleCmd(p, p.Activate())
	if len(svc.PlanCalls) != 0 || !strings.Contains(p.View(100, 30), "Recovered classification.") {
		t.Fatalf("source recovery did not assess the retained reviewed summary: plans %d view %q", len(svc.PlanCalls), p.View(100, 30))
	}
	if len(h.AssessmentRequests) != 2 || h.AssessmentRequests[0] != summary || h.AssessmentRequests[1] != summary {
		t.Fatalf("assessment requests = %v; want two for the retained summary", h.AssessmentRequests)
	}
}

func TestActivate_GivenRequestAlreadyPending_ShouldPreserveSingleRequest(t *testing.T) {
	svc := &sdktest.MockService{}
	p, h := newLifecyclePlugin(t, svc)
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: lifecycleSummary("aws_instance.reviewed"), Assessment: classifiedAs("aws_instance.reviewed", sdk.RiskLow, "Reviewed classification.")}
	cmd := p.Activate()
	if cmd == nil {
		t.Fatal("initial activation did not request an assessment")
	}
	if view := p.View(100, 30); !strings.Contains(view, "Loading") {
		t.Fatalf("pending Risk request must show the standard Loading marker, got %q", view)
	}
	if duplicate := p.Activate(); duplicate != nil {
		t.Fatal("repeated activation must not start a second in-flight request")
	}
	runLifecycleCmd(p, cmd)
	if !p.Ready() || len(h.AssessmentRequests) != 1 || len(svc.PlanCalls) != 0 || !strings.Contains(p.View(100, 30), "Reviewed classification.") {
		t.Fatalf("repeated activation invalidated first request: requests %d plans %d, view %q", len(h.AssessmentRequests), len(svc.PlanCalls), p.View(100, 30))
	}
}
