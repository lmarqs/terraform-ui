package plan

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/plugins/taint"
	"github.com/lmarqs/terraform-ui/plugins/untaint"
)

func (e *Plugin) actionTargets() []string {
	if pinned := e.PinnedAddresses(); len(pinned) > 0 {
		return pinned
	}
	change := e.SelectedChange()
	if change != nil {
		return []string{change.Resource.Address}
	}
	return nil
}

func (e *Plugin) buildActionFrame(batch bool) *frames.ActionFrame {
	pinCount := e.PinnedCount()
	multiTarget := batch && pinCount > 1

	title := ""
	if multiTarget {
		title = fmt.Sprintf("%d pinned resources", pinCount)
	} else {
		change := e.SelectedChange()
		if change != nil {
			title = change.Resource.Address
		}
	}

	targets := e.actionTargets()

	actions := []frames.Action{
		{
			Key:   "a",
			Label: "apply",
			Handler: func() tea.Cmd {
				return e.requestApply()
			},
		},
		{
			Key:   "A",
			Label: "auto-apply",
			Handler: func() tea.Cmd {
				return e.requestAutoApply()
			},
		},
		{
			Key:   "t",
			Label: "taint",
			Handler: func() tea.Cmd {
				return func() tea.Msg { return taint.TaintRequestMsg{Addresses: targets} }
			},
		},
		{
			Key:   "T",
			Label: "untaint",
			Handler: func() tea.Cmd {
				return func() tea.Msg { return untaint.UntaintRequestMsg{Addresses: targets} }
			},
		},
	}

	return frames.NewActionFrame(title, actions)
}
