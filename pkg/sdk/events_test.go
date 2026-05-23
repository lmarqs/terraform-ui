package sdk

import "testing"

func TestEvents_ShouldSatisfyEventInterface(t *testing.T) {
	events := []Event{
		ContextChangedEvent{},
		PlanCompletedEvent{ResourceCount: 5, PlanFile: "/tmp/plan"},
		PlanInvalidatedEvent{},
		LockDetectedEvent{Lock: &StateLock{ID: "abc"}},
		LockClearedEvent{},
		StateRefreshedEvent{},
	}
	for _, e := range events {
		e.event()
	}
}
