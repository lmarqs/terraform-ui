package scope

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
)

// Status represents the current state of the scope plugin.
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

// Plugin implements the scope picker — select a subdirectory within the project.
type Plugin struct {
	svc       sdk.Service
	cfg       config.Config
	log       *slog.Logger
	session   *sdk.Session
	status    Status
	scopes    []Scope
	filtered  []Scope
	selected  int
	active    int // -1 = no selection yet
	errMsg    string
	filtering bool
	filter    string
	filterFrm *frames.FilterFrame
}

// New creates a new scope plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc:    svc,
		log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		active: -1,
	}
}

func (p *Plugin) ID() string          { return "scope" }
func (p *Plugin) Name() string        { return "Scope" }
func (p *Plugin) Description() string { return "Select terraform scope (subdirectory)" }
func (p *Plugin) Ready() bool         { return p.status == StatusDone }
func (p *Plugin) Status() Status      { return p.status }
func (p *Plugin) ScopeCount() int     { return len(p.scopes) }

// Hints returns context-sensitive key hints for the status bar.
func (p *Plugin) Hints() []sdk.KeyHint {
	if p.filtering {
		return (sdk.HintSetNavigate | sdk.HintSetSelect | sdk.HintSetCancel).Hints()
	}
	if p.status == StatusDone && len(p.scopes) > 0 {
		return (sdk.HintSetNavigate | sdk.HintSetFilter | sdk.HintSetSelect | sdk.HintSetBack).Hints()
	}
	return (sdk.HintSetBack).Hints()
}

// Configure applies plugin-specific options from config.
func (p *Plugin) Configure(opts map[string]interface{}) error {
	return nil
}

// SetConfig provides the application configuration for scope discovery.
func (p *Plugin) SetConfig(cfg config.Config) {
	p.cfg = cfg
}

// Init initializes the plugin with shared context.
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	if ctx.Logger != nil {
		p.log = ctx.Logger
	}
	p.session = ctx.Session
	p.status = StatusIdle
	p.scopes = nil
	p.filtered = nil
	p.errMsg = ""
	p.selected = 0
	p.active = -1
	p.filtering = false
	p.filter = ""
	p.filterFrm = nil
	return nil
}

// Activate triggers scope discovery when the user enters the plugin.
func (p *Plugin) Activate() tea.Cmd {
	if p.status == StatusIdle || p.status == StatusError {
		p.status = StatusLoading
		p.log.Debug("scope.activate", "dir", p.cfg.Dir, "paths", p.cfg.Scope.Paths)
		return p.discover()
	}
	return nil
}

func (p *Plugin) discover() tea.Cmd {
	cfg := p.cfg
	return func() tea.Msg {
		paths, err := cfg.DiscoverScopes()
		if err != nil {
			return ScopeDiscoveredMsg{Err: err}
		}

		scopes := make([]Scope, 0, len(paths))
		absDir, _ := filepath.Abs(cfg.Dir)
		for _, path := range paths {
			scopes = append(scopes, Scope{
				Path:    path,
				Name:    deriveScopeName(path),
				AbsPath: filepath.Join(absDir, path),
			})
		}
		return ScopeDiscoveredMsg{Scopes: scopes}
	}
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ScopeDiscoveredMsg:
		if msg.Err != nil {
			p.status = StatusError
			p.errMsg = msg.Err.Error()
			p.log.Debug("scope.discover.error", "error", msg.Err.Error())
		} else {
			p.status = StatusDone
			p.scopes = msg.Scopes
			p.filtered = msg.Scopes
			p.log.Debug("scope.discover.complete", "scopes", len(msg.Scopes))
			if p.session != nil {
				p.session.Set(sdk.SessionKeyScopeCount, len(msg.Scopes))
			}
		}
		return p, nil

	case tea.KeyMsg:
		if p.filtering {
			return p.updateFilter(msg)
		}
		switch msg.String() {
		case "j", "down":
			if p.selected < len(p.filtered)-1 {
				p.selected++
			}
		case "k", "up":
			if p.selected > 0 {
				p.selected--
			}
		case "enter":
			return p, p.selectCurrent()
		case "r":
			return p, p.refresh()
		case "/":
			p.filtering = true
			p.filter = ""
			p.filtered = p.scopes
			p.selected = 0
			p.filterFrm = frames.NewFilterFrame(frames.FilterOpts{
				OnFilter:   func(q string) { p.setFilter(q) },
				OnSelect:   func() tea.Cmd { return p.selectCurrent() },
				OnNavigate: func(dir int) { p.navigate(dir) },
			})
			return p, nil
		}
	}
	return p, nil
}

func (p *Plugin) updateFilter(msg tea.KeyMsg) (sdk.Plugin, tea.Cmd) {
	result, cmd := p.filterFrm.Update(msg)
	if result == nil {
		p.filtering = false
		p.filterFrm = nil
		if p.filter == "" {
			p.filtered = p.scopes
		}
		return p, cmd
	}
	return p, cmd
}

func (p *Plugin) navigate(dir int) {
	if dir > 0 && p.selected < len(p.filtered)-1 {
		p.selected++
	} else if dir < 0 && p.selected > 0 {
		p.selected--
	}
}

func (p *Plugin) setFilter(query string) {
	p.filter = query
	if query == "" {
		p.filtered = p.scopes
	} else {
		lower := strings.ToLower(query)
		var results []Scope
		for _, s := range p.scopes {
			if strings.Contains(strings.ToLower(s.Path), lower) {
				results = append(results, s)
			}
		}
		p.filtered = results
	}
	p.selected = 0
}

func (p *Plugin) selectCurrent() tea.Cmd {
	if p.selected >= len(p.filtered) {
		return nil
	}
	s := p.filtered[p.selected]
	// Find the index in the original scopes list for the active marker.
	for i, sc := range p.scopes {
		if sc.Path == s.Path {
			p.active = i
			break
		}
	}
	if p.session != nil {
		p.session.Set(sdk.SessionKeyActiveScope, s.Path)
		p.session.Set(sdk.SessionKeyActiveScopeAbs, s.AbsPath)
	}
	return func() tea.Msg { return sdk.DeactivateMsg{} }
}

func (p *Plugin) refresh() tea.Cmd {
	p.status = StatusLoading
	p.errMsg = ""
	p.filtering = false
	p.filter = ""
	p.filterFrm = nil
	p.filtered = nil
	return p.discover()
}

// ActiveScope returns the currently active scope.
func (p *Plugin) ActiveScope() *Scope {
	if p.active >= 0 && p.active < len(p.scopes) {
		return &p.scopes[p.active]
	}
	return nil
}

// View renders the scope picker.
func (p *Plugin) View(width, height int) string {
	switch p.status {
	case StatusIdle, StatusLoading:
		return sdk.StyleFaintItalic.Render("Discovering scopes...")
	case StatusError:
		return sdk.StyleError.Render("Error: " + p.errMsg)
	case StatusDone:
		return p.renderScopes(width, height)
	default:
		return ""
	}
}

func (p *Plugin) renderScopes(width, height int) string {
	if len(p.scopes) == 0 {
		return sdk.StyleFaintItalic.Render(
			"No scopes configured. Add paths to tfui.yaml:\n\n" +
				"  scope:\n" +
				"    paths:\n" +
				"      - \"modules/*\"\n" +
				"      - \"envs/**\"",
		)
	}

	filterLine := ""
	if p.filtering && p.filterFrm != nil {
		filterLine = p.filterFrm.View(width, 1) + "\n\n"
	} else if p.filter != "" {
		filterLine = sdk.StyleKey.Render("filter: ") + p.filter + "\n\n"
	}

	filterHeight := 0
	if filterLine != "" {
		filterHeight = 2
	}

	maxVisible := height - 4 - filterHeight
	if maxVisible < 3 {
		maxVisible = 3
	}

	items := p.filtered

	startIdx := 0
	if p.selected >= maxVisible {
		startIdx = p.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(items) {
		endIdx = len(items)
	}

	activeScope := ""
	if p.active >= 0 && p.active < len(p.scopes) {
		activeScope = p.scopes[p.active].Path
	}

	var b strings.Builder
	for i := startIdx; i < endIdx; i++ {
		s := items[i]
		indicator := "  "
		name := s.Path
		if s.Path == activeScope {
			indicator = sdk.StyleSuccess.Render("* ")
			name = sdk.StyleKey.Render(s.Path)
		}
		row := fmt.Sprintf("%s%s", indicator, name)
		if i == p.selected {
			row = sdk.StyleSelected.Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d/%d scope(s)", len(items), len(p.scopes)))
	return filterLine + b.String() + "\n" + count
}

func deriveScopeName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}
