package output

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
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
	stack         *sdk.Stack
	fuzzy         *ui.FuzzyFilter[sdk.OutputValue]
	timer         ui.Timer
	status        sdk.Status
	outputs       []sdk.OutputValue
	filtered      []sdk.OutputValue
	filter        string
	filtering     bool
	errMsg        string
	selected      int
	cancelFn      context.CancelFunc
}

// New creates a new output plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		svc:   svc,
		log:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		fuzzy: ui.NewFuzzyFilter(func(o sdk.OutputValue) string { return o.Name }),
	}
	p.stack = sdk.NewStack()
	p.stack.Push(&listFrame{plugin: p})
	return p
}

func (p *Plugin) ID() string          { return "output" }
func (p *Plugin) Name() string        { return "Outputs" }
func (p *Plugin) Description() string { return "View terraform outputs" }
func (p *Plugin) Ready() bool         { return p.status == sdk.StatusDone }
func (p *Plugin) Status() sdk.Status  { return p.status }
func (p *Plugin) Selected() int       { return p.selected }
func (p *Plugin) Filter() string      { return p.filter }
func (p *Plugin) Filtering() bool     { return p.filtering }
func (p *Plugin) OutputCount() int    { return len(p.filtered) }
func (p *Plugin) TotalCount() int     { return len(p.outputs) }
func (p *Plugin) Count() (int, int)   { return len(p.filtered), len(p.outputs) }
func (p *Plugin) CursorPosition() (int, int) {
	if p.status != sdk.StatusDone || len(p.filtered) == 0 {
		return 0, 0
	}
	return p.selected + 1, len(p.filtered)
}
func (p *Plugin) Stack() *sdk.Stack { return p.stack }

// Configure applies plugin-specific options from config.
func (p *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// Init wires the plugin to its shared dependencies.
func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.svc = deps.Service
	p.log = deps.Logger
	p.reset()
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if ev.Next == nil {
		return nil
	}
	if ev.Next.Service != nil {
		p.svc = ev.Next.Service
	}
	p.reset()
	return nil
}

// reset clears all plugin state to initial values.
func (p *Plugin) reset() {
	p.status = sdk.StatusIdle
	p.outputs = nil
	p.filtered = nil
	p.filter = ""
	p.filtering = false
	p.errMsg = ""
	p.selected = 0
	p.fuzzy.SetItems(nil)
}

// Activate triggers output loading when the user enters the plugin.
func (p *Plugin) Activate() tea.Cmd {
	if p.status == sdk.StatusIdle || p.status == sdk.StatusError {
		p.status = sdk.StatusLoading
		return tea.Batch(p.loadOutputs(), p.timer.Start())
	}
	return nil
}

// Refresh reloads the outputs.
func (p *Plugin) Refresh() tea.Cmd {
	p.reset()
	p.status = sdk.StatusLoading
	return tea.Batch(p.loadOutputs(), p.timer.Start())
}

// Cancel aborts any in-flight terraform operation.
func (p *Plugin) Cancel() {
	if p.cancelFn != nil {
		p.cancelFn()
		p.cancelFn = nil
	}
}

func (p *Plugin) loadOutputs() tea.Cmd {
	p.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFn = cancel
	svc := p.svc
	return func() tea.Msg {
		outputs, err := svc.Output(ctx)
		return OutputResultMsg{Outputs: outputs, Err: err}
	}
}

// Update processes messages and returns the updated plugin.
func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return p, p.timer.Tick()

	case OutputResultMsg:
		p.timer.Stop()
		if msg.Err != nil {
			p.status = sdk.StatusError
			p.errMsg = msg.Err.Error()
			p.log.Debug("output.load.error", "error", msg.Err.Error())
		} else {
			p.status = sdk.StatusDone
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
	p.fuzzy.SetItems(p.outputs)
	p.fuzzy.SetQuery(filter)
	p.filtered = p.fuzzy.Results()
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
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Loading outputs...")

	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Loading terraform outputs... " + p.timer.FormatElapsed())

	case sdk.StatusError:
		return sdk.StyleError.Render("Error: " + p.errMsg)

	case sdk.StatusDone:
		return p.renderOutputs(width, height)

	default:
		return ""
	}
}

func (p *Plugin) renderOutputs(width, height int) string {
	filterLine := ""
	if p.filtering {
		filterLine = sdk.StyleKey.Render("/ ") + p.filter + "█\n\n"
	} else if p.filter != "" {
		filterLine = sdk.StyleKey.Render("ᗊ: ") + p.filter + "\n\n"
	}

	if len(p.filtered) == 0 {
		return filterLine + sdk.StyleFaintItalic.Render("No outputs found.")
	}

	var b strings.Builder

	// Calculate visible area
	maxVisible := height - 5
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

	for i := startIdx; i < endIdx; i++ {
		o := p.filtered[i]
		row := p.renderOutputRow(o)
		if i == p.selected {
			row = sdk.StyleSelected.Width(width).Render(row)
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}

	count := sdk.StyleFaint.Render(fmt.Sprintf("%d outputs", len(p.filtered)))
	if len(p.filtered) != len(p.outputs) {
		count = sdk.StyleFaint.Render(fmt.Sprintf("%d/%d outputs", len(p.filtered), len(p.outputs)))
	}

	return filterLine + b.String() + "\n" + count
}

// Output produces stdout content for standalone/CI mode.
func (p *Plugin) Output(jsonOutput bool) ([]byte, error) {
	if jsonOutput {
		outputMap := make(map[string]interface{}, len(p.outputs))
		for _, o := range p.outputs {
			entry := map[string]interface{}{
				"value":     o.Value,
				"type":      o.Type,
				"sensitive": o.Sensitive,
			}
			outputMap[o.Name] = entry
		}
		return sdk.MarshalJSON(outputMap), nil
	}

	var b strings.Builder
	sorted := make([]sdk.OutputValue, len(p.outputs))
	copy(sorted, p.outputs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, o := range sorted {
		val := "(sensitive)"
		if !o.Sensitive {
			val = fmt.Sprintf("%v", o.Value)
		}
		fmt.Fprintf(&b, "%s = %s\n", o.Name, val)
	}
	return []byte(b.String()), nil
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
