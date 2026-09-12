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

// Contains reports whether the address is currently pinned. Linear scan, for a
// single question. Callers asking about many addresses in one pass should take
// a Lookup instead.
func (p Pins) Contains(address string) bool {
	for _, a := range p {
		if a == address {
			return true
		}
	}
	return false
}

// Lookup returns a set-backed membership predicate. One gesture on a module row
// covers every resource beneath it, so a pin set is no longer single-digit and
// rendering asks about every descendant leaf of every visible row — a linear
// scan per question would make a keypress quadratic in the state's size.
func (p Pins) Lookup() func(address string) bool {
	set := p.set()
	return func(address string) bool {
		_, ok := set[address]
		return ok
	}
}

func (p Pins) set() map[string]struct{} {
	set := make(map[string]struct{}, len(p))
	for _, a := range p {
		set[a] = struct{}{}
	}
	return set
}

// Toggle returns a fresh Pins with set semantics over the supplied group: when
// every address is already pinned the whole group is removed, otherwise the
// missing ones are appended. A single address therefore flips, and a group
// reached from one gesture — pinning a module row that covers many resources —
// completes before it clears, so a partly-pinned group never inverts.
// The receiver is never mutated.
func (p Pins) Toggle(addresses ...string) Pins {
	pinned := p.set()
	group := make(map[string]struct{}, len(addresses))
	remove := true
	for _, a := range addresses {
		group[a] = struct{}{}
		if _, ok := pinned[a]; !ok {
			remove = false
		}
	}

	if remove {
		out := make(Pins, 0, len(p))
		for _, a := range p {
			if _, dropped := group[a]; !dropped {
				out = append(out, a)
			}
		}
		return out
	}

	out := make(Pins, 0, len(p)+len(addresses))
	out = append(out, p...)
	for _, a := range addresses {
		if _, ok := pinned[a]; ok {
			continue
		}
		pinned[a] = struct{}{}
		out = append(out, a)
	}
	return out
}

// Clone returns a defensive copy. nil input yields nil output.
func (p Pins) Clone() Pins {
	if p == nil {
		return nil
	}
	return append(Pins(nil), p...)
}
