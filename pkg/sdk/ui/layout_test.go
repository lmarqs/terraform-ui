package ui_test

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func TestHeightBudget(t *testing.T) {
	tests := []struct {
		name       string
		total      int
		deductions []int
		want       int
	}{
		{"no deductions", 40, nil, 40},
		{"single deduction", 40, []int{5}, 35},
		{"multiple deductions", 40, []int{2, 3, 2}, 33},
		{"clamps to floor", 10, []int{5, 4, 3}, 3},
		{"exactly at floor", 10, []int{7}, 3},
		{"below floor clamps up", 5, []int{10}, 3},
		{"zero total clamps", 0, nil, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ui.HeightBudget(tt.total, tt.deductions...)
			if got != tt.want {
				t.Errorf("HeightBudget(%d, %v) = %d, want %d", tt.total, tt.deductions, got, tt.want)
			}
		})
	}
}
