package init

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/editor"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func detectBinary(configured string) string {
	if configured != "" {
		return configured
	}
	if _, err := exec.LookPath("tofu"); err == nil {
		return "tofu"
	}
	return "terraform"
}

const (
	StatusMenu    = sdk.Status(10)
	StatusReview  = sdk.Status(11)
	StatusConfirm = sdk.Status(12)
)

// DetectedPattern represents a filesystem pattern found during scanning.
type DetectedPattern struct {
	Pattern string
	Enabled bool
}

// DetectionCompleteMsg is sent when filesystem scanning finishes.
type DetectionCompleteMsg struct {
	Binary   string
	Patterns []DetectedPattern
	Err      error
}

// WriteCompleteMsg is sent when the config file has been written.
type WriteCompleteMsg struct {
	Path string
	Err  error
}

// Plugin implements the init wizard that generates tfui.hcl interactively.
type Plugin struct {
	svc        sdk.Service
	dir        string
	status     sdk.Status
	binary     string
	patterns   []DetectedPattern
	selected   int
	errMsg     string
	preview    string
	configPath string
	hasConfig  bool
	menuItem   int
}

// New creates a new init plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc: svc,
	}
}

func (p *Plugin) ID() string          { return "init" }
func (p *Plugin) Name() string        { return "Init" }
func (p *Plugin) Description() string { return "Generate tfui.hcl configuration interactively" }
func (p *Plugin) Ready() bool         { return p.status == sdk.StatusDone }

// Hints returns context-sensitive key hints for the status bar.
func (p *Plugin) Hints() []sdk.KeyHint {
	switch p.status {
	case StatusMenu:
		return (sdk.HintSetSelect | sdk.HintSetBack).Hints()
	case sdk.StatusLoading:
		return (sdk.HintSetBack).Hints()
	case StatusReview:
		return (sdk.HintSetConfirm | sdk.HintSetCancel).Hints()
	case StatusConfirm:
		return (sdk.HintSetConfirm | sdk.HintSetCancel).Hints()
	case sdk.StatusDone:
		return (sdk.HintSetBack).Hints()
	case sdk.StatusError:
		return (sdk.HintSetBack).Hints()
	default:
		return (sdk.HintSetBack).Hints()
	}
}

// Configure applies plugin-specific options from config.
func (p *Plugin) Configure(opts map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context. Does not auto-detect —
// detection runs when the user activates the plugin.
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	p.dir = ctx.WorkingDir
	p.status = StatusMenu
	p.patterns = nil
	p.binary = ""
	p.errMsg = ""
	p.selected = 0
	p.preview = ""
	p.menuItem = 0
	p.configPath = filepath.Join(p.dir, config.HCLConfigFileName)
	_, err := os.Stat(p.configPath)
	p.hasConfig = err == nil
	return nil
}

// Activate triggers the menu when the user enters the plugin.
func (p *Plugin) Activate() tea.Cmd {
	p.configPath = filepath.Join(p.dir, config.HCLConfigFileName)
	_, err := os.Stat(p.configPath)
	p.hasConfig = err == nil
	p.status = StatusMenu
	p.menuItem = 0
	return nil
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case DetectionCompleteMsg:
		if msg.Err != nil {
			p.status = sdk.StatusError
			p.errMsg = msg.Err.Error()
		} else {
			p.status = StatusReview
			p.binary = msg.Binary
			p.patterns = msg.Patterns
		}
		return p, nil

	case WriteCompleteMsg:
		if msg.Err != nil {
			p.status = sdk.StatusError
			p.errMsg = msg.Err.Error()
		} else {
			p.status = sdk.StatusDone
		}
		return p, nil

	case editor.EditorClosedMsg:
		_, err := os.Stat(p.configPath)
		p.hasConfig = err == nil
		p.status = StatusMenu
		return p, nil

	case tea.KeyMsg:
		cmd := p.handleKey(msg)
		return p, cmd
	}
	return p, nil
}

func (p *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch p.status {
	case StatusMenu:
		return p.handleMenuKey(msg)
	case StatusReview:
		return p.handleReviewKey(msg)
	case StatusConfirm:
		return p.handleConfirmKey(msg)
	case sdk.StatusDone:
		if msg.String() == "esc" {
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	case sdk.StatusError:
		if msg.String() == "esc" {
			p.status = StatusMenu
		}
	}
	return nil
}

func (p *Plugin) handleMenuKey(msg tea.KeyMsg) tea.Cmd {
	menuLen := 1 // "Init new config"
	if p.hasConfig {
		menuLen = 2 // + "Edit existing config"
	}

	switch msg.String() {
	case "j", "down":
		if p.menuItem < menuLen-1 {
			p.menuItem++
		}
	case "k", "up":
		if p.menuItem > 0 {
			p.menuItem--
		}
	case "enter":
		if p.menuItem == 0 && p.hasConfig {
			return p.openEditor()
		}
		if (p.menuItem == 1 && p.hasConfig) || (p.menuItem == 0 && !p.hasConfig) {
			p.status = sdk.StatusLoading
			return p.detect()
		}
	case "e":
		if p.hasConfig {
			return p.openEditor()
		}
	case "esc":
		return func() tea.Msg { return sdk.DeactivateMsg{} }
	}
	return nil
}

func (p *Plugin) openEditor() tea.Cmd {
	loc := editor.SourceLocation{File: p.configPath, Line: 1}
	return editor.Open(loc)
}

func (p *Plugin) handleReviewKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if p.selected < len(p.patterns)-1 {
			p.selected++
		}
	case "k", "up":
		if p.selected > 0 {
			p.selected--
		}
	case " ":
		if p.selected < len(p.patterns) {
			p.patterns[p.selected].Enabled = !p.patterns[p.selected].Enabled
		}
	case "enter":
		p.preview = p.generateHCL()
		p.status = StatusConfirm
	}
	return nil
}

func (p *Plugin) handleConfirmKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		return p.writeConfig()
	case "esc":
		p.status = StatusReview
	}
	return nil
}

// View renders the init wizard UI.
func (p *Plugin) View(width, height int) string {
	switch p.status {
	case StatusMenu:
		return p.renderMenu(width, height)

	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Scanning filesystem for terraform projects...")

	case sdk.StatusError:
		return sdk.StyleError.Render("Error: " + p.errMsg)

	case StatusReview:
		return p.renderReview(width, height)

	case StatusConfirm:
		return p.renderConfirm(width, height)

	case sdk.StatusDone:
		return p.renderDone(width, height)

	default:
		return ""
	}
}

func (p *Plugin) renderMenu(width, height int) string {
	var b strings.Builder

	if p.hasConfig {
		configInfo := sdk.StyleSuccess.Render("✓") + " " + sdk.StyleFaint.Render(p.configPath)
		b.WriteString(configInfo + "\n\n")

		items := []string{
			"Edit config (open in $EDITOR)",
			"Re-init (detect patterns and regenerate)",
		}
		for i, item := range items {
			if i == p.menuItem {
				b.WriteString(sdk.StyleSelected.Width(width - 6).Render(" → " + item))
			} else {
				b.WriteString("   " + item)
			}
			b.WriteByte('\n')
		}
	} else {
		noConfig := sdk.StyleFaintItalic.Render("No tfui.hcl found in " + p.dir)
		b.WriteString(noConfig + "\n\n")

		item := "Init new config (detect patterns)"
		if p.menuItem == 0 {
			b.WriteString(sdk.StyleSelected.Width(width - 6).Render(" → " + item))
		} else {
			b.WriteString("   " + item)
		}
		b.WriteByte('\n')
	}

	editorName := editor.DetectEditor()
	editorInfo := sdk.StyleFaintItalic.Render(fmt.Sprintf("\neditor: %s", editorName))

	return b.String() + editorInfo
}

func (p *Plugin) renderReview(width, height int) string {
	var b strings.Builder

	// Binary detection
	binaryLabel := sdk.StyleKey.Render("Terraform binary: ")
	binaryValue := sdk.StyleSuccess.Render(p.binary)
	b.WriteString(binaryLabel + binaryValue + "\n\n")

	// Detected patterns
	if len(p.patterns) == 0 {
		noPatterns := sdk.StyleFaintItalic.Render("No terraform project patterns detected.")
		b.WriteString(noPatterns + "\n")
	} else {
		patternsLabel := sdk.StyleKey.Render("Detected patterns:")
		b.WriteString(patternsLabel + "\n\n")

		for i, pat := range p.patterns {
			checkbox := "[ ]"
			if pat.Enabled {
				checkbox = "[x]"
			}

			row := fmt.Sprintf("%s %s", checkbox, pat.Pattern)
			if i == p.selected {
				row = sdk.StyleSelected.Width(width - 6).Render(row)
			}
			b.WriteString(row + "\n")
		}
	}

	hint := sdk.StyleFaintItalic.Render("\nSpace toggle")

	return b.String() + hint
}

func (p *Plugin) renderConfirm(width, height int) string {
	previewLabel := sdk.StyleKey.Render("Preview (tfui.hcl):")
	preview := sdk.StyleFaint.Render(p.preview)

	return previewLabel + "\n\n" + preview
}

func (p *Plugin) renderDone(width, height int) string {
	successMsg := sdk.StyleSuccess.Render("tfui.hcl written successfully!")
	path := sdk.StyleFaint.Render(filepath.Join(p.dir, "tfui.hcl"))

	return successMsg + "\n" + path
}

func (p *Plugin) detect() tea.Cmd {
	dir := p.dir
	return func() tea.Msg {
		binary := detectBinary("")
		patterns := detectPatterns(dir)
		return DetectionCompleteMsg{
			Binary:   binary,
			Patterns: patterns,
		}
	}
}

func (p *Plugin) writeConfig() tea.Cmd {
	content := p.preview
	dir := p.dir
	return func() tea.Msg {
		path := filepath.Join(dir, config.HCLConfigFileName)
		err := os.WriteFile(path, []byte(content), 0644)
		return WriteCompleteMsg{Path: path, Err: err}
	}
}

func (p *Plugin) generateHCL() string {
	var enabled []string
	for _, pat := range p.patterns {
		if pat.Enabled {
			enabled = append(enabled, pat.Pattern)
		}
	}
	return buildHCL(p.binary, enabled)
}

// detectPatterns scans the directory for common terraform project layouts.
func detectPatterns(dir string) []DetectedPattern {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}

	candidates := []string{
		"modules/*",
		"envs/*",
		"infra/*",
		"services/*/terraform",
	}

	var patterns []DetectedPattern
	for _, candidate := range candidates {
		fullPattern := filepath.Join(absDir, candidate)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if config.HasTerraformFiles(match) {
				patterns = append(patterns, DetectedPattern{
					Pattern: candidate,
					Enabled: true,
				})
				break
			}
		}
	}

	// Check for root-level .tf files
	if config.HasTerraformFiles(absDir) {
		patterns = append(patterns, DetectedPattern{
			Pattern: ".",
			Enabled: true,
		})
	}

	return patterns
}

// GenerateConfig runs the detection logic non-interactively and returns HCL content.
func GenerateConfig(dir string) (string, error) {
	binary := detectBinary("")
	patterns := detectPatterns(dir)

	var enabled []string
	for _, pat := range patterns {
		if pat.Enabled {
			enabled = append(enabled, pat.Pattern)
		}
	}

	return buildHCL(binary, enabled), nil
}

func buildHCL(binary string, members []string) string {
	var b strings.Builder
	b.WriteString("# Generated by tfui init\n\n")
	fmt.Fprintf(&b, "terraform {\n  bin = %q\n}\n", binary)

	if len(members) > 0 {
		sort.Strings(members)
		b.WriteString("\nchdir {\n  members = [\n")
		for _, m := range members {
			fmt.Fprintf(&b, "    %q,\n", m)
		}
		b.WriteString("  ]\n}\n")
	}

	return b.String()
}
