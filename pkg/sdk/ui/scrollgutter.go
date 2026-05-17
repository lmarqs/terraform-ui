package ui

// ScrollGutterOpts configures the scroll gutter rendering.
type ScrollGutterOpts struct {
	ViewOffset     int
	TotalItems     int
	ViewportHeight int
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

	thumbSize := vpH * vpH / opts.TotalItems
	if thumbSize < 1 {
		thumbSize = 1
	}

	thumbStart := 0
	if opts.TotalItems > vpH {
		thumbStart = opts.ViewOffset * (vpH - thumbSize) / (opts.TotalItems - vpH)
	}
	if thumbStart < 0 {
		thumbStart = 0
	}
	if thumbStart+thumbSize > vpH {
		thumbStart = vpH - thumbSize
	}

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
		result[i] = line + ch
	}

	return result
}
