package ui

import (
	"reflect"
	"testing"
)

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

func TestWrapStyled_GivenWidthNarrowerThanAWideCharacter_ShouldDropItAndTerminate(t *testing.T) {
	got := wrapStyled("界a", 1)
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("wrapStyled = %q, want [\"a\"]", got)
	}
}

func TestScrollLeft_GivenNAtOrBeyondTheWidth_ShouldReturnEmpty(t *testing.T) {
	if result := scrollLeft("ab", 10); result != "" {
		t.Errorf("expected empty string when n >= width, got %q", result)
	}
}

const (
	flag       = "\U0001F1E7\U0001F1F7"
	family     = "\U0001F468\u200D\U0001F469\u200D\U0001F467"
	accentedA  = "á"
	redOn      = "\x1b[31m"
	styleReset = "\x1b[0m"
)

func TestWrapStyled_GivenMultiCodePointGrapheme_ShouldKeepItWholeAndNotRepeatFragments(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		want  []string
	}{
		{"ShouldKeepRegionalIndicatorPairWhole", flag + "X", 2, []string{flag, "X"}},
		{"ShouldKeepZWJSequenceWhole", family + "X", 2, []string{family, "X"}},
		{"ShouldKeepCombiningMarkWithItsBase", accentedA + "b", 1, []string{accentedA, "b"}},
		{"ShouldKeepGraphemeWholeInStyledString", redOn + flag + "X" + styleReset, 2, []string{redOn + flag + styleReset, "X" + styleReset}},
		{"ShouldKeepCombiningMarkWithItsBaseInStyledString", redOn + accentedA + "b" + styleReset, 1, []string{redOn + accentedA + styleReset, "b" + styleReset}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapStyled(tt.in, tt.width)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("wrapStyled(%q, %d) = %q, want %q", tt.in, tt.width, got, tt.want)
			}
		})
	}
}

func TestScrollLeft_GivenMultiCodePointGrapheme_ShouldRemoveItWhole(t *testing.T) {
	tests := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"ShouldRemoveRegionalIndicatorPair", flag + "X", 2, "X"},
		{"ShouldRemoveZWJSequence", family + "X", 2, "X"},
		{"ShouldRemoveCombiningMarkWithItsBase", accentedA + "b", 1, "b"},
		{"ShouldRemoveGraphemeWholeWhenTheCutFallsInsideIt", flag + "X", 1, "X"},
		{"ShouldRemoveGraphemeWholeInStyledString", redOn + flag + "X" + styleReset, 2, "X" + styleReset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scrollLeft(tt.in, tt.n)
			if got != tt.want {
				t.Errorf("scrollLeft(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
			}
		})
	}
}
