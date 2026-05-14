package sdk

import "testing"

func TestActionSymbol_WhenCalled_ShouldReturnStyledSymbol(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		want   string
	}{
		{"ShouldReturnPlusForCreate", ActionCreate, "+"},
		{"ShouldReturnTildeForUpdate", ActionUpdate, "~"},
		{"ShouldReturnMinusForDelete", ActionDelete, "-"},
		{"ShouldReturnReplaceSymForDeleteThenCreate", ActionDeleteThenCreate, "-/+"},
		{"ShouldReturnReplaceSymForCreateThenDelete", ActionCreateThenDelete, "-/+"},
		{"ShouldReturnArrowForRead", ActionRead, "<="},
		{"ShouldReturnSpaceForNoOp", ActionNoOp, " "},
		{"ShouldReturnSpaceForUnknown", Action("unknown"), " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ActionSymbol(tt.action)
			if got == "" {
				t.Fatal("ActionSymbol returned empty string")
			}
			if !contains(got, tt.want) {
				t.Errorf("ActionSymbol(%q) = %q, want to contain %q", tt.action, got, tt.want)
			}
		})
	}
}

func TestRiskBadge_WhenCalled_ShouldReturnStyledBadge(t *testing.T) {
	tests := []struct {
		name string
		risk RiskLevel
		want string
	}{
		{"ShouldReturnLowBadge", RiskLow, "[low]"},
		{"ShouldReturnMediumBadge", RiskMedium, "[medium]"},
		{"ShouldReturnHighBadge", RiskHigh, "[HIGH]"},
		{"ShouldReturnCriticalBadge", RiskCritical, "[CRITICAL]"},
		{"ShouldReturnEmptyForNone", RiskNone, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RiskBadge(tt.risk)
			if tt.want == "" {
				if got != "" {
					t.Errorf("RiskBadge(%d) = %q, want empty", tt.risk, got)
				}
				return
			}
			if !contains(got, tt.want) {
				t.Errorf("RiskBadge(%d) = %q, want to contain %q", tt.risk, got, tt.want)
			}
		})
	}
}

func TestTruncate_WhenCalled_ShouldTruncateCorrectly(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"ShouldReturnUnchangedWhenShort", "hello", 20, "hello"},
		{"ShouldTruncateWithEllipsis", "abcdefghijklmnop", 13, "abcdefghij..."},
		{"ShouldEnforceMinimumOfTen", "abcdefghijklmnop", 5, "abcdefg..."},
		{"ShouldHandleExactLength", "abcdefghij", 10, "abcdefghij"},
		{"ShouldTruncateOneOver", "abcdefghijk", 10, "abcdefg..."},
		{"ShouldHandleEmptyString", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestScrollWindow_WhenCalled_ShouldCalculateVisibleRange(t *testing.T) {
	tests := []struct {
		name            string
		selected        int
		total           int
		availableHeight int
		minVisible      int
		wantStart       int
		wantEnd         int
	}{
		{"ShouldStartAtZeroWhenSelectedInView", 2, 20, 10, 3, 0, 10},
		{"ShouldScrollWhenSelectedPastHeight", 15, 20, 10, 3, 6, 16},
		{"ShouldClampEndToTotal", 18, 20, 10, 3, 9, 19},
		{"ShouldUseMinVisibleWhenHeightTooSmall", 5, 20, 1, 5, 1, 6},
		{"ShouldHandleSelectedAtZero", 0, 20, 10, 3, 0, 10},
		{"ShouldHandleTotalLessThanHeight", 2, 5, 10, 3, 0, 5},
		{"ShouldHandleSelectedAtEnd", 19, 20, 10, 3, 10, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := ScrollWindow(tt.selected, tt.total, tt.availableHeight, tt.minVisible)
			if start != tt.wantStart {
				t.Errorf("ScrollWindow(%d, %d, %d, %d) start = %d, want %d",
					tt.selected, tt.total, tt.availableHeight, tt.minVisible, start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("ScrollWindow(%d, %d, %d, %d) end = %d, want %d",
					tt.selected, tt.total, tt.availableHeight, tt.minVisible, end, tt.wantEnd)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
