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
// Vertical navigation (cursor position, viewport offset) is NOT owned by the panel —
// these remain in the tree/list that understands the data structure. The panel takes
// them as inputs for rendering.
//
// Plugins provide pre-windowed Rows and call Render(). The panel formats what it's given.
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
	// Rows is the pre-windowed visible rows to render.
	Rows []string
	// Width is the total available width (including gutter).
	Width int
	// Height is the maximum number of visual lines to render.
	Height int
	// TotalItems is the total number of items in the full list (for gutter thumb).
	TotalItems int
	// Cursor is the index into Rows to highlight (-1 for none).
	Cursor int
	// ScrollOffset is the position in the full list (for gutter thumb positioning).
	ScrollOffset int
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
	if len(params.Rows) == 0 {
		return ""
	}

	hasGutter := params.TotalItems > params.Height
	contentWidth := ContentWidth(params.Width, hasGutter)

	var lines []string
	linesUsed := 0

	for i, row := range params.Rows {
		if linesUsed >= params.Height {
			break
		}

		if p.wrapMode {
			wrapped := wrapStyled(row, contentWidth)
			remaining := params.Height - linesUsed
			if len(wrapped) > remaining {
				wrapped = wrapped[:remaining]
			}
			if i == params.Cursor && len(wrapped) > 0 {
				wrapped[0] = p.applySelected(wrapped[0], contentWidth)
			}
			lines = append(lines, wrapped...)
			linesUsed += len(wrapped)
		} else {
			line := row
			if p.hScroll > 0 {
				line = scrollLeft(line, p.hScroll)
			}
			line = truncateStyled(line, contentWidth)

			if i == params.Cursor {
				line = p.applySelected(line, contentWidth)
			}

			lines = append(lines, line)
			linesUsed++
		}
	}

	if len(lines) == 0 {
		return ""
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
