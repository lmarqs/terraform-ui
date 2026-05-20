package context

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestPlugin_Lifecycle(t *testing.T) {
	p := New(nil).(*Plugin)

	if p.ID() != "context" {
		t.Errorf("ID() = %q, want %q", p.ID(), "context")
	}
	if p.Name() != "Context" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Context")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if !p.Ready() {
		t.Error("Ready() should always be true")
	}
	if p.Stack() == nil {
		t.Error("Stack() should not be nil")
	}
	if err := p.Configure(nil); err != nil {
		t.Errorf("Configure(nil) = %v, want nil", err)
	}
	if err := p.Configure(map[string]interface{}{"unknown": "value"}); err != nil {
		t.Errorf("Configure(unknown keys) = %v, want nil", err)
	}
}

func TestPlugin_WhenSetProjectDir_ShouldStoreDirectory(t *testing.T) {
	p := New(nil).(*Plugin)

	p.SetProjectDir("/my/project")
	if p.projectDir != "/my/project" {
		t.Errorf("projectDir = %q, want %q", p.projectDir, "/my/project")
	}
}

func TestPlugin_WhenSetMembers_ShouldStoreMembers(t *testing.T) {
	p := New(nil).(*Plugin)

	members := []string{"modules/vpc", "modules/ecs"}
	p.SetMembers(members)
	if len(p.members) != 2 {
		t.Errorf("members length = %d, want 2", len(p.members))
	}
}

func TestPlugin_WhenInitialized_ShouldStoreContext(t *testing.T) {
	tests := []struct {
		name      string
		ctx       *sdk.Context
		wantWS    string
		wantNilFn bool
	}{
		{
			name: "ShouldStoreWorkspaceAndLogger",
			ctx: &sdk.Context{
				Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
				Workspace: "staging",
			},
			wantWS: "staging",
		},
		{
			name: "ShouldKeepDefaultLoggerWhenNilLogger",
			ctx: &sdk.Context{
				Logger:    nil,
				Workspace: "production",
			},
			wantWS: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(nil).(*Plugin)
			cmd := p.Init(tt.ctx)

			if cmd != nil {
				t.Error("Init() should return nil cmd")
			}
			if p.workspace != tt.wantWS {
				t.Errorf("workspace = %q, want %q", p.workspace, tt.wantWS)
			}
			if p.log == nil {
				t.Error("log should not be nil after Init")
			}
		})
	}
}

func TestPlugin_WhenInitializedWithLogger_ShouldUseProvidedLogger(t *testing.T) {
	p := New(nil).(*Plugin)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	p.Init(&sdk.Context{Logger: logger, Workspace: "dev"})

	if p.log != logger {
		t.Error("should use provided logger")
	}
}

func TestPlugin_WhenInitializedWithNilLogger_ShouldKeepDefaultLogger(t *testing.T) {
	p := New(nil).(*Plugin)
	defaultLog := p.log

	p.Init(&sdk.Context{Logger: nil, Workspace: "dev"})

	if p.log != defaultLog {
		t.Error("should keep default logger when ctx.Logger is nil")
	}
}

func TestPlugin_WhenChdirChanged_ShouldUpdateChdir(t *testing.T) {
	p := New(nil).(*Plugin)

	cmd := p.HandleChdirChanged(sdk.ChdirChangedEvent{RelPath: "modules/vpc"})
	if cmd != nil {
		t.Error("HandleChdirChanged should return nil cmd")
	}
	if p.chdir != "modules/vpc" {
		t.Errorf("chdir = %q, want %q", p.chdir, "modules/vpc")
	}
}

func TestPlugin_WhenWorkspaceChanged_ShouldUpdateWorkspace(t *testing.T) {
	p := New(nil).(*Plugin)

	cmd := p.HandleWorkspaceChanged(sdk.WorkspaceChangedEvent{Name: "staging"})
	if cmd != nil {
		t.Error("HandleWorkspaceChanged should return nil cmd")
	}
	if p.workspace != "staging" {
		t.Errorf("workspace = %q, want %q", p.workspace, "staging")
	}
}

func TestPlugin_WhenActivated_ShouldPushFormFrame(t *testing.T) {
	p := New(nil).(*Plugin)

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() should return nil cmd")
	}
	if p.stack.Depth() != 1 {
		t.Errorf("stack depth = %d, want 1", p.stack.Depth())
	}
}

func TestPlugin_WhenActivatedWithSubFrames_ShouldClearAndPush(t *testing.T) {
	p := New(nil).(*Plugin)

	p.Activate()
	// Push a sub-frame to simulate navigation within the form
	p.stack.Push(p.buildForm())
	if p.stack.Depth() != 2 {
		t.Fatalf("stack depth = %d, want 2 before re-activate", p.stack.Depth())
	}

	// Activate clears sub-frames (keeping root) then pushes a new form
	p.Activate()
	if p.stack.Depth() != 2 {
		t.Errorf("stack depth = %d, want 2 (root kept by Clear + new push)", p.stack.Depth())
	}
}

func TestPlugin_WhenUpdated_ShouldReturnSelf(t *testing.T) {
	p := New(nil).(*Plugin)

	result, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if result != p {
		t.Error("Update should return self")
	}
	if cmd != nil {
		t.Error("Update should return nil cmd")
	}
}

func TestPlugin_WhenUpdatedWithNonKeyMsg_ShouldReturnSelf(t *testing.T) {
	p := New(nil).(*Plugin)

	result, cmd := p.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if result != p {
		t.Error("Update should return self for non-key messages")
	}
	if cmd != nil {
		t.Error("Update should return nil cmd for non-key messages")
	}
}

func TestPlugin_WhenViewedWithEmptyStack_ShouldReturnEmpty(t *testing.T) {
	p := New(nil).(*Plugin)

	output := p.View(80, 20)
	if output != "" {
		t.Errorf("View() with empty stack = %q, want empty", output)
	}
}

func TestPlugin_WhenViewedWithForm_ShouldShowValues(t *testing.T) {
	p := New(nil).(*Plugin)
	p.projectDir = "/my/project"
	p.chdir = "modules/east"
	p.workspace = "production"
	p.Activate()

	output := p.View(80, 20)
	if !strings.Contains(output, "/my/project") {
		t.Error("should show project dir in view")
	}
	if !strings.Contains(output, "modules/east") {
		t.Error("should show chdir in view")
	}
	if !strings.Contains(output, "production") {
		t.Error("should show workspace in view")
	}
}

func TestPlugin_WhenProjectDirEmpty_ShouldShowDot(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	output := p.View(80, 20)
	if !strings.Contains(output, ".") {
		t.Error("should show '.' for empty project dir")
	}
}

func TestPlugin_WhenChdirEmpty_ShouldShowDash(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	output := p.View(80, 20)
	if !strings.Contains(output, "-") {
		t.Error("should show '-' for empty chdir")
	}
}

func TestPlugin_WhenWorkspaceEmpty_ShouldShowDefault(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	output := p.View(80, 20)
	if !strings.Contains(output, "default") {
		t.Error("should show 'default' for empty workspace")
	}
}

func TestPlugin_WhenFormNavigated_ShouldHandleEnterOnChdir(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc", "modules/ecs"})
	p.Activate()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from chdir selection")
	}

	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.PluginID != "chdir" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "chdir")
	}
}

func TestPlugin_WhenFormNavigated_ShouldHandleEnterOnWorkspace(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from workspace selection")
	}

	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.PluginID != "workspace" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "workspace")
	}
}

func TestPlugin_WhenFormNavigated_ShouldMoveCursorDown(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc"})
	p.Activate()

	// Move down from chdir (index 1) to workspace (index 2)
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from workspace selection after moving down")
	}

	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.PluginID != "workspace" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "workspace")
	}
}

func TestPlugin_WhenFormNavigated_ShouldMoveCursorUp(t *testing.T) {
	p := New(nil).(*Plugin)
	p.SetMembers([]string{"modules/vpc"})
	p.Activate()

	// Move down then back up
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from chdir selection after moving up")
	}

	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.PluginID != "chdir" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "chdir")
	}
}

func TestPlugin_WhenFormEscPressed_ShouldPopFrame(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.stack.Depth() != 0 {
		t.Errorf("stack depth = %d, want 0 after esc", p.stack.Depth())
	}
}

func TestPlugin_WhenFormQPressed_ShouldNotPopFrame(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if p.stack.Depth() == 0 {
		t.Error("q should not pop form frame (app handles q globally)")
	}
}

func TestPlugin_WhenValuesSet_ShouldRenderSetValues(t *testing.T) {
	p := New(nil).(*Plugin)
	p.projectDir = "/custom/path"
	p.chdir = "modules/vpc"
	p.workspace = "staging"
	p.Activate()

	output := p.View(80, 20)
	if !strings.Contains(output, "/custom/path") {
		t.Error("should show custom project dir")
	}
	if !strings.Contains(output, "modules/vpc") {
		t.Error("should show custom chdir")
	}
	if !strings.Contains(output, "staging") {
		t.Error("should show custom workspace")
	}
}

func TestPlugin_WhenChdirNotSelectable_ShouldSkipToWorkspace(t *testing.T) {
	p := New(nil).(*Plugin)
	p.Activate()

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected workspace navigate command")
	}

	msg := cmd()
	nav, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.PluginID != "workspace" {
		t.Errorf("PluginID = %q, want %q", nav.PluginID, "workspace")
	}
}
