package tfimport

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// ImportRequestMsg requests navigation to the import plugin with a pre-filled address.
type ImportRequestMsg struct {
	Address string
}

// importSubmitMsg triggers confirmation after form input.
type importSubmitMsg struct {
	Address string
	ID      string
}

// importStartMsg triggers execution after confirmation.
type importStartMsg struct{}

// importResultMsg is sent when the import operation completes.
type importResultMsg struct {
	Address string
	ID      string
	Err     error
}

const (
	StatusForm = sdk.Status(10)
)

// Plugin implements the standalone import verb.
type Plugin struct {
	sdk.PluginBase
	timer    ui.Timer
	status   sdk.Status
	input    Input
	address  string
	id       string
	errMsg   string
	cancelFn context.CancelFunc
}

// New creates a new import plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{PluginBase: sdk.NewPluginBase("import", "Import", "Import existing infrastructure into terraform state")}
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

// Activate is the input port: cmd/tfui parses CLI flags into Input and hands
// the typed value to the plugin. When the cmd path provides both Addr and ID,
// the form is skipped and the plugin jumps straight to the confirm step.
// When only Addr is provided (TUI flow), the form runs starting at the ID
// field with Addr pre-filled.
func (p *Plugin) Activate(input Input) tea.Cmd {
	if p.status == sdk.StatusLoading {
		return nil
	}
	p.input = input
	p.address = input.Addr
	p.id = input.ID
	p.errMsg = ""
	if input.Addr != "" && input.ID != "" {
		p.status = sdk.StatusIdle
		return p.confirmImport()
	}
	p.status = StatusForm
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
	address := p.address
	id := p.id
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Import %s as %s?", id, address),
				func() tea.Cmd {
					return func() tea.Msg { return importStartMsg{} }
				},
			),
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

func (p *Plugin) executeImport() tea.Cmd {
	p.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFn = cancel
	p.status = sdk.StatusLoading
	svc := p.Svc
	log := p.Log
	address := p.address
	id := p.id
	return tea.Batch(func() tea.Msg {
		err := svc.Import(ctx, address, id)
		if err != nil {
			log.Debug("import.error", "address", address, "id", id, "error", err.Error())
		} else {
			log.Debug("import.success", "address", address, "id", id)
		}
		return importResultMsg{Address: address, ID: id, Err: err}
	}, p.timer.Start())
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return p, p.timer.Tick()

	case importSubmitMsg:
		p.address = msg.Address
		p.id = msg.ID
		return p, p.confirmImport()

	case importStartMsg:
		return p, p.executeImport()

	case importResultMsg:
		p.timer.Stop()
		if msg.Err != nil {
			p.status = sdk.StatusError
			p.errMsg = msg.Err.Error()
		} else {
			p.status = sdk.StatusDone
			return p, tea.Batch(
				func() tea.Msg { return sdk.StateRefreshedEvent{} },
				func() tea.Msg { return sdk.PlanInvalidatedEvent{} },
			)
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
			return p.executeImport()
		case "esc":
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	case StatusForm, sdk.StatusIdle:
		switch msg.String() {
		case "esc":
			return func() tea.Msg { return sdk.DeactivateMsg{} }
		}
	}
	return nil
}

func (p *Plugin) View(_, _ int) string {
	switch p.status {
	case StatusForm, sdk.StatusIdle:
		return sdk.StyleFaintItalic.Render("Import resource into terraform state...")
	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render(fmt.Sprintf("Importing %s... %s", p.address, p.timer.FormatElapsed()))
	case sdk.StatusDone:
		return sdk.StyleSuccess.Render(fmt.Sprintf("✓ Imported %s as %s", p.id, p.address)) +
			"\n" + sdk.StyleFaint.Render("Duration: "+p.timer.FormatElapsed())
	case sdk.StatusError:
		return sdk.StyleError.Render("✗ Import failed: " + p.errMsg)
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
		return (sdk.HintSetRetry | sdk.HintSetQuit).Hints()
	default:
		return (sdk.HintSetQuit).Hints()
	}
}

// HandleContextChanged implements sdk.ContextChangedHandler.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if !p.HandleContextChangedDefault(ev) {
		return nil
	}
	p.status = sdk.StatusIdle
	p.address = ""
	p.id = ""
	p.errMsg = ""
	return nil
}
