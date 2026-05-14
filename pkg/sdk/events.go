package sdk

import tea "github.com/charmbracelet/bubbletea"

type Event interface {
	tea.Msg
	event()
}

type ChdirChangedEvent struct {
	RelPath string
	AbsPath string
	Count   int
}

func (ChdirChangedEvent) event() {}

type WorkspaceChangedEvent struct {
	Name string
}

func (WorkspaceChangedEvent) event() {}

type WorkspaceCreatedEvent struct {
	Name string
}

func (WorkspaceCreatedEvent) event() {}

type PlanCompletedEvent struct {
	Summary       *PlanSummary
	ResourceCount int
	PlanFile      string
}

func (PlanCompletedEvent) event() {}

type PinsChangedEvent struct {
	Addresses []string
}

func (PinsChangedEvent) event() {}

type PlanInvalidatedEvent struct{}

func (PlanInvalidatedEvent) event() {}

type ChdirHandler interface {
	HandleChdirChanged(ChdirChangedEvent) tea.Cmd
}

type WorkspaceHandler interface {
	HandleWorkspaceChanged(WorkspaceChangedEvent) tea.Cmd
}

type PlanCompletedHandler interface {
	HandlePlanCompleted(PlanCompletedEvent) tea.Cmd
}

type PinsHandler interface {
	HandlePinsChanged(PinsChangedEvent) tea.Cmd
}

type PlanInvalidatedHandler interface {
	HandlePlanInvalidated(PlanInvalidatedEvent) tea.Cmd
}
