package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TimerTickMsg is sent each second while a timer is running.
type TimerTickMsg struct{}

// Timer tracks elapsed time for long-running operations.
// Embed in a plugin and call Start/Stop to manage the timer.
type Timer struct {
	startTime time.Time
	elapsed   time.Duration
	running   bool
}

// Start begins tracking elapsed time and returns a tick command.
func (t *Timer) Start() tea.Cmd {
	t.startTime = time.Now()
	t.elapsed = 0
	t.running = true
	return t.tick()
}

// Stop halts the timer.
func (t *Timer) Stop() {
	if t.running {
		t.elapsed = time.Since(t.startTime)
		t.running = false
	}
}

// Tick updates elapsed time. Call from Update when TimerTickMsg is received.
// Returns the next tick command if still running, nil otherwise.
func (t *Timer) Tick() tea.Cmd {
	if !t.running {
		return nil
	}
	t.elapsed = time.Since(t.startTime)
	return t.tick()
}

// Elapsed returns the current elapsed duration.
func (t *Timer) Elapsed() time.Duration {
	if t.running {
		return time.Since(t.startTime)
	}
	return t.elapsed
}

// Running returns whether the timer is active.
func (t *Timer) Running() bool {
	return t.running
}

// FormatElapsed returns a human-readable elapsed string (e.g., "5s", "1m30s").
func (t *Timer) FormatElapsed() string {
	return FormatDuration(t.Elapsed())
}

func (t *Timer) tick() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return TimerTickMsg{}
	})
}

// FormatDuration formats a duration as a compact string.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
