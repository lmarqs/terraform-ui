package riskassess

import (
	"context"
	"errors"
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
	for _, step := range []func(){func() {}, h.coord.CancelPlans, h.coord.Advance, func() {}} {
		step()
		id, cmd := h.coord.Request(h.current, reviewedRiskSummary())
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

func TestCancelPlans_GivenHostPlanForNilSummary_ShouldCancelTerraformContext(t *testing.T) {
	svc, started := blockingPlanService()
	h := newHarness(t, t.TempDir(), svc)
	done := make(chan tea.Msg, 1)
	cmd := h.request(nil)
	go func() { done <- cmd() }()
	planCtx := <-started

	h.coord.CancelPlans()

	select {
	case <-planCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("CancelPlans did not cancel the Terraform service Context of the host plan")
	}
	<-done
}

func TestCancelPlans_GivenSuppliedSummary_ShouldKeepEvaluatingIt(t *testing.T) {
	h := newHarness(t, t.TempDir(), &sdktest.MockService{})
	evaluating := make(chan context.Context, 1)
	release := make(chan struct{})
	cmd := h.requestWith(reviewedRiskSummary(), func(ctx context.Context, _ riskpolicy.Policy, _ sdk.PolicyEvidence) sdk.PolicyAssessment {
		evaluating <- ctx
		<-release
		return sdk.PolicyAssessment{Status: sdk.AssessmentEmpty}
	})
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	ctx := <-evaluating

	h.coord.CancelPlans()
	close(release)

	if ctx.Err() != nil {
		t.Fatalf("CancelPlans cancelled evaluation of a supplied summary: %v", ctx.Err())
	}
	h.dispatch((<-done).(Result))
	if event := onlyRiskEvent(t, h); event.Assessment.Status != sdk.AssessmentEmpty {
		t.Fatalf("supplied summary result = %+v, want the evaluated assessment", event.Assessment)
	}
}

func TestCancelPlans_GivenRequestedAgain_ShouldDispatchOnlyTheNewResult(t *testing.T) {
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
		stale := h.request(summary)
		h.coord.CancelPlans()
		current := h.request(summary)

		h.deliver(stale)
		if len(h.events) != 0 {
			t.Fatalf("cancelled request's result dispatched to the request issued after cancel: %+v", h.events)
		}
		h.deliver(current)

		ev := onlyRiskEvent(t, h)
		if ev.Err != nil || ev.Summary != planned || !ev.Assessment.Usable() {
			t.Fatalf("dispatched event = %+v, want the new request's usable assessment of %p", ev, planned)
		}
	}
}
