package state

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui/tree"
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

// ForceUnlockResultMsg is sent when a force-unlock operation completes.
type ForceUnlockResultMsg struct {
	Err error
}

// resourceItem wraps sdk.Resource to implement tree.Item.
type resourceItem struct {
	resource sdk.Resource
}

func (r resourceItem) Address() string { return r.resource.Address }

// Plugin implements the state browser feature.
type Plugin struct {
	svc           sdk.Service
	log           *slog.Logger
	session       *sdk.Session
	stack         *sdk.Stack
	status        Status
	resources     []sdk.Resource
	filtered      []sdk.Resource
	tree          *tree.Tree
	filter        string
	filtering     bool
	errMsg        string
	lockInfo      *sdk.StateLock
	viewWidth     int
	detail        string
	detailAddr    string
	detailScroll  int
	detailHScroll int
	detailWrap    bool
	scopedContext string
}

// New creates a new state browser plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		svc:  svc,
		log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		tree: tree.New(nil),
	}
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	return p
}

func (e *Plugin) ID() string          { return "state" }
func (e *Plugin) Name() string        { return "State Browser" }
func (e *Plugin) Description() string { return "Browse and inspect terraform state resources" }
func (e *Plugin) KeyBinding() string  { return "s" }
func (e *Plugin) Ready() bool         { return e.status == StatusDone || e.status == StatusShowingDetail }
func (e *Plugin) Status() Status      { return e.status }
func (e *Plugin) Selected() int       { return e.tree.Cursor() }
func (e *Plugin) Filter() string      { return e.filter }
func (e *Plugin) Filtering() bool     { return e.filtering }
func (e *Plugin) ResourceCount() int  { return len(e.filtered) }
func (e *Plugin) TotalCount() int     { return len(e.resources) }
func (e *Plugin) Stack() *sdk.Stack   { return e.stack }

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context. Does not auto-load state.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	e.log = ctx.Logger
	e.session = ctx.Session
	e.stack.Clear()
	e.status = StatusIdle
	e.resources = nil
	e.filtered = nil
	e.tree = tree.New(nil)
	e.filter = ""
	e.filtering = false
	e.errMsg = ""
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
			e.tree = tree.New(nil)
			e.filter = ""
			e.filtering = false
			e.errMsg = ""
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
		return e.loadState()
	}
	return nil
}

// Refresh reloads the state.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = StatusLoading
	e.resources = nil
	e.filtered = nil
	e.tree = tree.New(nil)
	e.filter = ""
	e.filtering = false
	e.errMsg = ""
	e.lockInfo = nil
	e.detail = ""
	e.detailAddr = ""
	if e.stack != nil {
		e.stack.Clear()
	}
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
			e.lockInfo = sdk.ParseLockError(e.errMsg)
			e.log.Debug("state.load.error", "error", msg.Err.Error())
		} else {
			e.status = StatusDone
			e.resources = msg.Resources
			e.filtered = msg.Resources
			e.rebuildTree()
			e.log.Debug("state.load.complete", "resources", len(msg.Resources))
		}
		return e, nil

	case ForceUnlockResultMsg:
		if msg.Err != nil {
			e.errMsg = fmt.Sprintf("Force-unlock failed: %s", msg.Err.Error())
			e.lockInfo = nil
			e.log.Debug("state.force-unlock.error", "error", msg.Err.Error())
		} else {
			e.lockInfo = nil
			e.log.Debug("state.force-unlock.success")
			return e, e.Refresh()
		}
		return e, nil

	case StateDeletedMsg:
		e.log.Debug("state.deleted", "address", msg.Address)
		return e, e.Refresh()

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
			e.stack.Push(&detailFrame{plugin: e})
		}
		return e, nil

	case tea.KeyMsg:
		cmd := e.stack.Update(msg)
		return e, cmd
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	return e.stack.Update(msg)
}

// MoveUp moves selection up.
func (e *Plugin) MoveUp() {
	e.tree.MoveUp()
}

// MoveDown moves selection down.
func (e *Plugin) MoveDown() {
	e.tree.MoveDown()
}

// MoveToStart moves selection to the first item.
func (e *Plugin) MoveToStart() {
	e.tree.MoveToStart()
}

// MoveToEnd moves selection to the last item.
func (e *Plugin) MoveToEnd() {
	e.tree.MoveToEnd()
}

func (e *Plugin) navigate(dir int) {
	if dir > 0 {
		e.MoveDown()
	} else {
		e.MoveUp()
	}
}

func (e *Plugin) panDetailRight() {
	maxLine := 0
	for _, line := range strings.Split(e.detail, "\n") {
		if len(line) > maxLine {
			maxLine = len(line)
		}
	}
	contentWidth := e.viewWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}
	maxScroll := maxLine - contentWidth
	if maxScroll < 0 {
		maxScroll = 0
	}
	e.detailHScroll += 10
	if e.detailHScroll > maxScroll {
		e.detailHScroll = maxScroll
	}
}

func (e *Plugin) panDetailLeft() {
	e.detailHScroll -= 10
	if e.detailHScroll < 0 {
		e.detailHScroll = 0
	}
}

// SetFilter filters resources using fzf fuzzy matching against the full address.
// Space-separated terms use AND logic. Results sorted by score (best first).
func (e *Plugin) SetFilter(filter string) {
	e.filter = filter
	if filter == "" {
		e.filtered = e.resources
		e.rebuildTree()
		e.log.Debug("state.filter", "filter", "", "results", len(e.resources))
		return
	}
	terms := strings.Fields(strings.ToLower(filter))
	type scored struct {
		resource sdk.Resource
		score    int
	}
	var results []scored
	slab := util.MakeSlab(100*1024, 2048)
	for _, r := range e.resources {
		input := util.RunesToChars([]rune(strings.ToLower(r.Address)))
		totalScore := 0
		matched := true
		for _, term := range terms {
			res, _ := algo.FuzzyMatchV2(false, true, true, &input, []rune(term), false, slab)
			if res.Score <= 0 {
				matched = false
				break
			}
			totalScore += res.Score
		}
		if matched {
			results = append(results, scored{r, totalScore})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	e.filtered = make([]sdk.Resource, len(results))
	for i, r := range results {
		e.filtered[i] = r.resource
	}
	e.rebuildTree()
	e.log.Debug("state.filter", "filter", filter, "results", len(e.filtered))
}

// rebuildTree reconstructs the tree from filtered resources, syncing pinned state.
// When a filter is active, all branches are auto-expanded to reveal matches.
func (e *Plugin) rebuildTree() {
	items := make([]tree.Item, len(e.filtered))
	for i, r := range e.filtered {
		items[i] = resourceItem{resource: r}
	}
	e.tree.SetItems(items)
	e.syncPinnedToTree()
	if e.filter != "" {
		e.tree.ExpandAll()
	}
}

// syncPinnedToTree updates the tree's pinned set from session state.
func (e *Plugin) syncPinnedToTree() {
	if e.session == nil {
		return
	}
	pinned, _ := sdk.GetTyped[[]string](e.session, "terraform.pinned")
	e.tree.SetPinned(pinned)
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

// SelectedResource returns the currently selected resource (empty if on branch node).
func (e *Plugin) SelectedResource() sdk.Resource {
	item := e.tree.CursorItem()
	if item != nil {
		return item.(resourceItem).resource
	}
	return sdk.Resource{}
}

// CursorNode returns the tree node at the cursor position.
func (e *Plugin) CursorNode() *tree.Node {
	return e.tree.CursorNode()
}

// InspectSelected loads detail for the selected resource (enter = inspect, like k9s).
func (e *Plugin) InspectSelected() tea.Cmd {
	if e.status == StatusLoading {
		return nil
	}
	r := e.SelectedResource()
	if r.Address == "" {
		e.log.Debug("state.inspect.skip", "reason", "empty address", "cursor", e.tree.Cursor(), "filtered", len(e.filtered))
		return nil
	}
	e.log.Debug("state.inspect.start", "address", r.Address)
	e.status = StatusLoading
	e.filtering = false
	// Pop the filter frame if active (inspect overrides filter)
	if e.stack.Peek() != nil && e.stack.Peek().ID() == "filter" {
		e.stack.Pop()
	}
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
		hint := sdk.StyleFaintItalic.Render("q to go back")
		return sdk.StylePadded.Render(title + "\n\n" + loading + "\n\n" + hint)

	case StatusError:
		title := sdk.StyleTitle.Render("State Browser")
		if e.lockInfo != nil {
			lockPanel := sdk.FormatLockInfo(e.lockInfo)
			hint := sdk.StyleFaintItalic.Render("u force-unlock  r retry  q back")
			return sdk.StylePadded.Render(title + "\n\n" + lockPanel + "\n" + hint)
		}
		errText := sdk.StyleError.Render("Error: " + e.errMsg)
		hint := sdk.StyleFaintItalic.Render("Press r to retry, q to go back")
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
	e.viewWidth = width
	title := sdk.StyleTitle.Render("State Browser")

	filterLine := ""
	if e.filtering {
		filterLine = sdk.StyleKey.Render("/ ") + e.filter + "█\n\n"
	} else if e.filter != "" {
		filterLine = sdk.StyleKey.Render("filter: ") + e.filter + "\n\n"
	}

	if e.tree.VisibleCount() == 0 {
		noResources := sdk.StyleFaintItalic.Render("No resources found.")
		return sdk.StylePadded.Render(title + "\n\n" + filterLine + noResources)
	}

	// Calculate visible area
	maxVisible := height - 7
	if maxVisible < 3 {
		maxVisible = 3
	}

	contentWidth := width - 6

	treeContent := e.tree.Render(tree.RenderOpts{
		Width:  contentWidth,
		Height: maxVisible,
		RenderLeaf: func(node *tree.Node, pinned bool) string {
			r := node.Item.(resourceItem).resource
			typeInfo := sdk.StyleFaint.Render(r.Type)
			return fmt.Sprintf("%s  %s", node.Label, typeInfo)
		},
		RenderBranch: func(node *tree.Node, pinned bool) string {
			indicator := "▶"
			if node.Expanded {
				indicator = "▼"
			}
			path := sdk.StyleKey.Render(node.Label)
			count := sdk.StyleFaint.Render(fmt.Sprintf(" (%d)", node.Count))
			return fmt.Sprintf("%s %s%s", indicator, path, count)
		},
		PinIndicator: sdk.StyleSuccess.Render("* "),
		SelectedStyle: func(s string, w int) string {
			return sdk.StyleSelected.Width(w).Render(s)
		},
	})

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
		hint = sdk.StyleFaintItalic.Render(fmt.Sprintf("Type to filter (space=AND)  ←→ pan  ^w wrap(%s)  Esc exit", wrapLabel))
	} else {
		hint = sdk.StyleFaintItalic.Render(fmt.Sprintf("↑↓ navigate  ←→ pan  Enter inspect  Space pin  d delete  e edit  / filter  ^w wrap(%s)", wrapLabel))
	}

	content := title + "\n\n" + filterLine + treeContent + "\n\n" + count + "\n" + hint
	return sdk.StylePadded.Render(content)
}

func (e *Plugin) renderResourceRow(r sdk.Resource) string {
	typeInfo := sdk.StyleFaint.Render(r.Type)
	row := fmt.Sprintf("%s  %s", r.Address, typeInfo)
	if r.Module != "" {
		module := sdk.StyleKey.Render(fmt.Sprintf("[%s]", r.Module))
		row += " " + module
	}
	return row
}

func (e *Plugin) renderDetail(width, height int) string {
	e.viewWidth = width
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
	} else {
		for i, line := range lines {
			if e.detailHScroll < len(line) {
				lines[i] = line[e.detailHScroll:]
			} else {
				lines[i] = ""
			}
			if len(lines[i]) > contentWidth {
				lines[i] = lines[i][:contentWidth]
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

	pinIndicator := ""
	if e.session != nil && e.isPinnedAddress(e.detailAddr) {
		pinIndicator = " " + sdk.StyleSuccess.Render("[pinned]")
	}

	hint := sdk.StyleFaintItalic.Render(fmt.Sprintf("↑↓ scroll  ←→ pan  ^w wrap(%s)  Space pin  d delete  e edit  Esc back", wrapIndicator))

	content := title + "\n" + address + pinIndicator + scrollInfo + "\n\n" + detail + "\n\n" + hint
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

// --- Context actions ---

// StateDeletedMsg is sent when a resource is successfully removed from state.
type StateDeletedMsg struct {
	Address string
}

// StateEditMsg requests the editor to open for this resource.
type StateEditMsg struct {
	Address string
}

func (e *Plugin) togglePin(address string) tea.Cmd {
	if e.session == nil {
		return nil
	}
	pinned, _ := sdk.GetTyped[[]string](e.session, "terraform.pinned")
	for i, a := range pinned {
		if a == address {
			pinned = append(pinned[:i], pinned[i+1:]...)
			e.session.Set("terraform.pinned", pinned)
			e.log.Debug("state.unpin", "address", address)
			e.tree.SetPinned(pinned)
			return nil
		}
	}
	pinned = append(pinned, address)
	e.session.Set("terraform.pinned", pinned)
	e.log.Debug("state.pin", "address", address)
	e.tree.SetPinned(pinned)
	return nil
}

func (e *Plugin) isPinnedAddress(address string) bool {
	if e.session == nil {
		return false
	}
	pinned, _ := sdk.GetTyped[[]string](e.session, "terraform.pinned")
	for _, a := range pinned {
		if a == address {
			return true
		}
	}
	return false
}

func (e *Plugin) requestDelete(address string) tea.Cmd {
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Remove %s from state?", address),
				func() tea.Cmd {
					return func() tea.Msg {
						err := svc.StateRm(context.Background(), address)
						if err != nil {
							log.Debug("state.rm.error", "address", address, "error", err.Error())
							return StateListMsg{Err: err}
						}
						log.Debug("state.rm.success", "address", address)
						return StateDeletedMsg{Address: address}
					}
				},
			),
		}
	}
}

func (e *Plugin) requestEdit(address string) tea.Cmd {
	return func() tea.Msg {
		return StateEditMsg{Address: address}
	}
}

func (e *Plugin) requestForceUnlock() tea.Cmd {
	lockID := e.lockInfo.ID
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Force-unlock %s? This is dangerous if another operation is running.", lockID),
				func() tea.Cmd {
					return func() tea.Msg {
						err := svc.ForceUnlock(context.Background(), lockID)
						if err != nil {
							log.Debug("state.force-unlock.error", "lockID", lockID, "error", err.Error())
						} else {
							log.Debug("state.force-unlock.success", "lockID", lockID)
						}
						return ForceUnlockResultMsg{Err: err}
					}
				},
			),
		}
	}
}
