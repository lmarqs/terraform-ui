package ui

// HeightBudget computes the usable row count by subtracting deductions from total,
// clamping to a minimum of 3.
func HeightBudget(total int, deductions ...int) int {
	for _, d := range deductions {
		total -= d
	}
	if total < 3 {
		return 3
	}
	return total
}
