package sdk

import tea "github.com/charmbracelet/bubbletea"

// KeyHint describes a single keybinding shown in the hint bar.
type KeyHint struct {
	Key         string
	Description string
}

// Frame is a composable view layer that lives in a navigation stack.
// Input is always routed to the topmost frame. Each frame renders its
// own view and declares which key hints to show.
type Frame interface {
	// ID returns a short identifier for debugging/logging.
	ID() string

	// Update processes a message. Returns nil to signal pop (back navigation).
	Update(msg tea.Msg) (Frame, tea.Cmd)

	// View renders this frame's content within the given dimensions.
	View(width, height int) string

	// Hints returns the key hints to display while this frame is active.
	Hints() []KeyHint
}

// Stackable is an optional interface plugins implement to use
// frame-based navigation. The app routes key input through the plugin's
// stack instead of calling Update directly.
type Stackable interface {
	Stack() *Stack
}

// FramePushMsg is returned as a tea.Cmd to request pushing a new frame.
type FramePushMsg struct {
	Frame Frame
}
