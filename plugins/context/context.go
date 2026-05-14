package context

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
)

// Plugin implements the context dashboard — shows Project, Chdir, Workspace.
type Plugin struct {
	svc        sdk.Service
	cfg        config.Config
	log        *slog.Logger
	stack      *sdk.Stack
	chdir      string
	workspace  string
	members    []string
	projectDir string
}

// New creates a new context plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		svc: svc,
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
func (p *Plugin) SetMembers(members []string, projectDir string) {
	p.members = members
	p.projectDir = projectDir
}

// Init initializes the plugin with shared context.
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
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
	frame := newPickerFrame("Chdir", p.members, p.chdir, func(selected string) tea.Cmd {
		absPath := filepath.Join(p.projectDir, selected)
		count := len(p.members)
		p.chdir = selected
		return func() tea.Msg {
			return sdk.ChdirChangedEvent{
				RelPath: selected,
				AbsPath: absPath,
				Count:   count,
			}
		}
	})
	p.stack.Push(frame)
	return nil
}

// workspaceListMsg carries the result of workspace listing.
type workspaceListMsg struct {
	workspaces []string
	err        error
}

func (p *Plugin) openWorkspacePicker() tea.Cmd {
	svc := p.svc
	return func() tea.Msg {
		workspaces, err := svc.WorkspaceList(context.Background())
		return workspaceListMsg{workspaces: workspaces, err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case workspaceListMsg:
		if msg.err != nil {
			return p, nil
		}
		frame := newPickerFrame("Workspace", msg.workspaces, p.workspace, func(selected string) tea.Cmd {
			p.workspace = selected
			svc := p.svc
			return func() tea.Msg {
				_ = svc.WorkspaceSelect(context.Background(), selected)
				return sdk.WorkspaceChangedEvent{Name: selected}
			}
		})
		p.stack.Push(frame)
	}
	return p, nil
}

// View renders via the stack.
func (p *Plugin) View(width, height int) string {
	return p.stack.View(width, height)
}
