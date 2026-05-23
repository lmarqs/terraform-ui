package plan

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui/tree"
	"github.com/lmarqs/terraform-ui/plugins/taint"
	"github.com/lmarqs/terraform-ui/plugins/untaint"
)

// listFrame is the root frame for the plan plugin's change list.
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
		if f.plugin.treeMode {
			node := f.plugin.CursorNode()
			if node != nil && node.Kind == tree.KindBranch {
				f.plugin.tree.Toggle()
				return f, nil
			}
		}
		return f, f.plugin.inspectSelected()
	case "/":
		f.plugin.filtering = true
		f.plugin.filter = ""
		f.plugin.filtered = f.plugin.sourceChanges()
		f.plugin.rebuildTree()
		f.plugin.stack.Push(&planFilterFrame{
			plugin: f.plugin,
			inner: frames.NewFilterFrame(frames.FilterOpts{
				OnFilter: func(q string) { f.plugin.SetFilter(q) },
				OnSelect: func() tea.Cmd {
					node := f.plugin.CursorNode()
					if node != nil && node.Kind == tree.KindBranch {
						f.plugin.tree.Toggle()
						return nil
					}
					return f.plugin.inspectSelected()
				},
				OnNavigate: func(dir int) { f.plugin.navigate(dir) },
				OnPin: func() tea.Cmd {
					node := f.plugin.CursorNode()
					if node == nil {
						return nil
					}
					return f.plugin.togglePin(node.Path)
				},
				OnToggle: func() {
					f.plugin.treeMode = !f.plugin.treeMode
					f.plugin.SetFilter(f.plugin.filter)
				},
			}),
		})
		return f, nil
	case " ":
		node := f.plugin.CursorNode()
		if node != nil {
			return f, f.plugin.togglePin(node.Path)
		}
	case "a":
		if f.plugin.status == sdk.StatusDone && f.plugin.summary != nil && len(f.plugin.summary.Changes) > 0 {
			return f, f.plugin.requestApply()
		}
	case "A":
		if f.plugin.status == sdk.StatusDone && f.plugin.summary != nil && len(f.plugin.summary.Changes) > 0 {
			return f, f.plugin.requestAutoApply()
		}
	case "t":
		if f.plugin.status == sdk.StatusDone {
			change := f.plugin.SelectedChange()
			if change != nil {
				return f, func() tea.Msg { return taint.TaintRequestMsg{Addresses: []string{change.Resource.Address}} }
			}
		}
	case "T":
		if f.plugin.status == sdk.StatusDone {
			change := f.plugin.SelectedChange()
			if change != nil {
				return f, func() tea.Msg { return untaint.UntaintRequestMsg{Addresses: []string{change.Resource.Address}} }
			}
		}
	case "e":
		if f.plugin.status == sdk.StatusDone {
			change := f.plugin.SelectedChange()
			if change != nil {
				return f, func() tea.Msg { return PlanEditMsg{Address: change.Resource.Address} }
			}
		}
	case "u":
		if f.plugin.status == sdk.StatusError && f.plugin.lockInfo != nil {
			return f, func() tea.Msg { return sdk.NavigateMsg{PluginID: "forceunlock"} }
		}
	case "l":
		if f.plugin.lastStream != nil {
			f.plugin.stack.Push(f.plugin.lastStream)
		}
	case "ctrl+r":
		if f.plugin.status == sdk.StatusError || f.plugin.status == sdk.StatusDone {
			return f, f.plugin.Refresh()
		}
	case "ctrl+t":
		f.plugin.treeMode = !f.plugin.treeMode
		f.plugin.SetFilter(f.plugin.filter)
	case "ctrl+w", "right", "left":
		f.plugin.listPanel.HandleKey(keyMsg)
	case "ctrl+p":
		f.plugin.pinnedOnly = !f.plugin.pinnedOnly
		f.plugin.SetFilter(f.plugin.filter)
	case "!":
		if f.plugin.PinnedCount() > 0 {
			f.plugin.stack.Push(f.plugin.buildActionFrame(true))
		}
	case "ctrl+u":
		return f, f.plugin.clearAllPins()
	case "]":
		if f.plugin.treeMode {
			f.plugin.tree.ExpandAll()
		}
	case "[":
		if f.plugin.treeMode {
			f.plugin.tree.CollapseAll()
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
		return (sdk.HintSetConfirm | sdk.HintSetQuit).Hints()
	case sdk.StatusLoading:
		return (sdk.HintSetQuit).Hints()
	case sdk.StatusError:
		return (sdk.HintSetRetry | sdk.HintSetQuit).Hints()
	case sdk.StatusDone:
		if f.plugin.summary == nil || len(f.plugin.summary.Changes) == 0 {
			hints := (sdk.HintSetRefresh | sdk.HintSetQuit).Hints()
			if f.plugin.lastStream != nil {
				hints = append(hints, sdk.KeyHint{Key: "l", Description: "log"})
			}
			return hints
		}
		set := sdk.HintSetInspect | sdk.HintSetPin | sdk.HintSetFilter | sdk.HintSetTree | sdk.HintSetRefresh | sdk.HintSetQuit
		if f.plugin.treeMode {
			set |= sdk.HintSetCollapse | sdk.HintSetExpand
		}
		if f.plugin.PinnedCount() > 0 {
			set |= sdk.HintSetClearPins
		}
		hints := set.Hints(sdk.HintSetOpts{TreeMode: f.plugin.treeMode, WrapMode: f.plugin.listPanel.WrapMode(), PinnedFilter: f.plugin.pinnedOnly})
		if f.plugin.lastStream != nil {
			hints = append(hints, sdk.KeyHint{Key: "l", Description: "log"})
		}
		return hints
	default:
		return (sdk.HintSetQuit).Hints()
	}
}

// detailFrame handles key routing for the plan change detail/inspect view.
type detailFrame struct {
	plugin *Plugin
}

func (f *detailFrame) ID() string { return "inspect" }

func (f *detailFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "esc":
		f.plugin.detail = ""
		f.plugin.detailAddr = ""
		f.plugin.detailScroll = 0
		f.plugin.detailPanel.ResetScroll()
		return nil, nil
	case "down":
		f.plugin.detailScroll++
	case "up":
		if f.plugin.detailScroll > 0 {
			f.plugin.detailScroll--
		}
	case "right", "left", "ctrl+w":
		f.plugin.detailPanel.HandleKey(keyMsg)
		if keyMsg.String() == "ctrl+w" {
			f.plugin.detailScroll = 0
		}
	case " ":
		return f, f.plugin.togglePin(f.plugin.detailAddr)
	case "e":
		return f, func() tea.Msg { return PlanEditMsg{Address: f.plugin.detailAddr} }
	}
	return f, nil
}

func (f *detailFrame) View(width, height int) string {
	return f.plugin.renderDetail(width, height)
}

func (f *detailFrame) Hints() []sdk.KeyHint {
	set := sdk.HintSetWrap | sdk.HintSetPin | sdk.HintSetBack
	return set.Hints(sdk.HintSetOpts{
		WrapMode: f.plugin.detailPanel.WrapMode(),
		Pinned:   f.plugin.isPinnedAddress(f.plugin.detailAddr),
	})
}

// planFilterFrame wraps FilterFrame with plugin-specific cleanup on pop.
type planFilterFrame struct {
	inner  *frames.FilterFrame
	plugin *Plugin
}

func (f *planFilterFrame) ID() string { return "filter" }

func (f *planFilterFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "]":
			if f.plugin.treeMode {
				f.plugin.tree.ExpandAll()
			}
			return f, nil
		case "[":
			if f.plugin.treeMode {
				f.plugin.tree.CollapseAll()
			}
			return f, nil
		case "ctrl+w", "right", "left":
			f.plugin.listPanel.HandleKey(keyMsg)
			return f, nil
		case "ctrl+p":
			f.plugin.pinnedOnly = !f.plugin.pinnedOnly
			f.plugin.SetFilter(f.plugin.filter)
			return f, nil
		}
	}

	result, cmd := f.inner.Update(msg)
	if result == nil {
		f.plugin.filtering = false
		return nil, cmd
	}
	return f, cmd
}

func (f *planFilterFrame) View(width, height int) string {
	return f.inner.View(width, height)
}

func (f *planFilterFrame) Hints() []sdk.KeyHint {
	set := sdk.HintSetInspect | sdk.HintSetPin | sdk.HintSetBack
	if f.plugin.treeMode {
		set |= sdk.HintSetCollapse | sdk.HintSetExpand
	}
	return set.Hints(sdk.HintSetOpts{TreeMode: f.plugin.treeMode})
}
