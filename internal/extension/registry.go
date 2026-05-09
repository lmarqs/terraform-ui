package extension

import (
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
)

// Registry holds all registered extensions.
type Registry struct {
	extensions []Extension
	byKey      map[string]Extension
	byID       map[string]Extension
	factories  map[string]ExtensionFactory
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		extensions: make([]Extension, 0),
		byKey:      make(map[string]Extension),
		byID:       make(map[string]Extension),
		factories:  make(map[string]ExtensionFactory),
	}
}

// RegisterFactory registers an extension factory by ID.
// Extensions are instantiated later when config is loaded.
func (r *Registry) RegisterFactory(id string, factory ExtensionFactory) {
	r.factories[id] = factory
}

// Build creates all extensions from registered factories, applying config.
// Extensions not in the config map are enabled by default.
func (r *Registry) Build(svc terraform.Service, configs map[string]config.ExtensionConfig) {
	for id, factory := range r.factories {
		cfg, exists := configs[id]

		// If not in config, default to enabled
		if !exists || cfg.IsEnabled() {
			ext := factory(svc)
			if exists && cfg.Options != nil {
				ext.Configure(cfg.Options)
			}

			r.extensions = append(r.extensions, ext)
			if key := ext.KeyBinding(); key != "" {
				r.byKey[key] = ext
			}
			r.byID[id] = ext
		}
	}
}

// All returns all enabled extensions.
func (r *Registry) All() []Extension {
	return r.extensions
}

// ByKey looks up an extension by its key binding.
func (r *Registry) ByKey(key string) (Extension, bool) {
	ext, ok := r.byKey[key]
	return ext, ok
}

// ByID looks up an extension by its ID.
func (r *Registry) ByID(id string) (Extension, bool) {
	ext, ok := r.byID[id]
	return ext, ok
}
