package chdir

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

type Plugin struct {
	sdk.PluginBase
	members  []string
	cursor   *ui.Cursor
	stack    *sdk.Stack
	selected bool
}

func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		PluginBase: sdk.NewPluginBase("chdir", "Chdir", "Select working directory from configured members"),
		cursor:     ui.NewCursor(),
	}
	p.Svc = svc
	p.stack = sdk.NewStack()
	return p
}

func (p *Plugin) Ready() bool       { return p.selected }
func (p *Plugin) Stack() *sdk.Stack { return p.stack }

func (p *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// SetMembers configures the list of chdir candidates. Path resolution is the
// App's responsibility — the plugin only emits relative paths.
func (p *Plugin) SetMembers(members []string) {
	p.members = members
	p.cursor.SetCount(len(members))
}

func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.InitBase(deps)
	return nil
}

func (p *Plugin) Activate() tea.Cmd {
	if len(p.members) == 0 {
		p.selected = true
		return nil
	}
	// A lone member needs no choice — including the implicit "." injected when
	// tfui.hcl declares no members. Auto-select it and skip the picker.
	if len(p.members) == 1 {
		return p.selectAt(0)
	}
	p.stack.Clear()
	p.stack.Push(&listFrame{plugin: p})
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler. Once the app's
// immutable Context has a WorkingDir, the chdir picker has done its job.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if ev.Next != nil && ev.Next.WorkingDir != "" {
		p.selected = true
	}
	return nil
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	return p, nil
}

func (p *Plugin) selectMember() tea.Cmd {
	return p.selectAt(p.cursor.Pos())
}

func (p *Plugin) selectAt(idx int) tea.Cmd {
	if idx < 0 || idx >= len(p.members) {
		return nil
	}

	member := p.members[idx]

	p.selected = true
	return func() tea.Msg {
		return sdk.ContextSwitchRequestMsg{Chdir: sdk.Chdir(member), Workspace: sdk.WorkspaceDefault}
	}
}

func (p *Plugin) View(width, height int) string {
	return p.stack.View(width, height)
}
