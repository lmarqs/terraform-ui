package init

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
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
	svc   sdk.Service
	timer ui.Timer
	stack *sdk.Stack

	// Form state (preserved across runs for re-fill)
	upgrade     bool
	reconfigure bool
	backend     bool
	extraArgs   string
	cancelFn    context.CancelFunc
}

// New creates a new init plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc:     svc,
		stack:   sdk.NewStack(),
		backend: true,
	}
}

func (p *Plugin) ID() string          { return "init" }
func (p *Plugin) Name() string        { return "Init" }
func (p *Plugin) Description() string { return "Initialize terraform working directory" }
func (p *Plugin) Ready() bool         { return true }
func (p *Plugin) Stack() *sdk.Stack   { return p.stack }

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Busy() bool {
	if top := p.stack.Peek(); top != nil {
		if rf, ok := top.(*resultFrame); ok {
			return rf.status == sdk.StatusLoading
		}
	}
	return false
}

func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	return nil
}

func (p *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	p.svc = p.svc.WithDir(evt.AbsPath)
	p.stack.Reset()
	return nil
}

func (p *Plugin) Activate() tea.Cmd {
	p.stack.Reset()
	p.stack.Push(p.buildForm())
	return nil
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg.(type) {
	case initSubmitMsg:
		p.stack.Reset()
		p.stack.Push(p.buildForm())
		rf := newResultFrame(&p.timer)
		p.stack.Push(rf)
		return p, p.submit()

	case InitResultMsg, ui.TimerTickMsg:
		if top := p.stack.Peek(); top != nil {
			result, cmd := top.Update(msg)
			if result == nil {
				p.stack.Pop()
			}
			return p, cmd
		}
	}
	return p, nil
}

func (p *Plugin) View(width, height int) string {
	return p.stack.View(width, height)
}

func (p *Plugin) buildForm() *frames.FormFrame {
	return frames.NewFormFrame(frames.FormOpts{
		Fields: []frames.FormField{
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

func (p *Plugin) submit() tea.Cmd {
	p.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFn = cancel
	svc := p.svc
	opts := sdk.InitOptions{
		Upgrade:     p.upgrade,
		Reconfigure: p.reconfigure,
	}
	if !p.backend {
		f := false
		opts.Backend = &f
	}
	if p.extraArgs != "" {
		opts.ExtraArgs = strings.Fields(p.extraArgs)
	}

	start := time.Now()
	return tea.Batch(func() tea.Msg {
		err := svc.Init(ctx, opts)
		return InitResultMsg{Err: err, Duration: time.Since(start)}
	}, p.timer.Start())
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
