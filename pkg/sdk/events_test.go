package sdk

import "testing"

func TestEvents_ShouldSatisfyEventInterface(t *testing.T) {
	events := []Event{
		ChdirChangedEvent{RelPath: "modules/vpc", AbsPath: "/abs", Count: 3},
		WorkspaceChangedEvent{Name: "prod"},
		PlanCompletedEvent{ResourceCount: 5, PlanFile: "/tmp/plan"},
		PinsChangedEvent{Addresses: []string{"aws_instance.web"}},
		PlanInvalidatedEvent{},
		WorkspaceCreatedEvent{Name: "staging"},
		LockDetectedEvent{Lock: &StateLock{ID: "abc"}},
		LockClearedEvent{},
		StateRefreshedEvent{},
	}
	for _, e := range events {
		e.event()
	}
}
