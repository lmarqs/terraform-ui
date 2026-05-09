package apply

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui/styles"
)

// Status represents the current state of the apply extension.
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

// Extension implements the terraform apply feature.
type Extension struct {
	svc       terraform.Service
	status    Status
	errMsg    string
	targets   []string
	startTime time.Time
	elapsed   time.Duration
	confirmed bool
}

// New creates a new apply extension.
func New() *Extension {
	return &Extension{}
}

func (e *Extension) Name() string        { return "Apply" }
func (e *Extension) Description() string  { return "Apply terraform changes to infrastructure" }
func (e *Extension) KeyBinding() string   { return "a" }
func (e *Extension) Ready() bool          { return e.status == StatusSuccess }
func (e *Extension) Status() Status       { return e.status }
func (e *Extension) Elapsed() time.Duration { return e.elapsed }
func (e *Extension) IsConfirming() bool   { return e.status == StatusConfirming }

// SetTargets configures resource targets for apply.
func (e *Extension) SetTargets(targets []string) {
	e.targets = targets
}

// Init initializes the extension with a terraform service.
func (e *Extension) Init(svc terraform.Service) tea.Cmd {
	e.svc = svc
	return nil
}

// RequestApply transitions to the confirmation state.
func (e *Extension) RequestApply() {
	e.status = StatusConfirming
	e.confirmed = false
	e.errMsg = ""
}

// Confirm executes the apply after user confirmation.
func (e *Extension) Confirm() tea.Cmd {
	e.confirmed = true
	e.status = StatusRunning
	e.startTime = time.Now()
	e.errMsg = ""
	return tea.Batch(e.runApply(), e.tick())
}

// Cancel aborts the apply confirmation.
func (e *Extension) Cancel() {
	e.status = StatusIdle
	e.confirmed = false
}

func (e *Extension) runApply() tea.Cmd {
	svc := e.svc
	targets := e.targets
	start := e.startTime
	return func() tea.Msg {
		err := svc.Apply(context.Background(), targets)
		return ApplyResultMsg{Err: err, Duration: time.Since(start)}
	}
}

func (e *Extension) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update processes messages and returns the updated extension.
func (e *Extension) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case ApplyResultMsg:
		e.elapsed = msg.Duration
		if msg.Err != nil {
			e.status = StatusError
			e.errMsg = msg.Err.Error()
		} else {
			e.status = StatusSuccess
		}
		return nil, true

	case TickMsg:
		if e.status == StatusRunning {
			e.elapsed = time.Since(e.startTime)
			return e.tick(), true
		}
		return nil, true

	case tea.KeyMsg:
		return e.handleKey(msg), true
	}
	return nil, false
}

func (e *Extension) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch e.status {
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

// View renders the apply extension.
func (e *Extension) View(width, height int) string {
	title := styles.StyleTitle.Render("Apply")

	switch e.status {
	case StatusIdle:
		placeholder := styles.StyleFaintItalic.Render("Run plan first, then apply changes here.")
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

func (e *Extension) renderConfirmation(width, height int) string {
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
