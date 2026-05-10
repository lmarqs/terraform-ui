package sdktest

import (
	"fmt"
	"strings"
)

func unifiedDiff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var b strings.Builder
	b.WriteString("--- expected\n+++ actual\n")

	maxLen := len(expectedLines)
	if len(actualLines) > maxLen {
		maxLen = len(actualLines)
	}

	for i := range maxLen {
		var exp, act string
		if i < len(expectedLines) {
			exp = expectedLines[i]
		}
		if i < len(actualLines) {
			act = actualLines[i]
		}

		if exp != act {
			if i < len(expectedLines) {
				b.WriteString(fmt.Sprintf("-%d: %s\n", i+1, exp))
			}
			if i < len(actualLines) {
				b.WriteString(fmt.Sprintf("+%d: %s\n", i+1, act))
			}
		}
	}

	return b.String()
}
