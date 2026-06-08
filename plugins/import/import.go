package tfimport

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// ImportRequestMsg requests navigation to the import plugin with a pre-filled address.
type ImportRequestMsg struct {
	Address string
}

// importSubmitMsg carries the address/ID gathered by the two-step form.
type importSubmitMsg struct {
	Address string
	ID      string
}

// Plugin implements the standalone import verb. Its input prelude is a
// two-step form (address, then ID) or a direct confirm; the run/result/render
// lifecycle is delegated to the embedded ActionRunner.
type Plugin struct {
	sdk.PluginBase
	sdk.ActionRunner
	input   Input
	address string
	id      string
}

// New creates a new import plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{PluginBase: sdk.NewPluginBase("import", "Import", "Import existing infrastructure into terraform state")}
	p.Svc = svc
	return p
}

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.InitBase(deps)
	p.InitRunner(p.Log)
	return nil
}

// Activate stores the typed input and either confirms directly (address + ID
// both supplied) or opens the two-step form.
func (p *Plugin) Activate(input Input) tea.Cmd {
	if p.Busy() {
		return nil
	}
	p.input = input
	p.address = input.Addr
	p.id = input.ID
	p.Reset()
	if input.Addr != "" && input.ID != "" {
		return p.confirmImport()
	}
	p.id = ""
	return p.requestAddress()
}

func (p *Plugin) requestAddress() tea.Cmd {
	address := p.address
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputText("Resource address:", address, func(addr string) tea.Cmd {
				if addr == "" {
					return func() tea.Msg { return sdk.DeactivateMsg{} }
				}
				return p.requestID(addr)
			}),
		}
	}
}

func (p *Plugin) requestID(address string) tea.Cmd {
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputText("Resource ID:", "", func(id string) tea.Cmd {
				if id == "" {
					return func() tea.Msg { return sdk.DeactivateMsg{} }
				}
				return func() tea.Msg { return importSubmitMsg{Address: address, ID: id} }
			}),
		}
	}
}

func (p *Plugin) confirmImport() tea.Cmd {
	p.Arm(p.spec())
	address, id := p.address, p.id
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Import %s as %s?", id, address),
				func() tea.Cmd { return p.Start() },
			),
		}
	}
}

func (p *Plugin) spec() sdk.ActionSpec {
	address, id := p.address, p.id
	svc := p.Svc
	return sdk.ActionSpec{
		Verb: "import",
		Run: func(ctx context.Context) ([]string, error) {
			if err := svc.Import(ctx, address, id); err != nil {
				return nil, err
			}
			return []string{address}, nil
		},
		OnSuccess:  []tea.Msg{sdk.StateRefreshedEvent{}, sdk.PlanInvalidatedEvent{}},
		Running:    func() string { return "Importing " + address },
		Done:       func([]string) string { return fmt.Sprintf("✓ Imported %s as %s", id, address) },
		ErrorLabel: "Import failed",
		OfferPlan:  true,
	}
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	if submit, ok := msg.(importSubmitMsg); ok {
		p.address = submit.Address
		p.id = submit.ID
		return p, p.confirmImport()
	}
	if handled, cmd := p.ActionRunner.Update(msg); handled {
		return p, cmd
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		return p, p.StandardKeys(key)
	}
	return p, nil
}

func (p *Plugin) View(_, _ int) string {
	if p.CurrentStatus() == sdk.StatusIdle {
		return sdk.StyleFaintItalic.Render("Import resource into terraform state...")
	}
	return p.ActionRunner.View()
}

// Hints uses q-to-quit (standalone verb) rather than the runner's esc-to-back.
func (p *Plugin) Hints() []sdk.KeyHint {
	switch p.CurrentStatus() {
	case sdk.StatusDone:
		return []sdk.KeyHint{{Key: "p", Description: "plan"}, sdk.HintCancel}
	case sdk.StatusError:
		return (sdk.HintSetRetry | sdk.HintSetQuit).Hints()
	default:
		return (sdk.HintSetQuit).Hints()
	}
}

// HandleContextChanged resets the runner and clears targets on a context switch.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if !p.HandleContextChangedDefault(ev) {
		return nil
	}
	p.Reset()
	p.address = ""
	p.id = ""
	return nil
}
