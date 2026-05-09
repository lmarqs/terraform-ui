package plugin

import (
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
)

// Registry holds all registered plugin factories and their instantiated plugins.
// It supports lookup by key binding or plugin ID.
type Registry struct {
	plugins   []Plugin
	byKey     map[string]Plugin
	byID      map[string]Plugin
	factories map[string]PluginFactory
}

// NewRegistry creates a new empty Registry with no registered factories or plugins.
func NewRegistry() *Registry {
	return &Registry{
		plugins:   make([]Plugin, 0),
		byKey:     make(map[string]Plugin),
		byID:      make(map[string]Plugin),
		factories: make(map[string]PluginFactory),
	}
}

// RegisterFactory registers a plugin factory by its unique ID. The factory is
// not invoked until Build is called, allowing all factories to be registered
// before configuration is applied.
func (r *Registry) RegisterFactory(id string, factory PluginFactory) {
	r.factories[id] = factory
}

// Build instantiates all enabled plugins from registered factories and applies
// their configuration. Plugins not present in the configs map are enabled by default.
func (r *Registry) Build(svc terraform.Service, configs map[string]config.PluginConfig) {
	for id, factory := range r.factories {
		cfg, exists := configs[id]

		// If not in config, default to enabled
		if !exists || cfg.IsEnabled() {
			plg := factory(svc)
			if exists && cfg.Options != nil {
				plg.Configure(cfg.Options)
			}

			r.plugins = append(r.plugins, plg)
			if key := plg.KeyBinding(); key != "" {
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
