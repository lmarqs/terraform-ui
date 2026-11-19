package sdk

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// RenderAttributeDiffs renders every attribute for mutation inspection.
func RenderAttributeDiffs(diffs []AttributeDiff) string {
	var b strings.Builder
	for _, diff := range diffs {
		if diff.Sensitive {
			fmt.Fprintf(&b, "  %s: (sensitive)\n", diff.Key)
			continue
		}
		if diff.ForcesNew {
			fmt.Fprintf(&b, "  %s (forces new):\n", diff.Key)
		} else {
			fmt.Fprintf(&b, "  %s:\n", diff.Key)
		}
		fmt.Fprintf(&b, "    - %s\n", diff.OldValue)
		fmt.Fprintf(&b, "    + %s\n", diff.NewValue)
	}
	return b.String()
}

// compactRowFixedWidth is the width of a compact row outside its key, values and suffix:
// a four-space indent, ": " after the key and " → " between the values.
const compactRowFixedWidth = 9

// RenderCompactAttributeDiffs renders two paths and an overflow count for a resource row.
func RenderCompactAttributeDiffs(diffs []AttributeDiff, width int) []string {
	if width <= 0 {
		return nil
	}
	var rows []string
	for _, diff := range diffs[:min(2, len(diffs))] {
		suffix := ""
		if diff.ForcesNew {
			suffix = " (forces new)"
		}
		value := "(sensitive)"
		if !diff.Sensitive {
			oldValue := strings.ReplaceAll(diff.OldValue, "\n", " ")
			newValue := strings.ReplaceAll(diff.NewValue, "\n", " ")
			budget := max(0, width-ansi.StringWidth(diff.Key)-compactRowFixedWidth-len(suffix))
			oldWidth := min(ansi.StringWidth(oldValue), budget/2)
			newWidth := max(0, budget-oldWidth)
			if oldValue == "(known after apply)" {
				oldWidth = len(oldValue)
			}
			if newValue == "(known after apply)" {
				newWidth = len(newValue)
			}
			value = ansi.Truncate(oldValue, oldWidth, "…") + " → " + ansi.Truncate(newValue, newWidth, "…")
		}
		value += suffix
		line := "    " + diff.Key + ": " + value
		if ansi.StringWidth(line) > width {
			rows = append(rows, strings.Split(ansi.Hardwrap("    "+diff.Key+suffix+":", width, false), "\n")...)
			rows = append(rows, strings.Split(ansi.Hardwrap("    "+strings.TrimSuffix(value, suffix), width, false), "\n")...)
		} else {
			rows = append(rows, line)
		}

	}

	if len(diffs) > 2 {
		rows = append(rows, ansi.Truncate(fmt.Sprintf("    +%d more", len(diffs)-2), max(0, width), ""))
	}
	return rows
}
