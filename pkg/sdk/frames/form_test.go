package frames

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

var _ sdk.Frame = (*FormFrame)(nil)

func TestFormFrame_ID(t *testing.T) {
	f := NewFormFrame(FormOpts{})
	if f.ID() != "form" {
		t.Fatalf("expected ID 'form', got %q", f.ID())
	}
}

func TestFormFrame_WhenCreated_ShouldStartCursorAtFirstSelectable(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "Name", Value: func() string { return "foo" }, Selectable: false},
			{Label: "Type", Value: func() string { return "bar" }, Selectable: true},
			{Label: "Region", Value: func() string { return "us-east-1" }, Selectable: true},
		},
	})

	if f.cursor != 1 {
		t.Fatalf("expected cursor at 1 (first selectable), got %d", f.cursor)
	}
}

func TestFormFrame_WhenCreated_WithNoSelectableFields_ShouldHaveCursorNegative(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "Name", Value: func() string { return "foo" }, Selectable: false},
			{Label: "Type", Value: func() string { return "bar" }, Selectable: false},
		},
	})

	if f.cursor != -1 {
		t.Fatalf("expected cursor at -1 (no selectable), got %d", f.cursor)
	}
}

func TestFormFrame_WhenCreated_WithNoFields_ShouldHaveCursorNegative(t *testing.T) {
	f := NewFormFrame(FormOpts{})

	if f.cursor != -1 {
		t.Fatalf("expected cursor at -1 (no fields), got %d", f.cursor)
	}
}

func TestFormFrame_WhenNavigatingDown_ShouldSkipNonSelectableFields(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
			{Label: "B", Value: func() string { return "b" }, Selectable: false},
			{Label: "C", Value: func() string { return "c" }, Selectable: false},
			{Label: "D", Value: func() string { return "d" }, Selectable: true},
		},
	})

	if f.cursor != 0 {
		t.Fatalf("expected initial cursor at 0, got %d", f.cursor)
	}

	f.Update(keyMsg("down"))
	if f.cursor != 3 {
		t.Fatalf("expected cursor at 3 (skipped non-selectable), got %d", f.cursor)
	}
}

func TestFormFrame_WhenNavigatingDown_WithJ_ShouldMove(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
			{Label: "B", Value: func() string { return "b" }, Selectable: true},
		},
	})

	f.Update(keyMsg("j"))
	if f.cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", f.cursor)
	}
}

func TestFormFrame_WhenNavigatingUp_ShouldSkipNonSelectableFields(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
			{Label: "B", Value: func() string { return "b" }, Selectable: false},
			{Label: "C", Value: func() string { return "c" }, Selectable: false},
			{Label: "D", Value: func() string { return "d" }, Selectable: true},
		},
	})

	// Move cursor to D (index 3)
	f.Update(keyMsg("down"))
	if f.cursor != 3 {
		t.Fatalf("expected cursor at 3 after down, got %d", f.cursor)
	}

	f.Update(keyMsg("up"))
	if f.cursor != 0 {
		t.Fatalf("expected cursor at 0 (skipped non-selectable), got %d", f.cursor)
	}
}

func TestFormFrame_WhenNavigatingUp_WithK_ShouldMove(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
			{Label: "B", Value: func() string { return "b" }, Selectable: true},
		},
	})

	f.Update(keyMsg("down"))
	f.Update(keyMsg("k"))
	if f.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", f.cursor)
	}
}

func TestFormFrame_WhenAtTopBound_ShouldNotMoveUp(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
			{Label: "B", Value: func() string { return "b" }, Selectable: true},
		},
	})

	f.Update(keyMsg("up"))
	if f.cursor != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", f.cursor)
	}
}

func TestFormFrame_WhenAtBottomBound_ShouldNotMoveDown(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
			{Label: "B", Value: func() string { return "b" }, Selectable: true},
		},
	})

	f.Update(keyMsg("down"))
	if f.cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", f.cursor)
	}

	f.Update(keyMsg("down"))
	if f.cursor != 1 {
		t.Fatalf("expected cursor to stay at 1 (at bottom), got %d", f.cursor)
	}
}

func TestFormFrame_WhenEnterOnSelectable_ShouldCallOnSelect(t *testing.T) {
	called := false
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true, OnSelect: func() tea.Cmd {
				called = true
				return nil
			}},
		},
	})

	result, _ := f.Update(keyMsg("enter"))
	if result == nil {
		t.Fatal("enter should not pop form frame")
	}
	if !called {
		t.Fatal("enter should call OnSelect")
	}
}

func TestFormFrame_WhenEnterOnSelectable_ShouldReturnCmd(t *testing.T) {
	type testMsg struct{}
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true, OnSelect: func() tea.Cmd {
				return func() tea.Msg { return testMsg{} }
			}},
		},
	})

	_, cmd := f.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal("enter should return cmd from OnSelect")
	}
	msg := cmd()
	if _, ok := msg.(testMsg); !ok {
		t.Fatal("expected testMsg from cmd")
	}
}

func TestFormFrame_WhenEnterOnNonSelectable_ShouldDoNothing(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: false},
		},
	})
	// cursor is -1 since no selectable fields

	result, cmd := f.Update(keyMsg("enter"))
	if result == nil {
		t.Fatal("enter on non-selectable should not pop")
	}
	if cmd != nil {
		t.Fatal("enter on non-selectable should not produce cmd")
	}
}

func TestFormFrame_WhenEnterOnSelectableWithNilOnSelect_ShouldDoNothing(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true, OnSelect: nil},
		},
	})

	result, cmd := f.Update(keyMsg("enter"))
	if result == nil {
		t.Fatal("enter with nil OnSelect should not pop")
	}
	if cmd != nil {
		t.Fatal("enter with nil OnSelect should not produce cmd")
	}
}

func TestFormFrame_WhenEsc_ShouldPop(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
		},
	})

	result, cmd := f.Update(keyMsg("esc"))
	if result != nil {
		t.Fatal("esc should pop form frame (return nil)")
	}
	if cmd != nil {
		t.Fatal("esc should not produce cmd")
	}
}

func TestFormFrame_WhenQ_ShouldNotPop(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
		},
	})

	result, _ := f.Update(keyMsg("q"))
	if result == nil {
		t.Fatal("q should not pop form frame (app handles q globally)")
	}
}

func TestFormFrame_WhenNonKeyMsg_ShouldIgnore(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
		},
	})

	result, cmd := f.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if result != f {
		t.Fatal("non-key msg should return same frame")
	}
	if cmd != nil {
		t.Fatal("non-key msg should not produce cmd")
	}
}

func TestFormFrame_WhenUnhandledKey_ShouldDoNothing(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "a" }, Selectable: true},
		},
	})

	result, cmd := f.Update(keyMsg("x"))
	if result != f {
		t.Fatal("unhandled key should return same frame")
	}
	if cmd != nil {
		t.Fatal("unhandled key should not produce cmd")
	}
}

func TestFormFrame_View_ShouldRenderAllFields(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "Name", Value: func() string { return "web-server" }, Selectable: false},
			{Label: "Region", Value: func() string { return "us-east-1" }, Selectable: true},
			{Label: "Type", Value: func() string { return "t3.micro" }, Selectable: true},
		},
	})

	view := f.View(80, 24)
	if !strings.Contains(view, "Name") {
		t.Fatal("view should contain field label 'Name'")
	}
	if !strings.Contains(view, "web-server") {
		t.Fatal("view should contain field value 'web-server'")
	}
	if !strings.Contains(view, "Region") {
		t.Fatal("view should contain field label 'Region'")
	}
	if !strings.Contains(view, "us-east-1") {
		t.Fatal("view should contain field value 'us-east-1'")
	}
	if !strings.Contains(view, "Type") {
		t.Fatal("view should contain field label 'Type'")
	}
	if !strings.Contains(view, "t3.micro") {
		t.Fatal("view should contain field value 't3.micro'")
	}
}

func TestFormFrame_View_ShouldShowCursorOnSelectedField(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "val-a" }, Selectable: true},
			{Label: "B", Value: func() string { return "val-b" }, Selectable: true},
		},
	})

	view := f.View(80, 24)
	lines := strings.Split(view, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	// First field (selected) should have cursor indicator
	if !strings.Contains(lines[0], "▸") {
		t.Fatal("selected field should have cursor indicator '▸'")
	}
}

func TestFormFrame_View_ShouldShowSelectableIndicator(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "A", Value: func() string { return "val" }, Selectable: true},
			{Label: "B", Value: func() string { return "val" }, Selectable: false},
		},
	})

	view := f.View(80, 24)
	// The view should contain the selectable suffix indicator (▸) for selectable fields
	// Count occurrences - should appear for the cursor and for the selectable indicator
	if !strings.Contains(view, "▸") {
		t.Fatal("view should contain selectable indicator")
	}
}

func TestFormFrame_View_WhenActionField_ShouldRenderWithActionStyle(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "Name", Value: func() string { return "web-server" }, Selectable: false},
			{Label: "Submit", Value: func() string { return "Apply Changes" }, Selectable: true, IsAction: true},
		},
	})

	view := f.View(80, 24)
	if !strings.Contains(view, "Apply Changes") {
		t.Fatal("view should contain action field value")
	}
}

func TestFormFrame_View_WhenActionFieldSelected_ShouldHighlight(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "Submit", Value: func() string { return "Apply Changes" }, Selectable: true, IsAction: true},
		},
	})

	// cursor starts at 0 (first selectable = the action field)
	view := f.View(80, 24)
	if !strings.Contains(view, "▸") {
		t.Fatal("selected action field should show cursor indicator")
	}
	if !strings.Contains(view, "Apply Changes") {
		t.Fatal("selected action field should show its value")
	}
}

func TestFormFrame_View_WhenActionFieldNotSelected_ShouldShowFaint(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "Region", Value: func() string { return "us-east-1" }, Selectable: true},
			{Label: "Submit", Value: func() string { return "Apply" }, Selectable: true, IsAction: true},
		},
	})

	// cursor starts at 0 (Region), Submit is at 1 (not selected)
	view := f.View(80, 24)
	if !strings.Contains(view, "Apply") {
		t.Fatal("non-selected action field should still show its value")
	}
}

func TestFormFrame_View_WhenActionFollowsDataField_ShouldInsertBlankLine(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "Name", Value: func() string { return "web" }, Selectable: false},
			{Label: "Submit", Value: func() string { return "Apply" }, Selectable: true, IsAction: true},
		},
	})

	view := f.View(80, 24)
	if !strings.Contains(view, "\n\n") {
		t.Fatal("action field following a data field should be preceded by a blank line")
	}
}

func TestFormFrame_View_WhenMultipleActions_ShouldNotInsertExtraBlankLines(t *testing.T) {
	f := NewFormFrame(FormOpts{
		Fields: []FormField{
			{Label: "Name", Value: func() string { return "web" }, Selectable: false},
			{Label: "Submit", Value: func() string { return "Submit" }, Selectable: true, IsAction: true},
			{Label: "Cancel", Value: func() string { return "Cancel" }, Selectable: true, IsAction: true},
		},
	})

	view := f.View(80, 24)
	// Between two action fields, no extra blank line should be inserted
	lines := strings.Split(view, "\n")
	blankCount := 0
	for _, line := range lines {
		if line == "" {
			blankCount++
		}
	}
	// Only one blank line: between data and first action
	if blankCount > 2 {
		t.Errorf("expected at most 2 blank lines (separator + trailing), got %d in:\n%s", blankCount, view)
	}
}

func TestFormFrame_Hints(t *testing.T) {
	f := NewFormFrame(FormOpts{})
	hints := f.Hints()
	if len(hints) != 3 {
		t.Fatalf("expected 3 hints, got %d", len(hints))
	}
	if hints[0].Key != "↑↓" || hints[0].Description != "navigate" {
		t.Fatalf("unexpected first hint: %v", hints[0])
	}
	if hints[1].Key != "Enter" || hints[1].Description != "select" {
		t.Fatalf("unexpected second hint: %v", hints[1])
	}
	if hints[2].Key != "Esc" || hints[2].Description != "cancel" {
		t.Fatalf("unexpected third hint: %v", hints[2])
	}
}
