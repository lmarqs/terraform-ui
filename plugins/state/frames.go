package state

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
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
		return f, f.plugin.InspectSelected()
	case "/":
		f.plugin.filtering = true
		f.plugin.filter = ""
		f.plugin.filtered = f.plugin.resources
		f.plugin.sortPinnedFirst()
		f.plugin.stack.Push(&stateFilterFrame{
			plugin: f.plugin,
			inner: frames.NewFilterFrame(frames.FilterOpts{
				OnFilter:   func(q string) { f.plugin.SetFilter(q) },
				OnSelect:   func() tea.Cmd { return f.plugin.InspectSelected() },
				OnNavigate: func(dir int) { f.plugin.navigate(dir) },
				OnPin: func() tea.Cmd {
					r := f.plugin.SelectedResource()
					if r.Address != "" {
						return f.plugin.togglePin(r.Address)
					}
					return nil
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
	case "w":
		f.plugin.detailWrap = !f.plugin.detailWrap
		f.plugin.listHScroll = 0
	case "ctrl+w":
		f.plugin.detailWrap = !f.plugin.detailWrap
		f.plugin.detailScroll = 0
		f.plugin.detailHScroll = 0
		f.plugin.listHScroll = 0
	case "right":
		f.plugin.panRight()
	case "left":
		f.plugin.panLeft()
	case "]":
		if f.plugin.depth < f.plugin.maxDepth() {
			f.plugin.depth++
			f.plugin.computeDisplayItems()
			f.plugin.selected = 0
			f.plugin.listHScroll = 0
		}
	case "[":
		if f.plugin.depth > 0 {
			f.plugin.depth--
			f.plugin.computeDisplayItems()
			f.plugin.selected = 0
			f.plugin.listHScroll = 0
		}
	case " ":
		item := f.plugin.SelectedItem()
		if item.IsGroup {
			return f, f.plugin.togglePin(item.GroupPath)
		}
		r := f.plugin.SelectedResource()
		if r.Address != "" {
			return f, f.plugin.togglePin(r.Address)
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
	return []sdk.KeyHint{
		sdk.HintNavigate,
		sdk.HintInspect,
		sdk.HintPin,
		sdk.HintDelete,
		sdk.HintEdit,
		sdk.HintFilter,
		{Key: "[/]", Description: fmt.Sprintf("depth(%d)", f.plugin.depth)},
		{Key: "^w", Description: fmt.Sprintf("wrap(%s)", wrapLabel(f.plugin.detailWrap))},
		sdk.HintBack,
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
	hints := []sdk.KeyHint{
		sdk.HintCancel,
		sdk.HintScroll,
		sdk.HintPan,
		{Key: "^w", Description: fmt.Sprintf("wrap(%s)", wrapLabel(f.plugin.detailWrap))},
		sdk.HintPin,
		sdk.HintDelete,
		sdk.HintEdit,
	}
	if f.plugin.isPinnedAddress(f.plugin.detailAddr) {
		hints = append(hints, sdk.KeyHint{Description: "[pinned]"})
	}
	return hints
}

// stateFilterFrame wraps FilterFrame with plugin-specific cleanup on pop.
type stateFilterFrame struct {
	inner  *frames.FilterFrame
	plugin *Plugin
}

func (f *stateFilterFrame) ID() string { return "filter" }

func (f *stateFilterFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
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
	return []sdk.KeyHint{
		sdk.HintCancel,
		sdk.HintInspect,
		sdk.HintNavigate,
		sdk.HintPin,
	}
}

func wrapLabel(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}
