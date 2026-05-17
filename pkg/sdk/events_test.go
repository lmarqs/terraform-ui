package sdk

import "testing"

func TestChdirChangedEvent_WhenCalled_ShouldSatisfyEventInterface(t *testing.T) {
	e := ChdirChangedEvent{RelPath: "modules/vpc", AbsPath: "/abs", Count: 3}
	e.event()
	var _ Event = e
}

func TestWorkspaceChangedEvent_WhenCalled_ShouldSatisfyEventInterface(t *testing.T) {
	e := WorkspaceChangedEvent{Name: "prod"}
	e.event()
	var _ Event = e
}

func TestPlanCompletedEvent_WhenCalled_ShouldSatisfyEventInterface(t *testing.T) {
	e := PlanCompletedEvent{ResourceCount: 5, PlanFile: "/tmp/plan"}
	e.event()
	var _ Event = e
}

func TestPinsChangedEvent_WhenCalled_ShouldSatisfyEventInterface(t *testing.T) {
	e := PinsChangedEvent{Addresses: []string{"aws_instance.web"}}
	e.event()
	var _ Event = e
}

func TestPlanInvalidatedEvent_WhenCalled_ShouldSatisfyEventInterface(t *testing.T) {
	e := PlanInvalidatedEvent{}
	e.event()
	var _ Event = e
}

func TestWorkspaceCreatedEvent_WhenCalled_ShouldSatisfyEventInterface(t *testing.T) {
	e := WorkspaceCreatedEvent{Name: "staging"}
	e.event()
	var _ Event = e
}

func TestLockDetectedEvent_WhenCalled_ShouldSatisfyEventInterface(t *testing.T) {
	e := LockDetectedEvent{Lock: &StateLock{ID: "abc"}}
	e.event()
	var _ Event = e
}

func TestLockClearedEvent_WhenCalled_ShouldSatisfyEventInterface(t *testing.T) {
	e := LockClearedEvent{}
	e.event()
	var _ Event = e
}

func TestStateRefreshedEvent_WhenCalled_ShouldSatisfyEventInterface(t *testing.T) {
	e := StateRefreshedEvent{}
	e.event()
	var _ Event = e
}
