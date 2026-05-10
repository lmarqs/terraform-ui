package sdk

import tea "github.com/charmbracelet/bubbletea"

// KeyHint describes a single keybinding shown in the hint bar.
type KeyHint struct {
	Key         string
	Description string
}

// Common hints reusable across plugins.
var (
	HintNavigate = KeyHint{Key: "↑↓", Description: "navigate"}
	HintScroll   = KeyHint{Key: "↑↓", Description: "scroll"}
	HintPan      = KeyHint{Key: "←→", Description: "pan"}
	HintBack     = KeyHint{Key: "q", Description: "back"}
	HintRefresh  = KeyHint{Key: "r", Description: "refresh"}
	HintRetry    = KeyHint{Key: "r", Description: "retry"}
	HintFilter   = KeyHint{Key: "/", Description: "filter"}
	HintPin      = KeyHint{Key: "Space", Description: "pin"}
	HintDelete   = KeyHint{Key: "d", Description: "delete"}
	HintEdit     = KeyHint{Key: "e", Description: "edit"}
	HintInspect  = KeyHint{Key: "Enter", Description: "inspect"}
	HintSelect   = KeyHint{Key: "Enter", Description: "select"}
	HintConfirm  = KeyHint{Key: "Enter", Description: "confirm"}
	HintCancel   = KeyHint{Key: "Esc", Description: "cancel"}
)

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
