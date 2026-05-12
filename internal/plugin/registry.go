package plugin

import (
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// PluginMeta holds external routing metadata for a plugin. Plugins themselves
// are invocation-agnostic — keybinding and menu visibility are controlled here.
type PluginMeta struct {
	Keybinding  string
	MenuVisible bool
}

// MenuItem represents a menu entry for display on the home screen.
type MenuItem struct {
	Key         string
	Name        string
	Description string
}

// Registry holds all registered plugin factories and their instantiated plugins.
// It supports lookup by key binding or plugin ID.
type Registry struct {
	plugins   []Plugin
	byKey     map[string]Plugin
	byID      map[string]Plugin
	factories map[string]PluginFactory
	meta      map[string]PluginMeta
	order     []string
}

// NewRegistry creates a new empty Registry with no registered factories or plugins.
func NewRegistry() *Registry {
	return &Registry{
		plugins:   make([]Plugin, 0),
		byKey:     make(map[string]Plugin),
		byID:      make(map[string]Plugin),
		factories: make(map[string]PluginFactory),
		meta:      make(map[string]PluginMeta),
		order:     make([]string, 0),
	}
}

// RegisterFactory registers a plugin factory by its unique ID along with its
// routing metadata. The factory is not invoked until Build is called, allowing
// all factories to be registered before configuration is applied.
func (r *Registry) RegisterFactory(id string, factory PluginFactory, meta PluginMeta) {
	r.factories[id] = factory
	r.meta[id] = meta
	r.order = append(r.order, id)
}

// Build instantiates all enabled plugins from registered factories and applies
// their configuration. Plugins not present in the configs map are enabled by default.
func (r *Registry) Build(svc sdk.Service, configs map[string]config.PluginConfig) {
	for _, id := range r.order {
		factory := r.factories[id]
		cfg, exists := configs[id]

		// If not in config, default to enabled
		if !exists || cfg.IsEnabled() {
			plg := factory(svc)
			if exists && cfg.Options != nil {
				_ = plg.Configure(cfg.Options)
			}

			r.plugins = append(r.plugins, plg)
			if key := r.meta[id].Keybinding; key != "" {
				r.byKey[key] = plg
			}
			r.byID[id] = plg
		}
	}
}

// All returns the slice of all enabled and instantiated plugins.
func (r *Registry) All() []Plugin {
	return r.plugins
}

// ByKey looks up a plugin by its key binding string and reports whether it was found.
func (r *Registry) ByKey(key string) (Plugin, bool) {
	plg, ok := r.byKey[key]
	return plg, ok
}

// ByID looks up a plugin by its unique identifier and reports whether it was found.
func (r *Registry) ByID(id string) (Plugin, bool) {
	plg, ok := r.byID[id]
	return plg, ok
}

// MenuItems returns the list of menu entries for plugins that are visible in the
// home menu, in registration order.
func (r *Registry) MenuItems() []MenuItem {
	var items []MenuItem
	for _, id := range r.order {
		m := r.meta[id]
		if !m.MenuVisible {
			continue
		}
		plg, ok := r.byID[id]
		if !ok {
			continue
		}
		items = append(items, MenuItem{
			Key:         m.Keybinding,
			Name:        plg.Name(),
			Description: plg.Description(),
		})
	}
	return items
}
