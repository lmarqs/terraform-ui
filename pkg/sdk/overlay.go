package sdk

import tea "github.com/charmbracelet/bubbletea"

// Overlay is a modal UI element that renders on top of the current view.
// While active, it captures all input. The App renders it centered over
// the existing content.
type Overlay interface {
	ID() string
	Open() tea.Cmd
	Update(msg tea.Msg) (Overlay, tea.Cmd)
	View(width, height int) string
	Hints() []KeyHint
}

// OverlayDismissMsg signals that the active overlay should close.
type OverlayDismissMsg struct{}
