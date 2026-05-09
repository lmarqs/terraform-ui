package blastradius

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

// Status represents the current state of the blast radius extension.
type Status int

const (
	StatusIdle Status = iota
	StatusReady
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
	Group terraform.ModuleGroup
	Score ImpactScore
}

// Extension implements the blast radius visualization feature.
type Extension struct {
	svc      terraform.Service
	status   Status
	modules  []ModuleImpact
	selected int
	expanded map[int]bool
	total    int
}

// New creates a new blast radius extension.
func New() *Extension {
	return &Extension{
		expanded: make(map[int]bool),
	}
}

func (e *Extension) Name() string        { return "Blast Radius" }
func (e *Extension) Description() string  { return "Visualize module-grouped changes with impact scores" }
func (e *Extension) KeyBinding() string   { return "b" }
func (e *Extension) Ready() bool          { return e.status == StatusReady }
func (e *Extension) Status() Status       { return e.status }
func (e *Extension) Selected() int        { return e.selected }
func (e *Extension) ModuleCount() int     { return len(e.modules) }
func (e *Extension) TotalChanges() int    { return e.total }

// Init initializes the extension with a terraform service.
func (e *Extension) Init(svc terraform.Service) tea.Cmd {
	e.svc = svc
	return nil
}

// Analyze processes a plan summary, groups by module, and calculates impact scores.
func (e *Extension) Analyze(summary *terraform.PlanSummary) {
	if summary == nil || len(summary.Changes) == 0 {
		e.status = StatusReady
		e.modules = nil
		e.total = 0
		return
	}

	e.total = len(summary.Changes)

	groups := terraform.GroupByModule(summary.Changes)
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

	e.status = StatusReady
	e.selected = 0
	e.expanded = make(map[int]bool)
}

// Update processes messages and returns the updated extension.
func (e *Extension) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return e.handleKey(msg), true
	}
	return nil, false
}

func (e *Extension) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		e.MoveDown()
	case "k", "up":
		e.MoveUp()
	case "enter", " ":
		e.ToggleExpand()
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
	if e.selected < len(e.modules)-1 {
		e.selected++
	}
}

// ToggleExpand toggles the expanded view for the selected module.
func (e *Extension) ToggleExpand() {
	e.expanded[e.selected] = !e.expanded[e.selected]
}

// SelectedModule returns the currently selected module impact.
func (e *Extension) SelectedModule() *ModuleImpact {
	if e.selected < len(e.modules) {
		return &e.modules[e.selected]
	}
	return nil
}

// View renders the blast radius extension.
func (e *Extension) View(width, height int) string {
	title := styles.StyleTitle.Render("Blast Radius")

	switch e.status {
	case StatusIdle:
		placeholder := styles.StyleFaintItalic.Render("Run a plan first to visualize blast radius...")
		return styles.StylePadded.Render(title + "\n\n" + placeholder)

	case StatusReady:
		return e.renderBlastRadius(width, height)

	default:
		return ""
	}
}

func (e *Extension) renderBlastRadius(width, height int) string {
	title := styles.StyleTitle.Render("Blast Radius")

	if len(e.modules) == 0 {
		noChanges := styles.StyleSuccess.Render("No changes. Blast radius is zero.")
		return styles.StylePadded.Render(title + "\n\n" + noChanges)
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
			row = styles.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')

		// Render expanded changes
		if e.expanded[i] {
			b.WriteString(e.renderModuleChanges(mi, width))
		}
	}

	hint := styles.StyleFaintItalic.Render("j/k navigate  Enter expand  Esc back")
	content := title + "\n\n" + b.String() + "\n" + hint
	return styles.StylePadded.Render(content)
}

func (e *Extension) renderOverallSummary() string {
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
		return styles.StyleRiskCritical.Render("CRITICAL BLAST RADIUS") + "  " + styles.StyleFaint.Render(summary)
	case ImpactHigh:
		return styles.StyleRiskHigh.Render("High blast radius") + "  " + styles.StyleFaint.Render(summary)
	case ImpactModerate:
		return styles.StyleRiskMedium.Render("Moderate blast radius") + "  " + styles.StyleFaint.Render(summary)
	default:
		return styles.StyleRiskLow.Render("Minimal blast radius") + "  " + styles.StyleFaint.Render(summary)
	}
}

func (e *Extension) renderModuleRow(mi ModuleImpact, idx int) string {
	indicator := ">"
	if e.expanded[idx] {
		indicator = "v"
	}

	module := mi.Group.Module
	impactBadge := renderImpactBadge(mi.Score)
	changeCount := styles.StyleFaint.Render(fmt.Sprintf("(%d changes)", len(mi.Group.Changes)))

	// Render action summary bar
	bar := renderActionBar(mi.Group.Summary)

	return fmt.Sprintf(" %s %s %s  %s  %s", indicator, module, changeCount, impactBadge, bar)
}

func (e *Extension) renderModuleChanges(mi ModuleImpact, width int) string {
	var b strings.Builder
	for _, change := range mi.Group.Changes {
		symbol := actionSymbol(change.Action)
		address := change.Resource.Address
		// Strip module prefix from address for cleaner display
		if mi.Group.Module != "root" && strings.HasPrefix(address, mi.Group.Module+".") {
			address = strings.TrimPrefix(address, mi.Group.Module+".")
		}

		risk := riskBadge(change.Risk)
		row := fmt.Sprintf("     %s %s", symbol, address)
		if risk != "" {
			row += " " + risk
		}
		if change.IsPhantom {
			row += " " + styles.StylePhantom.Render("(phantom)")
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
		return styles.StyleRiskCritical.Render("[CRITICAL]")
	case ImpactHigh:
		return styles.StyleRiskHigh.Render("[HIGH]")
	case ImpactModerate:
		return styles.StyleRiskMedium.Render("[moderate]")
	case ImpactMinimal:
		return styles.StyleRiskLow.Render("[minimal]")
	default:
		return ""
	}
}

func renderActionBar(summary terraform.ActionSummary) string {
	var parts []string
	if summary.Add > 0 {
		parts = append(parts, styles.StyleCreate.Render(fmt.Sprintf("+%d", summary.Add)))
	}
	if summary.Change > 0 {
		parts = append(parts, styles.StyleUpdate.Render(fmt.Sprintf("~%d", summary.Change)))
	}
	if summary.Destroy > 0 {
		parts = append(parts, styles.StyleDelete.Render(fmt.Sprintf("-%d", summary.Destroy)))
	}
	if summary.Replace > 0 {
		parts = append(parts, styles.StyleReplace.Render(fmt.Sprintf("-/+%d", summary.Replace)))
	}
	return strings.Join(parts, " ")
}

func calculateImpact(group terraform.ModuleGroup) ImpactScore {
	changeCount := len(group.Changes)
	maxRisk := terraform.RiskNone

	hasDestructive := false
	for _, c := range group.Changes {
		if c.Risk > maxRisk {
			maxRisk = c.Risk
		}
		if c.Action == terraform.ActionDelete || c.Action == terraform.ActionDeleteThenCreate || c.Action == terraform.ActionCreateThenDelete {
			hasDestructive = true
		}
	}

	switch {
	case maxRisk >= terraform.RiskCritical:
		return ImpactCritical
	case maxRisk >= terraform.RiskHigh || (hasDestructive && changeCount >= 3):
		return ImpactHigh
	case changeCount >= 3 || maxRisk >= terraform.RiskMedium:
		return ImpactModerate
	default:
		return ImpactMinimal
	}
}

func sortByImpact(modules []ModuleImpact) {
	// Simple insertion sort (module list is typically small)
	for i := 1; i < len(modules); i++ {
		for j := i; j > 0 && modules[j].Score > modules[j-1].Score; j-- {
			modules[j], modules[j-1] = modules[j-1], modules[j]
		}
	}
}

func actionSymbol(action terraform.Action) string {
	switch action {
	case terraform.ActionCreate:
		return styles.StyleCreate.Render("+")
	case terraform.ActionUpdate:
		return styles.StyleUpdate.Render("~")
	case terraform.ActionDelete:
		return styles.StyleDelete.Render("-")
	case terraform.ActionDeleteThenCreate, terraform.ActionCreateThenDelete:
		return styles.StyleReplace.Render("-/+")
	default:
		return " "
	}
}

func riskBadge(risk terraform.RiskLevel) string {
	switch risk {
	case terraform.RiskLow:
		return styles.StyleRiskLow.Render("[low]")
	case terraform.RiskMedium:
		return styles.StyleRiskMedium.Render("[medium]")
	case terraform.RiskHigh:
		return styles.StyleRiskHigh.Render("[HIGH]")
	case terraform.RiskCritical:
		return styles.StyleRiskCritical.Render("[CRITICAL]")
	default:
		return ""
	}
}
