package sdk

import "sync"

// PinService provides shared pinning operations for terraform resource addresses.
// A single instance is shared across all plugins via Context.
type PinService struct {
	mu      sync.RWMutex
	pinned  []string
}

// NewPinService creates an empty pin service.
func NewPinService() *PinService {
	return &PinService{}
}

// Toggle adds or removes an address from the pinned set.
// Returns true if the address is now pinned, false if unpinned.
func (p *PinService) Toggle(address string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, a := range p.pinned {
		if a == address {
			p.pinned = append(p.pinned[:i], p.pinned[i+1:]...)
			return false
		}
	}
	p.pinned = append(p.pinned, address)
	return true
}

// IsPinned checks if an address is currently in the pinned set.
func (p *PinService) IsPinned(address string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, a := range p.pinned {
		if a == address {
			return true
		}
	}
	return false
}

// All returns a copy of all pinned addresses.
func (p *PinService) All() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, len(p.pinned))
	copy(out, p.pinned)
	return out
}

// Set replaces the entire pinned set (for bulk operations like tree cascade).
func (p *PinService) Set(addresses []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pinned = addresses
}

// Count returns the number of currently pinned addresses.
func (p *PinService) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.pinned)
}
