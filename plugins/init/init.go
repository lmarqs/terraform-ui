package init

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	sdkframes "github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// initSubmitMsg triggers the init execution after form submission.
type initSubmitMsg struct{}

// InitResultMsg is sent when the init operation completes.
type InitResultMsg struct {
	Err      error
	Duration time.Duration
}

// Plugin implements the terraform init feature.
type Plugin struct {
	sdk.PluginBase
	timer ui.Timer
	stack *sdk.Stack
	lw    *sdkframes.LineWriter
	ch    <-chan string

	// Form state (preserved across runs for re-fill)
	upgrade        bool
	reconfigure    bool
	backend        bool
	backendConfigs []string
	extraArgs      string
	cancelFn       context.CancelFunc
}

// New creates a new init plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{
		PluginBase: sdk.NewPluginBase("init", "Init", "Initialize terraform working directory"),
		stack:      sdk.NewStack(),
		backend:    true,
	}
	p.Svc = svc
	return p
}

func (p *Plugin) Ready() bool       { return true }
func (p *Plugin) Stack() *sdk.Stack { return p.stack }

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Busy() bool {
	if top := p.stack.Peek(); top != nil {
		if rf, ok := top.(*resultFrame); ok {
			return rf.status == sdk.StatusLoading
		}
	}
	return false
}

func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.InitBase(deps)
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if !p.HandleContextChangedDefault(ev) {
		return nil
	}
	p.stack.Reset()
	return nil
}

func (p *Plugin) Activate() tea.Cmd {
	p.stack.Reset()
	p.stack.Push(p.buildForm())
	return nil
}

func (p *Plugin) ActivateWithArgs(args []string) tea.Cmd {
	p.resetState()
	p.parseArgs(args)
	p.stack.Reset()
	p.stack.Push(p.buildForm())
	return func() tea.Msg { return initSubmitMsg{} }
}

func (p *Plugin) resetState() {
	p.upgrade = false
	p.reconfigure = false
	p.backend = true
	p.backendConfigs = nil
	p.extraArgs = ""
}

func (p *Plugin) parseArgs(args []string) {
	for _, arg := range args {
		switch {
		case arg == "--upgrade":
			p.upgrade = true
		case arg == "--reconfigure":
			p.reconfigure = true
		case arg == "--backend=false":
			p.backend = false
		case arg == "--backend=true", arg == "--backend":
			p.backend = true
		case strings.HasPrefix(arg, "--backend-config="):
			p.backendConfigs = append(p.backendConfigs, strings.TrimPrefix(arg, "--backend-config="))
		}
	}
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg.(type) {
	case initSubmitMsg:
		p.stack.Reset()
		p.stack.Push(p.buildForm())
		lw, ch := sdkframes.NewLineWriter()
		p.lw = lw
		p.ch = ch
		sf := sdkframes.NewStreamFrame("terraform init", ch, p.Cancel)
		rf := newResultFrame(&p.timer, sf)
		p.stack.Push(rf)
		return p, p.submit(lw)

	case sdkframes.StreamLineMsg:
		if top := p.stack.Peek(); top != nil {
			_, cmd := top.Update(msg)
			return p, cmd
		}
		if p.ch != nil {
			return p, sdkframes.WaitForLine(p.ch)
		}

	case sdkframes.StreamDoneMsg:
		if top := p.stack.Peek(); top != nil {
			_, cmd := top.Update(msg)
			return p, cmd
		}

	case InitResultMsg, ui.TimerTickMsg:
		if top := p.stack.Peek(); top != nil {
			_, cmd := top.Update(msg)
			return p, cmd
		}
	}
	return p, nil
}

func (p *Plugin) View(width, height int) string {
	return p.stack.View(width, height)
}

func (p *Plugin) buildForm() *sdkframes.FormFrame {
	return sdkframes.NewFormFrame(sdkframes.FormOpts{
		Fields: []sdkframes.FormField{
			{
				Label:      "upgrade",
				Value:      func() string { return checkbox(p.upgrade) },
				Selectable: true,
				OnSelect:   func() tea.Cmd { p.upgrade = !p.upgrade; return nil },
			},
			{
				Label:      "reconfigure",
				Value:      func() string { return checkbox(p.reconfigure) },
				Selectable: true,
				OnSelect:   func() tea.Cmd { p.reconfigure = !p.reconfigure; return nil },
			},
			{
				Label:      "backend",
				Value:      func() string { return checkbox(p.backend) },
				Selectable: true,
				OnSelect:   func() tea.Cmd { p.backend = !p.backend; return nil },
			},
			{
				Label:      "extra args",
				Value:      func() string { return p.extraArgsDisplay() },
				Selectable: true,
				OnSelect:   p.editExtraArgs,
			},
			{
				Label:      "",
				Value:      func() string { return "Run terraform init" },
				Selectable: true,
				IsAction:   true,
				OnSelect:   p.submitFromForm,
			},
		},
	})
}

func (p *Plugin) editExtraArgs() tea.Cmd {
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputText("Extra args:", p.extraArgs, func(value string) tea.Cmd {
				p.extraArgs = value
				return nil
			}),
		}
	}
}

func (p *Plugin) submitFromForm() tea.Cmd {
	return func() tea.Msg { return initSubmitMsg{} }
}

// Cancel aborts any in-flight terraform operation.
func (p *Plugin) Cancel() {
	if p.cancelFn != nil {
		p.cancelFn()
		p.cancelFn = nil
	}
}

func (p *Plugin) submit(lw *sdkframes.LineWriter) tea.Cmd {
	p.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFn = cancel
	svc := p.Svc
	opts := sdk.InitOptions{
		Upgrade:       p.upgrade,
		Reconfigure:   p.reconfigure,
		BackendConfig: p.backendConfigs,
		Writer:        lw,
	}
	if !p.backend {
		opts.Backend = sdk.BackendDisabled
	}
	if p.extraArgs != "" {
		opts.ExtraArgs = strings.Fields(p.extraArgs)
	}

	ch := p.ch
	start := time.Now()
	return tea.Batch(
		func() tea.Msg {
			err := svc.Init(ctx, opts)
			lw.Close()
			return InitResultMsg{Err: err, Duration: time.Since(start)}
		},
		p.timer.Start(),
		sdkframes.WaitForLine(ch),
	)
}

func (p *Plugin) extraArgsDisplay() string {
	if p.extraArgs == "" {
		return "(none)"
	}
	return p.extraArgs
}

func checkbox(v bool) string {
	if v {
		return "[x]"
	}
	return "[ ]"
}

// SetJSONStdout is a temporary cmd-side setter used by the legacy
// Session.WithJSON path. Init does not vary its stdout content by JSON intent
// today, but accepts the setter for symmetry until Phase 3 migrates init to a
// typed Input.
func (p *Plugin) SetJSONStdout(_ bool) {}

// Stdout produces stdout content for standalone/CI mode.
func (p *Plugin) Stdout() ([]byte, error) {
	return []byte("Initialized successfully.\n"), nil
}
