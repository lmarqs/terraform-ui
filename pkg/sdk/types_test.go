package sdk

import "testing"

func TestRiskLevel_String_WhenCalled_ShouldReturnCorrectLabel(t *testing.T) {
	tests := []struct {
		name string
		risk RiskLevel
		want string
	}{
		{"ShouldReturnNone", RiskNone, "none"},
		{"ShouldReturnLow", RiskLow, "low"},
		{"ShouldReturnMedium", RiskMedium, "medium"},
		{"ShouldReturnHigh", RiskHigh, "high"},
		{"ShouldReturnCritical", RiskCritical, "critical"},
		{"ShouldReturnNoneForUnknown", RiskLevel(99), "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.risk.String()
			if got != tt.want {
				t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.risk, got, tt.want)
			}
		})
	}
}

func TestDiagnosticSeverity_IsError_WhenCalled_ShouldIdentifyErrors(t *testing.T) {
	tests := []struct {
		name     string
		severity DiagnosticSeverity
		want     bool
	}{
		{"ShouldReturnTrueForError", SeverityError, true},
		{"ShouldReturnFalseForWarning", SeverityWarning, false},
		{"ShouldReturnFalseForZero", DiagnosticSeverity(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.severity.IsError(); got != tt.want {
				t.Errorf("DiagnosticSeverity(%q).IsError() = %v, want %v", tt.severity, got, tt.want)
			}
		})
	}
}

func TestDiagnosticSeverity_IsWarning_WhenCalled_ShouldIdentifyWarnings(t *testing.T) {
	tests := []struct {
		name     string
		severity DiagnosticSeverity
		want     bool
	}{
		{"ShouldReturnTrueForWarning", SeverityWarning, true},
		{"ShouldReturnFalseForError", SeverityError, false},
		{"ShouldReturnFalseForZero", DiagnosticSeverity(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.severity.IsWarning(); got != tt.want {
				t.Errorf("DiagnosticSeverity(%q).IsWarning() = %v, want %v", tt.severity, got, tt.want)
			}
		})
	}
}

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
