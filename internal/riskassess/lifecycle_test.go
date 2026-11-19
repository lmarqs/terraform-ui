package riskassess

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmarqs/terraform-ui/internal/riskpolicy"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

// blockingPlanService runs a Terraform plan that lasts until its context ends,
// as exec.CommandContext does, and reports that context once the plan starts.
func blockingPlanService() (*sdktest.MockService, <-chan context.Context) {
	started := make(chan context.Context, 1)
	svc := &sdktest.MockService{}
	svc.PlanFn = func(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
		started <- ctx
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return svc, started
}

func TestRequest_ShouldIssueDistinctNonzeroIDsAcrossGenerationsAndCarryThemOnTheEvent(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	seen := map[sdk.AssessmentID]bool{}
	for _, step := range []func(){func() {}, func() { h.coord.Cancel(requester) }, h.coord.Advance, func() {}} {
		step()
		id, cmd := h.coord.Request(requester, h.current, reviewedRiskSummary())
		if id == 0 || seen[id] {
			t.Fatalf("issued ID %d after %v; want a nonzero ID never issued before", id, seen)
		}
		seen[id] = true
		if got := cmd().(Result).Event.Request; got != id {
			t.Fatalf("event Request = %d, want the issued ID %d", got, id)
		}
	}
}

func TestAdvance_GivenRequestIssuedBefore_ShouldDropItsResult(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	h.deliver(h.request(reviewedRiskSummary())) // positive control: a current result is dispatched
	onlyRiskEvent(t, h)
	h.events = nil

	for name, run := range map[string]func(tea.Cmd) Result{
		"while request runs": func(cmd tea.Cmd) Result { h.coord.Advance(); return cmd().(Result) },
		"before delivery":    func(cmd tea.Cmd) Result { result := cmd().(Result); h.coord.Advance(); return result },
	} {
		t.Run(name, func(t *testing.T) {
			h.dispatch(run(h.request(reviewedRiskSummary())))
			if len(h.events) != 0 {
				t.Fatalf("stale result dispatched after Advance: %+v", h.events)
			}
		})
	}
}

func TestAdvance_GivenSuppliedSummaryEvaluation_ShouldCancelEvaluationContext(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	evaluating := make(chan context.Context, 1)
	cmd := h.requestWith(reviewedRiskSummary(), func(ctx context.Context, _ riskpolicy.Policy, _ sdk.PolicyEvidence) sdk.PolicyAssessment {
		evaluating <- ctx
		<-ctx.Done()
		return sdk.PolicyAssessment{Status: sdk.AssessmentCancelled}
	})
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	ctx := <-evaluating

	h.coord.Advance()

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Advance did not cancel the evaluation context of a supplied summary")
	}
	result := (<-done).(Result)
	if result.Event.Assessment.Status != sdk.AssessmentCancelled {
		t.Fatalf("cancelled evaluation published %+v, want its cancelled assessment", result.Event.Assessment)
	}
	h.dispatch(result)
	if len(h.events) != 0 {
		t.Fatalf("result superseded by Advance was dispatched: %+v", h.events)
	}
}

func TestAdvance_GivenHostPlan_ShouldReportCancellationWithoutEvaluation(t *testing.T) {
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return reviewedRiskSummary(), nil }
	h := newHarness(t, t.TempDir(), svc)
	cmd := h.request(nil)
	h.coord.Advance()

	ev := cmd().(Result).Event
	if !errors.Is(ev.Err, context.Canceled) || ev.Summary != nil || ev.Assessment.Status != "" {
		t.Fatalf("cancelled host plan = %+v, want cancellation without summary or assessment", ev)
	}
}

func TestCancel_GivenHostPlanForNilSummary_ShouldCancelTerraformContext(t *testing.T) {
	svc, started := blockingPlanService()
	h := newHarness(t, t.TempDir(), svc)
	done := make(chan tea.Msg, 1)
	cmd := h.request(nil)
	go func() { done <- cmd() }()
	planCtx := <-started

	h.coord.Cancel(requester)

	select {
	case <-planCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Cancel did not cancel the Terraform service Context of the host plan")
	}
	<-done
}

func TestCancel_GivenRequestNotYetRun_ShouldNotRunPlan(t *testing.T) {
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return reviewedRiskSummary(), nil }
	h := newHarness(t, t.TempDir(), svc)
	cmd := h.request(nil)

	h.coord.Cancel(requester)
	ev := cmd().(Result).Event

	if len(svc.PlanCalls) != 0 {
		t.Fatalf("Service.Plan called %d times for a request cancelled before it ran, want 0", len(svc.PlanCalls))
	}
	if !errors.Is(ev.Err, context.Canceled) || ev.Summary != nil || ev.Assessment.Status != "" {
		t.Fatalf("cancelled request = %+v, want cancellation without summary or assessment", ev)
	}
}

func TestCancel_GivenTwoRequesters_ShouldCancelOnlyTheNamedRequestersEvaluation(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	evaluating := make(chan context.Context, 1)
	concurrentErr := make(chan error, 1)
	var calls atomic.Int32
	h.coord.assess = func(ctx context.Context, policy riskpolicy.Policy, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
		if calls.Add(1) == 1 {
			evaluating <- ctx
			<-ctx.Done()
			return sdk.PolicyAssessment{Status: sdk.AssessmentCancelled}
		}
		concurrentErr <- ctx.Err()
		return policy.Assess(ctx, evidence)
	}
	cmdA := h.requestAs("a", reviewedRiskSummary())
	cmdB := h.requestAs("b", reviewedRiskSummary())
	doneA := make(chan tea.Msg, 1)
	go func() { doneA <- cmdA() }()
	ctxA := <-evaluating

	h.coord.Cancel("a")

	select {
	case <-ctxA.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Cancel(a) did not cancel a's evaluation context")
	}
	h.dispatch(cmdB().(Result))
	if err := <-concurrentErr; err != nil {
		t.Fatalf("Cancel(a) cancelled b's evaluation context: %v", err)
	}
	if event := onlyRiskEvent(t, h); !event.Assessment.Usable() {
		t.Fatalf("b's result = %+v, want its usable assessment", event.Assessment)
	}
	if result := (<-doneA).(Result); result.Event.Assessment.Status != sdk.AssessmentCancelled {
		t.Fatalf("a's cancelled evaluation published %+v, want its cancelled assessment", result.Event.Assessment)
	}
}

// The host dispatches a cancelled request's result; its requester ignores it
// because the event carries the cancelled request's ID.
func TestCancel_GivenRequestedAgain_ShouldDispatchEachResultUnderItsOwnID(t *testing.T) {
	planned := reviewedRiskSummary()
	svc := &sdktest.MockService{}
	svc.PlanFn = func(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return planned, nil
	}
	for _, summary := range []*sdk.PlanSummary{nil, planned} {
		h := newHarness(t, t.TempDir(), svc)
		staleID, stale := h.coord.Request(requester, h.current, summary)
		h.coord.Cancel(requester)
		currentID, current := h.coord.Request(requester, h.current, summary)

		h.deliver(stale)
		h.deliver(current)

		if len(h.events) != 2 || h.events[0].Request != staleID || h.events[1].Request != currentID || staleID == currentID {
			t.Fatalf("dispatched %+v, want the cancelled result under ID %d then the new one under ID %d", h.events, staleID, currentID)
		}
		if ev := h.events[0]; ev.Err == nil && ev.Assessment.Status != sdk.AssessmentCancelled {
			t.Fatalf("cancelled request's result = %+v, want a cancellation", ev)
		}
		if ev := h.events[1]; ev.Err != nil || ev.Summary != planned || !ev.Assessment.Usable() {
			t.Fatalf("new request's result = %+v, want its usable assessment of %p", ev, planned)
		}
	}
}

// Cancelling one requester and requesting for it again must not drop another
// requester's pending result.
func TestCancel_GivenOtherRequesterPendingAndRequestedAgain_ShouldKeepTheOtherResultCurrent(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	_, pending := h.coord.Request("b", h.current, reviewedRiskSummary())
	h.coord.Cancel("a")
	h.coord.Request("a", h.current, reviewedRiskSummary())

	result := pending().(Result)

	if !h.coord.Current(result) {
		t.Fatal("b's pending result is dropped after a was cancelled and requested again")
	}
	if !result.Event.Assessment.Usable() {
		t.Fatalf("b's result = %+v, want its usable assessment", result.Event.Assessment)
	}
}

// A request cancelled before its Cmd runs still reaches the evaluator, which
// sees a done context and reports the cancellation.
func TestAdvance_GivenSuppliedSummaryRequestNotYetRun_ShouldEvaluateUnderCancelledContext(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	var evaluated error
	cmd := h.requestWith(reviewedRiskSummary(), func(ctx context.Context, policy riskpolicy.Policy, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
		evaluated = ctx.Err()
		return policy.Assess(ctx, evidence)
	})
	h.coord.Advance()

	result := cmd().(Result)

	if !errors.Is(evaluated, context.Canceled) || result.Event.Assessment.Status != "cancelled" || result.Event.Assessment.OverallLevel != nil {
		t.Fatalf("invalidated request ran with ctx err %v and assessed %+v, want cancelled", evaluated, result.Event.Assessment)
	}
}

func TestCancel_GivenSuppliedSummary_ShouldCancelItsEvaluation(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	evaluating := make(chan context.Context, 1)
	h.coord.assess = func(ctx context.Context, _ riskpolicy.Policy, _ sdk.PolicyEvidence) sdk.PolicyAssessment {
		evaluating <- ctx
		<-ctx.Done()
		return sdk.PolicyAssessment{Status: sdk.AssessmentCancelled}
	}
	id, cmd := h.coord.Request(requester, h.current, reviewedRiskSummary())
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	ctx := <-evaluating

	h.coord.Cancel(requester)

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Cancel did not cancel the evaluation context of a supplied summary")
	}
	h.dispatch((<-done).(Result))
	if event := onlyRiskEvent(t, h); event.Request != id || event.Assessment.Status != sdk.AssessmentCancelled {
		t.Fatalf("dispatched %+v, want the cancelled assessment under ID %d", event, id)
	}
}

// A request cancelled while evaluating a summary the host planned still
// delivers its result under its own ID, never the new request's.
func TestCancel_GivenRequestCancelledDuringEvaluationThenRequestedAgain_ShouldDispatchEachResultUnderItsOwnID(t *testing.T) {
	planned := reviewedRiskSummary()
	svc := &sdktest.MockService{}
	svc.PlanFn = func(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return planned, nil
	}
	h := newHarness(t, t.TempDir(), svc)
	evaluating := make(chan struct{})
	var calls atomic.Int32
	h.coord.assess = func(ctx context.Context, policy riskpolicy.Policy, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
		if calls.Add(1) == 1 {
			close(evaluating)
			<-ctx.Done()
			return sdk.PolicyAssessment{Status: sdk.AssessmentCancelled}
		}
		return policy.Assess(ctx, evidence)
	}
	staleID, staleCmd := h.coord.Request(requester, h.current, nil)
	stale := make(chan tea.Msg, 1)
	go func() { stale <- staleCmd() }()
	<-evaluating
	h.coord.Cancel(requester)
	currentID, current := h.coord.Request(requester, h.current, nil)

	h.dispatch((<-stale).(Result))
	h.deliver(current)

	if len(h.events) != 2 || h.events[0].Request != staleID || h.events[1].Request != currentID || staleID == currentID {
		t.Fatalf("dispatched %+v, want the cancelled result under ID %d then the new one under ID %d", h.events, staleID, currentID)
	}
	if ev := h.events[0]; ev.Err == nil && ev.Assessment.Status != sdk.AssessmentCancelled {
		t.Fatalf("cancelled request's result = %+v, want a cancellation", ev)
	}
	if ev := h.events[1]; ev.Err != nil || ev.Summary != planned || !ev.Assessment.Usable() {
		t.Fatalf("new request's result = %+v, want its usable assessment of %p", ev, planned)
	}
}
