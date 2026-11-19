package ui

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TimerTickMsg is sent each second while a timer is running.
type TimerTickMsg struct{}

// Timer tracks elapsed time for long-running operations.
// Embed in a plugin and call Start/Stop to manage the timer.
//
// A Timer owns at most one outstanding tick Cmd, so a restart or an unscoped
// TimerTickMsg meant for another timer cannot start a second tick chain.
type Timer struct {
	startTime time.Time
	elapsed   time.Duration
	// running and pending are read and written by tick Cmds, which run off the
	// Update loop. pending is true from the moment a tick Cmd is built until it
	// finishes.
	running atomic.Bool
	pending atomic.Bool
	// handoff makes Start's read of pending and a stopped tick's clearing of it
	// one step each: otherwise Start can leave the restart to a tick that is
	// ending, and the chain dies.
	handoff sync.Mutex
}

// Start begins tracking elapsed time and returns a tick command, or nil when a
// tick is still outstanding: it lands in Update and Tick continues the chain.
func (t *Timer) Start() tea.Cmd {
	t.startTime = time.Now()
	t.elapsed = 0
	t.handoff.Lock()
	t.running.Store(true)
	outstanding := t.pending.Load()
	t.handoff.Unlock()
	if outstanding {
		return nil
	}
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
// Returns the next tick command if still running, nil otherwise, and nil while
// this timer's own tick is outstanding: the message was a duplicate or meant for
// another timer.
func (t *Timer) Tick() tea.Cmd {
	if !t.running.Load() || t.pending.Load() {
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
// The caller must return the Cmd to the runtime: a Cmd that never runs leaves
// pending set and the timer without ticks.
func (t *Timer) tick() tea.Cmd {
	wait := tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return TimerTickMsg{}
	})
	t.pending.Store(true)
	return func() tea.Msg {
		t.handoff.Lock()
		if !t.running.Load() {
			t.pending.Store(false)
			t.handoff.Unlock()
			return nil
		}
		t.handoff.Unlock()
		msg := wait()
		t.pending.Store(false)
		return msg
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
