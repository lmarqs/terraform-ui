package apply

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
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
	svc       terraform.Service
	status    Status
	errMsg    string
	targets   []string
	startTime time.Time
	elapsed   time.Duration
	confirmed bool
}

// New creates a new apply plugin.
func New(svc terraform.Service) plugin.Plugin {
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
func (e *Plugin) Init(ctx *plugin.Context) tea.Cmd {
	e.svc = ctx.Service
	return nil
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
func (e *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
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
	title := styles.StyleTitle.Render("Apply")

	switch e.status {
	case StatusIdle:
		placeholder := styles.StyleFaintItalic.Render("Run plan first, then apply changes here.\nPress Enter to start apply.")
		return styles.StylePadded.Render(title + "\n\n" + placeholder)

	case StatusConfirming:
		return e.renderConfirmation(width, height)

	case StatusRunning:
		elapsed := formatDuration(e.elapsed)
		running := styles.StyleFaintItalic.Render("Applying changes... " + elapsed)
		spinner := styles.StyleUpdate.Render(">>>")
		return styles.StylePadded.Render(title + "\n\n" + spinner + " " + running)

	case StatusSuccess:
		elapsed := formatDuration(e.elapsed)
		success := styles.StyleSuccess.Render("Apply complete! Resources are up-to-date.")
		duration := styles.StyleFaint.Render("Duration: " + elapsed)
		hint := styles.StyleFaintItalic.Render("Press Esc to go back")
		return styles.StylePadded.Render(title + "\n\n" + success + "\n" + duration + "\n\n" + hint)

	case StatusError:
		errText := styles.StyleError.Render("Apply failed: " + e.errMsg)
		hint := styles.StyleFaintItalic.Render("Press r to retry, Esc to go back")
		return styles.StylePadded.Render(title + "\n\n" + errText + "\n\n" + hint)

	default:
		return ""
	}
}

func (e *Plugin) renderConfirmation(width, height int) string {
	title := styles.StyleTitle.Render("Apply")

	warning := styles.StyleRiskHigh.Render("Are you sure you want to apply these changes?")
	detail := styles.StyleFaint.Render("This will modify your infrastructure.")

	if len(e.targets) > 0 {
		detail = styles.StyleFaint.Render(fmt.Sprintf("Targeting %d resource(s).", len(e.targets)))
	}

	prompt := styles.StyleKey.Render("[y]es") + " / " + styles.StyleFaint.Render("[n]o")

	content := title + "\n\n" + warning + "\n" + detail + "\n\n" + prompt
	return styles.StylePadded.Render(content)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
