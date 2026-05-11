package macro

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// mockModel is a simple BubbleTea model for testing the driver.
type mockModel struct {
	keys    []string
	content string
	ready   bool
	width   int
	height  int
}

func (m mockModel) Init() tea.Cmd {
	return func() tea.Msg { return readyMsg{} }
}

type readyMsg struct{}

func (m mockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		m.keys = append(m.keys, key)
		m.content = "keys: " + strings.Join(m.keys, ", ")
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case readyMsg:
		m.ready = true
		m.content = "ready"
		return m, nil
	}
	return m, nil
}

func (m mockModel) View() string {
	if m.content == "" {
		return "empty"
	}
	return m.content
}

func TestDriverInit(t *testing.T) {
	model := mockModel{}
	d := NewDriver(model, 80, 24)
	d.Init()

	view := d.View()
	if view != "ready" {
		t.Errorf("view after init = %q, want %q", view, "ready")
	}
}

func TestDriverSendKey(t *testing.T) {
	model := mockModel{content: "initial"}
	d := NewDriver(model, 80, 24)

	d.SendKey("p")
	if !d.ViewContains("keys: p") {
		t.Errorf("view = %q, want to contain 'keys: p'", d.View())
	}

	d.SendKey("q")
	if !d.ViewContains("keys: p, q") {
		t.Errorf("view = %q, want to contain 'keys: p, q'", d.View())
	}
}

func TestDriverSendKeySpecial(t *testing.T) {
	model := mockModel{content: "initial"}
	d := NewDriver(model, 80, 24)

	d.SendKey("enter")
	if !d.ViewContains("enter") {
		t.Errorf("view = %q, should contain 'enter'", d.View())
	}
}

func TestDriverWaitUntil(t *testing.T) {
	model := mockModel{content: "waiting"}
	d := NewDriver(model, 80, 24)

	// Already satisfied
	err := d.WaitUntil(func(v string) bool { return v == "waiting" }, 100*time.Millisecond)
	if err != nil {
		t.Errorf("should succeed immediately: %v", err)
	}

	// Timeout
	err = d.WaitUntil(func(v string) bool { return v == "never" }, 50*time.Millisecond)
	if err == nil {
		t.Error("should timeout")
	}
}

func TestDriverViewContains(t *testing.T) {
	model := mockModel{content: "hello world"}
	d := NewDriver(model, 80, 24)

	if !d.ViewContains("hello") {
		t.Error("should contain 'hello'")
	}
	if !d.ViewContains("world") {
		t.Error("should contain 'world'")
	}
	if d.ViewContains("missing") {
		t.Error("should not contain 'missing'")
	}
}

func TestDriverResize(t *testing.T) {
	model := mockModel{}
	d := NewDriver(model, 80, 24)

	d.SendMsg(tea.WindowSizeMsg{Width: 120, Height: 40})
	// The model received the resize (verified by its internal state)
	// We can't inspect internal state directly, but the driver shouldn't panic
}

// asyncModel returns a command that resolves after processing
type asyncModel struct {
	count   int
	content string
}

func (m asyncModel) Init() tea.Cmd { return nil }

func (m asyncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.count++
		return m, func() tea.Msg { return countMsg(m.count) }
	case countMsg:
		m.content = "count: " + strings.Repeat("x", int(msg.(countMsg)))
		return m, nil
	}
	return m, nil
}

type countMsg int

func (m asyncModel) View() string {
	if m.content == "" {
		return "empty"
	}
	return m.content
}

func TestDriverHandlesCommandResults(t *testing.T) {
	model := asyncModel{}
	d := NewDriver(model, 80, 24)

	d.SendKey("a")
	if !d.ViewContains("count: x") {
		t.Errorf("command result should be processed, view = %q", d.View())
	}

	d.SendKey("b")
	if !d.ViewContains("count: xx") {
		t.Errorf("second command result, view = %q", d.View())
	}
}

// batchModel returns multiple commands at once
type batchModel struct {
	messages []string
}

func (m batchModel) Init() tea.Cmd { return nil }

type batchResultMsg string

func (m batchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Batch(
			func() tea.Msg { return batchResultMsg("first") },
			func() tea.Msg { return batchResultMsg("second") },
		)
	case batchResultMsg:
		m.messages = append(m.messages, string(msg))
		return m, nil
	}
	return m, nil
}

func (m batchModel) View() string {
	return strings.Join(m.messages, ",")
}

func TestDriverHandlesBatchCommands(t *testing.T) {
	model := batchModel{}
	d := NewDriver(model, 80, 24)

	d.SendKey("x")
	view := d.View()
	if !strings.Contains(view, "first") || !strings.Contains(view, "second") {
		t.Errorf("batch commands should both be processed, view = %q", view)
	}
}

func TestKeyToMsg(t *testing.T) {
	tests := []struct {
		key      string
		wantType tea.KeyType
	}{
		{"enter", tea.KeyEnter},
		{"esc", tea.KeyEsc},
		{"tab", tea.KeyTab},
		{"backspace", tea.KeyBackspace},
		{"up", tea.KeyUp},
		{"down", tea.KeyDown},
		{"left", tea.KeyLeft},
		{"right", tea.KeyRight},
		{"space", tea.KeySpace},
		{"ctrl+c", tea.KeyCtrlC},
		{"ctrl+w", tea.KeyCtrlW},
		{"p", tea.KeyRunes},
		{"q", tea.KeyRunes},
		{"/", tea.KeyRunes},
		{":", tea.KeyRunes},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			msg := keyToMsg(tt.key)
			if msg.Type != tt.wantType {
				t.Errorf("keyToMsg(%q).Type = %v, want %v", tt.key, msg.Type, tt.wantType)
			}
		})
	}
}
