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

// ContextChangeReason records why the app replaced the Context, so plugins
// react to the app's intent instead of inferring it from a Prev/Next diff. The
// zero value is ContextSwitched — the conservative default that triggers a full
// reset, so any event built without an explicit reason is safe.
type ContextChangeReason int

const (
	// ContextSwitched marks a chdir or workspace change (or the initial build):
	// plugins fully reset state derived from the previous Context.
	ContextSwitched ContextChangeReason = iota
	// ContextPinsChanged marks a pin toggle/clear: working dir and workspace are
	// unchanged, so plugins preserve UI and only re-sync the pin set.
	ContextPinsChanged
)

// ContextChangedEvent is dispatched by the app whenever the immutable Context
// is replaced (chdir change, workspace change, pin toggle). Reason carries the
// app's intent; plugins implement ContextChangedHandler and, unless the reason
// is ContextPinsChanged, perform a full reset of any derived state.
type ContextChangedEvent struct {
	Prev   *Context
	Next   *Context
	Reason ContextChangeReason
}

func (ContextChangedEvent) event() {}

// OnlyPinsChanged reports whether this change was a pure pin toggle/clear rather
// than a chdir/workspace switch. It reads the app-supplied Reason — NOT a
// Prev/Next diff — so a same-chdir re-selection (which also nulls pins) is
// correctly treated as a switch and drives a full reset, not a pin-only update.
func (e ContextChangedEvent) OnlyPinsChanged() bool {
	return e.Reason == ContextPinsChanged
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
