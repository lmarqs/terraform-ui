package frames

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Action defines a single item in the action palette.
type Action struct {
	Key      string
	Label    string
	Handler  func() tea.Cmd
	Disabled bool
}

// ActionFrame renders a centered bordered action palette overlay.
type ActionFrame struct {
	title   string
	actions []Action
}

// NewActionFrame creates an action palette frame.
func NewActionFrame(title string, actions []Action) *ActionFrame {
	return &ActionFrame{
		title:   title,
		actions: actions,
	}
}

func (f *ActionFrame) ID() string { return "actions" }

func (f *ActionFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	if keyMsg.String() == "esc" {
		return nil, nil
	}

	for _, a := range f.actions {
		if keyMsg.String() == a.Key {
			if a.Disabled {
				return f, nil
			}
			if a.Handler != nil {
				return nil, a.Handler()
			}
			return nil, nil
		}
	}

	return f, nil
}

func (f *ActionFrame) View(width, height int) string {
	var lines []string
	if f.title != "" {
		lines = append(lines, sdk.StyleTitle.Render(f.title))
		lines = append(lines, "")
	}
	for _, a := range f.actions {
		if a.Disabled {
			lines = append(lines, sdk.StyleFaint.Render(fmt.Sprintf("  %s  %s", a.Key, a.Label)))
		} else {
			key := sdk.StyleKey.Render(fmt.Sprintf("  %s", a.Key))
			lines = append(lines, key+"  "+a.Label)
		}
	}

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sdk.ColorPrimary).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (f *ActionFrame) Hints() []sdk.KeyHint {
	hints := make([]sdk.KeyHint, 0, len(f.actions)+1)
	for _, a := range f.actions {
		if !a.Disabled {
			hints = append(hints, sdk.KeyHint{Key: a.Key, Description: a.Label})
		}
	}
	hints = append(hints, sdk.KeyHint{Key: "Esc", Description: "cancel"})
	return hints
}
