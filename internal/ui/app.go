package ui

import (
	"context"
	"fmt"
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
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
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

// contextHolder is a heap-allocated indirection so that plugin closures
// captured at Init() see the App's *current* Context after every atomic
// replacement. App is passed by value through bubbletea's Update loop, so a
// closure that captures &App's current field would observe a stale copy;
// shared indirection through a holder makes the live read trivially correct.
type contextHolder struct {
	current *sdk.Context
}

type App struct {
	cfg         config.Config
	svc         sdk.Service
	registry    *plugin.Registry
	holder      *contextHolder
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
	activeChdir     sdk.Chdir     // tracks last known active chdir for header updates
	activeWorkspace sdk.Workspace // tracks current workspace for config re-resolution
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
	header := components.NewHeader(workDir, sdk.WorkspaceDefault.String())
	if cfg.Chdir != "" {
		header = header.WithChdir(cfg.Chdir)
	} else if cfg.BaseDir != "" {
		header = header.WithChdir(cfg.BaseDir)
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
		holder:        &contextHolder{},
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

	// Seed holder with a bootstrap Context so plugins that read deps.Context()
	// during Init never see a nil pointer. The first ContextChangedEvent
	// (dispatched once the user picks a chdir / workspace) replaces it.
	bootSvc := a.svc
	bootDir := a.cfg.WorkingDir()
	if a.cfg.Chdir != "" {
		bootDir = filepath.Join(a.cfg.Dir, a.cfg.Chdir)
		bootSvc = a.svc.WithDir(bootDir)
	}
	a.holder.current = &sdk.Context{
		WorkingDir: bootDir,
		Workspace:  sdk.WorkspaceDefault,
		Service:    bootSvc,
		Pins:       a.cfg.Targets,
		ExtraArgs:  a.cfg.ExtraArgs,
	}

	holder := a.holder
	deps := &plugin.PluginDeps{
		Logger:  logging.Logger(),
		Service: bootSvc,
		Context: func() *sdk.Context { return holder.current },
		Pin: func(address string) tea.Cmd {
			return func() tea.Msg { return sdk.PinToggleRequestMsg{Address: address} }
		},
		ClearPins: func() tea.Cmd {
			return func() tea.Msg { return sdk.PinClearRequestMsg{} }
		},
	}

	for _, p := range a.registry.All() {
		if cmd := p.Init(deps); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Defer context picker to Update() where mutations persist
	cmds = append(cmds, func() tea.Msg { return openContextOnStartupMsg{} })

	return tea.Batch(cmds...)
}

// --- Messages ---

type workspaceLoadedMsg struct {
	workspace sdk.Workspace
}

type openContextOnStartupMsg struct{}

// --- Async commands ---

func (a App) loadWorkspace() tea.Msg {
	ws, err := a.svc.Workspace(context.Background())
	if err != nil {
		return workspaceLoadedMsg{workspace: sdk.WorkspaceDefault}
	}
	return workspaceLoadedMsg{workspace: sdk.NewWorkspace(ws)}
}

// --- Update ---

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case workspaceLoadedMsg:
		a.activeWorkspace = msg.workspace
		a.header = a.header.WithWorkspace(msg.workspace.String())
		return a, nil

	case openContextOnStartupMsg:
		// Standalone mode: activate the target plugin directly
		if a.standalone != nil {
			if a.cfg.Chdir != "" {
				a.activeChdir = sdk.Chdir(a.cfg.Chdir)
			} else if a.cfg.BaseDir != "" {
				a.activeChdir = sdk.Chdir(a.cfg.BaseDir)
			}
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
				return a, func() tea.Msg {
					return sdk.ContextSwitchRequestMsg{Chdir: sdk.Chdir(a.cfg.Chdir), Workspace: sdk.WorkspaceDefault}
				}
			}
			if a.cfg.BaseDir != "" {
				a.activeChdir = sdk.Chdir(a.cfg.BaseDir)
			}
			return a, nil
		}
		// On startup, activate the chdir plugin directly for member selection
		if p, ok := a.registry.ByID("chdir"); ok {
			return a, a.navigateTo(p)
		}
		return a, nil

	case sdk.PinToggleRequestMsg:
		if !a.requireIdle("pin") {
			return a, nil
		}
		current := a.holder.current
		if current == nil {
			return a, nil
		}
		next := current.TogglePin(msg.Address)
		return a, a.bus.Dispatch(a.replaceContext(next)())

	case sdk.PinClearRequestMsg:
		if !a.requireIdle("pin") {
			return a, nil
		}
		current := a.holder.current
		if current == nil {
			return a, nil
		}
		next := current.WithPins(nil)
		return a, a.bus.Dispatch(a.replaceContext(next)())

	case sdk.ContextSwitchRequestMsg:
		// Single chokepoint: chdir/workspace plugins emit this when the user
		// picks a member or workspace. The App rebuilds Context (atomic
		// replacement, ADR-0018) and dispatches ContextChangedEvent on the
		// bus for plugins to react to. The App owns path resolution: relative
		// chdir → absolute via filepath.Join with the project root.
		//
		// Busy-guard: ADR-0016 forbids reentrant terraform calls; switching
		// chdir/workspace while a plugin holds DirLock would either deadlock
		// (ExecService rejects) or strand the running command pointing at the
		// old context. Reject upfront with a uniform message.
		if !a.requireIdle("context-switch") {
			return a, nil
		}
		chdir := msg.Chdir
		workspace := msg.Workspace
		absChdir := filepath.Join(a.cfg.Dir, chdir.String())
		a.activeChdir = chdir
		a.activeWorkspace = workspace
		a.lockInfo = nil
		a.staleState = false
		if a.rootCfg != nil && chdir != "" {
			childCfg, err := config.LoadChild(absChdir)
			if err != nil {
				logging.Logger().Debug("config.load_child", "dir", absChdir, "err", err)
			}
			a.childCfg = childCfg
		}
		a.header = a.header.WithChdir(chdir.String()).WithWorkspace(workspace.String()).WithLockInfo(nil).WithStale(false)
		ctxCmd := a.replaceContext(a.rebuildContext(chdir, absChdir, workspace))
		return a, a.popIfPushed(a.bus.Dispatch(ctxCmd()))

	case sdk.PlanCompletedEvent:
		return a, a.bus.Dispatch(msg)

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
			applyPlugin.SetPlanFile(msg.PlanFile)
			a.navStack = append(a.navStack, a.activePlugin)
			a.activePlugin = p
			var cmd tea.Cmd
			if msg.AutoApprove {
				cmd = applyPlugin.AutoApply()
			} else {
				cmd = applyPlugin.RequestApply()
			}
			logging.Logger().Debug("view.transition", "from", "plan", "to", "apply",
				"plan", msg.PlanFile, "auto_approve", msg.AutoApprove)
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

	// Stream messages route only to the active plugin (they originate from
	// the active plugin's own channel — broadcasting leaks them to plugins
	// with nil channels, causing a deadlock in headless mode).
	switch msg.(type) {
	case frames.StreamLineMsg, frames.StreamDoneMsg:
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
			wasEmpty := stackable.Stack().IsEmpty()
			if !wasEmpty {
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
	// Busy-guard: a plugin switch may run terraform on Activate (plan, state,
	// apply via the menu, etc). While ANY plugin is Busy() the dir lock is
	// held; a fresh terraform call from a newly-activated plugin would
	// deadlock at ExecService. Reject the switch instead.
	if !a.requireIdle("navigate:" + p.ID()) {
		return nil
	}
	nav := a.registry.NavBehaviorFor(p.ID())
	from := "home"
	if a.activePlugin != nil {
		from = a.activePlugin.ID()
		if c, ok := a.activePlugin.(sdk.Cancellable); ok {
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

// rebuildContext constructs a fresh, immutable Context snapshot from the
// app's config + the supplied chdir/workspace. It populates ALL six
// terraform-affecting exec fields (var-files, vars, parallelism, lock,
// lock-timeout, extra-args) from config.Resolve — NOT just var-files/vars.
// Pinned targets are preserved across rebuilds within the same chdir; on
// chdir change callers should pass nil targets to clear them.
func (a *App) rebuildContext(chdir sdk.Chdir, absChdir string, workspace sdk.Workspace) *sdk.Context {
	scopedSvc := a.svc
	if absChdir != "" {
		scopedSvc = a.svc.WithDir(absChdir)
	}
	next := &sdk.Context{
		Chdir:      chdir,
		WorkingDir: absChdir,
		Workspace:  workspace,
		Service:    scopedSvc,
		ExtraArgs:  a.cfg.ExtraArgs,
	}
	if a.rootCfg != nil {
		resolved := config.Resolve(a.rootCfg, a.childCfg, workspace.String())
		next.VarFiles = resolved.VarFiles()
		next.Vars = resolved.Vars()
		next.Parallelism = resolved.Parallelism()
		next.Lock = sdk.LockModeFromPtr(resolved.Lock())
		next.LockTimeout = sdk.LockTimeout(resolved.LockTimeout())
	}
	logging.Logger().Debug("context.rebuilt",
		"chdir", chdir,
		"workspace", workspace,
		"var_files", len(next.VarFiles),
		"parallelism", next.Parallelism,
	)
	return next
}

// replaceContext atomically swaps the active Context and returns a Cmd that
// dispatches a single ContextChangedEvent carrying both the previous and
// next snapshots. Callers must build `next` via rebuildContext (or
// WithPins on an existing Context) — never mutate `current` in place.
//
// Every replacement is logged so every transition is observable in the debug
// log: prev/next chdir, workspace, and target counts.
func (a *App) replaceContext(next *sdk.Context) tea.Cmd {
	prev := a.holder.current
	a.holder.current = next
	logging.Logger().Debug("context.replaced",
		"prev_chdir", contextChdir(prev),
		"next_chdir", contextChdir(next),
		"prev_workspace", contextWorkspace(prev),
		"next_workspace", contextWorkspace(next),
		"prev_targets", contextTargetCount(prev),
		"next_targets", contextTargetCount(next),
	)
	return func() tea.Msg {
		return sdk.ContextChangedEvent{Prev: prev, Next: next}
	}
}

func contextChdir(c *sdk.Context) string {
	if c == nil {
		return ""
	}
	return c.WorkingDir
}

func contextWorkspace(c *sdk.Context) string {
	if c == nil {
		return ""
	}
	return c.Workspace.String()
}

func contextTargetCount(c *sdk.Context) int {
	if c == nil {
		return 0
	}
	return c.Pins.Count()
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

// requireIdle is the universal busy-guard chokepoint. Every action that
// would mutate terraform inputs, replace the active Context, or start a
// new terraform call routes through this. While ANY registered plugin is
// holding a terraform operation (DirLock per ADR-0016), the action is
// rejected and `commandError` is set to a uniform message instructing the
// user to escape via `:q!`. UI-only actions (scrolling, filtering, esc)
// bypass this on purpose — see `.claude/rules/architecture.md`.
//
// reason is a short label used in the rejection message ("plan", "chdir",
// "quit", …) so the user knows which action was refused.
func (a *App) requireIdle(reason string) bool {
	for _, p := range a.registry.All() {
		if busy, ok := p.(sdk.Busy); ok && busy.Busy() {
			a.commandError = fmt.Sprintf("%s is running — press :q! to force-quit (blocked: %s)", p.ID(), reason)
			logging.Logger().Debug("busy.guard.reject", "reason", reason, "busy_plugin", p.ID())
			return false
		}
	}
	return true
}

func (a *App) cmdQuit() tea.Cmd {
	if !a.requireIdle("quit") {
		return nil
	}
	return tea.Quit
}

func (a *App) cmdForceQuit() tea.Cmd {
	for _, p := range a.registry.All() {
		if c, ok := p.(sdk.Cancellable); ok {
			c.Cancel()
		}
	}
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
	cursor, navigable := 0, 0
	if a.activePlugin != nil {
		title = a.activePlugin.Name()
		if c, ok := a.activePlugin.(sdk.Countable); ok {
			filtered, total = c.Count()
		}
		if p, ok := a.activePlugin.(sdk.Pinnable); ok {
			pinned = p.PinnedCount()
		}
		if pos, ok := a.activePlugin.(sdk.Positionable); ok {
			cursor, navigable = pos.CursorPosition()
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
	bordered := a.contentBorder.Render(content, title, filtered, total, pinned, cursor, navigable, a.width, contentHeight+borderChrome)

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
	padding := 2 // blank row after header + blank row before footer
	contentHeight := a.height - headerHeight - footerHeight - padding

	// Minimal header: context info on left, tfui on right
	headerStyle := lipgloss.NewStyle().
		Width(a.width)
	sep := sdk.StyleFaint.Render(" › ")
	projectStyle := lipgloss.NewStyle().Foreground(sdk.ColorPrimary).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(sdk.ColorText)

	var leftParts []string
	leftParts = append(leftParts, projectStyle.Render(filepath.Base(a.cfg.WorkingDir())))
	if !a.activeChdir.IsZero() {
		leftParts = append(leftParts, valueStyle.Render(a.activeChdir.String()))
	}
	if !a.activeWorkspace.IsZero() {
		leftParts = append(leftParts, valueStyle.Render(a.activeWorkspace.String()))
	}
	if a.lockInfo != nil {
		leftParts = append(leftParts, sdk.StyleError.Render("[locked]"))
	}
	left := strings.Join(leftParts, sep)
	right := sdk.StyleFaint.Render("tfui")
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

	return header + "\n\n" + content + "\n\n" + statusBar
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
