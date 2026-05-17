package macro

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const (
	ExitOK          = 0
	ExitAssertFail  = 1
	ExitSyntaxError = 2
	ExitTimeout     = 3
)

type RunError struct {
	Code    int
	Line    int
	Message string
}

func (e *RunError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}
	return e.Message
}

type Runner struct {
	driver   *Driver
	timeout  time.Duration
	recorder *Recorder
}

func NewRunner(driver *Driver) *Runner {
	return &Runner{
		driver:  driver,
		timeout: 5 * time.Second,
	}
}

// WithRecorder enables frame recording during execution.
func (r *Runner) WithRecorder(rec *Recorder) *Runner {
	r.recorder = rec
	return r
}

func (r *Runner) Execute(commands []Command) error {
	r.driver.Init()
	r.captureFrame()

	var execErr error
	for _, cmd := range commands {
		if err := r.executeOne(cmd); err != nil {
			execErr = err
			break
		}
		r.captureFrame()
	}

	if r.recorder != nil {
		_ = r.recorder.Finalize()
	}

	if execErr != nil {
		return execErr
	}
	return nil
}

func (r *Runner) captureFrame() {
	if r.recorder == nil {
		return
	}
	r.recorder.CaptureView(r.driver.View())
}

func (r *Runner) executeOne(cmd Command) error {
	switch cmd.Type {
	case CmdKey:
		r.driver.SendKey(cmd.Args[0])

	case CmdWaitReady:
		err := r.driver.WaitUntil(func(v string) bool {
			return v != "" && !strings.Contains(v, "Loading")
		}, r.timeout)
		if err != nil {
			return &RunError{Code: ExitTimeout, Line: cmd.Line, Message: "timeout waiting for ready"}
		}

	case CmdWaitView:
		substr := cmd.Args[0]
		err := r.driver.WaitUntil(func(v string) bool {
			return strings.Contains(v, substr)
		}, r.timeout)
		if err != nil {
			return &RunError{Code: ExitTimeout, Line: cmd.Line,
				Message: fmt.Sprintf("timeout waiting for view to contain %q", substr)}
		}

	case CmdAssertView:
		substr := cmd.Args[0]
		if !r.driver.ViewContains(substr) {
			return &RunError{Code: ExitAssertFail, Line: cmd.Line,
				Message: fmt.Sprintf("assertion failed: view does not contain %q", substr)}
		}

	case CmdScreenshot:
		path := cmd.Args[0]
		if err := os.WriteFile(path, []byte(r.driver.View()), 0644); err != nil {
			return &RunError{Code: ExitAssertFail, Line: cmd.Line,
				Message: fmt.Sprintf("screenshot write failed: %v", err)}
		}

	case CmdResize:
		w, _ := strconv.Atoi(cmd.Args[0])
		h, _ := strconv.Atoi(cmd.Args[1])
		r.driver.SendMsg(tea.WindowSizeMsg{Width: w, Height: h})

	case CmdSleep:
		dur, _ := time.ParseDuration(cmd.Args[0])
		time.Sleep(dur)

	case CmdEmit:
		r.driver.SendMsg(buildEmitMsg(cmd.Args))
	}

	return nil
}

func buildEmitMsg(args []string) tea.Msg {
	switch args[0] {
	case "stale":
		return sdk.PlanInvalidatedEvent{}
	case "refreshed":
		return sdk.StateRefreshedEvent{}
	case "lock":
		who := strings.Join(args[1:], " ")
		return sdk.LockDetectedEvent{Lock: &sdk.StateLock{Who: who, Created: time.Now()}}
	case "lock-clear":
		return sdk.LockClearedEvent{}
	default:
		return nil
	}
}
