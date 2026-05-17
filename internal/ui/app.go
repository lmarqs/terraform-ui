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
	sdkui "github.com/lmarqs/terraform-ui/pkg/sdk/ui"
	tfuiapply "github.com/lmarqs/terraform-ui/plugins/apply"
	tfuiimport "github.com/lmarqs/terraform-ui/plugins/import"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	tfuistate "github.com/lmarqs/terraform-ui/plugins/state"
	tfuitaint "github.com/lmarqs/terraform-ui/plugins/taint"
	tfuiuntaint "github.com/lmarqs/terraform-ui/plugins/untaint"
)

// StandaloneConfig configures the app to run a single plugin without
// home screen or inter-plugin navigation (the fzf model).
type StandaloneConfig struct {
	PluginID string   // which plugin is the app
	Args     []string // positional args (e.g., "mv", "src", "dst")
	JSONMode bool     // user passed -json
}

type App struct {
	cfg         config.Config
	svc         sdk.Service
	registry    *plugin.Registry
	pins        *sdk.PinService
	options     *sdk.ResolvedOptions
	bus         *sdk.EventBus
	sourceIndex *terraform.SourceIndex
	rootCfg     *config.RootConfig
	childCfg    *config.ChildConfig
	standalone  *StandaloneConfig
	width       int
	height      int

	header        components.Header
	contentBorder components.ContentBorder
	commandBar    components.CommandBar
	statusBar     components.StatusBar
	homeView      views.HomeView

	activePlugin    sdk.Plugin   // nil = home screen
	navStack        []sdk.Plugin // LIFO stack of return destinations; empty = no history
	activeOverlay   sdk.Overlay
	activeChdir     string // tracks last known active chdir for header updates
	activeWorkspace string // tracks current workspace for config re-resolution
	lockInfo        *sdk.StateLock
	staleState      bool
	commandMode     bool
	commandInput    string
	commandError    string

	inputActive   bool
	inputMode     sdk.InputRequestMode
	inputPrompt   string
	inputAnswer   string
	inputCallback func(string) tea.Cmd
}

func NewApp(cfg config.Config, svc sdk.Service, registry *plugin.Registry, rootCfg *config.RootConfig, standalone ...*StandaloneConfig) App {
	workDir := cfg.WorkingDir()
	sourceIndex, _ := terraform.NewSourceIndex(workDir)
	header := components.NewHeader(workDir, "default")
	if cfg.Chdir != "" {
		header = header.WithChdir(cfg.Chdir)
	} else if cfg.BaseDir != "" {
		header = header.WithChdir(cfg.BaseDir)
	}

	pins := sdk.NewPinService()
	opts := &sdk.ResolvedOptions{
		VarFiles:  cfg.VarFiles,
		Vars:      cfg.Vars,
		ExtraArgs: cfg.ExtraArgs,
	}

	bus := sdk.NewEventBus(registry.All())

	var childCfg *config.ChildConfig
	if rootCfg != nil {
		childCfg, _ = config.LoadChild(workDir)
	}

	var sc *StandaloneConfig
	if len(standalone) > 0 && standalone[0] != nil {
		sc = standalone[0]
	}

	return App{
		cfg:           cfg,
		svc:           svc,
		registry:      registry,
		pins:          pins,
		options:       opts,
		bus:           bus,
		sourceIndex:   sourceIndex,
		rootCfg:       rootCfg,
		childCfg:      childCfg,
		standalone:    sc,
		header:        header,
		contentBorder: components.NewContentBorder(),
		commandBar:    components.NewCommandBar(),
		statusBar:     components.NewStatusBar().WithBinaryName(filepath.Base(cfg.TerraformBinary())),
		homeView:      views.NewHomeView(registry.MenuItems()),
	}
}

// ActivePlugin returns the currently active plugin (for output extraction after TUI exit).
func (a App) ActivePlugin() sdk.Plugin {
	return a.activePlugin
}

// IsStandalone reports whether the app is running in standalone mode.
func (a App) IsStandalone() bool {
	return a.standalone != nil
}

func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{a.loadWorkspace}

	// Initialize all plugins
	ctx := &plugin.Context{
		WorkingDir: a.cfg.WorkingDir(),
		Workspace:  "default",
		Service:    a.svc,
		Logger:     logging.Logger(),
		Pins:       a.pins,
		Options:    a.options,
	}

	// If scope was pre-configured, scope the service for plugins
	if a.cfg.Chdir != "" {
		absScope := filepath.Join(a.cfg.Dir, a.cfg.Chdir)
		ctx.Service = a.svc.WithDir(absScope)
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
		a.activeWorkspace = msg.workspace
		a.resolveOptions(msg.workspace)
		a.header = a.header.WithWorkspace(msg.workspace)
		return a, nil

	case openContextOnStartupMsg:
		// Standalone mode: activate the target plugin directly
		if a.standalone != nil {
			if p, ok := a.registry.ByID(a.standalone.PluginID); ok {
				a.activePlugin = p
				var cmd tea.Cmd
				if activator, ok := p.(sdk.ActivateWithArgs); ok && len(a.standalone.Args) > 0 {
					cmd = activator.ActivateWithArgs(a.standalone.Args)
				} else if activatable, ok := p.(sdk.Activatable); ok {
					cmd = activatable.Activate()
				}
				return a, cmd
			}
			return a, nil
		}
		// Skip chdir picker if scope is pre-set or data is loaded externally
		if a.cfg.Chdir != "" || a.cfg.PreloadedData {
			if a.cfg.Chdir != "" {
				absScope := filepath.Join(a.cfg.Dir, a.cfg.Chdir)
				a.activeChdir = a.cfg.Chdir
				return a, a.bus.Dispatch(sdk.ChdirChangedEvent{
					RelPath: a.cfg.Chdir,
					AbsPath: absScope,
				})
			}
			return a, nil
		}
		// On startup, activate the chdir plugin directly for member selection
		if p, ok := a.registry.ByID("chdir"); ok {
			return a, a.navigateTo(p)
		}
		return a, nil

	case sdk.ChdirChangedEvent:
		a.activeChdir = msg.RelPath
		a.lockInfo = nil
		a.staleState = false
		if a.rootCfg != nil {
			childCfg, err := config.LoadChild(msg.AbsPath)
			if err != nil {
				logging.Logger().Debug("config.load_child", "dir", msg.AbsPath, "err", err)
			}
			a.childCfg = childCfg
		}
		a.resolveOptions(a.activeWorkspace)
		a.header = a.header.WithChdir(msg.RelPath).WithLockInfo(nil).WithStale(false)
		return a, a.popIfPushed(a.bus.Dispatch(msg))

	case sdk.PlanCompletedEvent:
		return a, a.bus.Dispatch(msg)

	case sdk.WorkspaceCreatedEvent:
		a.activeWorkspace = msg.Name
		a.resolveOptions(msg.Name)
		a.header = a.header.WithWorkspace(msg.Name)
		return a, a.bus.Dispatch(sdk.WorkspaceChangedEvent(msg))

	case sdk.WorkspaceChangedEvent:
		a.activeWorkspace = msg.Name
		a.lockInfo = nil
		a.staleState = false
		a.resolveOptions(msg.Name)
		a.header = a.header.WithWorkspace(msg.Name).WithLockInfo(nil).WithStale(false)
		return a, a.popIfPushed(a.bus.Dispatch(msg))

	case sdk.PlanInvalidatedEvent:
		a.staleState = true
		a.header = a.header.WithStale(true)
		return a, a.bus.Dispatch(msg)

	case sdk.LockDetectedEvent:
		a.lockInfo = msg.Lock
		a.header = a.header.WithLockInfo(msg.Lock)
		return a, a.bus.Dispatch(msg)

	case sdk.LockClearedEvent:
		a.lockInfo = nil
		a.header = a.header.WithLockInfo(nil)
		return a, a.bus.Dispatch(msg)

	case sdk.StateRefreshedEvent:
		a.staleState = false
		a.header = a.header.WithStale(false)
		return a, a.bus.Dispatch(msg)

	case sdk.OverlayDismissMsg:
		a.activeOverlay = nil
		return a, nil

	case sdk.NavigateMsg:
		if p, ok := a.registry.ByID(msg.PluginID); ok {
			// Standalone mode: only allow NavPush navigation (sub-states)
			if a.standalone != nil && a.registry.NavBehaviorFor(msg.PluginID) != plugin.NavPush {
				return a, nil
			}
			return a, a.navigateTo(p)
		}
		return a, nil

	case sdk.DeactivateMsg:
		if a.activePlugin != nil {
			if c, ok := a.activePlugin.(sdk.Cancellable); ok {
				c.Cancel()
			}
			if len(a.navStack) > 0 {
				a.navigateBack()
				return a, a.activate(a.activePlugin)
			}
			// Standalone mode: quit when root plugin deactivates
			if a.standalone != nil {
				return a, tea.Quit
			}
			prev := a.activePlugin.ID()
			a.activePlugin = nil
			logging.Logger().Debug("view.transition", "from", prev, "to", "home")
		}
		return a, nil

	case sdk.RequestInputMsg:
		a.inputActive = true
		a.inputMode = msg.Request.Mode
		a.inputPrompt = msg.Request.Prompt
		a.inputAnswer = msg.Request.Default
		a.inputCallback = msg.Request.Callback
		return a, nil

	case tfuistate.StateEditMsg:
		if a.sourceIndex == nil {
			return a, nil
		}
		if len(msg.Addresses) > 0 {
			var locs []editor.SourceLocation
			for _, addr := range msg.Addresses {
				if loc, ok := a.sourceIndex.Lookup(addr); ok {
					locs = append(locs, loc)
				}
			}
			if len(locs) > 0 {
				logging.Logger().Debug("editor.open.multiple", "count", len(locs))
				return a, editor.OpenMultiple(locs)
			}
			logging.Logger().Debug("editor.lookup.failed", "addresses", msg.Addresses)
			return a, nil
		}
		if loc, ok := a.sourceIndex.Lookup(msg.Address); ok {
			logging.Logger().Debug("editor.open", "address", msg.Address, "file", loc.File, "line", loc.Line)
			return a, editor.Open(loc)
		}
		logging.Logger().Debug("editor.lookup.failed", "address", msg.Address)
		return a, nil

	case tfuiplan.PlanEditMsg:
		if a.sourceIndex == nil {
			return a, nil
		}
		if loc, ok := a.sourceIndex.Lookup(msg.Address); ok {
			logging.Logger().Debug("editor.open", "address", msg.Address, "file", loc.File, "line", loc.Line)
			return a, editor.Open(loc)
		}
		logging.Logger().Debug("editor.lookup.failed", "address", msg.Address)
		return a, nil

	case editor.EditorClosedMsg:
		if msg.Modified {
			return a, func() tea.Msg { return sdk.PlanInvalidatedEvent{} }
		}
		return a, nil

	case tfuitaint.TaintRequestMsg:
		if p, ok := a.registry.ByID("taint"); ok {
			taintPlugin := p.(*tfuitaint.Plugin)
			taintPlugin.SetTargets(msg.Addresses)
			a.navStack = append(a.navStack, a.activePlugin)
			a.activePlugin = p
			logging.Logger().Debug("view.transition", "from", activeViewID(a.navStack), "to", "taint", "targets", len(msg.Addresses))
			return a, a.activate(p)
		}
		return a, nil

	case tfuiuntaint.UntaintRequestMsg:
		if p, ok := a.registry.ByID("untaint"); ok {
			untaintPlugin := p.(*tfuiuntaint.Plugin)
			untaintPlugin.SetTargets(msg.Addresses)
			a.navStack = append(a.navStack, a.activePlugin)
			a.activePlugin = p
			logging.Logger().Debug("view.transition", "from", activeViewID(a.navStack), "to", "untaint", "targets", len(msg.Addresses))
			return a, a.activate(p)
		}
		return a, nil

	case tfuiimport.ImportRequestMsg:
		if p, ok := a.registry.ByID("import"); ok {
			importPlugin := p.(*tfuiimport.Plugin)
			importPlugin.SetAddress(msg.Address)
			a.navStack = append(a.navStack, a.activePlugin)
			a.activePlugin = p
			logging.Logger().Debug("view.transition", "from", activeViewID(a.navStack), "to", "import")
			return a, a.activate(p)
		}
		return a, nil

	case tfuiplan.ApplyRequestMsg:
		if p, ok := a.registry.ByID("apply"); ok {
			applyPlugin := p.(*tfuiapply.Plugin)
			if pinned := a.pins.All(); len(pinned) > 0 {
				applyPlugin.SetTargets(pinned)
			}
			a.navStack = append(a.navStack, a.activePlugin)
			a.activePlugin = p
			cmd := applyPlugin.RequestApply()
			logging.Logger().Debug("view.transition", "from", "plan", "to", "apply", "targets", len(applyPlugin.Targets()))
			return a, cmd
		}
		return a, nil

	case tfuiplan.AutoApplyRequestMsg:
		if p, ok := a.registry.ByID("apply"); ok {
			applyPlugin := p.(*tfuiapply.Plugin)
			if pinned := a.pins.All(); len(pinned) > 0 {
				applyPlugin.SetTargets(pinned)
			}
			a.navStack = append(a.navStack, a.activePlugin)
			a.activePlugin = p
			cmd := applyPlugin.AutoApply()
			logging.Logger().Debug("view.transition", "from", "plan", "to", "apply", "auto_approve", true, "targets", len(applyPlugin.Targets()))
			return a, cmd
		}
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	}

	// Fan-out: broadcast events to all subscribing plugins
	if _, ok := msg.(sdk.Event); ok {
		busCmd := a.bus.Dispatch(msg)
		return a, busCmd
	}

	// If an overlay is active, route messages to it
	if a.activeOverlay != nil {
		updated, cmd := a.activeOverlay.Update(msg)
		if updated == nil {
			a.activeOverlay = nil
		} else {
			a.activeOverlay = updated
		}
		return a, cmd
	}

	// Timer ticks route only to the active plugin (ticks are unscoped —
	// broadcasting would cause exponential growth when multiple timers run).
	// Inactive timers resume via Activate() on re-entry.
	if _, ok := msg.(sdkui.TimerTickMsg); ok {
		if a.activePlugin != nil {
			updated, cmd := a.activePlugin.Update(msg)
			a.activePlugin = updated
			return a, cmd
		}
		return a, nil
	}

	// Broadcast result messages to all plugins — each handles only its own types
	var cmds []tea.Cmd
	for _, p := range a.registry.All() {
		updated, cmd := p.Update(msg)
		if p == a.activePlugin {
			a.activePlugin = updated
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return a, tea.Batch(cmds...)
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	a.commandError = ""

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
		} else {
			a.activeOverlay = updated
		}
		return a, cmd
	}

	// Input prompt mode
	if a.inputActive {
		switch a.inputMode {
		case sdk.InputRequestBool:
			// Confirmation mode: only handle y/n/esc
			switch msg.String() {
			case "y":
				if a.inputCallback != nil {
					cmd := a.inputCallback("y")
					a.inputActive = false
					a.inputMode = 0
					a.inputPrompt = ""
					a.inputAnswer = ""
					a.inputCallback = nil
					return a, cmd
				}
			case "n", "esc":
				a.inputActive = false
				a.inputMode = 0
				a.inputPrompt = ""
				a.inputAnswer = ""
				a.inputCallback = nil
			}
		default:
			// Text/select mode: enter submits, esc cancels, everything else is input
			switch msg.String() {
			case "esc":
				a.inputActive = false
				a.inputMode = 0
				a.inputPrompt = ""
				a.inputAnswer = ""
				a.inputCallback = nil
			case "enter":
				if a.inputCallback != nil {
					cmd := a.inputCallback(a.inputAnswer)
					a.inputActive = false
					a.inputMode = 0
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

	// Key capture: plugin wants exclusive keyboard input
	if a.activePlugin != nil {
		if capturer, ok := a.activePlugin.(sdk.KeyCapturer); ok && capturer.CapturesKeys() {
			switch msg.String() {
			case "ctrl+c":
				return a, tea.Quit
			case "ctrl+s":
				logging.Logger().Info("screen.capture", "content", a.View())
				return a, nil
			default:
				updated, cmd := a.activePlugin.Update(msg)
				a.activePlugin = updated
				return a, cmd
			}
		}
	}

	// Global keys
	switch msg.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "C":
		if a.standalone != nil {
			break
		}
		if p, ok := a.registry.ByID("context"); ok {
			return a, a.navigateTo(p)
		}
		return a, nil
	case "ctrl+s":
		logging.Logger().Info("screen.capture", "content", a.View())
		return a, nil
	case ":":
		if a.standalone != nil {
			break
		}
		a.commandMode = true
		a.commandInput = ""
		return a, nil
	case "q":
		// Standalone mode: q always quits (after clearing sub-frames)
		if a.standalone != nil {
			if a.activePlugin != nil {
				if stackable, ok := a.activePlugin.(sdk.Stackable); ok {
					if stackable.Stack().Depth() > 1 {
						stackable.Stack().Clear()
						return a, nil
					}
				}
			}
			return a, a.cmdQuit()
		}
		if a.activePlugin != nil {
			// For stackable plugins, clear sub-frames first before deactivating
			if stackable, ok := a.activePlugin.(sdk.Stackable); ok {
				if stackable.Stack().Depth() > 1 {
					stackable.Stack().Clear()
					return a, nil
				}
			}
			if busy, ok := a.activePlugin.(sdk.Busy); ok && busy.Busy() {
				// Don't cancel plugins holding a terraform lock
			} else if c, ok := a.activePlugin.(sdk.Cancellable); ok {
				c.Cancel()
			}
			prev := a.activePlugin.ID()
			a.activePlugin = nil
			a.navStack = nil
			logging.Logger().Debug("view.transition", "from", prev, "to", "home")
			return a, nil
		}
		return a, a.cmdQuit()
	}

	// If a plugin is active, delegate key to it
	if a.activePlugin != nil {
		// For stackable plugins, route keys through the navigation stack
		if stackable, ok := a.activePlugin.(sdk.Stackable); ok {
			cmd := stackable.Stack().Update(msg)
			if stackable.Stack().IsEmpty() {
				if len(a.navStack) > 0 {
					a.navigateBack()
					return a, tea.Batch(cmd, a.activate(a.activePlugin))
				}
				prev := a.activePlugin.ID()
				a.activePlugin = nil
				a.navStack = nil
				logging.Logger().Debug("view.transition", "from", prev, "to", "home")
			}
			return a, cmd
		}
		updated, cmd := a.activePlugin.Update(msg)
		a.activePlugin = updated
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
			return a, a.navigateTo(p)
		}
		return a, nil
	default:
		// Check if key matches a plugin binding
		if p, ok := a.registry.ByKey(msg.String()); ok {
			return a, a.navigateTo(p)
		}
	}
	return a, nil
}

func (a *App) navigateTo(p sdk.Plugin) tea.Cmd {
	nav := a.registry.NavBehaviorFor(p.ID())
	from := "home"
	if a.activePlugin != nil {
		from = a.activePlugin.ID()
		if busy, ok := a.activePlugin.(sdk.Busy); ok && busy.Busy() {
			// Don't cancel plugins holding a terraform lock
		} else if c, ok := a.activePlugin.(sdk.Cancellable); ok {
			c.Cancel()
		}
	}
	switch nav {
	case plugin.NavPush:
		a.navStack = append(a.navStack, a.activePlugin)
	default:
		a.navStack = nil
	}
	a.activePlugin = p
	logging.Logger().Debug("plugin.activate", "id", p.ID())
	logging.Logger().Debug("view.transition", "from", from, "to", p.ID())
	return a.activate(p)
}

func (a *App) navigateBack() {
	from := ""
	if a.activePlugin != nil {
		from = a.activePlugin.ID()
	}
	if len(a.navStack) == 0 {
		a.activePlugin = nil
		logging.Logger().Debug("view.transition", "from", from, "to", "home")
		return
	}
	prev := a.navStack[len(a.navStack)-1]
	a.navStack = a.navStack[:len(a.navStack)-1]
	a.activePlugin = prev
	to := "home"
	if prev != nil {
		to = prev.ID()
	}
	logging.Logger().Debug("view.transition", "from", from, "to", to)
}

// resolveOptions re-runs config resolution and updates the shared ResolvedOptions pointer.
// ExtraArgs is intentionally preserved — CLI passthrough (--) takes precedence over config.
func (a *App) resolveOptions(workspace string) {
	if a.rootCfg == nil {
		return
	}
	resolved := config.Resolve(a.rootCfg, a.childCfg, workspace)
	a.options.VarFiles = resolved.VarFiles()
	a.options.Vars = resolved.Vars()
	logging.Logger().Debug("options.resolved", "workspace", workspace, "var_files", len(a.options.VarFiles), "vars", len(a.options.Vars))
}

func (a *App) popIfPushed(busCmd tea.Cmd) tea.Cmd {
	if a.activePlugin != nil && a.registry.NavBehaviorFor(a.activePlugin.ID()) == plugin.NavPush {
		if len(a.navStack) > 0 {
			a.navigateBack()
			return tea.Batch(busCmd, a.activate(a.activePlugin))
		}
		a.activePlugin = nil
	}
	return busCmd
}

func (a App) activate(p sdk.Plugin) tea.Cmd {
	if activatable, ok := p.(sdk.Activatable); ok {
		return activatable.Activate()
	}
	return nil
}

type builtinCommand struct {
	name string
	fn   func(*App) tea.Cmd
}

var builtinCommands = []builtinCommand{
	{"q", (*App).cmdQuit},
	{"q!", (*App).cmdForceQuit},
}

func (a *App) cmdQuit() tea.Cmd {
	for _, p := range a.registry.All() {
		if busy, ok := p.(sdk.Busy); ok && busy.Busy() {
			a.commandError = "Operation in progress (use :q! to force)"
			return nil
		}
	}
	return tea.Quit
}

func (a *App) cmdForceQuit() tea.Cmd {
	return tea.Quit
}

func (a *App) executeCommand(input string) tea.Cmd {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	lower := strings.ToLower(input)
	for _, cmd := range builtinCommands {
		if cmd.name == lower {
			return cmd.fn(a)
		}
	}

	for _, p := range a.registry.All() {
		if strings.ToLower(p.ID()) == lower || strings.HasPrefix(strings.ToLower(p.Name()), lower) {
			return a.navigateTo(p)
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
	for _, cmd := range builtinCommands {
		if strings.HasPrefix(cmd.name, lower) {
			match = cmd.name
			count++
		}
	}
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
		for _, cmd := range builtinCommands {
			all = append(all, cmd.name)
		}
		for _, p := range a.registry.All() {
			all = append(all, p.ID())
		}
		return all
	}
	lower := strings.ToLower(a.commandInput)
	var matches []string
	for _, cmd := range builtinCommands {
		if strings.HasPrefix(cmd.name, lower) {
			matches = append(matches, cmd.name)
		}
	}
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

	// Standalone mode: minimal chrome (single header line + hint bar)
	if a.standalone != nil {
		return a.viewStandalone()
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
	if a.commandError != "" {
		errorStyle := lipgloss.NewStyle().
			Background(sdk.ColorBg).
			Foreground(sdk.ColorDanger).
			Padding(0, 1).
			Width(a.width)
		statusBar = errorStyle.Render(a.commandError)
	} else if a.inputActive {
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
		homeHints := []sdk.KeyHint{
			{Key: "↑↓", Description: "navigate"},
			{Key: "Enter", Description: "select"},
			{Key: ":", Description: "command"},
			{Key: "q", Description: "quit"},
		}
		statusBar = a.statusBar.RenderHints(homeHints, a.width)
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

func (a App) viewStandalone() string {
	headerHeight := 1
	footerHeight := 1
	contentHeight := a.height - headerHeight - footerHeight

	// Minimal header: tfui on left, context info on right
	headerStyle := lipgloss.NewStyle().
		Background(sdk.ColorBg).
		Foreground(sdk.ColorFaint).
		Width(a.width)
	left := " tfui"
	var rightParts []string
	rightParts = append(rightParts, filepath.Base(a.cfg.WorkingDir()))
	if a.activeChdir != "" {
		rightParts = append(rightParts, a.activeChdir)
	}
	if a.activeWorkspace != "" {
		rightParts = append(rightParts, a.activeWorkspace)
	}
	if a.lockInfo != nil {
		rightParts = append(rightParts, "[locked]")
	}
	right := strings.Join(rightParts, " │ ") + " "
	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	header := headerStyle.Render(left + strings.Repeat(" ", gap) + right)

	// Content
	var content string
	if a.activePlugin != nil {
		content = a.activePlugin.View(a.width, contentHeight)
	}

	// Status bar / input prompt
	var statusBar string
	if a.commandError != "" {
		errorStyle := lipgloss.NewStyle().
			Background(sdk.ColorBg).
			Foreground(sdk.ColorDanger).
			Padding(0, 1).
			Width(a.width)
		statusBar = errorStyle.Render(a.commandError)
	} else if a.inputActive {
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
	}

	// Overlay handling
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

	return header + "\n" + content + "\n" + statusBar
}

func activeViewID(navStack []sdk.Plugin) string {
	if len(navStack) == 0 {
		return "home"
	}
	last := navStack[len(navStack)-1]
	if last == nil {
		return "home"
	}
	return last.ID()
}
