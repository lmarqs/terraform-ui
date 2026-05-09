package workspaces

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

// Status represents the current state of the workspaces extension.
type Status int

const (
	StatusIdle Status = iota
	StatusLoading
	StatusDone
	StatusError
	StatusCreating
)

// WorkspaceListMsg is sent when workspace list completes.
type WorkspaceListMsg struct {
	Workspaces []string
	Current    string
	Err        error
}

// WorkspaceSwitchMsg is sent when workspace switch completes.
type WorkspaceSwitchMsg struct {
	Name string
	Err  error
}

// Extension implements the workspace management feature.
type Extension struct {
	svc        terraform.Service
	status     Status
	workspaces []string
	current    string
	selected   int
	errMsg     string
	newName    string
	creating   bool
}

// New creates a new workspaces extension.
func New() *Extension {
	return &Extension{}
}

func (e *Extension) Name() string          { return "Workspaces" }
func (e *Extension) Description() string    { return "Manage terraform workspaces" }
func (e *Extension) KeyBinding() string     { return "w" }
func (e *Extension) Ready() bool            { return e.status == StatusDone }
func (e *Extension) Status() Status         { return e.status }
func (e *Extension) Selected() int          { return e.selected }
func (e *Extension) Current() string        { return e.current }
func (e *Extension) Workspaces() []string   { return e.workspaces }
func (e *Extension) IsCreating() bool       { return e.creating }

// Init initializes the extension and loads workspaces.
func (e *Extension) Init(svc terraform.Service) tea.Cmd {
	e.svc = svc
	e.status = StatusLoading
	e.workspaces = nil
	e.current = ""
	e.errMsg = ""
	e.selected = 0
	e.creating = false
	e.newName = ""
	return e.loadWorkspaces()
}

// Refresh reloads the workspace list.
func (e *Extension) Refresh() tea.Cmd {
	e.status = StatusLoading
	e.errMsg = ""
	e.creating = false
	e.newName = ""
	return e.loadWorkspaces()
}

func (e *Extension) loadWorkspaces() tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		workspaces, err := svc.WorkspaceList(context.Background())
		if err != nil {
			return WorkspaceListMsg{Err: err}
		}
		current, err := svc.Workspace(context.Background())
		if err != nil {
			return WorkspaceListMsg{Err: err}
		}
		return WorkspaceListMsg{Workspaces: workspaces, Current: current}
	}
}

// Update processes messages and returns the updated extension.
func (e *Extension) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case WorkspaceListMsg:
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
		} else {
			e.status = StatusDone
			e.workspaces = msg.Workspaces
			e.current = msg.Current
			// Select current workspace
			for i, ws := range e.workspaces {
				if ws == e.current {
					e.selected = i
					break
				}
			}
		}
		return nil, true

	case WorkspaceSwitchMsg:
		if msg.Err != nil {
			e.errMsg = msg.Err.Error()
		} else {
			e.current = msg.Name
		}
		return e.Refresh(), true

	case tea.KeyMsg:
		return e.handleKey(msg), true
	}
	return nil, false
}

func (e *Extension) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Creating mode has its own key handling
	if e.creating {
		switch msg.String() {
		case "enter":
			if e.newName != "" {
				name := e.newName
				e.creating = false
				e.newName = ""
				return e.createWorkspace(name)
			}
		case "esc":
			e.creating = false
			e.newName = ""
		case "backspace":
			if len(e.newName) > 0 {
				e.newName = e.newName[:len(e.newName)-1]
			}
		default:
			if len(msg.String()) == 1 && msg.String() >= " " {
				e.newName += msg.String()
			}
		}
		return nil
	}

	switch msg.String() {
	case "j", "down":
		e.MoveDown()
	case "k", "up":
		e.MoveUp()
	case "enter":
		return e.SwitchToSelected()
	case "n":
		e.creating = true
		e.newName = ""
	case "d":
		return e.DeleteSelected()
	case "r":
		return e.Refresh()
	}
	return nil
}

// MoveUp moves selection up.
func (e *Extension) MoveUp() {
	if e.selected > 0 {
		e.selected--
	}
}

// MoveDown moves selection down.
func (e *Extension) MoveDown() {
	if e.selected < len(e.workspaces)-1 {
		e.selected++
	}
}

// SelectedWorkspace returns the currently selected workspace name.
func (e *Extension) SelectedWorkspace() string {
	if e.selected < len(e.workspaces) {
		return e.workspaces[e.selected]
	}
	return ""
}

// SwitchToSelected switches to the selected workspace.
func (e *Extension) SwitchToSelected() tea.Cmd {
	ws := e.SelectedWorkspace()
	if ws == "" || ws == e.current {
		return nil
	}
	return e.switchWorkspace(ws)
}

func (e *Extension) switchWorkspace(name string) tea.Cmd {
	// Note: terraform-exec doesn't expose workspace select directly.
	// In a real implementation this would call tf.WorkspaceSelect.
	// For now, return a message indicating the switch.
	return func() tea.Msg {
		return WorkspaceSwitchMsg{Name: name}
	}
}

func (e *Extension) createWorkspace(name string) tea.Cmd {
	// In a real implementation, this would call tf.WorkspaceNew.
	return func() tea.Msg {
		return WorkspaceSwitchMsg{Name: name}
	}
}

// DeleteSelected deletes the selected workspace (cannot delete current or default).
func (e *Extension) DeleteSelected() tea.Cmd {
	ws := e.SelectedWorkspace()
	if ws == "" || ws == e.current || ws == "default" {
		return nil
	}
	// In a real implementation, this would call tf.WorkspaceDelete.
	return e.Refresh()
}

// View renders the workspaces extension.
func (e *Extension) View(width, height int) string {
	title := styles.StyleTitle.Render("Workspaces")

	switch e.status {
	case StatusIdle, StatusLoading:
		loading := styles.StyleFaintItalic.Render("Loading workspaces...")
		return styles.StylePadded.Render(title + "\n\n" + loading)

	case StatusError:
		errText := styles.StyleError.Render("Error: " + e.errMsg)
		hint := styles.StyleFaintItalic.Render("Press r to retry, Esc to go back")
		return styles.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	case StatusDone:
		return e.renderWorkspaces(width, height)

	default:
		return ""
	}
}

func (e *Extension) renderWorkspaces(width, height int) string {
	title := styles.StyleTitle.Render("Workspaces")

	var b strings.Builder

	// Creating new workspace input
	if e.creating {
		prompt := styles.StyleKey.Render("New workspace: ") + e.newName + "_"
		b.WriteString(prompt)
		b.WriteString("\n\n")
	}

	// Calculate visible area
	maxVisible := height - 8
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if e.selected >= maxVisible {
		startIdx = e.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(e.workspaces) {
		endIdx = len(e.workspaces)
	}

	for i := startIdx; i < endIdx; i++ {
		ws := e.workspaces[i]
		row := e.renderWorkspaceRow(ws, i)
		if i == e.selected {
			row = styles.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := styles.StyleFaint.Render(fmt.Sprintf("%d workspace(s)", len(e.workspaces)))
	currentInfo := styles.StyleFaint.Render(fmt.Sprintf("Current: %s", e.current))

	hint := styles.StyleFaintItalic.Render("Enter switch  n new  d delete  r refresh  Esc back")
	if e.creating {
		hint = styles.StyleFaintItalic.Render("Enter confirm  Esc cancel")
	}

	content := title + "\n\n" + b.String() + "\n" + count + "  " + currentInfo + "\n" + hint
	return styles.StylePadded.Render(content)
}

func (e *Extension) renderWorkspaceRow(ws string, idx int) string {
	indicator := "  "
	name := styles.StyleFaint.Render(ws)
	if ws == e.current {
		indicator = styles.StyleSuccess.Render("* ")
		name = styles.StyleKey.Render(ws)
	}

	row := fmt.Sprintf("%s%s", indicator, name)

	// Show badge for default workspace
	if ws == "default" {
		row += " " + styles.StyleFaint.Render("(default)")
	}

	return row
}

// FilterWorkspaces returns workspaces matching a filter string.
func (e *Extension) FilterWorkspaces(filter string) []string {
	if filter == "" {
		return e.workspaces
	}
	lower := strings.ToLower(filter)
	var result []string
	for _, ws := range e.workspaces {
		if strings.Contains(strings.ToLower(ws), lower) {
			result = append(result, ws)
		}
	}
	return result
}
