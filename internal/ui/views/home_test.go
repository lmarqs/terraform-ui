package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// testPlugin implements sdk.Plugin for testing NewHomeView.
type testPlugin struct {
	id          string
	name        string
	description string
	keyBinding  string
}

func (p *testPlugin) ID() string                                        { return p.id }
func (p *testPlugin) Name() string                                      { return p.name }
func (p *testPlugin) Description() string                               { return p.description }
func (p *testPlugin) KeyBinding() string                                { return p.keyBinding }
func (p *testPlugin) Init(_ *sdk.Context) tea.Cmd                    { return nil }
func (p *testPlugin) Update(_ tea.Msg) (sdk.Plugin, tea.Cmd)         { return p, nil }
func (p *testPlugin) View(_, _ int) string                              { return "" }
func (p *testPlugin) Configure(_ map[string]interface{}) error          { return nil }
func (p *testPlugin) Ready() bool                                       { return true }

// Verify testPlugin satisfies the interface at compile time.
var _ sdk.Plugin = (*testPlugin)(nil)

func makeTestPlugins() []sdk.Plugin {
	return []sdk.Plugin{
		&testPlugin{id: "plan", name: "Plan", description: "Run terraform plan", keyBinding: "p"},
		&testPlugin{id: "apply", name: "Apply", description: "Run terraform apply", keyBinding: "a"},
		&testPlugin{id: "state", name: "State", description: "Browse state", keyBinding: "s"},
	}
}

// Ensure testPlugin doesn't need sdk.Service (satisfies factory pattern).
var _ sdk.PluginFactory = func(_ sdk.Service) sdk.Plugin {
	return &testPlugin{}
}

func TestNewHomeView_GeneratesMenuItems(t *testing.T) {
	plugins := makeTestPlugins()
	view := NewHomeView(plugins)

	items := view.Items()
	if len(items) != 3 {
		t.Fatalf("NewHomeView() items length = %d, want 3", len(items))
	}

	expected := []struct {
		key   string
		label string
		desc  string
	}{
		{"p", "Plan", "Run terraform plan"},
		{"a", "Apply", "Run terraform apply"},
		{"s", "State", "Browse state"},
	}

	for i, exp := range expected {
		if items[i].Key != exp.key {
			t.Errorf("items[%d].Key = %q, want %q", i, items[i].Key, exp.key)
		}
		if items[i].Label != exp.label {
			t.Errorf("items[%d].Label = %q, want %q", i, items[i].Label, exp.label)
		}
		if items[i].Description != exp.desc {
			t.Errorf("items[%d].Description = %q, want %q", i, items[i].Description, exp.desc)
		}
	}
}

func TestNewHomeView_EmptyPlugins(t *testing.T) {
	view := NewHomeView([]sdk.Plugin{})

	items := view.Items()
	if len(items) != 0 {
		t.Errorf("NewHomeView(empty) items length = %d, want 0", len(items))
	}
}

func TestHomeView_InitialSelection(t *testing.T) {
	plugins := makeTestPlugins()
	view := NewHomeView(plugins)

	if view.Selected() != 0 {
		t.Errorf("Initial Selected() = %d, want 0", view.Selected())
	}
}

func TestHomeView_MoveDown(t *testing.T) {
	plugins := makeTestPlugins()
	view := NewHomeView(plugins)

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
	plugins := makeTestPlugins()
	view := NewHomeView(plugins)

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
	plugins := makeTestPlugins()
	view := NewHomeView(plugins)

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
	plugins := makeTestPlugins()
	view := NewHomeView(plugins)

	output := view.Render(80, 24)

	if output == "" {
		t.Fatal("Render() returned empty string")
	}

	// Should contain the title
	if !strings.Contains(output, "terraform-ui") {
		t.Error("Render() should contain the title 'terraform-ui'")
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
	plugins := makeTestPlugins()
	view := NewHomeView(plugins)

	widths := []int{40, 80, 120, 200}
	for _, w := range widths {
		output := view.Render(w, 24)
		if output == "" {
			t.Errorf("Render(width=%d) returned empty string", w)
		}
	}
}

func TestHomeView_Render_NarrowWidth(t *testing.T) {
	plugins := makeTestPlugins()
	view := NewHomeView(plugins)

	// Even with very narrow width, should produce output
	output := view.Render(10, 5)
	if output == "" {
		t.Error("Render(10, 5) returned empty string")
	}
}
