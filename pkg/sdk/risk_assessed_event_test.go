package sdk

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type riskAssessedHandlerPlugin struct {
	baseMockPlugin
	events []RiskAssessedEvent
}

func (p *riskAssessedHandlerPlugin) HandleRiskAssessed(e RiskAssessedEvent) tea.Cmd {
	p.events = append(p.events, e)
	return nil
}

func TestRiskAssessedEvent_ShouldSatisfyEventInterface(t *testing.T) {
	var e Event = RiskAssessedEvent{}
	e.event()
}

func TestDispatch_WhenRiskAssessedEvent_ShouldRouteToHandler(t *testing.T) {
	handler := &riskAssessedHandlerPlugin{}
	other := &planInvalidatedHandlerPlugin{}
	bus := NewEventBus([]Plugin{handler, other})
	ev := RiskAssessedEvent{
		Summary:    &PlanSummary{},
		Assessment: PolicyAssessment{Status: AssessmentUnavailable, ErrorCode: ErrorSourceUnreadable},
		Err:        errors.New("plan failed"),
	}

	bus.Dispatch(ev)

	if len(handler.events) != 1 {
		t.Fatalf("RiskAssessedHandler received %d events, want 1", len(handler.events))
	}
	got := handler.events[0]
	if got.Summary != ev.Summary || got.Assessment.ErrorCode != ev.Assessment.ErrorCode || got.Err != ev.Err {
		t.Errorf("handler received %+v, want %+v", got, ev)
	}
	if other.called {
		t.Error("RiskAssessedEvent reached a plugin that does not handle it")
	}
}
