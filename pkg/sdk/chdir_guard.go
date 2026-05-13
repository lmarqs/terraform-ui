package sdk

// ChdirStatus reports the result of a chdir activation check.
type ChdirStatus int

const (
	ChdirUnchanged ChdirStatus = iota
	ChdirChanged
	ChdirRequired
)

// ChdirGuard encapsulates the "check if chdir changed, re-scope service" pattern
// that every data-fetching plugin repeats in its Activate() method.
type ChdirGuard struct {
	session      *Session
	svc          Service
	trackedChdir string
}

func NewChdirGuard(session *Session, svc Service) *ChdirGuard {
	return &ChdirGuard{session: session, svc: svc}
}

// Check reads the current chdir from the session and compares it to the last known chdir.
func (g *ChdirGuard) Check() (ChdirStatus, Service) {
	if g.session == nil {
		return ChdirUnchanged, g.svc
	}

	currentChdir, _ := GetTyped[string](g.session, SessionKeyActiveChdirAbs)
	chdirCount, _ := GetTyped[int](g.session, SessionKeyChdirCount)

	if currentChdir == "" {
		if chdirCount > 1 {
			return ChdirRequired, g.svc
		}
		return ChdirUnchanged, g.svc
	}

	if currentChdir == g.trackedChdir {
		return ChdirUnchanged, g.svc
	}

	g.trackedChdir = currentChdir
	scoped := g.svc.WithDir(currentChdir)
	return ChdirChanged, scoped
}

func (g *ChdirGuard) CurrentChdir() string {
	return g.trackedChdir
}

func (g *ChdirGuard) SetTracked(chdir string) {
	g.trackedChdir = chdir
}
