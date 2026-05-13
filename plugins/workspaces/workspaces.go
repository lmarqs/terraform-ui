package workspaces

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
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

// Plugin implements the workspace management feature.
type Plugin struct {
	svc           sdk.Service
	stack         *sdk.Stack
	status        sdk.Status
	workspaces    []string
	current       string
	selected      int
	errMsg        string
	newName       string
	creating      bool
	scopedContext string
}

// New creates a new workspaces plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		svc: svc,
	}
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	return p
}

func (e *Plugin) ID() string          { return "workspaces" }
func (e *Plugin) Name() string        { return "Workspaces" }
func (e *Plugin) Description() string { return "Manage terraform workspaces" }
func (e *Plugin) Ready() bool         { return e.status == sdk.StatusDone }
func (e *Plugin) Status() sdk.Status  { return e.status }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) Current() string     { return e.current }
func (e *Plugin) Workspaces() []string {
	return e.workspaces
}
func (e *Plugin) IsCreating() bool  { return e.creating }
func (e *Plugin) Stack() *sdk.Stack { return e.stack }

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context. Does not auto-load.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	e.reset()
	return nil
}

// HandleChdirChanged implements sdk.ChdirHandler.
func (e *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	e.svc = e.svc.WithDir(evt.AbsPath)
	e.scopedContext = evt.AbsPath
	e.reset()
	return nil
}

// reset clears all plugin state to initial values.
func (e *Plugin) reset() {
	e.status = sdk.StatusIdle
	e.workspaces = nil
	e.current = ""
	e.errMsg = ""
	e.selected = 0
	e.creating = false
	e.newName = ""
}

// Activate triggers workspace loading when the user enters the plugin.
func (e *Plugin) Activate() tea.Cmd {
	if e.status == sdk.StatusIdle || e.status == sdk.StatusError {
		e.status = sdk.StatusLoading
		return e.loadWorkspaces()
	}
	return nil
}

// Refresh reloads the workspace list.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = sdk.StatusLoading
	e.errMsg = ""
	e.creating = false
	e.newName = ""
	return e.loadWorkspaces()
}

func (e *Plugin) loadWorkspaces() tea.Cmd {
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

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case WorkspaceListMsg:
		if msg.Err != nil {
			e.status = sdk.StatusError
			e.errMsg = msg.Err.Error()
		} else {
			e.status = sdk.StatusDone
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
		return e, nil

	case WorkspaceSwitchMsg:
		if msg.Err != nil {
			e.errMsg = msg.Err.Error()
		} else {
			e.current = msg.Name
			refreshCmd := e.Refresh()
			eventCmd := func() tea.Msg {
				return sdk.WorkspaceChangedEvent{Name: msg.Name}
			}
			return e, tea.Batch(refreshCmd, eventCmd)
		}
		return e, e.Refresh()

	}
	return e, nil
}

// MoveUp moves selection up.
func (e *Plugin) MoveUp() {
	if e.selected > 0 {
		e.selected--
	}
}

// MoveDown moves selection down.
func (e *Plugin) MoveDown() {
	if e.selected < len(e.workspaces)-1 {
		e.selected++
	}
}

// SelectedWorkspace returns the currently selected workspace name.
func (e *Plugin) SelectedWorkspace() string {
	if e.selected < len(e.workspaces) {
		return e.workspaces[e.selected]
	}
	return ""
}

// SwitchToSelected switches to the selected workspace.
func (e *Plugin) SwitchToSelected() tea.Cmd {
	ws := e.SelectedWorkspace()
	if ws == "" || ws == e.current {
		return nil
	}
	return e.switchWorkspace(ws)
}

func (e *Plugin) switchWorkspace(name string) tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		err := svc.WorkspaceSelect(context.Background(), name)
		return WorkspaceSwitchMsg{Name: name, Err: err}
	}
}

func (e *Plugin) createWorkspace(name string) tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		err := svc.WorkspaceNew(context.Background(), name)
		return WorkspaceSwitchMsg{Name: name, Err: err}
	}
}

// DeleteSelected deletes the selected workspace (cannot delete current or default).
func (e *Plugin) DeleteSelected() tea.Cmd {
	ws := e.SelectedWorkspace()
	if ws == "" || ws == e.current || ws == "default" {
		return nil
	}
	svc := e.svc
	return func() tea.Msg {
		err := svc.WorkspaceDelete(context.Background(), ws)
		return WorkspaceSwitchMsg{Name: e.current, Err: err}
	}
}

// View renders the workspaces plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case sdk.StatusIdle, sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Loading workspaces...")

	case sdk.StatusError:
		return sdk.StyleError.Render("Error: " + e.errMsg)

	case sdk.StatusDone:
		return e.renderWorkspaces(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderWorkspaces(width, height int) string {
	var b strings.Builder

	// Creating new workspace input
	if e.creating {
		prompt := sdk.StyleKey.Render("New workspace: ") + e.newName + "_"
		b.WriteString(prompt)
		b.WriteString("\n\n")
	}

	// Calculate visible area
	maxVisible := height - 5
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
			row = sdk.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d workspace(s)", len(e.workspaces)))
	currentInfo := sdk.StyleFaint.Render(fmt.Sprintf("Current: %s", e.current))

	return b.String() + "\n" + count + "  " + currentInfo
}

func (e *Plugin) renderWorkspaceRow(ws string, idx int) string {
	indicator := "  "
	name := sdk.StyleFaint.Render(ws)
	if ws == e.current {
		indicator = sdk.StyleSuccess.Render("* ")
		name = sdk.StyleKey.Render(ws)
	}

	row := fmt.Sprintf("%s%s", indicator, name)

	// Show badge for default workspace
	if ws == "default" {
		row += " " + sdk.StyleFaint.Render("(default)")
	}

	return row
}
