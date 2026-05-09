package sdk

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Viewport manages scrollable, pannable content for plugins.
type Viewport struct {
	Lines       []string
	ScrollY     int
	ScrollX     int
	Width       int
	Height      int
	WrapEnabled bool
}

// NewViewport creates a viewport with the given dimensions.
func NewViewport(width, height int) *Viewport {
	return &Viewport{
		Width:  width,
		Height: height,
	}
}

// SetContent replaces the viewport's content and resets scroll position.
func (v *Viewport) SetContent(lines []string) {
	v.Lines = lines
	v.ScrollY = 0
	v.ScrollX = 0
}

// SetContentString splits a string by newlines and sets it as content.
func (v *Viewport) SetContentString(content string) {
	v.SetContent(strings.Split(content, "\n"))
}

// SetSize updates the viewport dimensions.
func (v *Viewport) SetSize(width, height int) {
	v.Width = width
	v.Height = height
}

// Render returns the visible portion of content as a string.
func (v *Viewport) Render() string {
	if len(v.Lines) == 0 {
		return ""
	}

	lines := v.Lines
	if v.WrapEnabled {
		lines = v.wrapLines(lines)
	}

	// Clamp scroll
	maxScrollY := len(lines) - v.Height
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	if v.ScrollY > maxScrollY {
		v.ScrollY = maxScrollY
	}

	// Slice visible window
	endIdx := v.ScrollY + v.Height
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	visible := lines[v.ScrollY:endIdx]

	// Apply horizontal scroll
	if !v.WrapEnabled && v.ScrollX > 0 {
		for i, line := range visible {
			if v.ScrollX < len(line) {
				visible[i] = line[v.ScrollX:]
			} else {
				visible[i] = ""
			}
		}
	}

	// Truncate to width
	if !v.WrapEnabled {
		for i, line := range visible {
			if len(line) > v.Width {
				visible[i] = line[:v.Width]
			}
		}
	}

	return strings.Join(visible, "\n")
}

// HandleKey processes navigation keys. Returns true if the key was consumed.
func (v *Viewport) HandleKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up":
		if v.ScrollY > 0 {
			v.ScrollY--
		}
		return true
	case "down":
		maxScrollY := len(v.effectiveLines()) - v.Height
		if maxScrollY < 0 {
			maxScrollY = 0
		}
		if v.ScrollY < maxScrollY {
			v.ScrollY++
		}
		return true
	case "left":
		v.ScrollX -= 10
		if v.ScrollX < 0 {
			v.ScrollX = 0
		}
		return true
	case "right":
		maxLine := v.maxLineLen()
		maxScroll := maxLine - v.Width
		if maxScroll < 0 {
			maxScroll = 0
		}
		v.ScrollX += 10
		if v.ScrollX > maxScroll {
			v.ScrollX = maxScroll
		}
		return true
	case "ctrl+w", "w":
		v.WrapEnabled = !v.WrapEnabled
		v.ScrollY = 0
		v.ScrollX = 0
		return true
	case "g":
		v.ScrollY = 0
		return true
	case "G":
		maxScrollY := len(v.effectiveLines()) - v.Height
		if maxScrollY < 0 {
			maxScrollY = 0
		}
		v.ScrollY = maxScrollY
		return true
	}
	return false
}

// ScrollInfo returns a scroll indicator string like "[5/120]".
func (v *Viewport) ScrollInfo() string {
	total := len(v.effectiveLines())
	if total <= v.Height {
		return ""
	}
	maxScroll := total - v.Height
	if maxScroll <= 0 {
		return ""
	}
	return fmt.Sprintf("[%d/%d]", v.ScrollY+1, maxScroll+1)
}

// TotalLines returns the total number of lines (after wrapping if enabled).
func (v *Viewport) TotalLines() int {
	return len(v.effectiveLines())
}

// AtTop reports whether the viewport is scrolled to the top.
func (v *Viewport) AtTop() bool {
	return v.ScrollY == 0
}

// AtBottom reports whether the viewport is scrolled to the bottom.
func (v *Viewport) AtBottom() bool {
	maxScroll := len(v.effectiveLines()) - v.Height
	if maxScroll <= 0 {
		return true
	}
	return v.ScrollY >= maxScroll
}

func (v *Viewport) effectiveLines() []string {
	if v.WrapEnabled {
		return v.wrapLines(v.Lines)
	}
	return v.Lines
}

func (v *Viewport) wrapLines(lines []string) []string {
	if v.Width <= 0 {
		return lines
	}
	var result []string
	for _, line := range lines {
		if len(line) <= v.Width {
			result = append(result, line)
			continue
		}
		for len(line) > v.Width {
			result = append(result, line[:v.Width])
			line = line[v.Width:]
		}
		if len(line) > 0 {
			result = append(result, line)
		}
	}
	return result
}

func (v *Viewport) maxLineLen() int {
	max := 0
	for _, line := range v.Lines {
		if len(line) > max {
			max = len(line)
		}
	}
	return max
}
