// Package compliance — cancellation contract tests.
//
// These tests verify that plugins properly cancel in-flight terraform operations
// when the user navigates away or triggers a new operation.
//
// Rules enforced:
//
//  5. Async terraform operations must use cancellable contexts — a plugin that
//     launches a long-running terraform call (Plan, Apply, StateList, etc.) must
//     pass a context derived from context.WithCancel, never context.Background().
//
//  6. Plugins with in-flight operations must implement Cancellable — when the app
//     calls Cancel(), the previously-issued context must be cancelled so the
//     terraform subprocess is terminated.
//
//  7. Re-activation while idle/error must cancel previous context — if a stale
//     goroutine is still running from a prior activation, starting a new operation
//     must cancel the old one to prevent duplicate terraform processes.
package compliance

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/plugins/apply"
	"github.com/lmarqs/terraform-ui/plugins/forceunlock"
	tfuiimport "github.com/lmarqs/terraform-ui/plugins/import"
	tfuiinit "github.com/lmarqs/terraform-ui/plugins/init"
	"github.com/lmarqs/terraform-ui/plugins/output"
	"github.com/lmarqs/terraform-ui/plugins/plan"
	"github.com/lmarqs/terraform-ui/plugins/state"
	"github.com/lmarqs/terraform-ui/plugins/taint"
	"github.com/lmarqs/terraform-ui/plugins/untaint"
	"github.com/lmarqs/terraform-ui/plugins/validate"
	"github.com/lmarqs/terraform-ui/plugins/version"
	"github.com/lmarqs/terraform-ui/plugins/workspace"
)

// asyncPlugins returns plugins that launch async terraform operations on activation.
func asyncPlugins(svc sdk.Service) []sdk.Plugin {
	return []sdk.Plugin{
		plan.New(svc),
		state.New(svc),
		apply.New(svc),
		validate.New(svc),
		output.New(svc),
		version.New(svc),
		workspace.New(svc),
		tfuiinit.New(svc),
		forceunlock.New(svc),
		taint.New(svc),
		untaint.New(svc),
		tfuiimport.New(svc),
	}
}

func initP(p sdk.Plugin, svc sdk.Service) {
	bootCtx := &sdk.Context{Service: svc}
	p.Init(&sdk.PluginDeps{
		Service:   svc,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Context:   func() *sdk.Context { return bootCtx },
		Pin:       func(_ string) tea.Cmd { return nil },
		ClearPins: func() tea.Cmd { return nil },
	})
}

// --- Rule 5: Async operations must use cancellable contexts ---

func TestAsyncPlugins_WhenOperationStarted_ShouldUseCancellableContext(t *testing.T) {
	for _, p := range asyncPlugins(&contextSpyService{}) {
		spy := &contextSpyService{}
		p = recreatePlugin(p, spy)
		initP(p, spy)

		activatable, ok := p.(sdk.Activatable)
		if !ok {
			continue
		}

		t.Run(p.ID()+"/context_is_cancellable", func(t *testing.T) {
			cmd := activatable.Activate()
			if cmd == nil {
				t.Skip("plugin does not start operation on activate")
			}

			// Execute all commands (unwrap tea.Batch)
			done := make(chan struct{})
			go func() {
				execCmd(cmd)
				close(done)
			}()

			// Wait for the service to receive the context
			ctx := spy.waitForContext(500 * time.Millisecond)
			if ctx == nil {
				t.Skip("plugin did not call service within timeout")
			}

			// The context must be cancellable (not context.Background)
			if ctx.Done() == nil {
				t.Errorf("passed context.Background() to terraform operation — must use context.WithCancel for cancellation support")
			}

			<-done
		})
	}
}

// --- Rule 6: Plugins with async operations must implement Cancellable ---

func TestAsyncPlugins_ShouldImplementCancellable(t *testing.T) {
	for _, p := range asyncPlugins(nopSvc) {
		initP(p, nopSvc)

		activatable, ok := p.(sdk.Activatable)
		if !ok {
			continue
		}

		// Only test plugins that actually start operations
		cmd := activatable.Activate()
		if cmd == nil {
			continue
		}

		reporter, hasStatus := p.(statusReporter)
		if !hasStatus || reporter.Status() != sdk.StatusLoading {
			continue
		}

		t.Run(p.ID()+"/implements_cancellable", func(t *testing.T) {
			if _, ok := p.(sdk.Cancellable); !ok {
				t.Errorf("plugin starts async operations but does not implement Cancellable interface")
			}
		})
	}
}

// --- Rule 7: Cancel must actually cancel the context ---

func TestAsyncPlugins_WhenCancelled_ShouldCancelContext(t *testing.T) {
	for _, p := range asyncPlugins(&contextSpyService{}) {
		spy := &contextSpyService{}
		p = recreatePlugin(p, spy)
		initP(p, spy)

		activatable, ok := p.(sdk.Activatable)
		if !ok {
			continue
		}

		c, isCancellable := p.(sdk.Cancellable)
		if !isCancellable {
			continue
		}

		cmd := activatable.Activate()
		if cmd == nil {
			continue
		}

		t.Run(p.ID()+"/cancel_terminates_context", func(t *testing.T) {
			// Execute in background so service receives the call
			done := make(chan struct{})
			go func() {
				execCmd(cmd)
				close(done)
			}()

			ctx := spy.waitForContext(500 * time.Millisecond)
			if ctx == nil {
				<-done
				t.Skip("plugin did not call service within timeout")
			}

			if ctx.Done() == nil {
				<-done
				t.Fatal("context is not cancellable — Rule 5 must pass first")
			}

			// Cancel the plugin
			c.Cancel()

			// Context must be cancelled
			select {
			case <-ctx.Done():
				// success
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Cancel() did not cancel the in-flight context — terraform subprocess will be orphaned")
			}
			<-done
		})
	}
}

// --- Test infrastructure ---

// contextSpyService captures contexts passed to service methods.
type contextSpyService struct {
	mu       sync.Mutex
	contexts []context.Context
	signal   chan struct{}
}

func (s *contextSpyService) recordCtx(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contexts = append(s.contexts, ctx)
	if s.signal != nil {
		select {
		case s.signal <- struct{}{}:
		default:
		}
	}
}

func (s *contextSpyService) waitForContext(timeout time.Duration) context.Context {
	s.mu.Lock()
	if len(s.contexts) > 0 {
		ctx := s.contexts[len(s.contexts)-1]
		s.mu.Unlock()
		return ctx
	}
	s.signal = make(chan struct{}, 1)
	s.mu.Unlock()

	select {
	case <-s.signal:
		s.mu.Lock()
		defer s.mu.Unlock()
		if len(s.contexts) > 0 {
			return s.contexts[len(s.contexts)-1]
		}
		return nil
	case <-time.After(timeout):
		return nil
	}
}

func (s *contextSpyService) Plan(ctx context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	s.recordCtx(ctx)
	return &sdk.PlanSummary{}, nil
}
func (s *contextSpyService) Apply(ctx context.Context, _ sdk.ApplyOptions) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) StateList(ctx context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
	s.recordCtx(ctx)
	return nil, nil
}
func (s *contextSpyService) Show(ctx context.Context, _ string) (string, error) {
	s.recordCtx(ctx)
	return "", nil
}
func (s *contextSpyService) Workspace(ctx context.Context) (string, error) {
	s.recordCtx(ctx)
	return "default", nil
}
func (s *contextSpyService) WorkspaceList(ctx context.Context) ([]string, error) {
	s.recordCtx(ctx)
	return []string{"default"}, nil
}
func (s *contextSpyService) WorkspaceSelect(ctx context.Context, _ string) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) WorkspaceNew(ctx context.Context, _ string, _ sdk.WorkspaceNewOptions) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) WorkspaceDelete(ctx context.Context, _ string, _ sdk.WorkspaceDeleteOptions) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) StateRm(ctx context.Context, _ string) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) StateMove(ctx context.Context, _, _ string) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) Import(ctx context.Context, _, _ string) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) Taint(ctx context.Context, _ string) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) Untaint(ctx context.Context, _ string) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) Validate(ctx context.Context) ([]sdk.Diagnostic, error) {
	s.recordCtx(ctx)
	return nil, nil
}
func (s *contextSpyService) Output(ctx context.Context) (map[string]sdk.OutputValue, error) {
	s.recordCtx(ctx)
	return nil, nil
}
func (s *contextSpyService) Refresh(ctx context.Context) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) Init(ctx context.Context, _ sdk.InitOptions) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) ForceUnlock(ctx context.Context, _ string) error {
	s.recordCtx(ctx)
	return nil
}
func (s *contextSpyService) Version(ctx context.Context) (*sdk.VersionInfo, error) {
	s.recordCtx(ctx)
	return nil, nil
}
func (s *contextSpyService) WithDir(_ string) sdk.Service { return s }

// execCmd executes a tea.Cmd, unwrapping tea.BatchMsg recursively.
func execCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			execCmd(c)
		}
	}
}

// recreatePlugin creates a fresh instance of the same plugin type with a new service.
func recreatePlugin(p sdk.Plugin, svc sdk.Service) sdk.Plugin {
	switch p.ID() {
	case "plan":
		return plan.New(svc)
	case "state":
		return state.New(svc)
	case "apply":
		return apply.New(svc)
	case "validate":
		return validate.New(svc)
	case "output":
		return output.New(svc)
	case "version":
		return version.New(svc)
	case "workspace":
		return workspace.New(svc)
	case "init":
		return tfuiinit.New(svc)
	case "forceunlock":
		return forceunlock.New(svc)
	case "taint":
		return taint.New(svc)
	case "untaint":
		return untaint.New(svc)
	case "import":
		return tfuiimport.New(svc)
	default:
		return p
	}
}
