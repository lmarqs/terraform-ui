package frames

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestInspectFrame_EscPops(t *testing.T) {
	f := NewInspectFrame(InspectOpts{
		Title:   "Detail",
		Address: "aws_instance.foo",
		Content: "some content",
	})

	result, _ := f.Update(keyMsg("esc"))
	if result != nil {
		t.Fatal("esc should pop inspect frame")
	}
}

func TestInspectFrame_ScrollKeys(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	f := NewInspectFrame(InspectOpts{
		Content: content,
	})

	f.View(80, 3) // initialize scroll clamping

	f.Update(keyMsg("down"))
	if f.ScrollY() != 1 {
		t.Fatalf("expected scroll 1, got %d", f.ScrollY())
	}

	f.Update(keyMsg("up"))
	if f.ScrollY() != 0 {
		t.Fatalf("expected scroll 0, got %d", f.ScrollY())
	}
}

func TestInspectFrame_ActionKeys(t *testing.T) {
	pinCalled := false
	deleteCalled := false

	f := NewInspectFrame(InspectOpts{
		Content: "details",
		Actions: []InspectAction{
			{Key: " ", Label: "pin", Handler: func() tea.Cmd { pinCalled = true; return nil }},
			{Key: "d", Label: "delete", Handler: func() tea.Cmd { deleteCalled = true; return nil }},
		},
	})

	result, _ := f.Update(keyMsg(" "))
	if result == nil {
		t.Fatal("action should not pop frame")
	}
	if !pinCalled {
		t.Fatal("space should trigger pin action")
	}

	result, _ = f.Update(keyMsg("d"))
	if result == nil {
		t.Fatal("action should not pop frame")
	}
	if !deleteCalled {
		t.Fatal("d should trigger delete action")
	}
}

func TestInspectFrame_UnknownKeysIgnored(t *testing.T) {
	f := NewInspectFrame(InspectOpts{Content: "details"})

	result, _ := f.Update(keyMsg("x"))
	if result == nil {
		t.Fatal("unknown key should not pop frame")
	}
}

func TestInspectFrame_HintsIncludeActions(t *testing.T) {
	f := NewInspectFrame(InspectOpts{
		Content: "details",
		Actions: []InspectAction{
			{Key: " ", Label: "pin"},
			{Key: "d", Label: "delete"},
		},
	})

	hints := f.Hints()
	// Base hints (Esc, ↑↓) + 2 actions = 4
	if len(hints) != 4 {
		t.Fatalf("expected 4 hints, got %d: %v", len(hints), hints)
	}
}

func TestInspectFrame_HintsPinnedIndicator(t *testing.T) {
	f := NewInspectFrame(InspectOpts{
		Content:  "details",
		IsPinned: func() bool { return true },
	})

	hints := f.Hints()
	found := false
	for _, h := range hints {
		if h.Description == "[pinned]" {
			found = true
		}
	}
	if !found {
		t.Fatal("should show [pinned] indicator when pinned")
	}
}

func TestInspectFrame_HintsNotPinned(t *testing.T) {
	f := NewInspectFrame(InspectOpts{
		Content:  "details",
		IsPinned: func() bool { return false },
	})

	hints := f.Hints()
	for _, h := range hints {
		if h.Description == "[pinned]" {
			t.Fatal("should not show [pinned] when not pinned")
		}
	}
}

func TestInspectFrame_ID(t *testing.T) {
	f := NewInspectFrame(InspectOpts{})
	if f.ID() != "inspect" {
		t.Fatalf("expected ID 'inspect', got %q", f.ID())
	}
}

func TestInspectFrame_ActionWithNilHandler(t *testing.T) {
	f := NewInspectFrame(InspectOpts{
		Content: "details",
		Actions: []InspectAction{
			{Key: "d", Label: "delete", Handler: nil},
		},
	})

	result, cmd := f.Update(keyMsg("d"))
	if result == nil {
		t.Fatal("action with nil handler should not pop frame")
	}
	if cmd != nil {
		t.Fatal("action with nil handler should not produce cmd")
	}
}

func TestInspectFrame_NonKeyMsgIgnored(t *testing.T) {
	f := NewInspectFrame(InspectOpts{Content: "details"})

	result, cmd := f.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if result != f {
		t.Fatal("non-key msg should return same frame")
	}
	if cmd != nil {
		t.Fatal("non-key msg should not produce cmd")
	}
}

func TestInspectFrame_HintsWithNilIsPinned(t *testing.T) {
	f := NewInspectFrame(InspectOpts{
		Content:  "details",
		IsPinned: nil,
	})

	hints := f.Hints()
	for _, h := range hints {
		if h.Description == "[pinned]" {
			t.Fatal("should not show [pinned] when IsPinned is nil")
		}
	}
}

func TestInspectFrame_View(t *testing.T) {
	f := NewInspectFrame(InspectOpts{
		Content: "line1\nline2",
	})

	view := f.View(80, 24)
	if view == "" {
		t.Fatal("view should not be empty")
	}
}

func TestInspectFrame_WhenGoToTopPressed_ShouldResetScrollToZero(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	f := NewInspectFrame(InspectOpts{Content: content})

	f.View(80, 3)
	f.Update(keyMsg("down"))
	f.Update(keyMsg("down"))
	if f.ScrollY() != 2 {
		t.Fatalf("expected scroll 2, got %d", f.ScrollY())
	}

	f.Update(keyMsg("g"))
	if f.ScrollY() != 0 {
		t.Fatalf("expected scroll 0 after 'g', got %d", f.ScrollY())
	}
}

func TestInspectFrame_WhenGoToBottomPressed_ShouldScrollToEnd(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	f := NewInspectFrame(InspectOpts{Content: content})

	result, cmd := f.Update(keyMsg("G"))
	if result == nil {
		t.Fatal("G should not pop frame")
	}
	if cmd != nil {
		t.Fatal("G should not produce cmd")
	}
	if f.ScrollY() != 10 {
		t.Fatalf("expected scroll 10 after 'G', got %d", f.ScrollY())
	}
}

func TestInspectFrame_WhenPanelKeyPressed_ShouldBeConsumedByPanel(t *testing.T) {
	f := NewInspectFrame(InspectOpts{Content: "line1\nline2\nline3"})

	result, cmd := f.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if result == nil {
		t.Fatal("left key should not pop frame")
	}
	if cmd != nil {
		t.Fatal("left key should not produce cmd")
	}

	result, cmd = f.Update(tea.KeyMsg{Type: tea.KeyRight})
	if result == nil {
		t.Fatal("right key should not pop frame")
	}
	if cmd != nil {
		t.Fatal("right key should not produce cmd")
	}

	result, cmd = f.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	if result == nil {
		t.Fatal("ctrl+w key should not pop frame")
	}
	if cmd != nil {
		t.Fatal("ctrl+w key should not produce cmd")
	}
}

func TestInspectFrame_WhenHeightIsZero_ShouldUseDefaultHeight(t *testing.T) {
	content := "line1\nline2\nline3"
	f := NewInspectFrame(InspectOpts{Content: content})

	view := f.View(80, 0)
	if view == "" {
		t.Fatal("view with zero height should use default and not be empty")
	}
}

func TestInspectFrame_WhenHeightIsNegative_ShouldUseDefaultHeight(t *testing.T) {
	content := "line1\nline2\nline3"
	f := NewInspectFrame(InspectOpts{Content: content})

	view := f.View(80, -5)
	if view == "" {
		t.Fatal("view with negative height should use default and not be empty")
	}
}

func TestInspectFrame_WhenScrollExceedsMax_ShouldClampToMaxScroll(t *testing.T) {
	content := "line1\nline2\nline3"
	f := NewInspectFrame(InspectOpts{Content: content})

	f.Update(keyMsg("G"))
	if f.ScrollY() != 3 {
		t.Fatalf("expected scroll 3 after G, got %d", f.ScrollY())
	}

	view := f.View(80, 5)
	if view == "" {
		t.Fatal("view should not be empty after clamping")
	}
	if f.ScrollY() != 0 {
		t.Fatalf("expected scroll clamped to 0 (3 lines, height 5), got %d", f.ScrollY())
	}
}

func TestInspectFrame_WhenEndIdxExceedsLines_ShouldClampToLineCount(t *testing.T) {
	content := "line1\nline2\nline3"
	f := NewInspectFrame(InspectOpts{Content: content})

	view := f.View(80, 10)
	if view == "" {
		t.Fatal("view should not be empty when height exceeds line count")
	}
}

func TestInspectFrame_WhenActionHandlerReturnsCmd_ShouldReturnCmd(t *testing.T) {
	expectedMsg := "test-action-result"
	f := NewInspectFrame(InspectOpts{
		Content: "details",
		Actions: []InspectAction{
			{Key: "t", Label: "taint", Handler: func() tea.Cmd {
				return func() tea.Msg { return expectedMsg }
			}},
		},
	})

	result, cmd := f.Update(keyMsg("t"))
	if result == nil {
		t.Fatal("action should not pop frame")
	}
	if cmd == nil {
		t.Fatal("action handler returning cmd should propagate")
	}
}

// Verify InspectFrame satisfies the Frame interface at compile time.
var _ sdk.Frame = (*InspectFrame)(nil)
