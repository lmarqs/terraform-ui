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
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// mockPlugin implements plugin.Plugin for app tests.
type mockPlugin struct {
	id         string
	name       string
	viewOutput string
	initCmd    tea.Cmd
}

func (m *mockPlugin) ID() string                                { return m.id }
func (m *mockPlugin) Name() string                              { return m.name }
func (m *mockPlugin) Description() string                       { return m.id + " description" }
func (m *mockPlugin) Init(_ *plugin.Context) tea.Cmd            { return m.initCmd }
func (m *mockPlugin) Update(_ tea.Msg) (plugin.Plugin, tea.Cmd) { return m, nil }
func (m *mockPlugin) View(_, _ int) string                      { return m.viewOutput }
func (m *mockPlugin) Configure(_ map[string]interface{}) error  { return nil }
func (m *mockPlugin) Ready() bool                               { return true }

// mockBusyPlugin implements plugin.Plugin and sdk.Busy for testing quit guards.
type mockBusyPlugin struct {
	mockPlugin
	busy bool
}

func (m *mockBusyPlugin) Busy() bool { return m.busy }

// mockService implements terraform.Service with no-op methods for testing.
type mockService struct {
	workspace    string
	workspaceErr error
}

func (s *mockService) Plan(_ context.Context, _ sdk.PlanOptions) (*terraform.PlanSummary, error) {
	return nil, nil
}
func (s *mockService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (s *mockService) StateList(_ context.Context, _ ...sdk.StateListOption) ([]terraform.Resource, error) {
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
func (s *mockService) WorkspaceNew(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	return nil
}
func (s *mockService) WorkspaceDelete(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	return nil
}
func (s *mockService) StateRm(_ context.Context, _ string) error            { return nil }
func (s *mockService) StateMove(_ context.Context, _, _ string) error       { return nil }
func (s *mockService) Import(_ context.Context, _, _ string) error          { return nil }
func (s *mockService) Taint(_ context.Context, _ string) error              { return nil }
func (s *mockService) Untaint(_ context.Context, _ string) error            { return nil }
func (s *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error) { return nil, nil }
func (s *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}
func (s *mockService) Refresh(_ context.Context) error                     { return nil }
func (s *mockService) Init(_ context.Context, _ sdk.InitOptions) error     { return nil }
func (s *mockService) ForceUnlock(_ context.Context, _ string) error       { return nil }
func (s *mockService) Version(_ context.Context) (*sdk.VersionInfo, error) { return nil, nil }
func (s *mockService) WithDir(_ string) terraform.Service                  { return s }

func setupTestApp() App {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	svc := &mockService{workspace: "default"}

	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	return NewApp(cfg, svc, registry, nil)
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

func TestApp_HandleKey_EscDelegatesToPlugin(t *testing.T) {
	app := setupTestApp()

	// Activate a plugin first
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ := app.Update(msg)
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("plugin should be active")
	}

	// Press esc — delegates to plugin (mock doesn't deactivate)
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = app.Update(msg)
	app = model.(App)

	if app.activePlugin == nil {
		t.Error("esc should delegate to plugin, plugin should still be active")
	}
}

func TestApp_DeactivateMsg(t *testing.T) {
	app := setupTestApp()

	// Activate a plugin first
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model, _ := app.Update(msg)
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("plugin should be active")
	}

	// Receive DeactivateMsg
	model, _ = app.Update(sdk.DeactivateMsg{})
	app = model.(App)

	if app.activePlugin != nil {
		t.Error("DeactivateMsg should deactivate the plugin")
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
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	svc := &mockService{workspace: "production"}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

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
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	svc := &mockService{workspace: "", workspaceErr: fmt.Errorf("connection failed")}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

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
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{
			id: "plan", name: "Plan", viewOutput: "plan view",
			initCmd: func() tea.Msg { return customMsg{} },
		}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.Build(nil, nil)

	app := NewApp(cfg, nil, registry, nil)
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init() should return a batch command including plugin init")
	}
}

func TestApp_OpenContextOnStartup_ActivatesChdirPlugin(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	svc := &mockService{workspace: "default"}

	registry := plugin.NewRegistry()
	registry.RegisterFactory("chdir", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "chdir", name: "Chdir", viewOutput: "chdir view"}
	}, plugin.PluginMeta{MenuVisible: false})
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)

	model, _ := app.Update(openContextOnStartupMsg{})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("openContextOnStartupMsg should activate chdir plugin")
	}
	if updated.activePlugin.ID() != "chdir" {
		t.Errorf("active plugin = %q, want %q", updated.activePlugin.ID(), "chdir")
	}
}

func TestApp_OpenContextOnStartup_SkipsWhenChdirSet(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Chdir:     "modules/vpc",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	svc := &mockService{workspace: "default"}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("chdir", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "chdir", name: "Chdir", viewOutput: "chdir view"}
	}, plugin.PluginMeta{MenuVisible: false})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)

	model, _ := app.Update(openContextOnStartupMsg{})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("openContextOnStartupMsg should not activate chdir when Chdir is set")
	}
}

func TestApp_OpenContextOnStartup_SkipsWhenPreloadedData(t *testing.T) {
	cfg := config.Config{
		Dir:           "/test/dir",
		PreloadedData: true,
		Terraform:     config.TerraformConfig{Bin: "terraform"},
	}

	svc := &mockService{workspace: "default"}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("chdir", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "chdir", name: "Chdir", viewOutput: "chdir view"}
	}, plugin.PluginMeta{MenuVisible: false})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)

	model, _ := app.Update(openContextOnStartupMsg{})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("openContextOnStartupMsg should not activate chdir when PreloadedData is true")
	}
}

func TestApp_ChdirChangedEvent_DeactivatesPlugin(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	svc := &mockService{workspace: "default"}

	registry := plugin.NewRegistry()
	registry.RegisterFactory("chdir", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "chdir", name: "Chdir", viewOutput: "chdir view"}
	}, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)

	// Simulate chdir plugin being active (as if startup activated it)
	model, _ := app.Update(openContextOnStartupMsg{})
	app = model.(App)
	if app.activePlugin == nil {
		t.Fatal("precondition: chdir plugin should be active")
	}

	// ChdirChangedEvent should deactivate the plugin and return to home
	model, _ = app.Update(sdk.ChdirChangedEvent{RelPath: "modules/vpc", AbsPath: "/test/dir/modules/vpc", Count: 2})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Errorf("ChdirChangedEvent should deactivate plugin, got %q", updated.activePlugin.ID())
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

func TestApp_CommandMode_ColonEnters(t *testing.T) {
	app := setupTestApp()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}
	model, _ := app.Update(msg)
	app = model.(App)

	if !app.commandMode {
		t.Error(": should enter command mode")
	}
	if app.commandInput != "" {
		t.Errorf("commandInput = %q, want empty", app.commandInput)
	}
}

func TestApp_CommandMode_TypingAndEnter(t *testing.T) {
	app := setupTestApp()

	// Enter command mode
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)

	// Type "state"
	for _, ch := range "state" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	if app.commandInput != "state" {
		t.Errorf("commandInput = %q, want %q", app.commandInput, "state")
	}

	// Press enter
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	if app.commandMode {
		t.Error("enter should exit command mode")
	}
	if app.activePlugin == nil {
		t.Fatal("enter with 'state' should activate state plugin")
	}
	if app.activePlugin.ID() != "state" {
		t.Errorf("active plugin = %q, want %q", app.activePlugin.ID(), "state")
	}
}

func TestApp_CommandMode_PrefixMatch(t *testing.T) {
	app := setupTestApp()

	// Enter command mode and type "st"
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	for _, ch := range "st" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	// Enter should auto-complete to "state" (only match)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("'st' + enter should activate state plugin via prefix match")
	}
	if app.activePlugin.ID() != "state" {
		t.Errorf("active plugin = %q, want %q", app.activePlugin.ID(), "state")
	}
}

func TestApp_CommandMode_TabAutocomplete(t *testing.T) {
	app := setupTestApp()

	// Enter command mode and type "st"
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	for _, ch := range "st" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	// Tab should complete to "state"
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = model.(App)

	if app.commandInput != "state" {
		t.Errorf("after tab: commandInput = %q, want %q", app.commandInput, "state")
	}
}

func TestApp_CommandMode_EscCancels(t *testing.T) {
	app := setupTestApp()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = model.(App)

	if app.commandMode {
		t.Error("esc should exit command mode")
	}
	if app.commandInput != "" {
		t.Errorf("esc should clear input, got %q", app.commandInput)
	}
}

func TestApp_CommandMode_Quit(t *testing.T) {
	app := setupTestApp()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = model.(App)

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal(":q should produce a quit command")
	}
}

func TestApp_CommandMode_ForceQuit(t *testing.T) {
	app := setupTestApp()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	for _, ch := range "q!" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal(":q! should produce a quit command")
	}
}

func TestApp_CommandMode_ColonFromActivePlugin(t *testing.T) {
	app := setupTestApp()

	// Activate plan plugin
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	app = model.(App)
	if app.activePlugin == nil || app.activePlugin.ID() != "plan" {
		t.Fatal("p should activate plan plugin")
	}

	// : should enter command mode even with active plugin
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)

	if !app.commandMode {
		t.Error(": with active plugin should still enter command mode")
	}

	// Type "state" and enter to switch
	for _, ch := range "state" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "state" {
		t.Errorf("should switch to state, got %v", app.activePlugin)
	}
}

func setupTestAppWithBusyPlugin(busy bool) App {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	svc := &mockService{workspace: "default"}

	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockBusyPlugin{
			mockPlugin: mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"},
			busy:       busy,
		}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	return NewApp(cfg, svc, registry, nil)
}

func TestApp_CommandMode_QuitBlockedWhenBusy(t *testing.T) {
	app := setupTestAppWithBusyPlugin(true)

	// Enter command mode, type :q, press enter
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = model.(App)

	var cmd tea.Cmd
	model, cmd = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	if cmd != nil {
		t.Fatal(":q should NOT produce quit command when a plugin is busy")
	}
	if app.commandError == "" {
		t.Fatal(":q should set commandError when blocked")
	}
}

func TestApp_CommandMode_ForceQuitBypassesBusy(t *testing.T) {
	app := setupTestAppWithBusyPlugin(true)

	// Enter command mode, type :q!, press enter
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	for _, ch := range "q!" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal(":q! should produce quit command even when a plugin is busy")
	}
}

func TestApp_CommandError_ClearedOnKeypress(t *testing.T) {
	app := setupTestAppWithBusyPlugin(true)

	// Trigger the error via :q
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	if app.commandError == "" {
		t.Fatal("commandError should be set")
	}

	// Any keypress should clear the error
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	app = model.(App)

	if app.commandError != "" {
		t.Errorf("commandError should be cleared on keypress, got %q", app.commandError)
	}
}

func TestApp_HandleKey_QuitFromHomeBlockedWhenBusy(t *testing.T) {
	app := setupTestAppWithBusyPlugin(true)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	model, cmd := app.Update(msg)
	app = model.(App)

	if cmd != nil {
		t.Fatal("q from home should NOT quit when a plugin is busy")
	}
	if app.commandError == "" {
		t.Fatal("q from home should set commandError when blocked")
	}
}

// --- Navigate-back tests ---

func setupTestAppWithTransientPlugins() App {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	svc := &mockService{workspace: "default"}

	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("chdir", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "chdir", name: "Chdir", viewOutput: "chdir view"}
	}, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("workspace", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "workspace", name: "Workspace", viewOutput: "workspace view"}
	}, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("context", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "context", name: "Context", viewOutput: "context view"}
	}, plugin.PluginMeta{MenuVisible: false})
	registry.Build(nil, nil)

	return NewApp(cfg, svc, registry, nil)
}

func TestApp_ChdirSelection_NavigatesBackToPreviousPlugin(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Activate "state" plugin via keybinding
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "state" {
		t.Fatal("precondition: state plugin should be active")
	}

	// Switch to "chdir" via command mode (simulates :chdir)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	for _, ch := range "chdir" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "chdir" {
		t.Fatal("precondition: chdir plugin should be active after :chdir command")
	}

	// Send ChdirChangedEvent (simulates user selecting a directory)
	model, _ = app.Update(sdk.ChdirChangedEvent{RelPath: "modules/vpc", AbsPath: "/test/dir/modules/vpc", Count: 2})
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("after ChdirChangedEvent, should navigate back to previous plugin, not home")
	}
	if app.activePlugin.ID() != "state" {
		t.Errorf("after ChdirChangedEvent, activePlugin = %q, want %q", app.activePlugin.ID(), "state")
	}
}

func TestApp_ChdirSelection_NavigatesToHomeWhenNoPrevious(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Activate chdir directly (no previous plugin, simulates startup)
	model, _ := app.Update(openContextOnStartupMsg{})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "chdir" {
		t.Fatal("precondition: chdir plugin should be active")
	}

	// Send ChdirChangedEvent with no previous plugin set
	model, _ = app.Update(sdk.ChdirChangedEvent{RelPath: "modules/vpc", AbsPath: "/test/dir/modules/vpc", Count: 2})
	app = model.(App)

	if app.activePlugin != nil {
		t.Errorf("after ChdirChangedEvent with no previous plugin, activePlugin should be nil (home), got %q", app.activePlugin.ID())
	}
}

func TestApp_WorkspaceSelection_NavigatesBackToPreviousPlugin(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Activate "state" plugin via keybinding
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "state" {
		t.Fatal("precondition: state plugin should be active")
	}

	// Switch to "workspace" via command mode
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	for _, ch := range "workspace" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "workspace" {
		t.Fatal("precondition: workspaces plugin should be active after :workspaces command")
	}

	// Send WorkspaceChangedEvent (simulates user selecting a workspace)
	model, _ = app.Update(sdk.WorkspaceChangedEvent{Name: "staging"})
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("after WorkspaceChangedEvent, should navigate back to previous plugin, not home")
	}
	if app.activePlugin.ID() != "state" {
		t.Errorf("after WorkspaceChangedEvent, activePlugin = %q, want %q", app.activePlugin.ID(), "state")
	}
}

func TestApp_WorkspaceSelection_FromContextDoesNotNavigateBack(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Activate "context" plugin directly
	if p, ok := app.registry.ByID("context"); ok {
		app.activePlugin = p
	}

	if app.activePlugin == nil || app.activePlugin.ID() != "context" {
		t.Fatal("precondition: context plugin should be active")
	}

	// Send WorkspaceChangedEvent while context is active
	model, _ := app.Update(sdk.WorkspaceChangedEvent{Name: "staging"})
	app = model.(App)

	// The context plugin should remain active (no navigate-back for context)
	if app.activePlugin == nil {
		t.Fatal("after WorkspaceChangedEvent from context, activePlugin should still be context, not nil")
	}
	if app.activePlugin.ID() != "context" {
		t.Errorf("after WorkspaceChangedEvent from context, activePlugin = %q, want %q", app.activePlugin.ID(), "context")
	}
}

func TestApp_ChdirSelection_FromContextDoesNotNavigateBack(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Activate "context" plugin directly (NavReplace — no returnTo saved)
	if p, ok := app.registry.ByID("context"); ok {
		app.activePlugin = p
	}

	// Send ChdirChangedEvent while context is active (simulates context's internal picker)
	model, _ := app.Update(sdk.ChdirChangedEvent{RelPath: "modules/vpc", AbsPath: "/test/dir/modules/vpc", Count: 2})
	app = model.(App)

	// Context plugin should remain active
	if app.activePlugin == nil {
		t.Fatal("after ChdirChangedEvent from context, activePlugin should still be context, not nil")
	}
	if app.activePlugin.ID() != "context" {
		t.Errorf("after ChdirChangedEvent from context, activePlugin = %q, want %q", app.activePlugin.ID(), "context")
	}
}

func TestApp_DeactivateMsg_NavigatesBackToReturnTo(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Manually set navStack to simulate NavPush from state → chdir
	statePlugin, _ := app.registry.ByID("state")
	app.navStack = []sdk.Plugin{statePlugin}

	// Activate chdir as the current plugin
	chdirPlugin, _ := app.registry.ByID("chdir")
	app.activePlugin = chdirPlugin

	// DeactivateMsg (esc) should navigate back to navStack top (state)
	model, _ := app.Update(sdk.DeactivateMsg{})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "state" {
		t.Errorf("after DeactivateMsg with navStack, activePlugin should be state, got %v", app.activePlugin)
	}
	if len(app.navStack) != 0 {
		t.Errorf("after DeactivateMsg, navStack should be empty, got %v", app.navStack)
	}
}

func TestApp_NavigateMsg_ActivatesTargetPlugin(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Activate "context" plugin
	if p, ok := app.registry.ByID("context"); ok {
		app.activePlugin = p
	}

	// NavigateMsg from context requesting navigation to "workspace"
	model, _ := app.Update(sdk.NavigateMsg{PluginID: "workspace"})
	app = model.(App)

	// Should navigate to workspaces (NavPush pushes context onto navStack)
	if app.activePlugin == nil || app.activePlugin.ID() != "workspace" {
		t.Fatalf("after NavigateMsg, activePlugin = %v, want workspaces", app.activePlugin)
	}
	if len(app.navStack) == 0 || app.navStack[len(app.navStack)-1] == nil || app.navStack[len(app.navStack)-1].ID() != "context" {
		t.Fatalf("after NavigateMsg, navStack top = %v, want context", app.navStack)
	}
}

func TestApp_NavigateMsg_UnknownPlugin(t *testing.T) {
	app := setupTestAppWithTransientPlugins()
	app.activePlugin = nil

	model, _ := app.Update(sdk.NavigateMsg{PluginID: "nonexistent"})
	app = model.(App)

	if app.activePlugin != nil {
		t.Errorf("NavigateMsg with unknown plugin should not activate anything, got %q", app.activePlugin.ID())
	}
}

func TestApp_DeactivateMsg_NavigatesBackWhenPushed(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Activate "context" then navigate to "workspace" (NavPush)
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	for _, ch := range "context" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	// Now navigate to workspaces (NavPush, pushes context onto navStack)
	model, _ = app.Update(sdk.NavigateMsg{PluginID: "workspace"})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "workspace" {
		t.Fatal("precondition: workspaces should be active")
	}
	if len(app.navStack) == 0 || app.navStack[len(app.navStack)-1] == nil || app.navStack[len(app.navStack)-1].ID() != "context" {
		t.Fatal("precondition: navStack top should be context")
	}

	// Cancel via DeactivateMsg (esc) — should go back to context, not home
	model, _ = app.Update(sdk.DeactivateMsg{})
	app = model.(App)

	if app.activePlugin == nil {
		t.Fatal("after esc/DeactivateMsg from pushed plugin, should return to context, not home")
	}
	if app.activePlugin.ID() != "context" {
		t.Errorf("after esc/DeactivateMsg, activePlugin = %q, want %q", app.activePlugin.ID(), "context")
	}
}

func TestApp_DeactivateMsg_GoesHomeWhenNotPushed(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Activate "state" directly (NavReplace, no returnTo)
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "state" {
		t.Fatal("precondition: state should be active")
	}

	// DeactivateMsg from a non-pushed plugin goes home
	model, _ = app.Update(sdk.DeactivateMsg{})
	app = model.(App)

	if app.activePlugin != nil {
		t.Errorf("after DeactivateMsg from non-pushed plugin, should go home, got %q", app.activePlugin.ID())
	}
}

func TestApp_NavStack_MultiLevelPush_PopsOneAtATime(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	statePlugin, _ := app.registry.ByID("state")
	chdirPlugin, _ := app.registry.ByID("chdir")
	wsPlugin, _ := app.registry.ByID("workspace")

	// Simulate: state active, push chdir, then push workspace
	app.activePlugin = statePlugin
	app.navigateTo(chdirPlugin) // NavPush: pushes state onto stack
	app.navigateTo(wsPlugin)    // NavPush: pushes chdir onto stack

	// Stack should be [state, chdir], active = workspace
	if len(app.navStack) != 2 {
		t.Fatalf("expected navStack depth 2, got %d", len(app.navStack))
	}
	if app.activePlugin.ID() != "workspace" {
		t.Fatalf("activePlugin = %q, want workspace", app.activePlugin.ID())
	}

	// Pop once: back to chdir
	model, _ := app.Update(sdk.DeactivateMsg{})
	app = model.(App)
	if app.activePlugin == nil || app.activePlugin.ID() != "chdir" {
		t.Errorf("after first pop, activePlugin = %v, want chdir", app.activePlugin)
	}
	if len(app.navStack) != 1 {
		t.Errorf("after first pop, navStack depth = %d, want 1", len(app.navStack))
	}

	// Pop again: back to state
	model, _ = app.Update(sdk.DeactivateMsg{})
	app = model.(App)
	if app.activePlugin == nil || app.activePlugin.ID() != "state" {
		t.Errorf("after second pop, activePlugin = %v, want state", app.activePlugin)
	}
	if len(app.navStack) != 0 {
		t.Errorf("after second pop, navStack depth = %d, want 0", len(app.navStack))
	}
}

func TestApp_NavStack_QKey_ClearsEntireStack(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	statePlugin, _ := app.registry.ByID("state")
	chdirPlugin, _ := app.registry.ByID("chdir")
	wsPlugin, _ := app.registry.ByID("workspace")

	// Build multi-level stack
	app.activePlugin = statePlugin
	app.navigateTo(chdirPlugin)
	app.navigateTo(wsPlugin)

	// Press q: should go home, clearing entire stack
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Errorf("q should go home, got activePlugin = %q", updated.activePlugin.ID())
	}
	if len(updated.navStack) != 0 {
		t.Errorf("q should clear entire navStack, got depth %d", len(updated.navStack))
	}
}

func TestApp_NavStack_NavReplace_ClearsStack(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	statePlugin, _ := app.registry.ByID("state")
	chdirPlugin, _ := app.registry.ByID("chdir")
	planPlugin, _ := app.registry.ByID("plan")

	// Build stack: state pushed, chdir active
	app.navStack = []sdk.Plugin{statePlugin}
	app.activePlugin = chdirPlugin

	// Navigate to plan (NavReplace) — should wipe the stack
	app.navigateTo(planPlugin)

	if len(app.navStack) != 0 {
		t.Errorf("NavReplace should clear navStack, got depth %d", len(app.navStack))
	}
	if app.activePlugin.ID() != "plan" {
		t.Errorf("activePlugin = %q, want plan", app.activePlugin.ID())
	}
}

func TestApp_NavStack_PushFromHome_PushesNilAndPopsToHome(t *testing.T) {
	app := setupTestAppWithTransientPlugins()
	app.activePlugin = nil

	chdirPlugin, _ := app.registry.ByID("chdir")
	app.navigateTo(chdirPlugin)

	if len(app.navStack) != 1 {
		t.Fatalf("navStack depth = %d, want 1", len(app.navStack))
	}
	if app.navStack[0] != nil {
		t.Errorf("navStack[0] = %v, want nil (home)", app.navStack[0])
	}

	// Pop should return to home (nil)
	app.navigateBack()
	if app.activePlugin != nil {
		t.Errorf("after pop, activePlugin = %v, want nil (home)", app.activePlugin)
	}
	if len(app.navStack) != 0 {
		t.Errorf("after pop, navStack depth = %d, want 0", len(app.navStack))
	}
}

func TestApp_NavStack_DeactivateMsg_MultiLevel_PopsOnce(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	statePlugin, _ := app.registry.ByID("state")
	chdirPlugin, _ := app.registry.ByID("chdir")
	wsPlugin, _ := app.registry.ByID("workspace")

	// Build stack manually: [state, chdir], active = workspace
	app.navStack = []sdk.Plugin{statePlugin, chdirPlugin}
	app.activePlugin = wsPlugin

	// DeactivateMsg should pop back to chdir only
	model, _ := app.Update(sdk.DeactivateMsg{})
	updated := model.(App)

	if updated.activePlugin == nil || updated.activePlugin.ID() != "chdir" {
		t.Errorf("after DeactivateMsg, activePlugin = %v, want chdir", updated.activePlugin)
	}
	if len(updated.navStack) != 1 {
		t.Errorf("after DeactivateMsg, navStack depth = %d, want 1", len(updated.navStack))
	}
	if updated.navStack[0].ID() != "state" {
		t.Errorf("navStack[0] = %q, want state", updated.navStack[0].ID())
	}
}

func TestApp_WorkspaceChanged_ShouldPreserveChdirInHeader(t *testing.T) {
	app := setupTestApp()
	app.width = 120
	app.height = 30

	// Set a chdir first
	model, _ := app.Update(sdk.ChdirChangedEvent{RelPath: "modules/vpc", AbsPath: "/test/modules/vpc", Count: 2})
	app = model.(App)

	// Verify chdir is in the header
	view := app.View()
	if !strings.Contains(view, "modules/vpc") {
		t.Fatal("precondition: header should contain chdir before workspace change")
	}

	// Now change workspace
	model, _ = app.Update(sdk.WorkspaceChangedEvent{Name: "staging"})
	app = model.(App)

	// Header should still show chdir
	view = app.View()
	if !strings.Contains(view, "modules/vpc") {
		t.Error("after WorkspaceChangedEvent, header should still show chdir 'modules/vpc'")
	}
	if !strings.Contains(view, "staging") {
		t.Error("after WorkspaceChangedEvent, header should show new workspace 'staging'")
	}
}

func TestApp_WorkspaceLoaded_ShouldPreserveChdirInHeader(t *testing.T) {
	app := setupTestApp()
	app.width = 120
	app.height = 30

	// Set a chdir first
	model, _ := app.Update(sdk.ChdirChangedEvent{RelPath: "modules/ecs", AbsPath: "/test/modules/ecs", Count: 2})
	app = model.(App)

	// Now simulate workspace initial load
	model, _ = app.Update(workspaceLoadedMsg{workspace: "production"})
	app = model.(App)

	// Header should still show chdir
	view := app.View()
	if !strings.Contains(view, "modules/ecs") {
		t.Error("after workspaceLoadedMsg, header should still show chdir 'modules/ecs'")
	}
	if !strings.Contains(view, "production") {
		t.Error("after workspaceLoadedMsg, header should show workspace 'production'")
	}
}

func TestApp_WorkspaceCreated_ShouldUpdateHeaderAndNotPop(t *testing.T) {
	app := setupTestAppWithTransientPlugins()
	app.width = 120
	app.height = 30

	// Activate state, then navigate to workspaces (NavPush)
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	app = model.(App)

	// Switch to workspaces via command mode
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app = model.(App)
	for _, ch := range "workspace" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	if app.activePlugin == nil || app.activePlugin.ID() != "workspace" {
		t.Fatal("precondition: workspaces should be active")
	}

	// Send WorkspaceCreatedEvent (not WorkspaceChangedEvent)
	model, _ = app.Update(sdk.WorkspaceCreatedEvent{Name: "new-feature"})
	app = model.(App)

	// Should NOT pop back — workspace plugin stays active
	if app.activePlugin == nil {
		t.Fatal("after WorkspaceCreatedEvent, plugin should still be active")
	}
	if app.activePlugin.ID() != "workspace" {
		t.Errorf("after WorkspaceCreatedEvent, activePlugin = %q, want %q (should NOT pop)", app.activePlugin.ID(), "workspace")
	}

	// Header should show new workspace
	view := app.View()
	if !strings.Contains(view, "new-feature") {
		t.Error("after WorkspaceCreatedEvent, header should show new workspace name")
	}
}

func TestApp_LockDetectedEvent_SetsLockInfoAndUpdatesHeader(t *testing.T) {
	app := setupTestApp()
	app.width = 120
	app.height = 30

	lock := &sdk.StateLock{ID: "abc-123", Who: "user@host"}
	model, _ := app.Update(sdk.LockDetectedEvent{Lock: lock})
	app = model.(App)

	if app.lockInfo == nil {
		t.Fatal("LockDetectedEvent should set lockInfo")
	}
	if app.lockInfo.ID != "abc-123" {
		t.Errorf("lockInfo.ID = %q, want %q", app.lockInfo.ID, "abc-123")
	}

	view := app.View()
	if !strings.Contains(view, "locked") {
		t.Error("header should contain 'locked' badge after LockDetectedEvent")
	}
}

func TestApp_LockClearedEvent_ClearsLockInfoAndUpdatesHeader(t *testing.T) {
	app := setupTestApp()
	app.width = 120
	app.height = 30

	// Set lock first
	lock := &sdk.StateLock{ID: "abc-123", Who: "user@host"}
	model, _ := app.Update(sdk.LockDetectedEvent{Lock: lock})
	app = model.(App)

	// Clear it
	model, _ = app.Update(sdk.LockClearedEvent{})
	app = model.(App)

	if app.lockInfo != nil {
		t.Error("LockClearedEvent should clear lockInfo")
	}

	view := app.View()
	if strings.Contains(view, "locked") {
		t.Error("header should not contain 'locked' badge after LockClearedEvent")
	}
}

func TestApp_PlanInvalidatedEvent_SetsStaleBadge(t *testing.T) {
	app := setupTestApp()
	app.width = 120
	app.height = 30

	model, _ := app.Update(sdk.PlanInvalidatedEvent{})
	app = model.(App)

	if !app.staleState {
		t.Error("PlanInvalidatedEvent should set staleState to true")
	}

	view := app.View()
	if !strings.Contains(view, "stale") {
		t.Error("header should contain 'stale' badge after PlanInvalidatedEvent")
	}
}

func TestApp_StateRefreshedEvent_ClearsStaleBadge(t *testing.T) {
	app := setupTestApp()
	app.width = 120
	app.height = 30

	// Set stale first
	model, _ := app.Update(sdk.PlanInvalidatedEvent{})
	app = model.(App)

	// Clear it
	model, _ = app.Update(sdk.StateRefreshedEvent{})
	app = model.(App)

	if app.staleState {
		t.Error("StateRefreshedEvent should set staleState to false")
	}

	view := app.View()
	if strings.Contains(view, "stale") {
		t.Error("header should not contain 'stale' badge after StateRefreshedEvent")
	}
}

func TestApp_WorkspaceChanged_ResolvesOptions(t *testing.T) {
	rootCfg := &config.RootConfig{
		Defaults: config.DefaultsConfig{
			VarFiles: []string{"common.tfvars"},
		},
	}
	childCfg := &config.ChildConfig{
		Workspaces: []config.WorkspaceConfig{
			{Name: "staging", VarFiles: []string{"staging.tfvars"}, Vars: map[string]string{"env": "stg"}},
			{Name: "production", VarFiles: []string{"prod.tfvars"}, Vars: map[string]string{"env": "prd"}},
		},
	}

	cfg := config.Config{Dir: "/test", Terraform: config.TerraformConfig{Bin: "terraform"}}
	svc := &mockService{workspace: "default"}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, rootCfg)
	app.childCfg = childCfg

	// Workspace switch to staging
	model, _ := app.Update(sdk.WorkspaceChangedEvent{Name: "staging"})
	app = model.(App)

	if len(app.options.VarFiles) != 2 || app.options.VarFiles[1] != "staging.tfvars" {
		t.Errorf("VarFiles after staging switch = %v, want [common.tfvars staging.tfvars]", app.options.VarFiles)
	}
	if app.options.Vars["env"] != "stg" {
		t.Errorf("Vars[env] = %q, want %q", app.options.Vars["env"], "stg")
	}

	// Workspace switch to production
	model, _ = app.Update(sdk.WorkspaceChangedEvent{Name: "production"})
	app = model.(App)

	if len(app.options.VarFiles) != 2 || app.options.VarFiles[1] != "prod.tfvars" {
		t.Errorf("VarFiles after prod switch = %v, want [common.tfvars prod.tfvars]", app.options.VarFiles)
	}
	if app.options.Vars["env"] != "prd" {
		t.Errorf("Vars[env] = %q, want %q", app.options.Vars["env"], "prd")
	}
}

func TestApp_WorkspaceChanged_NilRootCfg_NoOp(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test",
		Terraform: config.TerraformConfig{Bin: "terraform"},
		VarFiles:  []string{"original.tfvars"},
	}
	svc := &mockService{workspace: "default"}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)

	model, _ := app.Update(sdk.WorkspaceChangedEvent{Name: "staging"})
	app = model.(App)

	if len(app.options.VarFiles) != 1 || app.options.VarFiles[0] != "original.tfvars" {
		t.Errorf("Options should be unchanged without rootCfg, got VarFiles = %v", app.options.VarFiles)
	}
}
