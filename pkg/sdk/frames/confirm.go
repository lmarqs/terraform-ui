package frames

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// ConfirmFrame displays a y/n confirmation prompt and consumes all input
// except y, n, and esc. This prevents accidental actions while confirming.
type ConfirmFrame struct {
	Prompt string
	onYes  func() tea.Cmd
	onNo   func() tea.Cmd
}

// NewConfirmFrame creates a confirmation frame. onNo is optional (nil = just pop).
func NewConfirmFrame(prompt string, onYes func() tea.Cmd, onNo func() tea.Cmd) *ConfirmFrame {
	return &ConfirmFrame{
		Prompt: prompt,
		onYes:  onYes,
		onNo:   onNo,
	}
}

func (f *ConfirmFrame) ID() string { return "confirm" }

func (f *ConfirmFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "y", "Y":
		if f.onYes != nil {
			return nil, f.onYes()
		}
		return nil, nil
	case "n", "N", "esc":
		if f.onNo != nil {
			return nil, f.onNo()
		}
		return nil, nil
	}
	return f, nil
}

func (f *ConfirmFrame) View(width, height int) string {
	return fmt.Sprintf("%s (y/n)", f.Prompt)
}

func (f *ConfirmFrame) Hints() []sdk.KeyHint {
	return []sdk.KeyHint{
		{Key: "y", Description: "confirm"},
		{Key: "n", Description: "cancel"},
	}
}

