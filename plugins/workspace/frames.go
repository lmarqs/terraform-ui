package workspace

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
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

	switch keyMsg.String() {
	case "esc":
		return f, func() tea.Msg { return sdk.DeactivateMsg{} }
	case "j", "down":
		if f.plugin.status == sdk.StatusDone {
			f.plugin.MoveDown()
		}
	case "k", "up":
		if f.plugin.status == sdk.StatusDone {
			f.plugin.MoveUp()
		}
	case "enter":
		if f.plugin.status != sdk.StatusDone {
			return f, nil
		}
		return f, f.plugin.SwitchToSelected()
	case "s":
		if f.plugin.status != sdk.StatusDone {
			return f, nil
		}
		return f, f.plugin.SelectCurrent()
	case "n":
		if f.plugin.status != sdk.StatusDone {
			return f, nil
		}
		p := f.plugin
		return f, func() tea.Msg {
			return sdk.RequestInputMsg{
				Request: sdk.InputText("New workspace:", "", func(name string) tea.Cmd {
					if name == "" || !isValidWorkspaceName(name) {
						return nil
					}
					return p.startCreate(name)
				}),
			}
		}
	case "d":
		if f.plugin.status != sdk.StatusDone {
			return f, nil
		}
		ws := f.plugin.SelectedWorkspace()
		if ws == "" || ws == f.plugin.current || ws == "default" {
			return f, nil
		}
		p := f.plugin
		confirm := frames.NewConfirmFrame(
			fmt.Sprintf("Delete workspace %q?", ws),
			func() tea.Cmd { return p.deleteWorkspace(ws) },
			nil,
		)
		f.plugin.stack.Push(confirm)
		return f, nil
	case "ctrl+r":
		if f.plugin.status == sdk.StatusDone || f.plugin.status == sdk.StatusError {
			return f, f.plugin.Refresh()
		}
	}
	return f, nil
}

func (f *listFrame) View(width, height int) string {
	return f.plugin.View(width, height)
}

func (f *listFrame) Hints() []sdk.KeyHint {
	switch f.plugin.status {
	case sdk.StatusError:
		return (sdk.HintSetRetry | sdk.HintSetBack).Hints()
	case sdk.StatusDone:
		set := sdk.HintSetSelect | sdk.HintSetNew | sdk.HintSetRefresh | sdk.HintSetBack
		ws := f.plugin.SelectedWorkspace()
		if ws != "" && ws != f.plugin.current && ws != "default" {
			set |= sdk.HintSetDelete
		}
		hints := set.Hints()
		if ws != "" && ws != f.plugin.current {
			hints = append(hints[:1], append([]sdk.KeyHint{{Key: "s", Description: "select"}}, hints[1:]...)...)
		}
		return hints
	default:
		return (sdk.HintSetBack).Hints()
	}
}
