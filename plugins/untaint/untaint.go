package untaint

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// UntaintRequestMsg requests navigation to the untaint plugin with target addresses.
type UntaintRequestMsg struct {
	Addresses []string
}

// untaintStartMsg triggers execution after confirmation.
type untaintStartMsg struct{}

// untaintResultMsg is sent when the untaint operation completes.
type untaintResultMsg struct {
	Untainted []string
	Err       error
}

// Plugin implements the standalone untaint verb.
type Plugin struct {
	svc       sdk.Service
	log       *slog.Logger
	timer     ui.Timer
	status    sdk.Status
	addresses []string
	untainted []string
	errMsg    string
}

// New creates a new untaint plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc: svc,
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func (p *Plugin) ID() string          { return "untaint" }
func (p *Plugin) Name() string        { return "Untaint" }
func (p *Plugin) Description() string { return "Remove taint mark from resources" }
func (p *Plugin) Ready() bool         { return p.status == sdk.StatusDone }
func (p *Plugin) Busy() bool          { return p.status == sdk.StatusLoading }

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	p.log = ctx.Logger
	return nil
}

// SetTargets configures the addresses to untaint.
func (p *Plugin) SetTargets(addresses []string) {
	p.addresses = addresses
}

func (p *Plugin) Activate() tea.Cmd {
	if p.status == sdk.StatusLoading {
		return nil
	}
	p.status = sdk.StatusIdle
	p.errMsg = ""
	p.untainted = nil
	return p.confirmUntaint()
}

func (p *Plugin) confirmUntaint() tea.Cmd {
	if len(p.addresses) == 0 {
		return func() tea.Msg { return sdk.DeactivateMsg{} }
	}

	prompt := fmt.Sprintf("Untaint %s?", p.addresses[0])
	if len(p.addresses) > 1 {
		prompt = fmt.Sprintf("Untaint %d resources?\n  %s", len(p.addresses), strings.Join(p.addresses, "\n  "))
	}

	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(prompt, func() tea.Cmd {
				return func() tea.Msg { return untaintStartMsg{} }
			}),
		}
	}
}

func (p *Plugin) executeUntaint() tea.Cmd {
	p.status = sdk.StatusLoading
	svc := p.svc
	log := p.log
	addresses := p.addresses
	return tea.Batch(func() tea.Msg {
		var untainted []string
		for _, addr := range addresses {
			if err := svc.Untaint(context.Background(), addr); err != nil {
				log.Debug("untaint.error", "address", addr, "error", err.Error())
				return untaintResultMsg{Untainted: untainted, Err: fmt.Errorf("%s: %w", addr, err)}
			}
			untainted = append(untainted, addr)
		}
		log.Debug("untaint.success", "count", len(untainted))
		return untaintResultMsg{Untainted: untainted}
	}, p.timer.Start())
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return p, p.timer.Tick()

	case untaintStartMsg:
		return p, p.executeUntaint()

	case untaintResultMsg:
		p.timer.Stop()
		p.untainted = msg.Untainted
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
			return p.executeUntaint()
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
		return sdk.StyleFaintItalic.Render(fmt.Sprintf("Untainting %s... %s", label, p.timer.FormatElapsed()))
	case sdk.StatusDone:
		if len(p.untainted) == 1 {
			return sdk.StyleSuccess.Render(fmt.Sprintf("✓ Untainted %s", p.untainted[0])) +
				"\n" + sdk.StyleFaint.Render("Duration: "+p.timer.FormatElapsed())
		}
		return sdk.StyleSuccess.Render(fmt.Sprintf("✓ Untainted %d resources", len(p.untainted))) +
			"\n" + sdk.StyleFaint.Render("Duration: "+p.timer.FormatElapsed())
	case sdk.StatusError:
		return sdk.StyleError.Render("✗ Untaint failed: " + p.errMsg)
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

// HandleChdirChanged implements sdk.ChdirHandler.
func (p *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	p.svc = p.svc.WithDir(evt.AbsPath)
	p.status = sdk.StatusIdle
	p.addresses = nil
	p.errMsg = ""
	return nil
}
