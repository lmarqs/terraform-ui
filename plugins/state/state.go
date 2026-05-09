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
	filtering     bool
	errMsg        string
	selected      int
	detail        string
	detailAddr    string
	detailScroll  int
	detailHScroll int
	detailWrap    bool
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
func (e *Plugin) Filtering() bool     { return e.filtering }
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
	e.filtering = false
	e.errMsg = ""
	e.selected = 0
	e.detail = ""
	e.detailAddr = ""
	return nil
}

// Activate triggers state loading when the user enters the plugin.
func (e *Plugin) Activate() tea.Cmd {
	// Check if the active context changed since last activation
	if e.session != nil {
		currentContext, _ := sdk.GetTyped[string](e.session, sdk.SessionKeyActiveContextAbs)
		if currentContext != e.scopedContext {
			// Context changed — reset state
			e.status = StatusIdle
			e.resources = nil
			e.filtered = nil
			e.filter = ""
			e.filtering = false
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
		// Check if there's an active context to scope to
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
		e.filtering = true
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
	e.filtering = false
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
	// Detail view — arrows scroll, left/right pan, w toggles wrap, esc goes back
	if e.status == StatusShowingDetail {
		switch msg.String() {
		case "esc":
			e.status = StatusDone
			e.detail = ""
			e.detailAddr = ""
			e.detailScroll = 0
			e.detailHScroll = 0
			e.filtering = true
		case "down":
			e.detailScroll++
		case "up":
			if e.detailScroll > 0 {
				e.detailScroll--
			}
		case "right":
			e.detailHScroll += 10
		case "left":
			e.detailHScroll -= 10
			if e.detailHScroll < 0 {
				e.detailHScroll = 0
			}
		case "w":
			e.detailWrap = !e.detailWrap
			e.detailScroll = 0
			e.detailHScroll = 0
		}
		return nil
	}

	// Global: ctrl+w toggles wrap from any mode
	if msg.String() == "ctrl+w" {
		e.detailWrap = !e.detailWrap
		e.detailScroll = 0
		e.detailHScroll = 0
		return nil
	}

	// Filter mode: typing goes to filter, navigation and enter still work
	if e.filtering {
		switch msg.String() {
		case "esc":
			e.filtering = false
			return nil
		case "enter":
			return e.InspectSelected()
		case "/":
			return nil
		case "down":
			e.MoveDown()
			return nil
		case "up":
			e.MoveUp()
			return nil
		case "backspace", "ctrl+h", "delete":
			e.BackspaceFilter()
			return nil
		default:
			if len(msg.String()) == 1 && msg.String() >= " " {
				e.AppendFilter(msg.String())
			}
			return nil
		}
	}

	// Normal mode
	switch msg.String() {
	case "esc":
		return func() tea.Msg { return sdk.DeactivateMsg{} }
	case "down":
		e.MoveDown()
	case "up":
		e.MoveUp()
	case "enter":
		return e.InspectSelected()
	case "/":
		e.filtering = true
		e.filter = ""
		e.filtered = e.resources
	case "r":
		if e.status == StatusError || e.status == StatusDone {
			return e.Refresh()
		}
	case "G":
		e.MoveToEnd()
	case "g":
		e.MoveToStart()
	case "w":
		e.detailWrap = !e.detailWrap
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
// Supports fuzzy matching: space-separated terms must all match (e.g. "rds cluster").
func (e *Plugin) SetFilter(filter string) {
	e.filter = filter
	e.selected = 0
	if filter == "" {
		e.filtered = e.resources
		e.log.Debug("state.filter", "filter", "", "results", len(e.resources))
		return
	}
	terms := strings.Fields(strings.ToLower(filter))
	var result []sdk.Resource
	for _, r := range e.resources {
		text := strings.ToLower(r.Address + " " + r.Type + " " + r.Module)
		if matchAllTerms(text, terms) {
			result = append(result, r)
		}
	}
	e.filtered = result
	e.log.Debug("state.filter", "filter", filter, "results", len(e.filtered))
}

func matchAllTerms(text string, terms []string) bool {
	for _, term := range terms {
		if !fuzzyContains(text, term) {
			return false
		}
	}
	return true
}

func fuzzyContains(text, pattern string) bool {
	if strings.Contains(text, pattern) {
		return true
	}
	ti := 0
	for pi := 0; pi < len(pattern); pi++ {
		found := false
		for ti < len(text) {
			if text[ti] == pattern[pi] {
				ti++
				found = true
				break
			}
			ti++
		}
		if !found {
			return false
		}
	}
	return true
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

// InspectSelected loads detail for the selected resource (enter = inspect, like k9s).
func (e *Plugin) InspectSelected() tea.Cmd {
	if e.status == StatusLoading {
		return nil
	}
	r := e.SelectedResource()
	if r.Address == "" {
		e.log.Debug("state.inspect.skip", "reason", "empty address", "selected", e.selected, "filtered", len(e.filtered))
		return nil
	}
	e.log.Debug("state.inspect.start", "address", r.Address)
	e.status = StatusLoading
	e.filtering = false
	e.errMsg = "Loading " + r.Address + "..."
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
		msg := "Loading terraform state..."
		if e.errMsg != "" {
			msg = e.errMsg
		}
		loading := sdk.StyleFaintItalic.Render(msg)
		hint := sdk.StyleFaintItalic.Render("Esc to go back")
		return sdk.StylePadded.Render(title + "\n\n" + loading + "\n\n" + hint)

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
	if e.filtering {
		filterLine = sdk.StyleKey.Render("/ ") + e.filter + "█\n\n"
	} else if e.filter != "" {
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

	contentWidth := width - 6
	for i := startIdx; i < endIdx; i++ {
		r := e.filtered[i]
		row := e.renderResourceRow(r)
		if i == e.selected {
			if e.detailWrap {
				row = sdk.StyleSelected.Render(row)
			} else {
				row = sdk.StyleSelected.Width(contentWidth).Render(row)
			}
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d resources", len(e.filtered)))
	if len(e.filtered) != len(e.resources) {
		count = sdk.StyleFaint.Render(fmt.Sprintf("%d/%d resources", len(e.filtered), len(e.resources)))
	}

	wrapLabel := "off"
	if e.detailWrap {
		wrapLabel = "on"
	}
	var hint string
	if e.filtering {
		hint = sdk.StyleFaintItalic.Render(fmt.Sprintf("Type to filter  ^w wrap(%s)  Esc exit", wrapLabel))
	} else {
		hint = sdk.StyleFaintItalic.Render(fmt.Sprintf("↑↓ navigate  Enter inspect  / filter  ^w wrap(%s)  : switch  q back", wrapLabel))
	}

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

	// Fixed header takes 2 lines (title + address)
	headerLines := 4
	contentWidth := width - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	lines := strings.Split(e.detail, "\n")
	if e.detailWrap {
		lines = wrapLines(lines, contentWidth)
	} else if e.detailHScroll > 0 {
		for i, line := range lines {
			if e.detailHScroll < len(line) {
				lines[i] = line[e.detailHScroll:]
			} else {
				lines[i] = ""
			}
		}
	}

	maxLines := height - headerLines - 4
	if maxLines < 5 {
		maxLines = 5
	}

	// Clamp scroll
	maxScroll := len(lines) - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.detailScroll > maxScroll {
		e.detailScroll = maxScroll
	}

	// Slice visible window
	endIdx := e.detailScroll + maxLines
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	visible := lines[e.detailScroll:endIdx]

	detail := strings.Join(visible, "\n")

	scrollInfo := ""
	if maxScroll > 0 {
		scrollInfo = sdk.StyleFaint.Render(fmt.Sprintf(" [%d/%d]", e.detailScroll+1, maxScroll+1))
	}

	wrapIndicator := "off"
	if e.detailWrap {
		wrapIndicator = "on"
	}

	hint := sdk.StyleFaintItalic.Render(fmt.Sprintf("↑↓ scroll  ←→ pan  ^w wrap(%s)  Esc back", wrapIndicator))

	content := title + "\n" + address + scrollInfo + "\n\n" + detail + "\n\n" + hint
	return sdk.StylePadded.Render(content)
}

func wrapLines(lines []string, width int) []string {
	var result []string
	for _, line := range lines {
		if len(line) <= width {
			result = append(result, line)
			continue
		}
		for len(line) > width {
			result = append(result, line[:width])
			line = line[width:]
		}
		if len(line) > 0 {
			result = append(result, line)
		}
	}
	return result
}
