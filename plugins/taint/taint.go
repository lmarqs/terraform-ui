package taint

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// TaintRequestMsg requests navigation to the taint plugin with target addresses.
type TaintRequestMsg struct {
	Addresses []string
}

// taintStartMsg triggers execution after confirmation.
type taintStartMsg struct{}

// taintResultMsg is sent when the taint operation completes.
type taintResultMsg struct {
	Tainted []string
	Err     error
}

// Plugin implements the standalone taint verb.
type Plugin struct {
	sdk.PluginBase
	timer     ui.Timer
	status    sdk.Status
	input     Input
	addresses []string
	tainted   []string
	errMsg    string
	cancelFn  context.CancelFunc
}

// New creates a new taint plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{PluginBase: sdk.NewPluginBase("taint", "Taint", "Mark resources for recreation on next apply")}
	p.Svc = svc
	return p
}

func (p *Plugin) Ready() bool { return p.status == sdk.StatusDone }
func (p *Plugin) Busy() bool  { return p.status == sdk.StatusLoading }

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.InitBase(deps)
	return nil
}

// Activate stores the typed input and returns the initial command.
func (p *Plugin) Activate(input Input) tea.Cmd {
	if p.status == sdk.StatusLoading {
		return nil
	}
	p.input = input
	p.addresses = input.Addrs
	p.status = sdk.StatusIdle
	p.errMsg = ""
	p.tainted = nil
	return p.confirmTaint()
}

func (p *Plugin) confirmTaint() tea.Cmd {
	if len(p.addresses) == 0 {
		return func() tea.Msg { return sdk.DeactivateMsg{} }
	}

	prompt := fmt.Sprintf("Taint %s? (will recreate on next apply)", p.addresses[0])
	if len(p.addresses) > 1 {
		prompt = fmt.Sprintf("Taint %d resources? (will recreate on next apply)\n  %s", len(p.addresses), strings.Join(p.addresses, "\n  "))
	}

	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(prompt, func() tea.Cmd {
				return func() tea.Msg { return taintStartMsg{} }
			}),
		}
	}
}

// Cancel aborts any in-flight terraform operation.
func (p *Plugin) Cancel() {
	if p.cancelFn != nil {
		p.cancelFn()
		p.cancelFn = nil
	}
}

func (p *Plugin) executeTaint() tea.Cmd {
	p.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFn = cancel
	p.status = sdk.StatusLoading
	svc := p.Svc
	log := p.Log
	addresses := p.addresses
	return tea.Batch(func() tea.Msg {
		var tainted []string
		for _, addr := range addresses {
			if err := svc.Taint(ctx, addr); err != nil {
				log.Debug("taint.error", "address", addr, "error", err.Error())
				return taintResultMsg{Tainted: tainted, Err: fmt.Errorf("%s: %w", addr, err)}
			}
			tainted = append(tainted, addr)
		}
		log.Debug("taint.success", "count", len(tainted))
		return taintResultMsg{Tainted: tainted}
	}, p.timer.Start())
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return p, p.timer.Tick()

	case taintStartMsg:
		return p, p.executeTaint()

	case taintResultMsg:
		p.timer.Stop()
		p.tainted = msg.Tainted
		if msg.Err != nil {
			p.status = sdk.StatusError
			p.errMsg = msg.Err.Error()
		} else {
			p.status = sdk.StatusDone
			return p, func() tea.Msg { return sdk.PlanInvalidatedEvent{} }
		}
		return p, nil

	case tea.KeyMsg:
		return p, p.handleKey(msg)
	}
	return p, nil
}

func (p *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch p.status {
	case sdk.StatusDone:
		switch msg.String() {
		case "p":
			return func() tea.Msg { return sdk.NavigateMsg{PluginID: "plan"} }
		case "esc":
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	case sdk.StatusError:
		switch msg.String() {
		case "ctrl+r":
			return p.executeTaint()
		case "esc":
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	case sdk.StatusIdle:
		switch msg.String() {
		case "esc":
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	}
	return nil
}

func (p *Plugin) View(_, _ int) string {
	switch p.status {
	case sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Waiting for confirmation...")
	case sdk.StatusLoading:
		label := p.addresses[0]
		if len(p.addresses) > 1 {
			label = fmt.Sprintf("%d resources", len(p.addresses))
		}
		return sdk.StyleFaintItalic.Render(fmt.Sprintf("Tainting %s... %s", label, p.timer.FormatElapsed()))
	case sdk.StatusDone:
		if len(p.tainted) == 1 {
			return sdk.StyleSuccess.Render(fmt.Sprintf("✓ Tainted %s", p.tainted[0])) +
				"\n" + sdk.StyleFaint.Render("Duration: "+p.timer.FormatElapsed())
		}
		return sdk.StyleSuccess.Render(fmt.Sprintf("✓ Tainted %d resources", len(p.tainted))) +
			"\n" + sdk.StyleFaint.Render("Duration: "+p.timer.FormatElapsed())
	case sdk.StatusError:
		return sdk.StyleError.Render("✗ Taint failed: " + p.errMsg)
	default:
		return ""
	}
}

func (p *Plugin) Hints() []sdk.KeyHint {
	switch p.status {
	case sdk.StatusDone:
		return []sdk.KeyHint{
			{Key: "p", Description: "plan"},
			sdk.HintCancel,
		}
	case sdk.StatusError:
		return (sdk.HintSetRetry | sdk.HintSetBack).Hints()
	default:
		return (sdk.HintSetBack).Hints()
	}
}

// HandleContextChanged implements sdk.ContextChangedHandler.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if !p.HandleContextChangedDefault(ev) {
		return nil
	}
	p.status = sdk.StatusIdle
	p.addresses = nil
	p.errMsg = ""
	return nil
}
