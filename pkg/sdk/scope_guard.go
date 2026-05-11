package sdk

// ScopeStatus reports the result of a scope activation check.
type ScopeStatus int

const (
	// ScopeUnchanged means the scope has not changed since the last check.
	ScopeUnchanged ScopeStatus = iota
	// ScopeChanged means the scope changed — the plugin should reset its state.
	ScopeChanged
	// ScopeRequired means multiple scopes exist but none is selected.
	ScopeRequired
)

// ScopeGuard encapsulates the "check if scope changed, re-scope service" pattern
// that every data-fetching plugin repeats in its Activate() method.
type ScopeGuard struct {
	session      *Session
	svc          Service
	trackedScope string
}

// NewScopeGuard creates a guard that tracks scope changes via the session.
func NewScopeGuard(session *Session, svc Service) *ScopeGuard {
	return &ScopeGuard{session: session, svc: svc}
}

// Check reads the current scope from the session and compares it to the last known scope.
// Returns:
//   - ScopeChanged + re-scoped service: when the scope is new or different
//   - ScopeRequired + original service: when multi-scope env has no selection
//   - ScopeUnchanged + original service: when nothing changed
func (g *ScopeGuard) Check() (ScopeStatus, Service) {
	if g.session == nil {
		return ScopeUnchanged, g.svc
	}

	currentScope, _ := GetTyped[string](g.session, SessionKeyActiveScopeAbs)
	scopeCount, _ := GetTyped[int](g.session, SessionKeyScopeCount)

	if currentScope == "" {
		if scopeCount > 1 {
			return ScopeRequired, g.svc
		}
		return ScopeUnchanged, g.svc
	}

	if currentScope == g.trackedScope {
		return ScopeUnchanged, g.svc
	}

	g.trackedScope = currentScope
	scoped := g.svc.WithDir(currentScope)
	return ScopeChanged, scoped
}

// CurrentScope returns the last tracked scope path, or empty if no scope has been checked yet.
func (g *ScopeGuard) CurrentScope() string {
	return g.trackedScope
}

// SetTracked pre-seeds the guard with a known scope, so the next Check()
// won't report ScopeChanged for an already-active scope.
func (g *ScopeGuard) SetTracked(scope string) {
	g.trackedScope = scope
}
