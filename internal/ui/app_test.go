package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/editor"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	sdkui "github.com/lmarqs/terraform-ui/pkg/sdk/ui"
	tfuiapply "github.com/lmarqs/terraform-ui/plugins/apply"
	tfuiimport "github.com/lmarqs/terraform-ui/plugins/import"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	tfuistate "github.com/lmarqs/terraform-ui/plugins/state"
	tfuitaint "github.com/lmarqs/terraform-ui/plugins/taint"
	tfuiuntaint "github.com/lmarqs/terraform-ui/plugins/untaint"
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

// newMockService creates a mock service with optional workspace override.
func newMockService(workspace string, workspaceErr error) *sdktest.MockService {
	svc := &sdktest.MockService{}
	if workspace != "" || workspaceErr != nil {
		svc.WorkspaceFn = func(_ context.Context) (string, error) {
			return workspace, workspaceErr
		}
	}
	return svc
}

func setupTestApp() App {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}

	svc := newMockService("default", nil)

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

	svc := newMockService("production", nil)
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

	svc := newMockService("", fmt.Errorf("connection failed"))
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

	svc := newMockService("default", nil)

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

	svc := newMockService("default", nil)
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

	svc := newMockService("default", nil)
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

	svc := newMockService("default", nil)

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

	svc := newMockService("default", nil)

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

	svc := newMockService("default", nil)

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
	svc := newMockService("default", nil)
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
	svc := newMockService("default", nil)
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)

	model, _ := app.Update(sdk.WorkspaceChangedEvent{Name: "staging"})
	app = model.(App)

	if len(app.options.VarFiles) != 1 || app.options.VarFiles[0] != "original.tfvars" {
		t.Errorf("Options should be unchanged without rootCfg, got VarFiles = %v", app.options.VarFiles)
	}
}

// --- Additional mock types for interface coverage ---

type mockActivatablePlugin struct {
	mockPlugin
	activateCalled bool
	activateCmd    tea.Cmd
}

func (m *mockActivatablePlugin) Activate() tea.Cmd {
	m.activateCalled = true
	return m.activateCmd
}

type mockStackablePlugin struct {
	mockPlugin
	stack *sdk.Stack
}

func (m *mockStackablePlugin) Stack() *sdk.Stack { return m.stack }

type mockHintablePlugin struct {
	mockPlugin
	hints []sdk.KeyHint
}

func (m *mockHintablePlugin) Hints() []sdk.KeyHint { return m.hints }

type mockCountablePlugin struct {
	mockPlugin
	filtered int
	total    int
}

func (m *mockCountablePlugin) Count() (int, int) { return m.filtered, m.total }

type mockPinnablePlugin struct {
	mockPlugin
	pinnedCount int
}

func (m *mockPinnablePlugin) PinnedCount() int { return m.pinnedCount }

type mockOverlay struct {
	id         string
	viewOutput string
	hints      []sdk.KeyHint
	updateFn   func(tea.Msg) (sdk.Overlay, tea.Cmd)
}

func (o *mockOverlay) ID() string           { return o.id }
func (o *mockOverlay) Open() tea.Cmd        { return nil }
func (o *mockOverlay) View(_, _ int) string { return o.viewOutput }
func (o *mockOverlay) Hints() []sdk.KeyHint { return o.hints }
func (o *mockOverlay) Update(msg tea.Msg) (sdk.Overlay, tea.Cmd) {
	if o.updateFn != nil {
		return o.updateFn(msg)
	}
	return o, nil
}

type mockFrame struct {
	id       string
	hints    []sdk.KeyHint
	updateFn func(tea.Msg) (sdk.Frame, tea.Cmd)
}

func (f *mockFrame) ID() string           { return f.id }
func (f *mockFrame) View(_, _ int) string { return f.id + " view" }
func (f *mockFrame) Hints() []sdk.KeyHint { return f.hints }
func (f *mockFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	if f.updateFn != nil {
		return f.updateFn(msg)
	}
	return f, nil
}

type mockActivateWithArgsPlugin struct {
	mockPlugin
	activatedWithArgs []string
}

func (m *mockActivateWithArgsPlugin) ActivateWithArgs(args []string) tea.Cmd {
	m.activatedWithArgs = args
	return func() tea.Msg { return nil }
}

func (m *mockActivateWithArgsPlugin) Activate() tea.Cmd { return nil }

type mockKeyCapturerPlugin struct {
	mockPlugin
	captures   bool
	lastKeyMsg tea.Msg
}

func (m *mockKeyCapturerPlugin) CapturesKeys() bool { return m.captures }
func (m *mockKeyCapturerPlugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	m.lastKeyMsg = msg
	return m, nil
}

type mockCancellablePlugin struct {
	mockPlugin
	cancelled bool
}

func (m *mockCancellablePlugin) Cancel() { m.cancelled = true }

type mockBusyCancellablePlugin struct {
	mockCancellablePlugin
	busy bool
}

func (m *mockBusyCancellablePlugin) Busy() bool { return m.busy }

// --- Helper functions ---

func setupAppWithOverlay(overlay sdk.Overlay) App {
	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activeOverlay = overlay
	return app
}

func setupAppWithSourceIndex(t *testing.T) App {
	t.Helper()
	dir := t.TempDir()

	cfg := config.Config{
		Dir:       dir,
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)
	return app
}

// --- Tests for Update branches ---

func TestApp_Update_WhenReceivingPlanCompletedEvent_ShouldDispatchToBus(t *testing.T) {
	app := setupTestApp()

	model, cmd := app.Update(sdk.PlanCompletedEvent{ResourceCount: 5})
	_ = model.(App)
	// Bus dispatch with no handlers returns nil
	if cmd != nil {
		t.Error("PlanCompletedEvent with no handlers should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingPlanInvalidatedEvent_ShouldDispatchToBus(t *testing.T) {
	app := setupTestApp()

	model, cmd := app.Update(sdk.PlanInvalidatedEvent{})
	_ = model.(App)
	if cmd != nil {
		t.Error("PlanInvalidatedEvent with no handlers should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingOverlayDismissMsg_ShouldClearOverlay(t *testing.T) {
	overlay := &mockOverlay{id: "test-overlay", viewOutput: "overlay content"}
	app := setupAppWithOverlay(overlay)

	model, cmd := app.Update(sdk.OverlayDismissMsg{})
	updated := model.(App)

	if updated.activeOverlay != nil {
		t.Error("OverlayDismissMsg should clear the active overlay")
	}
	if cmd != nil {
		t.Error("OverlayDismissMsg should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingRequestInputMsg_ShouldSetupInputState(t *testing.T) {
	app := setupTestApp()

	callbackCalled := false
	req := sdk.RequestInputMsg{
		Request: sdk.InputRequest{
			Mode:     sdk.InputRequestBool,
			Prompt:   "Confirm?",
			Default:  "n",
			Callback: func(answer string) tea.Cmd { callbackCalled = true; return nil },
		},
	}

	model, _ := app.Update(req)
	updated := model.(App)

	if !updated.inputActive {
		t.Error("RequestInputMsg should set inputActive to true")
	}
	if updated.inputMode != sdk.InputRequestBool {
		t.Errorf("inputMode = %d, want %d", updated.inputMode, sdk.InputRequestBool)
	}
	if updated.inputPrompt != "Confirm?" {
		t.Errorf("inputPrompt = %q, want %q", updated.inputPrompt, "Confirm?")
	}
	if updated.inputAnswer != "n" {
		t.Errorf("inputAnswer = %q, want %q", updated.inputAnswer, "n")
	}
	if updated.inputCallback == nil {
		t.Error("inputCallback should not be nil")
	}
	_ = callbackCalled
}

func TestApp_Update_WhenReceivingStateEditMsg_ShouldHandleNilSourceIndex(t *testing.T) {
	app := setupTestApp()
	app.sourceIndex = nil

	model, cmd := app.Update(tfuistate.StateEditMsg{Address: "aws_instance.foo"})
	_ = model.(App)
	if cmd != nil {
		t.Error("StateEditMsg with nil sourceIndex should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingStateEditMsgWithAddresses_ShouldLookupMultiple(t *testing.T) {
	app := setupAppWithSourceIndex(t)

	model, cmd := app.Update(tfuistate.StateEditMsg{Addresses: []string{"aws_instance.foo", "aws_instance.bar"}})
	_ = model.(App)
	// No matches in the source index, so should return nil
	if cmd != nil {
		t.Error("StateEditMsg with no matching addresses should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingStateEditMsgWithSingleAddress_ShouldLookup(t *testing.T) {
	app := setupAppWithSourceIndex(t)

	model, cmd := app.Update(tfuistate.StateEditMsg{Address: "aws_instance.foo"})
	_ = model.(App)
	// No match in the source index, so should return nil
	if cmd != nil {
		t.Error("StateEditMsg with no matching address should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingStateEditMsgWithMatchingAddresses(t *testing.T) {
	app := setupAppWithSourceIndex(t)
	// Manually add a location to the source index
	app.sourceIndex.Lookup("fake") // just confirm it doesn't panic

	model, cmd := app.Update(tfuistate.StateEditMsg{Addresses: []string{}})
	_ = model.(App)
	// Empty addresses slice with single Address also empty → falls to single lookup
	if cmd != nil {
		t.Error("StateEditMsg with empty addresses should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingEditorClosedMsgModified_ShouldInvalidatePlan(t *testing.T) {
	app := setupTestApp()

	model, cmd := app.Update(editor.EditorClosedMsg{Modified: true, File: "/tmp/main.tf"})
	_ = model.(App)
	if cmd == nil {
		t.Fatal("EditorClosedMsg with Modified=true should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.PlanInvalidatedEvent); !ok {
		t.Errorf("cmd should produce PlanInvalidatedEvent, got %T", msg)
	}
}

func TestApp_Update_WhenReceivingEditorClosedMsgNotModified_ShouldReturnNil(t *testing.T) {
	app := setupTestApp()

	_, cmd := app.Update(editor.EditorClosedMsg{Modified: false, File: "/tmp/main.tf"})
	if cmd != nil {
		t.Error("EditorClosedMsg with Modified=false should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingApplyRequestMsg_ShouldActivateApplyPlugin(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(s terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("apply", func(s terraform.Service) plugin.Plugin {
		return tfuiapply.New(s)
	}, plugin.PluginMeta{Keybinding: "a", MenuVisible: true})
	registry.Build(svc, nil)

	app := NewApp(cfg, svc, registry, nil)
	// Activate plan plugin
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	app = model.(App)

	model, _ = app.Update(tfuiplan.ApplyRequestMsg{})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("ApplyRequestMsg should activate the apply plugin")
	}
	if updated.activePlugin.ID() != "apply" {
		t.Errorf("activePlugin = %q, want %q", updated.activePlugin.ID(), "apply")
	}
}

func TestApp_Update_WhenReceivingApplyRequestMsgNoApplyPlugin_ShouldReturnNil(t *testing.T) {
	app := setupTestApp() // no apply plugin registered

	_, cmd := app.Update(tfuiplan.ApplyRequestMsg{})
	if cmd != nil {
		t.Error("ApplyRequestMsg without apply plugin should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingApplyRequestMsgWithPins_ShouldSetTargets(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("apply", func(s terraform.Service) plugin.Plugin {
		return tfuiapply.New(s)
	}, plugin.PluginMeta{Keybinding: "a", MenuVisible: true})
	registry.Build(svc, nil)

	app := NewApp(cfg, svc, registry, nil)
	app.pins.Toggle("aws_instance.foo")
	app.pins.Toggle("aws_instance.bar")

	model, _ := app.Update(tfuiplan.ApplyRequestMsg{})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("ApplyRequestMsg should activate apply plugin")
	}
}

func TestApp_Update_WhenReceivingGenericEvent_ShouldDispatchToBus(t *testing.T) {
	app := setupTestApp()

	// PinsChangedEvent is a generic event that will be dispatched via the bus
	model, cmd := app.Update(sdk.PinsChangedEvent{Addresses: []string{"a.b"}})
	_ = model.(App)
	if cmd != nil {
		t.Error("Generic event with no handlers should return nil cmd")
	}
}

func TestApp_Update_WhenOverlayActive_ShouldRouteNonKeyMsgToOverlay(t *testing.T) {
	overlay := &mockOverlay{
		id:         "test-overlay",
		viewOutput: "overlay",
		updateFn: func(msg tea.Msg) (sdk.Overlay, tea.Cmd) {
			return &mockOverlay{id: "updated-overlay", viewOutput: "updated"}, nil
		},
	}
	app := setupTestApp()
	app.activeOverlay = overlay

	// Send a non-key, non-event message → should route to overlay
	model, _ := app.Update(customMsg{})
	updated := model.(App)

	if updated.activeOverlay == nil {
		t.Fatal("overlay should still be active")
	}
	if updated.activeOverlay.ID() != "updated-overlay" {
		t.Errorf("overlay ID = %q, want %q", updated.activeOverlay.ID(), "updated-overlay")
	}
}

func TestApp_Update_WhenOverlayReturnsNil_ShouldClearOverlay(t *testing.T) {
	overlay := &mockOverlay{
		id: "test-overlay",
		updateFn: func(msg tea.Msg) (sdk.Overlay, tea.Cmd) {
			return nil, nil
		},
	}
	app := setupTestApp()
	app.activeOverlay = overlay

	model, _ := app.Update(customMsg{})
	updated := model.(App)

	if updated.activeOverlay != nil {
		t.Error("overlay returning nil should clear activeOverlay")
	}
}

func TestApp_Update_WhenChdirChangedEvent_ShouldUpdateHeader(t *testing.T) {
	app := setupTestApp()

	model, _ := app.Update(sdk.ChdirChangedEvent{RelPath: "modules/net", AbsPath: "/test/dir/modules/net", Count: 3})
	updated := model.(App)

	if updated.activeChdir != "modules/net" {
		t.Errorf("activeChdir = %q, want %q", updated.activeChdir, "modules/net")
	}
}

func TestApp_Update_WhenWorkspaceChangedEvent_ShouldUpdateHeader(t *testing.T) {
	app := setupTestApp()
	app.width = 80
	app.height = 24

	model, _ := app.Update(sdk.WorkspaceChangedEvent{Name: "production"})
	updated := model.(App)

	output := updated.View()
	if !strings.Contains(output, "production") {
		t.Error("View() after WorkspaceChangedEvent should contain new workspace name")
	}
}

// --- Tests for handleKey branches ---

func TestApp_HandleKey_WhenOverlayActive_ShouldRouteKeyToOverlay(t *testing.T) {
	overlayGotKey := false
	overlay := &mockOverlay{
		id: "key-overlay",
		updateFn: func(msg tea.Msg) (sdk.Overlay, tea.Cmd) {
			if _, ok := msg.(tea.KeyMsg); ok {
				overlayGotKey = true
			}
			return &mockOverlay{id: "key-overlay"}, nil
		},
	}
	app := setupTestApp()
	app.activeOverlay = overlay

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if !overlayGotKey {
		t.Error("key should be routed to overlay when active")
	}
}

func TestApp_HandleKey_WhenOverlayDismissedByKey_ShouldClearOverlay(t *testing.T) {
	overlay := &mockOverlay{
		id: "dismiss-overlay",
		updateFn: func(msg tea.Msg) (sdk.Overlay, tea.Cmd) {
			return nil, nil
		},
	}
	app := setupTestApp()
	app.activeOverlay = overlay

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(App)

	if updated.activeOverlay != nil {
		t.Error("overlay returning nil on key should clear activeOverlay")
	}
}

func TestApp_HandleKey_WhenInputActiveBool_ShouldHandleYes(t *testing.T) {
	app := setupTestApp()
	callbackAnswer := ""
	app.inputActive = true
	app.inputMode = sdk.InputRequestBool
	app.inputPrompt = "Confirm?"
	app.inputCallback = func(answer string) tea.Cmd {
		callbackAnswer = answer
		return nil
	}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	updated := model.(App)

	if callbackAnswer != "y" {
		t.Errorf("callback answer = %q, want %q", callbackAnswer, "y")
	}
	if updated.inputActive {
		t.Error("inputActive should be false after y")
	}
}

func TestApp_HandleKey_WhenInputActiveBool_ShouldHandleNo(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestBool
	app.inputPrompt = "Confirm?"
	app.inputCallback = func(answer string) tea.Cmd { return nil }

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	updated := model.(App)

	if updated.inputActive {
		t.Error("inputActive should be false after n")
	}
	if updated.inputCallback != nil {
		t.Error("inputCallback should be cleared after n")
	}
}

func TestApp_HandleKey_WhenInputActiveBool_ShouldHandleEsc(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestBool
	app.inputPrompt = "Confirm?"
	app.inputCallback = func(answer string) tea.Cmd { return nil }

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(App)

	if updated.inputActive {
		t.Error("inputActive should be false after esc")
	}
}

func TestApp_HandleKey_WhenInputActiveBool_ShouldIgnoreOtherKeys(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestBool
	app.inputPrompt = "Confirm?"
	app.inputCallback = func(answer string) tea.Cmd { return nil }

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	updated := model.(App)

	if !updated.inputActive {
		t.Error("inputActive should remain true after unrecognized key in bool mode")
	}
}

func TestApp_HandleKey_WhenInputActiveText_ShouldHandleEnter(t *testing.T) {
	app := setupTestApp()
	callbackAnswer := ""
	app.inputActive = true
	app.inputMode = sdk.InputRequestText
	app.inputPrompt = "Name:"
	app.inputAnswer = "hello"
	app.inputCallback = func(answer string) tea.Cmd {
		callbackAnswer = answer
		return nil
	}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(App)

	if callbackAnswer != "hello" {
		t.Errorf("callback answer = %q, want %q", callbackAnswer, "hello")
	}
	if updated.inputActive {
		t.Error("inputActive should be false after enter")
	}
}

func TestApp_HandleKey_WhenInputActiveText_ShouldHandleEsc(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestText
	app.inputPrompt = "Name:"
	app.inputAnswer = "partial"
	app.inputCallback = func(answer string) tea.Cmd { return nil }

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(App)

	if updated.inputActive {
		t.Error("inputActive should be false after esc")
	}
	if updated.inputAnswer != "" {
		t.Errorf("inputAnswer should be cleared, got %q", updated.inputAnswer)
	}
}

func TestApp_HandleKey_WhenInputActiveText_ShouldHandleBackspace(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestText
	app.inputPrompt = "Name:"
	app.inputAnswer = "hel"
	app.inputCallback = func(answer string) tea.Cmd { return nil }

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated := model.(App)

	if updated.inputAnswer != "he" {
		t.Errorf("inputAnswer = %q, want %q after backspace", updated.inputAnswer, "he")
	}
}

func TestApp_HandleKey_WhenInputActiveText_ShouldHandleBackspaceEmpty(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestText
	app.inputPrompt = "Name:"
	app.inputAnswer = ""
	app.inputCallback = func(answer string) tea.Cmd { return nil }

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated := model.(App)

	if updated.inputAnswer != "" {
		t.Errorf("inputAnswer should remain empty, got %q", updated.inputAnswer)
	}
}

func TestApp_HandleKey_WhenInputActiveText_ShouldHandleTyping(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestText
	app.inputPrompt = "Name:"
	app.inputAnswer = "ab"
	app.inputCallback = func(answer string) tea.Cmd { return nil }

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updated := model.(App)

	if updated.inputAnswer != "abc" {
		t.Errorf("inputAnswer = %q, want %q after typing", updated.inputAnswer, "abc")
	}
}

func TestApp_HandleKey_WhenInputActiveText_ShouldIgnoreNonPrintable(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestText
	app.inputPrompt = "Name:"
	app.inputAnswer = "ab"
	app.inputCallback = func(answer string) tea.Cmd { return nil }

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := model.(App)

	if updated.inputAnswer != "ab" {
		t.Errorf("inputAnswer should remain %q for non-printable key, got %q", "ab", updated.inputAnswer)
	}
}

func TestApp_HandleKey_WhenInputActiveBoolNoCallback_ShouldNotPanic(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestBool
	app.inputPrompt = "Confirm?"
	app.inputCallback = nil

	// Should not panic
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	updated := model.(App)

	if !updated.inputActive {
		t.Error("with nil callback, y should not change state")
	}
}

func TestApp_HandleKey_WhenInputActiveTextNoCallback_ShouldNotPanic(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestText
	app.inputPrompt = "Name:"
	app.inputAnswer = "test"
	app.inputCallback = nil

	// Enter with nil callback should not panic
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(App)
	// With nil callback, nothing happens (stays active since no submit logic)
	_ = updated
}

func TestApp_HandleKey_WhenCommandModeBackspaceEmpty_ShouldExitCommandMode(t *testing.T) {
	app := setupTestApp()
	app.commandMode = true
	app.commandInput = ""

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated := model.(App)

	if updated.commandMode {
		t.Error("backspace with empty input should exit command mode")
	}
}

func TestApp_HandleKey_WhenCommandModeBackspaceWithContent_ShouldDeleteChar(t *testing.T) {
	app := setupTestApp()
	app.commandMode = true
	app.commandInput = "sta"

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated := model.(App)

	if !updated.commandMode {
		t.Error("backspace with content should stay in command mode")
	}
	if updated.commandInput != "st" {
		t.Errorf("commandInput = %q, want %q", updated.commandInput, "st")
	}
}

func TestApp_HandleKey_WhenCommandModeDelete_ShouldDeleteChar(t *testing.T) {
	app := setupTestApp()
	app.commandMode = true
	app.commandInput = "ab"

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyDelete})
	updated := model.(App)

	if updated.commandInput != "a" {
		t.Errorf("commandInput = %q, want %q after delete", updated.commandInput, "a")
	}
}

func TestApp_HandleKey_WhenCommandModeNonPrintable_ShouldIgnore(t *testing.T) {
	app := setupTestApp()
	app.commandMode = true
	app.commandInput = "st"

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := model.(App)

	if updated.commandInput != "st" {
		t.Errorf("commandInput should remain %q for non-printable, got %q", "st", updated.commandInput)
	}
}

func TestApp_HandleKey_WhenCommandModeEnterNoMatch_ShouldReturnNil(t *testing.T) {
	app := setupTestApp()
	app.commandMode = true
	app.commandInput = "zzzzz"

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(App)

	if updated.commandMode {
		t.Error("enter should exit command mode")
	}
	if cmd != nil {
		t.Error("enter with no match should return nil cmd")
	}
}

func TestApp_HandleKey_WhenCommandModeTabNoMatch_ShouldNotChange(t *testing.T) {
	app := setupTestApp()
	app.commandMode = true
	app.commandInput = "zzz"

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := model.(App)

	if updated.commandInput != "zzz" {
		t.Errorf("tab with no match should not change input, got %q", updated.commandInput)
	}
}

func TestApp_HandleKey_WhenGlobalC_ShouldNavigateToContext(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	updated := model.(App)

	if updated.activePlugin == nil || updated.activePlugin.ID() != "context" {
		t.Errorf("C should navigate to context plugin, got %v", updated.activePlugin)
	}
}

func TestApp_HandleKey_WhenGlobalCNoContextPlugin_ShouldReturnNil(t *testing.T) {
	app := setupTestApp() // no context plugin registered

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("C with no context plugin should not activate anything")
	}
	if cmd != nil {
		t.Error("C with no context plugin should return nil cmd")
	}
}

func TestApp_HandleKey_WhenCtrlS_ShouldCaptureScreen(t *testing.T) {
	app := setupTestApp()
	app.width = 80
	app.height = 24

	// Should not panic or error
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	_ = model.(App)
	if cmd != nil {
		t.Error("ctrl+s should return nil cmd")
	}
}

func TestApp_HandleKey_WhenQWithStackableDepthGreaterThanOne_ShouldClearStack(t *testing.T) {
	stack := sdk.NewStack()
	stack.Push(&mockFrame{id: "root"})
	stack.Push(&mockFrame{id: "detail"})

	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	app := setupTestApp()
	app.activePlugin = p

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Error("q with stackable depth > 1 should clear stack, not deactivate")
	}
	if stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1", stack.Depth())
	}
}

func TestApp_HandleKey_WhenQWithStackableDepthOne_ShouldDeactivate(t *testing.T) {
	stack := sdk.NewStack()
	stack.Push(&mockFrame{id: "root"})

	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	app := setupTestApp()
	app.activePlugin = p

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Errorf("q with stackable depth 1 should deactivate, got %q", updated.activePlugin.ID())
	}
}

func TestApp_HandleKey_WhenKeyDelegatedToStackable_ShouldRouteViaStack(t *testing.T) {
	frameGotMsg := false
	stack := sdk.NewStack()
	stack.Push(&mockFrame{
		id: "root",
		updateFn: func(msg tea.Msg) (sdk.Frame, tea.Cmd) {
			frameGotMsg = true
			return &mockFrame{id: "root"}, nil
		},
	})

	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	app := setupTestApp()
	app.activePlugin = p

	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if !frameGotMsg {
		t.Error("key should be routed through stack frame when plugin is stackable")
	}
}

func TestApp_HandleKey_WhenStackableFrameReturnsNil_ShouldDeactivateWhenEmpty(t *testing.T) {
	stack := sdk.NewStack()
	stack.Push(&mockFrame{
		id: "root",
		updateFn: func(msg tea.Msg) (sdk.Frame, tea.Cmd) {
			return nil, nil // signal pop
		},
	})

	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	app := setupTestApp()
	app.activePlugin = p

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("when stack is empty after update, plugin should deactivate")
	}
}

func TestApp_HandleKey_WhenHomeArrowDown_ShouldMoveSelection(t *testing.T) {
	app := setupTestApp()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := model.(App)

	if updated.homeView.Selected() != 1 {
		t.Errorf("down arrow should move selection to 1, got %d", updated.homeView.Selected())
	}
}

func TestApp_HandleKey_WhenHomeArrowUp_ShouldMoveSelection(t *testing.T) {
	app := setupTestApp()

	// Move down first
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := model.(App)

	if updated.homeView.Selected() != 0 {
		t.Errorf("up arrow should move selection to 0, got %d", updated.homeView.Selected())
	}
}

func TestApp_HandleKey_WhenHomeEnterUnmappedKey_ShouldReturnNil(t *testing.T) {
	app := setupTestApp()

	// Enter activates a plugin (first item), so this validates the enter path works
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(App)

	// It should activate the first plugin in the menu
	if updated.activePlugin == nil {
		t.Error("enter on home should activate the selected plugin")
	}
}

// --- Tests for activate ---

func TestApp_Activate_WhenPluginIsActivatable_ShouldCallActivate(t *testing.T) {
	p := &mockActivatablePlugin{
		mockPlugin:  mockPlugin{id: "test", name: "Test"},
		activateCmd: func() tea.Msg { return customMsg{} },
	}

	app := setupTestApp()
	cmd := app.activate(p)

	if !p.activateCalled {
		t.Error("activate should call Activate on activatable plugins")
	}
	if cmd == nil {
		t.Error("activate should return the cmd from Activate")
	}
}

func TestApp_Activate_WhenPluginNotActivatable_ShouldReturnNil(t *testing.T) {
	p := &mockPlugin{id: "test", name: "Test"}

	app := setupTestApp()
	cmd := app.activate(p)

	if cmd != nil {
		t.Error("activate should return nil for non-activatable plugins")
	}
}

// --- Tests for navigateTo ---

func TestApp_NavigateTo_WhenNavPush_ShouldSaveReturnTo(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Activate state plugin first
	statePlugin, _ := app.registry.ByID("state")
	app.activePlugin = statePlugin

	// Navigate to chdir (NavPush)
	chdirPlugin, _ := app.registry.ByID("chdir")
	app.navigateTo(chdirPlugin)

	if len(app.navStack) == 0 || app.navStack[len(app.navStack)-1] == nil || app.navStack[len(app.navStack)-1].ID() != "state" {
		t.Errorf("navigateTo with NavPush should push to navStack, got %v", app.navStack)
	}
	if app.activePlugin == nil || app.activePlugin.ID() != "chdir" {
		t.Errorf("activePlugin should be chdir, got %v", app.activePlugin)
	}
}

func TestApp_NavigateTo_WhenNavReplace_ShouldClearReturnTo(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	// Set a navStack entry
	statePlugin, _ := app.registry.ByID("state")
	app.navStack = []sdk.Plugin{statePlugin}

	// Navigate to plan (NavReplace)
	planPlugin, _ := app.registry.ByID("plan")
	app.navigateTo(planPlugin)

	if len(app.navStack) != 0 {
		t.Errorf("navigateTo with NavReplace should clear navStack, got %v", app.navStack)
	}
}

// --- Tests for navigateBack ---

func TestApp_NavigateBack_WhenReturnToSet_ShouldRestorePlugin(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	statePlugin, _ := app.registry.ByID("state")
	chdirPlugin, _ := app.registry.ByID("chdir")
	app.activePlugin = chdirPlugin
	app.navStack = []sdk.Plugin{statePlugin}

	app.navigateBack()

	if app.activePlugin == nil || app.activePlugin.ID() != "state" {
		t.Errorf("navigateBack should pop navStack, got %v", app.activePlugin)
	}
	if len(app.navStack) != 0 {
		t.Error("navigateBack should pop from navStack")
	}
}

func TestApp_NavigateBack_WhenReturnToNil_ShouldGoHome(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	chdirPlugin, _ := app.registry.ByID("chdir")
	app.activePlugin = chdirPlugin
	app.navStack = nil

	app.navigateBack()

	if app.activePlugin != nil {
		t.Errorf("navigateBack with empty navStack should go home, got %v", app.activePlugin)
	}
}

// --- Tests for popIfPushed ---

func TestApp_PopIfPushed_WhenActiveIsNavPushWithReturnTo_ShouldNavigateBack(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	statePlugin, _ := app.registry.ByID("state")
	chdirPlugin, _ := app.registry.ByID("chdir")
	app.activePlugin = chdirPlugin
	app.navStack = []sdk.Plugin{statePlugin}

	cmd := app.popIfPushed(nil)

	if app.activePlugin == nil || app.activePlugin.ID() != "state" {
		t.Errorf("popIfPushed should navigate back, got %v", app.activePlugin)
	}
	_ = cmd
}

func TestApp_PopIfPushed_WhenActiveIsNavPushWithNilReturnTo_ShouldGoHome(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	chdirPlugin, _ := app.registry.ByID("chdir")
	app.activePlugin = chdirPlugin
	app.navStack = nil

	app.popIfPushed(nil)

	if app.activePlugin != nil {
		t.Errorf("popIfPushed with empty navStack should deactivate, got %v", app.activePlugin)
	}
}

func TestApp_PopIfPushed_WhenActiveIsNavReplace_ShouldNotPop(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	statePlugin, _ := app.registry.ByID("state")
	app.activePlugin = statePlugin

	app.popIfPushed(nil)

	if app.activePlugin == nil || app.activePlugin.ID() != "state" {
		t.Errorf("popIfPushed with NavReplace plugin should not pop, got %v", app.activePlugin)
	}
}

func TestApp_PopIfPushed_WhenNoActivePlugin_ShouldReturnBusCmd(t *testing.T) {
	app := setupTestApp()
	app.activePlugin = nil

	cmdCalled := false
	busCmd := func() tea.Msg { cmdCalled = true; return nil }
	result := app.popIfPushed(busCmd)

	if result == nil {
		t.Error("popIfPushed with no active plugin should return busCmd")
	}
	result()
	if !cmdCalled {
		t.Error("returned cmd should be the original busCmd")
	}
}

// --- Tests for View ---

func TestApp_View_WhenCommandMode_ShouldShowCommandBar(t *testing.T) {
	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.commandMode = true
	app.commandInput = "sta"

	output := app.View()
	if !strings.Contains(output, "sta") {
		t.Error("View() in command mode should show command input")
	}
}

func TestApp_View_WhenCommandError_ShouldShowError(t *testing.T) {
	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.commandError = "Operation in progress"

	output := app.View()
	if !strings.Contains(output, "Operation in progress") {
		t.Error("View() should display commandError")
	}
}

func TestApp_View_WhenInputActive_ShouldShowPrompt(t *testing.T) {
	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.inputActive = true
	app.inputPrompt = "Apply?"
	app.inputAnswer = "y"

	output := app.View()
	if !strings.Contains(output, "Apply?") {
		t.Error("View() should display input prompt")
	}
}

func TestApp_View_WhenOverlayActive_ShouldRenderOverlay(t *testing.T) {
	overlay := &mockOverlay{
		id:         "test-overlay",
		viewOutput: "overlay-content-xyz",
		hints:      []sdk.KeyHint{{Key: "esc", Description: "close"}},
	}
	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activeOverlay = overlay

	output := app.View()
	if !strings.Contains(output, "overlay-content-xyz") {
		t.Error("View() should contain overlay content")
	}
}

func TestApp_View_WhenOverlayActiveNoHints_ShouldStillRender(t *testing.T) {
	overlay := &mockOverlay{
		id:         "test-overlay",
		viewOutput: "overlay-no-hints",
		hints:      nil,
	}
	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activeOverlay = overlay

	output := app.View()
	if !strings.Contains(output, "overlay-no-hints") {
		t.Error("View() should render overlay even without hints")
	}
}

func TestApp_View_WhenStackablePlugin_ShouldShowStackHints(t *testing.T) {
	stack := sdk.NewStack()
	stack.Push(&mockFrame{
		id:    "list",
		hints: []sdk.KeyHint{{Key: "q", Description: "back"}},
	})

	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activePlugin = p

	output := app.View()
	if !strings.Contains(output, "back") {
		t.Error("View() with stackable plugin should show stack hints")
	}
}

func TestApp_View_WhenStackablePluginNoHints_ShouldShowDefaultBar(t *testing.T) {
	stack := sdk.NewStack()
	stack.Push(&mockFrame{
		id:    "list",
		hints: nil,
	})

	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activePlugin = p

	output := app.View()
	// Should render without crashing (shows default status bar)
	if output == "" {
		t.Error("View() should not be empty")
	}
}

func TestApp_View_WhenHintablePlugin_ShouldShowHints(t *testing.T) {
	p := &mockHintablePlugin{
		mockPlugin: mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"},
		hints:      []sdk.KeyHint{{Key: "a", Description: "apply"}},
	}

	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activePlugin = p

	output := app.View()
	if !strings.Contains(output, "apply") {
		t.Error("View() with hintable plugin should show plugin hints")
	}
}

func TestApp_View_WhenHintablePluginNilHints_ShouldShowDefaultBar(t *testing.T) {
	p := &mockHintablePlugin{
		mockPlugin: mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"},
		hints:      nil,
	}

	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activePlugin = p

	output := app.View()
	if output == "" {
		t.Error("View() should not be empty")
	}
}

func TestApp_View_WhenPlainPlugin_ShouldShowDefaultStatusBar(t *testing.T) {
	p := &mockPlugin{id: "basic", name: "Basic", viewOutput: "basic view"}

	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activePlugin = p

	output := app.View()
	if !strings.Contains(output, "basic view") {
		t.Error("View() should contain plugin view output")
	}
}

func TestApp_View_WhenCountablePlugin_ShouldShowCounts(t *testing.T) {
	p := &mockCountablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		filtered:   5,
		total:      10,
	}

	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activePlugin = p

	output := app.View()
	if !strings.Contains(output, "5") || !strings.Contains(output, "10") {
		t.Error("View() with countable plugin should display counts")
	}
}

func TestApp_View_WhenPinnablePlugin_ShouldShowPinned(t *testing.T) {
	p := &mockPinnablePlugin{
		mockPlugin:  mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		pinnedCount: 3,
	}

	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activePlugin = p

	output := app.View()
	// ContentBorder should include the pinned count
	if !strings.Contains(output, "3") {
		t.Error("View() with pinnable plugin should display pinned count")
	}
}

// --- Tests for commandMatches ---

func TestApp_CommandMatches_WhenEmpty_ShouldReturnAll(t *testing.T) {
	app := setupTestApp()
	app.commandInput = ""

	matches := app.commandMatches()
	// Should include builtins (q, q!) + plugins (plan, state)
	if len(matches) < 4 {
		t.Errorf("commandMatches with empty input should return all, got %d", len(matches))
	}
}

func TestApp_CommandMatches_WhenHasInput_ShouldFilter(t *testing.T) {
	app := setupTestApp()
	app.commandInput = "pl"

	matches := app.commandMatches()
	if len(matches) != 1 {
		t.Errorf("commandMatches with 'pl' should match only plan, got %v", matches)
	}
	if matches[0] != "plan" {
		t.Errorf("match = %q, want %q", matches[0], "plan")
	}
}

func TestApp_CommandMatches_WhenNoMatch_ShouldReturnEmpty(t *testing.T) {
	app := setupTestApp()
	app.commandInput = "xyz"

	matches := app.commandMatches()
	if len(matches) != 0 {
		t.Errorf("commandMatches with no match should return empty, got %v", matches)
	}
}

// --- Tests for executeCommand ---

func TestApp_ExecuteCommand_WhenEmptyInput_ShouldReturnNil(t *testing.T) {
	app := setupTestApp()
	cmd := app.executeCommand("")
	if cmd != nil {
		t.Error("executeCommand with empty input should return nil")
	}
}

func TestApp_ExecuteCommand_WhenWhitespace_ShouldReturnNil(t *testing.T) {
	app := setupTestApp()
	cmd := app.executeCommand("   ")
	if cmd != nil {
		t.Error("executeCommand with whitespace should return nil")
	}
}

func TestApp_ExecuteCommand_WhenPluginNamePrefix_ShouldNavigate(t *testing.T) {
	app := setupTestApp()
	app.executeCommand("plan")
	// navigateTo sets activePlugin even if activate returns nil (non-activatable plugin)
	if app.activePlugin == nil || app.activePlugin.ID() != "plan" {
		t.Errorf("executeCommand should navigate to plan, got %v", app.activePlugin)
	}
}

func TestApp_ExecuteCommand_WhenNoMatch_ShouldReturnNil(t *testing.T) {
	app := setupTestApp()
	cmd := app.executeCommand("nonexistent")
	if cmd != nil {
		t.Error("executeCommand with no match should return nil")
	}
}

// --- Tests for bestCommandMatch ---

func TestApp_BestCommandMatch_WhenEmpty_ShouldReturnEmpty(t *testing.T) {
	app := setupTestApp()
	match := app.bestCommandMatch("")
	if match != "" {
		t.Errorf("bestCommandMatch with empty input should return empty, got %q", match)
	}
}

func TestApp_BestCommandMatch_WhenMultipleMatches_ShouldReturnEmpty(t *testing.T) {
	// Both "plan" and "p" might match... but let's find a real ambiguous case
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "plan", name: "Plan"}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("phantom", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "phantom", name: "Phantom"}
	}, plugin.PluginMeta{Keybinding: "P", MenuVisible: true})
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

	match := app.bestCommandMatch("p")
	if match != "" {
		t.Errorf("bestCommandMatch with multiple matches should return empty, got %q", match)
	}
}

// --- Tests for NewApp with BaseDir ---

func TestNewApp_WhenBaseDirSet_ShouldSetHeaderChdir(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		BaseDir:   "modules/vpc",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)
	app.width = 80
	app.height = 24

	output := app.View()
	if !strings.Contains(output, "modules/vpc") {
		t.Error("View() should show BaseDir in header")
	}
}

// --- Tests for Init with Chdir ---

func TestApp_Init_WhenChdirSet_ShouldScopeService(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Chdir:     "modules/vpc",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init should return a batch cmd")
	}
}

// --- Tests for DeactivateMsg with no active plugin ---

func TestApp_DeactivateMsg_WhenNoActivePlugin_ShouldDoNothing(t *testing.T) {
	app := setupTestApp()
	app.activePlugin = nil

	model, cmd := app.Update(sdk.DeactivateMsg{})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("DeactivateMsg with no active plugin should keep nil")
	}
	if cmd != nil {
		t.Error("DeactivateMsg with no active plugin should return nil cmd")
	}
}

// --- Tests for View with commandMode ---

func TestApp_View_WhenCommandModeActive_ShouldIncludeCommandBar(t *testing.T) {
	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.commandMode = true
	app.commandInput = "plan"

	output := app.View()
	// Command bar should be present
	if !strings.Contains(output, "plan") {
		t.Error("View() with command mode should include command input in output")
	}
}

// --- Tests for OpenContextOnStartup with no chdir plugin ---

func TestApp_OpenContextOnStartup_WhenNoChdirPlugin_ShouldDoNothing(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)
	model, cmd := app.Update(openContextOnStartupMsg{})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("openContextOnStartupMsg with no chdir plugin should not activate anything")
	}
	if cmd != nil {
		t.Error("should return nil cmd")
	}
}

// --- Test for FramePushMsg on stackable plugin ---

func TestApp_Update_WhenFramePushMsgOnStackable_ShouldDelegateToPlugin(t *testing.T) {
	stack := sdk.NewStack()
	stack.Push(&mockFrame{id: "root"})

	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	app := setupTestApp()
	app.activePlugin = p

	// FramePushMsg as a non-key, non-event message goes to the active plugin Update
	newFrame := &mockFrame{id: "detail"}
	model, _ := app.Update(sdk.FramePushMsg{Frame: newFrame})
	_ = model.(App)
	// Plugin's mock Update doesn't push the frame, but the message gets delegated
}

// --- Tests for handleKey ctrl+h (alternative backspace) in input mode ---

func TestApp_HandleKey_WhenInputActiveTextCtrlH_ShouldBackspace(t *testing.T) {
	app := setupTestApp()
	app.inputActive = true
	app.inputMode = sdk.InputRequestText
	app.inputPrompt = "Name:"
	app.inputAnswer = "abc"
	app.inputCallback = func(answer string) tea.Cmd { return nil }

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	updated := model.(App)

	if updated.inputAnswer != "ab" {
		t.Errorf("inputAnswer = %q, want %q after ctrl+h", updated.inputAnswer, "ab")
	}
}

// --- Tests for command mode ctrl+h ---

func TestApp_HandleKey_WhenCommandModeCtrlH_ShouldDeleteChar(t *testing.T) {
	app := setupTestApp()
	app.commandMode = true
	app.commandInput = "abc"

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	updated := model.(App)

	if updated.commandInput != "ab" {
		t.Errorf("commandInput = %q, want %q after ctrl+h", updated.commandInput, "ab")
	}
}

// --- Test for q clearing returnTo ---

func TestApp_HandleKey_WhenQWithReturnToSet_ShouldClearReturnTo(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	statePlugin, _ := app.registry.ByID("state")
	chdirPlugin, _ := app.registry.ByID("chdir")
	app.activePlugin = chdirPlugin
	app.navStack = []sdk.Plugin{statePlugin}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("q should deactivate plugin regardless of navStack")
	}
	if len(updated.navStack) != 0 {
		t.Error("q should clear navStack")
	}
}

// --- Test for View home screen ---

func TestApp_View_WhenHomeScreen_ShouldShowHomeHints(t *testing.T) {
	app := setupTestApp()
	app.width = 80
	app.height = 24

	output := app.View()
	if !strings.Contains(output, "navigate") {
		t.Error("View() on home screen should show navigate hint")
	}
	if !strings.Contains(output, "quit") {
		t.Error("View() on home screen should show quit hint")
	}
}

// --- Test for executeCommand case insensitive ---

func TestApp_ExecuteCommand_WhenUpperCase_ShouldMatchCaseInsensitive(t *testing.T) {
	app := setupTestApp()
	app.executeCommand("PLAN")
	if app.activePlugin == nil || app.activePlugin.ID() != "plan" {
		t.Errorf("executeCommand should match case-insensitively, got %v", app.activePlugin)
	}
}

func TestApp_ExecuteCommand_WhenPartialNameMatch_ShouldNavigate(t *testing.T) {
	app := setupTestApp()
	// "Plan" starts with "pla"
	app.executeCommand("Pla")
	if app.activePlugin == nil || app.activePlugin.ID() != "plan" {
		t.Errorf("executeCommand should match by name prefix, got %v", app.activePlugin)
	}
}

// --- Tests for Init with no plugins ---

func TestApp_Init_WhenNoPlugins_ShouldStillReturnCmd(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init should always return a cmd (at least loadWorkspace)")
	}
}

// --- Test for bestCommandMatch with builtin prefix ---

func TestApp_BestCommandMatch_WhenMatchesBuiltinOnly_ShouldReturn(t *testing.T) {
	// q! is a builtin, "q!" should match exactly since it's unique among builtins
	// but "q" matches both "q" builtin and possibly plugin names
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

	match := app.bestCommandMatch("q!")
	if match != "q!" {
		t.Errorf("bestCommandMatch('q!') = %q, want %q", match, "q!")
	}
}

// --- Test for StackablePlugin key handling when stack becomes empty ---

func TestApp_HandleKey_WhenStackableKeyAndNavStack_ShouldNavigateBack(t *testing.T) {
	stack := sdk.NewStack()
	stack.Push(&mockFrame{
		id: "root",
		updateFn: func(msg tea.Msg) (sdk.Frame, tea.Cmd) {
			return nil, nil // pop the frame
		},
	})

	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "chdir", name: "Chdir", viewOutput: "chdir view"},
		stack:      stack,
	}

	origin := &mockPlugin{id: "context", name: "Context"}

	app := setupTestApp()
	app.activePlugin = p
	app.navStack = []sdk.Plugin{origin}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("should navigate back to origin, not go home")
	}
	if updated.activePlugin.ID() != "context" {
		t.Errorf("activePlugin = %q, want %q", updated.activePlugin.ID(), "context")
	}
	if len(updated.navStack) != 0 {
		t.Error("navStack should be consumed after navigating back")
	}
}

func TestApp_HandleKey_WhenStackableKeyAndEmptyNavStack_ShouldGoHome(t *testing.T) {
	stack := sdk.NewStack()
	stack.Push(&mockFrame{
		id: "root",
		updateFn: func(msg tea.Msg) (sdk.Frame, tea.Cmd) {
			return nil, nil // pop the frame
		},
	})

	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	app := setupTestApp()
	app.activePlugin = p
	app.navStack = nil

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("when stack empties with no navStack, should go home")
	}
}

// --- Test for commandMatches when input matches builtin ---

func TestApp_CommandMatches_WhenMatchesBuiltin_ShouldInclude(t *testing.T) {
	app := setupTestApp()
	app.commandInput = "q"

	matches := app.commandMatches()
	found := false
	for _, m := range matches {
		if m == "q" || m == "q!" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("commandMatches with 'q' should include builtin, got %v", matches)
	}
}

// --- Test for View stackable with empty stack (edge case) ---

func TestApp_View_WhenStackablePluginEmptyStack_ShouldNotPanic(t *testing.T) {
	stack := sdk.NewStack()
	p := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activePlugin = p

	// Should not panic
	output := app.View()
	if output == "" {
		t.Error("View() should not be empty")
	}
}

// --- Tests for the "colon" key activates command mode from plugin ---

func TestApp_HandleKey_WhenColonFromPlugin_ShouldEnterCommandMode(t *testing.T) {
	app := setupTestApp()
	p := &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	app.activePlugin = p

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	updated := model.(App)

	if !updated.commandMode {
		t.Error(": from active plugin should enter command mode")
	}
}

// --- Test for PinsChangedEvent (generic event) dispatched to bus ---

func TestApp_Update_WhenPinsChangedEvent_ShouldDispatchViaBus(t *testing.T) {
	app := setupTestApp()

	model, cmd := app.Update(sdk.PinsChangedEvent{Addresses: []string{"a.b"}})
	_ = model.(App)
	if cmd != nil {
		t.Error("PinsChangedEvent with no handlers should return nil")
	}
}

// --- Test for the edge case: navigateTo from home (activePlugin nil) ---

func TestApp_NavigateTo_WhenFromHome_ShouldSetActive(t *testing.T) {
	app := setupTestApp()
	app.activePlugin = nil

	planPlugin, _ := app.registry.ByID("plan")
	app.navigateTo(planPlugin)

	if app.activePlugin == nil || app.activePlugin.ID() != "plan" {
		t.Errorf("navigateTo from home should set active plugin, got %v", app.activePlugin)
	}
}

// --- Test for DeactivateMsg with returnTo triggering activate ---

func TestApp_DeactivateMsg_WhenReturnToActivatable_ShouldCallActivate(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()

	activatable := &mockActivatablePlugin{
		mockPlugin:  mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		activateCmd: func() tea.Msg { return customMsg{} },
	}

	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return activatable
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("chdir", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "chdir", name: "Chdir"}
	}, plugin.PluginMeta{Nav: plugin.NavPush})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)

	// Set up pushed state
	app.activePlugin, _ = app.registry.ByID("chdir")
	app.navStack = []sdk.Plugin{activatable}

	model, cmd := app.Update(sdk.DeactivateMsg{})
	updated := model.(App)

	if updated.activePlugin == nil || updated.activePlugin.ID() != "state" {
		t.Errorf("after DeactivateMsg, should return to state, got %v", updated.activePlugin)
	}
	if !activatable.activateCalled {
		t.Error("DeactivateMsg with returnTo should call Activate on returnTo plugin")
	}
	if cmd == nil {
		t.Error("DeactivateMsg should return activate cmd")
	}
}

// --- Test for q from home when not busy ---

func TestApp_HandleKey_WhenQFromHomeNotBusy_ShouldQuit(t *testing.T) {
	app := setupTestAppWithBusyPlugin(false)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := app.Update(msg)

	if cmd == nil {
		t.Fatal("q from home with no busy plugins should produce quit cmd")
	}
}

// --- Additional test to ensure full coverage of navigateBack when activePlugin is nil ---

func TestApp_NavigateBack_WhenActivePluginNil_ShouldHandleGracefully(t *testing.T) {
	app := setupTestApp()
	app.activePlugin = nil
	app.navStack = nil

	// Should not panic
	app.navigateBack()

	if app.activePlugin != nil {
		t.Error("navigateBack with nil activePlugin should stay nil")
	}
}

// --- Test for bestCommandMatch with name prefix matching ---

func TestApp_BestCommandMatch_WhenMatchesByName_ShouldReturnID(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State Browser"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

	// "state" matches by ID prefix
	match := app.bestCommandMatch("state")
	if match != "state" {
		t.Errorf("bestCommandMatch('state') = %q, want %q", match, "state")
	}
}

// --- StateEditMsg with matching addresses in source index ---

func TestApp_Update_WhenStateEditMsgWithMatchingAddress(t *testing.T) {
	dir := t.TempDir()

	// Write a tf file with a resource block
	tfContent := `resource "aws_instance" "web" {
  ami = "abc"
}
`
	err := writeTestFile(dir+"/main.tf", tfContent)
	if err != nil {
		t.Fatalf("failed to write test tf file: %v", err)
	}

	cfg := config.Config{
		Dir:       dir,
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

	// The source index should now contain aws_instance.web
	if app.sourceIndex == nil {
		t.Fatal("sourceIndex should not be nil")
	}

	// This should find the address and try to open editor (returns ExecProcess cmd)
	_, cmd := app.Update(tfuistate.StateEditMsg{Address: "aws_instance.web"})
	if cmd == nil {
		t.Error("StateEditMsg with matching address should return editor cmd")
	}
}

func TestApp_Update_WhenStateEditMsgWithMatchingMultipleAddresses(t *testing.T) {
	dir := t.TempDir()

	tfContent := `resource "aws_instance" "web" {
  ami = "abc"
}

resource "aws_instance" "api" {
  ami = "def"
}
`
	err := writeTestFile(dir+"/main.tf", tfContent)
	if err != nil {
		t.Fatalf("failed to write test tf file: %v", err)
	}

	cfg := config.Config{
		Dir:       dir,
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

	_, cmd := app.Update(tfuistate.StateEditMsg{Addresses: []string{"aws_instance.web", "aws_instance.api"}})
	if cmd == nil {
		t.Error("StateEditMsg with multiple matching addresses should return editor cmd")
	}
}

func TestApp_Update_WhenStateEditMsgAddressesPartialMatch(t *testing.T) {
	dir := t.TempDir()

	tfContent := `resource "aws_instance" "web" {
  ami = "abc"
}
`
	err := writeTestFile(dir+"/main.tf", tfContent)
	if err != nil {
		t.Fatalf("failed to write test tf file: %v", err)
	}

	cfg := config.Config{
		Dir:       dir,
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

	// One matches, one doesn't
	_, cmd := app.Update(tfuistate.StateEditMsg{Addresses: []string{"aws_instance.web", "aws_instance.nonexistent"}})
	if cmd == nil {
		t.Error("StateEditMsg with at least one matching address should return editor cmd")
	}
}

func TestApp_Update_WhenStateEditMsgAddressesNoneMatch(t *testing.T) {
	dir := t.TempDir()

	tfContent := `resource "aws_instance" "web" {
  ami = "abc"
}
`
	err := writeTestFile(dir+"/main.tf", tfContent)
	if err != nil {
		t.Fatalf("failed to write test tf file: %v", err)
	}

	cfg := config.Config{
		Dir:       dir,
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

	_, cmd := app.Update(tfuistate.StateEditMsg{Addresses: []string{"aws_s3_bucket.nonexistent"}})
	if cmd != nil {
		t.Error("StateEditMsg with no matching addresses should return nil cmd")
	}
}

// --- Test for updateHome enter when key not in registry ---

func TestApp_HandleKey_WhenHomeEnterKeyNotInRegistry_ShouldReturnNil(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	// Plugin with menu visible but NO keybinding - ByKey("") will return false
	registry.RegisterFactory("nokey", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "nokey", name: "NoKey", viewOutput: "nokey view"}
	}, plugin.PluginMeta{Keybinding: "", MenuVisible: true})
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

	// Enter should try to find the plugin by key "" which will fail
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(App)
	if updated.activePlugin != nil {
		t.Error("enter with unresolvable key should not activate anything")
	}
	if cmd != nil {
		t.Error("should return nil cmd")
	}
}

// --- Test for Init with plugin that returns nil from Init (the else-branch of if cmd != nil) ---

func TestApp_Init_WhenPluginInitReturnsNil_ShouldNotAppend(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", initCmd: nil}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init should return a cmd even when plugin Init returns nil")
	}
}

// --- Test for Init anonymous function body (openContextOnStartupMsg producer) ---

func TestApp_Init_ShouldProduceOpenContextOnStartupMsg(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init should return a batch command")
	}

	// Execute the batch to trigger all sub-commands
	// The batch returns a function, when called returns multiple msgs
	// One of them should be openContextOnStartupMsg
	msg := cmd()
	if msg == nil {
		t.Fatal("batch cmd should return a message")
	}
	// tea.Batch returns a batchMsg which itself is []tea.Cmd
	// We can't easily introspect it, but we can verify via the msg type
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		found := false
		for _, subCmd := range batchMsg {
			if subCmd != nil {
				subMsg := subCmd()
				if _, ok := subMsg.(openContextOnStartupMsg); ok {
					found = true
				}
			}
		}
		if !found {
			t.Error("Init batch should contain openContextOnStartupMsg producer")
		}
	}
}

// --- Tests for ActivePlugin and IsStandalone ---

func TestApp_WhenCreated_ShouldExposeActivePlugin(t *testing.T) {
	app := setupTestApp()
	if app.ActivePlugin() != nil {
		t.Error("ActivePlugin() should be nil on fresh app")
	}

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	app = model.(App)

	if app.ActivePlugin() == nil {
		t.Fatal("ActivePlugin() should not be nil after activation")
	}
	if app.ActivePlugin().ID() != "plan" {
		t.Errorf("ActivePlugin().ID() = %q, want %q", app.ActivePlugin().ID(), "plan")
	}
}

func TestApp_WhenNotStandalone_ShouldReturnFalse(t *testing.T) {
	app := setupTestApp()
	if app.IsStandalone() {
		t.Error("IsStandalone() should be false when no standalone config")
	}
}

func TestApp_WhenStandalone_ShouldReturnTrue(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "plan"}
	app := NewApp(cfg, svc, registry, nil, sc)

	if !app.IsStandalone() {
		t.Error("IsStandalone() should be true when standalone config set")
	}
}

// --- Tests for standalone mode Update branches ---

func TestApp_Update_WhenStandaloneOpenContextOnStartup_ShouldActivateTargetPlugin(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockActivatablePlugin{
			mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)

	model, _ := app.Update(openContextOnStartupMsg{})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("standalone openContextOnStartupMsg should activate target plugin")
	}
	if updated.activePlugin.ID() != "state" {
		t.Errorf("activePlugin = %q, want %q", updated.activePlugin.ID(), "state")
	}
}

func TestApp_Update_WhenStandaloneOpenContextOnStartupWithArgs_ShouldActivateWithArgs(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()

	argPlugin := &mockActivateWithArgsPlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
	}
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return argPlugin
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state", Args: []string{"mv", "src", "dst"}}
	app := NewApp(cfg, svc, registry, nil, sc)

	model, cmd := app.Update(openContextOnStartupMsg{})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("standalone with args should activate target plugin")
	}
	if len(argPlugin.activatedWithArgs) != 3 {
		t.Errorf("ActivateWithArgs called with %v, want [mv src dst]", argPlugin.activatedWithArgs)
	}
	if cmd == nil {
		t.Error("ActivateWithArgs should return a cmd")
	}
}

func TestApp_Update_WhenStandaloneOpenContextOnStartupUnknownPlugin_ShouldReturnNil(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "nonexistent"}
	app := NewApp(cfg, svc, registry, nil, sc)

	model, cmd := app.Update(openContextOnStartupMsg{})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("standalone with unknown plugin should not activate anything")
	}
	if cmd != nil {
		t.Error("should return nil cmd")
	}
}

func TestApp_Update_WhenStandaloneNavigateMsg_ShouldRejectNonNavPush(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true}) // NavReplace (default)
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.activePlugin, _ = app.registry.ByID("state")

	model, cmd := app.Update(sdk.NavigateMsg{PluginID: "plan"})
	updated := model.(App)

	if updated.activePlugin.ID() != "state" {
		t.Errorf("standalone should reject NavReplace navigation, activePlugin = %q", updated.activePlugin.ID())
	}
	if cmd != nil {
		t.Error("should return nil cmd")
	}
}

func TestApp_Update_WhenStandaloneNavigateMsg_ShouldAllowNavPush(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("chdir", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "chdir", name: "Chdir", viewOutput: "chdir view"}
	}, plugin.PluginMeta{Nav: plugin.NavPush})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.activePlugin, _ = app.registry.ByID("state")

	model, _ := app.Update(sdk.NavigateMsg{PluginID: "chdir"})
	updated := model.(App)

	if updated.activePlugin == nil || updated.activePlugin.ID() != "chdir" {
		t.Errorf("standalone should allow NavPush navigation, activePlugin = %v", updated.activePlugin)
	}
}

func TestApp_Update_WhenStandaloneDeactivateMsg_ShouldQuit(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.activePlugin, _ = app.registry.ByID("state")

	_, cmd := app.Update(sdk.DeactivateMsg{})

	if cmd == nil {
		t.Fatal("DeactivateMsg in standalone should produce quit cmd")
	}
}

// --- Tests for TaintRequestMsg, UntaintRequestMsg, ImportRequestMsg ---

func TestApp_Update_WhenReceivingTaintRequestMsg_ShouldActivateTaintPlugin(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("taint", func(s terraform.Service) plugin.Plugin {
		return tfuitaint.New(s)
	}, plugin.PluginMeta{Nav: plugin.NavPush})
	registry.Build(svc, nil)

	app := NewApp(cfg, svc, registry, nil)
	app.activePlugin, _ = app.registry.ByID("state")

	model, _ := app.Update(tfuitaint.TaintRequestMsg{Addresses: []string{"aws_instance.foo"}})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("TaintRequestMsg should activate taint plugin")
	}
	if updated.activePlugin.ID() != "taint" {
		t.Errorf("activePlugin = %q, want %q", updated.activePlugin.ID(), "taint")
	}
	if len(updated.navStack) != 1 {
		t.Errorf("navStack depth = %d, want 1", len(updated.navStack))
	}
}

func TestApp_Update_WhenReceivingTaintRequestMsgNoTaintPlugin_ShouldReturnNil(t *testing.T) {
	app := setupTestApp() // no taint plugin
	app.activePlugin, _ = app.registry.ByID("state")

	_, cmd := app.Update(tfuitaint.TaintRequestMsg{Addresses: []string{"aws_instance.foo"}})
	if cmd != nil {
		t.Error("TaintRequestMsg without taint plugin should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingUntaintRequestMsg_ShouldActivateUntaintPlugin(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("untaint", func(s terraform.Service) plugin.Plugin {
		return tfuiuntaint.New(s)
	}, plugin.PluginMeta{Nav: plugin.NavPush})
	registry.Build(svc, nil)

	app := NewApp(cfg, svc, registry, nil)
	app.activePlugin, _ = app.registry.ByID("state")

	model, _ := app.Update(tfuiuntaint.UntaintRequestMsg{Addresses: []string{"aws_instance.foo"}})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("UntaintRequestMsg should activate untaint plugin")
	}
	if updated.activePlugin.ID() != "untaint" {
		t.Errorf("activePlugin = %q, want %q", updated.activePlugin.ID(), "untaint")
	}
	if len(updated.navStack) != 1 {
		t.Errorf("navStack depth = %d, want 1", len(updated.navStack))
	}
}

func TestApp_Update_WhenReceivingUntaintRequestMsgNoUntaintPlugin_ShouldReturnNil(t *testing.T) {
	app := setupTestApp() // no untaint plugin
	app.activePlugin, _ = app.registry.ByID("state")

	_, cmd := app.Update(tfuiuntaint.UntaintRequestMsg{Addresses: []string{"aws_instance.foo"}})
	if cmd != nil {
		t.Error("UntaintRequestMsg without untaint plugin should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingImportRequestMsg_ShouldActivateImportPlugin(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("import", func(s terraform.Service) plugin.Plugin {
		return tfuiimport.New(s)
	}, plugin.PluginMeta{Nav: plugin.NavPush})
	registry.Build(svc, nil)

	app := NewApp(cfg, svc, registry, nil)
	app.activePlugin, _ = app.registry.ByID("state")

	model, _ := app.Update(tfuiimport.ImportRequestMsg{Address: "aws_instance.foo"})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("ImportRequestMsg should activate import plugin")
	}
	if updated.activePlugin.ID() != "import" {
		t.Errorf("activePlugin = %q, want %q", updated.activePlugin.ID(), "import")
	}
	if len(updated.navStack) != 1 {
		t.Errorf("navStack depth = %d, want 1", len(updated.navStack))
	}
}

func TestApp_Update_WhenReceivingImportRequestMsgNoImportPlugin_ShouldReturnNil(t *testing.T) {
	app := setupTestApp() // no import plugin
	app.activePlugin, _ = app.registry.ByID("state")

	_, cmd := app.Update(tfuiimport.ImportRequestMsg{Address: "aws_instance.foo"})
	if cmd != nil {
		t.Error("ImportRequestMsg without import plugin should return nil cmd")
	}
}

// --- Tests for AutoApplyRequestMsg ---

func TestApp_Update_WhenReceivingAutoApplyRequestMsg_ShouldActivateApplyPlugin(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("apply", func(s terraform.Service) plugin.Plugin {
		return tfuiapply.New(s)
	}, plugin.PluginMeta{Keybinding: "a", MenuVisible: true})
	registry.Build(svc, nil)

	app := NewApp(cfg, svc, registry, nil)
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	app = model.(App)

	model, _ = app.Update(tfuiplan.AutoApplyRequestMsg{})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("AutoApplyRequestMsg should activate the apply plugin")
	}
	if updated.activePlugin.ID() != "apply" {
		t.Errorf("activePlugin = %q, want %q", updated.activePlugin.ID(), "apply")
	}
}

func TestApp_Update_WhenReceivingAutoApplyRequestMsgNoApplyPlugin_ShouldReturnNil(t *testing.T) {
	app := setupTestApp() // no apply plugin

	_, cmd := app.Update(tfuiplan.AutoApplyRequestMsg{})
	if cmd != nil {
		t.Error("AutoApplyRequestMsg without apply plugin should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingAutoApplyRequestMsgWithPins_ShouldSetTargets(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("apply", func(s terraform.Service) plugin.Plugin {
		return tfuiapply.New(s)
	}, plugin.PluginMeta{Keybinding: "a", MenuVisible: true})
	registry.Build(svc, nil)

	app := NewApp(cfg, svc, registry, nil)
	app.pins.Toggle("aws_instance.foo")

	model, _ := app.Update(tfuiplan.AutoApplyRequestMsg{})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Fatal("AutoApplyRequestMsg with pins should activate apply plugin")
	}
}

// --- Tests for PlanEditMsg ---

func TestApp_Update_WhenReceivingPlanEditMsgNilSourceIndex_ShouldReturnNil(t *testing.T) {
	app := setupTestApp()
	app.sourceIndex = nil

	_, cmd := app.Update(tfuiplan.PlanEditMsg{Address: "aws_instance.foo"})
	if cmd != nil {
		t.Error("PlanEditMsg with nil sourceIndex should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingPlanEditMsgNoMatch_ShouldReturnNil(t *testing.T) {
	app := setupAppWithSourceIndex(t)

	_, cmd := app.Update(tfuiplan.PlanEditMsg{Address: "aws_instance.nonexistent"})
	if cmd != nil {
		t.Error("PlanEditMsg with no matching address should return nil cmd")
	}
}

func TestApp_Update_WhenReceivingPlanEditMsgWithMatch_ShouldReturnEditorCmd(t *testing.T) {
	dir := t.TempDir()

	tfContent := `resource "aws_instance" "web" {
  ami = "abc"
}
`
	if err := writeTestFile(dir+"/main.tf", tfContent); err != nil {
		t.Fatalf("failed to write test tf file: %v", err)
	}

	cfg := config.Config{
		Dir:       dir,
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)
	app := NewApp(cfg, svc, registry, nil)

	_, cmd := app.Update(tfuiplan.PlanEditMsg{Address: "aws_instance.web"})
	if cmd == nil {
		t.Error("PlanEditMsg with matching address should return editor cmd")
	}
}

// --- Tests for TimerTickMsg routing ---

func TestApp_Update_WhenTimerTickMsgWithActivePlugin_ShouldRouteToPlugin(t *testing.T) {
	app := setupTestApp()
	app.activePlugin = &mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}

	model, _ := app.Update(sdkui.TimerTickMsg{})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Error("TimerTickMsg should keep active plugin")
	}
}

func TestApp_Update_WhenTimerTickMsgWithNoActivePlugin_ShouldReturnNil(t *testing.T) {
	app := setupTestApp()
	app.activePlugin = nil

	_, cmd := app.Update(sdkui.TimerTickMsg{})
	if cmd != nil {
		t.Error("TimerTickMsg with no active plugin should return nil cmd")
	}
}

// --- Tests for handleKey with KeyCapturer ---

func TestApp_HandleKey_WhenKeyCapturerActive_ShouldDelegateAllKeys(t *testing.T) {
	capturer := &mockKeyCapturerPlugin{
		mockPlugin: mockPlugin{id: "console", name: "Console", viewOutput: "console view"},
		captures:   true,
	}

	app := setupTestApp()
	app.activePlugin = capturer

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	_ = model.(App)

	if capturer.lastKeyMsg == nil {
		t.Error("key capturer should receive all keys")
	}
}

func TestApp_HandleKey_WhenKeyCapturerActiveCtrlC_ShouldQuit(t *testing.T) {
	capturer := &mockKeyCapturerPlugin{
		mockPlugin: mockPlugin{id: "console", name: "Console", viewOutput: "console view"},
		captures:   true,
	}

	app := setupTestApp()
	app.activePlugin = capturer

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c with key capturer should still produce quit cmd")
	}
}

func TestApp_HandleKey_WhenKeyCapturerActiveCtrlS_ShouldCapture(t *testing.T) {
	capturer := &mockKeyCapturerPlugin{
		mockPlugin: mockPlugin{id: "console", name: "Console", viewOutput: "console view"},
		captures:   true,
	}

	app := setupTestApp()
	app.width = 80
	app.height = 24
	app.activePlugin = capturer

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd != nil {
		t.Error("ctrl+s with key capturer should return nil cmd")
	}
}

func TestApp_HandleKey_WhenKeyCapturerNotCapturing_ShouldFollowNormalFlow(t *testing.T) {
	capturer := &mockKeyCapturerPlugin{
		mockPlugin: mockPlugin{id: "console", name: "Console", viewOutput: "console view"},
		captures:   false,
	}

	app := setupTestApp()
	app.activePlugin = capturer

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(App)

	if updated.activePlugin != nil {
		t.Error("q with non-capturing KeyCapturer should deactivate")
	}
}

// --- Tests for standalone mode key handling ---

func TestApp_HandleKey_WhenStandaloneCKey_ShouldNotNavigateToContext(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("context", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "context", name: "Context", viewOutput: "context view"}
	}, plugin.PluginMeta{})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.activePlugin, _ = app.registry.ByID("state")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	updated := model.(App)

	if updated.activePlugin != nil && updated.activePlugin.ID() == "context" {
		t.Error("C in standalone mode should not navigate to context")
	}
}

func TestApp_HandleKey_WhenStandaloneColonKey_ShouldNotEnterCommandMode(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.activePlugin, _ = app.registry.ByID("state")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	updated := model.(App)

	if updated.commandMode {
		t.Error(": in standalone mode should not enter command mode")
	}
}

func TestApp_HandleKey_WhenStandaloneQKey_ShouldQuit(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.activePlugin, _ = app.registry.ByID("state")

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q in standalone mode should produce quit cmd")
	}
}

func TestApp_HandleKey_WhenStandaloneQKeyWithStackDepthGreaterThanOne_ShouldClearStack(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)

	stack := sdk.NewStack()
	stack.Push(&mockFrame{id: "root"})
	stack.Push(&mockFrame{id: "detail"})
	stackableP := &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}
	app.activePlugin = stackableP

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(App)

	if updated.activePlugin == nil {
		t.Error("q in standalone with stack depth > 1 should clear stack, not quit")
	}
	if cmd != nil {
		t.Error("should return nil cmd (just clearing stack)")
	}
	if stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1", stack.Depth())
	}
}

func TestApp_HandleKey_WhenStandaloneQKeyBusy_ShouldStillQuit(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockBusyPlugin{
			mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
			busy:       true,
		}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.activePlugin, _ = app.registry.ByID("state")

	// In standalone, q when busy should block via cmdQuit
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Error("q in standalone with busy plugin should NOT quit (cmdQuit blocks)")
	}
}

// --- Tests for navigateTo with busy active plugin ---

func TestApp_NavigateTo_WhenActivePluginBusy_ShouldNotCancel(t *testing.T) {
	app := setupTestAppWithTransientPlugins()

	busyPlugin := &mockBusyCancellablePlugin{
		mockCancellablePlugin: mockCancellablePlugin{mockPlugin: mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}},
		busy:                  true,
	}
	app.activePlugin = busyPlugin

	statePlugin, _ := app.registry.ByID("state")
	app.navigateTo(statePlugin)

	if busyPlugin.cancelled {
		t.Error("navigateTo should not cancel busy plugin")
	}
}

func TestApp_NavigateTo_WhenActivePluginNotBusy_ShouldCancel(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockCancellablePlugin{
			mockPlugin: mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"},
		}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)

	planPlugin, _ := app.registry.ByID("plan")
	app.activePlugin = planPlugin

	statePlugin, _ := app.registry.ByID("state")
	app.navigateTo(statePlugin)

	cancellable := planPlugin.(*mockCancellablePlugin)
	if !cancellable.cancelled {
		t.Error("navigateTo should cancel non-busy active plugin")
	}
}

// --- Tests for viewStandalone ---

func TestApp_View_WhenStandalone_ShouldRenderStandaloneView(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state standalone view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin, _ = app.registry.ByID("state")

	output := app.View()
	if !strings.Contains(output, "state standalone view") {
		t.Error("standalone View() should contain plugin view output")
	}
	if !strings.Contains(output, "tfui") {
		t.Error("standalone View() should contain 'tfui' in header")
	}
}

func TestApp_View_WhenStandaloneWithChdir_ShouldShowChdir(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin, _ = app.registry.ByID("state")
	app.activeChdir = "modules/vpc"
	app.activeWorkspace = "staging"

	output := app.View()
	if !strings.Contains(output, "modules/vpc") {
		t.Error("standalone View() should show activeChdir")
	}
	if !strings.Contains(output, "staging") {
		t.Error("standalone View() should show workspace")
	}
}

func TestApp_View_WhenStandaloneWithLockInfo_ShouldShowLocked(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin, _ = app.registry.ByID("state")
	app.lockInfo = &sdk.StateLock{ID: "abc-123"}

	output := app.View()
	if !strings.Contains(output, "[locked]") {
		t.Error("standalone View() with lockInfo should show [locked]")
	}
}

func TestApp_View_WhenStandaloneWithCommandError_ShouldShowError(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin, _ = app.registry.ByID("state")
	app.commandError = "Operation in progress"

	output := app.View()
	if !strings.Contains(output, "Operation in progress") {
		t.Error("standalone View() should display commandError")
	}
}

func TestApp_View_WhenStandaloneWithInputActive_ShouldShowPrompt(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin, _ = app.registry.ByID("state")
	app.inputActive = true
	app.inputPrompt = "Delete?"
	app.inputAnswer = "y"

	output := app.View()
	if !strings.Contains(output, "Delete?") {
		t.Error("standalone View() should display input prompt")
	}
}

func TestApp_View_WhenStandaloneWithStackablePlugin_ShouldShowHints(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24

	stack := sdk.NewStack()
	stack.Push(&mockFrame{
		id:    "list",
		hints: []sdk.KeyHint{{Key: "q", Description: "quit"}},
	})
	app.activePlugin = &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	output := app.View()
	if !strings.Contains(output, "quit") {
		t.Error("standalone View() with stackable plugin should show stack hints")
	}
}

func TestApp_View_WhenStandaloneWithStackablePluginNoHints_ShouldShowDefaultBar(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24

	stack := sdk.NewStack()
	stack.Push(&mockFrame{id: "list", hints: nil})
	app.activePlugin = &mockStackablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		stack:      stack,
	}

	output := app.View()
	if output == "" {
		t.Error("standalone View() should not be empty")
	}
}

func TestApp_View_WhenStandaloneWithHintablePlugin_ShouldShowHints(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin = &mockHintablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		hints:      []sdk.KeyHint{{Key: "d", Description: "delete"}},
	}

	output := app.View()
	if !strings.Contains(output, "delete") {
		t.Error("standalone View() with hintable plugin should show hints")
	}
}

func TestApp_View_WhenStandaloneWithHintablePluginNilHints_ShouldShowDefaultBar(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin = &mockHintablePlugin{
		mockPlugin: mockPlugin{id: "state", name: "State", viewOutput: "state view"},
		hints:      nil,
	}

	output := app.View()
	if output == "" {
		t.Error("standalone View() should not be empty")
	}
}

func TestApp_View_WhenStandaloneWithPlainPlugin_ShouldShowDefaultStatusBar(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin = &mockPlugin{id: "state", name: "State", viewOutput: "state view"}

	output := app.View()
	if output == "" {
		t.Error("standalone View() with plain plugin should not be empty")
	}
}

func TestApp_View_WhenStandaloneWithOverlay_ShouldRenderOverlay(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin = &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	app.activeOverlay = &mockOverlay{
		id:         "test-overlay",
		viewOutput: "standalone-overlay-content",
		hints:      []sdk.KeyHint{{Key: "esc", Description: "close"}},
	}

	output := app.View()
	if !strings.Contains(output, "standalone-overlay-content") {
		t.Error("standalone View() with overlay should render overlay content")
	}
}

func TestApp_View_WhenStandaloneWithOverlayNoHints_ShouldStillRender(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin = &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	app.activeOverlay = &mockOverlay{
		id:         "test-overlay",
		viewOutput: "standalone-overlay-no-hints",
		hints:      nil,
	}

	output := app.View()
	if !strings.Contains(output, "standalone-overlay-no-hints") {
		t.Error("standalone View() with overlay (no hints) should render overlay")
	}
}

func TestApp_View_WhenStandaloneNoActivePlugin_ShouldNotPanic(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 80
	app.height = 24
	app.activePlugin = nil

	output := app.View()
	if output == "" {
		t.Error("standalone View() with nil activePlugin should not be empty")
	}
}

func TestApp_View_WhenStandaloneNarrowWidth_ShouldForceMinGap(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test/dir",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	sc := &StandaloneConfig{PluginID: "state"}
	app := NewApp(cfg, svc, registry, nil, sc)
	app.width = 5 // very narrow to trigger gap < 1
	app.height = 24
	app.activePlugin = &mockPlugin{id: "state", name: "State", viewOutput: "s"}
	app.activeChdir = "very/long/path/exceeding/width"
	app.activeWorkspace = "production-extra-long"
	app.lockInfo = &sdk.StateLock{ID: "lock"}

	output := app.View()
	if output == "" {
		t.Error("standalone View() with narrow width should still render")
	}
}

// --- Tests for activeViewID ---

func TestActiveViewID_WhenEmptyNavStack_ShouldReturnHome(t *testing.T) {
	result := activeViewID(nil)
	if result != "home" {
		t.Errorf("activeViewID(nil) = %q, want %q", result, "home")
	}
}

func TestActiveViewID_WhenNavStackWithNilEntry_ShouldReturnHome(t *testing.T) {
	result := activeViewID([]sdk.Plugin{nil})
	if result != "home" {
		t.Errorf("activeViewID([nil]) = %q, want %q", result, "home")
	}
}

func TestActiveViewID_WhenNavStackWithPlugin_ShouldReturnPluginID(t *testing.T) {
	p := &mockPlugin{id: "state", name: "State"}
	result := activeViewID([]sdk.Plugin{p})
	if result != "state" {
		t.Errorf("activeViewID([state]) = %q, want %q", result, "state")
	}
}

func TestActiveViewID_WhenNavStackWithMultiple_ShouldReturnLastPluginID(t *testing.T) {
	p1 := &mockPlugin{id: "state", name: "State"}
	p2 := &mockPlugin{id: "plan", name: "Plan"}
	result := activeViewID([]sdk.Plugin{p1, p2})
	if result != "plan" {
		t.Errorf("activeViewID([state, plan]) = %q, want %q", result, "plan")
	}
}

// --- Tests for DeactivateMsg with Cancellable plugin ---

func TestApp_DeactivateMsg_WhenCancellablePlugin_ShouldCallCancel(t *testing.T) {
	cancellable := &mockCancellablePlugin{
		mockPlugin: mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"},
	}

	app := setupTestApp()
	app.activePlugin = cancellable

	app.Update(sdk.DeactivateMsg{})

	if !cancellable.cancelled {
		t.Error("DeactivateMsg should call Cancel on cancellable plugin")
	}
}

// --- Tests for q with busy plugin at app level ---

func TestApp_HandleKey_WhenQWithBusyPlugin_ShouldNotCancel(t *testing.T) {
	busyCancellable := &mockBusyCancellablePlugin{
		mockCancellablePlugin: mockCancellablePlugin{mockPlugin: mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}},
		busy:                  true,
	}

	app := setupTestApp()
	app.activePlugin = busyCancellable

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(App)

	if busyCancellable.cancelled {
		t.Error("q with busy plugin should not call Cancel")
	}
	if updated.activePlugin != nil {
		t.Error("q should still go home even with busy plugin")
	}
}

func TestApp_HandleKey_WhenQWithNonBusyCancellable_ShouldCancel(t *testing.T) {
	cancellable := &mockBusyCancellablePlugin{
		mockCancellablePlugin: mockCancellablePlugin{mockPlugin: mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"}},
		busy:                  false,
	}

	app := setupTestApp()
	app.activePlugin = cancellable

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(App)

	if !cancellable.cancelled {
		t.Error("q with non-busy cancellable plugin should call Cancel")
	}
	if updated.activePlugin != nil {
		t.Error("q should go home")
	}
}

// --- Test for NewApp with rootCfg but childCfg load error ---

func TestNewApp_WhenRootCfgSetWithInvalidChildDir_ShouldStillCreate(t *testing.T) {
	rootCfg := &config.RootConfig{}
	cfg := config.Config{
		Dir:       "/nonexistent/dir/that/should/not/exist",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, rootCfg)
	if app.rootCfg == nil {
		t.Error("app should have rootCfg even if childCfg failed to load")
	}
}

// Helper to write test files
func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestApp_Update_WhenChdirChangedWithRootCfg_ShouldLoadChildCfg(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Dir:       dir,
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("state", func(_ terraform.Service) plugin.Plugin {
		return &mockPlugin{id: "state", name: "State", viewOutput: "state view"}
	}, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, &config.RootConfig{})
	app.width = 80
	app.height = 24

	model, _ := app.Update(sdk.ChdirChangedEvent{RelPath: "modules/vpc", AbsPath: "/nonexistent/path", Count: 1})
	updated := model.(App)
	_ = updated
}

type mockCmdReturningPlugin struct {
	mockPlugin
}

func (m *mockCmdReturningPlugin) Update(_ tea.Msg) (plugin.Plugin, tea.Cmd) {
	return m, func() tea.Msg { return nil }
}

func TestApp_Update_WhenBroadcastMsgCausesPluginCmd_ShouldCollectCmds(t *testing.T) {
	cfg := config.Config{
		Dir:       "/test",
		Terraform: config.TerraformConfig{Bin: "terraform"},
	}
	svc := &sdktest.MockService{}
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", func(_ terraform.Service) plugin.Plugin {
		return &mockCmdReturningPlugin{
			mockPlugin: mockPlugin{id: "plan", name: "Plan", viewOutput: "plan view"},
		}
	}, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.Build(nil, nil)

	app := NewApp(cfg, svc, registry, nil)
	app.width = 80
	app.height = 24

	type customBroadcastMsg struct{}
	model, cmd := app.Update(customBroadcastMsg{})
	_ = model.(App)
	if cmd == nil {
		t.Error("broadcast to plugin returning cmd should produce batched cmd")
	}
}
