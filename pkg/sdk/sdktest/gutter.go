package sdktest

import "strings"

// GutterColumn returns the scroll-gutter glyph ending each line of view that has one, top to
// bottom: "▲" and "▼" for the caps, "┃" for the thumb and "│" for the track.
func GutterColumn(view string) []string {
	var gutter []string
	for line := range strings.SplitSeq(view, "\n") {
		runes := []rune(line)
		if len(runes) == 0 {
			continue
		}
		if g := string(runes[len(runes)-1]); strings.Contains("▲▼┃│", g) {
			gutter = append(gutter, g)
		}
	}
	return gutter
}
