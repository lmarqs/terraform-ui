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
