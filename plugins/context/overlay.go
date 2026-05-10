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

func (o *Overlay) ID() string { return "scope-picker" }

func (o *Overlay) Open() tea.Cmd {
	if o.plugin.status == StatusIdle || o.plugin.status == StatusError {
		o.plugin.status = StatusLoading
		return o.plugin.discover()
	}
	return nil
}

func (o *Overlay) Update(msg tea.Msg) (sdk.Overlay, tea.Cmd) {
	switch msg := msg.(type) {
	case ScopeDiscoveredMsg:
		o.plugin.Update(msg)
		return o, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "C":
			return nil, nil
		case "j", "down":
			if o.selected < len(o.plugin.scopes)-1 {
				o.selected++
			}
		case "k", "up":
			if o.selected > 0 {
				o.selected--
			}
		case "enter":
			if o.selected >= len(o.plugin.scopes) {
				return nil, nil
			}
			o.plugin.active = o.selected
			p := o.plugin.scopes[o.selected]
			if o.plugin.session != nil {
				o.plugin.session.Set(sdk.SessionKeyActiveScope, p.Path)
				o.plugin.session.Set(sdk.SessionKeyActiveScopeAbs, p.AbsPath)
			}
			return nil, nil
		}
	}
	return o, nil
}

func (o *Overlay) View(width, height int) string {
	switch o.plugin.status {
	case StatusIdle, StatusLoading:
		return sdk.StyleFaintItalic.Render("Discovering scopes...")
	case StatusError:
		return sdk.StyleError.Render("Error: " + o.plugin.errMsg)
	}

	if len(o.plugin.scopes) == 0 {
		return sdk.StyleFaintItalic.Render("No scopes configured.")
	}

	title := sdk.StyleTitle.Render("Switch Scope")

	var b strings.Builder
	maxVisible := height - 6
	if maxVisible < 3 {
		maxVisible = 3
	}
	if maxVisible > len(o.plugin.scopes) {
		maxVisible = len(o.plugin.scopes)
	}

	startIdx := 0
	if o.selected >= maxVisible {
		startIdx = o.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(o.plugin.scopes) {
		endIdx = len(o.plugin.scopes)
	}

	for i := startIdx; i < endIdx; i++ {
		scope := o.plugin.scopes[i]
		indicator := "  "
		name := sdk.StyleFaint.Render(scope.Path)
		if o.plugin.active >= 0 && i == o.plugin.active {
			indicator = sdk.StyleSuccess.Render("* ")
			name = sdk.StyleKey.Render(scope.Path)
		}
		row := fmt.Sprintf("%s%s", indicator, name)
		if i == o.selected {
			row = sdk.StyleSelected.Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d scope(s)", len(o.plugin.scopes)))
	return title + "\n\n" + b.String() + "\n" + count
}

func (o *Overlay) Hints() []sdk.KeyHint {
	return (sdk.HintSetNavigate | sdk.HintSetSelect | sdk.HintSetCancel).Hints()
}
