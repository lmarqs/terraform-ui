package sdk

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type baseMockPlugin struct{}

func (baseMockPlugin) ID() string                               { return "mock" }
func (baseMockPlugin) Name() string                             { return "Mock" }
func (baseMockPlugin) Description() string                      { return "mock plugin" }
func (baseMockPlugin) Init(_ *Context) tea.Cmd                  { return nil }
func (baseMockPlugin) Update(_ tea.Msg) (Plugin, tea.Cmd)       { return nil, nil }
func (baseMockPlugin) View(_, _ int) string                     { return "" }
func (baseMockPlugin) Configure(_ map[string]interface{}) error { return nil }
func (baseMockPlugin) Ready() bool                              { return true }

type chdirHandlerPlugin struct {
	baseMockPlugin
	called bool
	event  ChdirChangedEvent
}

func (p *chdirHandlerPlugin) HandleChdirChanged(e ChdirChangedEvent) tea.Cmd {
	p.called = true
	p.event = e
	return nil
}

type workspaceHandlerPlugin struct {
	baseMockPlugin
	called bool
	event  WorkspaceChangedEvent
}

func (p *workspaceHandlerPlugin) HandleWorkspaceChanged(e WorkspaceChangedEvent) tea.Cmd {
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

type pinsHandlerPlugin struct {
	baseMockPlugin
	called bool
	event  PinsChangedEvent
}

func (p *pinsHandlerPlugin) HandlePinsChanged(e PinsChangedEvent) tea.Cmd {
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

type multiHandlerPlugin struct {
	baseMockPlugin
	chdirCalled     bool
	workspaceCalled bool
	pinsCalled      bool
}

func (p *multiHandlerPlugin) HandleChdirChanged(_ ChdirChangedEvent) tea.Cmd {
	p.chdirCalled = true
	return nil
}

func (p *multiHandlerPlugin) HandleWorkspaceChanged(_ WorkspaceChangedEvent) tea.Cmd {
	p.workspaceCalled = true
	return nil
}

func (p *multiHandlerPlugin) HandlePinsChanged(_ PinsChangedEvent) tea.Cmd {
	p.pinsCalled = true
	return nil
}

type cmdReturningChdirPlugin struct {
	baseMockPlugin
}

type testResultMsg struct{}

func (p *cmdReturningChdirPlugin) HandleChdirChanged(_ ChdirChangedEvent) tea.Cmd {
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

func TestNewEventBus_WhenPluginImplementsChdirHandler_ShouldDiscoverIt(t *testing.T) {
	p := &chdirHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	bus.Dispatch(ChdirChangedEvent{RelPath: "modules/vpc"})

	if !p.called {
		t.Error("ChdirHandler was not called, want called")
	}
}

func TestNewEventBus_WhenPluginImplementsMultipleHandlers_ShouldDiscoverAll(t *testing.T) {
	p := &multiHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	bus.Dispatch(ChdirChangedEvent{RelPath: "modules/vpc"})
	if !p.chdirCalled {
		t.Error("ChdirHandler was not called")
	}

	bus.Dispatch(WorkspaceChangedEvent{Name: "prod"})
	if !p.workspaceCalled {
		t.Error("WorkspaceHandler was not called")
	}

	bus.Dispatch(PinsChangedEvent{Addresses: []string{"aws_instance.web"}})
	if !p.pinsCalled {
		t.Error("PinsHandler was not called")
	}
}

func TestNewEventBus_WhenPluginImplementsNoHandlers_ShouldStillWork(t *testing.T) {
	p := &baseMockPlugin{}
	bus := NewEventBus([]Plugin{p})

	cmd := bus.Dispatch(ChdirChangedEvent{RelPath: "modules/vpc"})
	if cmd != nil {
		t.Error("Dispatch returned non-nil cmd with no handlers")
	}
}

func TestDispatch_WhenChdirChangedEvent_ShouldRouteToChdirHandler(t *testing.T) {
	p := &chdirHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	bus.Dispatch(ChdirChangedEvent{RelPath: "modules/vpc", AbsPath: "/project/modules/vpc", Count: 3})

	if !p.called {
		t.Fatal("HandleChdirChanged was not called")
	}
	if p.event.RelPath != "modules/vpc" {
		t.Errorf("event.RelPath = %q, want %q", p.event.RelPath, "modules/vpc")
	}
	if p.event.AbsPath != "/project/modules/vpc" {
		t.Errorf("event.AbsPath = %q, want %q", p.event.AbsPath, "/project/modules/vpc")
	}
	if p.event.Count != 3 {
		t.Errorf("event.Count = %d, want 3", p.event.Count)
	}
}

func TestDispatch_WhenWorkspaceChangedEvent_ShouldRouteToWorkspaceHandler(t *testing.T) {
	p := &workspaceHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	bus.Dispatch(WorkspaceChangedEvent{Name: "production"})

	if !p.called {
		t.Fatal("HandleWorkspaceChanged was not called")
	}
	if p.event.Name != "production" {
		t.Errorf("event.Name = %q, want %q", p.event.Name, "production")
	}
}

func TestDispatch_WhenPlanCompletedEvent_ShouldRouteToPlanCompletedHandler(t *testing.T) {
	p := &planCompletedHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	summary := &PlanSummary{
		Changes:  []PlanChange{{Resource: Resource{Address: "aws_instance.web"}, Action: ActionCreate}},
		ToCreate: 1,
	}
	bus.Dispatch(PlanCompletedEvent{Summary: summary, ResourceCount: 5, PlanFile: "/tmp/plan.json"})

	if !p.called {
		t.Fatal("HandlePlanCompleted was not called")
	}
	if p.event.Summary != summary {
		t.Error("event.Summary does not match")
	}
	if p.event.ResourceCount != 5 {
		t.Errorf("event.ResourceCount = %d, want 5", p.event.ResourceCount)
	}
	if p.event.PlanFile != "/tmp/plan.json" {
		t.Errorf("event.PlanFile = %q, want %q", p.event.PlanFile, "/tmp/plan.json")
	}
}

func TestDispatch_WhenPinsChangedEvent_ShouldRouteToPinsHandler(t *testing.T) {
	p := &pinsHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	addrs := []string{"aws_instance.web", "aws_s3_bucket.data"}
	bus.Dispatch(PinsChangedEvent{Addresses: addrs})

	if !p.called {
		t.Fatal("HandlePinsChanged was not called")
	}
	if len(p.event.Addresses) != 2 {
		t.Fatalf("event.Addresses length = %d, want 2", len(p.event.Addresses))
	}
	if p.event.Addresses[0] != "aws_instance.web" {
		t.Errorf("event.Addresses[0] = %q, want %q", p.event.Addresses[0], "aws_instance.web")
	}
	if p.event.Addresses[1] != "aws_s3_bucket.data" {
		t.Errorf("event.Addresses[1] = %q, want %q", p.event.Addresses[1], "aws_s3_bucket.data")
	}
}

func TestDispatch_WhenPlanInvalidatedEvent_ShouldRouteToPlanInvalidatedHandler(t *testing.T) {
	p := &planInvalidatedHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	bus.Dispatch(PlanInvalidatedEvent{})

	if !p.called {
		t.Fatal("HandlePlanInvalidated was not called")
	}
}

func TestDispatch_WhenNonEventMessage_ShouldReturnNil(t *testing.T) {
	p := &chdirHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	cmd := bus.Dispatch(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Dispatch(non-event) returned non-nil cmd, want nil")
	}
	if p.called {
		t.Error("handler was called for non-event message")
	}
}

func TestDispatch_WhenMultipleSubscribers_ShouldCallAll(t *testing.T) {
	p1 := &chdirHandlerPlugin{}
	p2 := &chdirHandlerPlugin{}
	bus := NewEventBus([]Plugin{p1, p2})

	bus.Dispatch(ChdirChangedEvent{RelPath: "modules/vpc"})

	if !p1.called {
		t.Error("first subscriber was not called")
	}
	if !p2.called {
		t.Error("second subscriber was not called")
	}
}

func TestDispatch_WhenHandlerReturnsCmd_ShouldReturnBatchedCmd(t *testing.T) {
	p := &cmdReturningChdirPlugin{}
	bus := NewEventBus([]Plugin{p})

	cmd := bus.Dispatch(ChdirChangedEvent{RelPath: "modules/vpc"})
	if cmd == nil {
		t.Fatal("Dispatch returned nil cmd, want non-nil when handler returns cmd")
	}
}

func TestDispatch_WhenAllHandlersReturnNilCmd_ShouldReturnNil(t *testing.T) {
	p1 := &chdirHandlerPlugin{}
	p2 := &chdirHandlerPlugin{}
	bus := NewEventBus([]Plugin{p1, p2})

	cmd := bus.Dispatch(ChdirChangedEvent{RelPath: "modules/vpc"})
	if cmd != nil {
		t.Error("Dispatch returned non-nil cmd, want nil when all handlers return nil")
	}
}

func TestDispatch_WhenMultipleHandlersReturnCmds_ShouldBatchAll(t *testing.T) {
	p1 := &cmdReturningChdirPlugin{}
	p2 := &cmdReturningChdirPlugin{}
	bus := NewEventBus([]Plugin{p1, p2})

	cmd := bus.Dispatch(ChdirChangedEvent{RelPath: "modules/vpc"})
	if cmd == nil {
		t.Fatal("Dispatch returned nil cmd, want batched cmd from multiple handlers")
	}
}

func TestDispatch_WhenHandlerReceivesEvent_ShouldPreserveFieldValues(t *testing.T) {
	p := &chdirHandlerPlugin{}
	bus := NewEventBus([]Plugin{p})

	bus.Dispatch(ChdirChangedEvent{
		RelPath: "modules/ecs",
		AbsPath: "/home/user/infra/modules/ecs",
		Count:   7,
	})

	if p.event.RelPath != "modules/ecs" {
		t.Errorf("RelPath = %q, want %q", p.event.RelPath, "modules/ecs")
	}
	if p.event.AbsPath != "/home/user/infra/modules/ecs" {
		t.Errorf("AbsPath = %q, want %q", p.event.AbsPath, "/home/user/infra/modules/ecs")
	}
	if p.event.Count != 7 {
		t.Errorf("Count = %d, want 7", p.event.Count)
	}
}

func TestDispatch_WhenMixedHandlersAndNonHandlers_ShouldOnlyCallMatchingHandlers(t *testing.T) {
	chdirPlugin := &chdirHandlerPlugin{}
	noHandlerPlugin := &baseMockPlugin{}
	workspacePlugin := &workspaceHandlerPlugin{}
	bus := NewEventBus([]Plugin{chdirPlugin, noHandlerPlugin, workspacePlugin})

	bus.Dispatch(ChdirChangedEvent{RelPath: "modules/vpc"})

	if !chdirPlugin.called {
		t.Error("chdirPlugin was not called")
	}
	if workspacePlugin.called {
		t.Error("workspacePlugin was called for ChdirChangedEvent, want not called")
	}
}
