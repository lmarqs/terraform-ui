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

// Toggle returns a fresh Pins with set semantics over the supplied group: when
// every address is already pinned the whole group is removed, otherwise the
// missing ones are appended. A single address therefore flips, and a group
// reached from one gesture — pinning a module row that covers many resources —
// completes before it clears, so a partly-pinned group never inverts.
// The receiver is never mutated.
func (p Pins) Toggle(addresses ...string) Pins {
	remove := true
	for _, a := range addresses {
		if !p.Contains(a) {
			remove = false
			break
		}
	}

	if remove {
		out := make(Pins, 0, len(p))
		for _, a := range p {
			if !containsAddress(addresses, a) {
				out = append(out, a)
			}
		}
		return out
	}

	out := make(Pins, 0, len(p)+len(addresses))
	out = append(out, p...)
	for _, a := range addresses {
		if !out.Contains(a) {
			out = append(out, a)
		}
	}
	return out
}

func containsAddress(addresses []string, address string) bool {
	for _, a := range addresses {
		if a == address {
			return true
		}
	}
	return false
}

// Clone returns a defensive copy. nil input yields nil output.
func (p Pins) Clone() Pins {
	if p == nil {
		return nil
	}
	return append(Pins(nil), p...)
}
