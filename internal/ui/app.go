package ui

import (
	"context"
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

	header    components.Header
	separator components.Separator
	statusBar components.StatusBar
	homeView  views.HomeView

	activePlugin  sdk.Plugin // nil = home screen
	activeOverlay sdk.Overlay
	activeContext string // tracks last known active context for header updates
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
	header := components.NewHeader(workDir, "default", cfg.TerraformBinary(), 0)
	if cfg.BaseDir != "" {
		header = header.WithContext(cfg.BaseDir)
	}
	return App{
		cfg:         cfg,
		svc:         svc,
		registry:    registry,
		session:     sdk.NewSession(),
		sourceIndex: sourceIndex,
		header:      header,
		separator:   components.NewSeparator(),
		statusBar:   components.NewStatusBar(),
		homeView:    views.NewHomeView(registry.All()),
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
		a.header = components.NewHeader(a.cfg.Dir, msg.workspace, a.cfg.TerraformBinary(), 0)
		return a, nil

	case sdk.OverlayDismissMsg:
		a.activeOverlay = nil
		a.syncActiveContext()
		return a, nil

	case sdk.DeactivateMsg:
		if a.activePlugin != nil {
			prev := a.activePlugin.ID()
			a.activePlugin = nil
			logging.Logger().Debug("view.transition", "from", prev, "to", "home")
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
			a.syncActiveContext()
		} else {
			a.activeOverlay = updated
		}
		return a, cmd
	}

	// If a plugin is active, delegate the message to it
	if a.activePlugin != nil {
		updated, cmd := a.activePlugin.Update(msg)
		a.activePlugin = updated
		a.syncActiveContext()
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
			a.syncActiveContext()
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
			a.syncActiveContext()
			return a, cmd
		}
		updated, cmd := a.activePlugin.Update(msg)
		a.activePlugin = updated
		a.syncActiveContext()
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
			if p.ID() == "context" {
				return a, a.openContextOverlay()
			}
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

// syncActiveContext checks if the active context changed in session and updates the header.
func (a *App) syncActiveContext() {
	if a.session == nil {
		return
	}
	if ctx, ok := sdk.GetTyped[string](a.session, sdk.SessionKeyActiveContext); ok {
		if ctx != a.activeContext {
			a.activeContext = ctx
			a.header = a.header.WithContext(ctx)
		}
	}
}

func (a *App) openContextOverlay() tea.Cmd {
	for _, p := range a.registry.All() {
		if p.ID() == "context" {
			if cp, ok := p.(*tfuicontext.Plugin); ok {
				overlay := tfuicontext.NewOverlay(cp)
				a.activeOverlay = overlay
				return overlay.Open()
			}
		}
	}
	return nil
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
			if p.ID() == "context" {
				return a.openContextOverlay()
			}
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
	separatorHeight := 2 // two separators (1 line each)
	statusBarHeight := 1
	contentHeight := a.height - headerHeight - separatorHeight - statusBarHeight

	h := a.header
	var content string
	if a.activePlugin != nil {
		h = h.WithActiveView(a.activePlugin.Name())
		content = a.activePlugin.View(a.width, contentHeight)
	} else {
		content = a.homeView.Render(a.width, contentHeight)
	}
	header := h.Render(a.width)

	contentStyle := lipgloss.NewStyle().
		Width(a.width).
		Height(contentHeight)
	content = contentStyle.Render(content)

	var statusBar string
	if a.inputActive {
		promptStyle := lipgloss.NewStyle().
			Background(sdk.ColorBg).
			Foreground(sdk.ColorText).
			Bold(true).
			Padding(0, 1).
			Width(a.width)
		statusBar = promptStyle.Render(a.inputPrompt + " " + a.inputAnswer + "█")
	} else if a.commandMode {
		cmdStyle := lipgloss.NewStyle().
			Background(sdk.ColorBg).
			Foreground(sdk.ColorText).
			Bold(true).
			Padding(0, 1).
			Width(a.width)
		matches := a.commandMatches()
		hint := ""
		if len(matches) > 0 {
			hint = "  " + sdk.StyleFaint.Render(strings.Join(matches, " | "))
		}
		statusBar = cmdStyle.Render(":" + a.commandInput + "█" + hint)
	} else if a.activePlugin != nil {
		if stackable, ok := a.activePlugin.(sdk.Stackable); ok {
			if hints := stackable.Stack().Hints(); hints != nil {
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

	sep := a.separator.Render(a.width)

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

	return header + "\n" + sep + "\n" + content + "\n" + sep + "\n" + statusBar
}
