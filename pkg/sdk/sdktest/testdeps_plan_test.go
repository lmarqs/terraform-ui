package sdktest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestNewDeps_RequestPlanCountsRequestsAndEmitsCannedEvent(t *testing.T) {
	svc := &MockService{}
	h := NewDeps(svc)
	canned := sdk.PlanFetchedEvent{Summary: &sdk.PlanSummary{}, Err: errors.New("canned")}
	h.PlanEvent = canned
	if h.Deps.RequestPlan == nil {
		t.Fatal("harness deps provide no RequestPlan fake")
	}

	id, cmd := h.Deps.RequestPlan()
	if id == 0 || cmd == nil {
		t.Fatalf("RequestPlan returned id=%d cmd=%v; want a nonzero ID and a cmd", id, cmd != nil)
	}
	got, ok := cmd().(sdk.PlanFetchedEvent)
	if !ok {
		t.Fatalf("expected PlanFetchedEvent, got %T", cmd())
	}
	canned.Request = id
	if !reflect.DeepEqual(got, canned) || got.Summary != canned.Summary {
		t.Errorf("event = %+v, want canned %+v", got, canned)
	}
	if h.PlanRequests != 1 || len(h.AssessmentRequests) != 0 || len(svc.PlanCalls) != 0 {
		t.Errorf("PlanRequests=%d AssessmentRequests=%d Service.Plan=%d; want one plan request and nothing else", h.PlanRequests, len(h.AssessmentRequests), len(svc.PlanCalls))
	}
}
