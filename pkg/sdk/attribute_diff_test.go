package sdk

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderAttributeDiffs_GivenValues_ShouldShowEveryPathAndMaskSensitiveValues(t *testing.T) {
	diffs := []AttributeDiff{
		{Key: "ami", OldValue: "ami-old", NewValue: "(known after apply)", ForcesNew: true},
		{Key: "password", OldValue: "secret-before", NewValue: "secret-after", Sensitive: true, ForcesNew: true},
		{Key: "tags.Owner", OldValue: "ops", NewValue: "platform"},
	}
	view := ansi.Strip(RenderAttributeDiffs(diffs))
	for _, want := range []string{"  ami (forces new):", "    - ami-old", "    + (known after apply)", "password: (sensitive)", "  tags.Owner:", "    - ops", "    + platform"} {
		if !strings.Contains(view, want) {
			t.Errorf("full diff missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "secret-before") || strings.Contains(view, "secret-after") {
		t.Errorf("full diff leaks sensitive values:\n%s", view)
	}
}

func TestRenderAttributeDiffs_GivenNoChanges_ShouldRenderNothing(t *testing.T) {
	if got := RenderAttributeDiffs(nil); got != "" {
		t.Errorf("empty diff = %q, want empty", got)
	}
}

func TestRenderCompactAttributeDiffs_GivenThreeChanges_ShouldShowTwoPathsAndOverflow(t *testing.T) {
	diffs := []AttributeDiff{{Key: "size", OldValue: "small", NewValue: "large"}, {Key: "name", OldValue: "old", NewValue: "new"}, {Key: "hidden", OldValue: "x", NewValue: "y"}}
	rows := RenderCompactAttributeDiffs(diffs, 80)
	view := ansi.Strip(strings.Join(rows, "\n"))
	for _, want := range []string{"size: small → large", "name: old → new", "+1 more"} {
		if !strings.Contains(view, want) {
			t.Errorf("compact diff missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "hidden") {
		t.Errorf("overflow path should only appear in full inspection:\n%s", view)
	}
}

func TestRenderCompactAttributeDiffs_GivenSensitiveAndUnknown_ShouldPreserveNotationWithoutSecrets(t *testing.T) {
	rows := RenderCompactAttributeDiffs([]AttributeDiff{
		{Key: "password", OldValue: "secret-before", NewValue: "secret-after", Sensitive: true},
		{Key: "ami", OldValue: strings.Repeat("old", 100), NewValue: "(known after apply)", ForcesNew: true},
	}, 60)
	view := ansi.Strip(strings.Join(rows, "\n"))
	for _, want := range []string{"(sensitive)", "(known after apply)", "forces new"} {
		if !strings.Contains(view, want) {
			t.Errorf("compact notation missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "secret-before") || strings.Contains(view, "secret-after") {
		t.Errorf("compact diff leaks secret:\n%s", view)
	}
	for _, row := range rows {
		if ansi.StringWidth(row) > 60 {
			t.Errorf("compact row exceeds 60 columns: %q", row)
		}
	}
}

func TestRenderCompactAttributeDiffs_GivenLongUnicodeValues_ShouldFitNarrowWidth(t *testing.T) {
	for _, width := range []int{1, 12, 36} {
		rows := RenderCompactAttributeDiffs([]AttributeDiff{{Key: "tags.Name", OldValue: strings.Repeat("界", 40), NewValue: strings.Repeat("世", 40)}}, width)
		if len(rows) == 0 {
			t.Fatalf("width %d discarded a changed path", width)
		}
		for _, row := range rows {
			if ansi.StringWidth(row) > width {
				t.Errorf("row exceeds %d columns: %q", width, row)
			}
		}
	}
}

func TestRenderCompactAttributeDiffs_GivenEmptyOrNoWidth_ShouldRenderNothing(t *testing.T) {
	for _, tc := range []struct {
		diffs []AttributeDiff
		width int
	}{{nil, 80}, {[]AttributeDiff{{Key: "name", OldValue: "old", NewValue: "new"}}, 0}} {
		if got := RenderCompactAttributeDiffs(tc.diffs, tc.width); len(got) != 0 {
			t.Errorf("compact diff = %q, want no rows", got)
		}
	}
}

func TestRenderCompactAttributeDiffs_GivenUnknownReplacementAtNarrowWidth_ShouldPreserveBothNotations(t *testing.T) {
	for _, width := range []int{40, 42} {
		rows := RenderCompactAttributeDiffs([]AttributeDiff{{Key: "ami", OldValue: strings.Repeat("old", 50), NewValue: "(known after apply)", ForcesNew: true}}, width)
		view := ansi.Strip(strings.Join(rows, "\n"))
		for _, want := range []string{"ami", "(known after apply)", "(forces new)"} {
			if !strings.Contains(view, want) {
				t.Errorf("width %d lost %q:\n%s", width, want, view)
			}
		}
		for _, row := range rows {
			if ansi.StringWidth(row) > width {
				t.Errorf("row exceeds %d columns: %q", width, row)
			}
		}
	}
}

func TestRenderCompactAttributeDiffs_GivenUnknownBeforeValue_ShouldKeepEvidenceAndOverflowWhenWrapped(t *testing.T) {
	diffs := []AttributeDiff{
		{Key: "version", OldValue: "(known after apply)", NewValue: "2"},
		{Key: "password", OldValue: "secret-before", NewValue: "secret-after", Sensitive: true},
		{Key: "omitted", OldValue: "one", NewValue: "two"},
	}
	rows := RenderCompactAttributeDiffs(diffs, 32)
	view := ansi.Strip(strings.Join(rows, "\n"))
	for _, want := range []string{"version", "(known after apply)", "(sensitive)", "+1 more"} {
		if !strings.Contains(view, want) {
			t.Errorf("wrapped compact evidence missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "secret-") || strings.Contains(view, "omitted") {
		t.Errorf("compact view exposed sensitive or overflow values:\n%s", view)
	}
	for _, row := range rows {
		if ansi.StringWidth(row) > 32 {
			t.Errorf("wrapped row exceeds 32 columns: %q", row)
		}
	}
}

func TestRenderCompactAttributeDiffs_GivenLongValues_ShouldFillTheWidthExactly(t *testing.T) {
	diffs := []AttributeDiff{{Key: "key", OldValue: strings.Repeat("a", 40), NewValue: strings.Repeat("b", 40)}}
	got := RenderCompactAttributeDiffs(diffs, 30)
	want := []string{"    key: aaaaaaaa… → bbbbbbbb…"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("rows = %q, want %q", got, want)
	}
	if width := ansi.StringWidth(got[0]); width != 30 {
		t.Errorf("row width = %d, want 30", width)
	}
}
