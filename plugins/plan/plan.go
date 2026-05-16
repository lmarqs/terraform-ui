package plan

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
	svc           sdk.Service
	log           *slog.Logger
	options       *sdk.ResolvedOptions
	stack         *sdk.Stack
	pins          *sdk.PinService
	fuzzy         *ui.FuzzyFilter[sdk.PlanChange]
	timer         ui.Timer
	status        sdk.Status
	summary       *sdk.PlanSummary
	filtered      []sdk.PlanChange
	tree          *tree.Tree
	treeMode      bool
	filterScores  map[string]int
	filter        string
	filtering     bool
	errMsg        string
	lockInfo      *sdk.StateLock
	targets       []string
	scopedContext string
	listHScroll   int
	listWrap      bool
	pinnedOnly    bool
	// detail view state
	detail        string
	detailAddr    string
	detailScroll  int
	detailHScroll int
	detailWrap    bool
	viewWidth     int
}

// New creates a new plan plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		svc:  svc,
		log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		tree: tree.New(nil),
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
func (e *Plugin) Targets() []string   { return e.targets }
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
	if e.pins != nil {
		return e.pins.Count()
	}
	return 0
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// SetTargets configures resource targets for the plan.
func (e *Plugin) SetTargets(targets []string) {
	e.targets = targets
}

// Init initializes the plugin with shared context. Does not auto-run plan —
// the user must explicitly activate the plugin to trigger a plan.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	e.log = ctx.Logger
	e.options = ctx.Options
	e.pins = ctx.Pins
	e.reset()
	return nil
}

// HandleLockCleared implements sdk.LockClearedHandler.
func (e *Plugin) HandleLockCleared(_ sdk.LockClearedEvent) tea.Cmd {
	e.lockInfo = nil
	return e.Refresh()
}

// HandleChdirChanged implements sdk.ChdirHandler.
func (e *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	e.svc = e.svc.WithDir(evt.AbsPath)
	e.scopedContext = evt.AbsPath
	e.reset()
	return nil
}

// HandlePlanInvalidated implements sdk.PlanInvalidatedHandler.
func (e *Plugin) HandlePlanInvalidated(_ sdk.PlanInvalidatedEvent) tea.Cmd {
	e.reset()
	return nil
}

// reset clears all plugin state to initial values.
func (e *Plugin) reset() {
	e.status = sdk.StatusIdle
	e.summary = nil
	e.filtered = nil
	e.tree = tree.New(nil)
	e.filter = ""
	e.filtering = false
	e.filterScores = nil
	e.errMsg = ""
	e.lockInfo = nil
	e.listHScroll = 0
	e.detail = ""
	e.detailAddr = ""
	e.detailScroll = 0
	e.detailHScroll = 0
	e.fuzzy.SetItems(nil)
}

// Activate triggers the plan when the user enters the plugin view.
func (e *Plugin) Activate() tea.Cmd {
	if e.status == sdk.StatusIdle || e.status == sdk.StatusError {
		e.status = sdk.StatusLoading
		e.log.Debug("plan.start", "targets", e.targets)
		return tea.Batch(e.runPlan(), e.timer.Start())
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
	e.listHScroll = 0
	e.detail = ""
	e.detailAddr = ""
	e.fuzzy.SetItems(nil)
	if e.stack != nil {
		e.stack.Clear()
	}
	return tea.Batch(e.runPlan(), e.timer.Start())
}

func (e *Plugin) runPlan() tea.Cmd {
	svc := e.svc
	opts := sdk.BuildPlanOptions(e.options, e.targets)
	return func() tea.Msg {
		summary, err := svc.Plan(context.Background(), opts)
		return PlanResultMsg{Summary: summary, Err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
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
			if e.lockInfo != nil {
				return e, func() tea.Msg { return sdk.LockDetectedEvent{Lock: e.lockInfo} }
			}
		} else {
			e.status = sdk.StatusDone
			e.summary = msg.Summary
			if msg.Summary != nil {
				e.filtered = msg.Summary.Changes
				e.fuzzy.SetItems(msg.Summary.Changes)
				e.pruneStale(msg.Summary.Changes)
				e.rebuildTree()
			}
			changes := 0
			if msg.Summary != nil {
				changes = len(msg.Summary.Changes)
			}
			e.log.Debug("plan.complete", "changes", changes)
			if msg.Summary != nil {
				return e, tea.Batch(
					func() tea.Msg {
						return sdk.PlanCompletedEvent{
							Summary:       msg.Summary,
							ResourceCount: changes,
						}
					},
					func() tea.Msg { return sdk.StateRefreshedEvent{} },
				)
			}
			return e, func() tea.Msg { return sdk.StateRefreshedEvent{} }
		}
		return e, nil
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
	if e.pins != nil {
		e.tree.SetPinned(e.pins.All())
	}
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

func (e *Plugin) togglePin(address string) tea.Cmd {
	e.tree.TogglePin()
	if e.pins != nil {
		e.pins.Set(e.tree.PinnedPaths())
		e.log.Debug("plan.pin.toggle", "address", address, "pinned_count", e.pins.Count())
	}
	return nil
}

func (e *Plugin) isPinnedAddress(address string) bool {
	if e.pins != nil {
		return e.pins.IsPinned(address)
	}
	return false
}

func (e *Plugin) pruneStale(changes []sdk.PlanChange) {
	if e.pins == nil || e.pins.Count() == 0 {
		return
	}
	valid := make(map[string]bool, len(changes))
	for _, c := range changes {
		valid[c.Resource.Address] = true
	}
	var retained []string
	for _, addr := range e.pins.All() {
		if valid[addr] {
			retained = append(retained, addr)
		}
	}
	if len(retained) != e.pins.Count() {
		e.pins.Set(retained)
		e.log.Debug("plan.pin.prune", "before", e.pins.Count()+len(retained), "after", len(retained))
	}
}

func (e *Plugin) clearAllPins() {
	if e.pins != nil {
		e.pins.Set(nil)
	}
	e.tree.SetPinned(nil)
	if e.pinnedOnly {
		e.pinnedOnly = false
		e.SetFilter(e.filter)
	}
	e.log.Debug("plan.pin.clear-all")
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
	e.detailHScroll = 0
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

	if top := e.stack.Peek(); top != nil && top.ID() != "list" {
		return top.View(width, height)
	}

	switch e.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Ready to plan.")

	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Running terraform plan... " + e.timer.FormatElapsed())

	case sdk.StatusError:
		if e.lockInfo != nil {
			return sdk.FormatLockInfo(e.lockInfo)
		}
		return sdk.StyleError.Render("Error: " + e.errMsg)

	case sdk.StatusDone:
		return e.renderResults(width, height)

	default:
		return ""
	}
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
			parts = append(parts, sdk.StyleKey.Render("filter: ")+e.filter)
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

	filterHeight := 0
	if e.filtering || e.filter != "" || e.pinnedOnly {
		filterHeight = 2
	}
	// summary + risk take ~3 lines
	summaryHeight := 3
	maxVisible := height - filterHeight - summaryHeight
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

	summary := e.renderSummaryLine()
	riskLine := e.renderOverallRisk()

	content := filterLine + treeContent + "\n\n" + summary
	if riskLine != "" {
		content += "\n" + riskLine
	}
	return content
}

func (e *Plugin) renderFlatList(contentWidth, maxVisible int) string {
	var b strings.Builder
	cursor := e.tree.Cursor()
	startIdx := e.tree.ViewOffset(maxVisible)
	endIdx := startIdx + maxVisible
	if endIdx > len(e.filtered) {
		endIdx = len(e.filtered)
	}

	for i := startIdx; i < endIdx; i++ {
		change := e.filtered[i]
		pinMark := "[ ] "
		if e.isPinnedAddress(change.Resource.Address) {
			pinMark = sdk.StyleSuccess.Render("[*] ")
		}
		row := e.formatChangeRow(pinMark, change, contentWidth)
		if i == cursor {
			if e.listWrap {
				row = sdk.StyleSelected.Width(contentWidth).Render(row)
			} else {
				row = sdk.StyleSelected.MaxWidth(contentWidth).Width(contentWidth).Render(row)
			}
		}
		if i > startIdx {
			b.WriteByte('\n')
		}
		b.WriteString(row)
	}
	return b.String()
}

func (e *Plugin) formatChangeRow(pinMark string, change sdk.PlanChange, contentWidth int) string {
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

	if !e.listWrap {
		if e.listHScroll > 0 {
			if e.listHScroll < len(full) {
				full = full[e.listHScroll:]
			} else {
				full = ""
			}
		}
		availWidth := contentWidth - 4
		if len(full) > availWidth {
			full = full[:availWidth]
		}
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

	return address + pinIndicator + scrollInfo + "\n\n" + detail
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

// ApplyRequestMsg signals the app to start applying the plan.
type ApplyRequestMsg struct{}

// AutoApplyRequestMsg signals the app to apply without confirmation.
type AutoApplyRequestMsg struct{}

func (e *Plugin) requestApply() tea.Cmd {
	return func() tea.Msg { return ApplyRequestMsg{} }
}

func (e *Plugin) requestAutoApply() tea.Cmd {
	return func() tea.Msg { return AutoApplyRequestMsg{} }
}
