package ui

import "testing"

func TestScrollLeft_GivenNonPositiveN_ShouldReturnOriginal(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"ShouldReturnOriginalWhenNIsZero", 0},
		{"ShouldReturnOriginalWhenNIsNegative", -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scrollLeft("hello", tt.n)
			if result != "hello" {
				t.Errorf("expected original string, got %q", result)
			}
		})
	}
}

func TestTruncateLeft_GivenNonPositiveN_ShouldReturnOriginal(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"ShouldReturnOriginalWhenNIsZero", 0},
		{"ShouldReturnOriginalWhenNIsNegative", -3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateLeft("hello", tt.n)
			if result != "hello" {
				t.Errorf("expected original string, got %q", result)
			}
		})
	}
}

func TestTruncateLeft_GivenNExceedingWidth_ShouldReturnEmpty(t *testing.T) {
	result := truncateLeft("abc", 10)
	if result != "" {
		t.Errorf("expected empty string when n >= totalWidth, got %q", result)
	}
}
