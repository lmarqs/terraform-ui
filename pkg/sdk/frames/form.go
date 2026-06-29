package frames

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// FormField represents a single field in a form.
type FormField struct {
	// Label is the field name displayed on the left.
	Label string
	// Value returns the current display value for the field.
	Value func() string
	// Selectable indicates whether the user can press Enter to act on this field.
	Selectable bool
	// IsAction renders the field as a distinct submit button rather than a data field.
	IsAction bool
	// Toggle marks a boolean field that can be flipped with Space (in addition
	// to Enter). Used for checkbox-style options.
	Toggle bool
	// OnSelect is called when Enter is pressed on a selectable field.
	OnSelect func() tea.Cmd
}

// FormOpts configures a FormFrame.
type FormOpts struct {
	Fields []FormField
}

// FormFrame is a reusable frame that renders labeled fields with cursor navigation.
// Users navigate with j/k and press Enter on selectable fields to trigger actions.
type FormFrame struct {
	fields []FormField
	cursor int
}

// NewFormFrame creates a form frame with the given options.
func NewFormFrame(opts FormOpts) *FormFrame {
	f := &FormFrame{
		fields: opts.Fields,
		cursor: -1,
	}
	// Start cursor at first selectable field
	for i, field := range f.fields {
		if field.Selectable {
			f.cursor = i
			break
		}
	}
	return f
}

func (f *FormFrame) ID() string { return "form" }

func (f *FormFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		f.moveDown()
	case "k", "up":
		f.moveUp()
	case "enter":
		if field, ok := f.focused(); ok && field.Selectable && field.OnSelect != nil {
			return f, field.OnSelect()
		}
	case " ":
		// Space toggles checkbox-style fields; other fields ignore it.
		if field, ok := f.focused(); ok && field.Toggle && field.OnSelect != nil {
			return f, field.OnSelect()
		}
	case "esc":
		return nil, nil
	}
	return f, nil
}

// focused returns the field under the cursor, if any.
func (f *FormFrame) focused() (FormField, bool) {
	if f.cursor >= 0 && f.cursor < len(f.fields) {
		return f.fields[f.cursor], true
	}
	return FormField{}, false
}

func (f *FormFrame) moveDown() {
	for i := f.cursor + 1; i < len(f.fields); i++ {
		if f.fields[i].Selectable {
			f.cursor = i
			return
		}
	}
}

func (f *FormFrame) moveUp() {
	for i := f.cursor - 1; i >= 0; i-- {
		if f.fields[i].Selectable {
			f.cursor = i
			return
		}
	}
}

func (f *FormFrame) View(width, height int) string {
	var b strings.Builder

	for i, field := range f.fields {
		if field.IsAction && i > 0 && !f.fields[i-1].IsAction {
			b.WriteByte('\n')
		}
		selected := i == f.cursor
		b.WriteString(f.renderField(field, selected))
		b.WriteByte('\n')
	}

	return b.String()
}

func (f *FormFrame) renderField(field FormField, selected bool) string {
	if field.IsAction {
		return f.renderAction(field, selected)
	}

	cursor := "  "
	if selected {
		cursor = sdk.StyleKey.Render("▸ ")
	}

	label := sdk.StyleFaint.Render(fmt.Sprintf("%-16s", field.Label))

	value := field.Value()
	if selected {
		value = sdk.StyleKey.Render(value)
	}

	suffix := ""
	if field.Selectable {
		suffix = "  " + sdk.StyleFaint.Render("▸")
	}

	return cursor + label + value + suffix
}

func (f *FormFrame) renderAction(field FormField, selected bool) string {
	label := field.Value()
	if selected {
		return sdk.StyleKey.Render("▸ " + label)
	}
	return "  " + sdk.StyleFaint.Render(label)
}

func (f *FormFrame) Hints() []sdk.KeyHint {
	hints := []sdk.KeyHint{{Key: "↑↓", Description: "navigate"}}
	if field, ok := f.focused(); ok && field.Toggle {
		hints = append(hints, sdk.HintToggle)
	} else {
		hints = append(hints, sdk.HintSelect)
	}
	return append(hints, sdk.HintCancel)
}
