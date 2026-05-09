package views

import (
	"fmt"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

// ModulesView displays discovered terraform projects in a monorepo.
type ModulesView struct {
	modules  []string
	selected int
	active   int
}

func NewModulesView() ModulesView {
	return ModulesView{}
}

func (v ModulesView) SetModules(modules []string, activeIdx int) ModulesView {
	v.modules = modules
	v.active = activeIdx
	v.selected = activeIdx
	return v
}

func (v ModulesView) MoveUp() ModulesView {
	if v.selected > 0 {
		v.selected--
	}
	return v
}

func (v ModulesView) MoveDown() ModulesView {
	if v.selected < len(v.modules)-1 {
		v.selected++
	}
	return v
}

func (v ModulesView) SelectedModule() string {
	if v.selected < len(v.modules) {
		return v.modules[v.selected]
	}
	return ""
}

func (v ModulesView) Render(width, height int) string {
	title := styles.StyleTitle.Render("Projects")

	if len(v.modules) == 0 {
		placeholder := styles.StyleFaintItalic.Render(
			"No projects configured. Add paths to tfui.yaml:\n\n" +
				"  projects:\n" +
				"    paths:\n" +
				"      - \"modules/*\"\n" +
				"      - \"envs/**\"",
		)
		return styles.StylePadded.Render(title + "\n\n" + placeholder)
	}

	var b strings.Builder
	for i, mod := range v.modules {
		indicator := "  "
		name := styles.StyleFaint.Render(mod)
		if i == v.active {
			indicator = "● "
			name = styles.StyleKey.Render(mod)
		}

		row := fmt.Sprintf("%s%s", indicator, name)
		if i == v.selected {
			row = styles.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	hint := styles.StyleFaintItalic.Render("Enter to select project")
	content := title + "\n\n" + b.String() + "\n" + hint
	return styles.StylePadded.Render(content)
}
