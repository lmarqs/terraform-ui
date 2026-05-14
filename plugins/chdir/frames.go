package chdir

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type listFrame struct {
	plugin *Plugin
}

func (f *listFrame) ID() string { return "list" }

func (f *listFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "esc":
		return nil, nil
	case "j", "down":
		f.plugin.cursor.MoveDown()
	case "k", "up":
		f.plugin.cursor.MoveUp()
	case "enter":
		return f, f.plugin.selectMember()
	}
	return f, nil
}

func (f *listFrame) View(width, height int) string {
	p := f.plugin
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

func (f *listFrame) Hints() []sdk.KeyHint {
	return []sdk.KeyHint{
		{Key: "enter", Description: "select"},
		{Key: "esc", Description: "back"},
	}
}
