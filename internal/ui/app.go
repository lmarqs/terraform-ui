package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/components"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
	"github.com/lmarqs/terraform-ui/internal/ui/views"
)

type View int

const (
	ViewHome View = iota
	ViewState
	ViewPlan
	ViewApply
	ViewRisk
	ViewBlastRadius
	ViewWorkspaces
	ViewModules
)

type App struct {
	cfg        config.Config
	svc        terraform.Service
	width      int
	height     int
	activeView View

	header         components.Header
	statusBar      components.StatusBar
	homeView       views.HomeView
	stateView      views.StateView
	planView       views.PlanView
	applyView      views.ApplyView
	workspacesView views.WorkspacesView
	modulesView    views.ModulesView

	stateFiltering bool
}

func NewApp(cfg config.Config, svc terraform.Service) App {
	return App{
		cfg:            cfg,
		svc:            svc,
		activeView:     ViewHome,
		header:         components.NewHeader(cfg.Dir, "default", 0),
		statusBar:      components.NewStatusBar(),
		homeView:       views.NewHomeView(),
		stateView:      views.NewStateView(),
		planView:       views.NewPlanView(),
		applyView:      views.NewApplyView(),
		workspacesView: views.NewWorkspacesView(),
		modulesView:    views.NewModulesView(),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadProjects, a.loadWorkspace)
}

// --- Messages ---

type projectsLoadedMsg struct {
	modules []string
}

type workspaceLoadedMsg struct {
	workspace string
}

type planResultMsg struct {
	summary *terraform.PlanSummary
	err     error
}

type stateResultMsg struct {
	resources []terraform.Resource
	err       error
}

type applyResultMsg struct {
	err error
}

// --- Async commands ---

func (a App) loadProjects() tea.Msg {
	modules, err := a.cfg.DiscoverProjects()
	if err != nil {
		return projectsLoadedMsg{modules: []string{a.cfg.Dir}}
	}
	return projectsLoadedMsg{modules: modules}
}

func (a App) loadWorkspace() tea.Msg {
	ws, err := a.svc.Workspace(context.Background())
	if err != nil {
		return workspaceLoadedMsg{workspace: "default"}
	}
	return workspaceLoadedMsg{workspace: ws}
}

func (a App) runPlan() tea.Msg {
	summary, err := a.svc.Plan(context.Background(), a.cfg.Targets)
	return planResultMsg{summary: summary, err: err}
}

func (a App) runStateList() tea.Msg {
	resources, err := a.svc.StateList(context.Background())
	return stateResultMsg{resources: resources, err: err}
}

func (a App) runApply() tea.Msg {
	err := a.svc.Apply(context.Background(), a.cfg.Targets)
	return applyResultMsg{err: err}
}

// --- Update ---

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsLoadedMsg:
		a.modulesView = a.modulesView.SetModules(msg.modules, 0)
		return a, nil

	case workspaceLoadedMsg:
		a.header = components.NewHeader(a.cfg.Dir, msg.workspace, 0)
		return a, nil

	case planResultMsg:
		if msg.err != nil {
			a.planView = a.planView.SetError(msg.err.Error())
		} else {
			a.planView = a.planView.SetResult(msg.summary)
		}
		return a, nil

	case stateResultMsg:
		if msg.err != nil {
			a.stateView = a.stateView.SetError(msg.err.Error())
		} else {
			a.stateView = a.stateView.SetResources(msg.resources)
		}
		return a, nil

	case applyResultMsg:
		if msg.err != nil {
			a.applyView = a.applyView.SetError(msg.err.Error())
		} else {
			a.applyView = a.applyView.SetSuccess()
		}
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	}

	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "q":
		if a.stateFiltering {
			// Don't quit when typing in filter
			break
		}
		if a.activeView != ViewHome {
			a.activeView = ViewHome
			a.stateFiltering = false
			return a, nil
		}
		return a, tea.Quit
	case "esc":
		if a.stateFiltering {
			a.stateFiltering = false
			a.stateView = a.stateView.SetFilter("")
			return a, nil
		}
		if a.activeView != ViewHome {
			a.activeView = ViewHome
			return a, nil
		}
		return a, nil
	}

	// View-specific keys
	switch a.activeView {
	case ViewHome:
		return a.updateHome(msg)
	case ViewPlan:
		return a.updatePlan(msg)
	case ViewState:
		return a.updateState(msg)
	case ViewApply:
		return a.updateApply(msg)
	case ViewWorkspaces:
		return a.updateWorkspaces(msg)
	case ViewModules:
		return a.updateModules(msg)
	}

	return a, nil
}

func (a App) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		a.homeView = a.homeView.MoveUp()
	case "down", "j":
		a.homeView = a.homeView.MoveDown()
	case "enter":
		return a.dispatchAction(a.homeView.SelectedItem())
	case "p":
		a.activeView = ViewPlan
		a.planView = a.planView.SetLoading()
		return a, a.runPlan
	case "r":
		a.activeView = ViewRisk
	case "b":
		a.activeView = ViewBlastRadius
	case "a":
		a.activeView = ViewApply
	case "s":
		a.activeView = ViewState
		a.stateView = a.stateView.SetLoading()
		return a, a.runStateList
	case "w":
		a.activeView = ViewWorkspaces
	case "m":
		a.activeView = ViewModules
	}
	return a, nil
}

func (a App) updatePlan(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		a.planView = a.planView.MoveUp()
	case "down", "j":
		a.planView = a.planView.MoveDown()
	case "enter":
		// Retry plan from idle or error state
		a.planView = a.planView.SetLoading()
		return a, a.runPlan
	case "a":
		// Apply from plan view
		a.activeView = ViewApply
		a.applyView = a.applyView.SetRunning()
		return a, a.runApply
	}
	return a, nil
}

func (a App) updateState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.stateFiltering {
		switch msg.String() {
		case "enter":
			a.stateFiltering = false
		case "backspace":
			a.stateView = a.stateView.BackspaceFilter()
		default:
			if len(msg.String()) == 1 {
				a.stateView = a.stateView.AppendFilter(msg.String())
			}
		}
		return a, nil
	}

	switch msg.String() {
	case "up", "k":
		a.stateView = a.stateView.MoveUp()
	case "down", "j":
		a.stateView = a.stateView.MoveDown()
	case "/":
		a.stateFiltering = true
	case "r":
		a.stateView = a.stateView.SetLoading()
		return a, a.runStateList
	}
	return a, nil
}

func (a App) updateApply(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if a.applyView.Status() == views.ApplyStatusIdle {
			a.applyView = a.applyView.SetRunning()
			return a, a.runApply
		}
	}
	return a, nil
}

func (a App) updateWorkspaces(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		a.workspacesView = a.workspacesView.MoveUp()
	case "down", "j":
		a.workspacesView = a.workspacesView.MoveDown()
	case "enter":
		// TODO: switch workspace
	}
	return a, nil
}

func (a App) updateModules(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		a.modulesView = a.modulesView.MoveUp()
	case "down", "j":
		a.modulesView = a.modulesView.MoveDown()
	case "enter":
		selected := a.modulesView.SelectedModule()
		if selected != "" {
			a.cfg.Dir = selected
			a.header = components.NewHeader(selected, "default", 0)
			a.activeView = ViewHome
		}
	}
	return a, nil
}

func (a App) dispatchAction(item views.MenuItem) (tea.Model, tea.Cmd) {
	switch item.Key {
	case "p":
		a.activeView = ViewPlan
		a.planView = a.planView.SetLoading()
		return a, a.runPlan
	case "r":
		a.activeView = ViewRisk
	case "b":
		a.activeView = ViewBlastRadius
	case "a":
		a.activeView = ViewApply
	case "s":
		a.activeView = ViewState
		a.stateView = a.stateView.SetLoading()
		return a, a.runStateList
	case "w":
		a.activeView = ViewWorkspaces
	}
	return a, nil
}

func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	headerHeight := 3
	statusBarHeight := 1
	contentHeight := a.height - headerHeight - statusBarHeight

	header := a.header.Render(a.width)

	var content string
	switch a.activeView {
	case ViewHome:
		content = a.homeView.Render(a.width, contentHeight)
	case ViewState:
		content = a.stateView.Render(a.width, contentHeight)
	case ViewPlan:
		content = a.planView.Render(a.width, contentHeight)
	case ViewApply:
		content = a.applyView.Render(a.width, contentHeight)
	case ViewRisk:
		content = renderPlaceholder("Risk Analysis", "Run plan first to see risk classification")
	case ViewBlastRadius:
		content = renderPlaceholder("Blast Radius", "Run plan first to see affected modules and dependencies")
	case ViewWorkspaces:
		content = a.workspacesView.Render(a.width, contentHeight)
	case ViewModules:
		content = a.modulesView.Render(a.width, contentHeight)
	}

	contentStyle := lipgloss.NewStyle().
		Width(a.width).
		Height(contentHeight)
	content = contentStyle.Render(content)

	statusBar := a.statusBar.Render(a.width)

	return header + "\n" + content + "\n" + statusBar
}

func renderPlaceholder(title, message string) string {
	return styles.StylePadded.Render(styles.StyleTitle.Render(title) + "\n\n" + styles.StyleFaintItalic.Render(message))
}
