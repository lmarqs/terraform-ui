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
		if f.plugin.status == sdk.StatusDone && f.plugin.summary != nil && len(f.plugin.summary.Changes) > 0 {
			return f, f.plugin.requestApply()
		}
	case "u":
		if f.plugin.status == sdk.StatusError && f.plugin.lockInfo != nil {
			return f, func() tea.Msg { return sdk.NavigateMsg{PluginID: "forceunlock"} }
		}
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
	switch f.plugin.status {
	case sdk.StatusIdle:
		return (sdk.HintSetConfirm | sdk.HintSetBack).Hints()
	case sdk.StatusLoading:
		return (sdk.HintSetBack).Hints()
	case sdk.StatusError:
		set := sdk.HintSetRetry | sdk.HintSetBack
		if f.plugin.lockInfo != nil {
			set |= sdk.HintSetUnlock
		}
		return set.Hints()
	case sdk.StatusDone:
		if f.plugin.summary == nil || len(f.plugin.summary.Changes) == 0 {
			return (sdk.HintSetRefresh | sdk.HintSetBack).Hints()
		}
		return (sdk.HintSetInspect | sdk.HintSetPin | sdk.HintSetApply | sdk.HintSetRefresh | sdk.HintSetBack).Hints()
	default:
		return (sdk.HintSetBack).Hints()
	}
}
