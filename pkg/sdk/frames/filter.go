package frames

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// FilterOpts configures a FilterFrame.
type FilterOpts struct {
	// OnFilter is called on each keystroke with the current query.
	OnFilter func(query string)
	// OnSelect is called when the user presses enter to confirm selection.
	OnSelect func() tea.Cmd
	// OnNavigate is called with +1 (down) or -1 (up) for cursor movement.
	OnNavigate func(dir int)
	// OnPin is called when space is pressed. If nil, space is treated as text input.
	OnPin func() tea.Cmd
}

// FilterFrame provides fzf-style live filtering. It consumes all
// printable key input — keys like "i", "d", "e" that are normally
// keybindings are treated as text input while this frame is active.
type FilterFrame struct {
	Query      string
	onFilter   func(query string)
	onSelect   func() tea.Cmd
	onNavigate func(dir int)
	onPin      func() tea.Cmd
}

// NewFilterFrame creates a filter frame with the given callbacks.
func NewFilterFrame(opts FilterOpts) *FilterFrame {
	return &FilterFrame{
		onFilter:   opts.OnFilter,
		onSelect:   opts.OnSelect,
		onNavigate: opts.OnNavigate,
		onPin:      opts.OnPin,
	}
}

func (f *FilterFrame) ID() string { return "filter" }

func (f *FilterFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "esc":
		return nil, nil
	case "enter":
		if f.onSelect != nil {
			return f, f.onSelect()
		}
		return f, nil
	case "down":
		if f.onNavigate != nil {
			f.onNavigate(1)
		}
		return f, nil
	case "up":
		if f.onNavigate != nil {
			f.onNavigate(-1)
		}
		return f, nil
	case " ":
		if f.onPin != nil {
			return f, f.onPin()
		}
		f.Query += " "
		if f.onFilter != nil {
			f.onFilter(f.Query)
		}
		return f, nil
	case "backspace", "ctrl+h", "delete":
		if len(f.Query) > 0 {
			f.Query = f.Query[:len(f.Query)-1]
			if f.onFilter != nil {
				f.onFilter(f.Query)
			}
		}
		return f, nil
	default:
		key := keyMsg.String()
		if len(key) == 1 && isFilterChar(key[0]) {
			f.Query += key
			if f.onFilter != nil {
				f.onFilter(f.Query)
			}
		}
		return f, nil
	}
}

func isFilterChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.'
}

func (f *FilterFrame) View(width, height int) string {
	return fmt.Sprintf("/ %s█", f.Query)
}

func (f *FilterFrame) Hints() []sdk.KeyHint {
	return []sdk.KeyHint{
		{Key: "Esc", Description: "cancel"},
		{Key: "Enter", Description: "select"},
	}
}
