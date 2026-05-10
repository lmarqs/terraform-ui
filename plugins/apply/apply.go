package apply

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Status represents the current state of the apply plugin.
type Status int

const (
	StatusIdle Status = iota
	StatusConfirming
	StatusRunning
	StatusSuccess
	StatusError
)

// ApplyResultMsg is sent when apply completes.
type ApplyResultMsg struct {
	Err      error
	Duration time.Duration
}

// TickMsg is sent during apply for elapsed time tracking.
type TickMsg time.Time

// Plugin implements the terraform apply feature.
type Plugin struct {
	svc            sdk.Service
	session        *sdk.Session
	status         Status
	errMsg         string
	targets        []string
	startTime      time.Time
	elapsed        time.Duration
	confirmed      bool
	totalResources int
	scopedContext  string
}

// New creates a new apply plugin.
func New(svc sdk.Service) sdk.Plugin {
	return &Plugin{
		svc: svc,
	}
}

func (e *Plugin) ID() string          { return "apply" }
func (e *Plugin) Name() string        { return "Apply" }
func (e *Plugin) Description() string { return "Apply terraform changes to infrastructure" }
func (e *Plugin) KeyBinding() string  { return "a" }
func (e *Plugin) Ready() bool         { return e.status == StatusSuccess }
func (e *Plugin) Status() Status      { return e.status }
func (e *Plugin) Elapsed() time.Duration {
	return e.elapsed
}
func (e *Plugin) IsConfirming() bool { return e.status == StatusConfirming }

// Configure applies plugin-specific options from config.
func (e *Plugin) Configure(cfg map[string]interface{}) error {
	return nil
}

// SetTargets configures resource targets for apply.
func (e *Plugin) SetTargets(targets []string) {
	e.targets = targets
}

// Init initializes the plugin with shared context.
func (e *Plugin) Init(ctx *sdk.Context) tea.Cmd {
	e.svc = ctx.Service
	e.session = ctx.Session
	if e.session != nil {
		if summary, ok := sdk.GetTyped[*sdk.PlanSummary](e.session, sdk.SessionKeyPlanSummary); ok {
			e.totalResources = len(summary.Changes)
		}
	}
	return nil
}

// Activate scopes the service to the active context before apply operations.
func (e *Plugin) Activate() tea.Cmd {
	// Check if the active context changed since last activation
	if e.session != nil {
		currentContext, _ := sdk.GetTyped[string](e.session, sdk.SessionKeyActiveScopeAbs)
		if currentContext != e.scopedContext {
			// Context changed — reset status
			e.status = StatusIdle
			e.errMsg = ""
			e.scopedContext = currentContext
			if currentContext != "" {
				e.svc = e.svc.WithDir(currentContext)
			}
		}

		if e.scopedContext == "" {
			if count, ok := sdk.GetTyped[int](e.session, sdk.SessionKeyScopeCount); ok && count > 1 {
				e.status = StatusError
				e.errMsg = "Select a context first (press c)"
				return nil
			}
		}
	}
	return nil
}

// TotalResources returns the total resource count read from the session plan summary.
func (e *Plugin) TotalResources() int {
	// Re-read from session in case plan ran after Init.
	if e.session != nil {
		if summary, ok := sdk.GetTyped[*sdk.PlanSummary](e.session, sdk.SessionKeyPlanSummary); ok {
			e.totalResources = len(summary.Changes)
		}
	}
	return e.totalResources
}

// RequestApply transitions to the confirmation state.
func (e *Plugin) RequestApply() {
	e.status = StatusConfirming
	e.confirmed = false
	e.errMsg = ""
}

// Confirm executes the apply after user confirmation.
func (e *Plugin) Confirm() tea.Cmd {
	e.confirmed = true
	e.status = StatusRunning
	e.startTime = time.Now()
	e.errMsg = ""
	return tea.Batch(e.runApply(), e.tick())
}

// Cancel aborts the apply confirmation.
func (e *Plugin) Cancel() {
	e.status = StatusIdle
	e.confirmed = false
}

func (e *Plugin) runApply() tea.Cmd {
	svc := e.svc
	targets := e.targets
	start := e.startTime
	return func() tea.Msg {
		err := svc.Apply(context.Background(), targets)
		return ApplyResultMsg{Err: err, Duration: time.Since(start)}
	}
}

func (e *Plugin) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update processes messages and returns the updated plugin.
func (e *Plugin) Update(msg tea.Msg) (sdk.Plugin, tea.Cmd) {
	switch msg := msg.(type) {
	case ApplyResultMsg:
		e.elapsed = msg.Duration
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
		} else {
			e.status = StatusSuccess
		}
		return e, nil

	case TickMsg:
		if e.status == StatusRunning {
			e.elapsed = time.Since(e.startTime)
			return e, e.tick()
		}
		return e, nil

	case tea.KeyMsg:
		cmd := e.handleKey(msg)
		return e, cmd
	}
	return e, nil
}

func (e *Plugin) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch e.status {
	case StatusIdle:
		switch msg.String() {
		case "enter":
			e.RequestApply()
		}
	case StatusConfirming:
		switch msg.String() {
		case "y", "Y", "enter":
			return e.Confirm()
		case "n", "N", "esc":
			e.Cancel()
		}
	case StatusError:
		switch msg.String() {
		case "r":
			return e.Confirm()
		}
	}
	return nil
}

// View renders the apply plugin.
func (e *Plugin) View(width, height int) string {
	switch e.status {
	case StatusIdle:
		return sdk.StyleFaintItalic.Render("Run plan first, then apply changes here.\nPress Enter to start apply.")

	case StatusConfirming:
		return e.renderConfirmation(width, height)

	case StatusRunning:
		elapsed := formatDuration(e.elapsed)
		running := sdk.StyleFaintItalic.Render("Applying changes... " + elapsed)
		spinner := sdk.StyleUpdate.Render(">>>")
		return spinner + " " + running

	case StatusSuccess:
		elapsed := formatDuration(e.elapsed)
		success := sdk.StyleSuccess.Render("Apply complete! Resources are up-to-date.")
		duration := sdk.StyleFaint.Render("Duration: " + elapsed)
		hint := sdk.StyleFaintItalic.Render("Press q to go back")
		return success + "\n" + duration + "\n\n" + hint

	case StatusError:
		errText := sdk.StyleError.Render("Apply failed: " + e.errMsg)
		hint := sdk.StyleFaintItalic.Render("Press r to retry, q to go back")
		return errText + "\n\n" + hint

	default:
		return ""
	}
}

func (e *Plugin) renderConfirmation(width, height int) string {
	warning := sdk.StyleRiskHigh.Render("Are you sure you want to apply these changes?")
	detail := sdk.StyleFaint.Render("This will modify your infrastructure.")

	if len(e.targets) > 0 {
		detail = sdk.StyleFaint.Render(fmt.Sprintf("Targeting %d resource(s).", len(e.targets)))
	}

	prompt := sdk.StyleKey.Render("[y]es") + " / " + sdk.StyleFaint.Render("[n]o")

	return warning + "\n" + detail + "\n\n" + prompt
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
