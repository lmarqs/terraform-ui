package risk

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
)

type listFrame struct{ plugin *Plugin }

func (f *listFrame) ID() string { return "list" }
func (f *listFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			f.plugin.inspectSelected()
		case "esc":
			return f, func() tea.Msg { return sdk.DeactivateMsg{} }
		default:
			f.plugin.handleKey(key)
		}
	}
	return f, nil
}
func (f *listFrame) View(width, height int) string { return f.plugin.View(width, height) }
func (f *listFrame) Hints() []sdk.KeyHint {
	if f.plugin.status == sdk.StatusDone && f.plugin.totalItems() > 0 {
		return (sdk.HintSetInspect | sdk.HintSetQuit).Hints()
	}
	return sdk.HintSetQuit.Hints()
}

func (e *Plugin) inspectSelected() {
	if e.status != sdk.StatusDone {
		return
	}
	index := e.selected
	for _, group := range e.groups {
		if index >= len(group.Changes) {
			index -= len(group.Changes)
			continue
		}
		change := group.Changes[index]
		var b strings.Builder
		fmt.Fprintf(&b, "Action:   %s %s\nAddress:  %s\nRisk:     %s\n\nAttributes:\n", sdk.ActionSymbol(change.Action), change.Action, change.Resource.Address, change.Risk.String())
		b.WriteString(sdk.RenderAttributeDiffs(change.AttributeDiffs))
		if e.assessment != nil {
			for _, row := range e.assessment.Resources {
				if row.Address == change.Resource.Address {
					for _, reason := range row.Reasons {
						fmt.Fprintf(&b, "\n%s\n", reason.Message)
					}
				}
			}
		}
		e.stack.Push(&inspectFrame{InspectFrame: frames.NewInspectFrame(frames.InspectOpts{Title: "Mutation", Address: change.Resource.Address, Content: b.String()})})
		return
	}
}

type inspectFrame struct {
	*frames.InspectFrame
	wrap bool
}

func (f *inspectFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+w" {
		f.wrap = !f.wrap
	}
	next, cmd := f.InspectFrame.Update(msg)
	if next == nil {
		return nil, cmd
	}
	return f, cmd
}
func (f *inspectFrame) Hints() []sdk.KeyHint {
	return (sdk.HintSetWrap | sdk.HintSetBack).Hints(sdk.HintSetOpts{WrapMode: f.wrap})
}
