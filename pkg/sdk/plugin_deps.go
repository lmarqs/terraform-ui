package sdk

import (
	tea "github.com/charmbracelet/bubbletea"

	"log/slog"
)

// PluginDeps is the dependency-injection container handed to each plugin at
// Init. It exposes process-lifetime collaborators (Service, Logger) plus two
// live channels into the App's immutable terraform Context: a getter and a
// pin toggle.
//
// Rule of thumb: if a value affects what terraform sees (targets, var-files,
// parallelism, …) it lives on Context — read it via Context() at the top of
// every terraform-affecting operation. PluginDeps fields are stable for the
// life of the plugin; Context is replaced atomically by the App on
// chdir/workspace/pin changes (ADR-0018).
type PluginDeps struct {
	// Logger is the structured logger for debug output. Plugins should use
	// this instead of the global slog to enable testability.
	Logger *slog.Logger
	// Service is the unscoped terraform service handle. Plugins that run
	// terraform should use Context().Service which is scoped to the active
	// chdir; Service is exposed only as a fallback for plugins that have
	// not yet observed a ContextChangedEvent.
	Service Service
	// Context returns the active immutable Context snapshot. Always returns
	// non-nil after the first ContextChangedEvent; before that, returns a
	// minimal bootstrap snapshot derived from Service.
	Context func() *Context
	// Pin toggles a single resource address into or out of the active
	// Context's Pins. Returns a tea.Cmd that emits the request to the
	// App; the actual Pins change becomes visible to the plugin on the
	// next ContextChangedEvent. Plugins must NEVER mutate Pins directly.
	Pin func(address string) tea.Cmd
	// ClearPins removes every pin from the active Context.
	// Returns a tea.Cmd with the same semantics as Pin.
	ClearPins func() tea.Cmd
}
