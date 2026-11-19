package ui

import (
	"fmt"
	"sync/atomic"
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
	running   atomic.Bool // read by tick Cmds, which run off the Update loop
}

// Start begins tracking elapsed time and returns a tick command.
func (t *Timer) Start() tea.Cmd {
	t.startTime = time.Now()
	t.elapsed = 0
	t.running.Store(true)
	return t.tick()
}

// Stop halts the timer.
func (t *Timer) Stop() {
	if t.running.Load() {
		t.elapsed = time.Since(t.startTime)
		t.running.Store(false)
	}
}

// Tick updates elapsed time. Call from Update when TimerTickMsg is received.
// Returns the next tick command if still running, nil otherwise.
func (t *Timer) Tick() tea.Cmd {
	if !t.running.Load() {
		return nil
	}
	t.elapsed = time.Since(t.startTime)
	return t.tick()
}

// Elapsed returns the current elapsed duration.
func (t *Timer) Elapsed() time.Duration {
	if t.running.Load() {
		return time.Since(t.startTime)
	}
	return t.elapsed
}

// Running returns whether the timer is active.
func (t *Timer) Running() bool {
	return t.running.Load()
}

// FormatElapsed returns a human-readable elapsed string (e.g., "5s", "1m30s").
func (t *Timer) FormatElapsed() string {
	return FormatDuration(t.Elapsed())
}

// tick waits a second, unless the timer already stopped: a synchronous driver
// runs it after the command it started has finished, and must not wait.
func (t *Timer) tick() tea.Cmd {
	wait := tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return TimerTickMsg{}
	})
	return func() tea.Msg {
		if !t.running.Load() {
			return nil
		}
		return wait()
	}
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
