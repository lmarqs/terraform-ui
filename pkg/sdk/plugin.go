package sdk

import tea "github.com/charmbracelet/bubbletea"

// Plugin is the interface that all tfui features must implement to participate
// in the plugin system. Each plugin provides a focused view accessible via a
// single key press from the home screen.
type Plugin interface {
	// ID returns the unique identifier used as the key in the tfui.yaml plugins map.
	ID() string

	// Name returns the human-readable display name shown in the status bar and home menu.
	Name() string

	// Description returns a one-line summary of the plugin's purpose.
	Description() string

	// Init initializes the plugin with shared context and returns an optional startup command.
	Init(ctx *Context) tea.Cmd

	// Update processes a bubbletea message and returns the updated plugin and an optional command.
	Update(msg tea.Msg) (Plugin, tea.Cmd)

	// View renders the plugin's UI within the given width and height constraints.
	View(width, height int) string

	// Configure applies plugin-specific options from the tfui.yaml configuration file.
	Configure(cfg map[string]interface{}) error

	// Ready reports whether the plugin has loaded its data and is ready to display.
	Ready() bool
}

// Activatable is an optional interface plugins can implement to perform work
// when the user navigates to them (e.g., trigger a plan on first visit).
type Activatable interface {
	Activate() tea.Cmd
}

// Countable is an optional interface plugins implement to report item counts
// for display in the content border title (e.g., "State Browser (30/1549)").
type Countable interface {
	Count() (filtered int, total int)
}

// Pinnable is an optional interface plugins implement to report pinned item count
// for display in the content border title.
type Pinnable interface {
	PinnedCount() int
}

// Hintable is an optional interface plugins implement to supply
// context-sensitive key hints for the status bar without needing a full Stack.
type Hintable interface {
	Hints() []KeyHint
}

// Busy is an optional interface plugins implement to report when they have a
// critical operation in progress that holds a terraform state lock. Used by :q
// to guard against accidental quit (which would kill terraform and leave a stale lock).
type Busy interface {
	Busy() bool
}

// DeactivateMsg is returned by a plugin's Update to signal the app should
// deactivate it and return to the home screen.
type DeactivateMsg struct{}

// NavigateMsg is returned by a plugin to request the app navigate to another
// plugin by ID. The app applies the target plugin's NavBehavior (push/replace).
type NavigateMsg struct {
	PluginID string
}

// KeyCapturer is an optional interface plugins implement to signal they need
// exclusive keyboard input (e.g., terraform console REPL). When CapturesKeys()
// returns true, the app routes all keys to the plugin except ctrl+c (quit).
type KeyCapturer interface {
	CapturesKeys() bool
}

// PluginFactory is a constructor function that creates a new plugin instance
// bound to the given terraform service.
type PluginFactory func(svc Service) Plugin
