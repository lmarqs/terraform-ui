package forceunlock

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// ForceUnlockStartMsg triggers the loading state and starts the unlock operation.
type ForceUnlockStartMsg struct {
	LockID string
}

// ForceUnlockResultMsg is sent when the force-unlock operation completes.
type ForceUnlockResultMsg struct {
	LockID string
	Err    error
}

// Plugin implements the standalone force-unlock feature.
type Plugin struct {
	svc      sdk.Service
	log      *slog.Logger
	timer    ui.Timer
	status   sdk.Status
	lockID   string
	lockInfo *sdk.StateLock
	errMsg   string
	cancelFn context.CancelFunc
}

// New creates a new force-unlock plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc: svc,
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func (p *Plugin) ID() string          { return "forceunlock" }
func (p *Plugin) Name() string        { return "Force Unlock" }
func (p *Plugin) Description() string { return "Remove a stale state lock" }
func (p *Plugin) Ready() bool         { return p.status == sdk.StatusDone }

func (p *Plugin) Configure(_ map[string]interface{}) error { return nil }

func (p *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	p.svc = ctx.Service
	p.log = ctx.Logger
	return nil
}

func (p *Plugin) Activate() tea.Cmd {
	if p.status == sdk.StatusLoading {
		return nil
	}
	p.status = sdk.StatusIdle
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
				func() tea.Cmd {
					return p.requestLockIDInput()
				},
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

// Cancel aborts any in-flight terraform operation.
func (p *Plugin) Cancel() {
	if p.cancelFn != nil {
		p.cancelFn()
		p.cancelFn = nil
	}
}

func (p *Plugin) executeUnlock(lockID string) tea.Cmd {
	p.Cancel()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFn = cancel
	p.lockID = lockID
	p.status = sdk.StatusLoading
	svc := p.svc
	log := p.log
	return tea.Batch(func() tea.Msg {
		err := svc.ForceUnlock(ctx, lockID)
		if err != nil {
			log.Debug("forceunlock.error", "lockID", lockID, "error", err.Error())
		} else {
			log.Debug("forceunlock.success", "lockID", lockID)
		}
		return ForceUnlockResultMsg{LockID: lockID, Err: err}
	}, p.timer.Start())
}

func (p *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.TimerTickMsg:
		return p, p.timer.Tick()

	case ForceUnlockStartMsg:
		return p, p.executeUnlock(msg.LockID)

	case ForceUnlockResultMsg:
		p.timer.Stop()
		if msg.Err != nil {
			p.status = sdk.StatusError
			p.errMsg = fmt.Sprintf("Force-unlock failed: %s", msg.Err.Error())
		} else {
			p.status = sdk.StatusDone
			p.lockInfo = nil
			return p, tea.Batch(
				func() tea.Msg { return sdk.LockClearedEvent{} },
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
	switch msg.String() {
	case "q", "esc":
		return func() tea.Msg { return sdk.DeactivateMsg{} }
	case "ctrl+r":
		if p.status == sdk.StatusError {
			return p.Activate()
		}
	}
	return nil
}

func (p *Plugin) View(_, _ int) string {
	switch p.status {
	case sdk.StatusIdle:
		if p.lockInfo != nil {
			return sdk.FormatLockInfo(p.lockInfo)
		}
		return sdk.StyleFaintItalic.Render("No active lock detected.")
	case sdk.StatusLoading:
		return sdk.StyleFaintItalic.Render(fmt.Sprintf("Force-unlocking %s... %s", p.lockID, p.timer.FormatElapsed()))
	case sdk.StatusDone:
		return sdk.StyleSuccess.Render(fmt.Sprintf("Lock %s released successfully", p.lockID))
	case sdk.StatusError:
		return sdk.StyleError.Render("Error: " + p.errMsg)
	default:
		return ""
	}
}

func (p *Plugin) Hints() []sdk.KeyHint {
	switch p.status {
	case sdk.StatusError:
		return (sdk.HintSetRetry | sdk.HintSetBack | sdk.HintSetQuit).Hints()
	case sdk.StatusDone:
		return (sdk.HintSetBack | sdk.HintSetQuit).Hints()
	default:
		return (sdk.HintSetBack | sdk.HintSetQuit).Hints()
	}
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

// HandleChdirChanged implements sdk.ChdirHandler.
func (p *Plugin) HandleChdirChanged(evt sdk.ChdirChangedEvent) tea.Cmd {
	p.svc = p.svc.WithDir(evt.AbsPath)
	p.status = sdk.StatusIdle
	p.lockInfo = nil
	p.lockID = ""
	p.errMsg = ""
	return nil
}
