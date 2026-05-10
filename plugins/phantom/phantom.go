package phantom

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Status represents the current state of the phantom plugin.
type Status int

const (
	StatusIdle Status = iota
	StatusReady
)

// PhantomChange holds a phantom change with its explanation.
type PhantomChange struct {
	Change      sdk.PlanChange
	Explanation string
	Attributes  []sdk.AttributeDiff
}

// Plugin implements the phantom change detection feature.
type Plugin struct {
	svc      sdk.Service
	status   Status
	phantoms []PhantomChange
	selected int
	expanded map[int]bool
	total    int
	real     int
}

// New creates a new phantom change detection plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc:      svc,
		expanded: make(map[int]bool),
	}
}

func (e *Plugin) ID() string          { return "phantom" }
func (e *Plugin) Name() string        { return "Phantom Changes" }
func (e *Plugin) Description() string { return "Detect and explain phantom (no-op) changes" }
func (e *Plugin) Ready() bool         { return e.status == StatusReady }
func (e *Plugin) Status() Status      { return e.status }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) PhantomCount() int   { return len(e.phantoms) }
func (e *Plugin) RealCount() int      { return e.real }
func (e *Plugin) TotalCount() int     { return e.total }

// Hints returns context-sensitive key hints for the status bar.
func (e *Plugin) Hints() []sdk.KeyHint {
	if e.status == StatusReady && len(e.phantoms) > 0 {
		return (sdk.HintSetNavigate | sdk.HintSetInspect | sdk.HintSetBack).Hints()
	}
	return (sdk.HintSetBack).Hints()
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	return nil
}

// Analyze processes a plan summary and identifies phantom changes.
func (e *Plugin) Analyze(summary *sdk.PlanSummary) {
	if summary == nil || len(summary.Changes) == 0 {
		e.status = StatusReady
		e.phantoms = nil
		e.total = 0
		e.real = 0
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
	e.status = StatusReady
	e.selected = 0
	e.expanded = make(map[int]bool)
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
	}
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
	case StatusIdle:
		return sdk.StyleFaintItalic.Render("Run a plan first to detect phantom changes...")

	case StatusReady:
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

	var b strings.Builder

	// Summary banner
	banner := sdk.StylePhantom.Render(fmt.Sprintf(
		"Detected %d phantom change(s) out of %d total",
		len(e.phantoms), e.total,
	))
	b.WriteString(banner)
	b.WriteString("\n")
	b.WriteString(sdk.StyleFaint.Render(
		"These changes appear in the plan but result in no actual infrastructure modification.",
	))
	b.WriteString("\n\n")

	// Calculate visible area
	maxVisible := height - 10
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if e.selected >= maxVisible {
		startIdx = e.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(e.phantoms) {
		endIdx = len(e.phantoms)
	}

	for i := startIdx; i < endIdx; i++ {
		pc := e.phantoms[i]
		row := e.renderPhantomRow(pc, i)
		if i == e.selected {
			row = sdk.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')

		// Show expanded details
		if e.expanded[i] {
			b.WriteString(e.renderPhantomDetails(pc, width))
		}
	}

	return b.String()
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

func (e *Plugin) renderPhantomDetails(pc PhantomChange, width int) string {
	var b strings.Builder

	// Explanation
	b.WriteString("   ")
	b.WriteString(sdk.StyleFaintItalic.Render("Reason: " + pc.Explanation))
	b.WriteByte('\n')

	// Show attribute diffs that are cosmetic
	for _, diff := range pc.Attributes {
		key := sdk.StyleKey.Render("     " + diff.Key)
		if diff.Sensitive {
			b.WriteString(key + ": " + sdk.StyleFaintItalic.Render("(sensitive)") + "\n")
			continue
		}
		old := sdk.Truncate(diff.OldValue, width/4)
		new := sdk.Truncate(diff.NewValue, width/4)
		b.WriteString(key + ": " + sdk.StyleFaint.Render(old+" = "+new) + "\n")
	}
	b.WriteByte('\n')
	return b.String()
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
