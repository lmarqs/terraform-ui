package frames

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
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
	Actions  []InspectAction
	IsPinned func() bool

	panel   *ui.ContentPanel
	lines   []string
	scrollY int
}

// NewInspectFrame creates an inspect frame with the given options.
func NewInspectFrame(opts InspectOpts) *InspectFrame {
	return &InspectFrame{
		Title:    opts.Title,
		Address:  opts.Address,
		Actions:  opts.Actions,
		IsPinned: opts.IsPinned,
		panel:    ui.NewContentPanel(),
		lines:    strings.Split(opts.Content, "\n"),
	}
}

func (f *InspectFrame) ID() string { return "inspect" }

// ScrollY returns the current vertical scroll offset.
func (f *InspectFrame) ScrollY() int { return f.scrollY }

func (f *InspectFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "esc":
		return nil, nil
	case "up":
		if f.scrollY > 0 {
			f.scrollY--
		}
		return f, nil
	case "down":
		f.scrollY++
		return f, nil
	case "g":
		f.scrollY = 0
		return f, nil
	case "G":
		f.scrollY = len(f.lines)
		return f, nil
	default:
		if f.panel.HandleKey(keyMsg) {
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
	if height <= 0 {
		height = 20
	}

	maxScroll := len(f.lines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if f.scrollY > maxScroll {
		f.scrollY = maxScroll
	}

	endIdx := f.scrollY + height
	if endIdx > len(f.lines) {
		endIdx = len(f.lines)
	}

	return f.panel.Render(ui.RenderParams{
		Rows:         f.lines[f.scrollY:endIdx],
		Width:        width,
		Height:       height,
		TotalItems:   len(f.lines),
		Cursor:       -1,
		ScrollOffset: f.scrollY,
	})
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
