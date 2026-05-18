package output

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// listFrame is the root frame for the output plugin.
type listFrame struct {
	plugin *Plugin
}

func (f *listFrame) ID() string { return "list" }

func (f *listFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	if f.plugin.filtering {
		switch keyMsg.String() {
		case "esc":
			f.plugin.filtering = false
		case "/":
			// no-op
		case "down":
			f.plugin.MoveDown()
		case "up":
			f.plugin.MoveUp()
		case "backspace", "ctrl+h", "delete":
			f.plugin.BackspaceFilter()
		default:
			if len(keyMsg.String()) == 1 && keyMsg.String() >= " " {
				f.plugin.AppendFilter(keyMsg.String())
			}
		}
		return f, nil
	}

	switch keyMsg.String() {
	case "esc":
		return f, func() tea.Msg { return sdk.DeactivateMsg{} }
	case "down", "j":
		f.plugin.MoveDown()
	case "up", "k":
		f.plugin.MoveUp()
	case "/":
		f.plugin.filtering = true
		f.plugin.filter = ""
		f.plugin.filtered = f.plugin.outputs
		f.plugin.selected = 0
	case "ctrl+r":
		if f.plugin.status == sdk.StatusError || f.plugin.status == sdk.StatusDone {
			return f, f.plugin.Refresh()
		}
	case "G":
		f.plugin.MoveToEnd()
	case "g":
		f.plugin.MoveToStart()
	}
	return f, nil
}

func (f *listFrame) View(width, height int) string {
	return f.plugin.View(width, height)
}

func (f *listFrame) Hints() []sdk.KeyHint {
	if f.plugin.filtering {
		return (sdk.HintSetCancel).Hints()
	}
	switch f.plugin.status {
	case sdk.StatusError:
		return (sdk.HintSetRetry | sdk.HintSetQuit).Hints()
	case sdk.StatusDone:
		return (sdk.HintSetFilter | sdk.HintSetRefresh | sdk.HintSetQuit).Hints()
	default:
		return (sdk.HintSetQuit).Hints()
	}
}
