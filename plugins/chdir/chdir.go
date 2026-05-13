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
	selected   bool
}

func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc:    svc,
		cursor: ui.NewCursor(),
	}
}

func (p *Plugin) ID() string          { return "chdir" }
func (p *Plugin) Name() string        { return "Chdir" }
func (p *Plugin) Description() string { return "Select working directory from configured members" }
func (p *Plugin) Ready() bool         { return p.selected }

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
	}
	return nil
}

func (p *Plugin) Hints() []sdk.KeyHint {
	return []sdk.KeyHint{
		{Key: "enter", Description: "select"},
		sdk.HintBack,
	}
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.cursor.MoveUp()
		case "down", "j":
			p.cursor.MoveDown()
		case "enter":
			return p, p.selectMember()
		}
	}
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
	if len(p.members) == 0 {
		return sdk.StyleFaintItalic.Render("No chdir members configured.")
	}

	lines := make([]string, 0, len(p.members))
	start, end := p.cursor.VisibleWindow(height)

	for i := start; i < end; i++ {
		member := p.members[i]
		if i == p.cursor.Pos() {
			lines = append(lines, sdk.StyleSelected.Render("▸ "+member))
		} else {
			lines = append(lines, "  "+member)
		}
	}

	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
