package sdk

const sessionKeyPinned = "terraform.pinned"

// PinService provides shared pinning operations for terraform resource addresses.
// Pin state is stored in the Session so all plugins see the same set.
type PinService struct {
	session *Session
}

// NewPinService creates a pin service backed by the given session store.
func NewPinService(session *Session) *PinService {
	return &PinService{session: session}
}

// Toggle adds or removes an address from the pinned set.
// Returns true if the address is now pinned, false if unpinned.
func (p *PinService) Toggle(address string) bool {
	if p.session == nil {
		return false
	}
	pinned := p.load()
	for i, a := range pinned {
		if a == address {
			pinned = append(pinned[:i], pinned[i+1:]...)
			p.save(pinned)
			return false
		}
	}
	pinned = append(pinned, address)
	p.save(pinned)
	return true
}

// IsPinned checks if an address is currently in the pinned set.
func (p *PinService) IsPinned(address string) bool {
	if p.session == nil {
		return false
	}
	for _, a := range p.load() {
		if a == address {
			return true
		}
	}
	return false
}

// All returns a copy of all pinned addresses.
func (p *PinService) All() []string {
	if p.session == nil {
		return nil
	}
	src := p.load()
	out := make([]string, len(src))
	copy(out, src)
	return out
}

// Set replaces the entire pinned set (for bulk operations like tree cascade).
func (p *PinService) Set(addresses []string) {
	if p.session == nil {
		return
	}
	p.save(addresses)
}

// Count returns the number of currently pinned addresses.
func (p *PinService) Count() int {
	if p.session == nil {
		return 0
	}
	return len(p.load())
}

func (p *PinService) load() []string {
	pinned, _ := GetTyped[[]string](p.session, sessionKeyPinned)
	return pinned
}

func (p *PinService) save(pinned []string) {
	p.session.Set(sessionKeyPinned, pinned)
}
