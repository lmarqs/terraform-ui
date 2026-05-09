package views

import (
	"fmt"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

type WorkspacesView struct {
	workspaces []string
	current    string
	selected   int
}

func NewWorkspacesView() WorkspacesView {
	return WorkspacesView{
		workspaces: []string{"default"},
		current:    "default",
	}
}

func (v WorkspacesView) SetWorkspaces(workspaces []string, current string) WorkspacesView {
	v.workspaces = workspaces
	v.current = current
	v.selected = 0
	for i, ws := range workspaces {
		if ws == current {
			v.selected = i
			break
		}
	}
	return v
}

func (v WorkspacesView) MoveUp() WorkspacesView {
	if v.selected > 0 {
		v.selected--
	}
	return v
}

func (v WorkspacesView) MoveDown() WorkspacesView {
	if v.selected < len(v.workspaces)-1 {
		v.selected++
	}
	return v
}

func (v WorkspacesView) SelectedWorkspace() string {
	if v.selected < len(v.workspaces) {
		return v.workspaces[v.selected]
	}
	return ""
}

func (v WorkspacesView) Render(width, height int) string {
	activeStyle := styles.StyleKey.Copy()

	var b strings.Builder
	for i, ws := range v.workspaces {
		indicator := "  "
		name := styles.StyleFaint.Render(ws)
		if ws == v.current {
			indicator = "● "
			name = activeStyle.Render(ws)
		}

		row := fmt.Sprintf("%s%s", indicator, name)
		if i == v.selected {
			row = styles.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	hint := styles.StyleFaintItalic.Render("Enter to switch  n new  d delete")
	content := styles.StyleTitle.Render("Workspaces") + "\n\n" + b.String() + "\n" + hint
	return styles.StylePadded.Render(content)
}
