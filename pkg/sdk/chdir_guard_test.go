package sdk

import "testing"

func TestChdirGuard_Check(t *testing.T) {
	tests := []struct {
		name          string
		sessionScope  string
		sessionCount  int
		wantStatus    ChdirStatus
		wantScopedDir string
		desc          string
	}{
		{
			name:          "first activation with scope set",
			sessionScope:  "/project/modules/prod",
			sessionCount:  2,
			wantStatus:    ChdirChanged,
			wantScopedDir: "/project/modules/prod",
			desc:          "should return ChdirChanged with the scoped service",
		},
		{
			name:          "no scope in single-scope project",
			sessionScope:  "",
			sessionCount:  1,
			wantStatus:    ChdirUnchanged,
			wantScopedDir: "",
			desc:          "single-scope project without active scope is fine",
		},
		{
			name:          "multi-scope with no scope selected",
			sessionScope:  "",
			sessionCount:  3,
			wantStatus:    ChdirRequired,
			wantScopedDir: "",
			desc:          "should require scope selection in multi-scope env",
		},
		{
			name:          "scope set but only one scope exists",
			sessionScope:  "/project",
			sessionCount:  1,
			wantStatus:    ChdirChanged,
			wantScopedDir: "/project",
			desc:          "even with one scope, if active_abs is set, use it",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewSession()
			if tt.sessionScope != "" {
				session.Set(SessionKeyActiveChdirAbs, tt.sessionScope)
			}
			session.Set(SessionKeyChdirCount, tt.sessionCount)

			svc := &scopeGuardMockService{dir: "/original"}
			guard := NewChdirGuard(session, svc)

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

func TestChdirGuard_Check_DetectsChange(t *testing.T) {
	session := NewSession()
	session.Set(SessionKeyActiveChdirAbs, "/project/modules/prod")
	session.Set(SessionKeyChdirCount, 2)

	svc := &scopeGuardMockService{dir: "/original"}
	guard := NewChdirGuard(session, svc)

	// First check: scope is new → ChdirChanged
	status, _ := guard.Check()
	if status != ChdirChanged {
		t.Fatalf("first Check() = %v, want ChdirChanged", status)
	}

	// Second check: same scope → ChdirUnchanged
	status, _ = guard.Check()
	if status != ChdirUnchanged {
		t.Fatalf("second Check() = %v, want ChdirUnchanged", status)
	}

	// Change scope in session
	session.Set(SessionKeyActiveChdirAbs, "/project/modules/staging")

	// Third check: scope changed → ChdirChanged
	status, scoped := guard.Check()
	if status != ChdirChanged {
		t.Fatalf("third Check() = %v, want ChdirChanged", status)
	}
	mock := scoped.(*scopeGuardMockService)
	if mock.dir != "/project/modules/staging" {
		t.Errorf("scoped dir = %q, want %q", mock.dir, "/project/modules/staging")
	}
}

func TestChdirGuard_CurrentChdir(t *testing.T) {
	session := NewSession()
	session.Set(SessionKeyActiveChdirAbs, "/project/modules/prod")
	session.Set(SessionKeyChdirCount, 2)

	svc := &scopeGuardMockService{dir: "/original"}
	guard := NewChdirGuard(session, svc)

	// Before first check, no scope tracked
	if got := guard.CurrentChdir(); got != "" {
		t.Errorf("before Check(), CurrentChdir() = %q, want empty", got)
	}

	guard.Check()

	if got := guard.CurrentChdir(); got != "/project/modules/prod" {
		t.Errorf("after Check(), CurrentChdir() = %q, want %q", got, "/project/modules/prod")
	}
}

func TestChdirGuard_NilSession(t *testing.T) {
	svc := &scopeGuardMockService{dir: "/original"}
	guard := NewChdirGuard(nil, svc)

	status, scoped := guard.Check()
	if status != ChdirUnchanged {
		t.Errorf("nil session: Check() = %v, want ChdirUnchanged", status)
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
