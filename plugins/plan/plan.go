package plan

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui/tree"
)

// PlanResultMsg is sent when the plan operation completes.
type PlanResultMsg struct {
	Summary *sdk.PlanSummary
	Err     error
}

// changeItem wraps sdk.PlanChange to implement tree.Item.
type changeItem struct {
	change sdk.PlanChange
}

func (c changeItem) Address() string { return c.change.Resource.Address }

// Plugin implements the plan review feature.
type Plugin struct {
	svc          sdk.Service
	log          *slog.Logger
	getCtx       func() *sdk.Context
	pinFn        func(string) tea.Cmd
	clearPinsFn  func() tea.Cmd
	stack        *sdk.Stack
	fuzzy        *ui.FuzzyFilter[sdk.PlanChange]
	timer        ui.Timer
	status       sdk.Status
	summary      *sdk.PlanSummary
	filtered     []sdk.PlanChange
	tree         *tree.Tree
	treeMode     bool
	filterScores map[string]int
	filter       string
	filtering    bool
	errMsg       string
	lockInfo     *sdk.StateLock
	stale        bool
	listPanel    *ui.ContentPanel
	pinnedOnly   bool
	cancelFn     context.CancelFunc
	lastStream   *frames.StreamFrame // retained for L key re-display after success
	streamCh     <-chan string       // stored so callers can batch WaitForLine separately
	planFile     string              // path to the plan artifact produced by the most recent run
	// detail view state
	detail       string
	detailAddr   string
	detailScroll int
	detailPanel  *ui.ContentPanel
	viewWidth    int
}

// New creates a new plan plugin.
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
		fuzzy: ui.NewFuzzyFilter(func(c sdk.PlanChange) string {
			return c.Resource.Address
		}),
	}
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	return p
}

func (e *Plugin) ID() string          { return "plan" }
func (e *Plugin) Name() string        { return "Plan" }
func (e *Plugin) Description() string { return "Review terraform plan changes" }
func (e *Plugin) Ready() bool         { return e.status == sdk.StatusDone }
func (e *Plugin) Status() sdk.Status  { return e.status }
func (e *Plugin) Busy() bool          { return e.status == sdk.StatusLoading }
func (e *Plugin) Stack() *sdk.Stack   { return e.stack }
func (e *Plugin) Summary() *sdk.PlanSummary {
	return e.summary
}

func (e *Plugin) Count() (int, int) {
	if e.summary == nil {
		return 0, 0
	}
	return len(e.filtered), len(e.summary.Changes)
}

func (e *Plugin) PinnedCount() int {
	return len(e.pinnedAddresses())
}

func (e *Plugin) pinnedAddresses() []string {
	return sdk.PinnedAddresses(e.getCtx)
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init wires the plugin to its shared dependencies. Does not auto-run plan —
// the user must explicitly activate the plugin to trigger a plan.
func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	e.svc = deps.Service
	e.log = deps.Logger
	e.getCtx = deps.Context
	e.pinFn = deps.Pin
	e.clearPinsFn = deps.ClearPins
	e.reset()
	return nil
}

// HandleLockCleared implements sdk.LockClearedHandler.
func (e *Plugin) HandleLockCleared(_ sdk.LockClearedEvent) tea.Cmd {
	e.lockInfo = nil
	e.reset()
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler. When only the
// pinned targets changed (same chdir + workspace), the cached plan summary
// becomes stale but other UI state is preserved; on chdir or workspace
// changes the plugin fully resets.
func (e *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if ev.Next == nil {
		return nil
	}
	if ev.OnlyPinsChanged() {
		e.stale = true
		return nil
	}
	if ev.Next.Service != nil {
		e.svc = ev.Next.Service
	}
	e.reset()
	return nil
}

// HandlePlanInvalidated implements sdk.PlanInvalidatedHandler.
func (e *Plugin) HandlePlanInvalidated(_ sdk.PlanInvalidatedEvent) tea.Cmd {
	if e.status == sdk.StatusDone {
		// Keep results visible; Activate() will re-plan on next entry
		e.stale = true
		return nil
	}
	e.reset()
	return nil
}

// reset clears all plugin state to initial values. Removes any stale plan
// artifact from disk so subsequent runs start clean.
func (e *Plugin) reset() {
	if e.planFile != "" {
		_ = os.Remove(e.planFile)
		e.planFile = ""
	}
	e.status = sdk.StatusIdle
	e.stale = false
	e.summary = nil
	e.filtered = nil
	e.tree = tree.New(nil)
	e.filter = ""
	e.filtering = false
	e.filterScores = nil
	e.errMsg = ""
	e.lockInfo = nil
	e.lastStream = nil
	e.streamCh = nil
	e.listPanel.ResetScroll()
	e.detail = ""
	e.detailAddr = ""
	e.detailScroll = 0
	e.detailPanel.ResetScroll()
	e.fuzzy.SetItems(nil)
	e.stack.Clear()
}

// Activate triggers the plan when the user enters the plugin view.
func (e *Plugin) Activate() tea.Cmd {
	if e.status == sdk.StatusIdle || e.status == sdk.StatusError || e.stale {
		e.stale = false
		e.status = sdk.StatusLoading
		e.log.Debug("plan.start")
		planCmd := e.runPlan()
		return tea.Batch(planCmd, frames.WaitForLine(e.streamCh), e.timer.Start())
	}
	if e.status == sdk.StatusLoading && e.timer.Running() {
		return e.timer.Tick()
	}
	return nil
}

// Refresh re-runs the plan.
func (e *Plugin) Refresh() tea.Cmd {
	e.status = sdk.StatusLoading
	e.summary = nil
	e.filtered = nil
	e.errMsg = ""
	e.lockInfo = nil
	e.filter = ""
	e.filtering = false
	e.filterScores = nil
	e.tree = tree.New(nil)
	e.listPanel.ResetScroll()
	e.detail = ""
	e.detailAddr = ""
	e.fuzzy.SetItems(nil)
	planCmd := e.runPlan()
	return tea.Batch(planCmd, frames.WaitForLine(e.streamCh), e.timer.Start())
}

// Cancel aborts any in-flight terraform operation.
func (e *Plugin) Cancel() {
	if e.cancelFn != nil {
		e.cancelFn()
		e.cancelFn = nil
	}
}

func (e *Plugin) runPlan() tea.Cmd {
	e.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFn = cancel

	lw, ch := frames.NewLineWriter()
	sf := frames.NewStreamFrame("terraform plan", ch, cancel)
	e.lastStream = sf
	e.streamCh = ch
	e.stack.Clear()
	e.stack.Push(sf)

	svc := e.svc
	var opts sdk.PlanOptions
	if e.getCtx != nil {
		opts = e.getCtx().PlanOptions()
	}
	opts.PlanFile = e.allocPlanFile()
	opts.Writer = lw
	return func() tea.Msg {
		summary, err := svc.Plan(ctx, opts)
		lw.Close()
		return PlanResultMsg{Summary: summary, Err: err}
	}
}

// allocPlanFile reserves a unique path for the plan artifact and stores it on
// the plugin so PlanCompletedEvent can hand it off to apply.
func (e *Plugin) allocPlanFile() string {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("tfui-%d-%d.tfplan", os.Getpid(), time.Now().UnixNano()))
	e.planFile = path
	return path
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg.(type) {
	case frames.StreamLineMsg, frames.StreamDoneMsg:
		cmd := e.stack.Update(msg)
		return e, cmd
	}

	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return e, e.timer.Tick()

	case PlanResultMsg:
		e.timer.Stop()
		if msg.Err != nil {
			e.status = sdk.StatusError
			e.errMsg = msg.Err.Error()
			e.lockInfo = sdk.ParseLockError(e.errMsg)
			e.log.Debug("plan.error", "error", msg.Err.Error())
			e.stack.Clear()
			e.stack.Push(&listFrame{plugin: e})
			if e.lockInfo != nil {
				return e, func() tea.Msg { return sdk.LockDetectedEvent{Lock: e.lockInfo} }
			}
			return e, nil
		} else {
			e.status = sdk.StatusDone
			e.summary = msg.Summary
			var pruneCmd tea.Cmd
			if msg.Summary != nil {
				e.filtered = msg.Summary.Changes
				e.fuzzy.SetItems(msg.Summary.Changes)
				pruneCmd = e.pruneStalePins(msg.Summary.Changes)
				e.rebuildTree()
			}
			// Pop the StreamFrame and restore the list frame so the tree is shown.
			e.stack.Clear()
			e.stack.Push(&listFrame{plugin: e})

			changes := 0
			if msg.Summary != nil {
				changes = len(msg.Summary.Changes)
			}
			e.log.Debug("plan.complete", "changes", changes)
			cmds := []tea.Cmd{}
			if pruneCmd != nil {
				cmds = append(cmds, pruneCmd)
			}
			if msg.Summary != nil {
				planFile := e.planFile
				cmds = append(cmds,
					func() tea.Msg {
						return sdk.PlanCompletedEvent{
							Summary:       msg.Summary,
							ResourceCount: changes,
							PlanFile:      planFile,
						}
					},
				)
			}
			cmds = append(cmds, func() tea.Msg { return sdk.StateRefreshedEvent{} })
			return e, tea.Batch(cmds...)
		}
	}
	return e, nil
}

// --- Navigation ---

func (e *Plugin) MoveUp() {
	e.tree.MoveUp()
}

func (e *Plugin) MoveDown() {
	e.tree.MoveDown()
}

func (e *Plugin) MoveToStart() {
	e.tree.MoveToStart()
}

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
	e.listPanel.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
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

// --- Filter ---

func (e *Plugin) sourceChanges() []sdk.PlanChange {
	if e.summary == nil {
		return nil
	}
	if !e.pinnedOnly {
		return e.summary.Changes
	}
	var result []sdk.PlanChange
	for _, c := range e.summary.Changes {
		if e.isPinnedAddress(c.Resource.Address) {
			result = append(result, c)
		}
	}
	return result
}

func (e *Plugin) SetFilter(filter string) {
	e.filter = filter
	source := e.sourceChanges()
	if filter == "" {
		e.filtered = source
		e.filterScores = nil
		e.rebuildTree()
		e.log.Debug("plan.filter", "filter", "", "results", len(source))
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
	for i, c := range ordered {
		e.filterScores[c.Resource.Address] = e.fuzzy.ScoreAt(i)
	}

	e.rebuildTree()
	e.log.Debug("plan.filter", "filter", filter, "results", len(e.filtered))
}

// --- Tree ---

func (e *Plugin) rebuildTree() {
	items := make([]tree.Item, len(e.filtered))
	for i, c := range e.filtered {
		items[i] = changeItem{change: c}
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

func (e *Plugin) syncPinnedToTree() {
	e.tree.SetPinned(e.pinnedAddresses())
}

// --- Selection ---

func (e *Plugin) SelectedChange() *sdk.PlanChange {
	item := e.tree.CursorItem()
	if item != nil {
		c := item.(changeItem).change
		return &c
	}
	return nil
}

func (e *Plugin) CursorNode() *tree.Node {
	return e.tree.CursorNode()
}

// --- Pins ---
//
// Pins are owned by the immutable Context. The plugin emits a request to the
// App and the next ContextChangedEvent (with OnlyPinsChanged=true) brings
// back the new pinned set. We do NOT mutate any local pin state in place —
// that's exactly the bug class ADR-0018 closes.

func (e *Plugin) togglePin(address string) tea.Cmd {
	e.log.Debug("plan.pin.toggle.request", "address", address)
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

// pruneStalePins drops pinned addresses no longer present in the latest plan
// summary. Returns a Cmd that asks the App to clear-then-restore the
// surviving subset (a single ContextChangedEvent), or nil if nothing to do.
func (e *Plugin) pruneStalePins(changes []sdk.PlanChange) tea.Cmd {
	pinned := e.pinnedAddresses()
	if len(pinned) == 0 {
		return nil
	}
	valid := make(map[string]bool, len(changes))
	for _, c := range changes {
		valid[c.Resource.Address] = true
	}
	var stale []string
	for _, addr := range pinned {
		if !valid[addr] {
			stale = append(stale, addr)
		}
	}
	if len(stale) == 0 {
		return nil
	}
	e.log.Debug("plan.pin.prune", "stale", len(stale), "remaining", len(pinned)-len(stale))
	cmds := make([]tea.Cmd, 0, len(stale))
	for _, addr := range stale {
		if e.pinFn != nil {
			cmds = append(cmds, e.pinFn(addr))
		}
	}
	return tea.Batch(cmds...)
}

func (e *Plugin) clearAllPins() tea.Cmd {
	e.tree.SetPinned(nil)
	if e.pinnedOnly {
		e.pinnedOnly = false
		e.SetFilter(e.filter)
	}
	e.log.Debug("plan.pin.clear-all.request")
	return e.clearPinsFn()
}

// --- Detail ---

func (e *Plugin) inspectSelected() tea.Cmd {
	change := e.SelectedChange()
	if change == nil {
		return nil
	}
	e.detail = e.buildInspectContent(change)
	e.detailAddr = change.Resource.Address
	e.detailScroll = 0
	e.detailPanel.ResetScroll()
	e.filtering = false
	if e.stack.Peek() != nil && e.stack.Peek().ID() == "filter" {
		e.stack.Pop()
	}
	e.stack.Push(&detailFrame{plugin: e})
	return nil
}

func (e *Plugin) buildInspectContent(change *sdk.PlanChange) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Action:   %s %s\n", sdk.ActionSymbol(change.Action), string(change.Action))
	fmt.Fprintf(&b, "Address:  %s\n", change.Resource.Address)
	if change.Resource.Module != "" {
		fmt.Fprintf(&b, "Module:   %s\n", change.Resource.Module)
	}
	fmt.Fprintf(&b, "Type:     %s\n", change.Resource.Type)
	fmt.Fprintf(&b, "Provider: %s\n", change.Resource.ProviderName)

	if change.Risk != sdk.RiskNone {
		fmt.Fprintf(&b, "Risk:     %s\n", sdk.RiskBadge(change.Risk))
	}
	if change.IsPhantom {
		b.WriteString("Phantom:  yes (no real change detected)\n")
	}

	if len(change.AttributeDiffs) > 0 {
		b.WriteString("\nAttributes:\n")
		for _, diff := range change.AttributeDiffs {
			if diff.Sensitive {
				fmt.Fprintf(&b, "  %s: (sensitive)\n", diff.Key)
				continue
			}
			if diff.ForcesNew {
				fmt.Fprintf(&b, "  %s (forces new):\n", diff.Key)
			} else {
				fmt.Fprintf(&b, "  %s:\n", diff.Key)
			}
			fmt.Fprintf(&b, "    - %s\n", diff.OldValue)
			fmt.Fprintf(&b, "    + %s\n", diff.NewValue)
		}
	}

	return b.String()
}

// --- View ---

func (e *Plugin) View(width, height int) string {
	e.viewWidth = width

	if top := e.stack.Peek(); top != nil && top.ID() != "list" && top.ID() != "filter" {
		return top.View(width, height)
	}

	switch e.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Ready to plan.")

	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Running terraform plan... " + e.timer.FormatElapsed())

	case sdk.StatusError:
		if e.lockInfo != nil {
			actions := []ui.ActionChip{{Key: "u", Label: "force-unlock"}}
			return sdk.FormatLockInfo(e.lockInfo) + ui.RenderActionsBar(actions, width)
		}
		return sdk.StyleError.Render("Error: " + e.errMsg)

	case sdk.StatusDone:
		return e.renderResults(width, height)

	default:
		return ""
	}
}

func (e *Plugin) CursorPosition() (int, int) {
	if e.status != sdk.StatusDone || e.summary == nil || len(e.summary.Changes) == 0 {
		return 0, 0
	}
	return e.tree.Cursor() + 1, e.tree.VisibleCount()
}

func (e *Plugin) listActions() []ui.ActionChip {
	chips := []ui.ActionChip{
		{Key: "e", Label: "edit"},
		{Key: "t", Label: "taint"},
		{Key: "T", Label: "untaint"},
		{Key: "a", Label: "apply"},
		{Key: "A", Label: "auto-apply"},
	}
	if e.PinnedCount() > 0 {
		chips = append(chips, ui.ActionChip{Key: "!", Label: "batch"})
	}
	return chips
}

func (e *Plugin) renderResults(width, height int) string {
	if e.summary == nil || len(e.summary.Changes) == 0 {
		return sdk.StyleSuccess.Render("No changes. Infrastructure is up-to-date.")
	}

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
		noChanges := sdk.StyleFaintItalic.Render("No matching changes.")
		return filterLine + noChanges
	}

	actions := e.listActions()

	filterHeight := 0
	if e.filtering || e.filter != "" || e.pinnedOnly {
		filterHeight = 2
	}
	summaryHeight := 3
	maxVisible := ui.HeightBudget(height, filterHeight, summaryHeight, ui.ActionsBarHeight)

	hasGutter := ui.NeedsGutter(e.tree.VisibleCount(), maxVisible)
	contentWidth := ui.ContentWidth(width, hasGutter)

	var rows []string
	if e.treeMode {
		rows = e.tree.RenderRows(tree.RenderOpts{
			Width:  contentWidth,
			Height: maxVisible,
			RenderLeaf: func(node *tree.Node, pinned bool) string {
				c := node.Item.(changeItem).change
				symbol := sdk.ActionSymbol(c.Action)
				risk := sdk.RiskBadge(c.Risk)
				full := symbol + " " + node.Label
				if risk != "" {
					full += " " + risk
				}
				if c.IsPhantom {
					full += " " + sdk.StylePhantom.Render("(phantom)")
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
		rows = e.buildFlatRows(contentWidth, maxVisible)
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

	summary := e.renderSummaryLine()
	riskLine := e.renderOverallRisk()

	content := filterLine + treeContent + "\n\n" + summary
	if riskLine != "" {
		content += "\n" + riskLine
	}
	content += ui.RenderActionsBar(actions, width)
	return content
}

func (e *Plugin) buildFlatRows(contentWidth, maxVisible int) []string {
	startIdx := e.tree.ViewOffset(maxVisible)
	endIdx := startIdx + maxVisible
	if endIdx > len(e.filtered) {
		endIdx = len(e.filtered)
	}

	rows := make([]string, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		change := e.filtered[i]
		pinMark := "[ ] "
		if e.isPinnedAddress(change.Resource.Address) {
			pinMark = sdk.StyleSuccess.Render("[*] ")
		}
		rows = append(rows, e.formatChangeRow(pinMark, change))
	}
	return rows
}

func (e *Plugin) formatChangeRow(pinMark string, change sdk.PlanChange) string {
	symbol := sdk.ActionSymbol(change.Action)
	address := change.Resource.Address
	risk := sdk.RiskBadge(change.Risk)

	full := symbol + " " + address
	if risk != "" {
		full += " " + risk
	}
	if change.IsPhantom {
		full += " " + sdk.StylePhantom.Render("(phantom)")
	}

	return pinMark + full
}

func (e *Plugin) detailActions() []ui.ActionChip {
	return []ui.ActionChip{
		{Key: "e", Label: "edit"},
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

	return address + pinIndicator + scrollInfo + "\n\n" + detail + ui.RenderActionsBar(actions, width)
}

func (e *Plugin) renderSummaryLine() string {
	s := e.summary
	parts := []string{}
	if s.ToCreate > 0 {
		parts = append(parts, sdk.StyleCreate.Render(fmt.Sprintf("%d to add", s.ToCreate)))
	}
	if s.ToUpdate > 0 {
		parts = append(parts, sdk.StyleUpdate.Render(fmt.Sprintf("%d to change", s.ToUpdate)))
	}
	if s.ToDelete > 0 {
		parts = append(parts, sdk.StyleDelete.Render(fmt.Sprintf("%d to destroy", s.ToDelete)))
	}
	if s.ToReplace > 0 {
		parts = append(parts, sdk.StyleReplace.Render(fmt.Sprintf("%d to replace", s.ToReplace)))
	}

	if len(parts) == 0 {
		return sdk.StyleFaint.Render("Plan: no changes")
	}
	return "Plan: " + strings.Join(parts, ", ")
}

func (e *Plugin) renderOverallRisk() string {
	if e.summary == nil || len(e.summary.Changes) == 0 {
		return ""
	}
	overall := sdk.OverallRisk(e.summary.Changes)
	switch overall {
	case sdk.RiskCritical:
		return sdk.StyleRiskCritical.Render("Overall risk: CRITICAL")
	case sdk.RiskHigh:
		return sdk.StyleRiskHigh.Render("Overall risk: HIGH")
	case sdk.RiskMedium:
		return sdk.StyleRiskMedium.Render("Overall risk: medium")
	case sdk.RiskLow:
		return sdk.StyleRiskLow.Render("Overall risk: low")
	default:
		return ""
	}
}

// PlanEditMsg signals the app to open $EDITOR at the resource's source file.
type PlanEditMsg struct {
	Address string
}

// ApplyRequestMsg signals the app to apply the plan artifact produced by the
// most recent plan run. PlanFile is the path to the saved plan; AutoApprove
// short-circuits the confirmation prompt when true.
type ApplyRequestMsg struct {
	PlanFile    string
	AutoApprove bool
}

// Output produces stdout content for standalone/CI mode.
func (e *Plugin) Output(jsonOutput bool) ([]byte, error) {
	if e.summary == nil {
		return nil, nil
	}

	if jsonOutput {
		type changeJSON struct {
			Address string `json:"address"`
			Action  string `json:"action"`
			Risk    string `json:"risk"`
			Phantom bool   `json:"phantom,omitempty"`
		}
		out := struct {
			Changes []changeJSON `json:"changes"`
			Summary struct {
				Add     int `json:"add"`
				Change  int `json:"change"`
				Destroy int `json:"destroy"`
			} `json:"summary"`
			Risk string `json:"risk"`
		}{
			Changes: make([]changeJSON, 0, len(e.summary.Changes)),
		}
		for _, c := range e.summary.Changes {
			out.Changes = append(out.Changes, changeJSON{
				Address: c.Resource.Address,
				Action:  string(c.Action),
				Risk:    c.Risk.String(),
				Phantom: c.IsPhantom,
			})
		}
		out.Summary.Add = e.summary.ToCreate
		out.Summary.Change = e.summary.ToUpdate + e.summary.ToReplace
		out.Summary.Destroy = e.summary.ToDelete
		out.Risk = sdk.OverallRisk(e.summary.Changes).String()
		return sdk.MarshalJSON(out), nil
	}

	var b strings.Builder
	for _, change := range e.summary.Changes {
		sym := plainActionSymbol(change.Action)
		fmt.Fprintf(&b, "%s %s\n", sym, change.Resource.Address)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "Plan: %d to add, %d to change, %d to destroy.\n",
		e.summary.ToCreate, e.summary.ToUpdate+e.summary.ToReplace, e.summary.ToDelete)
	if risk := sdk.OverallRisk(e.summary.Changes); risk > sdk.RiskNone {
		fmt.Fprintf(&b, "Risk: %s\n", risk)
	}
	return []byte(b.String()), nil
}

// ExitCode returns 2 when the plan has changes, 0 when clean.
func (e *Plugin) ExitCode() int {
	if e.summary != nil && len(e.summary.Changes) > 0 {
		return 2
	}
	return 0
}

func plainActionSymbol(action sdk.Action) string {
	switch action {
	case sdk.ActionCreate:
		return "+"
	case sdk.ActionUpdate:
		return "~"
	case sdk.ActionDelete:
		return "-"
	case sdk.ActionDeleteThenCreate, sdk.ActionCreateThenDelete:
		return "-/+"
	case sdk.ActionRead:
		return "<="
	default:
		return " "
	}
}

func (e *Plugin) requestApply() tea.Cmd {
	planFile := e.planFile
	return func() tea.Msg { return ApplyRequestMsg{PlanFile: planFile} }
}

func (e *Plugin) requestAutoApply() tea.Cmd {
	planFile := e.planFile
	return func() tea.Msg { return ApplyRequestMsg{PlanFile: planFile, AutoApprove: true} }
}
