package sdktest

import (
	"strings"
	"testing"
)

func TestUnifiedDiff_WhenInputsAreIdentical_ShouldReturnHeaderOnly(t *testing.T) {
	result := unifiedDiff("hello\nworld\n", "hello\nworld\n")
	expected := "--- expected\n+++ actual\n"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestUnifiedDiff_WhenInputsDiffer_ShouldShowChangedLines(t *testing.T) {
	result := unifiedDiff("hello\nworld\n", "hello\nearth\n")
	if !strings.Contains(result, "-2: world") {
		t.Fatalf("expected diff to contain '-2: world', got %q", result)
	}
	if !strings.Contains(result, "+2: earth") {
		t.Fatalf("expected diff to contain '+2: earth', got %q", result)
	}
	if strings.Contains(result, "-1:") || strings.Contains(result, "+1:") {
		t.Fatalf("expected no diff for matching line 1, got %q", result)
	}
}

func TestUnifiedDiff_WhenExpectedIsEmpty_ShouldShowAllAsAdditions(t *testing.T) {
	result := unifiedDiff("", "hello\nworld\n")
	if !strings.Contains(result, "+1: hello") {
		t.Fatalf("expected diff to contain '+1: hello', got %q", result)
	}
	if !strings.Contains(result, "+2: world") {
		t.Fatalf("expected diff to contain '+2: world', got %q", result)
	}
}

func TestUnifiedDiff_WhenActualIsEmpty_ShouldShowAllAsRemovals(t *testing.T) {
	result := unifiedDiff("hello\nworld\n", "")
	if !strings.Contains(result, "-1: hello") {
		t.Fatalf("expected diff to contain '-1: hello', got %q", result)
	}
	if !strings.Contains(result, "-2: world") {
		t.Fatalf("expected diff to contain '-2: world', got %q", result)
	}
}

func TestUnifiedDiff_WhenBothAreEmpty_ShouldReturnHeaderOnly(t *testing.T) {
	result := unifiedDiff("", "")
	expected := "--- expected\n+++ actual\n"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestUnifiedDiff_WhenActualHasMoreLines_ShouldShowAdditions(t *testing.T) {
	result := unifiedDiff("line1\n", "line1\nline2\nline3\n")
	if strings.Contains(result, "-1:") || strings.Contains(result, "+1:") {
		t.Fatalf("expected no diff for matching line 1, got %q", result)
	}
	if !strings.Contains(result, "+2: line2") {
		t.Fatalf("expected diff to contain '+2: line2', got %q", result)
	}
	if !strings.Contains(result, "+3: line3") {
		t.Fatalf("expected diff to contain '+3: line3', got %q", result)
	}
}

func TestUnifiedDiff_WhenExpectedHasMoreLines_ShouldShowRemovals(t *testing.T) {
	result := unifiedDiff("line1\nline2\nline3\n", "line1\n")
	if !strings.Contains(result, "-2: line2") {
		t.Fatalf("expected diff to contain '-2: line2', got %q", result)
	}
	if !strings.Contains(result, "-3: line3") {
		t.Fatalf("expected diff to contain '-3: line3', got %q", result)
	}
}

func TestUnifiedDiff_WhenMultipleLinesDiffer_ShouldShowAllDifferences(t *testing.T) {
	expected := "aaa\nbbb\nccc\n"
	actual := "xxx\nbbb\nyyy\n"
	result := unifiedDiff(expected, actual)

	if !strings.Contains(result, "-1: aaa") {
		t.Fatalf("expected diff to contain '-1: aaa', got %q", result)
	}
	if !strings.Contains(result, "+1: xxx") {
		t.Fatalf("expected diff to contain '+1: xxx', got %q", result)
	}
	if !strings.Contains(result, "-3: ccc") {
		t.Fatalf("expected diff to contain '-3: ccc', got %q", result)
	}
	if !strings.Contains(result, "+3: yyy") {
		t.Fatalf("expected diff to contain '+3: yyy', got %q", result)
	}
}
