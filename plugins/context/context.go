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

// ContextDiscoveredMsg is sent when project discovery completes.
type ContextDiscoveredMsg struct {
	Projects []Project
	Err      error
}

// Project represents a discovered terraform project in the monorepo.
type Project struct {
	// Path is the relative path from the monorepo root.
	Path string
	// Name is a display-friendly name derived from the path.
	Name string
	// AbsPath is the absolute path to the project.
	AbsPath string
}

// Plugin implements the monorepo project picker feature.
type Plugin struct {
	svc      sdk.Service
	cfg      config.Config
	log      *slog.Logger
	session  *sdk.Session
	stack    *sdk.Stack
	status   Status
	projects []Project
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
func (e *Plugin) Ready() bool         { return e.status == StatusDone }
func (e *Plugin) Status() Status      { return e.status }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) Active() int         { return e.active }
func (e *Plugin) ContextCount() int   { return len(e.projects) }
func (e *Plugin) Stack() *sdk.Stack   { return e.stack }

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
	e.projects = nil
	e.errMsg = ""
	e.selected = 0
	e.active = -1
	return nil
}

// Activate triggers context discovery when the user enters the plugin.
func (e *Plugin) Activate() tea.Cmd {
	if e.status == StatusIdle || e.status == StatusError {
		e.status = StatusLoading
		e.log.Debug("context.activate", "dir", e.cfg.Dir, "paths", e.cfg.Context.Paths)
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
		paths, err := cfg.DiscoverContext()
		if err != nil {
			return ContextDiscoveredMsg{Err: err}
		}

		projects := make([]Project, 0, len(paths))
		absDir, _ := filepath.Abs(cfg.Dir)
		for _, p := range paths {
			projects = append(projects, Project{
				Path:    p,
				Name:    deriveProjectName(p),
				AbsPath: filepath.Join(absDir, p),
			})
		}
		return ContextDiscoveredMsg{Projects: projects}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ContextDiscoveredMsg:
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
			e.log.Debug("context.discover.error", "error", msg.Err.Error())
		} else {
			e.status = StatusDone
			e.projects = msg.Projects
			e.log.Debug("context.discover.complete", "projects", len(msg.Projects))
			if e.session != nil {
				e.session.Set(sdk.SessionKeyContextCount, len(msg.Projects))
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
	if e.selected < len(e.projects)-1 {
		e.selected++
	}
}

// SelectCurrent marks the currently selected project as active and deactivates.
func (e *Plugin) SelectCurrent() tea.Cmd {
	if e.selected >= len(e.projects) {
		return nil
	}
	e.active = e.selected
	p := e.projects[e.selected]
	if e.session != nil {
		e.session.Set(sdk.SessionKeyActiveContext, p.Path)
		e.session.Set(sdk.SessionKeyActiveContextAbs, p.AbsPath)
	}
	return func() tea.Msg { return sdk.DeactivateMsg{} }
}

// ActiveProject returns the currently active project.
func (e *Plugin) ActiveProject() *Project {
	if e.active >= 0 && e.active < len(e.projects) {
		return &e.projects[e.active]
	}
	return nil
}

// SelectedProject returns the currently highlighted project.
func (e *Plugin) SelectedProject() *Project {
	if e.selected < len(e.projects) {
		return &e.projects[e.selected]
	}
	return nil
}

// View renders the context plugin.
func (e *Plugin) View(width, height int) string {
	title := sdk.StyleTitle.Render("Context")

	switch e.status {
	case StatusIdle, StatusLoading:
		loading := sdk.StyleFaintItalic.Render("Discovering context...")
		return sdk.StylePadded.Render(title + "\n\n" + loading)

	case StatusError:
		errText := sdk.StyleError.Render("Error: " + e.errMsg)
		return sdk.StylePadded.Render(title + "\n\n" + errText)

	case StatusDone:
		return e.renderProjects(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderProjects(width, height int) string {
	title := sdk.StyleTitle.Render("Context")

	if len(e.projects) == 0 {
		placeholder := sdk.StyleFaintItalic.Render(
			"No context configured. Add paths to tfui.yaml:\n\n" +
				"  context:\n" +
				"    paths:\n" +
				"      - \"modules/*\"\n" +
				"      - \"envs/**\"",
		)
		return sdk.StylePadded.Render(title + "\n\n" + placeholder)
	}

	var b strings.Builder

	maxVisible := height - 8
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if e.selected >= maxVisible {
		startIdx = e.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(e.projects) {
		endIdx = len(e.projects)
	}

	for i := startIdx; i < endIdx; i++ {
		project := e.projects[i]
		row := e.renderProjectRow(project, i)
		if i == e.selected {
			row = sdk.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d project(s)", len(e.projects)))

	content := title + "\n\n" + b.String() + "\n" + count
	return sdk.StylePadded.Render(content)
}

func (e *Plugin) renderProjectRow(project Project, idx int) string {
	isActive := false
	if e.active >= 0 {
		for i, p := range e.projects {
			if p.Path == project.Path && i == e.active {
				isActive = true
				break
			}
		}
	}

	indicator := "  "
	name := sdk.StyleFaint.Render(project.Path)
	if isActive {
		indicator = sdk.StyleSuccess.Render("* ")
		name = sdk.StyleKey.Render(project.Path)
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
		return []sdk.KeyHint{sdk.HintRetry, sdk.HintBack}
	case StatusDone:
		if len(f.plugin.projects) == 0 {
			return []sdk.KeyHint{sdk.HintBack}
		}
		return []sdk.KeyHint{sdk.HintNavigate, sdk.HintSelect, sdk.HintRefresh, sdk.HintBack}
	default:
		return nil
	}
}

// deriveProjectName creates a display name from a project path.
func deriveProjectName(path string) string {
	// Use the last path component as the name
	base := filepath.Base(path)
	if base == "." || base == "/" {
		return path
	}
	return base
}
