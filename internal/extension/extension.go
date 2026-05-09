package extension

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/terraform"
)

// Extension is the interface all features implement.
type Extension interface {
	// ID is the unique identifier used as the key in tfui.yaml extensions map.
	ID() string

	// Metadata
	Name() string
	Description() string
	KeyBinding() string // single key to activate from home (e.g., "p", "r", "s")

	// Lifecycle
	Init(ctx *Context) tea.Cmd
	Update(msg tea.Msg) (Extension, tea.Cmd)
	View(width, height int) string

	// Configure applies extension-specific config from tfui.yaml.
	Configure(cfg map[string]interface{}) error

	// Whether this extension is ready (has data loaded)
	Ready() bool
}

// ExtensionFactory creates an extension instance.
type ExtensionFactory func(svc terraform.Service) Extension
