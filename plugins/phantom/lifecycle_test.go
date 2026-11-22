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
	_ sdk.PlanFetchedHandler   = (*Plugin)(nil)
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

// newHarness returns an initialized plugin whose host answers every plan request
// with answer.
func newHarness(t *testing.T, answer sdk.PlanFetchedEvent) (*Plugin, *sdktest.PluginDepsHarness) {
	t.Helper()
	svc := &sdktest.MockService{}
	h := sdktest.NewDeps(svc)
	h.PlanEvent = answer
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

func deliver(p *Plugin, ev sdk.PlanFetchedEvent) {
	runCmd(p, func() tea.Msg { return ev })
}

func pending(t *testing.T) (*Plugin, *sdktest.PluginDepsHarness, tea.Cmd) {
	t.Helper()
	p, h := newHarness(t, sdk.PlanFetchedEvent{Summary: phantomSummary()})
	cmd := p.Activate()
	if p.Status() != sdk.StatusLoading {
		t.Fatalf("setup: status = %v, want loading", p.Status())
	}
	return p, h, cmd
}

func TestHandlePlanCompleted_ShouldFillThePhantomList(t *testing.T) {
	p, _ := newHarness(t, sdk.PlanFetchedEvent{})

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
	p, h := newHarness(t, sdk.PlanFetchedEvent{Summary: phantomSummary()})

	cmd := p.Activate()

	if cmd == nil {
		t.Fatal("Activate() with no plan returned no cmd, so nothing obtains one")
	}
	if p.Status() != sdk.StatusLoading || !p.Busy() {
		t.Fatalf("Status=%v Busy=%v; want loading while the host plans, which holds the directory lock", p.Status(), p.Busy())
	}
	runCmd(p, cmd)
	if h.PlanRequests != 1 || len(h.AssessmentRequests) != 0 {
		t.Fatalf("plan requests = %d, assessment requests = %d; want exactly one plan request and no risk assessment", h.PlanRequests, len(h.AssessmentRequests))
	}
	if !p.Ready() || p.Busy() || p.PhantomCount() != 1 || !strings.Contains(ansi.Strip(p.View(100, 30)), "aws_security_group.web") {
		t.Fatalf("Ready=%v Busy=%v PhantomCount=%d; want the phantom list filled", p.Ready(), p.Busy(), p.PhantomCount())
	}
}

func TestActivate_GivenReviewedPlan_ShouldKeepItsListWithoutAskingTheHost(t *testing.T) {
	p, h := newHarness(t, sdk.PlanFetchedEvent{})
	runCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()}))

	cmd := p.Activate()

	if cmd != nil || h.PlanRequests != 0 || p.Status() != sdk.StatusDone || p.PhantomCount() != 1 {
		t.Fatalf("cmd=%v requests=%d status=%v phantoms=%d; want the reviewed plan's list kept and no request", cmd != nil, h.PlanRequests, p.Status(), p.PhantomCount())
	}
}

func TestActivate_GivenRequestPending_ShouldNotAskAgain(t *testing.T) {
	p, h, first := pending(t)

	if cmd := p.Activate(); cmd != nil {
		t.Fatal("Activate() while loading returned a cmd, so a second plan would start")
	}
	runCmd(p, first)
	if h.PlanRequests != 1 {
		t.Fatalf("plan requests = %d, want 1", h.PlanRequests)
	}
}

func TestHandlePlanFetched_GivenAnotherRequestsEvent_ShouldIgnoreIt(t *testing.T) {
	p, _, _ := pending(t)

	deliver(p, sdk.PlanFetchedEvent{Request: 99, Summary: phantomSummary()})

	if p.Status() != sdk.StatusLoading || p.PhantomCount() != 0 {
		t.Fatalf("Status=%v PhantomCount=%d; another request's result settled the pending one", p.Status(), p.PhantomCount())
	}
}

func TestHandlePlanFetched_GivenNothingPending_ShouldIgnoreIt(t *testing.T) {
	p, _ := newHarness(t, sdk.PlanFetchedEvent{})

	deliver(p, sdk.PlanFetchedEvent{Request: 1, Summary: phantomSummary()})

	if p.Status() != sdk.StatusIdle {
		t.Fatalf("Status = %v, want idle: an unrequested result must not fill the list", p.Status())
	}
}

func TestHandlePlanFetched_GivenFailedPlan_ShouldShowTheErrorAndSettle(t *testing.T) {
	p, _ := newHarness(t, sdk.PlanFetchedEvent{Err: errors.New("terraform plan: no configuration files")})

	runCmd(p, p.Activate())

	view := ansi.Strip(p.View(100, 30))
	if p.Status() != sdk.StatusError || !p.Ready() || p.Busy() {
		t.Fatalf("Status=%v Ready=%v Busy=%v; want a settled error so a tape or headless run does not wait forever", p.Status(), p.Ready(), p.Busy())
	}
	if !strings.Contains(view, "plan unavailable: terraform plan: no configuration files") {
		t.Fatalf("view = %q; want terraform's error as is", view)
	}
}

func TestHandlePlanFetched_GivenNoSummaryAndNoError_ShouldStillSayPlanIsUnavailable(t *testing.T) {
	p, _ := newHarness(t, sdk.PlanFetchedEvent{})

	runCmd(p, p.Activate())

	if view := ansi.Strip(p.View(100, 30)); p.Status() != sdk.StatusError || view != "plan unavailable" {
		t.Fatalf("Status=%v view=%q; want a bare \"plan unavailable\", never a blank screen", p.Status(), view)
	}
}

func TestActivate_GivenAnErroredRun_ShouldAskAgain(t *testing.T) {
	p, h := newHarness(t, sdk.PlanFetchedEvent{Err: errors.New("boom")})
	runCmd(p, p.Activate())
	h.PlanEvent = sdk.PlanFetchedEvent{Summary: phantomSummary()}

	runCmd(p, p.Activate())

	if p.Status() != sdk.StatusDone || p.PhantomCount() != 1 {
		t.Fatalf("Status=%v PhantomCount=%d; want the retry to fill the list", p.Status(), p.PhantomCount())
	}
}

func TestCancel_GivenPendingRequest_ShouldReturnToIdleAndIgnoreItsLateResult(t *testing.T) {
	p, _, _ := pending(t)

	p.Cancel()
	deliver(p, sdk.PlanFetchedEvent{Request: 1, Summary: phantomSummary()})

	if p.Status() != sdk.StatusIdle || p.Busy() || p.PhantomCount() != 0 {
		t.Fatalf("Status=%v Busy=%v PhantomCount=%d; want idle with the late result ignored", p.Status(), p.Busy(), p.PhantomCount())
	}
}

func TestCancel_GivenSettledRun_ShouldKeepIt(t *testing.T) {
	p, _ := newHarness(t, sdk.PlanFetchedEvent{})
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
	deliver(p, sdk.PlanFetchedEvent{Request: 1, Err: errors.New("late")})

	if p.Status() != sdk.StatusDone || p.Busy() || p.PhantomCount() != 1 {
		t.Fatalf("Status=%v Busy=%v PhantomCount=%d; want the completed plan kept over the late request", p.Status(), p.Busy(), p.PhantomCount())
	}
}

func TestHandlePlanInvalidated_ShouldReturnToTheEmptyState(t *testing.T) {
	p, h := newHarness(t, sdk.PlanFetchedEvent{Summary: phantomSummary()})
	runCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()}))

	p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})

	if p.Status() != sdk.StatusIdle || p.PhantomCount() != 0 || !strings.Contains(ansi.Strip(p.View(100, 30)), "Run a plan first") {
		t.Fatalf("Status=%v PhantomCount=%d; want the empty state after the plan went stale", p.Status(), p.PhantomCount())
	}
	runCmd(p, p.Activate())
	if h.PlanRequests != 1 {
		t.Fatalf("plan requests = %d; want the stale plan dropped, so the next open asks the host", h.PlanRequests)
	}
}

func TestHandlePlanInvalidated_GivenRequestPending_ShouldIgnoreItsResult(t *testing.T) {
	p, _, _ := pending(t)

	p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
	deliver(p, sdk.PlanFetchedEvent{Request: 1, Summary: phantomSummary()})

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
			p, _ := newHarness(t, sdk.PlanFetchedEvent{})
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

	settled, _ := newHarness(t, sdk.PlanFetchedEvent{})
	settled.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()})
	if _, cmd := settled.Update(tea.KeyMsg{Type: tea.KeyEsc}); cmd != nil {
		t.Fatal("Esc on a settled list returned a cmd")
	}
}

func TestUpdate_GivenEscInTheErrorState_ShouldLeave(t *testing.T) {
	p, _ := newHarness(t, sdk.PlanFetchedEvent{Err: errors.New("boom")})
	runCmd(p, p.Activate())

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("Esc on the error screen returned no cmd, so Esc back is a hint for a key that does nothing")
	}
	if _, ok := cmd().(sdk.DeactivateMsg); !ok {
		t.Fatalf("Esc on the error screen returned %T, want sdk.DeactivateMsg", cmd())
	}
}

func TestUpdate_GivenCtrlRInTheErrorState_ShouldAskTheHostAgain(t *testing.T) {
	p, h := newHarness(t, sdk.PlanFetchedEvent{Err: errors.New("boom")})
	runCmd(p, p.Activate())
	h.PlanEvent = sdk.PlanFetchedEvent{Summary: phantomSummary()}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	runCmd(p, cmd)

	if h.PlanRequests != 2 || p.Status() != sdk.StatusDone || p.PhantomCount() != 1 {
		t.Fatalf("requests=%d status=%v phantoms=%d; want ctrl+r to retry and fill the list", h.PlanRequests, p.Status(), p.PhantomCount())
	}
}

func TestUpdate_GivenCtrlROutsideTheErrorState_ShouldDoNothing(t *testing.T) {
	p, h := newHarness(t, sdk.PlanFetchedEvent{})
	runCmd(p, p.HandlePlanCompleted(sdk.PlanCompletedEvent{Summary: phantomSummary()}))

	if _, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR}); cmd != nil || h.PlanRequests != 0 {
		t.Fatal("ctrl+r on a settled list started a request, but no hint offers it")
	}
}

func TestUpdate_GivenTimerTick_ShouldKeepTickingOnlyWhileLoading(t *testing.T) {
	p, _, activation := pending(t)
	sdktest.LandTick(t, activation)
	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd == nil {
		t.Fatal("tick while loading returned no next tick, so the elapsed time would freeze")
	}

	deliver(p, sdk.PlanFetchedEvent{Request: 1, Summary: phantomSummary()})

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
	failed, _ := newHarness(t, sdk.PlanFetchedEvent{Err: errors.New("boom")})
	runCmd(failed, failed.Activate())

	for name, tt := range map[string]struct {
		p    *Plugin
		want []sdk.KeyHint
	}{
		"loading": {loading, sdk.HintSetBack.Hints()},
		"error":   {failed, (sdk.HintSetRetry | sdk.HintSetBack).Hints()},
	} {
		if got := tt.p.Hints(); !slices.Equal(got, tt.want) {
			t.Errorf("%s: Hints() = %v, want %v", name, got, tt.want)
		}
	}
}
