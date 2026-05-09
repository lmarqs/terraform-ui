package projects

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

// Status represents the current state of the projects extension.
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

// Extension implements the monorepo project picker feature.
type Extension struct {
	svc      terraform.Service
	cfg      config.Config
	status   Status
	projects []Project
	selected int
	active   int
	errMsg   string
	filter   string
	filtered []Project
}

// New creates a new projects extension.
func New() *Extension {
	return &Extension{}
}

func (e *Extension) Name() string        { return "Projects" }
func (e *Extension) Description() string  { return "Navigate terraform projects in a monorepo" }
func (e *Extension) KeyBinding() string   { return "m" }
func (e *Extension) Ready() bool          { return e.status == StatusDone }
func (e *Extension) Status() Status       { return e.status }
func (e *Extension) Selected() int        { return e.selected }
func (e *Extension) Active() int          { return e.active }
func (e *Extension) Filter() string       { return e.filter }
func (e *Extension) ProjectCount() int    { return len(e.projects) }

// SetConfig provides the application configuration for project discovery.
func (e *Extension) SetConfig(cfg config.Config) {
	e.cfg = cfg
}

// Init initializes the extension and discovers projects.
func (e *Extension) Init(svc terraform.Service) tea.Cmd {
	e.svc = svc
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
func (e *Extension) Refresh() tea.Cmd {
	e.status = StatusLoading
	e.errMsg = ""
	e.filter = ""
	return e.discover()
}

func (e *Extension) discover() tea.Cmd {
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

// Update processes messages and returns the updated extension.
func (e *Extension) Update(msg tea.Msg) (tea.Cmd, bool) {
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
		return nil, true

	case tea.KeyMsg:
		return e.handleKey(msg), true
	}
	return nil, false
}

func (e *Extension) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		e.MoveDown()
	case "k", "up":
		e.MoveUp()
	case "enter":
		e.SelectCurrent()
	case "/":
		// Filter mode
	case "backspace":
		e.BackspaceFilter()
	case "r":
		return e.Refresh()
	default:
		if len(msg.String()) == 1 && msg.String() >= " " && msg.String() != "j" && msg.String() != "k" {
			// Only append to filter if we're in filter mode
			// For simplicity, typing any non-navigation char filters
		}
	}
	return nil
}

// MoveUp moves selection up.
func (e *Extension) MoveUp() {
	if e.selected > 0 {
		e.selected--
	}
}

// MoveDown moves selection down.
func (e *Extension) MoveDown() {
	if e.selected < len(e.filtered)-1 {
		e.selected++
	}
}

// SelectCurrent marks the currently selected project as active.
func (e *Extension) SelectCurrent() {
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
func (e *Extension) ActiveProject() *Project {
	if e.active < len(e.projects) {
		return &e.projects[e.active]
	}
	return nil
}

// SelectedProject returns the currently highlighted project.
func (e *Extension) SelectedProject() *Project {
	if e.selected < len(e.filtered) {
		return &e.filtered[e.selected]
	}
	return nil
}

// SetFilter sets the filter and refilters the project list.
func (e *Extension) SetFilter(filter string) {
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
func (e *Extension) AppendFilter(ch string) {
	e.SetFilter(e.filter + ch)
}

// BackspaceFilter removes the last character from the filter.
func (e *Extension) BackspaceFilter() {
	if len(e.filter) > 0 {
		e.SetFilter(e.filter[:len(e.filter)-1])
	}
}

// View renders the projects extension.
func (e *Extension) View(width, height int) string {
	title := styles.StyleTitle.Render("Projects")

	switch e.status {
	case StatusIdle, StatusLoading:
		loading := styles.StyleFaintItalic.Render("Discovering projects...")
		return styles.StylePadded.Render(title + "\n\n" + loading)

	case StatusError:
		errText := styles.StyleError.Render("Error: " + e.errMsg)
		hint := styles.StyleFaintItalic.Render("Press r to retry, Esc to go back")
		return styles.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	case StatusDone:
		return e.renderProjects(width, height)

	default:
		return ""
	}
}

func (e *Extension) renderProjects(width, height int) string {
	title := styles.StyleTitle.Render("Projects")

	if len(e.projects) == 0 {
		placeholder := styles.StyleFaintItalic.Render(
			"No projects configured. Add paths to tfui.yaml:\n\n" +
				"  projects:\n" +
				"    paths:\n" +
				"      - \"modules/*\"\n" +
				"      - \"envs/**\"",
		)
		return styles.StylePadded.Render(title + "\n\n" + placeholder)
	}

	var b strings.Builder

	// Filter line
	if e.filter != "" {
		filterLine := styles.StyleKey.Render("filter: ") + e.filter
		b.WriteString(filterLine)
		b.WriteString("\n\n")
	}

	if len(e.filtered) == 0 {
		noMatch := styles.StyleFaintItalic.Render("No projects match filter.")
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
				row = styles.StyleSelected.Width(width - 6).Render(row)
			}
			b.WriteString(row)
			b.WriteByte('\n')
		}
	}

	count := styles.StyleFaint.Render(fmt.Sprintf("%d project(s)", len(e.filtered)))
	if len(e.filtered) != len(e.projects) {
		count = styles.StyleFaint.Render(fmt.Sprintf("%d/%d project(s)", len(e.filtered), len(e.projects)))
	}

	hint := styles.StyleFaintItalic.Render("Enter select  / filter  r refresh  Esc back")

	content := title + "\n\n" + b.String() + "\n" + count + "\n" + hint
	return styles.StylePadded.Render(content)
}

func (e *Extension) renderProjectRow(project Project, idx int) string {
	// Check if this project is active (map filtered index to real index)
	isActive := false
	for i, p := range e.projects {
		if p.Path == project.Path && i == e.active {
			isActive = true
			break
		}
	}

	indicator := "  "
	name := styles.StyleFaint.Render(project.Path)
	if isActive {
		indicator = styles.StyleSuccess.Render("* ")
		name = styles.StyleKey.Render(project.Path)
	}

	row := fmt.Sprintf("%s%s", indicator, name)

	// Show the friendly name if different from path
	if project.Name != project.Path && project.Name != "" {
		row += " " + styles.StyleFaint.Render("("+project.Name+")")
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
