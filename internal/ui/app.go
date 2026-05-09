package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/ui/components"
	"github.com/lmarqs/terraform-ui/internal/ui/views"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type App struct {
	cfg      config.Config
	svc      sdk.Service
	registry *plugin.Registry
	session  *sdk.Session
	width    int
	height   int

	header    components.Header
	statusBar components.StatusBar
	homeView  views.HomeView

	activePlugin  sdk.Plugin // nil = home screen
	activeProject string     // tracks last known active project for header updates
}

func NewApp(cfg config.Config, svc sdk.Service, registry *plugin.Registry) App {
	return App{
		cfg:       cfg,
		svc:       svc,
		registry:  registry,
		session:   sdk.NewSession(),
		header:    components.NewHeader(cfg.Dir, "default", cfg.TerraformBinary, 0),
		statusBar: components.NewStatusBar(),
		homeView:  views.NewHomeView(registry.All()),
	}
}

func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{a.loadWorkspace}

	// Initialize all plugins
	ctx := &plugin.Context{
		Dir:       a.cfg.Dir,
		Workspace: "default",
		Service:   a.svc,
		Logger:    logging.Logger(),
		Session:   a.session,
	}
	for _, p := range a.registry.All() {
		if cmd := p.Init(ctx); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

// --- Messages ---

type workspaceLoadedMsg struct {
	workspace string
}

// --- Async commands ---

func (a App) loadWorkspace() tea.Msg {
	ws, err := a.svc.Workspace(context.Background())
	if err != nil {
		return workspaceLoadedMsg{workspace: "default"}
	}
	return workspaceLoadedMsg{workspace: ws}
}

// --- Update ---

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case workspaceLoadedMsg:
		a.header = components.NewHeader(a.cfg.Dir, msg.workspace, a.cfg.TerraformBinary, 0)
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	}

	// If a plugin is active, delegate the message to it
	if a.activePlugin != nil {
		updated, cmd := a.activePlugin.Update(msg)
		a.activePlugin = updated
		a.syncActiveProject()
		return a, cmd
	}

	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	activeView := "home"
	if a.activePlugin != nil {
		activeView = a.activePlugin.ID()
	}
	logging.Logger().Debug("key.press", "key", msg.String(), "view", activeView)

	// Global keys
	switch msg.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "q":
		if a.activePlugin != nil {
			prev := a.activePlugin.ID()
			a.activePlugin = nil
			logging.Logger().Debug("view.transition", "from", prev, "to", "home")
			return a, nil
		}
		return a, tea.Quit
	case "esc":
		if a.activePlugin != nil {
			prev := a.activePlugin.ID()
			a.activePlugin = nil
			logging.Logger().Debug("view.transition", "from", prev, "to", "home")
			return a, nil
		}
		return a, nil
	}

	// If a plugin is active, delegate key to it
	if a.activePlugin != nil {
		updated, cmd := a.activePlugin.Update(msg)
		a.activePlugin = updated
		a.syncActiveProject()
		return a, cmd
	}

	// Home screen key handling
	return a.updateHome(msg)
}

func (a App) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		a.homeView = a.homeView.MoveUp()
	case "down", "j":
		a.homeView = a.homeView.MoveDown()
	case "enter":
		item := a.homeView.SelectedItem()
		if p, ok := a.registry.ByKey(item.Key); ok {
			a.activePlugin = p
			logging.Logger().Debug("plugin.activate", "id", p.ID())
			logging.Logger().Debug("view.transition", "from", "home", "to", p.ID())
			return a, a.activatePlugin(p)
		}
		return a, nil
	default:
		// Check if key matches a plugin binding
		if p, ok := a.registry.ByKey(msg.String()); ok {
			a.activePlugin = p
			logging.Logger().Debug("plugin.activate", "id", p.ID())
			logging.Logger().Debug("view.transition", "from", "home", "to", p.ID())
			return a, a.activatePlugin(p)
		}
	}
	return a, nil
}

// syncActiveProject checks if the active project changed in session and updates the header.
func (a *App) syncActiveProject() {
	if a.session == nil {
		return
	}
	if project, ok := sdk.GetTyped[string](a.session, sdk.SessionKeyActiveProject); ok {
		if project != a.activeProject {
			a.activeProject = project
			a.header = a.header.WithProject(project)
		}
	}
}

func (a App) activatePlugin(p sdk.Plugin) tea.Cmd {
	if activatable, ok := p.(sdk.Activatable); ok {
		return activatable.Activate()
	}
	return nil
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
	if a.activePlugin != nil {
		content = a.activePlugin.View(a.width, contentHeight)
	} else {
		content = a.homeView.Render(a.width, contentHeight)
	}

	contentStyle := lipgloss.NewStyle().
		Width(a.width).
		Height(contentHeight)
	content = contentStyle.Render(content)

	statusBar := a.statusBar.Render(a.width)

	return header + "\n" + content + "\n" + statusBar
}
