package output

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Status represents the current state of the output plugin.
type Status int

const (
	StatusIdle Status = iota
	StatusLoading
	StatusDone
	StatusError
)

// OutputResultMsg is sent when the output fetch completes.
type OutputResultMsg struct {
	Outputs map[string]sdk.OutputValue
	Err     error
}

// Plugin implements the terraform outputs viewer.
type Plugin struct {
	svc           sdk.Service
	log           *slog.Logger
	session       *sdk.Session
	stack         *sdk.Stack
	status        Status
	outputs       []sdk.OutputValue
	filtered      []sdk.OutputValue
	filter        string
	filtering     bool
	errMsg        string
	selected      int
	scopedContext string
}

// New creates a new output plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		svc: svc,
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	return p
}

func (p *Plugin) ID() string          { return "output" }
func (p *Plugin) Name() string        { return "Outputs" }
func (p *Plugin) Description() string { return "View terraform outputs" }
func (p *Plugin) KeyBinding() string  { return "o" }
func (p *Plugin) Ready() bool         { return p.status == StatusDone }
func (p *Plugin) Status() Status      { return p.status }
func (p *Plugin) Selected() int       { return p.selected }
func (p *Plugin) Filter() string      { return p.filter }
func (p *Plugin) Filtering() bool     { return p.filtering }
func (p *Plugin) OutputCount() int    { return len(p.filtered) }
func (p *Plugin) TotalCount() int     { return len(p.outputs) }
func (p *Plugin) Stack() *sdk.Stack   { return p.stack }

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
	p.outputs = nil
	p.filtered = nil
	p.filter = ""
	p.filtering = false
	p.errMsg = ""
	p.selected = 0
	return nil
}

// Activate triggers output loading when the user enters the plugin.
func (p *Plugin) Activate() tea.Cmd {
	// Check if the active context changed since last activation
	if p.session != nil {
		currentContext, _ := sdk.GetTyped[string](p.session, sdk.SessionKeyActiveContextAbs)
		if currentContext != p.scopedContext {
			// Context changed — reset state
			p.status = StatusIdle
			p.outputs = nil
			p.filtered = nil
			p.filter = ""
			p.filtering = false
			p.errMsg = ""
			p.selected = 0
			p.scopedContext = currentContext
			if currentContext != "" {
				p.svc = p.svc.WithDir(currentContext)
			}
		}
	}

	if p.status == StatusIdle || p.status == StatusError {
		// Check if there's an active context to scope to
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
		return p.loadOutputs()
	}
	return nil
}

// Refresh reloads the outputs.
func (p *Plugin) Refresh() tea.Cmd {
	p.status = StatusLoading
	p.outputs = nil
	p.filtered = nil
	p.filter = ""
	p.filtering = false
	p.errMsg = ""
	p.selected = 0
	return p.loadOutputs()
}

func (p *Plugin) loadOutputs() tea.Cmd {
	svc := p.svc
	return func() tea.Msg {
		outputs, err := svc.Output(context.Background())
		return OutputResultMsg{Outputs: outputs, Err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case OutputResultMsg:
		if msg.Err != nil {
			p.status = StatusError
			p.errMsg = msg.Err.Error()
			p.log.Debug("output.load.error", "error", msg.Err.Error())
		} else {
			p.status = StatusDone
			p.outputs = sortedOutputs(msg.Outputs)
			p.filtered = p.outputs
			p.log.Debug("output.load.complete", "outputs", len(p.outputs))
		}
		return p, nil

	}
	return p, nil
}

func sortedOutputs(m map[string]sdk.OutputValue) []sdk.OutputValue {
	if m == nil {
		return nil
	}
	result := make([]sdk.OutputValue, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// MoveUp moves selection up.
func (p *Plugin) MoveUp() {
	if p.selected > 0 {
		p.selected--
	}
}

// MoveDown moves selection down.
func (p *Plugin) MoveDown() {
	if p.selected < len(p.filtered)-1 {
		p.selected++
	}
}

// MoveToStart moves selection to the first item.
func (p *Plugin) MoveToStart() {
	p.selected = 0
}

// MoveToEnd moves selection to the last item.
func (p *Plugin) MoveToEnd() {
	if len(p.filtered) > 0 {
		p.selected = len(p.filtered) - 1
	}
}

// SetFilter filters outputs using fzf's FuzzyMatchV2 algorithm on the output name.
func (p *Plugin) SetFilter(filter string) {
	p.filter = filter
	p.selected = 0
	if filter == "" {
		p.filtered = p.outputs
		p.log.Debug("output.filter", "filter", "", "results", len(p.outputs))
		return
	}
	terms := strings.Fields(strings.ToLower(filter))
	type scored struct {
		output sdk.OutputValue
		score  int
	}
	var results []scored
	slab := util.MakeSlab(100*1024, 2048)
	for _, o := range p.outputs {
		input := util.RunesToChars([]rune(strings.ToLower(o.Name)))
		totalScore := 0
		matched := true
		for _, term := range terms {
			res, _ := algo.FuzzyMatchV2(false, true, true, &input, []rune(term), false, slab)
			if res.Score <= 0 {
				matched = false
				break
			}
			totalScore += res.Score
		}
		if matched {
			results = append(results, scored{o, totalScore})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	p.filtered = make([]sdk.OutputValue, len(results))
	for i, r := range results {
		p.filtered[i] = r.output
	}
	p.log.Debug("output.filter", "filter", filter, "results", len(p.filtered))
}

// AppendFilter adds a character to the filter.
func (p *Plugin) AppendFilter(ch string) {
	p.SetFilter(p.filter + ch)
}

// BackspaceFilter removes the last character from the filter.
func (p *Plugin) BackspaceFilter() {
	if len(p.filter) > 0 {
		p.SetFilter(p.filter[:len(p.filter)-1])
	}
}

// FormatValue returns the display string for an output value, redacting sensitive values.
func FormatValue(o sdk.OutputValue) string {
	if o.Sensitive {
		return "(sensitive)"
	}
	return fmt.Sprintf("%v", o.Value)
}

// View renders the output plugin's UI.
func (p *Plugin) View(width, height int) string {
	switch p.status {
	case StatusIdle:
		title := sdk.StyleTitle.Render("Outputs")
		placeholder := sdk.StyleFaintItalic.Render("Loading outputs...")
		return sdk.StylePadded.Render(title + "\n\n" + placeholder)

	case StatusLoading:
		title := sdk.StyleTitle.Render("Outputs")
		loading := sdk.StyleFaintItalic.Render("Loading terraform outputs...")
		return sdk.StylePadded.Render(title + "\n\n" + loading)

	case StatusError:
		title := sdk.StyleTitle.Render("Outputs")
		errText := sdk.StyleError.Render("Error: " + p.errMsg)
		return sdk.StylePadded.Render(title + "\n\n" + errText)

	case StatusDone:
		return p.renderOutputs(width, height)

	default:
		return ""
	}
}

func (p *Plugin) renderOutputs(width, height int) string {
	title := sdk.StyleTitle.Render("Outputs")

	filterLine := ""
	if p.filtering {
		filterLine = sdk.StyleKey.Render("/ ") + p.filter + "█\n\n"
	} else if p.filter != "" {
		filterLine = sdk.StyleKey.Render("filter: ") + p.filter + "\n\n"
	}

	if len(p.filtered) == 0 {
		noOutputs := sdk.StyleFaintItalic.Render("No outputs found.")
		return sdk.StylePadded.Render(title + "\n\n" + filterLine + noOutputs)
	}

	var b strings.Builder

	// Calculate visible area
	maxVisible := height - 7
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if p.selected >= maxVisible {
		startIdx = p.selected - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(p.filtered) {
		endIdx = len(p.filtered)
	}

	contentWidth := width - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	for i := startIdx; i < endIdx; i++ {
		o := p.filtered[i]
		row := p.renderOutputRow(o)
		if i == p.selected {
			row = sdk.StyleSelected.Width(contentWidth).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d outputs", len(p.filtered)))
	if len(p.filtered) != len(p.outputs) {
		count = sdk.StyleFaint.Render(fmt.Sprintf("%d/%d outputs", len(p.filtered), len(p.outputs)))
	}

	content := title + "\n\n" + filterLine + b.String() + "\n" + count
	return sdk.StylePadded.Render(content)
}

func (p *Plugin) renderOutputRow(o sdk.OutputValue) string {
	name := o.Name
	typeStr := sdk.StyleFaint.Render(o.Type)
	value := FormatValue(o)
	if o.Sensitive {
		value = sdk.StyleFaintItalic.Render(value)
	}
	return fmt.Sprintf(" %s  %s  = %s", name, typeStr, value)
}
