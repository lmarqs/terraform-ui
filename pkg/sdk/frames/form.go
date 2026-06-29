package frames

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// FormField represents a single field in a form.
//
// Enter and Space are independent keys. Space operates the focused field
// (toggle a checkbox, open a value editor) via OnSpace. Enter confirms — it
// runs the field's OnSelect if set, otherwise the form-level OnSubmit. A field
// binds whichever it responds to.
type FormField struct {
	// Label is the field name displayed on the left.
	Label string
	// Value returns the current display value for the field.
	Value func() string
	// Selectable indicates whether the cursor can land on this field.
	Selectable bool
	// IsAction renders the field as a distinct submit button rather than a data field.
	IsAction bool
	// OnSelect is called when Enter is pressed on this field (per-field activate,
	// e.g. opening a picker). When nil, Enter falls back to FormOpts.OnSubmit.
	OnSelect func() tea.Cmd
	// OnSpace is called when Space is pressed on this field (operate the field).
	OnSpace func() tea.Cmd
	// SpaceHint labels the Space action in the hint bar (e.g. "toggle", "edit").
	SpaceHint string
}

// FormOpts configures a FormFrame.
type FormOpts struct {
	Fields []FormField
	// OnSubmit confirms the whole form. Enter triggers it from any field whose
	// OnSelect is nil, so a settings form runs without navigating to a button.
	OnSubmit func() tea.Cmd
	// SubmitHint labels the Enter action in the hint bar when OnSubmit is set
	// (e.g. "run"). Defaults to "confirm".
	SubmitHint string
}

// FormFrame is a reusable frame that renders labeled fields with cursor navigation.
// Navigate with j/k; Space operates the focused field, Enter confirms the form.
type FormFrame struct {
	fields     []FormField
	cursor     int
	onSubmit   func() tea.Cmd
	submitHint string
}

// NewFormFrame creates a form frame with the given options.
func NewFormFrame(opts FormOpts) *FormFrame {
	f := &FormFrame{
		fields:     opts.Fields,
		cursor:     -1,
		onSubmit:   opts.OnSubmit,
		submitHint: opts.SubmitHint,
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
		// Enter confirms: the field's own action if it has one, else the form.
		if field, ok := f.focused(); ok && field.OnSelect != nil {
			return f, field.OnSelect()
		}
		if f.onSubmit != nil {
			return f, f.onSubmit()
		}
	case " ":
		// Space operates the focused field.
		if field, ok := f.focused(); ok && field.OnSpace != nil {
			return f, field.OnSpace()
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
	if field, ok := f.focused(); ok && field.OnSpace != nil {
		desc := field.SpaceHint
		if desc == "" {
			desc = "select"
		}
		hints = append(hints, sdk.KeyHint{Key: "Space", Description: desc})
	}
	// Enter confirms: a form-level submit ("run"), else per-field select.
	if f.onSubmit != nil {
		desc := f.submitHint
		if desc == "" {
			desc = "confirm"
		}
		hints = append(hints, sdk.KeyHint{Key: "Enter", Description: desc})
	} else {
		hints = append(hints, sdk.HintSelect)
	}
	return append(hints, sdk.HintCancel)
}
