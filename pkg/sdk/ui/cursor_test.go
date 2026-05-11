package ui

import "testing"

func TestCursor_MoveUpDown(t *testing.T) {
	c := NewCursor()
	c.SetCount(5)

	if c.Pos() != 0 {
		t.Fatalf("initial Pos() = %d, want 0", c.Pos())
	}

	c.MoveDown()
	if c.Pos() != 1 {
		t.Errorf("after MoveDown: Pos() = %d, want 1", c.Pos())
	}

	c.MoveDown()
	c.MoveDown()
	c.MoveDown() // pos = 4
	c.MoveDown() // should clamp at 4
	if c.Pos() != 4 {
		t.Errorf("after clamp: Pos() = %d, want 4", c.Pos())
	}

	c.MoveUp()
	if c.Pos() != 3 {
		t.Errorf("after MoveUp: Pos() = %d, want 3", c.Pos())
	}

	c.MoveToStart()
	if c.Pos() != 0 {
		t.Errorf("after MoveToStart: Pos() = %d, want 0", c.Pos())
	}
	c.MoveUp() // should clamp at 0
	if c.Pos() != 0 {
		t.Errorf("after clamp at 0: Pos() = %d, want 0", c.Pos())
	}

	c.MoveToEnd()
	if c.Pos() != 4 {
		t.Errorf("after MoveToEnd: Pos() = %d, want 4", c.Pos())
	}
}

func TestCursor_SetCount_Clamps(t *testing.T) {
	c := NewCursor()
	c.SetCount(10)
	c.MoveToEnd() // pos = 9

	c.SetCount(5) // pos should clamp to 4
	if c.Pos() != 4 {
		t.Errorf("after shrink: Pos() = %d, want 4", c.Pos())
	}

	c.SetCount(0) // pos should be 0
	if c.Pos() != 0 {
		t.Errorf("after SetCount(0): Pos() = %d, want 0", c.Pos())
	}
}

func TestCursor_VisibleWindow(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		pos       int
		viewport  int
		wantStart int
		wantEnd   int
	}{
		{
			name:      "all items fit",
			count:     5,
			pos:       2,
			viewport:  10,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "cursor at top",
			count:     20,
			pos:       0,
			viewport:  5,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "cursor in middle",
			count:     20,
			pos:       10,
			viewport:  5,
			wantStart: 8,
			wantEnd:   13,
		},
		{
			name:      "cursor near end",
			count:     20,
			pos:       19,
			viewport:  5,
			wantStart: 15,
			wantEnd:   20,
		},
		{
			name:      "empty list",
			count:     0,
			pos:       0,
			viewport:  5,
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "viewport larger than count",
			count:     3,
			pos:       1,
			viewport:  10,
			wantStart: 0,
			wantEnd:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCursor()
			c.SetCount(tt.count)
			for i := 0; i < tt.pos; i++ {
				c.MoveDown()
			}
			start, end := c.VisibleWindow(tt.viewport)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("VisibleWindow(%d) = (%d, %d), want (%d, %d)",
					tt.viewport, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestCursor_ZeroCount(t *testing.T) {
	c := NewCursor()

	// Operations on zero-count cursor should not panic
	c.MoveDown()
	c.MoveUp()
	c.MoveToEnd()
	c.MoveToStart()

	if c.Pos() != 0 {
		t.Errorf("Pos() on zero-count = %d, want 0", c.Pos())
	}
}
