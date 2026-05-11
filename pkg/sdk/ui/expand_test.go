package ui

import "testing"

func TestExpandSet_Toggle(t *testing.T) {
	e := NewExpandSet()

	if e.IsExpanded(0) {
		t.Error("expected index 0 to not be expanded initially")
	}

	e.Toggle(0)
	if !e.IsExpanded(0) {
		t.Error("expected index 0 to be expanded after toggle")
	}

	e.Toggle(0)
	if e.IsExpanded(0) {
		t.Error("expected index 0 to not be expanded after second toggle")
	}
}

func TestExpandSet_CollapseAll(t *testing.T) {
	e := NewExpandSet()
	e.Toggle(0)
	e.Toggle(3)
	e.Toggle(7)

	e.CollapseAll()

	if e.IsExpanded(0) || e.IsExpanded(3) || e.IsExpanded(7) {
		t.Error("expected all items to be collapsed after CollapseAll")
	}
}

func TestExpandSet_IndependentIndices(t *testing.T) {
	e := NewExpandSet()
	e.Toggle(1)
	e.Toggle(5)

	if !e.IsExpanded(1) {
		t.Error("expected 1 to be expanded")
	}
	if !e.IsExpanded(5) {
		t.Error("expected 5 to be expanded")
	}
	if e.IsExpanded(3) {
		t.Error("expected 3 to not be expanded")
	}
}
