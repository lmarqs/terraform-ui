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
		case "ctrl+r":
			return f, f.plugin.retry()
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
	if f.plugin.outcome.kind == outcomeLoading {
		return sdk.HintSetBack.Hints()
	}
	if f.plugin.outcome.kind == outcomeInvalidThreshold {
		return sdk.HintSetBack.Hints()
	}
	var set sdk.HintSet
	if f.plugin.navigable() {
		set |= sdk.HintSetInspect | sdk.HintSetWrap
	}
	opts := sdk.HintSetOpts{WrapMode: f.plugin.panel.WrapMode()}
	if f.plugin.retryable() {
		return (set | sdk.HintSetRetry | sdk.HintSetBack).Hints(opts)
	}
	return (set | sdk.HintSetQuit).Hints(opts)
}

func (e *Plugin) inspectSelected() {
	if !e.Ready() || e.selected >= len(e.items) {
		return
	}
	item := e.items[e.selected]
	risk := "unavailable: " + string(item.row.ErrorCode)
	if item.row.Status != sdk.ResourceUnavailable {
		risk = item.row.Level.String()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Action:   %s %s\nAddress:  %s\nRisk:     %s\n\nAttributes:\n", sdk.ActionSymbol(item.change.Action), item.change.Action, item.change.Resource.Address, risk)
	b.WriteString(sdk.RenderAttributeDiffs(item.change.AttributeDiffs))
	for _, reason := range item.row.Reasons {
		fmt.Fprintf(&b, "\n%s\n", reason.Message)
	}
	e.stack.Push(&inspectFrame{InspectFrame: frames.NewInspectFrame(frames.InspectOpts{Title: "Mutation", Address: item.change.Resource.Address, Content: b.String()})})
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
