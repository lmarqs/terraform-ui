package workspace

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// WorkspaceListMsg is sent when workspace list completes.
type WorkspaceListMsg struct {
	Workspaces []string
	Current    string
	Err        error
}

// WorkspaceDeleteMsg is sent when workspace deletion completes.
type WorkspaceDeleteMsg struct {
	Err error
}

// WorkspaceSwitchMsg is sent when workspace switch completes.
type WorkspaceSwitchMsg struct {
	Name    string
	Err     error
	PopBack bool
}

// WorkspaceCreateMsg is sent when workspace creation completes.
type WorkspaceCreateMsg struct {
	Name string
	Err  error
}

// Plugin implements the workspace management feature.
type Plugin struct {
	svc           sdk.Service
	getCtx        func() *sdk.Context
	stack         *sdk.Stack
	timer         ui.Timer
	status        sdk.Status
	workspaces    []string
	current       string
	selected      int
	errMsg        string
	loadingMsg    string
	cancelFn      context.CancelFunc
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

func (e *Plugin) ID() string          { return "workspace" }
func (e *Plugin) Name() string        { return "Workspace" }
func (e *Plugin) Description() string { return "Manage terraform workspace" }
func (e *Plugin) Ready() bool         { return e.status == sdk.StatusDone }
func (e *Plugin) Status() sdk.Status  { return e.status }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) Current() string     { return e.current }
func (e *Plugin) Workspaces() []string {
	return e.workspaces
}
func (e *Plugin) Stack() *sdk.Stack { return e.stack }

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init wires the plugin to its shared dependencies. Does not auto-load.
func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	e.svc = deps.Service
	e.getCtx = deps.Context
	e.reset()
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler. Replaces the
// scoped service from the new Context's Service handle (already chdir-scoped
// by the app) and clears any in-memory workspace list.
func (e *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if ev.Next == nil {
		return nil
	}
	if ev.Next.Service != nil {
		e.svc = ev.Next.Service
	}
	e.reset()
	return nil
}

// reset clears all plugin state to initial values.
func (e *Plugin) reset() {
	e.status = sdk.StatusIdle
	e.workspaces = nil
	e.current = ""
	e.errMsg = ""
	e.loadingMsg = ""
	e.selected = 0
}

// Activate triggers workspace loading when the user enters the plugin.
func (e *Plugin) Activate() tea.Cmd {
	if e.status == sdk.StatusIdle || e.status == sdk.StatusError {
		e.status = sdk.StatusLoading
		return tea.Batch(e.loadWorkspaces(), e.timer.Start())
	}
	return nil
}

// Refresh reloads the workspace list.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = sdk.StatusLoading
	e.errMsg = ""
	e.loadingMsg = "Loading workspaces..."
	return tea.Batch(e.loadWorkspaces(), e.timer.Start())
}

// Cancel aborts any in-flight terraform operation.
func (e *Plugin) Cancel() {
	if e.cancelFn != nil {
		e.cancelFn()
		e.cancelFn = nil
	}
}

func (e *Plugin) loadWorkspaces() tea.Cmd {
	e.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel
	svc := e.svc
	return func() tea.Msg {
		workspaces, err := svc.WorkspaceList(ctx)
		if err != nil {
			return WorkspaceListMsg{Err: err}
		}
		current, err := svc.Workspace(ctx)
		if err != nil {
			return WorkspaceListMsg{Err: err}
		}
		return WorkspaceListMsg{Workspaces: workspaces, Current: current}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return e, e.timer.Tick()

	case WorkspaceListMsg:
		e.timer.Stop()
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
		e.timer.Stop()
		if msg.Err != nil {
			e.status = sdk.StatusError
			e.errMsg = msg.Err.Error()
			return e, nil
		}
		e.current = msg.Name
		if msg.PopBack {
			e.status = sdk.StatusIdle
			return e, func() tea.Msg {
				return sdk.ContextSwitchRequestMsg{Chdir: e.getCtx().Chdir, Workspace: msg.Name}
			}
		}
		return e, tea.Batch(e.Refresh(), func() tea.Msg {
			return sdk.ContextSwitchRequestMsg{Chdir: e.getCtx().Chdir, Workspace: msg.Name}
		})

	case WorkspaceCreateMsg:
		e.timer.Stop()
		if msg.Err != nil {
			e.status = sdk.StatusError
			e.errMsg = msg.Err.Error()
			return e, nil
		}
		e.current = msg.Name
		return e, tea.Batch(e.Refresh(), func() tea.Msg {
			return sdk.ContextSwitchRequestMsg{Chdir: e.getCtx().Chdir, Workspace: msg.Name}
		})

	case WorkspaceDeleteMsg:
		e.timer.Stop()
		if msg.Err != nil {
			e.status = sdk.StatusError
			e.errMsg = msg.Err.Error()
			return e, nil
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

// SwitchToSelected switches to the selected workspace and pops back, or deactivates if already current.
func (e *Plugin) SwitchToSelected() tea.Cmd {
	ws := e.SelectedWorkspace()
	if ws == "" {
		return nil
	}
	if ws == e.current {
		return func() tea.Msg { return sdk.DeactivateMsg{} }
	}
	e.status = sdk.StatusLoading
	e.loadingMsg = fmt.Sprintf("Switching to %s...", ws)
	return tea.Batch(e.selectWorkspace(ws, true), e.timer.Start())
}

// SelectCurrent selects the workspace under cursor and stays in the list.
func (e *Plugin) SelectCurrent() tea.Cmd {
	ws := e.SelectedWorkspace()
	if ws == "" || ws == e.current {
		return nil
	}
	e.status = sdk.StatusLoading
	e.loadingMsg = fmt.Sprintf("Selecting %s...", ws)
	return tea.Batch(e.selectWorkspace(ws, false), e.timer.Start())
}

func (e *Plugin) selectWorkspace(name string, popBack bool) tea.Cmd {
	e.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel
	svc := e.svc
	return func() tea.Msg {
		err := svc.WorkspaceSelect(ctx, name)
		return WorkspaceSwitchMsg{Name: name, Err: err, PopBack: popBack}
	}
}

func (e *Plugin) createWorkspace(name string) tea.Cmd {
	e.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel
	svc := e.svc
	return func() tea.Msg {
		err := svc.WorkspaceNew(ctx, name, sdk.WorkspaceNewOptions{})
		return WorkspaceCreateMsg{Name: name, Err: err}
	}
}

// deleteWorkspace starts deletion of the named workspace with loading feedback.
func (e *Plugin) deleteWorkspace(name string) tea.Cmd {
	e.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel
	e.status = sdk.StatusLoading
	e.loadingMsg = fmt.Sprintf("Deleting %s...", name)
	svc := e.svc
	return tea.Batch(func() tea.Msg {
		err := svc.WorkspaceDelete(ctx, name, sdk.WorkspaceDeleteOptions{})
		return WorkspaceDeleteMsg{Err: err}
	}, e.timer.Start())
}

// startCreate starts creation of a new workspace with loading feedback.
func (e *Plugin) startCreate(name string) tea.Cmd {
	e.status = sdk.StatusLoading
	e.loadingMsg = fmt.Sprintf("Creating %s...", name)
	return tea.Batch(e.createWorkspace(name), e.timer.Start())
}

// View renders the workspaces plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case sdk.StatusIdle, sdk.StatusLoading:
		msg := e.loadingMsg
		if msg == "" {
			msg = "Loading workspaces..."
		}
		return sdk.StyleFaintItalic.Render(msg + " " + e.timer.FormatElapsed())

	case sdk.StatusError:
		return sdk.StyleError.Render("Error: " + e.errMsg)

	case sdk.StatusDone:
		return e.renderWorkspaces(width, height)

	default:
		return ""
	}
}

func (e *Plugin) CursorPosition() (int, int) {
	if e.status != sdk.StatusDone || len(e.workspaces) == 0 {
		return 0, 0
	}
	return e.selected + 1, len(e.workspaces)
}

func (e *Plugin) workspaceActions() []ui.ActionChip {
	chips := []ui.ActionChip{
		{Key: "n", Label: "new"},
	}
	ws := e.SelectedWorkspace()
	if ws != "" && ws != e.current && ws != "default" {
		chips = append(chips, ui.ActionChip{Key: "d", Label: "delete"})
	}
	return chips
}

func (e *Plugin) renderWorkspaces(width, height int) string {
	var b strings.Builder

	actions := e.workspaceActions()
	maxVisible := ui.HeightBudget(height, 3, ui.ActionsBarHeight)

	startIdx := 0
	if e.selected >= maxVisible {
		startIdx = e.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(e.workspaces) {
		endIdx = len(e.workspaces)
	}

	rows := make([]string, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		rows = append(rows, e.renderWorkspaceRow(e.workspaces[i], i))
	}

	panel := ui.NewContentPanel()
	panel.SelectedStyle = func(s string, w int) string {
		return sdk.StyleSelected.Width(w).Render(s)
	}

	b.WriteString(panel.Render(ui.RenderParams{
		Rows:         rows,
		Width:        width,
		Height:       maxVisible,
		TotalItems:   len(e.workspaces),
		Cursor:       e.selected - startIdx,
		ScrollOffset: startIdx,
	}))
	b.WriteByte('\n')

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d workspace(s)", len(e.workspaces)))
	currentInfo := sdk.StyleFaint.Render(fmt.Sprintf("Current: %s", e.current))

	return b.String() + "\n" + count + "  " + currentInfo + ui.RenderActionsBar(actions, width)
}

func isValidWorkspaceName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, c := range name {
		isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		isPunct := c == '.' || c == '_' || c == '-'
		if !isAlpha && !isDigit && !isPunct {
			return false
		}
	}
	return true
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
