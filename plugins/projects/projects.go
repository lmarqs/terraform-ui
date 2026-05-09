package projects

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Status represents the current state of the projects sdk.
type Status int

const (
	StatusIdle Status = iota
	StatusLoading
	StatusDone
	StatusError
)

// ProjectsDiscoveredMsg is sent when project discovery completes.
type ProjectsDiscoveredMsg struct {
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
	status   Status
	projects []Project
	selected int
	active   int
	errMsg   string
	filter   string
	filtered []Project
}

// New creates a new projects sdk.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc: svc,
	}
}

func (e *Plugin) ID() string          { return "projects" }
func (e *Plugin) Name() string        { return "Projects" }
func (e *Plugin) Description() string { return "Navigate terraform projects in a monorepo" }
func (e *Plugin) KeyBinding() string  { return "m" }
func (e *Plugin) Ready() bool         { return e.status == StatusDone }
func (e *Plugin) Status() Status      { return e.status }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) Active() int         { return e.active }
func (e *Plugin) Filter() string      { return e.filter }
func (e *Plugin) ProjectCount() int   { return len(e.projects) }

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(opts map[string]interface{}) error {
	return nil
}

// SetConfig provides the application configuration for project discovery.
func (e *Plugin) SetConfig(cfg config.Config) {
	e.cfg = cfg
}

// Init initializes the plugin and discovers projects.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	e.status = StatusLoading
	e.projects = nil
	e.filtered = nil
	e.filter = ""
	e.errMsg = ""
	e.selected = 0
	e.active = 0
	return e.discover()
}

// Refresh re-discovers projects.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = StatusLoading
	e.errMsg = ""
	e.filter = ""
	return e.discover()
}

func (e *Plugin) discover() tea.Cmd {
	cfg := e.cfg
	return func() tea.Msg {
		paths, err := cfg.DiscoverProjects()
		if err != nil {
			return ProjectsDiscoveredMsg{Err: err}
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
		return ProjectsDiscoveredMsg{Projects: projects}
	}
}

// Update processes messages and returns the updated sdk.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ProjectsDiscoveredMsg:
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
		} else {
			e.status = StatusDone
			e.projects = msg.Projects
			e.filtered = msg.Projects
		}
		return e, nil

	case tea.KeyMsg:
		cmd := e.handleKey(msg)
		return e, cmd
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		e.MoveDown()
	case "k", "up":
		e.MoveUp()
	case "enter":
		e.SelectCurrent()
	case "/":
		// Filter mode
	case "backspace", "ctrl+h", "delete":
		e.BackspaceFilter()
	case "r":
		return e.Refresh()
	default:
		if len(msg.String()) == 1 && msg.String() >= " " && msg.String() != "j" && msg.String() != "k" {
			// Only append to filter if we're in filter mode
		}
	}
	return nil
}

// MoveUp moves selection up.
func (e *Plugin) MoveUp() {
	if e.selected > 0 {
		e.selected--
	}
}

// MoveDown moves selection down.
func (e *Plugin) MoveDown() {
	if e.selected < len(e.filtered)-1 {
		e.selected++
	}
}

// SelectCurrent marks the currently selected project as active.
func (e *Plugin) SelectCurrent() {
	if e.selected < len(e.filtered) {
		// Map filtered index back to project index
		selectedProject := e.filtered[e.selected]
		for i, p := range e.projects {
			if p.Path == selectedProject.Path {
				e.active = i
				break
			}
		}
	}
}

// ActiveProject returns the currently active project.
func (e *Plugin) ActiveProject() *Project {
	if e.active < len(e.projects) {
		return &e.projects[e.active]
	}
	return nil
}

// SelectedProject returns the currently highlighted project.
func (e *Plugin) SelectedProject() *Project {
	if e.selected < len(e.filtered) {
		return &e.filtered[e.selected]
	}
	return nil
}

// SetFilter sets the filter and refilters the project list.
func (e *Plugin) SetFilter(filter string) {
	e.filter = filter
	e.selected = 0
	if filter == "" {
		e.filtered = e.projects
		return
	}
	lower := strings.ToLower(filter)
	var result []Project
	for _, p := range e.projects {
		if strings.Contains(strings.ToLower(p.Path), lower) ||
			strings.Contains(strings.ToLower(p.Name), lower) {
			result = append(result, p)
		}
	}
	e.filtered = result
}

// AppendFilter adds a character to the filter.
func (e *Plugin) AppendFilter(ch string) {
	e.SetFilter(e.filter + ch)
}

// BackspaceFilter removes the last character from the filter.
func (e *Plugin) BackspaceFilter() {
	if len(e.filter) > 0 {
		e.SetFilter(e.filter[:len(e.filter)-1])
	}
}

// View renders the projects sdk.
func (e *Plugin) View(width, height int) string {
	title := sdk.StyleTitle.Render("Projects")

	switch e.status {
	case StatusIdle, StatusLoading:
		loading := sdk.StyleFaintItalic.Render("Discovering projects...")
		return sdk.StylePadded.Render(title + "\n\n" + loading)

	case StatusError:
		errText := sdk.StyleError.Render("Error: " + e.errMsg)
		hint := sdk.StyleFaintItalic.Render("Press r to retry, Esc to go back")
		return sdk.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	case StatusDone:
		return e.renderProjects(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderProjects(width, height int) string {
	title := sdk.StyleTitle.Render("Projects")

	if len(e.projects) == 0 {
		placeholder := sdk.StyleFaintItalic.Render(
			"No projects configured. Add paths to tfui.yaml:\n\n" +
				"  projects:\n" +
				"    paths:\n" +
				"      - \"modules/*\"\n" +
				"      - \"envs/**\"",
		)
		return sdk.StylePadded.Render(title + "\n\n" + placeholder)
	}

	var b strings.Builder

	// Filter line
	if e.filter != "" {
		filterLine := sdk.StyleKey.Render("filter: ") + e.filter
		b.WriteString(filterLine)
		b.WriteString("\n\n")
	}

	if len(e.filtered) == 0 {
		noMatch := sdk.StyleFaintItalic.Render("No projects match filter.")
		b.WriteString(noMatch)
	} else {
		// Calculate visible area
		maxVisible := height - 8
		if maxVisible < 3 {
			maxVisible = 3
		}

		startIdx := 0
		if e.selected >= maxVisible {
			startIdx = e.selected - maxVisible + 1
		}
		endIdx := startIdx + maxVisible
		if endIdx > len(e.filtered) {
			endIdx = len(e.filtered)
		}

		for i := startIdx; i < endIdx; i++ {
			project := e.filtered[i]
			row := e.renderProjectRow(project, i)
			if i == e.selected {
				row = sdk.StyleSelected.Width(width - 6).Render(row)
			}
			b.WriteString(row)
			b.WriteByte('\n')
		}
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d project(s)", len(e.filtered)))
	if len(e.filtered) != len(e.projects) {
		count = sdk.StyleFaint.Render(fmt.Sprintf("%d/%d project(s)", len(e.filtered), len(e.projects)))
	}

	hint := sdk.StyleFaintItalic.Render("Enter select  / filter  r refresh  Esc back")

	content := title + "\n\n" + b.String() + "\n" + count + "\n" + hint
	return sdk.StylePadded.Render(content)
}

func (e *Plugin) renderProjectRow(project Project, idx int) string {
	// Check if this project is active (map filtered index to real index)
	isActive := false
	for i, p := range e.projects {
		if p.Path == project.Path && i == e.active {
			isActive = true
			break
		}
	}

	indicator := "  "
	name := sdk.StyleFaint.Render(project.Path)
	if isActive {
		indicator = sdk.StyleSuccess.Render("* ")
		name = sdk.StyleKey.Render(project.Path)
	}

	row := fmt.Sprintf("%s%s", indicator, name)

	// Show the friendly name if different from path
	if project.Name != project.Path && project.Name != "" {
		row += " " + sdk.StyleFaint.Render("("+project.Name+")")
	}

	return row
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
