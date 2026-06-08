package forceunlock

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func newTestPlugin(svc *sdktest.MockService) (*Plugin, *sdktest.PluginDepsHarness) {
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	return p, h
}

// driveToTerminal sends a start message for lockID, then feeds the work result
// back so the runner reaches Done/Error. Returns the emitted event cmd.
func driveToTerminal(t *testing.T, p *Plugin, lockID string) tea.Cmd {
	t.Helper()
	_, cmd := p.Update(ForceUnlockStartMsg{LockID: lockID})
	if cmd == nil {
		t.Fatal("ForceUnlockStartMsg should start the unlock")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("start cmd = %T, want tea.BatchMsg", cmd())
	}
	_, event := p.Update(batch[0]()) // batch[0] is the work cmd → result
	return event
}

func TestPlugin_Lifecycle(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	if p.ID() != "forceunlock" {
		t.Errorf("ID() = %q, want %q", p.ID(), "forceunlock")
	}
	if p.Name() != "Force Unlock" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Force Unlock")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(nil); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	if p.Ready() {
		t.Error("Ready() should be false before activation")
	}
}

// TestPlugin_OptionalInterfaces pins the normalization: forceunlock now
// implements sdk.Busy (promoted from ActionRunner), so an in-flight unlock is
// guarded by requireIdle like the other action verbs.
func TestPlugin_OptionalInterfaces(t *testing.T) {
	p := New(&sdktest.MockService{})
	if _, ok := p.(sdk.Cancellable); !ok {
		t.Error("forceunlock must implement sdk.Cancellable")
	}
	if _, ok := p.(sdk.Busy); !ok {
		t.Error("forceunlock must implement sdk.Busy (normalized via ActionRunner)")
	}
}

func TestActivate(t *testing.T) {
	t.Run("lockInfo present → confirm", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.lockInfo = &sdk.StateLock{ID: "lock-abc-123", Who: "user@host"}
		req := p.Activate(Input{})().(sdk.RequestInputMsg)
		if req.Request.Mode != sdk.InputRequestBool {
			t.Errorf("mode = %v, want InputRequestBool", req.Request.Mode)
		}
	})

	t.Run("no lockInfo → offer manual entry", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		req := p.Activate(Input{})().(sdk.RequestInputMsg)
		if req.Request.Mode != sdk.InputRequestBool {
			t.Errorf("mode = %v, want InputRequestBool (manual-entry offer)", req.Request.Mode)
		}
	})

	t.Run("LockID provided → confirm", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		req := p.Activate(Input{LockID: "lock-from-cli"})().(sdk.RequestInputMsg)
		if req.Request.Mode != sdk.InputRequestBool {
			t.Errorf("mode = %v, want InputRequestBool", req.Request.Mode)
		}
	})

	t.Run("LockID + Force → start directly", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		msg := p.Activate(Input{LockID: "lock-from-cli", Force: true})()
		start, ok := msg.(ForceUnlockStartMsg)
		if !ok {
			t.Fatalf("got %T, want ForceUnlockStartMsg", msg)
		}
		if start.LockID != "lock-from-cli" {
			t.Errorf("LockID = %q, want lock-from-cli", start.LockID)
		}
	})

	t.Run("while busy → nil", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.Update(ForceUnlockStartMsg{LockID: "x"}) // → Loading
		if !p.Busy() {
			t.Fatal("precondition: should be Busy after start")
		}
		if cmd := p.Activate(Input{LockID: "y", Force: true}); cmd != nil {
			t.Error("Activate() while busy should return nil")
		}
	})
}

func TestConfirmUnlock(t *testing.T) {
	t.Run("accept yields a start message for the lock", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.lockInfo = &sdk.StateLock{ID: "lock-abc"}
		req := p.Activate(Input{})().(sdk.RequestInputMsg)
		startCmd := req.Request.Callback("y")
		if startCmd == nil {
			t.Fatal("confirm should return a start cmd")
		}
		start, ok := startCmd().(ForceUnlockStartMsg)
		if !ok || start.LockID != "lock-abc" {
			t.Errorf("got %#v, want ForceUnlockStartMsg{lock-abc}", startCmd())
		}
	})

	t.Run("decline returns nil", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.lockInfo = &sdk.StateLock{ID: "lock-abc"}
		req := p.Activate(Input{})().(sdk.RequestInputMsg)
		if req.Request.Callback("n") != nil {
			t.Error("declining should return nil")
		}
	})
}

func TestManualEntry(t *testing.T) {
	t.Run("full flow: confirm → enter id → confirm → start", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		offer := p.Activate(Input{})().(sdk.RequestInputMsg)
		idPrompt := offer.Request.Callback("y")().(sdk.RequestInputMsg)
		if idPrompt.Request.Mode != sdk.InputRequestText {
			t.Errorf("id prompt mode = %v, want Text", idPrompt.Request.Mode)
		}
		confirm := idPrompt.Request.Callback("manual-lock-id")().(sdk.RequestInputMsg)
		if confirm.Request.Mode != sdk.InputRequestBool {
			t.Errorf("confirm mode = %v, want Bool", confirm.Request.Mode)
		}
		start := confirm.Request.Callback("y")().(ForceUnlockStartMsg)
		if start.LockID != "manual-lock-id" {
			t.Errorf("LockID = %q, want manual-lock-id", start.LockID)
		}
	})

	t.Run("empty lock id deactivates", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		offer := p.Activate(Input{})().(sdk.RequestInputMsg)
		idPrompt := offer.Request.Callback("y")().(sdk.RequestInputMsg)
		cmd := idPrompt.Request.Callback("")
		if _, ok := cmd().(sdk.DeactivateMsg); !ok {
			t.Errorf("empty id → %T, want DeactivateMsg", cmd())
		}
	})

	t.Run("declining the offer returns nil", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		offer := p.Activate(Input{})().(sdk.RequestInputMsg)
		if offer.Request.Callback("n") != nil {
			t.Error("declining manual entry should return nil")
		}
	})
}

func TestSpecAndRun(t *testing.T) {
	t.Run("metadata wires force-unlock semantics", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		spec := p.spec("lock-1")
		if spec.Name != "forceunlock" {
			t.Errorf("Name = %q", spec.Name)
		}
		if len(spec.OnSuccess) != 2 {
			t.Fatalf("OnSuccess len = %d, want 2", len(spec.OnSuccess))
		}
		if _, ok := spec.OnSuccess[0].(sdk.LockClearedEvent); !ok {
			t.Errorf("OnSuccess[0] = %T, want LockClearedEvent", spec.OnSuccess[0])
		}
		if _, ok := spec.OnSuccess[1].(sdk.PlanInvalidatedEvent); !ok {
			t.Errorf("OnSuccess[1] = %T, want PlanInvalidatedEvent", spec.OnSuccess[1])
		}
	})

	t.Run("Run unlocks the given lock id", func(t *testing.T) {
		svc := &sdktest.MockService{}
		p, _ := newTestPlugin(svc)
		done, err := p.spec("lock-99").Run(context.Background())
		if err != nil || len(done) != 1 || done[0] != "lock-99" {
			t.Errorf("Run() = %v, %v", done, err)
		}
		if len(svc.ForceUnlockCalls) != 1 || svc.ForceUnlockCalls[0] != "lock-99" {
			t.Errorf("ForceUnlockCalls = %v, want [lock-99]", svc.ForceUnlockCalls)
		}
	})

	t.Run("Run surfaces unlock errors", func(t *testing.T) {
		svc := &sdktest.MockService{ForceUnlockFn: func(context.Context, string) error {
			return errors.New("denied")
		}}
		p, _ := newTestPlugin(svc)
		if _, err := p.spec("lock-1").Run(context.Background()); err == nil {
			t.Error("Run() should surface the unlock error")
		}
	})
}

func TestDriveToDone_ClearsLockAndEmitsEvents(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.lockInfo = &sdk.StateLock{ID: "lock-123"}

	eventCmd := driveToTerminal(t, p, "lock-123")
	if !p.Ready() {
		t.Fatalf("status = %v, want Done", p.CurrentStatus())
	}
	if p.lockInfo != nil {
		t.Error("lockInfo should be cleared on success")
	}
	if eventCmd == nil {
		t.Fatal("success should emit events")
	}
	batch := eventCmd().(tea.BatchMsg)
	var cleared, invalidated bool
	for _, sub := range batch {
		switch sub().(type) {
		case sdk.LockClearedEvent:
			cleared = true
		case sdk.PlanInvalidatedEvent:
			invalidated = true
		}
	}
	if !cleared || !invalidated {
		t.Errorf("events: cleared=%v invalidated=%v, want both", cleared, invalidated)
	}
	if len(svc.ForceUnlockCalls) != 1 || svc.ForceUnlockCalls[0] != "lock-123" {
		t.Errorf("ForceUnlockCalls = %v", svc.ForceUnlockCalls)
	}
}

func TestDriveToError_SetsMessage(t *testing.T) {
	svc := &sdktest.MockService{ForceUnlockFn: func(context.Context, string) error {
		return errors.New("denied")
	}}
	p, _ := newTestPlugin(svc)

	eventCmd := driveToTerminal(t, p, "lock-err")
	if p.CurrentStatus() != sdk.StatusError {
		t.Errorf("status = %v, want Error", p.CurrentStatus())
	}
	if p.ErrMessage() == "" {
		t.Error("ErrMessage should be set on failure")
	}
	if eventCmd != nil {
		t.Error("failure should not emit events")
	}
}

func TestKeys(t *testing.T) {
	t.Run("q deactivates", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		cmd := p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if _, ok := cmd().(sdk.DeactivateMsg); !ok {
			t.Errorf("q → %T, want DeactivateMsg", cmd())
		}
	})

	t.Run("esc deactivates (via Update)", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if _, ok := cmd().(sdk.DeactivateMsg); !ok {
			t.Errorf("esc → %T, want DeactivateMsg", cmd())
		}
	})

	t.Run("ctrl+r in error re-activates", func(t *testing.T) {
		svc := &sdktest.MockService{ForceUnlockFn: func(context.Context, string) error {
			return errors.New("x")
		}}
		p, _ := newTestPlugin(svc)
		p.lockInfo = &sdk.StateLock{ID: "lock-123"}
		driveToTerminal(t, p, "lock-123") // → Error
		if cmd := p.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR}); cmd == nil {
			t.Error("ctrl+r in error should re-activate")
		}
	})

	t.Run("ctrl+r when not in error does nothing", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		if cmd := p.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR}); cmd != nil {
			t.Error("ctrl+r outside error should return nil")
		}
	})

	t.Run("other keys do nothing", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		if cmd := p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}); cmd != nil {
			t.Error("unhandled key should return nil")
		}
	})
}

func TestView(t *testing.T) {
	t.Run("idle without lock", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		if p.View(80, 24) == "" {
			t.Error("idle no-lock view should not be empty")
		}
	})

	t.Run("idle with lock shows lock info", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.lockInfo = &sdk.StateLock{ID: "lock-abc", Who: "user@host"}
		if p.View(80, 24) == "" {
			t.Error("idle with-lock view should not be empty")
		}
	})

	t.Run("loading", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.Update(ForceUnlockStartMsg{LockID: "lock-xyz"}) // → Loading
		if p.View(80, 24) == "" {
			t.Error("loading view should not be empty")
		}
	})

	t.Run("done", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		driveToTerminal(t, p, "lock-xyz") // → Done
		if p.View(80, 24) == "" {
			t.Error("done view should not be empty")
		}
	})

	t.Run("error", func(t *testing.T) {
		svc := &sdktest.MockService{ForceUnlockFn: func(context.Context, string) error {
			return errors.New("boom")
		}}
		p, _ := newTestPlugin(svc)
		driveToTerminal(t, p, "lock-err") // → Error
		if p.View(80, 24) == "" {
			t.Error("error view should not be empty")
		}
	})
}

func TestHints(t *testing.T) {
	t.Run("idle offers back and quit", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		if len(p.Hints()) == 0 {
			t.Error("idle hints should not be empty")
		}
	})

	t.Run("error offers retry", func(t *testing.T) {
		svc := &sdktest.MockService{ForceUnlockFn: func(context.Context, string) error {
			return errors.New("x")
		}}
		p, _ := newTestPlugin(svc)
		driveToTerminal(t, p, "lock-1") // → Error
		var retry bool
		for _, h := range p.Hints() {
			if h.Description == "retry" {
				retry = true
			}
		}
		if !retry {
			t.Error("error hints should contain retry")
		}
	})
}

func TestLockEventHandlers(t *testing.T) {
	t.Run("HandleLockDetected stores lock", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		lock := &sdk.StateLock{ID: "lock-new", Who: "other@host"}
		if cmd := p.HandleLockDetected(sdk.LockDetectedEvent{Lock: lock}); cmd != nil {
			t.Error("HandleLockDetected should return nil cmd")
		}
		if p.lockInfo == nil || p.lockInfo.ID != "lock-new" {
			t.Errorf("lockInfo = %v, want stored lock", p.lockInfo)
		}
	})

	t.Run("HandleLockCleared clears lock", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.lockInfo = &sdk.StateLock{ID: "old-lock"}
		if cmd := p.HandleLockCleared(sdk.LockClearedEvent{}); cmd != nil {
			t.Error("HandleLockCleared should return nil cmd")
		}
		if p.lockInfo != nil {
			t.Error("HandleLockCleared should clear lockInfo")
		}
	})
}

func TestUpdate_TimerTickAndUnhandled(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})

	// Running timer ticks again; idle tick is handled with no cmd.
	p.Update(ForceUnlockStartMsg{LockID: "x"}) // starts the timer
	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd == nil {
		t.Error("running timer tick should return a tick cmd")
	}

	if self, cmd := p.Update(struct{}{}); self.(*Plugin) != p || cmd != nil {
		t.Error("unhandled msg should return same plugin and nil cmd")
	}
}

func TestHandleContextChanged(t *testing.T) {
	t.Run("resets state", func(t *testing.T) {
		svc := &sdktest.MockService{}
		p, _ := newTestPlugin(svc)
		p.lockInfo = &sdk.StateLock{ID: "abc"}
		driveToTerminal(t, p, "abc") // reach a terminal state
		cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: &sdk.Context{Service: svc}})
		if cmd != nil {
			t.Error("HandleContextChanged should return nil cmd")
		}
		if p.CurrentStatus() != sdk.StatusIdle || p.lockID != "" || p.lockInfo != nil {
			t.Errorf("not reset: status=%v lockID=%q lockInfo=%v", p.CurrentStatus(), p.lockID, p.lockInfo)
		}
	})

	t.Run("nil Next is a no-op", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.lockID = "keep"
		if cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil}); cmd != nil {
			t.Error("nil Next should return nil cmd")
		}
		if p.lockID != "keep" {
			t.Errorf("lockID mutated on no-op: %q", p.lockID)
		}
	})
}
