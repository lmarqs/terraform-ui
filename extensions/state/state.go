package state

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

// Status represents the current state of the state browser extension.
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
	Resources []terraform.Resource
	Err       error
}

// ResourceDetailMsg is sent when resource detail loads.
type ResourceDetailMsg struct {
	Address string
	Detail  string
	Err     error
}

// Extension implements the state browser feature.
type Extension struct {
	svc       terraform.Service
	status    Status
	resources []terraform.Resource
	filtered  []terraform.Resource
	filter    string
	errMsg    string
	selected  int
	detail    string
	detailAddr string
}

// New creates a new state browser extension.
func New() *Extension {
	return &Extension{}
}

func (e *Extension) Name() string        { return "State Browser" }
func (e *Extension) Description() string  { return "Browse and inspect terraform state resources" }
func (e *Extension) KeyBinding() string   { return "s" }
func (e *Extension) Ready() bool          { return e.status == StatusDone || e.status == StatusShowingDetail }
func (e *Extension) Status() Status       { return e.status }
func (e *Extension) Selected() int        { return e.selected }
func (e *Extension) Filter() string       { return e.filter }
func (e *Extension) ResourceCount() int   { return len(e.filtered) }
func (e *Extension) TotalCount() int      { return len(e.resources) }

// Init initializes the extension and loads state.
func (e *Extension) Init(svc terraform.Service) tea.Cmd {
	e.svc = svc
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

// Refresh reloads the state.
func (e *Extension) Refresh() tea.Cmd {
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

func (e *Extension) loadState() tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		resources, err := svc.StateList(context.Background())
		return StateListMsg{Resources: resources, Err: err}
	}
}

func (e *Extension) loadDetail(address string) tea.Cmd {
	svc := e.svc
	return func() tea.Msg {
		detail, err := svc.Show(context.Background(), address)
		return ResourceDetailMsg{Address: address, Detail: detail, Err: err}
	}
}

// Update processes messages and returns the updated extension.
func (e *Extension) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case StateListMsg:
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
		} else {
			e.status = StatusDone
			e.resources = msg.Resources
			e.filtered = msg.Resources
		}
		return nil, true

	case ResourceDetailMsg:
		if msg.Err != nil {
			e.errMsg = msg.Err.Error()
			e.status = StatusDone
		} else {
			e.detail = msg.Detail
			e.detailAddr = msg.Address
			e.status = StatusShowingDetail
		}
		return nil, true

	case tea.KeyMsg:
		return e.handleKey(msg), true
	}
	return nil, false
}

func (e *Extension) handleKey(msg tea.KeyMsg) tea.Cmd {
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
	case "backspace":
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

// MoveToStart moves selection to the first item.
func (e *Extension) MoveToStart() {
	e.selected = 0
}

// MoveToEnd moves selection to the last item.
func (e *Extension) MoveToEnd() {
	if len(e.filtered) > 0 {
		e.selected = len(e.filtered) - 1
	}
}

// SetFilter sets the filter string and refilters the resource list.
func (e *Extension) SetFilter(filter string) {
	e.filter = filter
	e.selected = 0
	if filter == "" {
		e.filtered = e.resources
		return
	}
	lower := strings.ToLower(filter)
	var result []terraform.Resource
	for _, r := range e.resources {
		if strings.Contains(strings.ToLower(r.Address), lower) ||
			strings.Contains(strings.ToLower(r.Type), lower) ||
			strings.Contains(strings.ToLower(r.Module), lower) {
			result = append(result, r)
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

// ClearFilter clears the filter.
func (e *Extension) ClearFilter() {
	e.SetFilter("")
}

// SelectedResource returns the currently selected resource.
func (e *Extension) SelectedResource() terraform.Resource {
	if e.selected < len(e.filtered) {
		return e.filtered[e.selected]
	}
	return terraform.Resource{}
}

// InspectSelected loads detailed info about the selected resource.
func (e *Extension) InspectSelected() tea.Cmd {
	r := e.SelectedResource()
	if r.Address == "" {
		return nil
	}
	return e.loadDetail(r.Address)
}

// View renders the state browser extension.
func (e *Extension) View(width, height int) string {
	switch e.status {
	case StatusIdle:
		title := styles.StyleTitle.Render("State Browser")
		placeholder := styles.StyleFaintItalic.Render("Loading state...")
		return styles.StylePadded.Render(title + "\n\n" + placeholder)

	case StatusLoading:
		title := styles.StyleTitle.Render("State Browser")
		loading := styles.StyleFaintItalic.Render("Loading terraform state...")
		return styles.StylePadded.Render(title + "\n\n" + loading)

	case StatusError:
		title := styles.StyleTitle.Render("State Browser")
		errText := styles.StyleError.Render("Error: " + e.errMsg)
		hint := styles.StyleFaintItalic.Render("Press r to retry, Esc to go back")
		return styles.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	case StatusShowingDetail:
		return e.renderDetail(width, height)

	case StatusDone:
		return e.renderResources(width, height)

	default:
		return ""
	}
}

func (e *Extension) renderResources(width, height int) string {
	title := styles.StyleTitle.Render("State Browser")

	filterLine := ""
	if e.filter != "" {
		filterLine = styles.StyleKey.Render("filter: ") + e.filter + "\n\n"
	}

	if len(e.filtered) == 0 {
		noResources := styles.StyleFaintItalic.Render("No resources found.")
		return styles.StylePadded.Render(title + "\n\n" + filterLine + noResources)
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
			row = styles.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := styles.StyleFaint.Render(fmt.Sprintf("%d resources", len(e.filtered)))
	if len(e.filtered) != len(e.resources) {
		count = styles.StyleFaint.Render(fmt.Sprintf("%d/%d resources", len(e.filtered), len(e.resources)))
	}

	hint := styles.StyleFaintItalic.Render("j/k navigate  Enter inspect  / filter  r refresh  Esc back")

	content := title + "\n\n" + filterLine + b.String() + "\n" + count + "\n" + hint
	return styles.StylePadded.Render(content)
}

func (e *Extension) renderResourceRow(r terraform.Resource) string {
	address := r.Address
	typeInfo := styles.StyleFaint.Render(r.Type)

	row := fmt.Sprintf(" %s  %s", address, typeInfo)
	if r.Module != "" {
		module := styles.StyleKey.Render(fmt.Sprintf("[%s]", r.Module))
		row += " " + module
	}
	return row
}

func (e *Extension) renderDetail(width, height int) string {
	title := styles.StyleTitle.Render("Resource Detail")
	address := styles.StyleKey.Render(e.detailAddr)

	// Truncate detail to visible area
	lines := strings.Split(e.detail, "\n")
	maxLines := height - 6
	if maxLines < 5 {
		maxLines = 5
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, styles.StyleFaint.Render("... (truncated)"))
	}

	detail := strings.Join(lines, "\n")
	hint := styles.StyleFaintItalic.Render("Esc/q to go back")

	content := title + "\n" + address + "\n\n" + detail + "\n\n" + hint
	return styles.StylePadded.Render(content)
}
