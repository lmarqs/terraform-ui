package validate

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Status represents the current state of the validate plugin.
type Status int

const (
	StatusIdle Status = iota
	StatusLoading
	StatusDone
	StatusError
)

// ValidateResultMsg is sent when the validate operation completes.
type ValidateResultMsg struct {
	Diagnostics []sdk.Diagnostic
	Err         error
}

// Plugin implements the terraform validate feature.
type Plugin struct {
	svc           sdk.Service
	log           *slog.Logger
	session       *sdk.Session
	status        Status
	diagnostics   []sdk.Diagnostic
	errMsg        string
	selected      int
	expanded      map[int]bool
	scopedContext string
}

// New creates a new validate plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		expanded: make(map[int]bool),
		svc:      svc,
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func (p *Plugin) ID() string          { return "validate" }
func (p *Plugin) Name() string        { return "Validate" }
func (p *Plugin) Description() string { return "Run terraform validate" }
func (p *Plugin) KeyBinding() string  { return "v" }
func (p *Plugin) Ready() bool         { return p.status == StatusDone }
func (p *Plugin) Status() Status      { return p.status }
func (p *Plugin) Selected() int       { return p.selected }

func (p *Plugin) Diagnostics() []sdk.Diagnostic {
	return p.diagnostics
}

// Configure applies plugin-specific options from config.
func (p *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context.
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	p.log = ctx.Logger
	p.session = ctx.Session
	p.status = StatusIdle
	p.diagnostics = nil
	p.errMsg = ""
	p.selected = 0
	p.expanded = make(map[int]bool)
	return nil
}

// Activate triggers validate when the user enters the plugin view.
func (p *Plugin) Activate() tea.Cmd {
	// Check if the active context changed since last activation
	if p.session != nil {
		currentContext, _ := sdk.GetTyped[string](p.session, sdk.SessionKeyActiveContextAbs)
		if currentContext != p.scopedContext {
			p.status = StatusIdle
			p.diagnostics = nil
			p.errMsg = ""
			p.selected = 0
			p.expanded = make(map[int]bool)
			p.scopedContext = currentContext
			if currentContext != "" {
				p.svc = p.svc.WithDir(currentContext)
			}
		}
	}

	if p.status == StatusIdle || p.status == StatusError {
		if p.session != nil {
			if dir, ok := sdk.GetTyped[string](p.session, sdk.SessionKeyActiveContextAbs); ok && dir != "" {
				p.svc = p.svc.WithDir(dir)
				p.scopedContext = dir
			} else if count, ok := sdk.GetTyped[int](p.session, sdk.SessionKeyContextCount); ok && count > 1 {
				p.status = StatusError
				p.errMsg = "Select a context first (press c)"
				return nil
			}
		}
		p.status = StatusLoading
		p.log.Debug("validate.start")
		return p.runValidate()
	}
	return nil
}

// Refresh re-runs terraform validate.
func (p *Plugin) Refresh() tea.Cmd {
	p.status = StatusLoading
	p.diagnostics = nil
	p.errMsg = ""
	p.selected = 0
	p.expanded = make(map[int]bool)
	return p.runValidate()
}

func (p *Plugin) runValidate() tea.Cmd {
	svc := p.svc
	return func() tea.Msg {
		diags, err := svc.Validate(context.Background())
		return ValidateResultMsg{Diagnostics: diags, Err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ValidateResultMsg:
		if msg.Err != nil {
			p.status = StatusError
			p.errMsg = msg.Err.Error()
			p.log.Debug("validate.error", "error", msg.Err.Error())
		} else {
			p.status = StatusDone
			p.diagnostics = sortDiagnostics(msg.Diagnostics)
			p.log.Debug("validate.complete", "diagnostics", len(p.diagnostics))
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
	case "r":
		if p.status == StatusError || p.status == StatusDone {
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
	if p.diagnostics != nil && len(p.diagnostics) > 0 {
		p.selected = len(p.diagnostics) - 1
	}
}

// ToggleExpand toggles detail expansion for the selected diagnostic.
func (p *Plugin) ToggleExpand() {
	p.expanded[p.selected] = !p.expanded[p.selected]
}

// IsExpanded returns whether a diagnostic row is expanded.
func (p *Plugin) IsExpanded(idx int) bool {
	return p.expanded[idx]
}

// View renders the validate plugin.
func (p *Plugin) View(width, height int) string {
	switch p.status {
	case StatusIdle:
		title := sdk.StyleTitle.Render("Validate")
		placeholder := sdk.StyleFaintItalic.Render("Press Enter to run terraform validate...")
		return sdk.StylePadded.Render(title + "\n\n" + placeholder)

	case StatusLoading:
		title := sdk.StyleTitle.Render("Validate")
		loading := sdk.StyleFaintItalic.Render("Running terraform validate...")
		return sdk.StylePadded.Render(title + "\n\n" + loading)

	case StatusError:
		title := sdk.StyleTitle.Render("Validate")
		errText := sdk.StyleError.Render("Error: " + p.errMsg)
		hint := sdk.StyleFaintItalic.Render("Press r to retry, q to go back")
		return sdk.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	case StatusDone:
		return p.renderResults(width, height)

	default:
		return ""
	}
}

func (p *Plugin) renderResults(width, height int) string {
	title := sdk.StyleTitle.Render("Validate")

	if len(p.diagnostics) == 0 {
		success := sdk.StyleSuccess.Render("✓ Configuration is valid")
		hint := sdk.StyleFaintItalic.Render("Press r to re-validate, q to go back")
		return sdk.StylePadded.Render(title + "\n\n" + success + "\n\n" + hint)
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
			row = sdk.StyleSelected.Width(width - 6).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')

		// Render expanded detail
		if p.expanded[i] && diag.Detail != "" {
			detail := sdk.StyleFaint.Render("    " + diag.Detail)
			b.WriteString(detail)
			b.WriteByte('\n')
		}
	}

	summary := p.renderSummaryLine()
	hint := sdk.StyleFaintItalic.Render("j/k navigate  Enter expand  r refresh  q back")

	content := title + "\n\n" + b.String() + "\n" + summary + "\n" + hint
	return sdk.StylePadded.Render(content)
}

func (p *Plugin) renderDiagnosticRow(diag sdk.Diagnostic) string {
	var icon string
	if diag.Severity == "error" {
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
		if d.Severity == "error" {
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

// sortDiagnostics returns diagnostics sorted with errors first, then warnings.
func sortDiagnostics(diags []sdk.Diagnostic) []sdk.Diagnostic {
	if diags == nil {
		return nil
	}
	sorted := make([]sdk.Diagnostic, 0, len(diags))
	for _, d := range diags {
		if d.Severity == "error" {
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
