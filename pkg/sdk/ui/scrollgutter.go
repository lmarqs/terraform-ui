package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ScrollGutterOpts configures the scroll gutter rendering.
type ScrollGutterOpts struct {
	ViewOffset     int
	TotalItems     int
	ViewportHeight int
	Width          int
}

const (
	gutterTop   = "▲"
	gutterBot   = "▼"
	gutterThumb = "┃"
	gutterTrack = "│"
)

// RenderScrollGutter appends a right-edge scrollbar gutter to each line.
// Returns lines unchanged if TotalItems <= ViewportHeight (no overflow).
func RenderScrollGutter(lines []string, opts ScrollGutterOpts) []string {
	if lines == nil {
		return nil
	}
	if opts.TotalItems <= opts.ViewportHeight {
		return lines
	}

	vpH := opts.ViewportHeight
	if vpH <= 0 {
		return lines
	}

	// The thumb moves within the cells between the caps: rows 1 to trackLen.
	trackLen := min(vpH, len(lines)) - 2
	thumbSize := max(1, min(trackLen, trackLen*vpH/opts.TotalItems))
	thumbStart := max(0, min(trackLen-thumbSize, opts.ViewOffset*(trackLen-thumbSize)/(opts.TotalItems-vpH)))

	result := make([]string, len(lines))
	for i, line := range lines {
		var ch string
		switch {
		case i == 0:
			ch = gutterTop
		case i == len(lines)-1:
			ch = gutterBot
		case i >= thumbStart+1 && i < thumbStart+1+thumbSize:
			ch = gutterThumb
		default:
			ch = gutterTrack
		}
		if opts.Width > 0 {
			pad := opts.Width - lipgloss.Width(line)
			if pad > 0 {
				line = line + strings.Repeat(" ", pad)
			}
		}
		result[i] = line + ch
	}

	return result
}
