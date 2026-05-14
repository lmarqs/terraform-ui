package context

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// pickerFrame is a simple list picker that pops itself after selection.
type pickerFrame struct {
	title    string
	items    []string
	cursor   *ui.Cursor
	onSelect func(string) tea.Cmd
}

func newPickerFrame(title string, items []string, current string, onSelect func(string) tea.Cmd) *pickerFrame {
	cursor := ui.NewCursor()
	cursor.SetCount(len(items))
	for i, item := range items {
		if item == current {
			for cursor.Pos() < i {
				cursor.MoveDown()
			}
			break
		}
	}
	return &pickerFrame{
		title:    title,
		items:    items,
		cursor:   cursor,
		onSelect: onSelect,
	}
}

func (f *pickerFrame) ID() string { return "picker" }

func (f *pickerFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		f.cursor.MoveDown()
	case "k", "up":
		f.cursor.MoveUp()
	case "g":
		f.cursor.MoveToStart()
	case "G":
		f.cursor.MoveToEnd()
	case "enter":
		if len(f.items) > 0 {
			selected := f.items[f.cursor.Pos()]
			cmd := f.onSelect(selected)
			return nil, cmd
		}
	case "esc", "q":
		return nil, nil
	}
	return f, nil
}

func (f *pickerFrame) View(width, height int) string {
	if len(f.items) == 0 {
		return sdk.StyleFaintItalic.Render("No items available.")
	}

	start, end := f.cursor.VisibleWindow(height)

	var b strings.Builder
	for i := start; i < end; i++ {
		item := f.items[i]
		if i == f.cursor.Pos() {
			b.WriteString(sdk.StyleSelected.Render("▸ " + item))
		} else {
			b.WriteString("  " + item)
		}
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (f *pickerFrame) Hints() []sdk.KeyHint {
	return []sdk.KeyHint{
		{Key: "↑↓", Description: "navigate"},
		{Key: "Enter", Description: "select"},
		{Key: "esc", Description: "back"},
	}
}
