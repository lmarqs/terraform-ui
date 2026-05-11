package macro

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Driver provides programmatic control over a BubbleTea model.
// It dispatches messages synchronously and executes returned commands,
// feeding their results back into the model.
type Driver struct {
	model  tea.Model
	width  int
	height int
}

// NewDriver creates a driver wrapping the given model at the specified dimensions.
func NewDriver(model tea.Model, width, height int) *Driver {
	return &Driver{
		model:  model,
		width:  width,
		height: height,
	}
}

// Init calls the model's Init() and processes all returned commands.
func (d *Driver) Init() {
	d.SendMsg(tea.WindowSizeMsg{Width: d.width, Height: d.height})

	cmd := d.model.Init()
	d.processCmd(cmd)
}

// SendKey sends a key event to the model and processes all resulting commands.
func (d *Driver) SendKey(key string) {
	msg := keyToMsg(key)
	d.SendMsg(msg)
}

// SendMsg sends an arbitrary message and processes all resulting commands.
func (d *Driver) SendMsg(msg tea.Msg) {
	var cmd tea.Cmd
	d.model, cmd = d.model.Update(msg)
	d.processCmd(cmd)
}

// View returns the current rendered view.
func (d *Driver) View() string {
	return d.model.View()
}

// WaitUntil repeatedly processes pending commands until the predicate returns true
// or the timeout expires.
func (d *Driver) WaitUntil(predicate func(view string) bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if predicate(d.View()) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %v waiting for condition", timeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// ViewContains checks if the current view contains the given substring.
func (d *Driver) ViewContains(substr string) bool {
	return strings.Contains(d.View(), substr)
}

// processCmd executes a tea.Cmd and feeds the result back into the model.
// Handles tea.Batch (multiple commands) recursively.
func (d *Driver) processCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}

	msg := cmd()
	if msg == nil {
		return
	}

	switch msg := msg.(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			d.processCmd(c)
		}
	default:
		d.SendMsg(msg)
	}
}

// keyToMsg converts a key string to a tea.KeyMsg.
func keyToMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+w":
		return tea.KeyMsg{Type: tea.KeyCtrlW}
	case "ctrl+t":
		return tea.KeyMsg{Type: tea.KeyCtrlT}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	default:
		if len(key) == 1 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}
		// Multi-char key names that BubbleTea recognizes
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}
