package state

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
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

// StateDeletedMsg is sent when a resource is successfully removed from state.
type StateDeletedMsg struct {
	Address string
}

// StateMovedMsg is sent when a resource is successfully moved.
type StateMovedMsg struct {
	Source string
	Dest   string
}

// StateTaintedMsg is sent when resources are successfully tainted.
type StateTaintedMsg struct {
	Addresses []string
}

// StateUntaintedMsg is sent when resources are successfully untainted.
type StateUntaintedMsg struct {
	Addresses []string
}

// StateImportedMsg is sent when a resource is successfully imported.
type StateImportedMsg struct {
	Address string
	ID      string
}

// StateEditMsg requests the editor to open for this resource.
type StateEditMsg struct {
	Address   string
	Addresses []string // if set, open multiple files
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
	guard         *sdk.ScopeGuard
	pins          *sdk.PinService
	fuzzy         *ui.FuzzyFilter[sdk.Resource]
	status        Status
	resources     []sdk.Resource
	filtered      []sdk.Resource
	tree          *tree.Tree
	treeMode      bool
	filterScores  map[string]int
	filter        string
	filtering     bool
	errMsg        string
	lockInfo      *sdk.StateLock
	viewWidth     int
	listHScroll   int
	listWrap      bool
	pinnedOnly    bool
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
		svc:   svc,
		log:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		tree:  tree.New(nil),
		fuzzy: ui.NewFuzzyFilter(func(r sdk.Resource) string { return r.Address }),
	}
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	return p
}

func (e *Plugin) ID() string          { return "state" }
func (e *Plugin) Name() string        { return "State Browser" }
func (e *Plugin) Description() string { return "Browse and inspect terraform state resources" }
func (e *Plugin) Ready() bool         { return e.status == StatusDone || e.status == StatusShowingDetail }
func (e *Plugin) Status() Status      { return e.status }
func (e *Plugin) Selected() int       { return e.tree.Cursor() }
func (e *Plugin) Filter() string      { return e.filter }
func (e *Plugin) Filtering() bool     { return e.filtering }
func (e *Plugin) ResourceCount() int  { return len(e.filtered) }
func (e *Plugin) TotalCount() int     { return len(e.resources) }
func (e *Plugin) Count() (int, int)   { return len(e.filtered), len(e.resources) }
func (e *Plugin) Stack() *sdk.Stack   { return e.stack }

func (e *Plugin) PinnedCount() int {
	if e.pins != nil {
		return e.pins.Count()
	}
	return 0
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context. Does not auto-load state.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	e.log = ctx.Logger
	e.session = ctx.Session
	e.guard = sdk.NewScopeGuard(ctx.Session, ctx.Service)
	e.pins = sdk.NewPinService(ctx.Session)
	e.stack.Clear()
	e.reset()
	return nil
}

// reset clears all plugin state to initial values.
func (e *Plugin) reset() {
	e.status = StatusIdle
	e.resources = nil
	e.filtered = nil
	e.tree = tree.New(nil)
	e.filter = ""
	e.filtering = false
	e.errMsg = ""
	e.detail = ""
	e.detailAddr = ""
	e.fuzzy.SetItems(nil)
}

// Activate triggers state loading when the user enters the plugin.
func (e *Plugin) Activate() tea.Cmd {
	// Sync guard with any externally-set scope (e.g., from prior activation)
	if e.scopedContext != "" && e.guard.CurrentScope() == "" {
		e.guard.SetTracked(e.scopedContext)
	}

	scopeStatus, svc := e.guard.Check()
	switch scopeStatus {
	case sdk.ScopeChanged:
		e.svc = svc
		e.scopedContext = e.guard.CurrentScope()
		e.reset()
		e.status = StatusLoading
		return e.loadState()
	case sdk.ScopeRequired:
		e.status = StatusError
		e.errMsg = "Select a context first (press c)"
		return nil
	}

	if e.status == StatusIdle || e.status == StatusError {
		if e.session != nil {
			if dir, ok := sdk.GetTyped[string](e.session, sdk.SessionKeyActiveScopeAbs); ok && dir != "" {
				e.svc = e.svc.WithDir(dir)
				e.scopedContext = dir
			}
		}
		e.status = StatusLoading
		return e.loadState()
	}
	return nil
}

// Refresh reloads the state.
func (e *Plugin) Refresh() tea.Cmd {
	e.reset()
	e.status = StatusLoading
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
			e.fuzzy.SetItems(msg.Resources)
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

	case StateMovedMsg:
		e.log.Debug("state.moved", "source", msg.Source, "dest", msg.Dest)
		return e, e.Refresh()

	case StateTaintedMsg:
		e.log.Debug("state.tainted", "addresses", msg.Addresses)
		return e, e.Refresh()

	case StateUntaintedMsg:
		e.log.Debug("state.untainted", "addresses", msg.Addresses)
		return e, e.Refresh()

	case StateImportedMsg:
		e.log.Debug("state.imported", "address", msg.Address, "id", msg.ID)
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

func (e *Plugin) panListRight() {
	e.listHScroll += 10
}

func (e *Plugin) panListLeft() {
	e.listHScroll -= 10
	if e.listHScroll < 0 {
		e.listHScroll = 0
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

// sourceResources returns the base set of resources to filter from,
// applying the pinnedOnly pre-filter if active.
func (e *Plugin) sourceResources() []sdk.Resource {
	if !e.pinnedOnly {
		return e.resources
	}
	var result []sdk.Resource
	for _, r := range e.resources {
		if e.isPinnedAddress(r.Address) {
			result = append(result, r)
		}
	}
	return result
}

// SetFilter filters resources. In tree mode, uses fzf matching preserving original order.
// In flat mode, uses fzf fuzzy matching sorted by score.
func (e *Plugin) SetFilter(filter string) {
	e.filter = filter
	source := e.sourceResources()
	if filter == "" {
		e.filtered = source
		e.filterScores = nil
		e.rebuildTree()
		e.log.Debug("state.filter", "filter", "", "results", len(source))
		return
	}

	e.fuzzy.SetItems(source)
	e.fuzzy.SetQuery(filter)
	if e.treeMode {
		e.filtered = e.fuzzy.OriginalOrder()
	} else {
		e.filtered = e.fuzzy.Results()
	}

	e.filterScores = make(map[string]int)
	ordered := e.fuzzy.OriginalOrder()
	for i, r := range ordered {
		e.filterScores[r.Address] = e.fuzzy.ScoreAt(i)
	}

	e.rebuildTree()
	e.log.Debug("state.filter", "filter", filter, "results", len(e.filtered))
}

// rebuildTree reconstructs the tree from filtered resources, syncing pinned state.
func (e *Plugin) rebuildTree() {
	items := make([]tree.Item, len(e.filtered))
	for i, r := range e.filtered {
		items[i] = resourceItem{resource: r}
	}
	if e.treeMode {
		e.tree = tree.New(items)
	} else {
		e.tree = tree.New(items, tree.WithSplitFunc(func(addr string) []string {
			return []string{addr}
		}), tree.WithPreserveOrder())
	}
	e.syncPinnedToTree()
	if e.treeMode && e.filter != "" {
		e.tree.ExpandAll()
	}
}

// syncPinnedToTree updates the tree's pinned set from the PinService or session.
func (e *Plugin) syncPinnedToTree() {
	if e.pins != nil {
		e.tree.SetPinned(e.pins.All())
		return
	}
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

// InspectSelected loads detail for the selected resource.
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
		return sdk.StyleFaintItalic.Render("Loading state...")

	case StatusLoading:
		msg := "Loading terraform state..."
		if e.errMsg != "" {
			msg = e.errMsg
		}
		return sdk.StyleFaintItalic.Render(msg)

	case StatusError:
		if e.lockInfo != nil {
			lockPanel := sdk.FormatLockInfo(e.lockInfo)
			hint := sdk.StyleFaintItalic.Render("u force-unlock")
			return lockPanel + "\n\n" + hint
		}
		return sdk.StyleError.Render("Error: " + e.errMsg)

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

	filterLine := ""
	if e.filtering {
		filterLine = sdk.StyleKey.Render("/ ") + e.filter + "█"
		if e.pinnedOnly {
			filterLine += " " + sdk.StyleSuccess.Render("[pinned]")
		}
		filterLine += "\n\n"
	} else if e.filter != "" || e.pinnedOnly {
		parts := []string{}
		if e.filter != "" {
			parts = append(parts, sdk.StyleKey.Render("filter: ")+e.filter)
		}
		if e.pinnedOnly {
			parts = append(parts, sdk.StyleSuccess.Render("[pinned]"))
		}
		filterLine = strings.Join(parts, " ") + "\n\n"
	}

	if len(e.filtered) == 0 {
		noResources := sdk.StyleFaintItalic.Render("No resources found.")
		return filterLine + noResources
	}

	filterHeight := 0
	if e.filtering || e.filter != "" {
		filterHeight = 2
	}
	maxVisible := height - filterHeight
	if maxVisible < 3 {
		maxVisible = 3
	}

	contentWidth := width - 6

	var treeContent string
	if e.treeMode {
		treeContent = e.tree.Render(tree.RenderOpts{
			Width:  contentWidth,
			Height: maxVisible,
			RenderLeaf: func(node *tree.Node, pinned bool) string {
				r := node.Item.(resourceItem).resource
				full := node.Label + "  " + r.Type
				if e.filter != "" {
					if score, ok := e.filterScores[r.Address]; ok {
						full += fmt.Sprintf(" [%d]", score)
					}
				}
				if e.listHScroll > 0 {
					if e.listHScroll < len(full) {
						full = full[e.listHScroll:]
					} else {
						full = ""
					}
				}
				return full
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
			PinIndicators: &tree.PinIndicators{
				None:    "[ ] ",
				Full:    sdk.StyleSuccess.Render("[*] "),
				Partial: sdk.StyleUpdate.Render("[-] "),
			},
			SelectedStyle: func(s string, w int) string {
				return sdk.StyleSelected.Width(w).Render(s)
			},
			TruncateRow: func(s string, w int) string {
				if e.listWrap {
					return s
				}
				return lipgloss.NewStyle().MaxWidth(w).Render(s)
			},
		})
	} else {
		treeContent = e.renderFlatList(contentWidth, maxVisible)
	}

	return filterLine + treeContent
}

func (e *Plugin) renderFlatList(contentWidth, maxVisible int) string {
	var b strings.Builder
	cursor := e.tree.Cursor()
	startIdx := e.tree.ViewOffset(maxVisible)
	endIdx := startIdx + maxVisible
	if endIdx > len(e.filtered) {
		endIdx = len(e.filtered)
	}

	linesUsed := 0
	for i := startIdx; i < endIdx; i++ {
		r := e.filtered[i]
		pinMark := "[ ] "
		if e.isPinnedAddress(r.Address) {
			pinMark = sdk.StyleSuccess.Render("[*] ")
		}
		row := e.formatResourceRow(pinMark, r, contentWidth)
		if i == cursor {
			if e.listWrap {
				row = sdk.StyleSelected.Width(contentWidth).Render(row)
			} else {
				row = sdk.StyleSelected.MaxWidth(contentWidth).Width(contentWidth).Render(row)
			}
		}
		rowLines := strings.Count(row, "\n") + 1
		if e.listWrap && linesUsed+rowLines > maxVisible {
			break
		}
		if i > startIdx {
			b.WriteByte('\n')
		}
		b.WriteString(row)
		linesUsed += rowLines
	}
	return b.String()
}

func (e *Plugin) formatResourceRow(pinMark string, r sdk.Resource, contentWidth int) string {
	full := r.Address + "  " + r.Type
	if e.listWrap {
		return pinMark + full
	}
	if e.listHScroll > 0 {
		if e.listHScroll < len(full) {
			full = full[e.listHScroll:]
		} else {
			full = ""
		}
	}
	availWidth := contentWidth - 4 // pin mark visual width
	if len(full) > availWidth {
		full = full[:availWidth]
	}
	return pinMark + full
}

func (e *Plugin) renderDetail(width, height int) string {
	e.viewWidth = width
	address := sdk.StyleKey.Render(e.detailAddr)

	headerLines := 2
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

	maxLines := height - headerLines
	if maxLines < 5 {
		maxLines = 5
	}

	maxScroll := len(lines) - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.detailScroll > maxScroll {
		e.detailScroll = maxScroll
	}

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

	pinIndicator := ""
	if e.isPinnedAddress(e.detailAddr) {
		pinIndicator = " " + sdk.StyleSuccess.Render("[pinned]")
	}

	content := address + pinIndicator + scrollInfo + "\n\n" + detail
	return content
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

func (e *Plugin) clearAllPins() {
	if e.pins != nil {
		e.pins.Set(nil)
	}
	e.tree.SetPinned(nil)
	if e.pinnedOnly {
		e.pinnedOnly = false
		e.SetFilter(e.filter)
	}
	e.log.Debug("state.pin.clear-all")
}

func (e *Plugin) togglePin(address string) tea.Cmd {
	if e.session == nil {
		return nil
	}
	e.tree.TogglePin()
	e.pins.Set(e.tree.PinnedPaths())
	e.log.Debug("state.pin.toggle", "address", address, "pinned_count", e.pins.Count())
	return nil
}

func (e *Plugin) isPinnedAddress(address string) bool {
	if e.pins != nil {
		return e.pins.IsPinned(address)
	}
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

func (e *Plugin) requestEditMultiple(addresses []string) tea.Cmd {
	return func() tea.Msg {
		return StateEditMsg{Addresses: addresses}
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
