package plan

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
)

func (e *Plugin) actionTargets() []string {
	if e.HasPins() {
		return e.PinnedAddresses()
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
				return func() tea.Msg { return sdk.TaintRequestMsg{Addresses: targets} }
			},
		},
		{
			Key:   "T",
			Label: "untaint",
			Handler: func() tea.Cmd {
				return func() tea.Msg { return sdk.UntaintRequestMsg{Addresses: targets} }
			},
		},
	}

	return frames.NewActionFrame(title, actions)
}
