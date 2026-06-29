package sdk

import (
	"strings"
	"testing"
)

func TestPlanSummary_SummaryLine_GivenActionCounts_ShouldRenderEachKind(t *testing.T) {
	s := &PlanSummary{ToCreate: 1, ToUpdate: 2, ToDelete: 3, ToReplace: 4}

	line := s.SummaryLine()
	for _, want := range []string{"Plan:", "1 to add", "2 to change", "3 to destroy", "4 to replace"} {
		if !strings.Contains(line, want) {
			t.Errorf("SummaryLine() = %q, missing %q", line, want)
		}
	}
}

func TestPlanSummary_SummaryLine_GivenZeroCount_ShouldOmitAction(t *testing.T) {
	s := &PlanSummary{ToCreate: 1}

	line := s.SummaryLine()
	if !strings.Contains(line, "1 to add") {
		t.Errorf("SummaryLine() = %q, want '1 to add'", line)
	}
	for _, omitted := range []string{"to change", "to destroy", "to replace"} {
		if strings.Contains(line, omitted) {
			t.Errorf("SummaryLine() = %q, should omit %q for a zero count", line, omitted)
		}
	}
}

func TestPlanSummary_SummaryLine_GivenNoChanges_ShouldSayNoChanges(t *testing.T) {
	s := &PlanSummary{}

	if line := s.SummaryLine(); !strings.Contains(line, "no changes") {
		t.Errorf("SummaryLine() = %q, want 'no changes'", line)
	}
}
