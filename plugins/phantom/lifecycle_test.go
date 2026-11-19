package phantom

import (
	"errors"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

var (
	_ sdk.Activatable          = (*Plugin)(nil)
	_ sdk.PlanCompletedHandler = (*Plugin)(nil)
	_ sdk.RiskAssessedHandler  = (*Plugin)(nil)
	_ sdk.Busy                 = (*Plugin)(nil)
	_ sdk.Cancellable          = (*Plugin)(nil)
)

func phantomSummary() *sdk.PlanSummary {
	return &sdk.PlanSummary{Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "aws_security_group.web"}, Action: sdk.ActionUpdate, IsPhantom: true, AttributeDiffs: []sdk.AttributeDiff{
			{Key: "tags", OldValue: `["a","b"]`, NewValue: `["b","a"]`},
		}},
		{Resource: sdk.Resource{Address: "aws_instance.real"}, Action: sdk.ActionUpdate},
	}}
}

// newHarness returns an initialized plugin whose host answers every assessment
// request with answer.
func newHarness(t *testing.T, answer sdk.RiskAssessedEvent) (*Plugin, *sdktest.PluginDepsHarness) {
	t.Helper()
	svc := &sdktest.MockService{}
	h := sdktest.NewDeps(svc)
	h.AssessmentEvent = answer
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	return p, h
}

// runCmd runs cmd depth-first like macro.Driver: plugin messages go to Update,
// events go through an EventBus as the App dispatches them.
func runCmd(p *Plugin, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			runCmd(p, sub)
		}
		return
	}
	if _, ok := msg.(sdk.Event); ok {
		runCmd(p, sdk.NewEventBus([]sdk.Plugin{p}).Dispatch(msg))
		return
	}
	if _, ok := msg.(ui.TimerTickMsg); ok {
		return // a tick advances the clock and would chain forever while a request is pending
	}
	_, next := p.Update(msg)
	runCmd(p, next)
}

func deliver(p *Plugin, ev sdk.RiskAssessedEvent) {
	runCmd(p, func() tea.Msg { return ev })
}

func pending(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness, tea.Cmd) {
	t.Helper()
	p, h := newHarness(t, sdk.RiskAssessedEvent{Summary: phantomSummary()})
	cmd := p.Activate()
	if p.Status() != sdk.StatusLoading {
		t.Fatalf("setup: status = %v, want loading", p.Status())
	}
	return p, h, cmd
}

func TestHandlePlanCompleted_ShouldFillThePhantomList(t *testing.T) {
	p, _ := newHarness(t, sdk.RiskAssessedEvent{})

	runCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()}))

	view := ansi.Strip(p.View(100, 30))
	if !p.Ready() || p.PhantomCount() != 1 || !strings.Contains(view, "aws_security_group.web") {
		t.Fatalf("Ready=%v PhantomCount=%d view=%q; want the plan's phantom change listed", p.Ready(), p.PhantomCount(), view)
	}
	if strings.Contains(view, "Run a plan first") {
		t.Fatalf("view = %q; still the empty state after a plan completed", view)
	}
}

func TestActivate_GivenNoPlan_ShouldAskTheHostToPlanAndFillTheList(t *testing.T) {
	p, h := newHarness(t, sdk.RiskAssessedEvent{Summary: phantomSummary()})

	cmd := p.Activate()

	if cmd == nil {
		t.Fatal("Activate() with no plan returned no cmd, so nothing obtains one")
	}
	if p.Status() != sdk.StatusLoading || !p.Busy() {
		t.Fatalf("Status=%v Busy=%v; want loading while the host plans, which holds the directory lock", p.Status(), p.Busy())
	}
	runCmd(p, cmd)
	if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != nil {
		t.Fatalf("assessment requests = %v; want exactly one with a nil summary, so the host plans", h.AssessmentRequests)
	}
	if !p.Ready() || p.Busy() || p.PhantomCount() != 1 || !strings.Contains(ansi.Strip(p.View(100, 30)), "aws_security_group.web") {
		t.Fatalf("Ready=%v Busy=%v PhantomCount=%d; want the phantom list filled", p.Ready(), p.Busy(), p.PhantomCount())
	}
}

func TestActivate_GivenReviewedPlan_ShouldKeepItsListWithoutAskingTheHost(t *testing.T) {
	p, h := newHarness(t, sdk.RiskAssessedEvent{})
	runCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()}))

	cmd := p.Activate()

	if cmd != nil || len(h.AssessmentRequests) != 0 || p.Status() != sdk.StatusDone || p.PhantomCount() != 1 {
		t.Fatalf("cmd=%v requests=%d status=%v phantoms=%d; want the reviewed plan's list kept and no request", cmd != nil, len(h.AssessmentRequests), p.Status(), p.PhantomCount())
	}
}

func TestActivate_GivenRequestPending_ShouldNotAskAgain(t *testing.T) {
	p, h, first := pending(t)

	if cmd := p.Activate(); cmd != nil {
		t.Fatal("Activate() while loading returned a cmd, so a second plan would start")
	}
	runCmd(p, first)
	if len(h.AssessmentRequests) != 1 {
		t.Fatalf("assessment requests = %d, want 1", len(h.AssessmentRequests))
	}
}

func TestHandleRiskAssessed_GivenAnotherRequestsEvent_ShouldIgnoreIt(t *testing.T) {
	p, _, _ := pending(t)

	deliver(p, sdk.RiskAssessedEvent{Request: 99, Summary: phantomSummary()})

	if p.Status() != sdk.StatusLoading || p.PhantomCount() != 0 {
		t.Fatalf("Status=%v PhantomCount=%d; another request's result settled the pending one", p.Status(), p.PhantomCount())
	}
}

func TestHandleRiskAssessed_GivenNothingPending_ShouldIgnoreIt(t *testing.T) {
	p, _ := newHarness(t, sdk.RiskAssessedEvent{})

	deliver(p, sdk.RiskAssessedEvent{Request: 1, Summary: phantomSummary()})

	if p.Status() != sdk.StatusIdle {
		t.Fatalf("Status = %v, want idle: an unrequested result must not fill the list", p.Status())
	}
}

func TestHandleRiskAssessed_GivenFailedPlan_ShouldShowTheErrorAndSettle(t *testing.T) {
	p, _ := newHarness(t, sdk.RiskAssessedEvent{Err: errors.New("terraform plan: no configuration files")})

	runCmd(p, p.Activate())

	view := ansi.Strip(p.View(100, 30))
	if p.Status() != sdk.StatusError || !p.Ready() || p.Busy() {
		t.Fatalf("Status=%v Ready=%v Busy=%v; want a settled error so a tape or headless run does not wait forever", p.Status(), p.Ready(), p.Busy())
	}
	if !strings.Contains(view, "plan unavailable: terraform plan: no configuration files") {
		t.Fatalf("view = %q; want terraform's error as is", view)
	}
}

func TestHandleRiskAssessed_GivenNoSummaryAndACode_ShouldShowTheCodeAndHint(t *testing.T) {
	p, _ := newHarness(t, sdk.RiskAssessedEvent{
		Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorConfigurationInvalid},
		Hint:       "fix the policy",
	})

	runCmd(p, p.Activate())

	view := ansi.Strip(p.View(100, 30))
	if p.Status() != sdk.StatusError || !strings.Contains(view, string(sdk.ErrorConfigurationInvalid)) || !strings.Contains(view, "fix the policy") {
		t.Fatalf("Status=%v view=%q; want the failure code and hint", p.Status(), view)
	}
}

func TestHandleRiskAssessed_GivenNoSummaryAndNoCode_ShouldStillSayPlanIsUnavailable(t *testing.T) {
	p, _ := newHarness(t, sdk.RiskAssessedEvent{})

	runCmd(p, p.Activate())

	if view := ansi.Strip(p.View(100, 30)); p.Status() != sdk.StatusError || view != "plan unavailable" {
		t.Fatalf("Status=%v view=%q; want a bare \"plan unavailable\", never a blank screen", p.Status(), view)
	}
}

func TestActivate_GivenAnErroredRun_ShouldAskAgain(t *testing.T) {
	p, h := newHarness(t, sdk.RiskAssessedEvent{Err: errors.New("boom")})
	runCmd(p, p.Activate())
	h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: phantomSummary()}

	runCmd(p, p.Activate())

	if p.Status() != sdk.StatusDone || p.PhantomCount() != 1 {
		t.Fatalf("Status=%v PhantomCount=%d; want the retry to fill the list", p.Status(), p.PhantomCount())
	}
}

func TestCancel_GivenPendingRequest_ShouldReturnToIdleAndIgnoreItsLateResult(t *testing.T) {
	p, _, _ := pending(t)

	p.Cancel()
	deliver(p, sdk.RiskAssessedEvent{Request: 1, Summary: phantomSummary()})

	if p.Status() != sdk.StatusIdle || p.Busy() || p.PhantomCount() != 0 {
		t.Fatalf("Status=%v Busy=%v PhantomCount=%d; want idle with the late result ignored", p.Status(), p.Busy(), p.PhantomCount())
	}
}

func TestCancel_GivenSettledRun_ShouldKeepIt(t *testing.T) {
	p, _ := newHarness(t, sdk.RiskAssessedEvent{})
	runCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()}))

	p.Cancel()
	p.Shutdown()

	if p.Status() != sdk.StatusDone || p.PhantomCount() != 1 {
		t.Fatalf("Status=%v PhantomCount=%d; Cancel must not discard a finished list", p.Status(), p.PhantomCount())
	}
}

func TestHandlePlanCompleted_GivenRequestPending_ShouldTakeThePlanAndIgnoreTheRequest(t *testing.T) {
	p, _, _ := pending(t)

	p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()})
	deliver(p, sdk.RiskAssessedEvent{Request: 1, Err: errors.New("late")})

	if p.Status() != sdk.StatusDone || p.Busy() || p.PhantomCount() != 1 {
		t.Fatalf("Status=%v Busy=%v PhantomCount=%d; want the completed plan kept over the late request", p.Status(), p.Busy(), p.PhantomCount())
	}
}

func TestHandlePlanInvalidated_ShouldReturnToTheEmptyState(t *testing.T) {
	p, h := newHarness(t, sdk.RiskAssessedEvent{Summary: phantomSummary()})
	runCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()}))

	p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})

	if p.Status() != sdk.StatusIdle || p.PhantomCount() != 0 || !strings.Contains(ansi.Strip(p.View(100, 30)), "Run a plan first") {
		t.Fatalf("Status=%v PhantomCount=%d; want the empty state after the plan went stale", p.Status(), p.PhantomCount())
	}
	runCmd(p, p.Activate())
	if len(h.AssessmentRequests) != 1 {
		t.Fatalf("assessment requests = %d; want the stale plan dropped, so the next open asks the host", len(h.AssessmentRequests))
	}
}

func TestHandlePlanInvalidated_GivenRequestPending_ShouldIgnoreItsResult(t *testing.T) {
	p, _, _ := pending(t)

	p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
	deliver(p, sdk.RiskAssessedEvent{Request: 1, Summary: phantomSummary()})

	if p.Status() != sdk.StatusIdle || p.PhantomCount() != 0 {
		t.Fatalf("Status=%v PhantomCount=%d; a result for an invalidated plan must not fill the list", p.Status(), p.PhantomCount())
	}
}

func TestHandleContextChanged_ShouldDropThePlanOnlyWhenTheContextChanges(t *testing.T) {
	next := &sdk.Context{Service: &sdktest.MockService{}}
	tests := []struct {
		name       string
		ev         sdk.ContextChangedEvent
		wantStatus sdk.Status
	}{
		{"switch", sdk.ContextChangedEvent{Next: next, Reason: sdk.ContextSwitched}, sdk.StatusIdle},
		{"no next context", sdk.ContextChangedEvent{}, sdk.StatusDone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := newHarness(t, sdk.RiskAssessedEvent{})
			p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()})

			p.HandleContextChanged(tt.ev)

			if p.Status() != tt.wantStatus {
				t.Fatalf("Status = %v, want %v", p.Status(), tt.wantStatus)
			}
		})
	}
}

func TestUpdate_GivenEsc_ShouldLeaveOnlyWhileLoading(t *testing.T) {
	p, _, _ := pending(t)

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc while loading returned no cmd, so the user cannot leave a running plan")
	}
	if _, ok := cmd().(sdk.DeactivateMsg); !ok {
		t.Fatalf("Esc while loading returned %T, want sdk.DeactivateMsg", cmd())
	}

	settled, _ := newHarness(t, sdk.RiskAssessedEvent{})
	settled.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()})
	if _, cmd := settled.Update(tea.KeyMsg{Type: tea.KeyEsc}); cmd != nil {
		t.Fatal("Esc on a settled list returned a cmd")
	}
}

func TestUpdate_GivenTimerTick_ShouldKeepTickingOnlyWhileLoading(t *testing.T) {
	p, _, activation := pending(t)
	sdktest.LandTick(t, activation)
	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd == nil {
		t.Fatal("tick while loading returned no next tick, so the elapsed time would freeze")
	}

	deliver(p, sdk.RiskAssessedEvent{Request: 1, Summary: phantomSummary()})

	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd != nil {
		t.Fatal("tick after the answer returned a next tick, so the timer would never stop")
	}
}

func TestView_GivenLoading_ShouldShowElapsedTime(t *testing.T) {
	p, _, _ := pending(t)

	view := p.View(80, 20)

	want := sdk.StyleFaintItalic.Render("Loading plan... 0s")
	if view != want {
		t.Fatalf("View() = %q, want %q", view, want)
	}
}

func TestHints_ShouldFollowTheStatus(t *testing.T) {
	loading, _, _ := pending(t)
	failed, _ := newHarness(t, sdk.RiskAssessedEvent{Err: errors.New("boom")})
	runCmd(failed, failed.Activate())

	for name, tt := range map[string]struct {
		p    *Plugin
		want []sdk.KeyHint
	}{
		"loading": {loading, sdk.HintSetBack.Hints()},
		"error":   {failed, sdk.HintSetQuit.Hints()},
	} {
		if got := tt.p.Hints(); !slices.Equal(got, tt.want) {
			t.Errorf("%s: Hints() = %v, want %v", name, got, tt.want)
		}
	}
}
