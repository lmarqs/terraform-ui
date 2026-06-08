package taint

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// TaintRequestMsg requests navigation to the taint plugin with target addresses.
type TaintRequestMsg struct {
	Addresses []string
}

// Plugin implements the standalone taint verb. The confirm prelude is its own;
// the run/result/render lifecycle is delegated to the embedded ActionRunner.
type Plugin struct {
	sdk.PluginBase
	sdk.ActionRunner
	input     Input
	addresses []string
}

// New creates a new taint plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{PluginBase: sdk.NewPluginBase("taint", "Taint", "Mark resources for recreation on next apply")}
	p.Svc = svc
	return p
}

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.InitBase(deps)
	p.InitRunner(p.Log)
	return nil
}

// Activate stores the typed input, arms the runner with the taint spec, and
// requests confirmation. Returns immediately if already running or if there is
// nothing to taint.
func (p *Plugin) Activate(input Input) tea.Cmd {
	if p.Busy() {
		return nil
	}
	p.input = input
	p.addresses = input.Addrs
	if len(p.addresses) == 0 {
		return func() tea.Msg { return sdk.DeactivateMsg{} }
	}
	p.Arm(p.spec())
	return p.confirm()
}

func (p *Plugin) confirm() tea.Cmd {
	prompt := fmt.Sprintf("Taint %s? (will recreate on next apply)", p.addresses[0])
	if len(p.addresses) > 1 {
		prompt = fmt.Sprintf("Taint %d resources? (will recreate on next apply)\n  %s", len(p.addresses), strings.Join(p.addresses, "\n  "))
	}
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(prompt, func() tea.Cmd { return p.Start() }),
		}
	}
}

func (p *Plugin) spec() sdk.ActionSpec {
	addrs := p.addresses
	svc := p.Svc
	return sdk.ActionSpec{
		Verb: "taint",
		Run: func(ctx context.Context) ([]string, error) {
			var done []string
			for _, addr := range addrs {
				if err := svc.Taint(ctx, addr); err != nil {
					return done, fmt.Errorf("%s: %w", addr, err)
				}
				done = append(done, addr)
			}
			return done, nil
		},
		OnSuccess:  []tea.Msg{sdk.PlanInvalidatedEvent{}},
		Idle:       "Waiting for confirmation...",
		Running:    func() string { return "Tainting " + label(addrs) },
		Done:       doneLabel,
		ErrorLabel: "Taint failed",
		OfferPlan:  true,
	}
}

func label(addrs []string) string {
	if len(addrs) > 1 {
		return fmt.Sprintf("%d resources", len(addrs))
	}
	return addrs[0]
}

func doneLabel(done []string) string {
	if len(done) == 1 {
		return "✓ Tainted " + done[0]
	}
	return fmt.Sprintf("✓ Tainted %d resources", len(done))
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	if handled, cmd := p.ActionRunner.Update(msg); handled {
		return p, cmd
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		return p, p.StandardKeys(key)
	}
	return p, nil
}

func (p *Plugin) View(_, _ int) string { return p.ActionRunner.View() }

// HandleContextChanged resets the runner and clears targets on a context switch.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if !p.HandleContextChangedDefault(ev) {
		return nil
	}
	p.Reset()
	p.addresses = nil
	return nil
}
