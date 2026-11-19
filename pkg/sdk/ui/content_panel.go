package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// GutterWidth is the number of columns reserved for the scroll gutter and its margin.
const GutterWidth = 2

// NeedsGutter returns true if the total item count exceeds the viewport height.
func NeedsGutter(totalItems, height int) bool {
	return totalItems > height
}

// ContentWidth returns the available width for row content given whether a gutter is present.
func ContentWidth(width int, hasGutter bool) int {
	if hasGutter {
		return width - GutterWidth
	}
	return width
}

// ContentPanel is a stateful rendering component that owns the horizontal layout
// concerns of a scrollable content area: truncation, horizontal scroll, wrap toggle,
// cursor highlighting, and scroll gutter alignment.
//
// Vertical navigation (which rows exist, the cursor) stays with the tree or list that
// understands the data structure. Callers either window the rows themselves and call
// Render, or pass every row to RenderWindow, which picks the window from the offset it
// returned last frame; the caller stores that offset.
type ContentPanel struct {
	hScroll  int
	wrapMode bool

	// SelectedStyle applies cursor highlighting. If nil, uses default.
	SelectedStyle func(s string, width int) string
}

// NewContentPanel creates a panel with default state.
func NewContentPanel() *ContentPanel {
	return &ContentPanel{}
}

// RenderParams holds the per-frame inputs needed for rendering.
type RenderParams struct {
	// Rows is the pre-windowed visible rows to render; RenderWindow takes every row instead.
	Rows []string
	// Width is the total available width (including gutter).
	Width int
	// Height is the maximum number of visual lines to render.
	Height int
	// TotalItems is the total number of items in the full list (for gutter thumb).
	// RenderWindow ignores it and counts Rows.
	TotalItems int
	// Cursor is the index into Rows to highlight (-1 for none).
	Cursor int
	// ScrollOffset is the position in the full list (for gutter thumb positioning).
	// For RenderWindow it is the offset RenderWindow returned last frame: rows with wrap
	// off, wrapped lines with wrap on.
	ScrollOffset int
	// CursorEnd is one past the last row of the cursor's block, such as a row and the
	// detail rows under it. Zero or less means the block is the cursor row alone.
	CursorEnd int
}

// HScroll returns the current horizontal scroll offset.
func (p *ContentPanel) HScroll() int {
	return p.hScroll
}

// WrapMode returns whether wrap mode is active.
func (p *ContentPanel) WrapMode() bool {
	return p.wrapMode
}

// HandleKey processes horizontal navigation keys.
// Returns true if the key was consumed.
func (p *ContentPanel) HandleKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "left":
		p.hScroll -= 10
		if p.hScroll < 0 {
			p.hScroll = 0
		}
		return true
	case "right":
		if !p.wrapMode {
			p.hScroll += 10
		}
		return true
	case "ctrl+w":
		p.wrapMode = !p.wrapMode
		p.hScroll = 0
		return true
	}
	return false
}

// ResetScroll resets horizontal scroll (e.g. on filter change or reload).
func (p *ContentPanel) ResetScroll() {
	p.hScroll = 0
}

// Render produces the final output string with all layout applied.
func (p *ContentPanel) Render(params RenderParams) string {
	if len(params.Rows) == 0 || params.Height <= 0 {
		return ""
	}

	if p.wrapMode {
		// Rows outside the slice count as one line each: the panel never sees them wrapped.
		above := max(0, params.ScrollOffset)
		below := max(0, params.TotalItems-above-len(params.Rows))
		output, _ := p.renderWrapped(params, 0, above, below)
		return output
	}

	hasGutter := params.TotalItems > params.Height
	contentWidth := ContentWidth(params.Width, hasGutter)

	var lines []string
	for i, row := range params.Rows[:min(len(params.Rows), params.Height)] {
		line := row
		if p.hScroll > 0 {
			line = scrollLeft(line, p.hScroll)
		}
		line = truncateStyled(line, contentWidth)

		if i == params.Cursor {
			line = p.applySelected(line, contentWidth)
		}

		lines = append(lines, line)
	}

	if hasGutter {
		lines = RenderScrollGutter(lines, ScrollGutterOpts{
			ViewOffset:     params.ScrollOffset,
			TotalItems:     params.TotalItems,
			ViewportHeight: params.Height,
			Width:          params.Width - 1,
		})
	}

	return strings.Join(lines, "\n")
}

// RenderWindow renders a list from every row rather than a pre-windowed slice. It reads
// Rows, Cursor, CursorEnd, Width, Height and ScrollOffset, the offset it returned last frame.
// It scrolls from that offset only as far as needed to show the cursor's block, measuring
// in rows with wrap off and in wrapped lines with wrap on, so the cursor row's first line
// is always drawn. The gutter thumb uses the same measure. The caller keeps the returned
// offset for the next frame.
func (p *ContentPanel) RenderWindow(params RenderParams) (string, int) {
	rows, height := params.Rows, params.Height
	if height <= 0 {
		return "", params.ScrollOffset
	}
	if !p.wrapMode {
		start, end := blockRows(params.Cursor, params.CursorEnd, len(rows))
		offset := scrollTo(params.ScrollOffset, start, end, height, len(rows))
		return p.Render(RenderParams{
			Rows:         rows[offset:min(len(rows), offset+height)],
			Width:        params.Width,
			Height:       height,
			TotalItems:   len(rows),
			Cursor:       params.Cursor - offset,
			ScrollOffset: offset,
		}), offset
	}

	return p.renderWrapped(params, params.ScrollOffset, 0, 0)
}

// renderWrapped draws params.Rows wrapped, scrolled from offset (in lines) just far enough
// to show the cursor's block. above and below are lines outside Rows, which place the
// gutter thumb and decide whether the gutter is drawn. It returns the new offset.
func (p *ContentPanel) renderWrapped(params RenderParams, offset, above, below int) (string, int) {
	rows, height := params.Rows, params.Height
	lines, start, end := p.wrapRows(rows, params.Cursor, params.CursorEnd, params.Width)
	hasGutter := above+len(lines)+below > height
	if hasGutter {
		lines, start, end = p.wrapRows(rows, params.Cursor, params.CursorEnd, ContentWidth(params.Width, true))
	}
	offset = scrollTo(offset, start, end, height, len(lines))
	visible := lines[offset:min(len(lines), offset+height)]
	if hasGutter {
		visible = RenderScrollGutter(visible, ScrollGutterOpts{
			ViewOffset:     above + offset,
			TotalItems:     above + len(lines) + below,
			ViewportHeight: height,
			Width:          params.Width - 1,
		})
	}
	return strings.Join(visible, "\n"), offset
}

// blockRows returns the row span [start, end) of the cursor's block, with start -1 when
// the cursor is outside the rows.
func blockRows(cursor, cursorEnd, total int) (int, int) {
	if cursor < 0 || cursor >= total {
		return -1, -1
	}
	return cursor, min(total, max(cursorEnd, cursor+1))
}

// wrapRows wraps every row to width, styles the cursor row's first line as selected, and
// returns the lines with the line span [start, end) of the cursor's block.
func (p *ContentPanel) wrapRows(rows []string, cursor, cursorEnd, width int) ([]string, int, int) {
	blockStart, blockEnd := blockRows(cursor, cursorEnd, len(rows))
	var lines []string
	start, end := -1, -1
	for i, row := range rows {
		wrapped := wrapStyled(row, width)
		if i == blockStart {
			start = len(lines)
			wrapped[0] = p.applySelected(wrapped[0], width)
		}
		lines = append(lines, wrapped...)
		if i < blockEnd {
			end = len(lines)
		}
	}
	return lines, start, end
}

// scrollTo returns the offset nearest to offset that shows the span [start, end) within
// height units of total, keeping start in view when the span is taller than height.
func scrollTo(offset, start, end, height, total int) int {
	if start >= 0 {
		if start < offset {
			offset = start
		}
		if end > offset+height {
			offset = min(start, end-height)
		}
	}
	return max(0, min(offset, total-height))
}

// wrapStyled breaks a styled string into multiple lines of at most width visible characters.
func wrapStyled(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	visWidth := ansi.StringWidth(s)
	if visWidth <= width {
		return []string{s}
	}
	var result []string
	for ansi.StringWidth(s) > width {
		result = append(result, ansi.Truncate(s, width, ""))
		s = truncateLeft(s, width)
	}
	if ansi.StringWidth(s) > 0 {
		result = append(result, s)
	}
	return result
}

func (p *ContentPanel) applySelected(s string, width int) string {
	if p.SelectedStyle != nil {
		return p.SelectedStyle(s, width)
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}

// truncateStyled truncates a possibly-styled string to maxWidth visible characters
// without breaking ANSI escape sequences.
func truncateStyled(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	return ansi.Truncate(s, maxWidth, "")
}

// scrollLeft removes the first n visible characters from a styled string,
// preserving ANSI escape sequences.
func scrollLeft(s string, n int) string {
	if n <= 0 {
		return s
	}
	w := ansi.StringWidth(s)
	if n >= w {
		return ""
	}
	return truncateLeft(s, n)
}

// truncateLeft removes the first n visible characters from an ANSI string,
// preserving escape sequences that apply to remaining content.
func truncateLeft(s string, n int) string {
	if n <= 0 {
		return s
	}

	stripped := ansi.Strip(s)
	totalWidth := ansi.StringWidth(stripped)
	if n >= totalWidth {
		return ""
	}

	var result strings.Builder
	visibleCount := 0
	i := 0
	inEscape := false

	for i < len(s) {
		b := s[i]

		if b == 0x1b {
			inEscape = true
			if visibleCount >= n {
				result.WriteByte(b)
			}
			i++
			continue
		}

		if inEscape {
			if visibleCount >= n {
				result.WriteByte(b)
			}
			if b != '[' && b != ';' && (b < '0' || b > '9') {
				inEscape = false
			}
			i++
			continue
		}

		visibleCount++
		if visibleCount > n {
			result.WriteByte(b)
		}
		i++
	}

	return result.String()
}
