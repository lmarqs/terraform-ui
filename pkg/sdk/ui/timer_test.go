package ui

import (
	"testing"
	"time"
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
	var timer Timer
	timer.Start()
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
	timer.running = true

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
	var timer Timer
	timer.Start()
	cmd := timer.Tick()
	if cmd == nil {
		t.Fatal("Tick() should return next tick cmd when running")
	}
	msg := cmd()
	if _, ok := msg.(TimerTickMsg); !ok {
		t.Errorf("tick cmd should produce TimerTickMsg, got %T", msg)
	}
}
