package sdk

import (
	"io"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
)

// PluginBase is an embeddable struct that absorbs the dependency-injection
// boilerplate every plugin used to repeat: storing the Service / Logger / pin
// callbacks, surfacing ID / Name / Description, and rebinding the scoped
// Service when the context changes.
//
// Plugins compose it like so:
//
//	type Plugin struct {
//	    sdk.PluginBase
//	    // plugin-specific fields…
//	}
//
//	func New(svc sdk.Service) sdk.Plugin {
//	    return &Plugin{PluginBase: sdk.NewPluginBase("plan", "Plan", "Review terraform plan changes")}
//	}
//
//	func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
//	    e.InitBase(deps)
//	    e.reset()
//	    return nil
//	}
type PluginBase struct {
	id, name, description string

	// Svc is the chdir-scoped Service — rebound on every ContextChangedEvent
	// by HandleContextChangedDefault. Plugins that have not yet observed an
	// event fall back to the unscoped Service captured at Init.
	Svc Service
	// Log is the structured logger. Always non-nil after NewPluginBase.
	Log *slog.Logger
	// GetCtx returns the live immutable Context snapshot. Set by InitBase.
	GetCtx func() *Context
	// PinFn toggles a single resource address. Set by InitBase.
	PinFn func(string) tea.Cmd
	// ClearPinsFn removes every pin. Set by InitBase.
	ClearPinsFn func() tea.Cmd
}

// NewPluginBase builds a base with metadata fields populated and a discard
// logger pre-seeded so plugins remain safe to use before Init is called.
func NewPluginBase(id, name, description string) PluginBase {
	return PluginBase{
		id:          id,
		name:        name,
		description: description,
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// ID returns the plugin's stable identifier.
func (b *PluginBase) ID() string { return b.id }

// Name returns the human-readable display name.
func (b *PluginBase) Name() string { return b.name }

// Description returns the one-line summary of the plugin's purpose.
func (b *PluginBase) Description() string { return b.description }

// InitBase wires deps onto the base. Embedders call this from their own
// Init(deps *PluginDeps) tea.Cmd before any plugin-specific setup.
func (b *PluginBase) InitBase(deps *PluginDeps) {
	b.Svc = deps.Service
	if deps.Logger != nil {
		b.Log = deps.Logger
	}
	b.GetCtx = deps.Context
	b.PinFn = deps.Pin
	b.ClearPinsFn = deps.ClearPins
}

// HandleContextChangedDefault implements the standard service-rebind that
// almost every plugin needs. Returns true if the embedder should proceed with
// its own reset / refresh; false if the event is a no-op (Next == nil).
//
// Pins-unaware plugins (console, output, validate, …) call this directly and
// full-reset on true. Plugins that distinguish a pin toggle from a switch
// (plan, apply, state) go through ReactToContext, which calls this on the
// switch path.
func (b *PluginBase) HandleContextChangedDefault(ev ContextChangedEvent) bool {
	if ev.Next == nil {
		return false
	}
	if ev.Next.Service != nil {
		b.Svc = ev.Next.Service
	}
	return true
}

// ReactToContext is the canonical shape for reacting to a Context replacement,
// shared by the plugins that distinguish a pin toggle from a chdir/workspace
// switch (plan, apply, state). It:
//   - no-ops when Next is nil;
//   - on a pins-only change (OnlyPinsChanged), runs onPins and preserves state —
//     the service is unchanged, so it is not rebound;
//   - otherwise rebinds the service (HandleContextChangedDefault) and runs
//     onReset for the full reset.
//
// onPins may be nil (e.g. apply has nothing to re-sync on a pin change); onReset
// is always required. Plugins that only ever full-reset should keep calling
// HandleContextChangedDefault directly rather than pass a trivial onPins.
func (b *PluginBase) ReactToContext(ev ContextChangedEvent, onPins, onReset func()) tea.Cmd {
	if ev.Next == nil {
		return nil
	}
	if ev.OnlyPinsChanged() {
		if onPins != nil {
			onPins()
		}
		return nil
	}
	b.HandleContextChangedDefault(ev)
	onReset()
	return nil
}

// PinnedAddresses returns the addresses pinned in the current Context.
// Returns nil when GetCtx is unset (e.g., before Init), so it is always safe
// to call.
func (b *PluginBase) PinnedAddresses() []string {
	if b.GetCtx == nil {
		return nil
	}
	return []string(b.GetCtx().Pins)
}

// PinnedCount satisfies sdk.Pinnable. Returns 0 when GetCtx is unset
// (e.g., before Init).
func (b *PluginBase) PinnedCount() int {
	if b.GetCtx == nil {
		return 0
	}
	return b.GetCtx().Pins.Count()
}

// HasPins reports whether the active Context has at least one pinned address.
func (b *PluginBase) HasPins() bool {
	if b.GetCtx == nil {
		return false
	}
	return b.GetCtx().Pins.HasAny()
}

// IsPinned reports whether the supplied address is currently pinned.
func (b *PluginBase) IsPinned(address string) bool {
	if b.GetCtx == nil {
		return false
	}
	return b.GetCtx().Pins.Contains(address)
}
