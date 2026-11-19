package main

import (
	"bytes"
	"os"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

const (
	forcedExitNote = "terraform was stopped without finishing\n"
	waitingNotice  = "Waiting for terraform to stop… (Ctrl+C again to force)\n"
	interruptWait  = 2 * time.Second
)

type interruptHarness struct {
	mu     sync.Mutex
	events []string
	stderr bytes.Buffer

	running      bool
	terminalHeld atomic.Bool
	heldChecks   chan struct{}
	heldGate     chan struct{}

	shutdownBlock   chan struct{}
	shutdownStarted chan struct{}
	waitBlock       chan struct{}
	releaseOnce     sync.Once
	waitStarted     chan struct{}
	waitOnce        sync.Once
	endRunCalls     chan struct{}
	endRunBlock     chan struct{}
	exitCalls       chan int
	signals         chan os.Signal
	guard           *interruptGuard
}

func (h *interruptHarness) record(event string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
}

func (h *interruptHarness) snapshot() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.events...)
}

func (h *interruptHarness) Interrupt() { h.record("interrupt") }

func (h *interruptHarness) Running() bool { return h.running }

func (h *interruptHarness) Wait() {
	h.record("wait")
	h.waitOnce.Do(func() { close(h.waitStarted) })
	if h.waitBlock != nil {
		<-h.waitBlock
	}
}

func (h *interruptHarness) releaseWait() {
	h.releaseOnce.Do(func() { close(h.waitBlock) })
}

func (h *interruptHarness) Write(p []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, "stderr:"+string(p))
	return h.stderr.Write(p)
}

func (h *interruptHarness) stderrText() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stderr.String()
}

func (h *interruptHarness) endRun() {
	h.record("endRun")
	h.endRunCalls <- struct{}{}
	if h.endRunBlock != nil {
		<-h.endRunBlock
	}
}

// held reports terminalHeld and, while it is set, tells the test that a
// signal was checked against it. With heldGate set, it tells the test and
// holds the signal until the gate closes, then reports the terminal free.
func (h *interruptHarness) held() bool {
	if h.heldGate != nil {
		h.heldChecks <- struct{}{}
		<-h.heldGate
		return false
	}
	if !h.terminalHeld.Load() {
		return false
	}
	h.heldChecks <- struct{}{}
	return true
}

func (h *interruptHarness) shutdown() {
	h.record("shutdown")
	if h.shutdownBlock != nil {
		close(h.shutdownStarted)
		<-h.shutdownBlock
	}
}

func (h *interruptHarness) exit(code int) {
	h.record("exit:" + strconv.Itoa(code))
	h.exitCalls <- code
}

func newInterruptHarness(t *testing.T, ownProcessGroup, running, blockWait bool) *interruptHarness {
	t.Helper()
	h := &interruptHarness{
		running:     running,
		waitStarted: make(chan struct{}),
		endRunCalls: make(chan struct{}, 4),
		exitCalls:   make(chan int, 4),
		heldChecks:  make(chan struct{}, 4),
		signals:     make(chan os.Signal, 4),
	}
	if blockWait {
		h.waitBlock = make(chan struct{})
		t.Cleanup(h.releaseWait)
	}
	h.guard = newInterruptGuard(h, h.signals, ownProcessGroup, h.held, h, h.shutdown, h.exit)
	h.guard.Watch(h.endRun)
	return h
}

func awaitInterrupt[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(interruptWait):
		t.Fatalf("timed out waiting for %s", what)
		var zero T
		return zero
	}
}

func (h *interruptHarness) firstSignal(t *testing.T, sig os.Signal) {
	t.Helper()
	h.signals <- sig
	awaitInterrupt(t, h.endRunCalls, "endRun after first signal")
}

func TestInterruptGuard_FirstSignal_ShouldInterruptOnlyWithOwnProcessGroupAndEndRun(t *testing.T) {
	tests := []struct {
		name            string
		sig             os.Signal
		ownProcessGroup bool
		want            []string
	}{
		{"SIGINT with own process group", syscall.SIGINT, true, []string{"interrupt", "endRun"}},
		{"SIGINT without own process group", syscall.SIGINT, false, []string{"endRun"}},
		{"SIGTERM with own process group", syscall.SIGTERM, true, []string{"interrupt", "endRun"}},
		{"SIGTERM without own process group", syscall.SIGTERM, false, []string{"endRun"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInterruptHarness(t, tt.ownProcessGroup, true, false)

			h.firstSignal(t, tt.sig)

			if got := h.snapshot(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("events = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInterruptGuard_SignalWhileTerminalHeldByEditor_ShouldHaveNoEffect(t *testing.T) {
	tests := []struct {
		name string
		sig  os.Signal
	}{
		{"SIGINT", syscall.SIGINT},
		{"SIGTERM", syscall.SIGTERM},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInterruptHarness(t, true, true, false)
			h.terminalHeld.Store(true)
			h.signals <- tt.sig
			h.signals <- tt.sig
			awaitInterrupt(t, h.heldChecks, "first signal checked while held")
			awaitInterrupt(t, h.heldChecks, "second signal checked while held")
			h.terminalHeld.Store(false)

			h.firstSignal(t, tt.sig)

			want := []string{"interrupt", "endRun"}
			if got := h.snapshot(); !reflect.DeepEqual(got, want) {
				t.Errorf("events = %v, want %v", got, want)
			}
		})
	}
}

func TestInterruptGuard_SecondSignal_ShouldNoteShutdownThenExit(t *testing.T) {
	tests := []struct {
		name     string
		first    os.Signal
		sig      os.Signal
		wantCode int
		wantExit string
	}{
		{"SIGINT exits 130", syscall.SIGINT, syscall.SIGINT, 130, "exit:130"},
		{"SIGTERM exits 143", syscall.SIGTERM, syscall.SIGTERM, 143, "exit:143"},
		{"SIGINT then SIGTERM exits 143", syscall.SIGINT, syscall.SIGTERM, 143, "exit:143"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInterruptHarness(t, true, true, false)
			h.firstSignal(t, tt.first)

			h.signals <- tt.sig

			if got := awaitInterrupt(t, h.exitCalls, "forced exit"); got != tt.wantCode {
				t.Errorf("exit code = %d, want %d", got, tt.wantCode)
			}
			want := []string{"interrupt", "endRun", "stderr:" + forcedExitNote, "shutdown", tt.wantExit}
			if got := h.snapshot(); !reflect.DeepEqual(got, want) {
				t.Errorf("events = %v, want %v", got, want)
			}
		})
	}
}

func TestInterruptGuard_SignalWhileFinishWaits_ShouldForceExit(t *testing.T) {
	tests := []struct {
		name     string
		sig      os.Signal
		wantCode int
		wantExit string
	}{
		{"SIGINT exits 130", syscall.SIGINT, 130, "exit:130"},
		{"SIGTERM exits 143", syscall.SIGTERM, 143, "exit:143"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInterruptHarness(t, true, true, true)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				h.guard.Finish()
			}()
			awaitInterrupt(t, h.waitStarted, "Wait to start")

			h.signals <- tt.sig

			if got := awaitInterrupt(t, h.exitCalls, "forced exit"); got != tt.wantCode {
				t.Errorf("exit code = %d, want %d", got, tt.wantCode)
			}
			want := []string{
				"interrupt", "stderr:" + waitingNotice, "wait",
				"stderr:" + forcedExitNote, "shutdown", tt.wantExit,
			}
			if got := h.snapshot(); !reflect.DeepEqual(got, want) {
				t.Errorf("events = %v, want %v", got, want)
			}
			h.releaseWait()
			awaitInterrupt(t, finished, "Finish to return")
		})
	}
}

func TestInterruptGuard_Finish_WithoutSignal_ShouldInterruptAndWaitOnlyWhenRunning(t *testing.T) {
	tests := []struct {
		name         string
		running      bool
		want         []string
		wantStderr   string
		wantCutShort bool
	}{
		{"nothing running", false, []string{"interrupt"}, "", false},
		{
			"terraform running",
			true,
			[]string{"interrupt", "stderr:" + waitingNotice, "wait"},
			waitingNotice,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInterruptHarness(t, true, tt.running, false)

			got, cutShort := h.guard.Finish()

			if got != nil {
				t.Errorf("Finish() signal = %v, want nil", got)
			}
			if cutShort != tt.wantCutShort {
				t.Errorf("Finish() cutShort = %v, want %v", cutShort, tt.wantCutShort)
			}
			if events := h.snapshot(); !reflect.DeepEqual(events, tt.want) {
				t.Errorf("events = %v, want %v", events, tt.want)
			}
			if gotErr := h.stderrText(); gotErr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", gotErr, tt.wantStderr)
			}
		})
	}
}

func TestInterruptGuard_Finish_AfterSignal_ShouldReturnSignalWithoutSecondInterrupt(t *testing.T) {
	tests := []struct {
		name    string
		sig     os.Signal
		running bool
		want    []string
	}{
		{"SIGINT, nothing running", syscall.SIGINT, false, []string{"interrupt", "endRun"}},
		{"SIGTERM, nothing running", syscall.SIGTERM, false, []string{"interrupt", "endRun"}},
		{
			"SIGINT, terraform running",
			syscall.SIGINT,
			true,
			[]string{"interrupt", "endRun", "stderr:" + waitingNotice, "wait"},
		},
		{
			"SIGTERM, terraform running",
			syscall.SIGTERM,
			true,
			[]string{"interrupt", "endRun", "stderr:" + waitingNotice, "wait"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInterruptHarness(t, true, tt.running, false)
			h.firstSignal(t, tt.sig)

			got, cutShort := h.guard.Finish()

			if got != tt.sig {
				t.Errorf("Finish() signal = %v, want %v", got, tt.sig)
			}
			if cutShort {
				t.Errorf("Finish() cutShort = true, want false after a signal")
			}
			if events := h.snapshot(); !reflect.DeepEqual(events, tt.want) {
				t.Errorf("events = %v, want %v", events, tt.want)
			}
		})
	}
}

func TestInterruptExitCode_ShouldMapSignalToConventionalCode(t *testing.T) {
	tests := []struct {
		sig  os.Signal
		want int
	}{
		{syscall.SIGINT, 130},
		{os.Interrupt, 130},
		{syscall.SIGTERM, 143},
	}
	for _, tt := range tests {
		t.Run(tt.sig.String(), func(t *testing.T) {
			if got := interruptExitCode(tt.sig); got != tt.want {
				t.Errorf("interruptExitCode(%v) = %d, want %d", tt.sig, got, tt.want)
			}
		})
	}
}

func TestInterruptGuard_SignalAfterFinish_ShouldShutdownThenExitWithoutNote(t *testing.T) {
	tests := []struct {
		name     string
		first    os.Signal
		sig      os.Signal
		wantCode int
		want     []string
	}{
		{"SIGINT, no prior signal", nil, syscall.SIGINT, 130, []string{"interrupt", "shutdown", "exit:130"}},
		{"SIGTERM, no prior signal", nil, syscall.SIGTERM, 143, []string{"interrupt", "shutdown", "exit:143"}},
		{
			"SIGTERM after a first SIGINT",
			syscall.SIGINT, syscall.SIGTERM, 143,
			[]string{"interrupt", "endRun", "shutdown", "exit:143"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInterruptHarness(t, true, false, false)
			if tt.first != nil {
				h.firstSignal(t, tt.first)
			}
			h.guard.Finish()

			h.signals <- tt.sig

			if got := awaitInterrupt(t, h.exitCalls, "exit"); got != tt.wantCode {
				t.Errorf("exit code = %d, want %d", got, tt.wantCode)
			}
			if got := h.snapshot(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("events = %v, want %v", got, tt.want)
			}
			if got := h.stderrText(); got != "" {
				t.Errorf("stderr = %q, want empty", got)
			}
		})
	}
}

func TestInterruptGuard_SignalDuringShutdown_ShouldExitAfterShutdownWithoutRunningItAgain(t *testing.T) {
	h := newInterruptHarness(t, true, false, false)
	h.shutdownBlock = make(chan struct{})
	h.shutdownStarted = make(chan struct{})
	h.guard.Finish()
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		h.guard.shutdownOnce()
	}()
	awaitInterrupt(t, h.shutdownStarted, "shutdown to start")

	h.signals <- syscall.SIGINT
	select {
	case code := <-h.exitCalls:
		t.Fatalf("exit(%d) before shutdown returned", code)
	case <-time.After(50 * time.Millisecond):
	}
	close(h.shutdownBlock)

	if got := awaitInterrupt(t, h.exitCalls, "exit"); got != 130 {
		t.Errorf("exit code = %d, want 130", got)
	}
	awaitInterrupt(t, shutdownDone, "shutdown to return")
	want := []string{"interrupt", "shutdown", "exit:130"}
	if got := h.snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}

func TestInterruptGuard_ShutdownAfterForcedExit_ShouldNotRunShutdownAgain(t *testing.T) {
	h := newInterruptHarness(t, true, true, false)
	h.firstSignal(t, syscall.SIGINT)
	h.signals <- syscall.SIGINT
	awaitInterrupt(t, h.exitCalls, "forced exit")

	h.guard.shutdownOnce()

	want := []string{"interrupt", "endRun", "stderr:" + forcedExitNote, "shutdown", "exit:130"}
	if got := h.snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}

func TestInterruptGuard_SignalPendingWhenStopRuns_ShouldExitBeforeStopReturns(t *testing.T) {
	tests := []struct {
		name     string
		sig      os.Signal
		wantCode int
		wantExit string
	}{
		{"SIGINT exits 130", syscall.SIGINT, 130, "exit:130"},
		{"SIGTERM exits 143", syscall.SIGTERM, 143, "exit:143"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newInterruptHarness(t, true, false, false)
			h.heldGate = make(chan struct{})
			h.guard.Finish()
			h.signals <- tt.sig
			awaitInterrupt(t, h.heldChecks, "watcher to take the signal")

			stopped := make(chan struct{})
			go func() {
				defer close(stopped)
				h.guard.stop(func() { close(h.signals) })
				h.record("stopped")
			}()
			select {
			case <-stopped:
				t.Fatal("stop returned before the watcher handled the pending signal")
			case <-time.After(50 * time.Millisecond):
			}
			close(h.heldGate)

			awaitInterrupt(t, stopped, "stop to return")
			want := []string{"interrupt", "shutdown", tt.wantExit, "stopped"}
			if got := h.snapshot(); !reflect.DeepEqual(got, want) {
				t.Errorf("events = %v, want %v", got, want)
			}
		})
	}
}

func TestInterruptGuard_SecondSignalWhileEndRunBlocks_ShouldForceExit(t *testing.T) {
	h := newInterruptHarness(t, true, true, false)
	h.endRunBlock = make(chan struct{})
	t.Cleanup(func() { close(h.endRunBlock) })
	h.firstSignal(t, syscall.SIGINT)

	h.signals <- syscall.SIGTERM

	if got := awaitInterrupt(t, h.exitCalls, "forced exit while endRun blocks"); got != 143 {
		t.Errorf("exit code = %d, want 143", got)
	}
	want := []string{"interrupt", "endRun", "stderr:" + forcedExitNote, "shutdown", "exit:143"}
	if got := h.snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}
