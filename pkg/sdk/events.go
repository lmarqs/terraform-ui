package sdk

import tea "github.com/charmbracelet/bubbletea"

type Event interface {
	tea.Msg
	event()
}

// ContextSwitchRequestMsg is emitted by the chdir / workspace plugins to
// request the App rebuild and replace the immutable Context. Both fields are
// required — emitters read the current Context to populate the unchanged field.
//
// The App owns path resolution: Chdir is the relative member path, joined with
// the project root to produce the absolute path for terraform.
//
// Plugins do NOT subscribe to this message — it is a request to the App.
// Subscribe to ContextChangedEvent for notifications about Context updates.
type ContextSwitchRequestMsg struct {
	Chdir     Chdir     // relative member path (required)
	Workspace Workspace // workspace name (required)
}

// PinToggleRequestMsg asks the App to toggle a single pinned address on the
// active Context (added if absent, removed if present). Plugins emit this via
// PluginDeps.Pin; the App responds by rebuilding Context.WithPins and
// dispatching a ContextChangedEvent.
type PinToggleRequestMsg struct {
	Address string
}

// PinClearRequestMsg asks the App to remove every pin from the active Context.
// Plugins emit this via PluginDeps.ClearPins.
type PinClearRequestMsg struct{}

// ContextChangedEvent is dispatched by the app whenever the immutable Context
// is replaced (chdir change, workspace change, pin toggle). Plugins should
// implement ContextChangedHandler and perform a full reset of any state
// derived from the previous Context.
type ContextChangedEvent struct {
	Prev *Context
	Next *Context
}

func (ContextChangedEvent) event() {}

// OnlyPinsChanged reports whether the only difference between Prev and
// Next is the Pins slice. Plugins can use this to skip full UI resets on
// pure pin toggles. Returns false when Prev is nil (initial Context build).
func (e ContextChangedEvent) OnlyPinsChanged() bool {
	if e.Prev == nil || e.Next == nil {
		return false
	}
	if e.Prev.WorkingDir != e.Next.WorkingDir {
		return false
	}
	if e.Prev.Workspace != e.Next.Workspace {
		return false
	}
	return true
}

// ContextChangedHandler is implemented by plugins that need to react to the
// app replacing its immutable Context.
type ContextChangedHandler interface {
	HandleContextChanged(ContextChangedEvent) tea.Cmd
}

type PlanCompletedEvent struct {
	Summary       *PlanSummary
	ResourceCount int
	PlanFile      string
}

func (PlanCompletedEvent) event() {}

type PlanInvalidatedEvent struct{}

func (PlanInvalidatedEvent) event() {}

type LockDetectedEvent struct {
	Lock *StateLock
}

func (LockDetectedEvent) event() {}

type LockClearedEvent struct{}

func (LockClearedEvent) event() {}

type StateRefreshedEvent struct{}

func (StateRefreshedEvent) event() {}

type PlanCompletedHandler interface {
	HandlePlanCompleted(PlanCompletedEvent) tea.Cmd
}

type PlanInvalidatedHandler interface {
	HandlePlanInvalidated(PlanInvalidatedEvent) tea.Cmd
}

type LockDetectedHandler interface {
	HandleLockDetected(LockDetectedEvent) tea.Cmd
}

type LockClearedHandler interface {
	HandleLockCleared(LockClearedEvent) tea.Cmd
}

type StateRefreshedHandler interface {
	HandleStateRefreshed(StateRefreshedEvent) tea.Cmd
}
