package context

import (
	"io"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
)

// Plugin implements the context dashboard — shows Project, Chdir, Workspace.
type Plugin struct {
	cfg       config.Config
	log       *slog.Logger
	stack     *sdk.Stack
	chdir     string
	workspace string
	members   []string
}

// New creates a new context plugin.
func New(_ sdk.Service) sdk.Plugin {
	p := &Plugin{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.stack = sdk.NewStack()
	return p
}

func (p *Plugin) ID() string          { return "context" }
func (p *Plugin) Name() string        { return "Context" }
func (p *Plugin) Description() string { return "View and manage working context" }
func (p *Plugin) Ready() bool         { return true }
func (p *Plugin) Stack() *sdk.Stack   { return p.stack }

// Configure applies plugin-specific options from config.
func (p *Plugin) Configure(opts map[string]interface{}) error {
	return nil
}

// SetConfig provides the application configuration.
func (p *Plugin) SetConfig(cfg config.Config) {
	p.cfg = cfg
}

// SetMembers provides the list of chdir members and project directory.
func (p *Plugin) SetMembers(members []string, _ string) {
	p.members = members
}

// Init initializes the plugin with shared context.
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	if ctx.Logger != nil {
		p.log = ctx.Logger
	}
	p.workspace = ctx.Workspace
	return nil
}

// HandleChdirChanged implements sdk.ChdirHandler.
func (p *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	p.chdir = evt.RelPath
	return nil
}

// HandleWorkspaceChanged implements sdk.WorkspaceHandler.
func (p *Plugin) HandleWorkspaceChanged(evt sdk.WorkspaceChangedEvent) tea.Cmd {
	p.workspace = evt.Name
	return nil
}

// Activate builds the form frame and pushes it onto the stack.
func (p *Plugin) Activate() tea.Cmd {
	p.stack.Clear()
	p.stack.Push(p.buildForm())
	return nil
}

func (p *Plugin) buildForm() *frames.FormFrame {
	return frames.NewFormFrame(frames.FormOpts{
		Fields: []frames.FormField{
			{
				Label:      "Project",
				Value:      p.projectValue,
				Selectable: false,
			},
			{
				Label:      "Chdir",
				Value:      p.chdirValue,
				Selectable: len(p.members) > 0,
				OnSelect:   p.openChdirPicker,
			},
			{
				Label:      "Workspace",
				Value:      p.workspaceValue,
				Selectable: true,
				OnSelect:   p.openWorkspacePicker,
			},
		},
	})
}

func (p *Plugin) projectValue() string {
	if p.cfg.Dir != "" {
		return p.cfg.Dir
	}
	return "."
}

func (p *Plugin) chdirValue() string {
	if p.chdir != "" {
		return p.chdir
	}
	return "-"
}

func (p *Plugin) workspaceValue() string {
	if p.workspace != "" {
		return p.workspace
	}
	return "default"
}

func (p *Plugin) openChdirPicker() tea.Cmd {
	return func() tea.Msg {
		return sdk.NavigateMsg{PluginID: "chdir"}
	}
}

func (p *Plugin) openWorkspacePicker() tea.Cmd {
	return func() tea.Msg {
		return sdk.NavigateMsg{PluginID: "workspaces"}
	}
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	return p, nil
}

// View renders via the stack.
func (p *Plugin) View(width, height int) string {
	return p.stack.View(width, height)
}
