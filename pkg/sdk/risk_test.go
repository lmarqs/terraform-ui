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

func TestOverallRisk_WhenEmpty_ShouldReturnRiskNone(t *testing.T) {
	result := OverallRisk(nil)
	if result != RiskNone {
		t.Errorf("OverallRisk(nil) = %d, want %d (RiskNone)", result, RiskNone)
	}
}

func TestOverallRisk_WhenAllSameLevel_ShouldReturnThatLevel(t *testing.T) {
	changes := []PlanChange{
		{Risk: RiskMedium},
		{Risk: RiskMedium},
		{Risk: RiskMedium},
	}

	result := OverallRisk(changes)
	if result != RiskMedium {
		t.Errorf("OverallRisk = %d, want %d (RiskMedium)", result, RiskMedium)
	}
}

func TestOverallRisk_WhenMixed_ShouldReturnHighest(t *testing.T) {
	tests := []struct {
		name    string
		changes []PlanChange
		want    RiskLevel
	}{
		{
			"ShouldReturnCriticalWhenPresent",
			[]PlanChange{{Risk: RiskLow}, {Risk: RiskCritical}, {Risk: RiskMedium}},
			RiskCritical,
		},
		{
			"ShouldReturnHighWhenNoCritical",
			[]PlanChange{{Risk: RiskLow}, {Risk: RiskHigh}, {Risk: RiskMedium}},
			RiskHigh,
		},
		{
			"ShouldReturnMediumWhenNoHigh",
			[]PlanChange{{Risk: RiskLow}, {Risk: RiskMedium}, {Risk: RiskNone}},
			RiskMedium,
		},
		{
			"ShouldReturnLowWhenNoMedium",
			[]PlanChange{{Risk: RiskNone}, {Risk: RiskLow}, {Risk: RiskNone}},
			RiskLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OverallRisk(tt.changes)
			if got != tt.want {
				t.Errorf("OverallRisk = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestOverallRisk_WhenSingleChange_ShouldReturnItsRisk(t *testing.T) {
	changes := []PlanChange{{Risk: RiskHigh}}
	result := OverallRisk(changes)
	if result != RiskHigh {
		t.Errorf("OverallRisk = %d, want %d (RiskHigh)", result, RiskHigh)
	}
}
