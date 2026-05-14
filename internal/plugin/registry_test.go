package plugin

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
)

// mockPlugin implements the Plugin interface for testing purposes.
type mockPlugin struct {
	id          string
	name        string
	description string
	configured  map[string]interface{}
	ready       bool
}

func (m *mockPlugin) ID() string                         { return m.id }
func (m *mockPlugin) Name() string                       { return m.name }
func (m *mockPlugin) Description() string                { return m.description }
func (m *mockPlugin) Init(_ *Context) tea.Cmd            { return nil }
func (m *mockPlugin) Update(_ tea.Msg) (Plugin, tea.Cmd) { return m, nil }
func (m *mockPlugin) View(_, _ int) string               { return "mock view" }
func (m *mockPlugin) Configure(cfg map[string]interface{}) error {
	m.configured = cfg
	return nil
}
func (m *mockPlugin) Ready() bool { return m.ready }

func newMockFactory(id, name string) PluginFactory {
	return func(_ terraform.Service) Plugin {
		return &mockPlugin{
			id:    id,
			name:  name,
			ready: true,
		}
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()

	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if len(r.plugins) != 0 {
		t.Errorf("NewRegistry().plugins length = %d, want 0", len(r.plugins))
	}
	if len(r.byKey) != 0 {
		t.Errorf("NewRegistry().byKey length = %d, want 0", len(r.byKey))
	}
	if len(r.byID) != 0 {
		t.Errorf("NewRegistry().byID length = %d, want 0", len(r.byID))
	}
	if len(r.factories) != 0 {
		t.Errorf("NewRegistry().factories length = %d, want 0", len(r.factories))
	}
}

func TestRegisterFactory(t *testing.T) {
	r := NewRegistry()

	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})
	r.RegisterFactory("state", newMockFactory("state", "State"), PluginMeta{Keybinding: "s", MenuVisible: true})

	if len(r.factories) != 2 {
		t.Errorf("factories length = %d, want 2", len(r.factories))
	}

	if _, ok := r.factories["plan"]; !ok {
		t.Error("factory 'plan' not registered")
	}
	if _, ok := r.factories["state"]; !ok {
		t.Error("factory 'state' not registered")
	}
}

func TestBuild_CreatesPlugins(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})
	r.RegisterFactory("state", newMockFactory("state", "State"), PluginMeta{Keybinding: "s", MenuVisible: true})

	r.Build(nil, nil)

	if len(r.plugins) != 2 {
		t.Fatalf("Build() plugins length = %d, want 2", len(r.plugins))
	}
}

func TestBuild_DisabledConfigSkipsPlugin(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})
	r.RegisterFactory("state", newMockFactory("state", "State"), PluginMeta{Keybinding: "s", MenuVisible: true})

	falseVal := false
	configs := map[string]config.PluginConfig{
		"state": {Enabled: &falseVal},
	}

	r.Build(nil, configs)

	if len(r.plugins) != 1 {
		t.Fatalf("Build() with disabled plugin: plugins length = %d, want 1", len(r.plugins))
	}

	if r.plugins[0].Name() != "Plan" {
		t.Errorf("Build() remaining plugin = %q, want %q", r.plugins[0].Name(), "Plan")
	}
}

func TestBuild_ConfigOptionsCallsConfigure(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})

	trueVal := true
	configs := map[string]config.PluginConfig{
		"plan": {
			Enabled: &trueVal,
			Options: map[string]interface{}{
				"refresh_interval": 30,
				"show_diffs":       true,
			},
		},
	}

	r.Build(nil, configs)

	if len(r.plugins) != 1 {
		t.Fatalf("Build() plugins length = %d, want 1", len(r.plugins))
	}

	mp := r.plugins[0].(*mockPlugin)
	if mp.configured == nil {
		t.Fatal("Configure() was not called")
	}
	if mp.configured["refresh_interval"] != 30 {
		t.Errorf("configured[refresh_interval] = %v, want 30", mp.configured["refresh_interval"])
	}
	if mp.configured["show_diffs"] != true {
		t.Errorf("configured[show_diffs] = %v, want true", mp.configured["show_diffs"])
	}
}

func TestBuild_NoConfigDefaultsToEnabled(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})

	// Empty config map means plugin is not mentioned at all
	configs := map[string]config.PluginConfig{}

	r.Build(nil, configs)

	if len(r.plugins) != 1 {
		t.Fatalf("Build() with empty configs: plugins length = %d, want 1", len(r.plugins))
	}
}

func TestAll(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})
	r.RegisterFactory("state", newMockFactory("state", "State"), PluginMeta{Keybinding: "s", MenuVisible: true})
	r.Build(nil, nil)

	all := r.All()
	if len(all) != 2 {
		t.Errorf("All() length = %d, want 2", len(all))
	}
}

func TestByKey(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})
	r.RegisterFactory("state", newMockFactory("state", "State"), PluginMeta{Keybinding: "s", MenuVisible: true})
	r.Build(nil, nil)

	// Find existing key
	p, ok := r.ByKey("p")
	if !ok {
		t.Fatal("ByKey(\"p\") not found")
	}
	if p.Name() != "Plan" {
		t.Errorf("ByKey(\"p\").Name() = %q, want %q", p.Name(), "Plan")
	}

	s, ok := r.ByKey("s")
	if !ok {
		t.Fatal("ByKey(\"s\") not found")
	}
	if s.Name() != "State" {
		t.Errorf("ByKey(\"s\").Name() = %q, want %q", s.Name(), "State")
	}

	// Key not found
	_, ok = r.ByKey("x")
	if ok {
		t.Error("ByKey(\"x\") should not be found")
	}
}

func TestByID(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})
	r.RegisterFactory("state", newMockFactory("state", "State"), PluginMeta{Keybinding: "s", MenuVisible: true})
	r.Build(nil, nil)

	// Find existing ID
	p, ok := r.ByID("plan")
	if !ok {
		t.Fatal("ByID(\"plan\") not found")
	}
	if p.Name() != "Plan" {
		t.Errorf("ByID(\"plan\").Name() = %q, want %q", p.Name(), "Plan")
	}

	// ID not found
	_, ok = r.ByID("nonexistent")
	if ok {
		t.Error("ByID(\"nonexistent\") should not be found")
	}
}

func TestByKey_EmptyKeyBinding(t *testing.T) {
	r := NewRegistry()
	// Plugin with no key binding
	r.RegisterFactory("nokey", func(_ terraform.Service) Plugin {
		return &mockPlugin{
			id:    "nokey",
			name:  "No Key",
			ready: true,
		}
	}, PluginMeta{Keybinding: "", MenuVisible: false})
	r.Build(nil, nil)

	// Should be findable by ID but not by key
	_, ok := r.ByKey("")
	if ok {
		t.Error("ByKey(\"\") should not find plugins with empty key binding")
	}

	p, ok := r.ByID("nokey")
	if !ok {
		t.Fatal("ByID(\"nokey\") should be found")
	}
	if p.Name() != "No Key" {
		t.Errorf("ByID(\"nokey\").Name() = %q, want %q", p.Name(), "No Key")
	}
}

func TestMenuItems(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("context", newMockFactory("context", "Context"), PluginMeta{Keybinding: "", MenuVisible: false})
	r.RegisterFactory("state", newMockFactory("state", "State"), PluginMeta{Keybinding: "s", MenuVisible: true})
	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})
	r.Build(nil, nil)

	items := r.MenuItems()
	if len(items) != 2 {
		t.Fatalf("MenuItems() length = %d, want 2", len(items))
	}

	// Should be in registration order
	if items[0].Key != "s" {
		t.Errorf("items[0].Key = %q, want %q", items[0].Key, "s")
	}
	if items[0].Name != "State" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "State")
	}
	if items[1].Key != "p" {
		t.Errorf("items[1].Key = %q, want %q", items[1].Key, "p")
	}
	if items[1].Name != "Plan" {
		t.Errorf("items[1].Name = %q, want %q", items[1].Name, "Plan")
	}
}

func TestMenuItems_DisabledPlugin(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("state", newMockFactory("state", "State"), PluginMeta{Keybinding: "s", MenuVisible: true})
	r.RegisterFactory("plan", newMockFactory("plan", "Plan"), PluginMeta{Keybinding: "p", MenuVisible: true})

	falseVal := false
	configs := map[string]config.PluginConfig{
		"plan": {Enabled: &falseVal},
	}
	r.Build(nil, configs)

	items := r.MenuItems()
	if len(items) != 1 {
		t.Fatalf("MenuItems() with disabled plugin: length = %d, want 1", len(items))
	}
	if items[0].Name != "State" {
		t.Errorf("items[0].Name = %q, want %q", items[0].Name, "State")
	}
}

func TestNavBehaviorFor(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("state", newMockFactory("state", "State"), PluginMeta{Keybinding: "s", MenuVisible: true, Nav: NavReplace})
	r.RegisterFactory("workspaces", newMockFactory("workspaces", "Workspaces"), PluginMeta{Keybinding: "w", MenuVisible: true, Nav: NavPush})
	r.Build(nil, nil)

	t.Run("ShouldReturnNavPushForPushPlugin", func(t *testing.T) {
		if got := r.NavBehaviorFor("workspaces"); got != NavPush {
			t.Errorf("NavBehaviorFor(\"workspaces\") = %d, want NavPush (%d)", got, NavPush)
		}
	})

	t.Run("ShouldReturnNavReplaceForReplacePlugin", func(t *testing.T) {
		if got := r.NavBehaviorFor("state"); got != NavReplace {
			t.Errorf("NavBehaviorFor(\"state\") = %d, want NavReplace (%d)", got, NavReplace)
		}
	})

	t.Run("ShouldReturnNavReplaceForUnknownPlugin", func(t *testing.T) {
		if got := r.NavBehaviorFor("nonexistent"); got != NavReplace {
			t.Errorf("NavBehaviorFor(\"nonexistent\") = %d, want NavReplace (%d)", got, NavReplace)
		}
	})
}

func TestBuild_PreservesRegistrationOrder(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("alpha", newMockFactory("alpha", "Alpha"), PluginMeta{Keybinding: "a", MenuVisible: true})
	r.RegisterFactory("beta", newMockFactory("beta", "Beta"), PluginMeta{Keybinding: "b", MenuVisible: true})
	r.RegisterFactory("gamma", newMockFactory("gamma", "Gamma"), PluginMeta{Keybinding: "g", MenuVisible: true})
	r.Build(nil, nil)

	all := r.All()
	if len(all) != 3 {
		t.Fatalf("All() length = %d, want 3", len(all))
	}
	if all[0].ID() != "alpha" {
		t.Errorf("All()[0].ID() = %q, want %q", all[0].ID(), "alpha")
	}
	if all[1].ID() != "beta" {
		t.Errorf("All()[1].ID() = %q, want %q", all[1].ID(), "beta")
	}
	if all[2].ID() != "gamma" {
		t.Errorf("All()[2].ID() = %q, want %q", all[2].ID(), "gamma")
	}
}
