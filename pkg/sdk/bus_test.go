package sdk

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type baseMockPlugin struct{}

func (baseMockPlugin) ID() string                               { return "mock" }
func (baseMockPlugin) Name() string                             { return "Mock" }
func (baseMockPlugin) Description() string                      { return "mock plugin" }
func (baseMockPlugin) Init(_ *PluginDeps) tea.Cmd               { return nil }
func (baseMockPlugin) Update(_ tea.Msg) (Plugin, tea.Cmd)       { return nil, nil }
func (baseMockPlugin) View(_, _ int) string                     { return "" }
func (baseMockPlugin) Configure(_ map[string]interface{}) error { return nil }
func (baseMockPlugin) Ready() bool                              { return true }

type contextHandlerPlugin struct {
	baseMockPlugin
	called bool
	event  ContextChangedEvent
}

func (p *contextHandlerPlugin) HandleContextChanged(e ContextChangedEvent) tea.Cmd {
	p.called = true
	p.event = e
	return nil
}

type planCompletedHandlerPlugin struct {
	baseMockPlugin
	called bool
	event  PlanCompletedEvent
}

func (p *planCompletedHandlerPlugin) HandlePlanCompleted(e PlanCompletedEvent) tea.Cmd {
	p.called = true
	p.event = e
	return nil
}

type planInvalidatedHandlerPlugin struct {
	baseMockPlugin
	called bool
}

func (p *planInvalidatedHandlerPlugin) HandlePlanInvalidated(_ PlanInvalidatedEvent) tea.Cmd {
	p.called = true
	return nil
}

type lockDetectedHandlerPlugin struct {
	baseMockPlugin
	called bool
}

func (p *lockDetectedHandlerPlugin) HandleLockDetected(_ LockDetectedEvent) tea.Cmd {
	p.called = true
	return nil
}

type lockClearedHandlerPlugin struct {
	baseMockPlugin
	called bool
}

func (p *lockClearedHandlerPlugin) HandleLockCleared(_ LockClearedEvent) tea.Cmd {
	p.called = true
	return nil
}

type stateRefreshedHandlerPlugin struct {
	baseMockPlugin
	called bool
}

func (p *stateRefreshedHandlerPlugin) HandleStateRefreshed(_ StateRefreshedEvent) tea.Cmd {
	p.called = true
	return nil
}

type cmdReturningContextHandlerPlugin struct {
	baseMockPlugin
}

type testResultMsg struct{}

func (p *cmdReturningContextHandlerPlugin) HandleContextChanged(_ ContextChangedEvent) tea.Cmd {
	return func() tea.Msg { return testResultMsg{} }
}

func TestNewEventBus_WhenNilPlugins_ShouldNotPanic(t *testing.T) {
	bus := NewEventBus(nil)
	if bus == nil {
		t.Fatal("NewEventBus(nil) returned nil, want non-nil")
	}
}

func TestNewEventBus_WhenEmptyPlugins_ShouldNotPanic(t *testing.T) {
	bus := NewEventBus([]Plugin{})
	if bus == nil {
		t.Fatal("NewEventBus([]) returned nil, want non-nil")
	}
}

func TestNewEventBus_WhenPluginImplementsContextHandler_ShouldDiscoverIt(t *testing.T) {
	p := &contextHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	prev := &Context{WorkingDir: "/old"}
	next := &Context{WorkingDir: "/new"}
	bus.Dispatch(ContextChangedEvent{Prev: prev, Next: next})

	if !p.called {
		t.Error("ContextChangedHandler was not called, want called")
	}
	if p.event.Prev != prev || p.event.Next != next {
		t.Errorf("event Prev/Next not preserved")
	}
}

func TestNewEventBus_WhenPluginImplementsNoHandlers_ShouldStillWork(t *testing.T) {
	p := &baseMockPlugin{}
	bus := NewEventBus([]Plugin{p})

	cmd := bus.Dispatch(ContextChangedEvent{})
	if cmd != nil {
		t.Error("Dispatch returned non-nil cmd with no handlers")
	}
}

func TestDispatch_WhenContextChangedEvent_ShouldRouteToHandler(t *testing.T) {
	p := &contextHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})
	bus.Dispatch(ContextChangedEvent{Next: &Context{WorkingDir: "/x"}})
	if !p.called {
		t.Error("handler not called")
	}
}

func TestDispatch_WhenPlanCompletedEvent_ShouldRouteToHandler(t *testing.T) {
	p := &planCompletedHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})
	bus.Dispatch(PlanCompletedEvent{ResourceCount: 3})
	if !p.called {
		t.Error("handler not called")
	}
	if p.event.ResourceCount != 3 {
		t.Errorf("event.ResourceCount = %d, want 3", p.event.ResourceCount)
	}
}

func TestDispatch_WhenPlanInvalidatedEvent_ShouldRouteToHandler(t *testing.T) {
	p := &planInvalidatedHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})
	bus.Dispatch(PlanInvalidatedEvent{})
	if !p.called {
		t.Error("handler not called")
	}
}

func TestDispatch_WhenLockDetectedEvent_ShouldRouteToHandler(t *testing.T) {
	p := &lockDetectedHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})
	bus.Dispatch(LockDetectedEvent{})
	if !p.called {
		t.Error("handler not called")
	}
}

func TestDispatch_WhenLockClearedEvent_ShouldRouteToHandler(t *testing.T) {
	p := &lockClearedHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})
	bus.Dispatch(LockClearedEvent{})
	if !p.called {
		t.Error("handler not called")
	}
}

func TestDispatch_WhenStateRefreshedEvent_ShouldRouteToHandler(t *testing.T) {
	p := &stateRefreshedHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})
	bus.Dispatch(StateRefreshedEvent{})
	if !p.called {
		t.Error("handler not called")
	}
}

func TestDispatch_WhenUnknownMsg_ShouldReturnNil(t *testing.T) {
	p := &contextHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	type unknownMsg struct{}
	cmd := bus.Dispatch(unknownMsg{})
	if cmd != nil {
		t.Error("Dispatch(unknown) returned non-nil cmd")
	}
	if p.called {
		t.Error("handler called for unknown message")
	}
}

func TestDispatch_WhenMultipleSubscribers_ShouldCallAll(t *testing.T) {
	p1 := &contextHandlerPlugin{}
	p2 := &contextHandlerPlugin{}
	bus := NewEventBus([]Plugin{p1, p2})

	bus.Dispatch(ContextChangedEvent{Next: &Context{}})

	if !p1.called || !p2.called {
		t.Error("not all subscribers called")
	}
}

func TestDispatch_WhenHandlerReturnsCmd_ShouldReturnCmd(t *testing.T) {
	p := &cmdReturningContextHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	cmd := bus.Dispatch(ContextChangedEvent{})
	if cmd == nil {
		t.Fatal("Dispatch returned nil cmd, want non-nil")
	}
}

func TestDispatch_WhenAllHandlersReturnNilCmd_ShouldReturnNil(t *testing.T) {
	p1 := &contextHandlerPlugin{}
	p2 := &contextHandlerPlugin{}
	bus := NewEventBus([]Plugin{p1, p2})

	cmd := bus.Dispatch(ContextChangedEvent{})
	if cmd != nil {
		t.Error("Dispatch returned non-nil when all handlers return nil")
	}
}

func TestDispatch_WhenMultipleHandlersReturnCmds_ShouldBatchAll(t *testing.T) {
	p1 := &cmdReturningContextHandlerPlugin{}
	p2 := &cmdReturningContextHandlerPlugin{}
	bus := NewEventBus([]Plugin{p1, p2})

	cmd := bus.Dispatch(ContextChangedEvent{})
	if cmd == nil {
		t.Fatal("Dispatch returned nil, want batched cmd")
	}
}
