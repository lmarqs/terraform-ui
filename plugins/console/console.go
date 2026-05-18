package console

import (
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const StatusEvaluating = sdk.Status(10)

// replEntry holds a single expression and its result.
type replEntry struct {
	Expr   string
	Result string
	Error  string
}

// ReplResultMsg is sent when an expression evaluation completes.
type ReplResultMsg struct {
	Expr   string
	Output string
	Err    error
}

// Plugin implements the terraform console REPL feature.
type Plugin struct {
	svc           sdk.Service
	log           *slog.Logger
	status        sdk.Status
	history       []replEntry
	input         string
	historyIdx    int // -1 means current input, 0+ means recalling from history
	scrollY       int
	dir           string
	binaryPath    string
	errMsg        string
	scopedContext string
	pastInputs    []string // previous expressions for up/down recall
	savedInput    string   // saved current input when browsing history
}

// New creates a new REPL plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc:        svc,
		log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		historyIdx: -1,
	}
}

func (p *Plugin) ID() string          { return "console" }
func (p *Plugin) Name() string        { return "Console" }
func (p *Plugin) Description() string { return "Terraform console (REPL)" }
func (p *Plugin) Ready() bool         { return p.status == sdk.StatusDone }

// CapturesKeys implements sdk.KeyCapturer.
func (p *Plugin) CapturesKeys() bool {
	return p.status == sdk.StatusDone || p.status == StatusEvaluating
}

// Hints returns context-sensitive key hints for the status bar.
func (p *Plugin) Hints() []sdk.KeyHint {
	switch p.status {
	case sdk.StatusDone, StatusEvaluating:
		return []sdk.KeyHint{
			{Key: "Enter", Description: "evaluate"},
			{Key: "↑↓", Description: "history"},
			{Key: "^U", Description: "clear"},
			{Key: "esc", Description: "back"},
		}
	case sdk.StatusError:
		return (sdk.HintSetQuit).Hints()
	default:
		return (sdk.HintSetQuit).Hints()
	}
}

// Configure applies plugin-specific options from config.
func (p *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init initializes the plugin with shared context.
func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	p.log = ctx.Logger
	p.dir = ctx.WorkingDir
	p.status = sdk.StatusIdle
	p.reset()
	return nil
}

// HandleChdirChanged implements sdk.ChdirHandler.
func (p *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	p.svc = p.svc.WithDir(evt.AbsPath)
	p.scopedContext = evt.AbsPath
	p.dir = evt.AbsPath
	p.reset()
	return nil
}

// reset clears repl-specific state.
func (p *Plugin) reset() {
	p.history = nil
	p.input = ""
	p.historyIdx = -1
	p.scrollY = 0
	p.errMsg = ""
	p.pastInputs = nil
	p.savedInput = ""
}

// Activate is called when the user navigates to this plugin.
func (p *Plugin) Activate() tea.Cmd {
	// Detect terraform binary if not already set
	if p.binaryPath == "" {
		p.binaryPath = detectBinary()
	}

	p.status = sdk.StatusDone
	return nil
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ReplResultMsg:
		p.status = sdk.StatusDone
		entry := replEntry{Expr: msg.Expr}
		if msg.Err != nil {
			entry.Error = msg.Err.Error()
		} else {
			entry.Result = strings.TrimSpace(msg.Output)
		}
		p.history = append(p.history, entry)
		// Auto-scroll to bottom
		p.scrollY = len(p.history)
		return p, nil

	case tea.KeyMsg:
		cmd := p.handleKey(msg)
		return p, cmd
	}
	return p, nil
}

func (p *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		if p.status == StatusEvaluating {
			return nil
		}
		return func() tea.Msg { return sdk.DeactivateMsg{} }

	case "q":
		if p.input == "" && p.status != StatusEvaluating {
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
		p.input += "q"
		p.historyIdx = -1
		return nil

	case "ctrl+u":
		p.input = ""
		p.historyIdx = -1
		return nil

	case "enter":
		if p.status == StatusEvaluating {
			return nil
		}
		expr := strings.TrimSpace(p.input)
		if expr == "" {
			return nil
		}
		p.pastInputs = append(p.pastInputs, expr)
		p.input = ""
		p.historyIdx = -1
		p.savedInput = ""
		p.status = StatusEvaluating
		return p.evaluate(expr)

	case "up":
		if len(p.pastInputs) == 0 {
			return nil
		}
		if p.historyIdx == -1 {
			// Save current input and start browsing from the end
			p.savedInput = p.input
			p.historyIdx = len(p.pastInputs) - 1
		} else if p.historyIdx > 0 {
			p.historyIdx--
		}
		p.input = p.pastInputs[p.historyIdx]
		return nil

	case "down":
		if p.historyIdx == -1 {
			return nil
		}
		if p.historyIdx < len(p.pastInputs)-1 {
			p.historyIdx++
			p.input = p.pastInputs[p.historyIdx]
		} else {
			// Back to current input
			p.historyIdx = -1
			p.input = p.savedInput
		}
		return nil

	case "backspace", "ctrl+h":
		if len(p.input) > 0 {
			p.input = p.input[:len(p.input)-1]
		}
		return nil

	default:
		if len(msg.String()) == 1 && msg.String() >= " " {
			p.input += msg.String()
			p.historyIdx = -1
		}
		return nil
	}
}

// evaluate runs the expression through terraform console.
func (p *Plugin) evaluate(expr string) tea.Cmd {
	dir := p.dir
	binary := p.binaryPath
	return func() tea.Msg {
		cmd := exec.Command(binary, "console")
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader(expr + "\n")
		out, err := cmd.Output()
		return ReplResultMsg{Expr: expr, Output: string(out), Err: err}
	}
}

// View renders the REPL plugin UI.
func (p *Plugin) View(width, height int) string {
	switch p.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Activating...")

	case sdk.StatusError:
		return sdk.StyleError.Render("Error: " + p.errMsg)

	case sdk.StatusDone, StatusEvaluating:
		return p.renderREPL(width, height)

	default:
		return ""
	}
}

func (p *Plugin) renderREPL(width, height int) string {
	// Reserve lines for: input(1) + blank(1) + hint(1)
	headerLines := 3
	maxHistoryLines := height - headerLines
	if maxHistoryLines < 3 {
		maxHistoryLines = 3
	}

	// Build history lines
	var histLines []string
	for _, entry := range p.history {
		prompt := sdk.StyleKey.Render("> ") + entry.Expr
		histLines = append(histLines, prompt)
		if entry.Error != "" {
			histLines = append(histLines, sdk.StyleError.Render("Error: "+entry.Error))
		} else if entry.Result != "" {
			resultLines := strings.Split(entry.Result, "\n")
			histLines = append(histLines, resultLines...)
		}
		histLines = append(histLines, "")
	}

	// Handle scrolling
	visible := histLines
	if len(histLines) > maxHistoryLines {
		start := len(histLines) - maxHistoryLines
		visible = histLines[start:]
	}

	var b strings.Builder
	for _, line := range visible {
		b.WriteString(line + "\n")
	}

	// Input line
	inputPrefix := sdk.StyleKey.Render("> ")
	cursor := "█" // block cursor
	inputLine := inputPrefix + p.input + cursor

	if p.status == StatusEvaluating {
		inputLine = inputPrefix + sdk.StyleFaintItalic.Render("evaluating...")
	}

	return b.String() + inputLine
}

// detectBinary finds the terraform/tofu binary on PATH.
func detectBinary() string {
	if _, err := exec.LookPath("tofu"); err == nil {
		return "tofu"
	}
	return "terraform"
}

// Exported getters for testing.

func (p *Plugin) Status() sdk.Status   { return p.status }
func (p *Plugin) Input() string        { return p.input }
func (p *Plugin) History() []replEntry { return p.history }
func (p *Plugin) HistoryIdx() int      { return p.historyIdx }
func (p *Plugin) PastInputs() []string { return p.pastInputs }
func (p *Plugin) Dir() string          { return p.dir }
func (p *Plugin) BinaryPath() string   { return p.binaryPath }
func (p *Plugin) ErrMsg() string       { return p.errMsg }

// SetBinaryPath allows tests to override the binary path.
func (p *Plugin) SetBinaryPath(path string) { p.binaryPath = path }

// HistoryEntry returns the entry at the given index for testing.
func (p *Plugin) HistoryEntry(i int) (expr, result, errStr string) {
	if i < 0 || i >= len(p.history) {
		return "", "", ""
	}
	e := p.history[i]
	return e.Expr, e.Result, e.Error
}

// HistoryLen returns the number of history entries.
func (p *Plugin) HistoryLen() int {
	return len(p.history)
}

// Evaluate exposes the evaluate method for testing (returns the tea.Cmd).
func (p *Plugin) Evaluate(expr string) tea.Cmd {
	return p.evaluate(expr)
}

// FormatHistoryEntry formats a single history entry for display (exported for tests).
func FormatHistoryEntry(entry replEntry) string {
	prompt := fmt.Sprintf("> %s", entry.Expr)
	if entry.Error != "" {
		return prompt + "\n" + "Error: " + entry.Error
	}
	if entry.Result != "" {
		return prompt + "\n" + entry.Result
	}
	return prompt
}
