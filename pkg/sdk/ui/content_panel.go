package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// GutterWidth is the number of columns reserved for the scroll gutter and its margin.
const GutterWidth = 2

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
// returned last frame; the caller stores that offset. ScrollBlock moves the window inside
// a cursor block taller than it.
type ContentPanel struct {
	hScroll  int
	wrapMode bool
	block    blockScroll

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
	// For RenderWindow it is the offset RenderWindow returned last frame: the index of the
	// window's top row.
	ScrollOffset int
	// CursorEnd is one past the last row of the cursor's block, such as a row and the
	// detail rows under it. Zero or less means the block is the cursor row alone.
	CursorEnd int
}

// blockScroll is how far RenderWindow has moved into the cursor's block, in the unit it
// measured the block in last frame: rows with wrap off, wrapped lines with wrap on.
type blockScroll struct {
	cursor int // the cursor the scroll belongs to; another cursor starts at the block's top
	into   int
	max    int  // the block's height minus the window's, or 0 when the block fits
	jumped bool // ScrollBlockTo set into for the next frame, whatever its cursor
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

// ResetScroll resets horizontal scroll and the scroll inside the cursor's block (e.g. on
// filter change or reload).
func (p *ContentPanel) ResetScroll() {
	p.hScroll = 0
	p.block = blockScroll{}
}

// ScrollBlock moves the next RenderWindow by delta lines inside the cursor's block when
// the block drawn last frame is taller than the window. It returns false, moving nothing,
// when that would pass the block's first or last line, so the caller moves the cursor.
// It reads the block RenderWindow measured, so a caller that uses it must not call Render.
func (p *ContentPanel) ScrollBlock(delta int) bool {
	into := p.block.into + delta
	if into < 0 || into > p.block.max {
		return false
	}
	p.block.into = into
	return true
}

// ScrollBlockTo makes the next RenderWindow show line of the cursor's block, or the block's
// last page when line is past it, even when the cursor moves before that frame. A jump
// uses it: 0 for the block's first line, math.MaxInt for its last.
func (p *ContentPanel) ScrollBlockTo(line int) {
	p.block.into, p.block.jumped = max(0, line), true
}

// Render produces the final output string with all layout applied.
func (p *ContentPanel) Render(params RenderParams) string {
	if len(params.Rows) == 0 || params.Height <= 0 {
		return ""
	}

	// With wrap on, Render takes no cursor: the caller scrolls by rows, so the gutter counts
	// rows, and narrowing the content for wrapped lines would push lines past where the
	// caller can scroll. RenderWindow draws the cursor with wrap on.
	hasGutter := params.TotalItems > params.Height
	contentWidth := ContentWidth(params.Width, hasGutter)

	var lines []string
	if p.wrapMode {
		lines, _ = p.wrapRows(params.Rows, -1, contentWidth)
	} else {
		for i, row := range params.Rows {
			// The offset is shared by every row; a row shorter than it shows its tail, so the
			// cursor never lands on a blank line.
			offset := min(p.hScroll, ansi.StringWidth(row)-contentWidth)
			line := truncateStyled(scrollLeft(row, offset), contentWidth)

			if i == params.Cursor {
				line = p.applySelected(line, contentWidth)
			}

			lines = append(lines, line)
		}
	}
	lines = lines[:min(len(lines), params.Height)]

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
// It scrolls from that offset only as far as needed to show the cursor's block, so the
// cursor row's first line is drawn unless ScrollBlock moved the window down a block taller
// than it. With wrap on it measures in wrapped lines, so
// the window can start partway through its top row, and the gutter thumb uses the same
// measure. The returned offset is the index of the window's top row in both wrap modes;
// the caller keeps it for the next frame.
func (p *ContentPanel) RenderWindow(params RenderParams) (string, int) {
	rows, height := params.Rows, params.Height
	if height <= 0 {
		return "", params.ScrollOffset
	}
	if params.Cursor != p.block.cursor && !p.block.jumped {
		p.block.into = 0
	}
	p.block.cursor, p.block.jumped = params.Cursor, false
	if !p.wrapMode {
		start, end := blockRows(params.Cursor, params.CursorEnd, len(rows))
		offset := p.scrollTo(params.ScrollOffset, start, end, height, len(rows))
		return p.Render(RenderParams{
			Rows:         rows[offset:min(len(rows), offset+height)],
			Width:        params.Width,
			Height:       height,
			TotalItems:   len(rows),
			Cursor:       params.Cursor - offset,
			ScrollOffset: offset,
		}), offset
	}

	return p.renderWrapped(params, params.ScrollOffset)
}

// renderWrapped draws params.Rows wrapped, starting from the first line of row top and
// scrolling just far enough to show the cursor's block. It returns the row at the top of
// the window.
func (p *ContentPanel) renderWrapped(params RenderParams, top int) (string, int) {
	rows, height := params.Rows, params.Height
	width := params.Width
	lines, starts := p.wrapRows(rows, params.Cursor, width)
	hasGutter := len(lines) > height
	if hasGutter {
		width = ContentWidth(width, true)
		lines, starts = p.wrapRows(rows, params.Cursor, width)
	}
	start, end := blockRows(params.Cursor, params.CursorEnd, len(rows))
	if start >= 0 {
		start, end = starts[start], starts[end]
	}
	offset := p.scrollTo(starts[min(max(0, top), len(rows))], start, end, height, len(lines))
	visible := lines[offset:min(len(lines), offset+height)]
	if hasGutter {
		visible = RenderScrollGutter(visible, ScrollGutterOpts{
			ViewOffset:     offset,
			TotalItems:     len(lines),
			ViewportHeight: height,
			Width:          params.Width - 1,
		})
	}
	top = 0
	for top+1 < len(rows) && starts[top+1] <= offset {
		top++
	}
	return strings.Join(visible, "\n"), top
}

// blockRows returns the row span [start, end) of the cursor's block, with start -1 when
// the cursor is outside the rows.
func blockRows(cursor, cursorEnd, total int) (int, int) {
	if cursor < 0 || cursor >= total {
		return -1, -1
	}
	return cursor, min(total, max(cursorEnd, cursor+1))
}

// wrapRows wraps every row to width and styles the cursor row's first line as selected.
// It returns the lines and, for each row, the index of its first line, followed by the
// line count.
func (p *ContentPanel) wrapRows(rows []string, cursor, width int) ([]string, []int) {
	var lines []string
	starts := make([]int, 0, len(rows)+1)
	for i, row := range rows {
		wrapped := wrapStyled(row, width)
		if i == cursor {
			wrapped[0] = p.applySelected(wrapped[0], width)
		}
		starts = append(starts, len(lines))
		lines = append(lines, wrapped...)
	}
	return lines, append(starts, len(lines))
}

// scrollTo returns the offset nearest to offset that shows the span [start, end) within
// height units of total. A span taller than height shows from start, moved down as far as
// ScrollBlock asked; scrollTo records the span's overflow for ScrollBlock.
func (p *ContentPanel) scrollTo(offset, start, end, height, total int) int {
	p.block.max = 0
	if start >= 0 {
		p.block.max = max(0, end-start-height)
	}
	p.block.into = min(p.block.into, p.block.max)
	if p.block.into > 0 {
		return start + p.block.into
	}
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
		prefix := ansi.Truncate(s, width, "")
		// A wide character that does not fit stays out of the prefix, so cut where the prefix ends.
		cut := ansi.StringWidth(prefix)
		if cut == 0 {
			// The first character is wider than the line: drop it, as truncateStyled does.
			s = truncateLeft(s, 1)
			continue
		}
		result = append(result, prefix)
		s = truncateLeft(s, cut)
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

// truncateLeft removes the first n cells from an ANSI string, preserving escape sequences
// that apply to remaining content. A grapheme cluster that starts before cell n is removed whole.
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

		// Walk grapheme clusters, as ansi.Truncate does, so a flag or ZWJ sequence is never split.
		cluster, _, width, _ := uniseg.FirstGraphemeClusterInString(s[i:], -1)
		if visibleCount >= n {
			result.WriteString(cluster)
		}
		visibleCount += width
		i += len(cluster)
	}

	return result.String()
}
