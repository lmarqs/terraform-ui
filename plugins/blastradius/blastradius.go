package blastradius

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// ImpactScore represents the calculated impact of a module group.
type ImpactScore int

const (
	ImpactMinimal  ImpactScore = iota // fewer than 3 changes, none above low risk
	ImpactModerate                    // 3+ changes or a medium-risk change
	ImpactHigh                        // a high-risk change, or 3+ changes with a destructive one
	ImpactCritical                    // a critical-risk change
)

func (s ImpactScore) String() string {
	switch s {
	case ImpactMinimal:
		return "minimal"
	case ImpactModerate:
		return "moderate"
	case ImpactHigh:
		return "high"
	case ImpactCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ModuleImpact holds a module group with its calculated impact score.
type ModuleImpact struct {
	Group       sdk.ModuleGroup
	Score       ImpactScore
	Unavailable bool
	NotAssessed bool
}

// Plugin implements the blast radius visualization feature.
type Plugin struct {
	sdk.PluginBase
	status      sdk.Status
	modules     []ModuleImpact
	selected    int
	expanded    map[int]bool
	total       int
	summary     *sdk.PlanSummary
	assessment  *sdk.PolicyAssessment
	rows        sdk.AssessmentRows
	requestRisk func(*sdk.PlanSummary) (sdk.AssessmentID, tea.Cmd)
	pending     sdk.AssessmentID
	timer       ui.Timer
	panel       *ui.ContentPanel
	offset      int
}

// New creates a new blast radius plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		PluginBase: sdk.NewPluginBase("blastradius", "Blast Radius", "Visualize module-grouped changes with impact scores"),
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
func (e *Plugin) ModuleCount() int   { return len(e.modules) }
func (e *Plugin) TotalChanges() int  { return e.total }
func (e *Plugin) Count() (int, int)  { return len(e.modules), e.total }
func (e *Plugin) CursorPosition() (int, int) {
	if e.status != sdk.StatusDone || len(e.modules) == 0 {
		return 0, 0
	}
	return e.selected + 1, len(e.modules)
}

// Hints returns context-sensitive key hints for the status bar.
func (e *Plugin) Hints() []sdk.KeyHint {
	if e.status == sdk.StatusLoading {
		return sdk.HintSetBack.Hints()
	}
	var set sdk.HintSet
	if e.navigable() {
		set |= sdk.HintSetInspect | sdk.HintSetWrap
	}
	opts := sdk.HintSetOpts{WrapMode: e.panel.WrapMode()}
	if e.failed() {
		return (set | sdk.HintSetRetry | sdk.HintSetBack).Hints(opts)
	}
	return (set | sdk.HintSetQuit).Hints(opts)
}

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init wires the plugin to its shared dependencies.
func (e *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	e.InitBase(deps)
	e.requestRisk = deps.RequestAssessment
	return nil
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return e, e.timer.Tick()
	case tea.KeyMsg:
		switch {
		case msg.String() == "esc":
			return e, func() tea.Msg { return sdk.DeactivateMsg{} }
		case msg.String() == "ctrl+r" && e.failed():
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

// navigable reports whether the list shows modules to move through and pan.
func (e *Plugin) navigable() bool { return e.status == sdk.StatusDone && len(e.modules) > 0 }

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
	if e.selected < len(e.modules)-1 {
		e.selected++
	}
}

// ToggleExpand toggles the expanded view for the selected module.
func (e *Plugin) ToggleExpand() {
	e.expanded[e.selected] = !e.expanded[e.selected]
}

// SelectedModule returns the currently selected module impact.
func (e *Plugin) SelectedModule() *ModuleImpact {
	if e.selected < len(e.modules) {
		return &e.modules[e.selected]
	}
	return nil
}

// View renders the blast radius plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Run a plan first to visualize blast radius...")

	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Loading risk assessment... " + e.timer.FormatElapsed())
	case sdk.StatusError:
		return sdk.StyleError.Render("risk unavailable: cancelled")
	case sdk.StatusDone:
		return e.renderBlastRadius(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderBlastRadius(width, height int) string {
	if len(e.modules) == 0 {
		if !e.assessment.Usable() {
			return sdk.StyleError.Render("Blast radius unavailable: incomplete risk assessment")
		}
		return sdk.StyleSuccess.Render("No changes. Blast radius is zero.")
	}

	rows := []string{}
	cursor, cursorEnd := 0, 0
	for i, mi := range e.modules {
		if i == e.selected {
			cursor = len(rows)
		}
		rows = append(rows, e.renderModuleRow(mi, i))
		if e.expanded[i] {
			rows = append(rows, e.renderModuleChanges(mi)...)
		}
		if i == e.selected {
			cursorEnd = len(rows)
		}
	}

	// The summary and the blank line under it take two rows, unless the cursor's module
	// is taller than what is left: it gets them, so its last line shows.
	listHeight, header := height-2, ansi.Truncate(e.renderOverallSummary(), max(0, width), "")+"\n\n"
	if cursorEnd-cursor > listHeight {
		listHeight, header = height, ""
	}
	content, offset := e.panel.RenderWindow(ui.RenderParams{Rows: rows, Width: width, Height: max(1, listHeight), Cursor: cursor, CursorEnd: cursorEnd, ScrollOffset: e.offset})
	e.offset = offset

	lines := strings.Split(header+content, "\n")
	return strings.Join(lines[:min(max(0, height), len(lines))], "\n")
}

func (e *Plugin) renderOverallSummary() string {
	moduleCount := len(e.modules)
	maxImpact := ImpactMinimal
	for _, m := range e.modules {
		if m.Unavailable {
			return sdk.StyleError.Render("Blast radius unavailable: incomplete risk assessment") + "  " + sdk.StyleFaint.Render(fmt.Sprintf("%d module(s) affected, %d total change(s)", moduleCount, e.total))
		}
		if m.Score > maxImpact {
			maxImpact = m.Score
		}
	}

	summary := fmt.Sprintf("%d module(s) affected, %d total change(s)", moduleCount, e.total)

	switch maxImpact {
	case ImpactCritical:
		return sdk.StyleRiskCritical.Render("CRITICAL BLAST RADIUS") + "  " + sdk.StyleFaint.Render(summary)
	case ImpactHigh:
		return sdk.StyleRiskHigh.Render("High blast radius") + "  " + sdk.StyleFaint.Render(summary)
	case ImpactModerate:
		return sdk.StyleRiskMedium.Render("Moderate blast radius") + "  " + sdk.StyleFaint.Render(summary)
	default:
		return sdk.StyleRiskLow.Render("Minimal blast radius") + "  " + sdk.StyleFaint.Render(summary)
	}
}

func (e *Plugin) renderModuleRow(mi ModuleImpact, idx int) string {
	indicator := ">"
	if e.expanded[idx] {
		indicator = "v"
	}

	module := mi.Group.Module
	impactBadge := renderImpactBadge(mi.Score)
	if mi.Unavailable {
		impactBadge = sdk.StyleError.Render("[unavailable]")
	}
	if mi.NotAssessed {
		impactBadge += " Risk: not assessed"
	}
	changeCount := sdk.StyleFaint.Render(fmt.Sprintf("(%d changes)", len(mi.Group.Changes)))

	// Render action summary bar
	bar := renderActionBar(mi.Group.Summary)

	return fmt.Sprintf(" %s %s %s  %s  %s", indicator, module, changeCount, impactBadge, bar)
}

// renderModuleChanges returns the rows of an expanded module's changes, ending in a blank row.
func (e *Plugin) renderModuleChanges(mi ModuleImpact) []string {
	var rows []string
	for _, change := range mi.Group.Changes {
		symbol := sdk.ActionSymbol(change.Action)
		address := change.Resource.Address
		// Strip module prefix from address for cleaner display
		if mi.Group.Module != "root" && strings.HasPrefix(address, mi.Group.Module+".") {
			address = strings.TrimPrefix(address, mi.Group.Module+".")
		}

		risk := sdk.RiskBadge(change.Risk)
		resource := e.resourceAssessment(change.Resource.InstanceKey())
		switch resource.Status {
		case sdk.ResourceNotAssessed:
			risk = "Risk: not assessed"
		case sdk.ResourceClassified:
		default:
			risk = "risk unavailable: " + string(resource.ErrorCode)
		}
		row := fmt.Sprintf("     %s %s", symbol, address)
		if risk != "" {
			row += " " + risk
		}
		if change.IsPhantom {
			row += " " + sdk.StylePhantom.Render("(phantom)")
		}
		rows = append(rows, row)
		for _, reason := range resource.Reasons {
			rows = append(rows, "       "+reason.Message)
		}
	}
	return append(rows, "")
}

func renderImpactBadge(score ImpactScore) string {
	switch score {
	case ImpactCritical:
		return sdk.StyleRiskCritical.Render("[CRITICAL]")
	case ImpactHigh:
		return sdk.StyleRiskHigh.Render("[HIGH]")
	case ImpactModerate:
		return sdk.StyleRiskMedium.Render("[moderate]")
	case ImpactMinimal:
		return sdk.StyleRiskLow.Render("[minimal]")
	default:
		return ""
	}
}

func renderActionBar(summary sdk.ActionSummary) string {
	var parts []string
	if summary.Add > 0 {
		parts = append(parts, sdk.StyleCreate.Render(fmt.Sprintf("+%d", summary.Add)))
	}
	if summary.Change > 0 {
		parts = append(parts, sdk.StyleUpdate.Render(fmt.Sprintf("~%d", summary.Change)))
	}
	if summary.Destroy > 0 {
		parts = append(parts, sdk.StyleDelete.Render(fmt.Sprintf("-%d", summary.Destroy)))
	}
	if summary.Replace > 0 {
		parts = append(parts, sdk.StyleReplace.Render(fmt.Sprintf("-/+%d", summary.Replace)))
	}
	return strings.Join(parts, " ")
}

func calculateImpact(group sdk.ModuleGroup) ImpactScore {
	changeCount := len(group.Changes)
	maxRisk := sdk.RiskNone

	hasDestructive := false
	for _, c := range group.Changes {
		if c.Risk > maxRisk {
			maxRisk = c.Risk
		}
		if c.Action == sdk.ActionDelete || c.Action == sdk.ActionDeleteThenCreate || c.Action == sdk.ActionCreateThenDelete {
			hasDestructive = true
		}
	}

	switch {
	case maxRisk >= sdk.RiskCritical:
		return ImpactCritical
	case maxRisk >= sdk.RiskHigh || (hasDestructive && changeCount >= 3):
		return ImpactHigh
	case changeCount >= 3 || maxRisk >= sdk.RiskMedium:
		return ImpactModerate
	default:
		return ImpactMinimal
	}
}

func sortByImpact(modules []ModuleImpact) {
	sort.Slice(modules, func(i, j int) bool {
		if modules[i].Unavailable != modules[j].Unavailable {
			return modules[i].Unavailable
		}
		if modules[i].Score != modules[j].Score {
			return modules[i].Score > modules[j].Score
		}
		return modules[i].Group.Module < modules[j].Group.Module
	})
}
