package main

import (
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/lmarqs/terraform-ui/internal/editor"
	tfexec "github.com/lmarqs/terraform-ui/internal/terraform/exec"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const (
	forcedExitMessage = "terraform was stopped without finishing\n"
	waitingMessage    = "Waiting for terraform to stop… (Ctrl+C again to force)\n"
)

// terraformRuns is what the guard needs from exec.Runs.
type terraformRuns interface {
	Interrupt()
	Wait()
	Running() bool
}

// interruptGuard owns SIGINT and SIGTERM for one run: the first signal
// interrupts terraform and ends the run, a later one forces the exit.
type interruptGuard struct {
	runs            terraformRuns
	signals         <-chan os.Signal
	ownProcessGroup bool
	terminalHeld    func() bool
	stderr          io.Writer
	shutdown        func()
	exit            func(int)
	done            chan struct{}
	quit            chan struct{}
	watching        bool

	mu           sync.Mutex
	received     os.Signal
	finishing    bool
	finished     bool
	shutdownDone sync.Once
}

// newInterruptGuard builds a guard. runs may be a no-op tracker (macro runs);
// signals is the channel signal.Notify feeds; ownProcessGroup is
// exec.TerraformHasOwnProcessGroup; terminalHeld is editor.HoldsTerminal;
// shutdown is registry.Shutdown; exit is os.Exit in production.
func newInterruptGuard(runs terraformRuns, signals <-chan os.Signal, ownProcessGroup bool,
	terminalHeld func() bool, stderr io.Writer, shutdown func(), exit func(int)) *interruptGuard {
	return &interruptGuard{
		runs:            runs,
		signals:         signals,
		ownProcessGroup: ownProcessGroup,
		terminalHeld:    terminalHeld,
		stderr:          stderr,
		shutdown:        shutdown,
		exit:            exit,
		done:            make(chan struct{}),
		quit:            make(chan struct{}),
	}
}

// Watch reads signals until stop. A signal while an editor
// holds the terminal is dropped: it is the user's Ctrl+C for the editor. The
// first other signal is recorded, interrupts terraform when it has its own
// process group (otherwise the terminal already signalled it), and calls
// endRun. A later signal, or any signal while Finish runs, forces the exit.
// A signal after Finish returned, when terraform has finished, exits after
// plugin cleanup without the forced-exit note. Watch closes done when it
// stops reading.
func (g *interruptGuard) Watch(endRun func()) {
	g.watching = true
	go func() {
		defer close(g.done)
		for {
			select {
			case <-g.quit:
				return
			case sig := <-g.signals:
				if g.terminalHeld() {
					continue
				}
				switch g.record(sig) {
				case firstSignal:
					if g.ownProcessGroup {
						g.runs.Interrupt()
					}
					endRun()
				case signalWhileTerraformMayRun:
					_, _ = io.WriteString(g.stderr, forcedExitMessage)
					g.exitBy(sig)
					return
				case signalAfterTerraform:
					g.exitBy(sig)
					return
				}
			}
		}
	}()
}

type signalState int

const (
	firstSignal signalState = iota
	signalWhileTerraformMayRun
	signalAfterTerraform
)

func (g *interruptGuard) record(sig os.Signal) signalState {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch {
	case g.finished:
		return signalAfterTerraform
	case g.received != nil || g.finishing:
		return signalWhileTerraformMayRun
	}
	g.received = sig
	return firstSignal
}

func (g *interruptGuard) exitBy(sig os.Signal) {
	g.shutdownOnce()
	g.exit(interruptExitCode(sig))
}

// shutdownOnce runs plugin cleanup the first time it is called; a concurrent
// call waits for that run to end.
func (g *interruptGuard) shutdownOnce() {
	g.shutdownDone.Do(g.shutdown)
}

// stop runs plugin cleanup unless a signal already ran it, then ends the
// watcher and waits for it. The signals stay registered with the runtime and
// their channel stays open: Late reports one that arrives from here on, so the
// process exit code can still reflect it. A signal the watcher already took
// exits the process during that wait. Without a watcher, as after a panic
// before Watch, it does not wait. A stderr that is never drained can stall the
// wait, as it stalls the TUI.
func (g *interruptGuard) stop() {
	g.shutdownOnce()
	close(g.quit)
	if g.watching {
		<-g.done
	}
}

// Late returns a signal that arrived after stop ended the watcher, or nil.
func (g *interruptGuard) Late() os.Signal {
	select {
	case sig := <-g.signals:
		return sig
	default:
		return nil
	}
}

// Finish runs once the program or headless driver returned. Without a prior
// signal it interrupts terraform; if terraform still runs, it says so on
// stderr and waits for it. It returns the first signal received, or nil, and
// cutShort when the run ended without a signal while terraform still ran: a
// quit cancelled terraform, so the run must not report the plugin's result.
func (g *interruptGuard) Finish() (sig os.Signal, cutShort bool) {
	g.mu.Lock()
	g.finishing = true
	received := g.received
	g.mu.Unlock()

	if received == nil {
		g.runs.Interrupt()
	}
	running := g.runs.Running()
	if running {
		_, _ = io.WriteString(g.stderr, waitingMessage)
		g.runs.Wait()
	}

	g.mu.Lock()
	g.finished = true
	g.mu.Unlock()
	return received, received == nil && running
}

// interruptExitCode maps SIGINT to 130 and SIGTERM to 143.
func interruptExitCode(sig os.Signal) int {
	if sig == syscall.SIGTERM {
		return 143
	}
	return 130
}

// noTerraformRuns is the tracker of a run that starts no terraform process.
type noTerraformRuns struct{}

func (noTerraformRuns) Interrupt()    {}
func (noTerraformRuns) Wait()         {}
func (noTerraformRuns) Running() bool { return false }

func terraformRunsOf(svc sdk.Service) terraformRuns {
	if execSvc, ok := svc.(*tfexec.ExecService); ok {
		return execSvc.Runs()
	}
	return noTerraformRuns{}
}

// startInterruptGuard makes tfui the owner of SIGINT and SIGTERM for one run.
// The returned shutdown runs plugin cleanup unless a signal already ran it and
// ends the watcher; a run defers it. The signals are never given back: the
// process exits right after the run, and a signal handed back earlier would be
// dropped when it was inherited as ignored, so the exit code could miss it.
func startInterruptGuard(svc sdk.Service, stderr io.Writer, cleanup func()) (*interruptGuard, func()) {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	guard := newInterruptGuard(terraformRunsOf(svc), signals, tfexec.TerraformHasOwnProcessGroup,
		editor.HoldsTerminal, stderr, cleanup, os.Exit)
	return guard, guard.stop
}
