package plugin

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/terraform"
)

// Plugin is the interface all features implement.
type Plugin interface {
	// ID is the unique identifier used as the key in tfui.yaml plugins map.
	ID() string

	// Metadata
	Name() string
	Description() string
	KeyBinding() string // single key to activate from home (e.g., "p", "r", "s")

	// Lifecycle
	Init(ctx *Context) tea.Cmd
	Update(msg tea.Msg) (Plugin, tea.Cmd)
	View(width, height int) string

	// Configure applies plugin-specific config from tfui.yaml.
	Configure(cfg map[string]interface{}) error

	// Whether this plugin is ready (has data loaded)
	Ready() bool
}

// PluginFactory creates a plugin instance.
type PluginFactory func(svc terraform.Service) Plugin
