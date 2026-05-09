package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

func NewHomeView() HomeView {
	return HomeView{
		items: []MenuItem{
			{Key: "p", Label: "Plan", Description: "Run terraform plan and review changes"},
			{Key: "r", Label: "Risk Analysis", Description: "Classify changes by risk level (critical/high/medium/low)"},
			{Key: "b", Label: "Blast Radius", Description: "Visualize affected modules and resource dependencies"},
			{Key: "a", Label: "Apply", Description: "Apply changes with live per-resource progress"},
			{Key: "s", Label: "State", Description: "Browse and inspect terraform state resources"},
			{Key: "w", Label: "Workspaces", Description: "List, switch, and manage workspaces"},
			{Key: "m", Label: "Projects", Description: "Select terraform project in monorepo (from tfui.yaml)"},
		},
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

	hint := styles.StyleFaintItalic.Render("Press a key or use ↑↓ + Enter to select an action")

	content := styles.StyleTitle.Render("terraform-ui") + "\n\n" + b.String() + "\n" + hint
	return styles.StylePadded.Render(content)
}
