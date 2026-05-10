package plan

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// listFrame is the root frame for the plan plugin.
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
		f.plugin.MoveDown()
	case "k", "up":
		f.plugin.MoveUp()
	case "enter", "i":
		f.plugin.ToggleExpand()
	case " ":
		if change := f.plugin.SelectedChange(); change != nil {
			f.plugin.togglePin(change.Resource.Address)
		}
	case "a":
		if f.plugin.status == StatusDone && f.plugin.summary != nil && len(f.plugin.summary.Changes) > 0 {
			return f, f.plugin.requestApply()
		}
	case "u":
		if f.plugin.status == StatusError && f.plugin.lockInfo != nil {
			return f, f.plugin.requestForceUnlock()
		}
	case "r":
		if f.plugin.status == StatusError || f.plugin.status == StatusDone {
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
	switch f.plugin.status {
	case StatusError:
		if f.plugin.lockInfo != nil {
			return []sdk.KeyHint{{Key: "u", Description: "force-unlock"}, sdk.HintRetry, sdk.HintBack}
		}
		return []sdk.KeyHint{sdk.HintRetry, sdk.HintBack}
	case StatusDone:
		if f.plugin.summary == nil || len(f.plugin.summary.Changes) == 0 {
			return []sdk.KeyHint{sdk.HintRefresh, sdk.HintBack}
		}
		return []sdk.KeyHint{
			sdk.HintNavigate,
			sdk.HintInspect,
			sdk.HintPin,
			{Key: "a", Description: "apply"},
			sdk.HintRefresh,
			sdk.HintBack,
		}
	default:
		return nil
	}
}
