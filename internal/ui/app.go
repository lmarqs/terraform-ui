package ui

import (
	"context"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/editor"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/components"
	"github.com/lmarqs/terraform-ui/internal/ui/views"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuicontext "github.com/lmarqs/terraform-ui/plugins/context"
	tfuistate "github.com/lmarqs/terraform-ui/plugins/state"
)

type App struct {
	cfg         config.Config
	svc         sdk.Service
	registry    *plugin.Registry
	session     *sdk.Session
	sourceIndex *terraform.SourceIndex
	width       int
	height      int

	header        components.Header
	contentBorder components.ContentBorder
	commandBar    components.CommandBar
	statusBar     components.StatusBar
	homeView      views.HomeView

	activePlugin  sdk.Plugin // nil = home screen
	activeOverlay sdk.Overlay
	activeScope   string // tracks last known active scope for header updates
	commandMode   bool
	commandInput  string

	inputActive   bool
	inputPrompt   string
	inputAnswer   string
	inputCallback func(string) tea.Cmd
}

func NewApp(cfg config.Config, svc sdk.Service, registry *plugin.Registry) App {
	workDir := cfg.WorkingDir()
	sourceIndex, _ := terraform.NewSourceIndex(workDir)
	header := components.NewHeader(workDir, "default")
	if cfg.BaseDir != "" {
		header = header.WithScope(cfg.BaseDir)
	}
	return App{
		cfg:           cfg,
		svc:           svc,
		registry:      registry,
		session:       sdk.NewSession(),
		sourceIndex:   sourceIndex,
		header:        header,
		contentBorder: components.NewContentBorder(),
		commandBar:    components.NewCommandBar(),
		statusBar:     components.NewStatusBar().WithBinaryName(filepath.Base(cfg.TerraformBinary())),
		homeView:      views.NewHomeView(registry.MenuItems()),
	}
}

func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{a.loadWorkspace}

	// Initialize all plugins
	ctx := &plugin.Context{
		WorkingDir: a.cfg.WorkingDir(),
		Workspace:  "default",
		Service:    a.svc,
		Logger:     logging.Logger(),
		Session:    a.session,
	}
	for _, p := range a.registry.All() {
		if cmd := p.Init(ctx); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Defer context picker to Update() where mutations persist
	cmds = append(cmds, func() tea.Msg { return openContextOnStartupMsg{} })

	return tea.Batch(cmds...)
}

// --- Messages ---

type workspaceLoadedMsg struct {
	workspace string
}

type openContextOnStartupMsg struct{}

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
		a.header = components.NewHeader(a.cfg.Dir, msg.workspace)
		return a, nil

	case openContextOnStartupMsg:
		// On startup, activate the scope plugin directly for scope selection
		if p, ok := a.registry.ByID("scope"); ok {
			a.activePlugin = p
			return a, a.activatePlugin(p)
		}
		return a, nil

	case sdk.OverlayDismissMsg:
		a.activeOverlay = nil
		a.syncActiveScope()
		return a, nil

	case sdk.DeactivateMsg:
		if a.activePlugin != nil {
			prev := a.activePlugin.ID()
			a.activePlugin = nil
			logging.Logger().Debug("view.transition", "from", prev, "to", "home")
		}
		a.syncActiveScope()
		return a, nil

	case tfuicontext.NavigateToMsg:
		if p, ok := a.registry.ByID(msg.PluginID); ok {
			a.activePlugin = p
			logging.Logger().Debug("view.transition", "to", msg.PluginID)
			return a, a.activatePlugin(p)
		}
		return a, nil

	case sdk.RequestInputMsg:
		a.inputActive = true
		a.inputPrompt = msg.Request.Prompt
		a.inputAnswer = msg.Request.Default
		a.inputCallback = msg.Request.Callback
		return a, nil

	case tfuistate.StateEditMsg:
		if a.sourceIndex != nil {
			if loc, ok := a.sourceIndex.Lookup(msg.Address); ok {
				return a, editor.Open(loc)
			}
		}
		return a, nil

	case editor.EditorClosedMsg:
		if msg.Modified {
			// Invalidate plan cache since file was edited
			if a.session != nil {
				a.session.Set("plan.invalidated", true)
			}
		}
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	}

	// If an overlay is active, route messages to it
	if a.activeOverlay != nil {
		updated, cmd := a.activeOverlay.Update(msg)
		if updated == nil {
			a.activeOverlay = nil
			a.syncActiveScope()
		} else {
			a.activeOverlay = updated
		}
		return a, cmd
	}

	// If a plugin is active, delegate the message to it
	if a.activePlugin != nil {
		updated, cmd := a.activePlugin.Update(msg)
		a.activePlugin = updated
		a.syncActiveScope()
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

	// Overlay captures all input when active
	if a.activeOverlay != nil {
		updated, cmd := a.activeOverlay.Update(msg)
		if updated == nil {
			a.activeOverlay = nil
			a.syncActiveScope()
		} else {
			a.activeOverlay = updated
		}
		return a, cmd
	}

	// Input prompt mode
	if a.inputActive {
		switch msg.String() {
		case "y":
			if a.inputCallback != nil {
				cmd := a.inputCallback("y")
				a.inputActive = false
				a.inputPrompt = ""
				a.inputAnswer = ""
				a.inputCallback = nil
				return a, cmd
			}
		case "n", "esc":
			a.inputActive = false
			a.inputPrompt = ""
			a.inputAnswer = ""
			a.inputCallback = nil
		case "enter":
			if a.inputCallback != nil {
				cmd := a.inputCallback(a.inputAnswer)
				a.inputActive = false
				a.inputPrompt = ""
				a.inputAnswer = ""
				a.inputCallback = nil
				return a, cmd
			}
		case "backspace", "ctrl+h":
			if len(a.inputAnswer) > 0 {
				a.inputAnswer = a.inputAnswer[:len(a.inputAnswer)-1]
			}
		default:
			if len(msg.String()) == 1 && msg.String() >= " " {
				a.inputAnswer += msg.String()
			}
		}
		return a, nil
	}

	// Command input mode
	if a.commandMode {
		switch msg.String() {
		case "esc":
			a.commandMode = false
			a.commandInput = ""
		case "enter":
			a.commandMode = false
			input := a.commandInput
			// Auto-complete: if input matches a single plugin, use it
			if match := a.bestCommandMatch(input); match != "" {
				input = match
			}
			cmd := a.executeCommand(input)
			a.commandInput = ""
			return a, cmd
		case "tab":
			if match := a.bestCommandMatch(a.commandInput); match != "" {
				a.commandInput = match
			}
		case "backspace", "ctrl+h", "delete":
			if len(a.commandInput) > 0 {
				a.commandInput = a.commandInput[:len(a.commandInput)-1]
			} else {
				a.commandMode = false
			}
		default:
			if len(msg.String()) == 1 && msg.String() >= " " {
				a.commandInput += msg.String()
			}
		}
		return a, nil
	}

	// Global keys
	switch msg.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "C":
		if p, ok := a.registry.ByID("context"); ok {
			a.activePlugin = p
			return a, a.activatePlugin(p)
		}
		return a, nil
	case "ctrl+s":
		logging.Logger().Info("screen.capture", "content", a.View())
		return a, nil
	case ":":
		a.commandMode = true
		a.commandInput = ""
		return a, nil
	case "q":
		if a.activePlugin != nil {
			// For stackable plugins, clear sub-frames first before deactivating
			if stackable, ok := a.activePlugin.(sdk.Stackable); ok {
				if stackable.Stack().Depth() > 1 {
					stackable.Stack().Clear()
					return a, nil
				}
			}
			prev := a.activePlugin.ID()
			a.activePlugin = nil
			logging.Logger().Debug("view.transition", "from", prev, "to", "home")
			return a, nil
		}
		return a, tea.Quit
	}

	// If a plugin is active, delegate key to it
	if a.activePlugin != nil {
		// For stackable plugins, route keys through the navigation stack
		if stackable, ok := a.activePlugin.(sdk.Stackable); ok {
			cmd := stackable.Stack().Update(msg)
			a.syncActiveScope()
			return a, cmd
		}
		updated, cmd := a.activePlugin.Update(msg)
		a.activePlugin = updated
		a.syncActiveScope()
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

// syncActiveScope checks if the active scope changed in session and updates the header.
func (a *App) syncActiveScope() {
	if a.session == nil {
		return
	}
	if scope, ok := sdk.GetTyped[string](a.session, sdk.SessionKeyActiveScope); ok {
		if scope != a.activeScope {
			a.activeScope = scope
			a.header = a.header.WithScope(scope)
		}
	}
}

func (a App) activatePlugin(p sdk.Plugin) tea.Cmd {
	if activatable, ok := p.(sdk.Activatable); ok {
		return activatable.Activate()
	}
	return nil
}

func (a *App) executeCommand(input string) tea.Cmd {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	lower := strings.ToLower(input)
	for _, p := range a.registry.All() {
		if strings.ToLower(p.ID()) == lower || strings.HasPrefix(strings.ToLower(p.Name()), lower) {
			prev := ""
			if a.activePlugin != nil {
				prev = a.activePlugin.ID()
			}
			a.activePlugin = p
			logging.Logger().Debug("plugin.activate", "id", p.ID())
			if prev != "" {
				logging.Logger().Debug("view.transition", "from", prev, "to", p.ID())
			} else {
				logging.Logger().Debug("view.transition", "from", "home", "to", p.ID())
			}
			return a.activatePlugin(p)
		}
	}
	return nil
}

func (a App) bestCommandMatch(input string) string {
	if input == "" {
		return ""
	}
	lower := strings.ToLower(input)
	var match string
	count := 0
	for _, p := range a.registry.All() {
		id := strings.ToLower(p.ID())
		name := strings.ToLower(p.Name())
		if strings.HasPrefix(id, lower) || strings.HasPrefix(name, lower) {
			match = p.ID()
			count++
		}
	}
	if count == 1 {
		return match
	}
	return ""
}

func (a App) commandMatches() []string {
	if a.commandInput == "" {
		var all []string
		for _, p := range a.registry.All() {
			all = append(all, p.ID())
		}
		return all
	}
	lower := strings.ToLower(a.commandInput)
	var matches []string
	for _, p := range a.registry.All() {
		id := strings.ToLower(p.ID())
		name := strings.ToLower(p.Name())
		if strings.HasPrefix(id, lower) || strings.HasPrefix(name, lower) {
			matches = append(matches, p.ID())
		}
	}
	return matches
}

func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	headerHeight := 3
	footerHeight := 1
	borderChrome := 2
	commandBarHeight := 0
	if a.commandMode {
		commandBarHeight = 3
	}
	contentHeight := a.height - headerHeight - commandBarHeight - borderChrome - footerHeight

	title := "Home"
	filtered, total, pinned := 0, 0, 0
	if a.activePlugin != nil {
		title = a.activePlugin.Name()
		if c, ok := a.activePlugin.(sdk.Countable); ok {
			filtered, total = c.Count()
		}
		if p, ok := a.activePlugin.(sdk.Pinnable); ok {
			pinned = p.PinnedCount()
		}
	}

	var content string
	innerWidth := a.width - 2
	if a.activePlugin != nil {
		content = a.activePlugin.View(innerWidth, contentHeight)
	} else {
		content = a.homeView.Render(innerWidth, contentHeight)
	}

	header := a.header.Render(a.width)
	bordered := a.contentBorder.Render(content, title, filtered, total, pinned, a.width, contentHeight+borderChrome)

	var statusBar string
	if a.inputActive {
		promptStyle := lipgloss.NewStyle().
			Background(sdk.ColorBg).
			Foreground(sdk.ColorText).
			Bold(true).
			Padding(0, 1).
			Width(a.width)
		statusBar = promptStyle.Render(a.inputPrompt + " " + a.inputAnswer + "█")
	} else if a.activePlugin != nil {
		if stackable, ok := a.activePlugin.(sdk.Stackable); ok {
			if hints := stackable.Stack().Hints(); hints != nil {
				statusBar = a.statusBar.RenderHints(hints, a.width)
			} else {
				statusBar = a.statusBar.Render(a.width)
			}
		} else if hintable, ok := a.activePlugin.(sdk.Hintable); ok {
			if hints := hintable.Hints(); hints != nil {
				statusBar = a.statusBar.RenderHints(hints, a.width)
			} else {
				statusBar = a.statusBar.Render(a.width)
			}
		} else {
			statusBar = a.statusBar.Render(a.width)
		}
	} else {
		statusBar = a.statusBar.Render(a.width)
	}

	if a.activeOverlay != nil {
		overlayContent := a.activeOverlay.View(a.width, a.height)
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(sdk.ColorPrimary).
			Padding(1, 2).
			Width(a.width / 2).
			MaxHeight(a.height - 4)
		box := boxStyle.Render(overlayContent)
		overlayView := lipgloss.Place(a.width, a.height-1, lipgloss.Center, lipgloss.Center, box)
		if hints := a.activeOverlay.Hints(); hints != nil {
			statusBar = a.statusBar.RenderHints(hints, a.width)
		}
		return overlayView + "\n" + statusBar
	}

	var parts []string
	parts = append(parts, header)
	if a.commandMode {
		parts = append(parts, a.commandBar.Render(a.commandInput, a.commandMatches(), a.width))
	}
	parts = append(parts, bordered)
	parts = append(parts, statusBar)
	return strings.Join(parts, "\n")
}
