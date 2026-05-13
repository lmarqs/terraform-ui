package sdk

import tea "github.com/charmbracelet/bubbletea"

type EventBus struct {
	chdirHandlers           []ChdirHandler
	workspaceHandlers       []WorkspaceHandler
	planCompletedHandlers   []PlanCompletedHandler
	pinsHandlers            []PinsHandler
	planInvalidatedHandlers []PlanInvalidatedHandler
}

func NewEventBus(plugins []Plugin) *EventBus {
	b := &EventBus{}
	for _, p := range plugins {
		if h, ok := p.(ChdirHandler); ok {
			b.chdirHandlers = append(b.chdirHandlers, h)
		}
		if h, ok := p.(WorkspaceHandler); ok {
			b.workspaceHandlers = append(b.workspaceHandlers, h)
		}
		if h, ok := p.(PlanCompletedHandler); ok {
			b.planCompletedHandlers = append(b.planCompletedHandlers, h)
		}
		if h, ok := p.(PinsHandler); ok {
			b.pinsHandlers = append(b.pinsHandlers, h)
		}
		if h, ok := p.(PlanInvalidatedHandler); ok {
			b.planInvalidatedHandlers = append(b.planInvalidatedHandlers, h)
		}
	}
	return b
}

func (b *EventBus) Dispatch(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch e := msg.(type) {
	case ChdirChangedEvent:
		for _, h := range b.chdirHandlers {
			if cmd := h.HandleChdirChanged(e); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case WorkspaceChangedEvent:
		for _, h := range b.workspaceHandlers {
			if cmd := h.HandleWorkspaceChanged(e); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case PlanCompletedEvent:
		for _, h := range b.planCompletedHandlers {
			if cmd := h.HandlePlanCompleted(e); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case PinsChangedEvent:
		for _, h := range b.pinsHandlers {
			if cmd := h.HandlePinsChanged(e); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case PlanInvalidatedEvent:
		for _, h := range b.planInvalidatedHandlers {
			if cmd := h.HandlePlanInvalidated(e); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	default:
		return nil
	}

	if len(cmds) == 0 {
		return nil
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}
