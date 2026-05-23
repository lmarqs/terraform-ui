package sdk

import tea "github.com/charmbracelet/bubbletea"

type EventBus struct {
	contextHandlers         []ContextChangedHandler
	planCompletedHandlers   []PlanCompletedHandler
	planInvalidatedHandlers []PlanInvalidatedHandler
	lockDetectedHandlers    []LockDetectedHandler
	lockClearedHandlers     []LockClearedHandler
	stateRefreshedHandlers  []StateRefreshedHandler
}

func NewEventBus(plugins []Plugin) *EventBus {
	b := &EventBus{}
	for _, p := range plugins {
		if h, ok := p.(ContextChangedHandler); ok {
			b.contextHandlers = append(b.contextHandlers, h)
		}
		if h, ok := p.(PlanCompletedHandler); ok {
			b.planCompletedHandlers = append(b.planCompletedHandlers, h)
		}
		if h, ok := p.(PlanInvalidatedHandler); ok {
			b.planInvalidatedHandlers = append(b.planInvalidatedHandlers, h)
		}
		if h, ok := p.(LockDetectedHandler); ok {
			b.lockDetectedHandlers = append(b.lockDetectedHandlers, h)
		}
		if h, ok := p.(LockClearedHandler); ok {
			b.lockClearedHandlers = append(b.lockClearedHandlers, h)
		}
		if h, ok := p.(StateRefreshedHandler); ok {
			b.stateRefreshedHandlers = append(b.stateRefreshedHandlers, h)
		}
	}
	return b
}

func (b *EventBus) Dispatch(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch e := msg.(type) {
	case ContextChangedEvent:
		for _, h := range b.contextHandlers {
			cmds = append(cmds, h.HandleContextChanged(e))
		}
	case PlanCompletedEvent:
		for _, h := range b.planCompletedHandlers {
			cmds = append(cmds, h.HandlePlanCompleted(e))
		}
	case PlanInvalidatedEvent:
		for _, h := range b.planInvalidatedHandlers {
			cmds = append(cmds, h.HandlePlanInvalidated(e))
		}
	case LockDetectedEvent:
		for _, h := range b.lockDetectedHandlers {
			cmds = append(cmds, h.HandleLockDetected(e))
		}
	case LockClearedEvent:
		for _, h := range b.lockClearedHandlers {
			cmds = append(cmds, h.HandleLockCleared(e))
		}
	case StateRefreshedEvent:
		for _, h := range b.stateRefreshedHandlers {
			cmds = append(cmds, h.HandleStateRefreshed(e))
		}
	default:
		return nil
	}

	// tea.Batch tolerates nil entries; filter empty/single cases for clarity.
	if len(cmds) == 0 {
		return nil
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}
