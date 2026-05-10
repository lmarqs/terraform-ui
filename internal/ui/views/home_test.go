package views

import (
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/plugin"
)

func makeTestItems() []plugin.MenuItem {
	return []plugin.MenuItem{
		{Key: "p", Name: "Plan", Description: "Run terraform plan"},
		{Key: "a", Name: "Apply", Description: "Run terraform apply"},
		{Key: "s", Name: "State", Description: "Browse state"},
	}
}

func TestNewHomeView_GeneratesMenuItems(t *testing.T) {
	items := makeTestItems()
	view := NewHomeView(items)

	got := view.Items()
	if len(got) != 3 {
		t.Fatalf("NewHomeView() items length = %d, want 3", len(got))
	}

	expected := []struct {
		key  string
		name string
		desc string
	}{
		{"p", "Plan", "Run terraform plan"},
		{"a", "Apply", "Run terraform apply"},
		{"s", "State", "Browse state"},
	}

	for i, exp := range expected {
		if got[i].Key != exp.key {
			t.Errorf("items[%d].Key = %q, want %q", i, got[i].Key, exp.key)
		}
		if got[i].Name != exp.name {
			t.Errorf("items[%d].Name = %q, want %q", i, got[i].Name, exp.name)
		}
		if got[i].Description != exp.desc {
			t.Errorf("items[%d].Description = %q, want %q", i, got[i].Description, exp.desc)
		}
	}
}

func TestNewHomeView_EmptyItems(t *testing.T) {
	view := NewHomeView([]plugin.MenuItem{})

	items := view.Items()
	if len(items) != 0 {
		t.Errorf("NewHomeView(empty) items length = %d, want 0", len(items))
	}
}

func TestHomeView_InitialSelection(t *testing.T) {
	items := makeTestItems()
	view := NewHomeView(items)

	if view.Selected() != 0 {
		t.Errorf("Initial Selected() = %d, want 0", view.Selected())
	}
}

func TestHomeView_MoveDown(t *testing.T) {
	items := makeTestItems()
	view := NewHomeView(items)

	view = view.MoveDown()
	if view.Selected() != 1 {
		t.Errorf("MoveDown() Selected() = %d, want 1", view.Selected())
	}

	view = view.MoveDown()
	if view.Selected() != 2 {
		t.Errorf("MoveDown() Selected() = %d, want 2", view.Selected())
	}

	// Should not go past the last item
	view = view.MoveDown()
	if view.Selected() != 2 {
		t.Errorf("MoveDown() past end Selected() = %d, want 2", view.Selected())
	}
}

func TestHomeView_MoveUp(t *testing.T) {
	items := makeTestItems()
	view := NewHomeView(items)

	// Move down first, then back up
	view = view.MoveDown()
	view = view.MoveDown()
	view = view.MoveUp()
	if view.Selected() != 1 {
		t.Errorf("MoveUp() Selected() = %d, want 1", view.Selected())
	}

	view = view.MoveUp()
	if view.Selected() != 0 {
		t.Errorf("MoveUp() Selected() = %d, want 0", view.Selected())
	}

	// Should not go below 0
	view = view.MoveUp()
	if view.Selected() != 0 {
		t.Errorf("MoveUp() past start Selected() = %d, want 0", view.Selected())
	}
}

func TestHomeView_SelectedItem(t *testing.T) {
	items := makeTestItems()
	view := NewHomeView(items)

	item := view.SelectedItem()
	if item.Key != "p" {
		t.Errorf("SelectedItem().Key = %q, want %q", item.Key, "p")
	}

	view = view.MoveDown()
	item = view.SelectedItem()
	if item.Key != "a" {
		t.Errorf("SelectedItem().Key after MoveDown = %q, want %q", item.Key, "a")
	}

	view = view.MoveDown()
	item = view.SelectedItem()
	if item.Key != "s" {
		t.Errorf("SelectedItem().Key after 2x MoveDown = %q, want %q", item.Key, "s")
	}
}

func TestHomeView_Render(t *testing.T) {
	items := makeTestItems()
	view := NewHomeView(items)

	output := view.Render(80, 24)

	if output == "" {
		t.Fatal("Render() returned empty string")
	}

	// Should contain plugin names
	if !strings.Contains(output, "Plan") {
		t.Error("Render() should contain 'Plan'")
	}
	if !strings.Contains(output, "Apply") {
		t.Error("Render() should contain 'Apply'")
	}
	if !strings.Contains(output, "State") {
		t.Error("Render() should contain 'State'")
	}

	// Should contain key bindings
	if !strings.Contains(output, "p") {
		t.Error("Render() should contain key binding 'p'")
	}

	// Should contain the hint text
	if !strings.Contains(output, "Press a key") {
		t.Error("Render() should contain hint text")
	}
}

func TestHomeView_Render_DifferentWidths(t *testing.T) {
	items := makeTestItems()
	view := NewHomeView(items)

	widths := []int{40, 80, 120, 200}
	for _, w := range widths {
		output := view.Render(w, 24)
		if output == "" {
			t.Errorf("Render(width=%d) returned empty string", w)
		}
	}
}

func TestHomeView_Render_NarrowWidth(t *testing.T) {
	items := makeTestItems()
	view := NewHomeView(items)

	// Even with very narrow width, should produce output
	output := view.Render(10, 5)
	if output == "" {
		t.Error("Render(10, 5) returned empty string")
	}
}
