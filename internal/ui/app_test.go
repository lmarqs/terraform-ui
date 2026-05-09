package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/terraform"
)

// mockPlugin implements plugin.Plugin for app tests.
type mockPlugin struct {
	id         string
	name       string
	key        string
	viewOutput string
	initCmd    tea.Cmd
}

func (m *mockPlugin) ID() string                                { return m.id }
func (m *mockPlugin) Name() string                              { return m.name }
func (m *mockPlugin) Description() string                       { return m.id + " description" }
func (m *mockPlugin) KeyBinding() string                        { return m.key }
func (m *mockPlugin) Init(_ *plugin.Context) tea.Cmd            { return m.initCmd }
func (m *mockPlugin) Update(_ tea.Msg) (plugin.Plugin, tea.Cmd) { return m, nil }
func (m *mockPlugin) View(_, _ int) string                      { return m.viewOutput }
func (m *mockPlugin) Configure(_ map[string]interface{}) error  { return nil }
func (m *mockPlugin) Ready() bool                               { return true }

// mockService implements terraform.Service with no-op methods for testing.
type mockService struct {
	workspace    string
	workspaceErr error
}

func (s *mockService) Plan(_ context.Context, _ []string) (*terraform.PlanSummary, error) {
	return nil, nil
}
func (s *mockService) Apply(_ context.Context, _ []string) error { return nil }
func (s *mockService) StateList(_ context.Context) ([]terraform.Resource, error) {
	return nil, nil
}
func (s *mockService) Show(_ context.Context, _ string) (string, error) { return "", nil }
func (s *mockService) Workspace(_ context.Context) (string, error) {
	return s.workspace, s.workspaceErr
}
func (s *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"default"}, nil
}
func (s *mockService) WorkspaceSelect(_ context.Context, _ string) error { return nil }
func (s *mockService) WorkspaceNew(_ context.Context, _ string) error    { return nil }
func (s *mockService) WorkspaceDelete(_ context.Context, _ string) error { return nil }
func (s *mockService) WithDir(_ string) terraform.Service                { return s }

func setupTestApp() App {
	cfg := config.Config{
		Dir:             "/test/dir",
		TerraformBinary: "terraform",
		Mode:            "progress",
	}

	svc := &mockService{workspace: "default"}

	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "plan", name: "Plan", key: "p", viewOutput: "plan view"}
	})
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", key: "s", viewOutput: "state view"}
	})
	registry.Build(nil, nil)

	return NewApp(cfg, svc, registry)
}

func TestNewApp(t *testing.T) {
	app := setupTestApp()

	if app.cfg.Dir != "/test/dir" {
		t.Errorf("NewApp().cfg.Dir = %q, want %q", app.cfg.Dir, "/test/dir")
	}
	if app.activePlugin != nil {
		t.Error("NewApp() should start with no active plugin")
	}
}

func TestApp_View_Loading(t *testing.T) {
	app := setupTestApp()
	// width and height are 0 by default
	output := app.View()
	if output != "Loading..." {
		t.Errorf("View() with no size = %q, want %q", output, "Loading...")
	}
}

func TestApp_View_WithSize(t *testing.T) {
	app := setupTestApp()
	app.width = 80
	app.height = 24

	output := app.View()
	if output == "" {
		t.Fatal("View() with size returned empty string")
	}
	if output == "Loading..." {
		t.Error("View() with size should not return 'Loading...'")
	}
}

func TestApp_Update_WindowSize(t *testing.T) {
	app := setupTestApp()

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	model, _ := app.Update(msg)
	updated := model.(App)

	if updated.width != 100 {
		t.Errorf("Update(WindowSizeMsg).width = %d, want 100", updated.width)
	}
	if updated.height != 30 {
		t.Errorf("Update(WindowSizeMsg).height = %d, want 30", updated.height)
	}
}

func TestApp_Update_WorkspaceLoaded(t *testing.T) {
	app := setupTestApp()
	app.width = 120
	app.height = 30

	msg := workspaceLoadedMsg{workspace: "staging"}
	model, _ := app.Update(msg)
	updated := model.(App)

	// Verify the workspace is reflected in the rendered output
	output := updated.View()
	if output == "" {
		t.Fatal("View() should not be empty after workspace loaded")
	}
	// The header should contain the workspace name
	if !strings.Contains(output, "staging") {
		t.Error("View() after workspaceLoadedMsg should contain workspace name 'staging'")
	}
}

func TestApp_HandleKey_Quit(t *testing.T) {
	app := setupTestApp()

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := app.Update(msg)

	if cmd == nil {
		t.Fatal("ctrl+c should produce a quit command")
	}
}

func TestApp_HandleKey_QuitFromHome(t *testing.T) {
	app := setupTestApp()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := app.Update(msg)

	if cmd == nil {
		t.Fatal("q from home should produce a quit command")
	}
}

func TestApp_HandleKey_EscFromHome(t *testing.T) {
	app := setupTestApp()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := app.Update(msg)

	// Esc from home does nothing (no plugin to return from)
	if cmd != nil {
		t.Error("esc from home should produce nil command")
	}
}

func TestApp_HandleKey_PluginActivation(t *testing.T) {
	app := setupTestApp()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ := app.Update(msg)
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("pressing 'p' should activate the plan plugin")
	}
	if updated.activePlugin.Name() != "Plan" {
		t.Errorf("active plugin = %q, want %q", updated.activePlugin.Name(), "Plan")
	}
}

func TestApp_HandleKey_EscReturnsToHome(t *testing.T) {
	app := setupTestApp()

	// Activate a plugin first
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ := app.Update(msg)
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("plugin should be active")
	}

	// Press esc to return
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = app.Update(msg)
	app = model.(App)

	if app.activePlugin != nil {
		t.Error("esc should deactivate the plugin")
	}
}

func TestApp_HandleKey_QReturnsToHome(t *testing.T) {
	app := setupTestApp()

	// Activate a plugin first
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	model, _ := app.Update(msg)
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("plugin should be active")
	}

	// Press q to return (not quit)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	model, _ = app.Update(msg)
	app = model.(App)

	if app.activePlugin != nil {
		t.Error("q with active plugin should return to home, not quit")
	}
}

func TestApp_HomeNavigation_Down(t *testing.T) {
	app := setupTestApp()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	model, _ := app.Update(msg)
	updated := model.(App)

	if updated.homeView.Selected() != 1 {
		t.Errorf("j key should move selection down, got %d", updated.homeView.Selected())
	}
}

func TestApp_HomeNavigation_Up(t *testing.T) {
	app := setupTestApp()

	// Move down first
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	model, _ := app.Update(msg)
	app = model.(App)

	// Move back up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	model, _ = app.Update(msg)
	updated := model.(App)

	if updated.homeView.Selected() != 0 {
		t.Errorf("k key should move selection up, got %d", updated.homeView.Selected())
	}
}

func TestApp_HomeNavigation_Enter(t *testing.T) {
	app := setupTestApp()

	// Move to second item and press enter
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	model, _ := app.Update(msg)
	app = model.(App)

	msg2 := tea.KeyMsg{Type: tea.KeyEnter}
	model, _ = app.Update(msg2)
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("enter should activate the selected plugin")
	}
	// The second item depends on map iteration order, just verify a plugin was activated
	name := updated.activePlugin.Name()
	if name != "Plan" && name != "State" {
		t.Errorf("active plugin = %q, want one of Plan/State", name)
	}
}

func TestApp_Init_ReturnsCmd(t *testing.T) {
	app := setupTestApp()

	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init() should return a command (for workspace loading)")
	}
}

func TestApp_LoadWorkspace_Success(t *testing.T) {
	cfg := config.Config{
		Dir:             "/test/dir",
		TerraformBinary: "terraform",
		Mode:            "progress",
	}

	svc := &mockService{workspace: "production"}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry)

	// Call loadWorkspace directly
	msg := app.loadWorkspace()
	wsMsg, ok := msg.(workspaceLoadedMsg)
	if !ok {
		t.Fatalf("loadWorkspace() returned %T, want workspaceLoadedMsg", msg)
	}
	if wsMsg.workspace != "production" {
		t.Errorf("loadWorkspace().workspace = %q, want %q", wsMsg.workspace, "production")
	}
}

func TestApp_LoadWorkspace_Error(t *testing.T) {
	cfg := config.Config{
		Dir:             "/test/dir",
		TerraformBinary: "terraform",
		Mode:            "progress",
	}

	svc := &mockService{workspace: "", workspaceErr: fmt.Errorf("connection failed")}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry)

	// Call loadWorkspace directly - error should return "default"
	msg := app.loadWorkspace()
	wsMsg, ok := msg.(workspaceLoadedMsg)
	if !ok {
		t.Fatalf("loadWorkspace() returned %T, want workspaceLoadedMsg", msg)
	}
	if wsMsg.workspace != "default" {
		t.Errorf("loadWorkspace() on error: workspace = %q, want %q", wsMsg.workspace, "default")
	}
}

func TestApp_Init_WithPluginInitCmd(t *testing.T) {
	cfg := config.Config{
		Dir:             "/test/dir",
		TerraformBinary: "terraform",
		Mode:            "progress",
	}

	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{
			id: "plan", name: "Plan", key: "p", viewOutput: "plan view",
			initCmd: func() tea.Msg { return customMsg{} },
		}
	})
	registry.Build(nil, nil)

	app := NewApp(cfg, nil, registry)
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init() should return a batch command including plugin init")
	}
}

// customMsg is a tea.Msg that doesn't match any case in Update.
type customMsg struct{}

func TestApp_DelegateNonKeyMsgToActivePlugin(t *testing.T) {
	app := setupTestApp()

	// Activate a plugin
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ := app.Update(msg)
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("plugin should be active")
	}

	// Send a custom message (not KeyMsg, not WindowSizeMsg, not workspaceLoadedMsg)
	// This should fall through to the active plugin delegation path
	model, _ = app.Update(customMsg{})
	updated := model.(App)

	// Plugin should still be active
	if updated.activePlugin == nil {
		t.Error("plugin should still be active after custom message delegation")
	}
}

func TestApp_NonKeyMsgWithNoActivePlugin(t *testing.T) {
	app := setupTestApp()

	// Send a custom message with no active plugin
	model, cmd := app.Update(customMsg{})
	updated := model.(App)

	// Should return with no command and no active plugin
	if updated.activePlugin != nil {
		t.Error("no plugin should be active")
	}
	if cmd != nil {
		t.Error("cmd should be nil for unhandled message with no active plugin")
	}
}

func TestApp_ActivePluginKeyDelegation(t *testing.T) {
	app := setupTestApp()

	// Activate a plugin
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ := app.Update(msg)
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("plugin should be active")
	}

	// Send a key that is not q/esc/ctrl+c - should be delegated to active plugin
	unknownKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	model, _ = app.Update(unknownKey)
	updated := model.(App)

	// Plugin should still be active (mock plugin does not deactivate)
	if updated.activePlugin == nil {
		t.Error("plugin should still be active after unknown key delegation")
	}
}

func TestApp_View_ActivePlugin(t *testing.T) {
	app := setupTestApp()
	app.width = 80
	app.height = 24

	// Activate a plugin
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ := app.Update(msg)
	app = model.(App)

	output := app.View()
	if output == "" {
		t.Fatal("View() with active plugin returned empty string")
	}
	// Should contain the plugin's view output
	if !strings.Contains(output, "plan view") {
		t.Error("View() should contain the active plugin's view output")
	}
}
