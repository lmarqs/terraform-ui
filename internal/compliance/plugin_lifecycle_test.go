// Package compliance enforces the plugin lifecycle contract across all plugins.
//
// These tests verify structural guarantees that the message routing system
// (ADR-0012) depends on. A failure here means a plugin violates the contract
// and will cause bugs under real navigation (stuck loading, lost results,
// tick explosions).
//
// Rules enforced:
//
//  1. Event handlers must return nil — they run on ALL plugins regardless of
//     which is active. Any tea.Cmd returned produces messages that broadcast
//     to all plugins, and the handler's context (cursor, filter) may be stale.
//
//  2. Update must ignore unrecognized message types — broadcast routing delivers
//     every plugin's result messages to every other plugin. Plugins must pass
//     through cleanly for types they don't own.
//
//  3. TimerTickMsg on an inactive timer must return nil — prevents exponential
//     tick growth when multiple plugins receive the same tick during broadcast.
//
//  4. Activate while Loading must not double-start — when a user navigates away
//     and back, the in-flight operation is still running. Activate must only
//     resume the timer display, not launch a duplicate operation.
package compliance

import (
	"context"
	"io"
	"log/slog"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
	"github.com/lmarqs/terraform-ui/plugins/apply"
	"github.com/lmarqs/terraform-ui/plugins/blastradius"
	"github.com/lmarqs/terraform-ui/plugins/chdir"
	"github.com/lmarqs/terraform-ui/plugins/console"
	tfuicontext "github.com/lmarqs/terraform-ui/plugins/context"
	"github.com/lmarqs/terraform-ui/plugins/forceunlock"
	tfuiimport "github.com/lmarqs/terraform-ui/plugins/import"
	tfuiinit "github.com/lmarqs/terraform-ui/plugins/init"
	"github.com/lmarqs/terraform-ui/plugins/output"
	"github.com/lmarqs/terraform-ui/plugins/phantom"
	"github.com/lmarqs/terraform-ui/plugins/plan"
	"github.com/lmarqs/terraform-ui/plugins/risk"
	"github.com/lmarqs/terraform-ui/plugins/state"
	"github.com/lmarqs/terraform-ui/plugins/taint"
	"github.com/lmarqs/terraform-ui/plugins/untaint"
	"github.com/lmarqs/terraform-ui/plugins/validate"
	"github.com/lmarqs/terraform-ui/plugins/version"
	"github.com/lmarqs/terraform-ui/plugins/workspace"
)

var nopSvc = &nopService{}

func allPlugins() []sdk.Plugin {
	return []sdk.Plugin{
		apply.New(nopSvc),
		blastradius.New(nopSvc),
		chdir.New(nopSvc),
		console.New(nopSvc),
		tfuicontext.New(nopSvc),
		forceunlock.New(nopSvc),
		tfuiimport.New(nopSvc),
		tfuiinit.New(nopSvc),
		output.New(nopSvc),
		phantom.New(nopSvc),
		plan.New(nopSvc),
		risk.New(nopSvc),
		state.New(nopSvc),
		taint.New(nopSvc),
		untaint.New(nopSvc),
		validate.New(nopSvc),
		version.New(nopSvc),
		workspace.New(nopSvc),
	}
}

func initPlugin(p sdk.Plugin) {
	p.Init(&sdk.Context{
		WorkingDir: "/tmp",
		Service:    nopSvc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Pins:       sdk.NewPinService(),
		Options:    &sdk.ResolvedOptions{},
	})
}

// --- Rule 1: Event handlers must return nil ---

func TestAllPlugins_WhenEventHandlerCalled_ShouldReturnNilCmd(t *testing.T) {
	for _, p := range allPlugins() {
		initPlugin(p)
		id := p.ID()

		if h, ok := p.(sdk.ChdirHandler); ok {
			t.Run(id+"/HandleChdirChanged", func(t *testing.T) {
				cmd := h.HandleChdirChanged(sdk.ChdirChangedEvent{RelPath: "mod", AbsPath: "/tmp/mod"})
				if cmd != nil {
					t.Errorf("returned non-nil cmd — event handlers must not start async operations (ADR-0012)")
				}
			})
		}
		if h, ok := p.(sdk.WorkspaceHandler); ok {
			t.Run(id+"/HandleWorkspaceChanged", func(t *testing.T) {
				cmd := h.HandleWorkspaceChanged(sdk.WorkspaceChangedEvent{Name: "dev"})
				if cmd != nil {
					t.Errorf("returned non-nil cmd — event handlers must not start async operations (ADR-0012)")
				}
			})
		}
		if h, ok := p.(sdk.PlanCompletedHandler); ok {
			t.Run(id+"/HandlePlanCompleted", func(t *testing.T) {
				cmd := h.HandlePlanCompleted(sdk.PlanCompletedEvent{ResourceCount: 5})
				if cmd != nil {
					t.Errorf("returned non-nil cmd — event handlers must not start async operations (ADR-0012)")
				}
			})
		}
		if h, ok := p.(sdk.PinsHandler); ok {
			t.Run(id+"/HandlePinsChanged", func(t *testing.T) {
				cmd := h.HandlePinsChanged(sdk.PinsChangedEvent{Addresses: []string{"a"}})
				if cmd != nil {
					t.Errorf("returned non-nil cmd — event handlers must not start async operations (ADR-0012)")
				}
			})
		}
		if h, ok := p.(sdk.PlanInvalidatedHandler); ok {
			t.Run(id+"/HandlePlanInvalidated", func(t *testing.T) {
				cmd := h.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
				if cmd != nil {
					t.Errorf("returned non-nil cmd — event handlers must not start async operations (ADR-0012)")
				}
			})
		}
		if h, ok := p.(sdk.LockDetectedHandler); ok {
			t.Run(id+"/HandleLockDetected", func(t *testing.T) {
				cmd := h.HandleLockDetected(sdk.LockDetectedEvent{Lock: &sdk.StateLock{ID: "x"}})
				if cmd != nil {
					t.Errorf("returned non-nil cmd — event handlers must not start async operations (ADR-0012)")
				}
			})
		}
		if h, ok := p.(sdk.LockClearedHandler); ok {
			t.Run(id+"/HandleLockCleared", func(t *testing.T) {
				cmd := h.HandleLockCleared(sdk.LockClearedEvent{})
				if cmd != nil {
					t.Errorf("returned non-nil cmd — event handlers must not start async operations (ADR-0012)")
				}
			})
		}
		if h, ok := p.(sdk.StateRefreshedHandler); ok {
			t.Run(id+"/HandleStateRefreshed", func(t *testing.T) {
				cmd := h.HandleStateRefreshed(sdk.StateRefreshedEvent{})
				if cmd != nil {
					t.Errorf("returned non-nil cmd — event handlers must not start async operations (ADR-0012)")
				}
			})
		}
	}
}

// --- Rule 2: Update must ignore unrecognized messages ---

type foreignMsg struct{}

func TestAllPlugins_WhenUpdateReceivesForeignMessage_ShouldReturnSelfAndNil(t *testing.T) {
	for _, p := range allPlugins() {
		initPlugin(p)
		t.Run(p.ID(), func(t *testing.T) {
			updated, cmd := p.Update(foreignMsg{})
			if updated != p {
				t.Errorf("returned different instance — broadcast safety requires pass-through for unknown types")
			}
			if cmd != nil {
				t.Errorf("returned non-nil cmd — broadcast safety requires nil for unknown types")
			}
		})
	}
}

// --- Rule 3: TimerTickMsg on inactive timer must not leak ---

func TestAllPlugins_WhenTimerTickReceivedWithInactiveTimer_ShouldReturnNilCmd(t *testing.T) {
	for _, p := range allPlugins() {
		initPlugin(p)
		t.Run(p.ID(), func(t *testing.T) {
			_, cmd := p.Update(ui.TimerTickMsg{})
			if cmd != nil {
				t.Errorf("returned non-nil cmd — inactive timer must not propagate ticks (prevents exponential growth)")
			}
		})
	}
}

// --- Rule 4: Activate while Loading must not double-start ---

type statusReporter interface {
	Status() sdk.Status
}

func TestAllPlugins_WhenActivatedWhileLoading_ShouldNotRestartOperation(t *testing.T) {
	for _, p := range allPlugins() {
		initPlugin(p)

		activatable, ok := p.(sdk.Activatable)
		if !ok {
			continue
		}
		reporter, hasStatus := p.(statusReporter)
		if !hasStatus {
			continue
		}

		// First activation starts the operation
		activatable.Activate()

		// Only test plugins that entered Loading
		if reporter.Status() != sdk.StatusLoading {
			continue
		}

		t.Run(p.ID(), func(t *testing.T) {
			cmd := activatable.Activate()
			if cmd == nil {
				return
			}
			msg := cmd()
			switch msg.(type) {
			case ui.TimerTickMsg, tea.BatchMsg:
				// Timer tick or batch of ticks — valid for resuming display
			default:
				t.Errorf("produced %T — must only resume timer, not restart the operation", msg)
			}
		})
	}
}

// --- Test infrastructure ---

type nopService struct{}

func (s *nopService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (s *nopService) Apply(_ context.Context, _ sdk.ApplyOptions) error { return nil }
func (s *nopService) StateList(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
	return nil, nil
}
func (s *nopService) Show(_ context.Context, _ string) (string, error)   { return "", nil }
func (s *nopService) Workspace(_ context.Context) (string, error)        { return "default", nil }
func (s *nopService) WorkspaceList(_ context.Context) ([]string, error)  { return []string{"default"}, nil }
func (s *nopService) WorkspaceSelect(_ context.Context, _ string) error  { return nil }
func (s *nopService) WorkspaceNew(_ context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	return nil
}
func (s *nopService) WorkspaceDelete(_ context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	return nil
}
func (s *nopService) StateRm(_ context.Context, _ string) error      { return nil }
func (s *nopService) StateMove(_ context.Context, _, _ string) error { return nil }
func (s *nopService) Import(_ context.Context, _, _ string) error    { return nil }
func (s *nopService) Taint(_ context.Context, _ string) error        { return nil }
func (s *nopService) Untaint(_ context.Context, _ string) error      { return nil }
func (s *nopService) Validate(_ context.Context) ([]sdk.Diagnostic, error) {
	return nil, nil
}
func (s *nopService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}
func (s *nopService) Refresh(_ context.Context) error                     { return nil }
func (s *nopService) Init(_ context.Context, _ sdk.InitOptions) error     { return nil }
func (s *nopService) ForceUnlock(_ context.Context, _ string) error       { return nil }
func (s *nopService) Version(_ context.Context) (*sdk.VersionInfo, error) { return nil, nil }
func (s *nopService) WithDir(_ string) sdk.Service                        { return s }
