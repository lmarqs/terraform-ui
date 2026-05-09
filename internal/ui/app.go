package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/internal/config"
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
}

func NewApp(cfg config.Config) App {
	return App{
		cfg:            cfg,
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
	return a.loadProjects
}

type projectsLoadedMsg struct {
	modules []string
}

func (a App) loadProjects() tea.Msg {
	modules, err := a.cfg.DiscoverProjects()
	if err != nil {
		return projectsLoadedMsg{modules: []string{a.cfg.Dir}}
	}
	return projectsLoadedMsg{modules: modules}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsLoadedMsg:
		a.modulesView = a.modulesView.SetModules(msg.modules, 0)
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if a.activeView != ViewHome {
				a.activeView = ViewHome
				return a, nil
			}
			return a, tea.Quit

		case "esc":
			a.activeView = ViewHome
			return a, nil

		case "?":
			return a, nil

		case "/":
			return a, nil
		}

		switch a.activeView {
		case ViewHome:
			return a.updateHome(msg)
		case ViewWorkspaces:
			return a.updateWorkspaces(msg)
		case ViewModules:
			return a.updateModules(msg)
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.header = components.NewHeader(a.cfg.Dir, "default", 0)
		a.statusBar = components.NewStatusBar()
		return a, nil
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
	case "r":
		a.activeView = ViewRisk
	case "b":
		a.activeView = ViewBlastRadius
	case "a":
		a.activeView = ViewApply
	case "s":
		a.activeView = ViewState
	case "w":
		a.activeView = ViewWorkspaces
	case "m":
		a.activeView = ViewModules
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
	case "r":
		a.activeView = ViewRisk
	case "b":
		a.activeView = ViewBlastRadius
	case "a":
		a.activeView = ViewApply
	case "s":
		a.activeView = ViewState
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
