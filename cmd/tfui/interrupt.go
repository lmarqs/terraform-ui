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
	}
}

// Watch reads signals until the channel closes. A signal while an editor
// holds the terminal is dropped: it is the user's Ctrl+C for the editor. The
// first other signal is recorded, interrupts terraform when it has its own
// process group (otherwise the terminal already signalled it), and starts
// endRun without waiting for it: program.Quit blocks while the program runs an
// editor, and a later signal must still force the exit. A later signal, or any
// signal while Finish runs, forces the exit. A signal after Finish returned,
// when terraform has finished, exits after plugin cleanup without the
// forced-exit note. Watch closes done when it stops reading.
func (g *interruptGuard) Watch(endRun func()) {
	g.watching = true
	go func() {
		defer close(g.done)
		for sig := range g.signals {
			if g.terminalHeld() {
				continue
			}
			switch g.record(sig) {
			case firstSignal:
				if g.ownProcessGroup {
					g.runs.Interrupt()
				}
				go endRun()
			case signalWhileTerraformMayRun:
				_, _ = io.WriteString(g.stderr, forcedExitMessage)
				g.exitBy(sig)
				return
			case signalAfterTerraform:
				g.exitBy(sig)
				return
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

// stop runs plugin cleanup unless a signal already ran it, then calls release
// to give the signals back and close their channel, and waits for the watcher.
// A signal that arrived before release is still handled: its watcher exits the
// process during that wait, so the run cannot end with its own exit code.
// Without a watcher, as after a panic before Watch, it does not wait.
func (g *interruptGuard) stop(release func()) {
	g.shutdownOnce()
	release()
	if g.watching {
		<-g.done
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
// The returned shutdown runs plugin cleanup unless a signal already ran it,
// then gives the signals back their default action; a run defers it so tfui
// owns the signals until cleanup has run.
func startInterruptGuard(svc sdk.Service, stderr io.Writer, cleanup func()) (*interruptGuard, func()) {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	guard := newInterruptGuard(terraformRunsOf(svc), signals, tfexec.TerraformHasOwnProcessGroup,
		editor.HoldsTerminal, stderr, cleanup, os.Exit)
	return guard, func() {
		guard.stop(func() {
			signal.Stop(signals)
			close(signals)
		})
	}
}
