package extension

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/terraform"
)

// Extension is the interface all features implement.
type Extension interface {
	// Metadata
	Name() string
	Description() string
	KeyBinding() string // single key to activate from home (e.g., "p", "r", "s")

	// Lifecycle
	Init(svc terraform.Service) tea.Cmd
	Update(msg tea.Msg) (Extension, tea.Cmd)
	View(width, height int) string

	// Whether this extension is ready (has data loaded)
	Ready() bool
}
