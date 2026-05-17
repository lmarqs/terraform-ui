package blastradius

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// ImpactScore represents the calculated impact of a module group.
type ImpactScore int

const (
	ImpactMinimal  ImpactScore = iota // 1-2 changes, all low risk
	ImpactModerate                    // 3-5 changes or medium risk
	ImpactHigh                        // 6+ changes or high/critical risk
	ImpactCritical                    // destructive operations on critical infra
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
	Group sdk.ModuleGroup
	Score ImpactScore
}

// Plugin implements the blast radius visualization feature.
type Plugin struct {
	svc      sdk.Service
	status   sdk.Status
	modules  []ModuleImpact
	selected int
	expanded map[int]bool
	total    int
}

// New creates a new blast radius plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc:      svc,
		expanded: make(map[int]bool),
	}
}

func (e *Plugin) ID() string          { return "blastradius" }
func (e *Plugin) Name() string        { return "Blast Radius" }
func (e *Plugin) Description() string { return "Visualize module-grouped changes with impact scores" }
func (e *Plugin) Ready() bool         { return e.status == sdk.StatusDone }
func (e *Plugin) Status() sdk.Status  { return e.status }
func (e *Plugin) Selected() int       { return e.selected }
func (e *Plugin) ModuleCount() int    { return len(e.modules) }
func (e *Plugin) TotalChanges() int   { return e.total }
func (e *Plugin) Count() (int, int)   { return len(e.modules), e.total }
func (e *Plugin) CursorPosition() (int, int) {
	if e.status != sdk.StatusDone || len(e.modules) == 0 {
		return 0, 0
	}
	return e.selected + 1, len(e.modules)
}

// Hints returns context-sensitive key hints for the status bar.
func (e *Plugin) Hints() []sdk.KeyHint {
	if e.status == sdk.StatusDone && len(e.modules) > 0 {
		return (sdk.HintSetInspect | sdk.HintSetBack).Hints()
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

// Analyze processes a plan summary, groups by module, and calculates impact scores.
func (e *Plugin) Analyze(summary *sdk.PlanSummary) {
	if summary == nil || len(summary.Changes) == 0 {
		e.status = sdk.StatusDone
		e.modules = nil
		e.total = 0
		return
	}

	e.total = len(summary.Changes)

	groups := sdk.GroupByModule(summary.Changes)
	e.modules = make([]ModuleImpact, 0, len(groups))

	for _, g := range groups {
		score := calculateImpact(g)
		e.modules = append(e.modules, ModuleImpact{
			Group: g,
			Score: score,
		})
	}

	// Sort by impact score descending (highest impact first)
	sortByImpact(e.modules)

	e.status = sdk.StatusDone
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

	case sdk.StatusDone:
		return e.renderBlastRadius(width, height)

	default:
		return ""
	}
}

func (e *Plugin) renderBlastRadius(width, height int) string {
	if len(e.modules) == 0 {
		return sdk.StyleSuccess.Render("No changes. Blast radius is zero.")
	}

	var b strings.Builder

	// Overall blast summary
	overallLine := e.renderOverallSummary()
	b.WriteString(overallLine)
	b.WriteString("\n\n")

	// Calculate visible area
	maxVisible := height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}

	startIdx := 0
	if e.selected >= maxVisible {
		startIdx = e.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(e.modules) {
		endIdx = len(e.modules)
	}

	for i := startIdx; i < endIdx; i++ {
		mi := e.modules[i]
		row := e.renderModuleRow(mi, i)
		if i == e.selected {
			row = sdk.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')

		// Render expanded changes
		if e.expanded[i] {
			b.WriteString(e.renderModuleChanges(mi, width))
		}
	}

	return b.String()
}

func (e *Plugin) renderOverallSummary() string {
	moduleCount := len(e.modules)
	maxImpact := ImpactMinimal
	for _, m := range e.modules {
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
	changeCount := sdk.StyleFaint.Render(fmt.Sprintf("(%d changes)", len(mi.Group.Changes)))

	// Render action summary bar
	bar := renderActionBar(mi.Group.Summary)

	return fmt.Sprintf(" %s %s %s  %s  %s", indicator, module, changeCount, impactBadge, bar)
}

func (e *Plugin) renderModuleChanges(mi ModuleImpact, width int) string {
	var b strings.Builder
	for _, change := range mi.Group.Changes {
		symbol := sdk.ActionSymbol(change.Action)
		address := change.Resource.Address
		// Strip module prefix from address for cleaner display
		if mi.Group.Module != "root" && strings.HasPrefix(address, mi.Group.Module+".") {
			address = strings.TrimPrefix(address, mi.Group.Module+".")
		}

		risk := sdk.RiskBadge(change.Risk)
		row := fmt.Sprintf("     %s %s", symbol, address)
		if risk != "" {
			row += " " + risk
		}
		if change.IsPhantom {
			row += " " + sdk.StylePhantom.Render("(phantom)")
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return b.String()
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
		return modules[i].Score > modules[j].Score
	})
}
