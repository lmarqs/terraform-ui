package sdk

import "testing"

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
