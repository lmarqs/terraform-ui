package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type HomeView struct {
	selected int
	items    []plugin.MenuItem
}

// NewHomeView creates a home view with menu items from the registry's MenuItems().
func NewHomeView(items []plugin.MenuItem) HomeView {
	return HomeView{
		items: items,
	}
}

func (v HomeView) Selected() int                 { return v.selected }
func (v HomeView) Items() []plugin.MenuItem      { return v.items }
func (v HomeView) SelectedItem() plugin.MenuItem { return v.items[v.selected] }

func (v HomeView) MoveUp() HomeView {
	if v.selected > 0 {
		v.selected--
	}
	return v
}

func (v HomeView) MoveDown() HomeView {
	if v.selected < len(v.items)-1 {
		v.selected++
	}
	return v
}

func (v HomeView) Render(width, height int) string {
	labelStyle := lipgloss.NewStyle().Bold(true).Width(16)

	var b strings.Builder
	for i, item := range v.items {
		key := sdk.StyleKey.Width(3).Render(fmt.Sprintf("[%s]", item.Key))
		label := labelStyle.Render(item.Name)
		desc := sdk.StyleFaint.Render(item.Description)

		row := fmt.Sprintf("%s %s %s", key, label, desc)
		if i == v.selected {
			row = sdk.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	hint := sdk.StyleFaintItalic.Render("Press a key or use j/k + Enter to select an action")

	return b.String() + "\n" + hint
}
