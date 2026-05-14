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
