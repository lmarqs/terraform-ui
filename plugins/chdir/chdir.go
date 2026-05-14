package chdir

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

type Plugin struct {
	svc        sdk.Service
	members    []string
	projectDir string
	cursor     *ui.Cursor
	stack      *sdk.Stack
	selected   bool
}

func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		svc:    svc,
		cursor: ui.NewCursor(),
	}
	p.stack = sdk.NewStack()
	return p
}

func (p *Plugin) ID() string          { return "chdir" }
func (p *Plugin) Name() string        { return "Chdir" }
func (p *Plugin) Description() string { return "Select working directory from configured members" }
func (p *Plugin) Ready() bool         { return p.selected }
func (p *Plugin) Stack() *sdk.Stack   { return p.stack }

func (p *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

func (p *Plugin) SetMembers(members []string, projectDir string) {
	p.members = members
	p.projectDir = projectDir
	p.cursor.SetCount(len(members))
}

func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	return nil
}

func (p *Plugin) Activate() tea.Cmd {
	if len(p.members) == 0 {
		p.selected = true
		return nil
	}
	p.stack.Clear()
	p.stack.Push(&listFrame{plugin: p})
	return nil
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	return p, nil
}

func (p *Plugin) selectMember() tea.Cmd {
	if len(p.members) == 0 {
		return nil
	}
	idx := p.cursor.Pos()
	if idx >= len(p.members) {
		return nil
	}

	member := p.members[idx]
	absPath := filepath.Join(p.projectDir, member)
	count := len(p.members)

	p.selected = true
	return func() tea.Msg {
		return sdk.ChdirChangedEvent{
			RelPath: member,
			AbsPath: absPath,
			Count:   count,
		}
	}
}

func (p *Plugin) View(width, height int) string {
	return p.stack.View(width, height)
}
