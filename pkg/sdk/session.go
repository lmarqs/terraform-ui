package sdk

import "sync"

// Session holds shared state between plugins within a single TUI session.
// Data is not persisted — it lives only for the lifetime of the process.
type Session struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewSession creates an empty session store.
func NewSession() *Session {
	return &Session{data: make(map[string]interface{})}
}

// Set stores a value under the given key.
func (s *Session) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Get retrieves a value by key. The second return value reports whether the key exists.
func (s *Session) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// GetTyped is a generic helper for type-safe retrieval.
func GetTyped[T any](s *Session, key string) (T, bool) {
	v, ok := s.Get(key)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := v.(T)
	return typed, ok
}
