package context

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Overlay is a modal picker for quick context switching.
type Overlay struct {
	plugin   *Plugin
	selected int
}

// NewOverlay creates a context picker overlay backed by the given plugin.
func NewOverlay(p *Plugin) *Overlay {
	sel := 0
	if p.active >= 0 {
		sel = p.active
	}
	return &Overlay{plugin: p, selected: sel}
}

func (o *Overlay) ID() string { return "context-picker" }

func (o *Overlay) Open() tea.Cmd {
	if o.plugin.status == StatusIdle || o.plugin.status == StatusError {
		o.plugin.status = StatusLoading
		return o.plugin.discover()
	}
	return nil
}

func (o *Overlay) Update(msg tea.Msg) (sdk.Overlay, tea.Cmd) {
	switch msg := msg.(type) {
	case ContextDiscoveredMsg:
		o.plugin.Update(msg)
		return o, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "C":
			return nil, nil
		case "j", "down":
			if o.selected < len(o.plugin.projects)-1 {
				o.selected++
			}
		case "k", "up":
			if o.selected > 0 {
				o.selected--
			}
		case "enter":
			if o.selected >= len(o.plugin.projects) {
				return nil, nil
			}
			o.plugin.active = o.selected
			p := o.plugin.projects[o.selected]
			if o.plugin.session != nil {
				o.plugin.session.Set(sdk.SessionKeyActiveContext, p.Path)
				o.plugin.session.Set(sdk.SessionKeyActiveContextAbs, p.AbsPath)
			}
			return nil, nil
		}
	}
	return o, nil
}

func (o *Overlay) View(width, height int) string {
	switch o.plugin.status {
	case StatusIdle, StatusLoading:
		return sdk.StyleFaintItalic.Render("Discovering projects...")
	case StatusError:
		return sdk.StyleError.Render("Error: " + o.plugin.errMsg)
	}

	if len(o.plugin.projects) == 0 {
		return sdk.StyleFaintItalic.Render("No projects configured.")
	}

	title := sdk.StyleTitle.Render("Switch Context")

	var b strings.Builder
	maxVisible := height - 6
	if maxVisible < 3 {
		maxVisible = 3
	}
	if maxVisible > len(o.plugin.projects) {
		maxVisible = len(o.plugin.projects)
	}

	startIdx := 0
	if o.selected >= maxVisible {
		startIdx = o.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(o.plugin.projects) {
		endIdx = len(o.plugin.projects)
	}

	for i := startIdx; i < endIdx; i++ {
		proj := o.plugin.projects[i]
		indicator := "  "
		name := sdk.StyleFaint.Render(proj.Path)
		if o.plugin.active >= 0 && i == o.plugin.active {
			indicator = sdk.StyleSuccess.Render("* ")
			name = sdk.StyleKey.Render(proj.Path)
		}
		row := fmt.Sprintf("%s%s", indicator, name)
		if i == o.selected {
			row = sdk.StyleSelected.Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d project(s)", len(o.plugin.projects)))
	return title + "\n\n" + b.String() + "\n" + count
}

func (o *Overlay) Hints() []sdk.KeyHint {
	return (sdk.HintSetNavigate | sdk.HintSetSelect | sdk.HintSetCancel).Hints()
}
