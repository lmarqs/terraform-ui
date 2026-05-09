package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

type MenuItem struct {
	Key         string
	Label       string
	Description string
}

type HomeView struct {
	selected int
	items    []MenuItem
}

// NewHomeView creates a home view with menu items generated from the plugin registry.
func NewHomeView(plugins []plugin.Plugin) HomeView {
	items := make([]MenuItem, 0, len(plugins))
	for _, p := range plugins {
		items = append(items, MenuItem{
			Key:         p.KeyBinding(),
			Label:       p.Name(),
			Description: p.Description(),
		})
	}
	return HomeView{
		items: items,
	}
}

func (v HomeView) Selected() int          { return v.selected }
func (v HomeView) Items() []MenuItem      { return v.items }
func (v HomeView) SelectedItem() MenuItem { return v.items[v.selected] }

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
		key := styles.StyleKey.Width(3).Render(fmt.Sprintf("[%s]", item.Key))
		label := labelStyle.Render(item.Label)
		desc := styles.StyleFaint.Render(item.Description)

		row := fmt.Sprintf("%s %s %s", key, label, desc)
		if i == v.selected {
			row = styles.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	hint := styles.StyleFaintItalic.Render("Press a key or use j/k + Enter to select an action")

	content := styles.StyleTitle.Render("terraform-ui") + "\n\n" + b.String() + "\n" + hint
	return styles.StylePadded.Render(content)
}
