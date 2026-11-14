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

	mu        sync.Mutex
	received  os.Signal
	finishing bool
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
	}
}

// Watch reads signals until the channel closes. A signal while an editor
// holds the terminal is dropped: it is the user's Ctrl+C for the editor. The
// first other signal is recorded, interrupts terraform when it has its own
// process group (otherwise the terminal already signalled it), and calls
// endRun. A later signal, or any signal once Finish has started, forces the
// exit.
func (g *interruptGuard) Watch(endRun func()) {
	go func() {
		for sig := range g.signals {
			if g.terminalHeld() {
				continue
			}
			if !g.recordFirst(sig) {
				g.forceExit(sig)
				return
			}
			if g.ownProcessGroup {
				g.runs.Interrupt()
			}
			endRun()
		}
	}()
}

func (g *interruptGuard) recordFirst(sig os.Signal) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.received != nil || g.finishing {
		return false
	}
	g.received = sig
	return true
}

func (g *interruptGuard) forceExit(sig os.Signal) {
	_, _ = io.WriteString(g.stderr, forcedExitMessage)
	g.shutdown()
	g.exit(interruptExitCode(sig))
}

// Finish runs once the program or headless driver returned. Without a prior
// signal it interrupts terraform; if terraform still runs, it says so on
// stderr and waits for it. It returns the first signal received, or nil.
func (g *interruptGuard) Finish() os.Signal {
	g.mu.Lock()
	g.finishing = true
	received := g.received
	g.mu.Unlock()

	if received == nil {
		g.runs.Interrupt()
	}
	if g.runs.Running() {
		_, _ = io.WriteString(g.stderr, waitingMessage)
		g.runs.Wait()
	}
	return received
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
// The returned stop restores the default handling; calls after the first do
// nothing.
func startInterruptGuard(svc sdk.Service, stderr io.Writer, shutdown func()) (*interruptGuard, func()) {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	guard := newInterruptGuard(terraformRunsOf(svc), signals, tfexec.TerraformHasOwnProcessGroup,
		editor.HoldsTerminal, stderr, shutdown, os.Exit)
	var once sync.Once
	return guard, func() {
		once.Do(func() {
			signal.Stop(signals)
			close(signals)
		})
	}
}
