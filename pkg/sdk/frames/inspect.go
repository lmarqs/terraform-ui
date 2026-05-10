package frames

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// InspectAction defines a context action available in the inspect view.
type InspectAction struct {
	Key     string
	Label   string
	Handler func() tea.Cmd
}

// InspectOpts configures an InspectFrame.
type InspectOpts struct {
	Title    string
	Address  string
	Content  string
	Actions  []InspectAction
	IsPinned func() bool
}

// InspectFrame displays scrollable detail content with context actions.
type InspectFrame struct {
	Title    string
	Address  string
	Viewport *sdk.Viewport
	Actions  []InspectAction
	IsPinned func() bool
}

// NewInspectFrame creates an inspect frame with the given options.
func NewInspectFrame(opts InspectOpts) *InspectFrame {
	vp := sdk.NewViewport(80, 20)
	vp.SetContentString(opts.Content)
	return &InspectFrame{
		Title:    opts.Title,
		Address:  opts.Address,
		Viewport: vp,
		Actions:  opts.Actions,
		IsPinned: opts.IsPinned,
	}
}

func (f *InspectFrame) ID() string { return "inspect" }

func (f *InspectFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "esc":
		return nil, nil
	default:
		if f.Viewport.HandleKey(keyMsg) {
			return f, nil
		}
		for _, action := range f.Actions {
			if keyMsg.String() == action.Key {
				if action.Handler != nil {
					return f, action.Handler()
				}
				return f, nil
			}
		}
		return f, nil
	}
}

func (f *InspectFrame) View(width, height int) string {
	f.Viewport.SetSize(width, height)
	return f.Viewport.Render()
}

func (f *InspectFrame) Hints() []sdk.KeyHint {
	hints := []sdk.KeyHint{
		{Key: "Esc", Description: "back"},
		{Key: "↑↓", Description: "scroll"},
	}
	for _, action := range f.Actions {
		hints = append(hints, sdk.KeyHint{Key: action.Key, Description: action.Label})
	}
	if f.IsPinned != nil && f.IsPinned() {
		hints = append(hints, sdk.KeyHint{Key: "", Description: "[pinned]"})
	}
	return hints
}
