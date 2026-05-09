package extension

// Registry holds all registered extensions.
type Registry struct {
	extensions []Extension
	byKey      map[string]Extension
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		extensions: make([]Extension, 0),
		byKey:      make(map[string]Extension),
	}
}

// Register adds an extension to the registry.
func (r *Registry) Register(ext Extension) {
	r.extensions = append(r.extensions, ext)
	if key := ext.KeyBinding(); key != "" {
		r.byKey[key] = ext
	}
}

// All returns all registered extensions.
func (r *Registry) All() []Extension {
	return r.extensions
}

// ByKey looks up an extension by its key binding.
func (r *Registry) ByKey(key string) (Extension, bool) {
	ext, ok := r.byKey[key]
	return ext, ok
}
