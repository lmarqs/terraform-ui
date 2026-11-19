package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTimer_Start(t *testing.T) {
	var timer Timer
	cmd := timer.Start()
	if cmd == nil {
		t.Error("Start() returned nil cmd, want tick cmd")
	}
	if !timer.Running() {
		t.Error("Running() = false after Start, want true")
	}
}

func TestTimer_Stop(t *testing.T) {
	var timer Timer
	timer.Start()
	time.Sleep(10 * time.Millisecond)
	timer.Stop()

	if timer.Running() {
		t.Error("Running() = true after Stop, want false")
	}
	if timer.Elapsed() == 0 {
		t.Error("Elapsed() = 0 after Stop, want > 0")
	}
}

func TestTimer_Tick_WhenRunning(t *testing.T) {
	t.Parallel()
	var timer Timer
	landTick(t, timer.Start())
	cmd := timer.Tick()
	if cmd == nil {
		t.Error("Tick() returned nil when running, want next tick cmd")
	}
}

func TestTimer_Tick_WhenStopped(t *testing.T) {
	var timer Timer
	cmd := timer.Tick()
	if cmd != nil {
		t.Error("Tick() returned non-nil when stopped, want nil")
	}
}

func TestTimer_FormatElapsed(t *testing.T) {
	var timer Timer
	timer.startTime = time.Now().Add(-5 * time.Second)
	timer.running.Store(true)

	result := timer.FormatElapsed()
	if result != "5s" && result != "4s" {
		t.Errorf("FormatElapsed() = %q, want ~5s", result)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m0s"},
		{90 * time.Second, "1m30s"},
		{125 * time.Second, "2m5s"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.d)
		if got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestTimer_StopWhenNotRunning(t *testing.T) {
	var timer Timer
	timer.Stop()
	if timer.Elapsed() != 0 {
		t.Errorf("Elapsed() = %v after Stop on fresh timer, want 0", timer.Elapsed())
	}
}

func TestTimer_Tick_WhenRunning_ShouldReturnCmdThatProducesTimerTickMsg(t *testing.T) {
	t.Parallel()
	var timer Timer
	cmd := timer.Start()
	if cmd == nil {
		t.Fatal("Start() should return a tick cmd")
	}
	msg := cmd()
	if _, ok := msg.(TimerTickMsg); !ok {
		t.Errorf("tick cmd should produce TimerTickMsg, got %T", msg)
	}
}

func TestTimer_Tick_WhenCalledWhileRunning_ShouldReturnCmdThatProducesTimerTickMsg(t *testing.T) {
	t.Parallel()
	var timer Timer
	landTick(t, timer.Start())
	cmd := timer.Tick()
	if cmd == nil {
		t.Fatal("Tick() should return next tick cmd when running")
	}
	msg := cmd()
	if _, ok := msg.(TimerTickMsg); !ok {
		t.Errorf("tick cmd should produce TimerTickMsg, got %T", msg)
	}
}

// Drivers that run Cmds synchronously, such as the macro driver and headless
// sessions, run a tick only after the command it started has answered and
// stopped the timer. That tick must end at once instead of sleeping a second.
func TestTimer_Tick_WhenStoppedBeforeTheTickRuns_ShouldProduceNoMsgWithoutWaiting(t *testing.T) {
	var timer Timer
	cmd := timer.Start()
	timer.Stop()

	begin := time.Now()
	msg := cmd()

	if msg != nil {
		t.Errorf("tick cmd of a stopped timer produced %T, want nil", msg)
	}
	if waited := time.Since(begin); waited > 500*time.Millisecond {
		t.Errorf("tick cmd of a stopped timer waited %v, want it to return at once", waited)
	}
}

// landTick runs a tick Cmd to completion, as the runtime does, and checks it
// delivered a TimerTickMsg. It waits the real second.
func landTick(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("tick cmd is nil")
	}
	if _, ok := cmd().(TimerTickMsg); !ok {
		t.Fatal("tick cmd did not produce TimerTickMsg")
	}
}

// A tick that is still waiting is the timer's one chain: a restart must leave
// it to continue instead of starting a second chain beside it.
func TestTimer_Start_WhenStoppedAndRestartedWhileATickIsOutstanding_ShouldContinueTheOutstandingChain(t *testing.T) {
	t.Parallel()
	var timer Timer
	outstanding := timer.Start()
	timer.Stop()

	if restart := timer.Start(); restart != nil {
		t.Fatal("Start() returned a second tick cmd while one is outstanding, want nil")
	}
	if !timer.Running() {
		t.Fatal("Running() = false after the restart, want true")
	}

	if _, ok := outstanding().(TimerTickMsg); !ok {
		t.Fatal("outstanding tick cmd did not produce TimerTickMsg after the restart")
	}
	successor := timer.Tick()
	if successor == nil {
		t.Fatal("Tick() returned nil after the outstanding tick landed, want one successor")
	}
	if extra := timer.Tick(); extra != nil {
		t.Error("a second Tick() returned a cmd, want nil: the successor is already outstanding")
	}
}

// A tick that was already delivered no longer waits, so a restart before it
// reaches Update starts the chain that the delivered tick then joins.
func TestTimer_Start_WhenTheTickIsInFlight_ShouldStartTheChainAndTheLateTickJoinsIt(t *testing.T) {
	t.Parallel()
	var timer Timer
	inFlight := timer.Start()
	if _, ok := inFlight().(TimerTickMsg); !ok {
		t.Fatal("tick cmd did not produce TimerTickMsg")
	}
	timer.Stop()

	restart := timer.Start()
	if restart == nil {
		t.Fatal("Start() returned nil with no tick outstanding, want a tick cmd")
	}
	if late := timer.Tick(); late != nil {
		t.Error("Tick() for the in-flight tick returned a cmd, want nil: the restart's tick is outstanding")
	}
}

// A tick that belongs to another timer (the App routes unscoped ticks to the
// active plugin) must not start a second chain on a timer that already waits.
func TestTimer_Tick_WhenThisTimersTickIsOutstanding_ShouldReturnNil(t *testing.T) {
	var timer Timer
	timer.Start()

	if cmd := timer.Tick(); cmd != nil {
		t.Error("Tick() returned a cmd while this timer's tick is outstanding, want nil")
	}
}

func TestTimer_Start_WhenTheStoppedTickEndedAtOnce_ShouldReturnFreshTick(t *testing.T) {
	var timer Timer
	stopped := timer.Start()
	timer.Stop()
	if msg := stopped(); msg != nil {
		t.Fatalf("tick cmd of a stopped timer produced %T, want nil", msg)
	}

	if cmd := timer.Start(); cmd == nil {
		t.Error("Start() returned nil after the stopped tick ended, want a fresh tick cmd")
	}
}
