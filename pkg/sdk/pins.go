package sdk

// Pins is the set of resource addresses pinned by the user within a Context.
// It owns counting, presence checks, and the immutable toggle semantics that
// keep Context snapshots safe to share. Pins are scoped to a Context — they
// die on chdir/workspace switch.
type Pins []string

// Count returns the number of pinned addresses.
func (p Pins) Count() int { return len(p) }

// HasAny reports whether at least one address is pinned.
func (p Pins) HasAny() bool { return len(p) > 0 }

// Contains reports whether the address is currently pinned. Linear scan;
// pin sets are small (single-digit typical).
func (p Pins) Contains(address string) bool {
	for _, a := range p {
		if a == address {
			return true
		}
	}
	return false
}

// Toggle returns a fresh Pins with address added if absent, or removed if
// present. The receiver is never mutated.
func (p Pins) Toggle(address string) Pins {
	for i, a := range p {
		if a == address {
			out := make(Pins, 0, len(p)-1)
			out = append(out, p[:i]...)
			out = append(out, p[i+1:]...)
			return out
		}
	}
	out := make(Pins, 0, len(p)+1)
	out = append(out, p...)
	out = append(out, address)
	return out
}

// Clone returns a defensive copy. nil input yields nil output.
func (p Pins) Clone() Pins {
	if p == nil {
		return nil
	}
	return append(Pins(nil), p...)
}
