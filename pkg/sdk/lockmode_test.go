package sdk

import "testing"

func TestLockModeFromPtr_WhenCalled_ShouldConvertCorrectly(t *testing.T) {
	trueVal := true
	falseVal := false
	tests := []struct {
		name string
		ptr  *bool
		want LockMode
	}{
		{"ShouldReturnDefaultForNil", nil, LockDefault},
		{"ShouldReturnEnabledForTrue", &trueVal, LockEnabled},
		{"ShouldReturnDisabledForFalse", &falseVal, LockDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LockModeFromPtr(tt.ptr); got != tt.want {
				t.Errorf("LockModeFromPtr(%v) = %v, want %v", tt.ptr, got, tt.want)
			}
		})
	}
}

func TestLockTimeout_String_WhenCalled_ShouldReturnUnderlyingString(t *testing.T) {
	tests := []struct {
		name    string
		timeout LockTimeout
		want    string
	}{
		{"ShouldReturnValue", LockTimeout("5m"), "5m"},
		{"ShouldReturnEmpty", LockTimeout(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.timeout.String(); got != tt.want {
				t.Errorf("LockTimeout(%q).String() = %q, want %q", string(tt.timeout), got, tt.want)
			}
		})
	}
}

func TestLockMode_Or_WhenCalled_ShouldPreferExplicitChoice(t *testing.T) {
	tests := []struct {
		name     string
		mode     LockMode
		fallback LockMode
		want     LockMode
	}{
		{"ShouldFallBackWhenUnspecified", LockDefault, LockDisabled, LockDisabled},
		{"ShouldKeepEnabledOverFallback", LockEnabled, LockDisabled, LockEnabled},
		{"ShouldKeepDisabledOverFallback", LockDisabled, LockEnabled, LockDisabled},
		{"ShouldStayUnspecifiedWhenBothAre", LockDefault, LockDefault, LockDefault},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.Or(tt.fallback); got != tt.want {
				t.Errorf("LockMode(%v).Or(%v) = %v, want %v", tt.mode, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestLockTimeout_Or_WhenCalled_ShouldPreferExplicitChoice(t *testing.T) {
	tests := []struct {
		name     string
		timeout  LockTimeout
		fallback LockTimeout
		want     LockTimeout
	}{
		{"ShouldFallBackWhenEmpty", LockTimeout(""), LockTimeout("30s"), LockTimeout("30s")},
		{"ShouldKeepValueOverFallback", LockTimeout("10s"), LockTimeout("30s"), LockTimeout("10s")},
		{"ShouldStayEmptyWhenBothAre", LockTimeout(""), LockTimeout(""), LockTimeout("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.timeout.Or(tt.fallback); got != tt.want {
				t.Errorf("LockTimeout(%q).Or(%q) = %q, want %q", tt.timeout, tt.fallback, got, tt.want)
			}
		})
	}
}
