package context

import (
	"io"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
)

// Plugin implements the context dashboard — shows Project, Chdir, Workspace.
type Plugin struct {
	projectDir string
	log        *slog.Logger
	stack      *sdk.Stack
	chdir      string
	workspace  string
	members    []string
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

// SetProjectDir sets the project directory for display.
func (p *Plugin) SetProjectDir(dir string) {
	p.projectDir = dir
}

// SetMembers provides the list of chdir members.
func (p *Plugin) SetMembers(members []string) {
	p.members = members
}

// Init wires the plugin to its shared dependencies. The boot-time workspace
// is read from deps.Context(); subsequent changes arrive via
// ContextChangedEvent.
func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	if deps.Logger != nil {
		p.log = deps.Logger
	}
	if deps.Context != nil {
		if ctx := deps.Context(); ctx != nil {
			p.workspace = ctx.Workspace
		}
	}
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler. The context
// plugin mirrors the active chdir + workspace so its form reflects the
// current state.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if ev.Next == nil {
		return nil
	}
	p.chdir = ev.Next.WorkingDir
	p.workspace = ev.Next.Workspace
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
	if p.projectDir != "" {
		return p.projectDir
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
		return sdk.NavigateMsg{PluginID: "workspace"}
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
