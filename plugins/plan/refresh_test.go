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
)

// snapshotService is a Service whose plan is an immutable snapshot, as `-plan -`
// seeds it.
type snapshotService struct{ *sdktest.MockService }

func (snapshotService) PlanSnapshot() bool { return true }

func hasHint(hints []sdk.KeyHint, key string) bool {
	for _, h := range hints {
		if h.Key == key {
			return true
		}
	}
	return false
}

func TestRefresh_ShouldAskTheServiceToRereadASeededPlan(t *testing.T) {
	summary := &sdk.PlanSummary{ToCreate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate}}}
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return summary, nil }}
	p := newTestPlugin(svc)
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)

	drainAssessmentPlan(t, p, p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR}), nil)

	if len(svc.PlanCalls) != 2 {
		t.Fatalf("plan calls = %d, want 2 (Activate, ^r)", len(svc.PlanCalls))
	}
	if svc.PlanCalls[0].Reread {
		t.Error("Activate asked to re-read the plan file, want the seeded plan served as is")
	}
	if !svc.PlanCalls[1].Reread {
		t.Error("^r did not ask to re-read the plan file, want a fresh read")
	}
}

func TestRefresh_WhenThePlanFileCannotBeRead_ShouldShowTheErrorInPlace(t *testing.T) {
	summary := &sdk.PlanSummary{ToCreate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate}}}
	fail := false
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		if fail {
			return nil, errors.New("reading plan file: open f.json: no such file or directory")
		}
		return summary, nil
	}}
	p := newTestPlugin(svc)
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	fail = true

	drainAssessmentPlan(t, p, p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR}), nil)

	if p.status != sdk.StatusError {
		t.Fatalf("status = %v, want Error", p.status)
	}
	if view := ansi.Strip(p.View(100, 20)); !strings.Contains(view, "reading plan file: open f.json: no such file or directory") {
		t.Errorf("view = %q, want the read error as it came", view)
	}
	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want the list frame alone", p.stack.Depth())
	}
}

func TestRefresh_GivenAnImmutableSnapshot_ShouldNotOfferOrRunIt(t *testing.T) {
	withChanges := &sdk.PlanSummary{ToCreate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate}}}
	tests := []struct {
		name    string
		status  sdk.Status
		summary *sdk.PlanSummary
	}{
		{"DoneWithChanges", sdk.StatusDone, withChanges},
		{"DoneWithoutChanges", sdk.StatusDone, &sdk.PlanSummary{}},
		{"Error", sdk.StatusError, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &sdktest.MockService{}
			p := newTestPlugin(snapshotService{svc})
			p.status = tc.status
			p.summary = tc.summary

			if hasHint(p.stack.Hints(), "^r") {
				t.Errorf("hints = %v, want no ^r for an immutable snapshot", p.stack.Hints())
			}
			if cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR}); cmd != nil {
				t.Error("^r returned a command, want the key inert")
			}
			if p.status != tc.status || len(svc.PlanCalls) != 0 {
				t.Errorf("status = %v, plan calls = %d; want the plan untouched", p.status, len(svc.PlanCalls))
			}
		})
	}
}

func TestActivate_ShouldRereadASeededPlanOnlyWhenRetryingAfterAnError(t *testing.T) {
	tests := []struct {
		name   string
		status sdk.Status
		want   bool
	}{
		{"FirstEntry", sdk.StatusIdle, false},
		{"AfterError", sdk.StatusError, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &sdktest.MockService{}
			p := newTestPlugin(svc)
			p.status = tc.status

			drainAssessmentPlan(t, p, p.Activate(Input{}), nil)

			if len(svc.PlanCalls) != 1 || svc.PlanCalls[0].Reread != tc.want {
				t.Errorf("plan calls = %+v, want one with Reread=%v", svc.PlanCalls, tc.want)
			}
		})
	}
}
