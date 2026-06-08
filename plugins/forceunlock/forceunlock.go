package forceunlock

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// ForceUnlockStartMsg triggers execution after confirmation (or directly when
// --force skips the prompt).
type ForceUnlockStartMsg struct {
	LockID string
}

// Plugin implements the standalone force-unlock feature. Its prelude is bespoke
// (lock-aware idle view, manual-entry fallback, danger confirm); the
// run/result/cancel lifecycle is delegated to the embedded ActionRunner.
type Plugin struct {
	sdk.PluginBase
	sdk.ActionRunner
	lockID   string
	lockInfo *sdk.StateLock
}

// New creates a new force-unlock plugin.
func New(svc sdk.Service) sdk.Plugin {
	p := &Plugin{PluginBase: sdk.NewPluginBase("forceunlock", "Force Unlock", "Remove a stale state lock")}
	p.Svc = svc
	return p
}

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Init(deps *sdk.PluginDeps) tea.Cmd {
	p.InitBase(deps)
	p.InitRunner(p.Log)
	return nil
}

// Activate resolves the lock ID from the input, the detected lock, or a manual
// prompt, then confirms (unless --force skips it).
func (p *Plugin) Activate(input Input) tea.Cmd {
	if p.Busy() {
		return nil
	}
	p.Reset()
	if input.LockID != "" {
		if input.Force {
			return func() tea.Msg { return ForceUnlockStartMsg{LockID: input.LockID} }
		}
		return p.confirmUnlock(input.LockID)
	}
	if p.lockInfo != nil {
		return p.confirmUnlock(p.lockInfo.ID)
	}
	return p.offerManualEntry()
}

func (p *Plugin) offerManualEntry() tea.Cmd {
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				"No active lock detected. Enter lock ID manually?",
				func() tea.Cmd { return p.requestLockIDInput() },
			),
		}
	}
}

func (p *Plugin) requestLockIDInput() tea.Cmd {
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputText("Lock ID:", "", func(lockID string) tea.Cmd {
				if lockID == "" {
					return func() tea.Msg { return sdk.DeactivateMsg{} }
				}
				return p.confirmUnlock(lockID)
			}),
		}
	}
}

func (p *Plugin) confirmUnlock(lockID string) tea.Cmd {
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Force-unlock %s? This is dangerous if another operation is running.", lockID),
				func() tea.Cmd {
					return func() tea.Msg { return ForceUnlockStartMsg{LockID: lockID} }
				},
			),
		}
	}
}

// start arms the runner with the unlock spec for lockID and begins execution.
func (p *Plugin) start(lockID string) tea.Cmd {
	p.lockID = lockID
	p.Arm(p.spec(lockID))
	return p.Start()
}

func (p *Plugin) spec(lockID string) sdk.ActionSpec {
	svc := p.Svc
	return sdk.ActionSpec{
		Name: "forceunlock",
		Run: func(ctx context.Context) ([]string, error) {
			if err := svc.ForceUnlock(ctx, lockID); err != nil {
				return nil, err
			}
			return []string{lockID}, nil
		},
		OnSuccess: []tea.Msg{sdk.LockClearedEvent{}, sdk.PlanInvalidatedEvent{}},
	}
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	if handled, cmd := p.ActionRunner.Update(msg); handled {
		if p.Ready() {
			p.lockInfo = nil
		}
		return p, cmd
	}
	switch msg := msg.(type) {
	case ForceUnlockStartMsg:
		return p, p.start(msg.LockID)
	case tea.KeyMsg:
		return p, p.handleKey(msg)
	}
	return p, nil
}

func (p *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q", "esc":
		return func() tea.Msg { return sdk.DeactivateMsg{} }
	case "ctrl+r":
		if p.CurrentStatus() == sdk.StatusError {
			return p.Activate(Input{})
		}
	}
	return nil
}

func (p *Plugin) View(_, _ int) string {
	switch p.CurrentStatus() {
	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render(fmt.Sprintf("Force-unlocking %s... %s", p.lockID, p.Elapsed()))
	case sdk.StatusDone:
		return sdk.StyleSuccess.Render(fmt.Sprintf("Lock %s released successfully", p.lockID))
	case sdk.StatusError:
		return sdk.StyleError.Render("Error: " + p.ErrMessage())
	}
	// Idle: show the detected lock (if any) or the no-lock placeholder.
	if p.lockInfo != nil {
		return sdk.FormatLockInfo(p.lockInfo)
	}
	return sdk.StyleFaintItalic.Render("No active lock detected.")
}

func (p *Plugin) Hints() []sdk.KeyHint {
	if p.CurrentStatus() == sdk.StatusError {
		return (sdk.HintSetRetry | sdk.HintSetBack | sdk.HintSetQuit).Hints()
	}
	return (sdk.HintSetBack | sdk.HintSetQuit).Hints()
}

// HandleLockDetected implements sdk.LockDetectedHandler.
func (p *Plugin) HandleLockDetected(evt sdk.LockDetectedEvent) tea.Cmd {
	p.lockInfo = evt.Lock
	return nil
}

// HandleLockCleared implements sdk.LockClearedHandler.
func (p *Plugin) HandleLockCleared(_ sdk.LockClearedEvent) tea.Cmd {
	p.lockInfo = nil
	return nil
}

// HandleContextChanged implements sdk.ContextChangedHandler.
func (p *Plugin) HandleContextChanged(ev sdk.ContextChangedEvent) tea.Cmd {
	if !p.HandleContextChangedDefault(ev) {
		return nil
	}
	p.Reset()
	p.lockInfo = nil
	p.lockID = ""
	return nil
}
