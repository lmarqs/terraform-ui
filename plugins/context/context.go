package context

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Status represents the current state of the projects plugin.
type Status int

const (
	StatusIdle Status = iota
	StatusLoading
	StatusDone
	StatusError
)

// ScopeDiscoveredMsg is sent when scope discovery completes.
type ScopeDiscoveredMsg struct {
	Scopes []Scope
	Err    error
}

// Scope represents a discovered terraform scope (subdirectory) in the monorepo.
type Scope struct {
	// Path is the relative path from the monorepo root.
	Path string
	// Name is a display-friendly name derived from the path.
	Name string
	// AbsPath is the absolute path to the scope.
	AbsPath string
}

// Plugin implements the monorepo scope picker feature.
type Plugin struct {
	svc      sdk.Service
	cfg      config.Config
	log      *slog.Logger
	session  *sdk.Session
	stack    *sdk.Stack
	status   Status
	scopes   []Scope
	selected int
	active   int // -1 = no selection yet
	errMsg   string
}

// New creates a new context plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		svc:    svc,
		log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		active: -1,
	}
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	return p
}

func (e *Plugin) ID() string          { return "context" }
func (e *Plugin) Name() string        { return "Context" }
func (e *Plugin) Description() string { return "Select terraform project scope" }
func (e *Plugin) KeyBinding() string  { return "" }
func (e *Plugin) Ready() bool       { return e.status == StatusDone }
func (e *Plugin) Status() Status    { return e.status }
func (e *Plugin) Selected() int     { return e.selected }
func (e *Plugin) Active() int       { return e.active }
func (e *Plugin) ScopeCount() int   { return len(e.scopes) }
func (e *Plugin) Stack() *sdk.Stack { return e.stack }

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(opts map[string]interface{}) error {
	return nil
}

// SetConfig provides the application configuration for context discovery.
func (e *Plugin) SetConfig(cfg config.Config) {
	e.cfg = cfg
}

// Init initializes the plugin with shared context. Does not auto-discover.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	if ctx.Logger != nil {
		e.log = ctx.Logger
	}
	e.session = ctx.Session
	e.status = StatusIdle
	e.scopes = nil
	e.errMsg = ""
	e.selected = 0
	e.active = -1
	return nil
}

// Activate triggers context discovery when the user enters the plugin.
func (e *Plugin) Activate() tea.Cmd {
	if e.status == StatusIdle || e.status == StatusError {
		e.status = StatusLoading
		e.log.Debug("context.activate", "dir", e.cfg.Dir, "paths", e.cfg.Scope.Paths)
		return e.discover()
	}
	return nil
}

// Refresh re-discovers context.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = StatusLoading
	e.errMsg = ""
	return e.discover()
}

func (e *Plugin) discover() tea.Cmd {
	cfg := e.cfg
	return func() tea.Msg {
		paths, err := cfg.DiscoverScopes()
		if err != nil {
			return ScopeDiscoveredMsg{Err: err}
		}

		scopes := make([]Scope, 0, len(paths))
		absDir, _ := filepath.Abs(cfg.Dir)
		for _, p := range paths {
			scopes = append(scopes, Scope{
				Path:    p,
				Name:    deriveScopeName(p),
				AbsPath: filepath.Join(absDir, p),
			})
		}
		return ScopeDiscoveredMsg{Scopes: scopes}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ScopeDiscoveredMsg:
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
			e.log.Debug("context.discover.error", "error", msg.Err.Error())
		} else {
			e.status = StatusDone
			e.scopes = msg.Scopes
			e.log.Debug("context.discover.complete", "scopes", len(msg.Scopes))
			if e.session != nil {
				e.session.Set(sdk.SessionKeyScopeCount, len(msg.Scopes))
			}
		}
		return e, nil
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
	if e.selected < len(e.scopes)-1 {
		e.selected++
	}
}

// SelectCurrent marks the currently selected scope as active and deactivates.
func (e *Plugin) SelectCurrent() tea.Cmd {
	if e.selected >= len(e.scopes) {
		return nil
	}
	e.active = e.selected
	p := e.scopes[e.selected]
	if e.session != nil {
		e.session.Set(sdk.SessionKeyActiveScope, p.Path)
		e.session.Set(sdk.SessionKeyActiveScopeAbs, p.AbsPath)
	}
	return func() tea.Msg { return sdk.DeactivateMsg{} }
}

// ActiveScope returns the currently active scope.
func (e *Plugin) ActiveScope() *Scope {
	if e.active >= 0 && e.active < len(e.scopes) {
		return &e.scopes[e.active]
	}
	return nil
}

// SelectedScope returns the currently highlighted scope.
func (e *Plugin) SelectedScope() *Scope {
	if e.selected < len(e.scopes) {
		return &e.scopes[e.selected]
	}
	return nil
}

// View renders the context plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case StatusIdle, StatusLoading:
		return sdk.StyleFaintItalic.Render("Discovering scopes...")

	case StatusError:
		return sdk.StyleError.Render("Error: " + e.errMsg)

	case StatusDone:
		return e.renderScopes(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderScopes(width, height int) string {
	if len(e.scopes) == 0 {
		return sdk.StyleFaintItalic.Render(
			"No scopes configured. Add paths to tfui.yaml:\n\n" +
				"  scope:\n" +
				"    paths:\n" +
				"      - \"modules/*\"\n" +
				"      - \"envs/**\"",
		)
	}

	var b strings.Builder

	maxVisible := height - 4
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if e.selected >= maxVisible {
		startIdx = e.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(e.scopes) {
		endIdx = len(e.scopes)
	}

	for i := startIdx; i < endIdx; i++ {
		scope := e.scopes[i]
		row := e.renderScopeRow(scope, i)
		if i == e.selected {
			row = sdk.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d scope(s)", len(e.scopes)))

	return b.String() + "\n" + count
}

func (e *Plugin) renderScopeRow(scope Scope, idx int) string {
	isActive := false
	if e.active >= 0 {
		for i, s := range e.scopes {
			if s.Path == scope.Path && i == e.active {
				isActive = true
				break
			}
		}
	}

	indicator := "  "
	name := sdk.StyleFaint.Render(scope.Path)
	if isActive {
		indicator = sdk.StyleSuccess.Render("* ")
		name = sdk.StyleKey.Render(scope.Path)
	}

	return fmt.Sprintf("%s%s", indicator, name)
}

// listFrame is the root frame for the context plugin's project list.
type listFrame struct {
	plugin *Plugin
}

func (f *listFrame) ID() string { return "list" }

func (f *listFrame) Update(msg tea.Msg) (sdk.Frame, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}

	switch keyMsg.String() {
	case "esc":
		return f, func() tea.Msg { return sdk.DeactivateMsg{} }
	case "j", "down":
		f.plugin.MoveDown()
	case "k", "up":
		f.plugin.MoveUp()
	case "enter":
		return f, f.plugin.SelectCurrent()
	case "r":
		return f, f.plugin.Refresh()
	}
	return f, nil
}

func (f *listFrame) View(width, height int) string {
	return f.plugin.View(width, height)
}

func (f *listFrame) Hints() []sdk.KeyHint {
	switch f.plugin.status {
	case StatusError:
		return (sdk.HintSetRetry | sdk.HintSetBack).Hints()
	case StatusDone:
		if len(f.plugin.scopes) == 0 {
			return sdk.HintSetBack.Hints()
		}
		return (sdk.HintSetNavigate | sdk.HintSetSelect | sdk.HintSetRefresh | sdk.HintSetBack).Hints()
	default:
		return nil
	}
}

// deriveScopeName creates a display name from a scope path.
func deriveScopeName(path string) string {
	// Use the last path component as the name
	base := filepath.Base(path)
	if base == "." || base == "/" {
		return path
	}
	return base
}
