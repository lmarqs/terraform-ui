package sdktest

import (
	"io"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// PluginDepsHarness wires a plugin's PluginDeps to a mutable Context plus
// pin-request capture, replacing the boilerplate every plugin test used to
// repeat. The Ctx pointer is the live snapshot that deps.Context() returns;
// tests mutate Ctx (or assign a new one) to simulate a context replacement.
//
// PinRequests captures every address pinned via deps.Pin; ClearPinsCount
// counts deps.ClearPins invocations. The harness does not auto-replay these
// requests onto Ctx — that is the App's job. Tests that need the next
// snapshot to reflect a pin should mutate Ctx.Pins explicitly.
type PluginDepsHarness struct {
	Ctx             *sdk.Context
	Deps            *sdk.PluginDeps
	PinRequests     []string
	ClearPinsCount  int
}

// NewDeps returns a harness whose Context is seeded from the supplied
// Service. Logger discards output; Pin and ClearPins record into the harness.
func NewDeps(svc sdk.Service) *PluginDepsHarness {
	h := &PluginDepsHarness{
		Ctx: &sdk.Context{
			WorkingDir: "/tmp",
			Workspace:  "default",
			Service:    svc,
		},
	}
	h.Deps = &sdk.PluginDeps{
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Service: svc,
		Context: func() *sdk.Context { return h.Ctx },
		Pin: func(address string) tea.Cmd {
			return func() tea.Msg {
				h.PinRequests = append(h.PinRequests, address)
				return sdk.PinToggleRequestMsg{Address: address}
			}
		},
		ClearPins: func() tea.Cmd {
			return func() tea.Msg {
				h.ClearPinsCount++
				return sdk.PinClearRequestMsg{}
			}
		},
	}
	return h
}
