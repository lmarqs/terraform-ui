package phantom

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// PhantomChange holds a phantom change with its explanation.
type PhantomChange struct {
	Change      sdk.PlanChange
	Explanation string
	Attributes  []sdk.AttributeDiff
}

// Plugin implements the phantom change detection feature.
type Plugin struct {
	sdk.PluginBase
	status   sdk.Status
	phantoms []PhantomChange
	selected int
	expanded map[int]bool
	total    int
	real     int
	panel    *ui.ContentPanel
	offset   int

	requestPlan func(*sdk.PlanSummary) (sdk.AssessmentID, tea.Cmd)
	pending     sdk.AssessmentID
	timer       ui.Timer
	failure     string
}

// New creates a new phantom change detection plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		PluginBase: sdk.NewPluginBase("phantom", "Phantom Changes", "Detect and explain phantom (no-op) changes"),
		expanded:   make(map[int]bool),
		panel:      ui.NewContentPanel(),
	}
	p.panel.SelectedStyle = func(s string, width int) string { return sdk.StyleSelected.Width(width).Render(s) }
	p.Svc = svc
	return p
}

func (e *Plugin) Ready() bool        { return e.status == sdk.StatusDone || e.status == sdk.StatusError }
func (e *Plugin) Status() sdk.Status { return e.status }
func (e *Plugin) Selected() int      { return e.selected }
func (e *Plugin) PhantomCount() int  { return len(e.phantoms) }
func (e *Plugin) RealCount() int     { return e.real }
func (e *Plugin) TotalCount() int    { return e.total }
func (e *Plugin) CursorPosition() (int, int) {
	if e.status != sdk.StatusDone || len(e.phantoms) == 0 {
		return 0, 0
	}
	return e.selected + 1, len(e.phantoms)
}

// Hints returns context-sensitive key hints for the status bar.
func (e *Plugin) Hints() []sdk.KeyHint {
	if e.status == sdk.StatusLoading {
		return sdk.HintSetBack.Hints()
	}
	if e.status == sdk.StatusError {
		return (sdk.HintSetRetry | sdk.HintSetBack).Hints()
	}
	if e.navigable() {
		return (sdk.HintSetInspect | sdk.HintSetWrap | sdk.HintSetQuit).Hints(sdk.HintSetOpts{WrapMode: e.panel.WrapMode()})
	}
	return (sdk.HintSetQuit).Hints()
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init wires the plugin to its shared dependencies.
func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	e.InitBase(deps)
	e.requestPlan = deps.RequestAssessment
	return nil
}

// Analyze processes a plan summary and identifies phantom changes.
func (e *Plugin) Analyze(summary *sdk.PlanSummary) {
	if summary == nil || len(summary.Changes) == 0 {
		e.status = sdk.StatusDone
		e.phantoms = nil
		e.total = 0
		e.real = 0
		e.resetView()
		return
	}

	e.total = len(summary.Changes)
	e.phantoms = make([]PhantomChange, 0)

	for _, change := range summary.Changes {
		if change.IsPhantom {
			pc := PhantomChange{
				Change:      change,
				Explanation: explainPhantom(change),
				Attributes:  change.AttributeDiffs,
			}
			e.phantoms = append(e.phantoms, pc)
		}
	}

	e.real = e.total - len(e.phantoms)
	e.status = sdk.StatusDone
	e.selected = 0
	e.expanded = make(map[int]bool)
	e.resetView()
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return e, e.timer.Tick()
	case tea.KeyMsg:
		switch {
		case msg.String() == "esc" && (e.status == sdk.StatusLoading || e.status == sdk.StatusError):
			return e, func() tea.Msg { return sdk.DeactivateMsg{} }
		case msg.String() == "ctrl+r" && e.status == sdk.StatusError:
			return e, e.Activate()
		}
		e.handleKey(msg)
		return e, nil
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "j", "down":
		e.MoveDown()
	case "k", "up":
		e.MoveUp()
	case "enter", "i":
		e.ToggleExpand()
	case "left", "right", "ctrl+w":
		if e.navigable() {
			e.panel.HandleKey(msg)
		}
	}
}

// navigable reports whether the list shows phantoms to move through and pan.
func (e *Plugin) navigable() bool { return e.status == sdk.StatusDone && len(e.phantoms) > 0 }

// resetView returns the window and the pan to the top left of the list.
func (e *Plugin) resetView() {
	e.offset = 0
	e.panel.ResetScroll()
}

// MoveUp moves selection up.
func (e *Plugin) MoveUp() {
	if e.selected > 0 {
		e.selected--
	}
}

// MoveDown moves selection down.
func (e *Plugin) MoveDown() {
	if e.selected < len(e.phantoms)-1 {
		e.selected++
	}
}

// ToggleExpand toggles the detail view for the selected phantom change.
func (e *Plugin) ToggleExpand() {
	e.expanded[e.selected] = !e.expanded[e.selected]
}

// View renders the phantom change detection plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Run a plan first to detect phantom changes...")

	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Loading plan... " + e.timer.FormatElapsed())

	case sdk.StatusError:
		return sdk.StyleError.Render(e.failure)

	case sdk.StatusDone:
		return e.renderPhantoms(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderPhantoms(width, height int) string {
	if len(e.phantoms) == 0 {
		noPhantoms := sdk.StyleSuccess.Render("No phantom changes detected.")
		summary := sdk.StyleFaint.Render(fmt.Sprintf("All %d changes are real modifications.", e.total))
		return noPhantoms + "\n" + summary
	}

	rows := []string{}
	cursor, cursorEnd := 0, 0
	for i, pc := range e.phantoms {
		if i == e.selected {
			cursor = len(rows)
		}
		rows = append(rows, e.renderPhantomRow(pc, i))
		if e.expanded[i] {
			rows = append(rows, e.renderPhantomDetails(pc, width)...)
		}
		if i == e.selected {
			cursorEnd = len(rows)
		}
	}

	// The banner, its note and the blank line under them take three rows, unless the
	// cursor's phantom is taller than what is left: it gets them, so its last line shows.
	listHeight, header := height-3, e.renderBanner(width)
	if cursorEnd-cursor > listHeight {
		listHeight, header = height, ""
	}
	content, offset := e.panel.RenderWindow(ui.RenderParams{Rows: rows, Width: width, Height: max(1, listHeight), Cursor: cursor, CursorEnd: cursorEnd, ScrollOffset: e.offset})
	e.offset = offset

	lines := strings.Split(header+content, "\n")
	return strings.Join(lines[:min(max(0, height), len(lines))], "\n")
}

// renderBanner is the summary and its note, each cut to width, and the blank line under them.
func (e *Plugin) renderBanner(width int) string {
	banner := sdk.StylePhantom.Render(fmt.Sprintf(
		"Detected %d phantom change(s) out of %d total",
		len(e.phantoms), e.total,
	))
	note := sdk.StyleFaint.Render(
		"These changes appear in the plan but result in no actual infrastructure modification.",
	)
	return ansi.Truncate(banner, max(0, width), "") + "\n" + ansi.Truncate(note, max(0, width), "") + "\n\n"
}

func (e *Plugin) renderPhantomRow(pc PhantomChange, idx int) string {
	indicator := ">"
	if e.expanded[idx] {
		indicator = "v"
	}

	address := sdk.StylePhantom.Render(pc.Change.Resource.Address)
	attrCount := sdk.StyleFaint.Render(fmt.Sprintf("(%d attrs)", len(pc.Attributes)))

	return fmt.Sprintf(" %s %s %s  %s", indicator, sdk.StylePhantom.Render("~"), address, attrCount)
}

// renderPhantomDetails returns the rows of an expanded phantom's reason and attribute diffs,
// ending in a blank row.
func (e *Plugin) renderPhantomDetails(pc PhantomChange, width int) []string {
	rows := []string{"   " + sdk.StyleFaintItalic.Render("Reason: "+pc.Explanation)}

	// Show attribute diffs that are cosmetic
	for _, diff := range pc.Attributes {
		key := sdk.StyleKey.Render("     " + diff.Key)
		if diff.Sensitive {
			rows = append(rows, key+": "+sdk.StyleFaintItalic.Render("(sensitive)"))
			continue
		}
		old := sdk.Truncate(diff.OldValue, width/4)
		new := sdk.Truncate(diff.NewValue, width/4)
		rows = append(rows, key+": "+sdk.StyleFaint.Render(old+" = "+new))
	}
	return append(rows, "")
}

func explainPhantom(change sdk.PlanChange) string {
	if len(change.AttributeDiffs) == 0 {
		return "empty diff detected"
	}

	// Check common patterns
	hasJSON := false
	hasOrdering := false
	for _, diff := range change.AttributeDiffs {
		if strings.Contains(diff.Key, "json") || strings.Contains(diff.Key, "policy") {
			hasJSON = true
		}
		if strings.Contains(diff.Key, "tags") || strings.Contains(diff.Key, "labels") {
			hasOrdering = true
		}
	}

	switch {
	case hasJSON:
		return "JSON/policy field reordering or whitespace difference"
	case hasOrdering:
		return "tag/label ordering difference (cosmetic)"
	default:
		return "semantically equivalent values with different serialization"
	}
}
