package sdk

import "testing"

func TestScopeGuard_Check(t *testing.T) {
	tests := []struct {
		name          string
		sessionScope  string
		sessionCount  int
		wantStatus    ScopeStatus
		wantScopedDir string
		desc          string
	}{
		{
			name:          "first activation with scope set",
			sessionScope:  "/project/modules/prod",
			sessionCount:  2,
			wantStatus:    ScopeChanged,
			wantScopedDir: "/project/modules/prod",
			desc:          "should return ScopeChanged with the scoped service",
		},
		{
			name:          "no scope in single-scope project",
			sessionScope:  "",
			sessionCount:  1,
			wantStatus:    ScopeUnchanged,
			wantScopedDir: "",
			desc:          "single-scope project without active scope is fine",
		},
		{
			name:          "multi-scope with no scope selected",
			sessionScope:  "",
			sessionCount:  3,
			wantStatus:    ScopeRequired,
			wantScopedDir: "",
			desc:          "should require scope selection in multi-scope env",
		},
		{
			name:          "scope set but only one scope exists",
			sessionScope:  "/project",
			sessionCount:  1,
			wantStatus:    ScopeChanged,
			wantScopedDir: "/project",
			desc:          "even with one scope, if active_abs is set, use it",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewSession()
			if tt.sessionScope != "" {
				session.Set(SessionKeyActiveScopeAbs, tt.sessionScope)
			}
			session.Set(SessionKeyScopeCount, tt.sessionCount)

			svc := &scopeGuardMockService{dir: "/original"}
			guard := NewScopeGuard(session, svc)

			status, scoped := guard.Check()

			if status != tt.wantStatus {
				t.Errorf("Check() status = %v, want %v", status, tt.wantStatus)
			}

			if tt.wantScopedDir != "" {
				mock, ok := scoped.(*scopeGuardMockService)
				if !ok {
					t.Fatal("expected scoped service to be *scopeGuardMockService")
				}
				if mock.dir != tt.wantScopedDir {
					t.Errorf("scoped service dir = %q, want %q", mock.dir, tt.wantScopedDir)
				}
			}
		})
	}
}

func TestScopeGuard_Check_DetectsChange(t *testing.T) {
	session := NewSession()
	session.Set(SessionKeyActiveScopeAbs, "/project/modules/prod")
	session.Set(SessionKeyScopeCount, 2)

	svc := &scopeGuardMockService{dir: "/original"}
	guard := NewScopeGuard(session, svc)

	// First check: scope is new → ScopeChanged
	status, _ := guard.Check()
	if status != ScopeChanged {
		t.Fatalf("first Check() = %v, want ScopeChanged", status)
	}

	// Second check: same scope → ScopeUnchanged
	status, _ = guard.Check()
	if status != ScopeUnchanged {
		t.Fatalf("second Check() = %v, want ScopeUnchanged", status)
	}

	// Change scope in session
	session.Set(SessionKeyActiveScopeAbs, "/project/modules/staging")

	// Third check: scope changed → ScopeChanged
	status, scoped := guard.Check()
	if status != ScopeChanged {
		t.Fatalf("third Check() = %v, want ScopeChanged", status)
	}
	mock := scoped.(*scopeGuardMockService)
	if mock.dir != "/project/modules/staging" {
		t.Errorf("scoped dir = %q, want %q", mock.dir, "/project/modules/staging")
	}
}

func TestScopeGuard_CurrentScope(t *testing.T) {
	session := NewSession()
	session.Set(SessionKeyActiveScopeAbs, "/project/modules/prod")
	session.Set(SessionKeyScopeCount, 2)

	svc := &scopeGuardMockService{dir: "/original"}
	guard := NewScopeGuard(session, svc)

	// Before first check, no scope tracked
	if got := guard.CurrentScope(); got != "" {
		t.Errorf("before Check(), CurrentScope() = %q, want empty", got)
	}

	guard.Check()

	if got := guard.CurrentScope(); got != "/project/modules/prod" {
		t.Errorf("after Check(), CurrentScope() = %q, want %q", got, "/project/modules/prod")
	}
}

func TestScopeGuard_NilSession(t *testing.T) {
	svc := &scopeGuardMockService{dir: "/original"}
	guard := NewScopeGuard(nil, svc)

	status, scoped := guard.Check()
	if status != ScopeUnchanged {
		t.Errorf("nil session: Check() = %v, want ScopeUnchanged", status)
	}
	if scoped != svc {
		t.Error("nil session: scoped service should be the original service")
	}
}

// scopeGuardMockService is a minimal mock that only implements WithDir.
type scopeGuardMockService struct {
	Service
	dir string
}

func (m *scopeGuardMockService) WithDir(dir string) Service {
	return &scopeGuardMockService{dir: dir}
}
