package plan

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func allHigh(summary *sdk.PlanSummary) sdk.PolicyAssessment {
	high := sdk.RiskHigh
	assessment := sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &high}
	for _, change := range summary.Changes {
		assessment.Resources = append(assessment.Resources, sdk.PolicyResourceAssessment{Address: change.Resource.Address, Status: sdk.ResourceClassified, Level: &high})
	}
	return assessment
}

func moduleSummary() *sdk.PlanSummary {
	return &sdk.PlanSummary{ToUpdate: 5, Changes: []sdk.PlanChange{
		{Resource: sdk.Resource{Address: "module.app.aws_instance.a"}, Action: sdk.ActionUpdate},
		{Resource: sdk.Resource{Address: "module.app.aws_instance.b"}, Action: sdk.ActionUpdate},
		{Resource: sdk.Resource{Address: "module.db.aws_db_instance.main"}, Action: sdk.ActionUpdate},
		{Resource: sdk.Resource{Address: "aws_iam_role.deploy"}, Action: sdk.ActionUpdate},
		{Resource: sdk.Resource{Address: "aws_s3_bucket.logs"}, Action: sdk.ActionUpdate},
	}}
}

func pressKey(t *testing.T, p *Plugin, key tea.KeyMsg) tea.Cmd {
	t.Helper()
	return p.stack.Update(key)
}

func TestTogglePin_GivenCursorOnRow3OfExpandedTree_ShouldKeepCursorExpansionAndBadges(t *testing.T) {
	summary := moduleSummary()
	p, _, h := newAssessedPlugin(t, summary, allHigh(summary))
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	pressKey(t, p, tea.KeyMsg{Type: tea.KeyCtrlT})
	pressKey(t, p, tea.KeyMsg{Type: tea.KeyEnter})
	pressKey(t, p, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pressKey(t, p, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	row, visible := p.CursorPosition()
	if row != 3 || !strings.Contains(ansi.Strip(p.View(120, 40)), "aws_instance.a") || !strings.Contains(p.View(120, 40), "[HIGH]") {
		t.Fatalf("setup: cursor row=%d view=%q; want row 3 inside expanded module.app with HIGH badges", row, ansi.Strip(p.View(120, 40)))
	}
	selected := p.SelectedChange().Resource.Address

	pinMsg, ok := pressKey(t, p, tea.KeyMsg{Type: tea.KeySpace})().(sdk.PinToggleRequestMsg)
	if !ok {
		t.Fatal("space on a leaf did not request a pin toggle")
	}
	prev := h.Ctx
	h.Ctx = prev.TogglePins(pinMsg.Addresses...)
	reassess := sdk.NewEventBus([]sdk.Plugin{p}).Dispatch(sdk.ContextChangedEvent{Prev: prev, Next: h.Ctx, Reason: sdk.ContextPinsChanged})

	assertKept := func(phase string) {
		t.Helper()
		view := ansi.Strip(p.View(120, 40))
		gotRow, gotVisible := p.CursorPosition()
		if gotRow != 3 || gotVisible != visible || p.SelectedChange() == nil || p.SelectedChange().Resource.Address != selected {
			t.Fatalf("%s: cursor row=%d of %d on %v; want row 3 of %d on %s", phase, gotRow, gotVisible, p.SelectedChange(), visible, selected)
		}
		if !strings.Contains(view, "aws_instance.a") {
			t.Fatalf("%s: module.app collapsed after the pin toggle:\n%s", phase, view)
		}
		if strings.Contains(view, "input_incomplete") {
			t.Fatalf("%s: rows show input_incomplete for a pending reassessment:\n%s", phase, view)
		}
	}
	assertKept("reassessment pending")
	drainAssessmentPlan(t, p, reassess, nil)
	assertKept("reassessment delivered")
	if view := p.View(120, 40); !strings.Contains(view, "[HIGH]") {
		t.Fatalf("delivered reassessment shows no host badge:\n%s", ansi.Strip(view))
	}
}

func TestActivate_GivenFinishedPlan_ShouldNeverShowRunningPlanWhileReassessing(t *testing.T) {
	summary := moduleSummary()
	p, svc, _ := newAssessedPlugin(t, summary, allHigh(summary))
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)

	pending := p.Activate(Input{})

	view := ansi.Strip(p.View(120, 40))
	if strings.Contains(view, "Running terraform plan") || !strings.Contains(view, "aws_s3_bucket.logs") {
		t.Fatalf("re-entered finished Plan view = %q; want the reviewed rows, not a running plan", view)
	}
	drainAssessmentPlan(t, p, pending, nil)
	if len(svc.PlanCalls) != 1 {
		t.Fatalf("re-entry planned %d times; want the reviewed plan reused", len(svc.PlanCalls))
	}
}

// TestCancel_GivenReassessmentPending_ShouldKeepPlanDoneAndReuseItOnNextEntry
// models the App during a reassessment: Esc cancels only a plugin that is not
// Busy (app.go Esc handling), while DeactivateMsg and :q! always cancel.
func TestCancel_GivenReassessmentPending_ShouldKeepPlanDoneAndReuseItOnNextEntry(t *testing.T) {
	for _, tc := range []struct {
		name  string
		leave func(p *Plugin)
	}{
		{"esc", func(p *Plugin) {
			if !p.Busy() {
				p.Cancel()
			}
		}},
		{"deactivate or force quit", func(p *Plugin) { p.Cancel() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := moduleSummary()
			p, svc, _ := newAssessedPlugin(t, summary, allHigh(summary))
			drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
			pending := p.Activate(Input{})

			tc.leave(p)

			if p.Status() != sdk.StatusDone || p.ExitCode() != 2 {
				t.Fatalf("leaving during reassessment: status=%v exit=%d; want the good plan to stay Done", p.Status(), p.ExitCode())
			}
			if view := ansi.Strip(p.View(120, 40)); strings.Contains(view, "cancelled") || !strings.Contains(view, "aws_s3_bucket.logs") {
				t.Fatalf("leaving during reassessment view = %q; want the reviewed rows", view)
			}
			drainAssessmentPlan(t, p, pending, nil)
			next := p.Activate(Input{})
			if view := ansi.Strip(p.View(120, 40)); strings.Contains(view, "Running terraform plan") {
				t.Fatalf("next entry view = %q; want no running plan", view)
			}
			drainAssessmentPlan(t, p, next, nil)
			if len(svc.PlanCalls) != 1 {
				t.Fatalf("next entry after leaving planned %d times; want 1", len(svc.PlanCalls))
			}
		})
	}
}

func TestCancel_GivenReassessmentPending_ShouldReleaseBusy(t *testing.T) {
	summary := moduleSummary()
	p, _, _ := newAssessedPlugin(t, summary, allHigh(summary))
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	p.Activate(Input{})
	p.Cancel()
	if p.Busy() {
		t.Fatal("Plan stays busy after Cancel; DeactivateMsg and :q! could not release it")
	}
}

func TestHandlePlanInvalidated_GivenIdleOrError_ShouldNotMarkPlanDone(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(t *testing.T, p *Plugin)
	}{
		{"idle", func(*testing.T, *Plugin) {}},
		{"failed replan", func(t *testing.T, p *Plugin) {
			drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
			drainAssessmentPlan(t, p, p.Refresh(), nil)
		}},
		{"cancelled replan", func(t *testing.T, p *Plugin) {
			drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
			p.Refresh()
			p.Cancel()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := moduleSummary()
			calls := 0
			p, svc, _ := newAssessedPlugin(t, summary, allHigh(summary))
			svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
				calls++
				if calls > 1 {
					return nil, errors.New("terraform planning failed")
				}
				return summary, nil
			}
			tc.arrange(t, p)
			before := p.Status()
			if before == sdk.StatusDone {
				t.Fatalf("setup left status Done; want Idle or Error")
			}

			drainAssessmentPlan(t, p, func() tea.Msg { return sdk.PlanInvalidatedEvent{} }, nil)

			if p.Status() == sdk.StatusDone {
				t.Fatalf("PlanInvalidatedEvent turned status %v into Done", before)
			}
		})
	}
}

// TestActivate_GivenFreshPlan_ShouldStayBusyUntilAssessmentArrives walks the
// Cmds depth-first, as the macro driver does for headless `tfui plan`, which
// prints Stdout and Stderr as soon as Ready() turns true.
func TestActivate_GivenFreshPlan_ShouldStayBusyUntilAssessmentArrives(t *testing.T) {
	low := sdk.RiskLow
	summary := &sdk.PlanSummary{ToDelete: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}
	p, svc, h := newAssessedPlugin(t, summary, sdk.PolicyAssessment{Status: sdk.AssessmentComplete, OverallLevel: &low, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_db_instance.database", Status: sdk.ResourceClassified, Level: &low}}})
	bus := sdk.NewEventBus([]sdk.Plugin{p})

	var firstReady []byte
	sawReady, sawResult := false, false
	var walk func(tea.Cmd)
	walk = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range batch {
				walk(sub)
			}
			return
		}
		if _, ok := msg.(ui.TimerTickMsg); ok {
			return
		}
		var next tea.Cmd
		if _, ok := msg.(sdk.Event); ok {
			next = bus.Dispatch(msg)
		} else {
			_, next = p.Update(msg)
		}
		if _, ok := msg.(PlanResultMsg); ok {
			sawResult = true
			if !p.Busy() || p.Ready() {
				t.Errorf("after the plan result, before its assessment: busy=%v ready=%v; want busy and not ready", p.Busy(), p.Ready())
			}
		}
		if !sawReady && p.Ready() {
			sawReady = true
			firstReady, _ = p.Stdout()
		}
		walk(next)
	}
	walk(p.Activate(Input{}))

	if !sawResult || !sawReady {
		t.Fatalf("plan result seen=%v ready seen=%v; want both", sawResult, sawReady)
	}
	if !strings.Contains(string(firstReady), "Risk: low") {
		t.Fatalf("Stdout at first Ready = %q; want the host's LOW classification", firstReady)
	}
	if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != summary || len(svc.PlanCalls) != 1 {
		t.Fatalf("assessment requests=%v plan calls=%d; want one host request for the planned summary", h.AssessmentRequests, len(svc.PlanCalls))
	}
	if p.Busy() {
		t.Fatal("Plan still busy after its assessment arrived")
	}
}

// Plan requests assessments only for a summary it holds, so a nil-summary event
// answers another plugin's host-planned request.
func TestHandleRiskAssessed_GivenNoSummaryAndNilSummaryEvent_ShouldIgnoreIt(t *testing.T) {
	for _, tc := range []struct {
		name string
		ev   sdk.RiskAssessedEvent
	}{
		{"policy failure", sdk.RiskAssessedEvent{Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorSourceUnreadable}, Hint: "check risk_policy"}},
		{"host plan error", sdk.RiskAssessedEvent{Err: errors.New("terraform plan failed")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newTestPlugin(&sdktest.MockService{})

			sdk.NewEventBus([]sdk.Plugin{p}).Dispatch(tc.ev)

			if got := p.Assessment(); got != nil {
				t.Fatalf("Plan without a summary took another request's assessment: %+v", got)
			}
		})
	}
}

func TestView_GivenPlanDoneAndAssessmentPending_ShouldShowRiskAssessmentInProgressUntilAssessed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		summary *sdk.PlanSummary
		filter  string
	}{
		{"changes listed", moduleSummary(), ""},
		{"no changes", &sdk.PlanSummary{}, ""},
		{"filter matches nothing", moduleSummary(), "zzzzzz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assessed := sdk.RiskAssessedEvent{Summary: tc.summary, Assessment: allHigh(tc.summary)}
			p, _, _ := newAssessedPlugin(t, tc.summary, assessed.Assessment, withholdAssessment(&assessed.Request))
			drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
			if p.Status() != sdk.StatusDone || !p.Busy() {
				t.Fatalf("setup: status=%v busy=%v; want Done and busy while the assessment is pending", p.Status(), p.Busy())
			}
			if tc.filter != "" {
				p.SetFilter(tc.filter)
				if len(p.filtered) != 0 {
					t.Fatalf("setup: filter %q matched %d changes; want none", tc.filter, len(p.filtered))
				}
			}

			const indicator = "Loading risk assessment..."
			if view := ansi.Strip(p.View(120, 40)); !strings.Contains(view, indicator) {
				t.Fatalf("view while the assessment is pending = %q; want it to contain %q", view, indicator)
			}

			drainAssessmentPlan(t, p, func() tea.Msg { return assessed }, nil)
			if view := ansi.Strip(p.View(120, 40)); strings.Contains(view, indicator) {
				t.Fatalf("view after the assessment arrived = %q; want no %q", view, indicator)
			}
		})
	}
}
