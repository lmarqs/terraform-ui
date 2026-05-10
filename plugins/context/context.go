package context

import (
	"io"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
)

// NavigateToMsg signals the app to navigate to a specific plugin by ID.
type NavigateToMsg struct {
	PluginID string
}

// Plugin implements the context dashboard — shows Project, Scope, Workspace.
type Plugin struct {
	svc     sdk.Service
	cfg     config.Config
	log     *slog.Logger
	session *sdk.Session
	stack   *sdk.Stack
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

// Init initializes the plugin with shared context.
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	if ctx.Logger != nil {
		p.log = ctx.Logger
	}
	p.session = ctx.Session
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
				Label:      "Scope",
				Value:      p.scopeValue,
				Selectable: true,
				OnSelect:   func() tea.Cmd { return func() tea.Msg { return NavigateToMsg{PluginID: "scope"} } },
			},
			{
				Label:      "Workspace",
				Value:      p.workspaceValue,
				Selectable: true,
				OnSelect:   func() tea.Cmd { return func() tea.Msg { return NavigateToMsg{PluginID: "workspaces"} } },
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

func (p *Plugin) scopeValue() string {
	if p.session != nil {
		if v, ok := sdk.GetTyped[string](p.session, sdk.SessionKeyActiveScope); ok && v != "" {
			return v
		}
	}
	return "-"
}

func (p *Plugin) workspaceValue() string {
	if p.session != nil {
		if v, ok := sdk.GetTyped[string](p.session, sdk.SessionKeyWorkspace); ok && v != "" {
			return v
		}
	}
	return "default"
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	return p, nil
}

// View renders via the stack.
func (p *Plugin) View(width, height int) string {
	return p.stack.View(width, height)
}
