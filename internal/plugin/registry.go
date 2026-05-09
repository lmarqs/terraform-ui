package plugin

import (
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
)

// Registry holds all registered plugins.
type Registry struct {
	plugins   []Plugin
	byKey     map[string]Plugin
	byID      map[string]Plugin
	factories map[string]PluginFactory
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins:   make([]Plugin, 0),
		byKey:     make(map[string]Plugin),
		byID:      make(map[string]Plugin),
		factories: make(map[string]PluginFactory),
	}
}

// RegisterFactory registers a plugin factory by ID.
// Plugins are instantiated later when config is loaded.
func (r *Registry) RegisterFactory(id string, factory PluginFactory) {
	r.factories[id] = factory
}

// Build creates all plugins from registered factories, applying config.
// Plugins not in the config map are enabled by default.
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

// All returns all enabled plugins.
func (r *Registry) All() []Plugin {
	return r.plugins
}

// ByKey looks up a plugin by its key binding.
func (r *Registry) ByKey(key string) (Plugin, bool) {
	plg, ok := r.byKey[key]
	return plg, ok
}

// ByID looks up a plugin by its ID.
func (r *Registry) ByID(id string) (Plugin, bool) {
	plg, ok := r.byID[id]
	return plg, ok
}
