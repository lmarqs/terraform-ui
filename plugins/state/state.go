package state

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui/tree"
)

const StatusShowingDetail = sdk.Status(10)

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

// StateDeletedMsg is sent when a resource is successfully removed from state.
type StateDeletedMsg struct {
	Address string
}

// StateMovedMsg is sent when a resource is successfully moved.
type StateMovedMsg struct {
	Source string
	Dest   string
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
	stack         *sdk.Stack
	getCtx        func() *sdk.Context
	pinFn         func(string) tea.Cmd
	clearPinsFn   func() tea.Cmd
	fuzzy         *ui.FuzzyFilter[sdk.Resource]
	timer         ui.Timer
	status        sdk.Status
	resources     []sdk.Resource
	filtered      []sdk.Resource
	tree          *tree.Tree
	treeMode      bool
	filterScores  map[string]int
	filter        string
	filtering     bool
	mutating      bool
	errMsg        string
	lockInfo      *sdk.StateLock
	viewWidth     int
	listPanel     *ui.ContentPanel
	pinnedOnly    bool
	detail        string
	detailAddr    string
	detailScroll  int
	detailPanel   *ui.ContentPanel
	cancelFn      context.CancelFunc
}

// New creates a new state browser plugin.
func New(svc sdk.Service) sdk.Plugin {
	listPanel := ui.NewContentPanel()
	listPanel.SelectedStyle = func(s string, w int) string {
		return sdk.StyleSelected.Width(w).Render(s)
	}
	detailPanel := ui.NewContentPanel()

	p := &Plugin{
		svc:         svc,
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		tree:        tree.New(nil),
		listPanel:   listPanel,
		detailPanel: detailPanel,
		fuzzy:       ui.NewFuzzyFilter(func(r sdk.Resource) string { return r.Address }),
	}
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	return p
}

func (e *Plugin) ID() string          { return "state" }
func (e *Plugin) Name() string        { return "State Browser" }
func (e *Plugin) Description() string { return "Browse and inspect terraform state resources" }
func (e *Plugin) Ready() bool         { return e.status == sdk.StatusDone || e.status == StatusShowingDetail }
func (e *Plugin) Status() sdk.Status  { return e.status }
func (e *Plugin) Busy() bool          { return e.mutating }
func (e *Plugin) Selected() int       { return e.tree.Cursor() }
func (e *Plugin) Filter() string      { return e.filter }
func (e *Plugin) Filtering() bool     { return e.filtering }
func (e *Plugin) ResourceCount() int  { return len(e.filtered) }
func (e *Plugin) TotalCount() int     { return len(e.resources) }
func (e *Plugin) Count() (int, int)   { return len(e.filtered), len(e.resources) }
func (e *Plugin) Stack() *sdk.Stack   { return e.stack }

func (e *Plugin) PinnedCount() int { return len(e.pinnedAddresses()) }

func (e *Plugin) pinnedAddresses() []string {
	return sdk.PinnedAddresses(e.getCtx)
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init wires the plugin to its shared dependencies. Does not auto-load state.
func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	e.svc = deps.Service
	e.log = deps.Logger
	e.getCtx = deps.Context
	e.pinFn = deps.Pin
	e.clearPinsFn = deps.ClearPins
	e.stack.Clear()
	e.reset()
	return nil
}

// HandlePlanInvalidated implements sdk.PlanInvalidatedHandler.
func (e *Plugin) HandlePlanInvalidated(_ sdk.PlanInvalidatedEvent) tea.Cmd {
	e.reset()
	return nil
}

// HandleLockCleared implements sdk.LockClearedHandler.
func (e *Plugin) HandleLockCleared(_ sdk.LockClearedEvent) tea.Cmd {
	e.lockInfo = nil
	e.reset()
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler. Pins are scoped
// to the active Context — they die on context switch (the very bug this
// overhaul exists to fix).
func (e *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if ev.Next == nil {
		return nil
	}
	if ev.Next.Service != nil {
		e.svc = ev.Next.Service
	}
	e.clearAllPins()
	e.reset()
	return nil
}

// reset clears all plugin state to initial values.
func (e *Plugin) reset() {
	e.status = sdk.StatusIdle
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
	if e.status == sdk.StatusIdle || e.status == sdk.StatusError {
		e.status = sdk.StatusLoading
		return tea.Batch(e.loadState(), e.timer.Start())
	}
	if e.status == sdk.StatusLoading && e.timer.Running() {
		return e.timer.Tick()
	}
	return nil
}

// Refresh reloads the state.
func (e *Plugin) Refresh() tea.Cmd {
	e.reset()
	e.status = sdk.StatusLoading
	if e.stack != nil {
		e.stack.Clear()
	}
	return tea.Batch(e.loadState(sdk.SkipCache()), e.timer.Start())
}

// Cancel aborts any in-flight terraform operation.
func (e *Plugin) Cancel() {
	if e.cancelFn != nil {
		e.cancelFn()
		e.cancelFn = nil
	}
}

func (e *Plugin) loadState(opts ...sdk.StateListOption) tea.Cmd {
	e.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel
	svc := e.svc
	return func() tea.Msg {
		resources, err := svc.StateList(ctx, opts...)
		return StateListMsg{Resources: resources, Err: err}
	}
}

func (e *Plugin) loadDetail(address string) tea.Cmd {
	e.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel
	svc := e.svc
	return func() tea.Msg {
		detail, err := svc.Show(ctx, address)
		return ResourceDetailMsg{Address: address, Detail: detail, Err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return e, e.timer.Tick()

	case StateListMsg:
		e.timer.Stop()
		e.mutating = false
		if msg.Err != nil {
			e.status = sdk.StatusError
			e.errMsg = msg.Err.Error()
			e.lockInfo = sdk.ParseLockError(e.errMsg)
			e.log.Debug("state.load.error", "error", msg.Err.Error())
			if e.lockInfo != nil {
				return e, func() tea.Msg { return sdk.LockDetectedEvent{Lock: e.lockInfo} }
			}
		} else {
			e.status = sdk.StatusDone
			e.resources = msg.Resources
			e.filtered = msg.Resources
			e.fuzzy.SetItems(msg.Resources)
			e.rebuildTree()
			e.log.Debug("state.load.complete", "resources", len(msg.Resources))
			return e, func() tea.Msg { return sdk.StateRefreshedEvent{} }
		}
		return e, nil

	case StateDeletedMsg:
		e.mutating = false
		e.log.Debug("state.deleted", "address", msg.Address)
		return e, e.Refresh()

	case StateMovedMsg:
		e.mutating = false
		e.log.Debug("state.moved", "source", msg.Source, "dest", msg.Dest)
		return e, e.Refresh()

	case ResourceDetailMsg:
		e.timer.Stop()
		if msg.Err != nil {
			e.errMsg = msg.Err.Error()
			e.status = sdk.StatusDone
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

func (e *Plugin) panListLeft() {
	e.listPanel.HandleKey(tea.KeyMsg{Type: tea.KeyLeft})
}

func (e *Plugin) panDetailRight() {
	e.detailPanel.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
}

func (e *Plugin) panDetailLeft() {
	e.detailPanel.HandleKey(tea.KeyMsg{Type: tea.KeyLeft})
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

// syncPinnedToTree updates the tree's pinned set from the active Context.
func (e *Plugin) syncPinnedToTree() {
	e.tree.SetPinned(e.pinnedAddresses())
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
	if e.status == sdk.StatusLoading {
		return nil
	}
	r := e.SelectedResource()
	if r.Address == "" {
		e.log.Debug("state.inspect.skip", "reason", "empty address", "cursor", e.tree.Cursor(), "filtered", len(e.filtered))
		return nil
	}
	e.log.Debug("state.inspect.start", "address", r.Address)
	e.status = sdk.StatusLoading
	e.filtering = false
	if e.stack.Peek() != nil && e.stack.Peek().ID() == "filter" {
		e.stack.Pop()
	}
	e.errMsg = "Loading " + r.Address + "..."
	return tea.Batch(e.loadDetail(r.Address), e.timer.Start())
}

// View renders the state browser plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Loading state...")

	case sdk.StatusLoading:
		msg := "Loading terraform state..."
		if e.errMsg != "" {
			msg = e.errMsg
		}
		return sdk.StyleFaintItalic.Render(msg + " " + e.timer.FormatElapsed())

	case sdk.StatusError:
		if e.lockInfo != nil {
			actions := []ui.ActionChip{{Key: "u", Label: "force-unlock"}}
			return sdk.FormatLockInfo(e.lockInfo) + ui.RenderActionsBar(actions, width)
		}
		return sdk.StyleError.Render("Error: " + e.errMsg)

	case StatusShowingDetail:
		return e.renderDetail(width, height)

	case sdk.StatusDone:
		return e.renderResources(width, height)

	default:
		return ""
	}
}

func (e *Plugin) CursorPosition() (int, int) {
	if e.status != sdk.StatusDone || len(e.filtered) == 0 {
		return 0, 0
	}
	return e.tree.Cursor() + 1, e.tree.VisibleCount()
}

func (e *Plugin) listActions() []ui.ActionChip {
	chips := []ui.ActionChip{
		{Key: "d", Label: "delete"},
		{Key: "e", Label: "edit"},
		{Key: "t", Label: "taint"},
		{Key: "T", Label: "untaint"},
	}
	if e.PinnedCount() > 0 {
		chips = append(chips, ui.ActionChip{Key: "!", Label: "batch"})
	}
	return chips
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
			parts = append(parts, sdk.StyleKey.Render("ᗊ: ")+e.filter)
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

	actions := e.listActions()

	filterHeight := 0
	if e.filtering || e.filter != "" {
		filterHeight = 2
	}
	maxVisible := ui.HeightBudget(height, filterHeight, ui.ActionsBarHeight)

	hasGutter := ui.NeedsGutter(e.tree.VisibleCount(), maxVisible)
	contentWidth := ui.ContentWidth(width, hasGutter)

	var rows []string
	if e.treeMode {
		rows = e.tree.RenderRows(tree.RenderOpts{
			Width:  contentWidth,
			Height: maxVisible,
			RenderLeaf: func(node *tree.Node, pinned bool) string {
				r := node.Item.(resourceItem).resource
				full := node.Label + "  " + r.Type
				if r.Tainted {
					full += " " + sdk.StyleUpdate.Render("[tainted]")
				}
				if e.filter != "" {
					if score, ok := e.filterScores[r.Address]; ok {
						full += fmt.Sprintf(" [%d]", score)
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
		})
	} else {
		rows = e.buildFlatRows(maxVisible)
	}

	viewOffset := e.tree.ViewOffset(maxVisible)
	treeContent := e.listPanel.Render(ui.RenderParams{
		Width:        width,
		Height:       maxVisible,
		TotalItems:   e.tree.VisibleCount(),
		Cursor:       e.tree.Cursor() - viewOffset,
		ScrollOffset: viewOffset,
		Rows:         rows,
	})

	return filterLine + treeContent + ui.RenderActionsBar(actions, width)
}

func (e *Plugin) buildFlatRows(maxVisible int) []string {
	startIdx := e.tree.ViewOffset(maxVisible)
	endIdx := startIdx + maxVisible
	if endIdx > len(e.filtered) {
		endIdx = len(e.filtered)
	}

	rows := make([]string, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		r := e.filtered[i]
		pinMark := "[ ] "
		if e.isPinnedAddress(r.Address) {
			pinMark = sdk.StyleSuccess.Render("[*] ")
		}
		rows = append(rows, e.formatResourceRow(pinMark, r))
	}
	return rows
}

func (e *Plugin) formatResourceRow(pinMark string, r sdk.Resource) string {
	full := r.Address + "  " + r.Type
	if r.Tainted {
		full += " " + sdk.StyleUpdate.Render("[tainted]")
	}
	return pinMark + full
}

func (e *Plugin) detailActions() []ui.ActionChip {
	return []ui.ActionChip{
		{Key: "d", Label: "delete"},
		{Key: "e", Label: "edit"},
		{Key: "t", Label: "taint"},
		{Key: "T", Label: "untaint"},
	}
}

func (e *Plugin) renderDetail(width, height int) string {
	e.viewWidth = width
	address := sdk.StyleKey.Render(e.detailAddr)

	actions := e.detailActions()

	headerLines := 2
	maxLines := ui.HeightBudget(height, headerLines, ui.ActionsBarHeight)

	lines := strings.Split(e.detail, "\n")

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

	detail := e.detailPanel.Render(ui.RenderParams{
		Rows:         lines[e.detailScroll:endIdx],
		Width:        width,
		Height:       maxLines,
		TotalItems:   len(lines),
		Cursor:       -1,
		ScrollOffset: e.detailScroll,
	})

	scrollInfo := ""
	if maxScroll > 0 {
		scrollInfo = sdk.StyleFaint.Render(fmt.Sprintf(" [%d/%d]", e.detailScroll+1, maxScroll+1))
	}

	pinIndicator := ""
	if e.isPinnedAddress(e.detailAddr) {
		pinIndicator = " " + sdk.StyleSuccess.Render("[pinned]")
	}

	taintIndicator := ""
	if e.isTaintedAddress(e.detailAddr) {
		taintIndicator = " " + sdk.StyleUpdate.Render("[tainted]")
	}

	return address + taintIndicator + pinIndicator + scrollInfo + "\n\n" + detail + ui.RenderActionsBar(actions, width)
}

// --- Context actions ---
//
// Pins live on the immutable Context. Mutations route through deps.Pin /
// deps.ClearPins; the next ContextChangedEvent (OnlyPinsChanged=true)
// brings back the new set.

func (e *Plugin) clearAllPins() tea.Cmd {
	e.tree.SetPinned(nil)
	if e.pinnedOnly {
		e.pinnedOnly = false
		e.SetFilter(e.filter)
	}
	e.log.Debug("state.pin.clear-all.request")
	return e.clearPinsFn()
}

func (e *Plugin) togglePin(address string) tea.Cmd {
	e.log.Debug("state.pin.toggle.request", "address", address)
	return e.pinFn(address)
}

func (e *Plugin) isPinnedAddress(address string) bool {
	for _, a := range e.pinnedAddresses() {
		if a == address {
			return true
		}
	}
	return false
}

func (e *Plugin) isTaintedAddress(address string) bool {
	for _, r := range e.resources {
		if r.Address == address {
			return r.Tainted
		}
	}
	return false
}

// Output produces stdout content for standalone/CI mode.
func (e *Plugin) Output(jsonOutput bool) ([]byte, error) {
	if jsonOutput {
		type resourceJSON struct {
			Address string `json:"address"`
			Type    string `json:"type"`
			Tainted bool   `json:"tainted,omitempty"`
		}
		out := make([]resourceJSON, 0, len(e.resources))
		for _, r := range e.resources {
			out = append(out, resourceJSON{
				Address: r.Address,
				Type:    r.Type,
				Tainted: r.Tainted,
			})
		}
		return sdk.MarshalJSON(out), nil
	}

	var b strings.Builder
	for _, r := range e.resources {
		b.WriteString(r.Address)
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}
