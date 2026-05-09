package state

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Status represents the current state of the state browser plugin.
type Status int

const (
	StatusIdle Status = iota
	StatusLoading
	StatusDone
	StatusError
	StatusShowingDetail
)

// StateListMsg is sent when state list completes.
type StateListMsg struct {
	Resources []sdk.Resource
	Err       error
}

// ResourceDetailMsg is sent when resource detail loads.
type ResourceDetailMsg struct {
	Address string
	Detail  string
	Err     error
}

// Plugin implements the state browser feature.
type Plugin struct {
	svc           sdk.Service
	log           *slog.Logger
	session       *sdk.Session
	status        Status
	resources     []sdk.Resource
	filtered      []sdk.Resource
	filter        string
	errMsg        string
	selected      int
	detail        string
	detailAddr    string
	scopedContext string
}

// New creates a new state browser plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc: svc,
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func (e *Plugin) ID() string          { return "state" }
func (e *Plugin) Name() string        { return "State Browser" }
func (e *Plugin) Description() string { return "Browse and inspect terraform state resources" }
func (e *Plugin) KeyBinding() string  { return "s" }
func (e *Plugin) Ready() bool         { return e.status == StatusDone || e.status == StatusShowingDetail }
func (e *Plugin) Status() Status      { return e.status }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) Filter() string      { return e.filter }
func (e *Plugin) ResourceCount() int  { return len(e.filtered) }
func (e *Plugin) TotalCount() int     { return len(e.resources) }

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context. Does not auto-load state.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	e.log = ctx.Logger
	e.session = ctx.Session
	e.status = StatusIdle
	e.resources = nil
	e.filtered = nil
	e.filter = ""
	e.errMsg = ""
	e.selected = 0
	e.detail = ""
	e.detailAddr = ""
	return nil
}

// Activate triggers state loading when the user enters the plugin.
func (e *Plugin) Activate() tea.Cmd {
	// Check if the active project changed since last activation
	if e.session != nil {
		currentContext, _ := sdk.GetTyped[string](e.session, sdk.SessionKeyActiveContextAbs)
		if currentContext != e.scopedContext {
			// Project changed — reset state
			e.status = StatusIdle
			e.resources = nil
			e.filtered = nil
			e.filter = ""
			e.errMsg = ""
			e.selected = 0
			e.detail = ""
			e.detailAddr = ""
			e.scopedContext = currentContext
			if currentContext != "" {
				e.svc = e.svc.WithDir(currentContext)
			}
		}
	}

	if e.status == StatusIdle || e.status == StatusError {
		// Check if there's an active project to scope to
		if e.session != nil {
			if dir, ok := sdk.GetTyped[string](e.session, sdk.SessionKeyActiveContextAbs); ok && dir != "" {
				e.svc = e.svc.WithDir(dir)
				e.scopedContext = dir
			} else if count, ok := sdk.GetTyped[int](e.session, sdk.SessionKeyContextCount); ok && count > 1 {
				e.status = StatusError
				e.errMsg = "Select a context first (press c)"
				return nil
			}
		}
		e.status = StatusLoading
		return e.loadState()
	}
	return nil
}

// Refresh reloads the state.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = StatusLoading
	e.resources = nil
	e.filtered = nil
	e.filter = ""
	e.errMsg = ""
	e.selected = 0
	e.detail = ""
	e.detailAddr = ""
	return e.loadState()
}

func (e *Plugin) loadState() tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		resources, err := svc.StateList(context.Background())
		return StateListMsg{Resources: resources, Err: err}
	}
}

func (e *Plugin) loadDetail(address string) tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		detail, err := svc.Show(context.Background(), address)
		return ResourceDetailMsg{Address: address, Detail: detail, Err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case StateListMsg:
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
			e.log.Debug("state.load.error", "error", msg.Err.Error())
		} else {
			e.status = StatusDone
			e.resources = msg.Resources
			e.filtered = msg.Resources
			e.log.Debug("state.load.complete", "resources", len(msg.Resources))
		}
		return e, nil

	case ResourceDetailMsg:
		if msg.Err != nil {
			e.errMsg = msg.Err.Error()
			e.status = StatusDone
			e.log.Debug("state.inspect.error", "address", msg.Address, "error", msg.Err.Error())
		} else {
			e.detail = msg.Detail
			e.detailAddr = msg.Address
			e.status = StatusShowingDetail
			e.log.Debug("state.inspect", "address", msg.Address)
		}
		return e, nil

	case tea.KeyMsg:
		cmd := e.handleKey(msg)
		return e, cmd
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Detail view has its own key handling
	if e.status == StatusShowingDetail {
		switch msg.String() {
		case "esc", "q":
			e.status = StatusDone
			e.detail = ""
			e.detailAddr = ""
		}
		return nil
	}

	switch msg.String() {
	case "j", "down":
		e.MoveDown()
	case "k", "up":
		e.MoveUp()
	case "enter":
		return e.InspectSelected()
	case "r":
		if e.status == StatusError || e.status == StatusDone {
			return e.Refresh()
		}
	case "G":
		e.MoveToEnd()
	case "g":
		e.MoveToStart()
	case "/":
		// Filter mode is handled by character input below
	case "backspace", "ctrl+h", "delete":
		e.BackspaceFilter()
	default:
		// Single printable characters go to filter
		if len(msg.String()) == 1 && msg.String() >= " " {
			e.AppendFilter(msg.String())
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

// MoveToStart moves selection to the first item.
func (e *Plugin) MoveToStart() {
	e.selected = 0
}

// MoveToEnd moves selection to the last item.
func (e *Plugin) MoveToEnd() {
	if len(e.filtered) > 0 {
		e.selected = len(e.filtered) - 1
	}
}

// SetFilter sets the filter string and refilters the resource list.
func (e *Plugin) SetFilter(filter string) {
	e.filter = filter
	e.selected = 0
	if filter == "" {
		e.filtered = e.resources
		e.log.Debug("state.filter", "filter", "", "results", len(e.resources))
		return
	}
	lower := strings.ToLower(filter)
	var result []sdk.Resource
	for _, r := range e.resources {
		if strings.Contains(strings.ToLower(r.Address), lower) ||
			strings.Contains(strings.ToLower(r.Type), lower) ||
			strings.Contains(strings.ToLower(r.Module), lower) {
			result = append(result, r)
		}
	}
	e.filtered = result
	e.log.Debug("state.filter", "filter", filter, "results", len(e.filtered))
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

// SelectedResource returns the currently selected resource.
func (e *Plugin) SelectedResource() sdk.Resource {
	if e.selected < len(e.filtered) {
		return e.filtered[e.selected]
	}
	return sdk.Resource{}
}

// InspectSelected loads detailed info about the selected resource.
func (e *Plugin) InspectSelected() tea.Cmd {
	r := e.SelectedResource()
	if r.Address == "" {
		return nil
	}
	return e.loadDetail(r.Address)
}

// View renders the state browser plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case StatusIdle:
		title := sdk.StyleTitle.Render("State Browser")
		placeholder := sdk.StyleFaintItalic.Render("Loading state...")
		return sdk.StylePadded.Render(title + "\n\n" + placeholder)

	case StatusLoading:
		title := sdk.StyleTitle.Render("State Browser")
		loading := sdk.StyleFaintItalic.Render("Loading terraform state...")
		return sdk.StylePadded.Render(title + "\n\n" + loading)

	case StatusError:
		title := sdk.StyleTitle.Render("State Browser")
		errText := sdk.StyleError.Render("Error: " + e.errMsg)
		hint := sdk.StyleFaintItalic.Render("Press r to retry, Esc to go back")
		return sdk.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	case StatusShowingDetail:
		return e.renderDetail(width, height)

	case StatusDone:
		return e.renderResources(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderResources(width, height int) string {
	title := sdk.StyleTitle.Render("State Browser")

	filterLine := ""
	if e.filter != "" {
		filterLine = sdk.StyleKey.Render("filter: ") + e.filter + "\n\n"
	}

	if len(e.filtered) == 0 {
		noResources := sdk.StyleFaintItalic.Render("No resources found.")
		return sdk.StylePadded.Render(title + "\n\n" + filterLine + noResources)
	}

	var b strings.Builder

	// Calculate visible area
	maxVisible := height - 7
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
		r := e.filtered[i]
		row := e.renderResourceRow(r)
		if i == e.selected {
			row = sdk.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d resources", len(e.filtered)))
	if len(e.filtered) != len(e.resources) {
		count = sdk.StyleFaint.Render(fmt.Sprintf("%d/%d resources", len(e.filtered), len(e.resources)))
	}

	hint := sdk.StyleFaintItalic.Render("j/k navigate  Enter inspect  / filter  r refresh  Esc back")

	content := title + "\n\n" + filterLine + b.String() + "\n" + count + "\n" + hint
	return sdk.StylePadded.Render(content)
}

func (e *Plugin) renderResourceRow(r sdk.Resource) string {
	address := r.Address
	typeInfo := sdk.StyleFaint.Render(r.Type)

	row := fmt.Sprintf(" %s  %s", address, typeInfo)
	if r.Module != "" {
		module := sdk.StyleKey.Render(fmt.Sprintf("[%s]", r.Module))
		row += " " + module
	}
	return row
}

func (e *Plugin) renderDetail(width, height int) string {
	title := sdk.StyleTitle.Render("Resource Detail")
	address := sdk.StyleKey.Render(e.detailAddr)

	// Truncate detail to visible area
	lines := strings.Split(e.detail, "\n")
	maxLines := height - 6
	if maxLines < 5 {
		maxLines = 5
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, sdk.StyleFaint.Render("... (truncated)"))
	}

	detail := strings.Join(lines, "\n")
	hint := sdk.StyleFaintItalic.Render("Esc/q to go back")

	content := title + "\n" + address + "\n\n" + detail + "\n\n" + hint
	return sdk.StylePadded.Render(content)
}
