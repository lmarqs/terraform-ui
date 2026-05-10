package sdk

import tea "github.com/charmbracelet/bubbletea"

// Stack manages a LIFO collection of Frames. Input is always routed
// to the top frame. When a frame returns nil from Update, it is popped.
type Stack struct {
	frames []Frame
}

// NewStack creates an empty stack.
func NewStack() *Stack {
	return &Stack{}
}

// Push adds a frame to the top of the stack.
func (s *Stack) Push(f Frame) {
	s.frames = append(s.frames, f)
}

// Pop removes and returns the top frame. Returns nil if empty.
func (s *Stack) Pop() Frame {
	if len(s.frames) == 0 {
		return nil
	}
	top := s.frames[len(s.frames)-1]
	s.frames = s.frames[:len(s.frames)-1]
	return top
}

// Peek returns the top frame without removing it. Returns nil if empty.
func (s *Stack) Peek() Frame {
	if len(s.frames) == 0 {
		return nil
	}
	return s.frames[len(s.frames)-1]
}

// Depth returns the number of frames on the stack.
func (s *Stack) Depth() int {
	return len(s.frames)
}

// IsEmpty reports whether the stack has no frames.
func (s *Stack) IsEmpty() bool {
	return len(s.frames) == 0
}

// Update routes a message to the top frame.
// If the frame returns nil, it is popped (back navigation).
// If it returns a different frame, the top is replaced in-place.
func (s *Stack) Update(msg tea.Msg) tea.Cmd {
	if len(s.frames) == 0 {
		return nil
	}
	top := s.frames[len(s.frames)-1]
	result, cmd := top.Update(msg)
	if result == nil {
		s.frames = s.frames[:len(s.frames)-1]
		return cmd
	}
	if result != top {
		s.frames[len(s.frames)-1] = result
	}
	return cmd
}

// View renders the top frame's view. Returns empty string if empty.
func (s *Stack) View(width, height int) string {
	if len(s.frames) == 0 {
		return ""
	}
	return s.frames[len(s.frames)-1].View(width, height)
}

// Hints returns the top frame's hints. Returns nil if empty.
func (s *Stack) Hints() []KeyHint {
	if len(s.frames) == 0 {
		return nil
	}
	return s.frames[len(s.frames)-1].Hints()
}

// Clear removes all frames except the bottom one (root).
// If the stack is empty, this is a no-op.
func (s *Stack) Clear() {
	if len(s.frames) > 1 {
		s.frames = s.frames[:1]
	}
}

// Reset removes all frames from the stack.
func (s *Stack) Reset() {
	s.frames = s.frames[:0]
}
