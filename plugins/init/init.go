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
	svc    sdk.Service
	status sdk.Status
	timer  ui.Timer
	stack  *sdk.Stack
	errMsg string

	// Form state (preserved across runs for re-fill)
	upgrade     bool
	reconfigure bool
	backend     bool
	extraArgs   string
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
func (p *Plugin) Ready() bool         { return p.status == sdk.StatusDone }
func (p *Plugin) Stack() *sdk.Stack   { return p.stack }

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	return nil
}

func (p *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	p.svc = p.svc.WithDir(evt.AbsPath)
	p.reset()
	return nil
}

func (p *Plugin) Activate() tea.Cmd {
	if p.status == sdk.StatusLoading {
		return nil
	}
	p.status = sdk.StatusIdle
	p.errMsg = ""
	p.stack.Clear()
	p.stack.Push(p.buildForm())
	return nil
}

func (p *Plugin) Hints() []sdk.KeyHint {
	if top := p.stack.Peek(); top != nil {
		return top.Hints()
	}
	switch p.status {
	case sdk.StatusLoading:
		return (sdk.HintSetCancel).Hints()
	case sdk.StatusDone:
		return []sdk.KeyHint{
			{Key: "Enter", Description: "re-run"},
			sdk.HintRefresh,
			sdk.HintBack,
		}
	case sdk.StatusError:
		return []sdk.KeyHint{
			{Key: "Enter", Description: "re-run"},
			sdk.HintRetry,
			sdk.HintBack,
		}
	default:
		return (sdk.HintSetBack).Hints()
	}
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return p, p.timer.Tick()

	case initSubmitMsg:
		p.stack.Reset()
		return p, p.submit()

	case InitResultMsg:
		p.timer.Stop()
		if msg.Err != nil {
			p.status = sdk.StatusError
			p.errMsg = msg.Err.Error()
			return p, nil
		}
		p.status = sdk.StatusDone
		return p, func() tea.Msg { return sdk.PlanInvalidatedEvent{} }

	case tea.KeyMsg:
		if top := p.stack.Peek(); top != nil {
			cmd := p.stack.Update(msg)
			return p, cmd
		}
		return p, p.handleKey(msg)
	}

	return p, nil
}

func (p *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch p.status {
	case sdk.StatusDone:
		switch msg.String() {
		case "enter":
			p.stack.Push(p.buildForm())
			return nil
		case "ctrl+r":
			return p.submit()
		}
	case sdk.StatusError:
		switch msg.String() {
		case "enter":
			p.stack.Push(p.buildForm())
			return nil
		case "ctrl+r":
			return p.submit()
		}
	}
	return nil
}

func (p *Plugin) View(width, height int) string {
	if top := p.stack.Peek(); top != nil {
		return top.View(width, height)
	}

	switch p.status {
	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render("Running terraform init... " + p.timer.FormatElapsed())
	case sdk.StatusDone:
		return sdk.StyleSuccess.Render("Terraform initialized successfully.") + "\n" +
			sdk.StyleFaint.Render("Duration: "+p.timer.FormatElapsed())
	case sdk.StatusError:
		return sdk.StyleError.Render("Init failed: " + p.errMsg)
	default:
		return ""
	}
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

func (p *Plugin) submit() tea.Cmd {
	p.status = sdk.StatusLoading
	p.errMsg = ""

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
		err := svc.Init(context.Background(), opts)
		return InitResultMsg{Err: err, Duration: time.Since(start)}
	}, p.timer.Start())
}

func (p *Plugin) reset() {
	p.status = sdk.StatusIdle
	p.errMsg = ""
	p.stack.Clear()
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
