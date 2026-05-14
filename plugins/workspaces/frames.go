package workspaces

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// listFrame is the root frame for the workspaces plugin.
type listFrame struct {
	plugin *Plugin
}

func (f *listFrame) ID() string { return "list" }

func (f *listFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	if f.plugin.creating {
		switch keyMsg.String() {
		case "enter":
			if f.plugin.newName != "" {
				name := f.plugin.newName
				f.plugin.creating = false
				f.plugin.newName = ""
				return f, f.plugin.createWorkspace(name)
			}
		case "esc":
			f.plugin.creating = false
			f.plugin.newName = ""
		case "backspace", "ctrl+h", "delete":
			if len(f.plugin.newName) > 0 {
				f.plugin.newName = f.plugin.newName[:len(f.plugin.newName)-1]
			}
		default:
			if len(keyMsg.String()) == 1 && keyMsg.String() >= " " {
				f.plugin.newName += keyMsg.String()
			}
		}
		return f, nil
	}

	switch keyMsg.String() {
	case "esc":
		return f, func() tea.Msg { return sdk.DeactivateMsg{} }
	case "j", "down":
		f.plugin.MoveDown()
	case "k", "up":
		f.plugin.MoveUp()
	case "enter":
		return f, f.plugin.SwitchToSelected()
	case "n":
		f.plugin.creating = true
		f.plugin.newName = ""
	case "d":
		return f, f.plugin.DeleteSelected()
	case "ctrl+r":
		return f, f.plugin.Refresh()
	}
	return f, nil
}

func (f *listFrame) View(width, height int) string {
	return f.plugin.View(width, height)
}

func (f *listFrame) Hints() []sdk.KeyHint {
	if f.plugin.creating {
		return (sdk.HintSetConfirm | sdk.HintSetCancel).Hints()
	}
	switch f.plugin.status {
	case sdk.StatusError:
		return (sdk.HintSetRetry | sdk.HintSetBack).Hints()
	case sdk.StatusDone:
		return (sdk.HintSetSelect | sdk.HintSetNew | sdk.HintSetDelete | sdk.HintSetRefresh | sdk.HintSetBack).Hints()
	default:
		return (sdk.HintSetBack).Hints()
	}
}
