package ui

// Cursor manages index-based selection with bounds checking and viewport windowing.
type Cursor struct {
	pos   int
	count int
}

// NewCursor creates a cursor at position 0 with no items.
func NewCursor() *Cursor {
	return &Cursor{}
}

// SetCount updates the total item count and clamps position if needed.
func (c *Cursor) SetCount(n int) {
	c.count = n
	c.clamp()
}

// Pos returns the current cursor position.
func (c *Cursor) Pos() int {
	return c.pos
}

// MoveUp decrements the cursor position, clamping at 0.
func (c *Cursor) MoveUp() {
	if c.pos > 0 {
		c.pos--
	}
}

// MoveDown increments the cursor position, clamping at count-1.
func (c *Cursor) MoveDown() {
	if c.count > 0 && c.pos < c.count-1 {
		c.pos++
	}
}

// MoveToStart moves the cursor to position 0.
func (c *Cursor) MoveToStart() {
	c.pos = 0
}

// MoveToEnd moves the cursor to the last item.
func (c *Cursor) MoveToEnd() {
	if c.count > 0 {
		c.pos = c.count - 1
	}
}

// VisibleWindow calculates the visible range [start, end) for a given viewport height.
// The cursor is kept within the visible window.
func (c *Cursor) VisibleWindow(viewportHeight int) (start, end int) {
	if c.count == 0 || viewportHeight <= 0 {
		return 0, 0
	}
	if c.count <= viewportHeight {
		return 0, c.count
	}

	half := viewportHeight / 2
	start = c.pos - half
	if start < 0 {
		start = 0
	}
	end = start + viewportHeight
	if end > c.count {
		end = c.count
		start = end - viewportHeight
	}
	return start, end
}

func (c *Cursor) clamp() {
	if c.count == 0 {
		c.pos = 0
		return
	}
	if c.pos >= c.count {
		c.pos = c.count - 1
	}
}
