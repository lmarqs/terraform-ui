package sdktest

import (
	"reflect"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestNewDeps_RequestAssessmentRecordsSummaryAndEmitsCannedEvent(t *testing.T) {
	svc := &MockService{}
	h := NewDeps(svc)
	canned := sdk.RiskAssessedEvent{
		Summary:    &sdk.PlanSummary{},
		Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentComplete, PlanID: "canned"},
	}
	h.AssessmentEvent = canned
	requested := &sdk.PlanSummary{}
	if h.Deps.RequestAssessment == nil {
		t.Fatal("harness deps provide no RequestAssessment fake")
	}

	cmd := h.Deps.RequestAssessment(requested)
	if cmd == nil {
		t.Fatal("RequestAssessment returned nil cmd")
	}
	msg := cmd()

	got, ok := msg.(sdk.RiskAssessedEvent)
	if !ok {
		t.Fatalf("expected RiskAssessedEvent, got %T", msg)
	}
	if !reflect.DeepEqual(got, canned) || got.Summary != canned.Summary {
		t.Errorf("event = %+v, want canned %+v", got, canned)
	}
	if len(h.AssessmentRequests) != 1 || h.AssessmentRequests[0] != requested {
		t.Errorf("AssessmentRequests = %v, want [%p]", h.AssessmentRequests, requested)
	}
	if len(svc.PlanCalls) != 0 {
		t.Errorf("fake RequestAssessment ran Service.Plan %d times", len(svc.PlanCalls))
	}
}
