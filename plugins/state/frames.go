package state

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui/tree"
)

// listFrame is the root frame for the state plugin's resource list.
// It handles navigation, inspect, pin, delete, edit, and entering filter mode.
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
	case "down":
		f.plugin.MoveDown()
	case "up":
		f.plugin.MoveUp()
	case "enter", "i":
		if f.plugin.treeMode {
			node := f.plugin.CursorNode()
			if node != nil && node.Kind == tree.KindBranch {
				f.plugin.tree.Toggle()
				return f, nil
			}
		}
		return f, f.plugin.InspectSelected()
	case "/":
		f.plugin.filtering = true
		f.plugin.filter = ""
		f.plugin.filtered = f.plugin.resources
		f.plugin.rebuildTree()
		f.plugin.stack.Push(&stateFilterFrame{
			plugin: f.plugin,
			inner: frames.NewFilterFrame(frames.FilterOpts{
				OnFilter: func(q string) { f.plugin.SetFilter(q) },
				OnSelect: func() tea.Cmd {
					node := f.plugin.CursorNode()
					if node != nil && node.Kind == tree.KindBranch {
						f.plugin.tree.Toggle()
						return nil
					}
					return f.plugin.InspectSelected()
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
	case "r":
		if f.plugin.status == StatusError || f.plugin.status == StatusDone {
			return f, f.plugin.Refresh()
		}
	case "u":
		if f.plugin.status == StatusError && f.plugin.lockInfo != nil {
			return f, f.plugin.requestForceUnlock()
		}
	case "G":
		f.plugin.MoveToEnd()
	case "g":
		f.plugin.MoveToStart()
	case "ctrl+t":
		f.plugin.treeMode = !f.plugin.treeMode
		f.plugin.SetFilter(f.plugin.filter)
	case "]":
		if f.plugin.treeMode {
			f.plugin.tree.ExpandAll()
		}
	case "[":
		if f.plugin.treeMode {
			f.plugin.tree.CollapseAll()
		}
	case " ":
		node := f.plugin.CursorNode()
		if node != nil {
			return f, f.plugin.togglePin(node.Path)
		}
	case "d":
		r := f.plugin.SelectedResource()
		if r.Address != "" {
			return f, f.plugin.requestDelete(r.Address)
		}
	case "e":
		r := f.plugin.SelectedResource()
		if r.Address != "" {
			return f, f.plugin.requestEdit(r.Address)
		}
	}
	return f, nil
}

func (f *listFrame) View(width, height int) string {
	return f.plugin.View(width, height)
}

func (f *listFrame) Hints() []sdk.KeyHint {
	switch f.plugin.status {
	case StatusLoading:
		return (sdk.HintSetBack).Hints()
	case StatusError:
		set := sdk.HintSetRetry | sdk.HintSetBack
		if f.plugin.lockInfo != nil {
			set |= sdk.HintSetUnlock
		}
		return set.Hints()
	default:
		set := sdk.HintSetNavigate | sdk.HintSetInspect | sdk.HintSetPin | sdk.HintSetFilter | sdk.HintSetTree | sdk.HintSetBack
		if f.plugin.treeMode {
			set |= sdk.HintSetCollapse
		}
		return set.Hints(sdk.HintSetOpts{TreeMode: f.plugin.treeMode})
	}
}

// detailFrame handles key routing for the resource detail/inspect view.
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
		f.plugin.status = StatusDone
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
		f.plugin.panDetailRight()
	case "left":
		f.plugin.panDetailLeft()
	case "w":
		f.plugin.detailWrap = !f.plugin.detailWrap
		f.plugin.detailScroll = 0
		f.plugin.detailHScroll = 0
	case "ctrl+w":
		f.plugin.detailWrap = !f.plugin.detailWrap
		f.plugin.detailScroll = 0
		f.plugin.detailHScroll = 0
	case " ":
		return f, f.plugin.togglePin(f.plugin.detailAddr)
	case "d":
		return f, f.plugin.requestDelete(f.plugin.detailAddr)
	case "e":
		return f, f.plugin.requestEdit(f.plugin.detailAddr)
	}
	return f, nil
}

func (f *detailFrame) View(width, height int) string {
	return f.plugin.renderDetail(width, height)
}

func (f *detailFrame) Hints() []sdk.KeyHint {
	set := sdk.HintSetScroll | sdk.HintSetPan | sdk.HintSetWrap | sdk.HintSetPin | sdk.HintSetDelete | sdk.HintSetEdit | sdk.HintSetCancel
	return set.Hints(sdk.HintSetOpts{
		WrapMode: f.plugin.detailWrap,
		Pinned:   f.plugin.isPinnedAddress(f.plugin.detailAddr),
	})
}

// stateFilterFrame wraps FilterFrame with plugin-specific cleanup on pop.
type stateFilterFrame struct {
	inner  *frames.FilterFrame
	plugin *Plugin
}

func (f *stateFilterFrame) ID() string { return "filter" }

func (f *stateFilterFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
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
		}
	}

	result, cmd := f.inner.Update(msg)
	if result == nil {
		f.plugin.filtering = false
		return nil, cmd
	}
	return f, cmd
}

func (f *stateFilterFrame) View(width, height int) string {
	return f.inner.View(width, height)
}

func (f *stateFilterFrame) Hints() []sdk.KeyHint {
	set := sdk.HintSetNavigate | sdk.HintSetInspect | sdk.HintSetPin | sdk.HintSetCancel
	if f.plugin.treeMode {
		set |= sdk.HintSetCollapse
	}
	return set.Hints(sdk.HintSetOpts{TreeMode: f.plugin.treeMode})
}
