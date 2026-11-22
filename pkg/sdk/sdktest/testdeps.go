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
// PinRequests captures every address pinned via deps.Pin, flattened across
// groups in request order — one gesture on a module row appends every resource
// address beneath it. ClearPinsCount
// counts deps.ClearPins invocations. The harness does not auto-replay these
// requests onto Ctx — that is the App's job. Tests that need the next
// snapshot to reflect a pin should mutate Ctx.Pins explicitly.
type PluginDepsHarness struct {
	Ctx            *sdk.Context
	Deps           *sdk.PluginDeps
	PinRequests    []string
	ClearPinsCount int
	// AssessmentEvent is the canned result deps.RequestAssessment emits, its
	// Request set to the ID that call returned.
	AssessmentEvent sdk.RiskAssessedEvent
	// AssessmentRequests records the summary of every deps.RequestAssessment run.
	AssessmentRequests []*sdk.PlanSummary
	lastAssessment     sdk.AssessmentID
	// PlanEvent is the canned result deps.RequestPlan emits, its Request set to
	// the ID that call returned.
	PlanEvent sdk.PlanFetchedEvent
	// PlanRequests counts the deps.RequestPlan Cmds that ran.
	PlanRequests int
	lastPlan     sdk.PlanRequestID
}

// NewDeps returns a harness whose Context is seeded from the supplied
// Service. Logger discards output; Pin, ClearPins, RequestAssessment and
// RequestPlan record into the harness.
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
		Pin: func(addresses ...string) tea.Cmd {
			return func() tea.Msg {
				h.PinRequests = append(h.PinRequests, addresses...)
				return sdk.PinToggleRequestMsg{Addresses: addresses}
			}
		},
		ClearPins: func() tea.Cmd {
			return func() tea.Msg {
				h.ClearPinsCount++
				return sdk.PinClearRequestMsg{}
			}
		},
		RequestAssessment: func(summary *sdk.PlanSummary) (sdk.AssessmentID, tea.Cmd) {
			h.lastAssessment++
			id := h.lastAssessment
			return id, func() tea.Msg {
				h.AssessmentRequests = append(h.AssessmentRequests, summary)
				ev := h.AssessmentEvent
				ev.Request = id
				return ev
			}
		},
		RequestPlan: func() (sdk.PlanRequestID, tea.Cmd) {
			h.lastPlan++
			id := h.lastPlan
			return id, func() tea.Msg {
				h.PlanRequests++
				ev := h.PlanEvent
				ev.Request = id
				return ev
			}
		},
	}
	return h
}
