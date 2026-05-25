package validate

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// ValidateResultMsg is sent when the validate operation completes.
type ValidateResultMsg struct {
	Diagnostics []sdk.Diagnostic
	Err         error
}

// Plugin implements the terraform validate feature.
type Plugin struct {
	sdk.PluginBase
	expander    *ui.ExpandSet
	timer       ui.Timer
	status      sdk.Status
	diagnostics []sdk.Diagnostic
	errMsg      string
	selected    int
	input       Input
	cancelFn    context.CancelFunc
}

// New creates a new validate plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		PluginBase: sdk.NewPluginBase("validate", "Validate", "Run terraform validate"),
		expander:   ui.NewExpandSet(),
	}
	p.Svc = svc
	return p
}

func (p *Plugin) Ready() bool        { return p.status == sdk.StatusDone }
func (p *Plugin) Status() sdk.Status { return p.status }
func (p *Plugin) Selected() int      { return p.selected }

func (p *Plugin) Diagnostics() []sdk.Diagnostic {
	return p.diagnostics
}

// Hints returns context-sensitive key hints for the status bar.
func (p *Plugin) Hints() []sdk.KeyHint {
	switch p.status {
	case sdk.StatusIdle:
		return (sdk.HintSetConfirm | sdk.HintSetQuit).Hints()
	case sdk.StatusLoading:
		return (sdk.HintSetQuit).Hints()
	case sdk.StatusError:
		return (sdk.HintSetRetry | sdk.HintSetQuit).Hints()
	case sdk.StatusDone:
		if len(p.diagnostics) == 0 {
			return (sdk.HintSetRefresh | sdk.HintSetQuit).Hints()
		}
		return (sdk.HintSetInspect | sdk.HintSetRefresh | sdk.HintSetQuit).Hints()
	default:
		return (sdk.HintSetQuit).Hints()
	}
}

// Configure applies plugin-specific options from config.
func (p *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init wires the plugin to its shared dependencies.
func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.InitBase(deps)
	p.reset()
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if !p.HandleContextChangedDefault(ev) {
		return nil
	}
	p.reset()
	return nil
}

// reset clears all plugin state to initial values.
func (p *Plugin) reset() {
	p.status = sdk.StatusIdle
	p.diagnostics = nil
	p.errMsg = ""
	p.selected = 0
	p.expander.CollapseAll()
}

// Activate stores the typed input and returns the initial command.
func (p *Plugin) Activate(input Input) tea.Cmd {
	p.input = input
	if p.status == sdk.StatusIdle || p.status == sdk.StatusError {
		p.status = sdk.StatusLoading
		p.Log.Debug("validate.start")
		return tea.Batch(p.runValidate(), p.timer.Start())
	}
	return nil
}

// Refresh re-runs terraform validate.
func (p *Plugin) Refresh() tea.Cmd {
	p.status = sdk.StatusLoading
	p.diagnostics = nil
	p.errMsg = ""
	p.selected = 0
	p.expander.CollapseAll()
	return tea.Batch(p.runValidate(), p.timer.Start())
}

// Cancel aborts any in-flight terraform operation.
func (p *Plugin) Cancel() {
	if p.cancelFn != nil {
		p.cancelFn()
		p.cancelFn = nil
	}
}

func (p *Plugin) runValidate() tea.Cmd {
	p.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFn = cancel
	svc := p.Svc
	return func() tea.Msg {
		diags, err := svc.Validate(ctx)
		return ValidateResultMsg{Diagnostics: diags, Err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return p, p.timer.Tick()

	case ValidateResultMsg:
		p.timer.Stop()
		if msg.Err != nil {
			p.status = sdk.StatusError
			p.errMsg = msg.Err.Error()
			p.Log.Debug("validate.error", "error", msg.Err.Error())
		} else {
			p.status = sdk.StatusDone
			p.diagnostics = sortDiagnostics(msg.Diagnostics)
			p.Log.Debug("validate.complete", "diagnostics", len(p.diagnostics))
		}
		return p, nil

	case tea.KeyMsg:
		cmd := p.handleKey(msg)
		return p, cmd
	}
	return p, nil
}

func (p *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		p.MoveDown()
	case "k", "up":
		p.MoveUp()
	case "enter", "i":
		p.ToggleExpand()
	case "ctrl+r":
		if p.status == sdk.StatusError || p.status == sdk.StatusDone {
			return p.Refresh()
		}
	case "G":
		p.MoveToEnd()
	case "g":
		p.MoveToStart()
	}
	return nil
}

// MoveUp moves selection up.
func (p *Plugin) MoveUp() {
	if p.selected > 0 {
		p.selected--
	}
}

// MoveDown moves selection down.
func (p *Plugin) MoveDown() {
	if p.diagnostics != nil && p.selected < len(p.diagnostics)-1 {
		p.selected++
	}
}

// MoveToStart moves selection to the first item.
func (p *Plugin) MoveToStart() {
	p.selected = 0
}

// MoveToEnd moves selection to the last item.
func (p *Plugin) MoveToEnd() {
	if len(p.diagnostics) > 0 {
		p.selected = len(p.diagnostics) - 1
	}
}

// ToggleExpand toggles detail expansion for the selected diagnostic.
func (p *Plugin) ToggleExpand() {
	p.expander.Toggle(p.selected)
}

// IsExpanded returns whether a diagnostic row is expanded.
func (p *Plugin) IsExpanded(idx int) bool {
	return p.expander.IsExpanded(idx)
}

// View renders the validate plugin.
func (p *Plugin) View(width, height int) string {
	switch p.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Ready to validate.")

	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Running terraform validate... " + p.timer.FormatElapsed())

	case sdk.StatusError:
		return sdk.StyleError.Render("Error: " + p.errMsg)

	case sdk.StatusDone:
		return p.renderResults(width, height)

	default:
		return ""
	}
}

func (p *Plugin) renderResults(width, height int) string {
	if len(p.diagnostics) == 0 {
		return sdk.StyleSuccess.Render("✓ Configuration is valid")
	}

	var b strings.Builder

	// Calculate visible area
	maxVisible := height - 6
	if maxVisible < 3 {
		maxVisible = 3
	}

	// Determine scroll window
	startIdx := 0
	if p.selected >= maxVisible {
		startIdx = p.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(p.diagnostics) {
		endIdx = len(p.diagnostics)
	}

	for i := startIdx; i < endIdx; i++ {
		diag := p.diagnostics[i]
		row := p.renderDiagnosticRow(diag)
		if i == p.selected {
			row = sdk.StyleSelected.Width(width).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')

		// Render expanded detail
		if p.expander.IsExpanded(i) && diag.Detail != "" {
			detail := sdk.StyleFaint.Render("    " + diag.Detail)
			b.WriteString(detail)
			b.WriteByte('\n')
		}
	}

	summary := p.renderSummaryLine()

	return b.String() + "\n" + summary
}

func (p *Plugin) renderDiagnosticRow(diag sdk.Diagnostic) string {
	var icon string
	if diag.Severity.IsError() {
		icon = sdk.StyleError.Render("✗")
	} else {
		icon = sdk.StyleUpdate.Render("⚠")
	}

	location := ""
	if diag.File != "" {
		if diag.Line > 0 {
			location = sdk.StyleFaint.Render(fmt.Sprintf(" %s:%d", diag.File, diag.Line))
		} else {
			location = sdk.StyleFaint.Render(" " + diag.File)
		}
	}

	return fmt.Sprintf(" %s %s%s", icon, diag.Summary, location)
}

func (p *Plugin) renderSummaryLine() string {
	errors := 0
	warnings := 0
	for _, d := range p.diagnostics {
		if d.Severity.IsError() {
			errors++
		} else {
			warnings++
		}
	}

	parts := []string{}
	if errors > 0 {
		parts = append(parts, sdk.StyleError.Render(fmt.Sprintf("%d error(s)", errors)))
	}
	if warnings > 0 {
		parts = append(parts, sdk.StyleUpdate.Render(fmt.Sprintf("%d warning(s)", warnings)))
	}
	return strings.Join(parts, ", ")
}

// Stdout produces stdout content for standalone/CI mode. The plugin reads
// p.input.JSON to decide between human-readable and JSON output.
func (p *Plugin) Stdout() ([]byte, error) {
	if p.input.JSON {
		errorCount, warningCount := 0, 0
		for _, d := range p.diagnostics {
			if d.Severity.IsError() {
				errorCount++
			} else {
				warningCount++
			}
		}
		type diagJSON struct {
			Severity string `json:"severity"`
			Summary  string `json:"summary"`
			Detail   string `json:"detail,omitempty"`
			File     string `json:"file,omitempty"`
			Line     int    `json:"line,omitempty"`
		}
		out := struct {
			Valid        bool       `json:"valid"`
			ErrorCount   int        `json:"error_count"`
			WarningCount int        `json:"warning_count"`
			Diagnostics  []diagJSON `json:"diagnostics"`
		}{
			Valid:        errorCount == 0,
			ErrorCount:   errorCount,
			WarningCount: warningCount,
			Diagnostics:  make([]diagJSON, 0, len(p.diagnostics)),
		}
		for _, d := range p.diagnostics {
			out.Diagnostics = append(out.Diagnostics, diagJSON{
				Severity: string(d.Severity),
				Summary:  d.Summary,
				Detail:   d.Detail,
				File:     d.File,
				Line:     d.Line,
			})
		}
		return sdk.MarshalJSON(out), nil
	}

	if len(p.diagnostics) == 0 {
		return []byte("✓ Configuration is valid\n"), nil
	}
	var b strings.Builder
	for _, d := range p.diagnostics {
		icon := "✗"
		if d.Severity.IsWarning() {
			icon = "⚠"
		}
		fmt.Fprintf(&b, "%s %s", icon, d.Summary)
		if d.File != "" {
			fmt.Fprintf(&b, " (%s", d.File)
			if d.Line > 0 {
				fmt.Fprintf(&b, ":%d", d.Line)
			}
			b.WriteString(")")
		}
		b.WriteString("\n")
		if d.Detail != "" {
			fmt.Fprintf(&b, "  %s\n", d.Detail)
		}
	}
	return []byte(b.String()), nil
}

// ExitCode returns 1 if validation has errors, 0 otherwise.
func (p *Plugin) ExitCode() int {
	for _, d := range p.diagnostics {
		if d.Severity.IsError() {
			return 1
		}
	}
	return 0
}

func sortDiagnostics(diags []sdk.Diagnostic) []sdk.Diagnostic {
	if diags == nil {
		return nil
	}
	sorted := make([]sdk.Diagnostic, 0, len(diags))
	for _, d := range diags {
		if d.Severity.IsError() {
			sorted = append(sorted, d)
		}
	}
	for _, d := range diags {
		if d.Severity != "error" {
			sorted = append(sorted, d)
		}
	}
	return sorted
}
