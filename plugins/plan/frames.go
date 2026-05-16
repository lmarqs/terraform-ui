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
	case "ctrl+r":
		if f.plugin.status == sdk.StatusError || f.plugin.status == sdk.StatusDone {
			return f, f.plugin.Refresh()
		}
	case "ctrl+t":
		f.plugin.treeMode = !f.plugin.treeMode
		f.plugin.SetFilter(f.plugin.filter)
	case "ctrl+w":
		f.plugin.listWrap = !f.plugin.listWrap
		f.plugin.listHScroll = 0
	case "ctrl+p":
		f.plugin.pinnedOnly = !f.plugin.pinnedOnly
		f.plugin.SetFilter(f.plugin.filter)
	case "ctrl+u":
		f.plugin.clearAllPins()
	case "right":
		if !f.plugin.listWrap {
			f.plugin.panListRight()
		}
	case "left":
		if !f.plugin.listWrap {
			f.plugin.panListLeft()
		}
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
		set := sdk.HintSetInspect | sdk.HintSetPin | sdk.HintSetFilter | sdk.HintSetTree | sdk.HintSetApply | sdk.HintSetEdit | sdk.HintSetTaint | sdk.HintSetUntaint | sdk.HintSetRefresh | sdk.HintSetBack
		if f.plugin.treeMode {
			set |= sdk.HintSetCollapse | sdk.HintSetExpand
		}
		if f.plugin.PinnedCount() > 0 {
			set |= sdk.HintSetClearPins
		}
		return set.Hints(sdk.HintSetOpts{TreeMode: f.plugin.treeMode, WrapMode: f.plugin.listWrap, PinnedFilter: f.plugin.pinnedOnly})
	default:
		return (sdk.HintSetBack).Hints()
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
		f.plugin.detailHScroll = 0
		return nil, nil
	case "down":
		f.plugin.detailScroll++
	case "up":
		if f.plugin.detailScroll > 0 {
			f.plugin.detailScroll--
		}
	case "right":
		if !f.plugin.detailWrap {
			f.plugin.panDetailRight()
		}
	case "left":
		if !f.plugin.detailWrap {
			f.plugin.panDetailLeft()
		}
	case "ctrl+w":
		f.plugin.detailWrap = !f.plugin.detailWrap
		f.plugin.detailScroll = 0
		f.plugin.detailHScroll = 0
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
	set := sdk.HintSetWrap | sdk.HintSetPin | sdk.HintSetEdit | sdk.HintSetCancel
	return set.Hints(sdk.HintSetOpts{
		WrapMode: f.plugin.detailWrap,
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
		case "ctrl+w":
			f.plugin.listWrap = !f.plugin.listWrap
			f.plugin.listHScroll = 0
			return f, nil
		case "ctrl+p":
			f.plugin.pinnedOnly = !f.plugin.pinnedOnly
			f.plugin.SetFilter(f.plugin.filter)
			return f, nil
		case "right":
			if !f.plugin.listWrap {
				f.plugin.panListRight()
			}
			return f, nil
		case "left":
			if !f.plugin.listWrap {
				f.plugin.panListLeft()
			}
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
	set := sdk.HintSetInspect | sdk.HintSetPin | sdk.HintSetCancel
	if f.plugin.treeMode {
		set |= sdk.HintSetCollapse | sdk.HintSetExpand
	}
	return set.Hints(sdk.HintSetOpts{TreeMode: f.plugin.treeMode})
}
