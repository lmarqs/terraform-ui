package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/plugins/apply"
	"github.com/lmarqs/terraform-ui/plugins/taint"
)

// TestRunPlugin_WhenApplyAutoApproveCI_ShouldTerminateWithoutKeystrokes is
// the per-plugin CI-termination invariant guard for apply. Hangs in CI are a
// per-plugin contract violation; this test catches them.
//
// `--ci` selects the macro driver (silent stderr); `--auto-approve` drives
// apply's lifecycle past the only prompt; `--macro` swaps the backend so we
// don't try to invoke a real terraform binary.
func TestRunPlugin_WhenApplyAutoApproveCI_ShouldTerminateWithoutKeystrokes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0644); err != nil {
		t.Fatalf("write tf: %v", err)
	}
	session := &Session{
		cfg:          config.Config{Dir: dir},
		ciMode:       true,
		silentStderr: true,
		macro:        MacroSpec{TapeURI: writeTempTape(t, "")},
		effects:      Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}},
	}
	done := make(chan error, 1)
	go func() {
		done <- runPluginWithCapturedStdout(session, "apply", apply.Input{AutoApprove: true})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunPlugin error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunPlugin --ci hung waiting for input")
	}
}

// TestRunPlugin_WhenApplyMacroBackend_ShouldRecordTerraformApply verifies that
// `tfui apply --auto-approve --macro tape.txt --ci` swaps the backend to
// MacroService, runs the model headlessly, and the recorded `terraform apply`
// call lands on stdout. No terraform binary is invoked. The tape may be empty
// since AutoApprove bypasses the only prompt.
func TestRunPlugin_WhenApplyMacroBackend_ShouldRecordTerraformApply(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0644); err != nil {
		t.Fatalf("write tf: %v", err)
	}
	var stdoutBuf bytes.Buffer
	session := &Session{
		cfg:          config.Config{Dir: dir},
		ciMode:       true,
		silentStderr: true,
		macro:        MacroSpec{TapeURI: writeTempTape(t, "")},
		effects:      Effects{Stdout: &stdoutBuf, Stderr: &bytes.Buffer{}},
	}
	err := runPluginWithCapturedStdout(session, "apply", apply.Input{AutoApprove: true})
	if err != nil {
		t.Fatalf("RunPlugin error = %v", err)
	}
	if want := "terraform -chdir=" + dir + " apply -auto-approve\n"; stdoutBuf.String() != want {
		t.Errorf("stdout = %q, want %q", stdoutBuf.String(), want)
	}
}

// runPluginWithCapturedStdout is a tiny adapter that calls Session.RunPlugin
// with apply's typed Input — keeps the test bodies linear.
func runPluginWithCapturedStdout(session *Session, id string, input apply.Input) error {
	return session.RunPlugin(context.Background(), id, func(p sdk.Plugin) tea.Cmd {
		return p.(*apply.Plugin).Activate(input)
	})
}

// TestRunPlugin_WhenTaintAutoConfirmCI_ShouldTerminateWithoutKeystrokes is
// the per-plugin CI-termination invariant guard for the action verbs.
// AutoConfirm drives the lifecycle past the confirm prelude — without it the
// headless driver waits on a prompt that can never be answered.
func TestRunPlugin_WhenTaintAutoConfirmCI_ShouldTerminateWithoutKeystrokes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0644); err != nil {
		t.Fatalf("write tf: %v", err)
	}
	session := &Session{
		cfg:          config.Config{Dir: dir},
		ciMode:       true,
		silentStderr: true,
		macro:        MacroSpec{TapeURI: writeTempTape(t, "")},
		effects:      Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}},
	}
	done := make(chan error, 1)
	go func() {
		done <- session.RunPlugin(context.Background(), "taint", func(p sdk.Plugin) tea.Cmd {
			return p.(*taint.Plugin).Activate(taint.Input{Addrs: []string{"null_resource.x"}, AutoConfirm: true})
		})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunPlugin error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunPlugin --ci hung waiting for taint confirmation")
	}
}

// TestResolveSilentStderrFrom_ModeMatrix pins the headless (silent-stderr)
// decision matrix: explicit --ci wins, CI=1 (exact match) wins, agent
// environments (CLAUDECODE, AI_AGENT) win, otherwise a non-TTY stderr means
// headless.
func TestResolveSilentStderrFrom_ModeMatrix(t *testing.T) {
	tests := []struct {
		name        string
		ciMode      bool
		env         map[string]string
		stderrIsTTY bool
		want        bool
	}{
		{"ShouldSilenceOnCIFlag", true, nil, true, true},
		{"ShouldSilenceOnCIEnvExactlyOne", false, map[string]string{"CI": "1"}, true, true},
		{"ShouldNotSilenceOnCIEnvTrue", false, map[string]string{"CI": "true"}, true, false},
		{"ShouldSilenceOnClaudeCodeEnv", false, map[string]string{"CLAUDECODE": "1"}, true, true},
		{"ShouldSilenceOnAIAgentEnv", false, map[string]string{"AI_AGENT": "claude-code"}, true, true},
		{"ShouldNotSilenceOnInteractiveTTY", false, nil, true, false},
		{"ShouldSilenceOnNonTTYStderr", false, nil, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(k string) string { return tt.env[k] }
			stderrIsTTY := func() bool { return tt.stderrIsTTY }
			got := resolveSilentStderrFrom(tt.ciMode, getenv, stderrIsTTY)
			if got != tt.want {
				t.Errorf("resolveSilentStderrFrom(%v, env=%v, tty=%v) = %v, want %v",
					tt.ciMode, tt.env, tt.stderrIsTTY, got, tt.want)
			}
		})
	}
}

// writeTempTape writes a tape file with the given content and returns its
// absolute path. Empty content is valid — the macro runner handles it cleanly.
func writeTempTape(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "tape-*.txt")
	if err != nil {
		t.Fatalf("create tape: %v", err)
	}
	if content != "" {
		if _, err := f.WriteString(content); err != nil {
			t.Fatalf("write tape: %v", err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close tape: %v", err)
	}
	return f.Name()
}

// TestRunPluginFrom_WhenEntryIsTheVerb_ShouldRecordTerraformApply pins that
// entering through the verb's own plugin behaves as RunPlugin does.
func TestRunPluginFrom_WhenEntryIsTheVerb_ShouldRecordTerraformApply(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0644); err != nil {
		t.Fatalf("write tf: %v", err)
	}
	var stdoutBuf bytes.Buffer
	session := &Session{
		cfg:          config.Config{Dir: dir},
		ciMode:       true,
		silentStderr: true,
		macro:        MacroSpec{TapeURI: writeTempTape(t, "")},
		effects:      Effects{Stdout: &stdoutBuf, Stderr: &bytes.Buffer{}},
	}

	done := make(chan error, 1)
	go func() {
		done <- session.RunPluginFrom(context.Background(), "apply", "apply", func(p sdk.Plugin) tea.Cmd {
			return p.(*apply.Plugin).Activate(apply.Input{AutoApprove: true})
		})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunPluginFrom error = %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("RunPluginFrom hung")
	}

	if want := "terraform -chdir=" + dir + " apply -auto-approve\n"; stdoutBuf.String() != want {
		t.Errorf("stdout = %q, want %q", stdoutBuf.String(), want)
	}
}

func TestRunPluginFrom_WhenEntryPluginDisabled_ShouldReturnDisabledError(t *testing.T) {
	disabled := false
	session := &Session{
		cfg: config.Config{
			Dir:     t.TempDir(),
			Plugins: map[string]config.PluginConfig{"plan": {Enabled: &disabled}},
		},
		silentStderr: true,
		effects:      Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}},
	}

	err := session.RunPluginFrom(context.Background(), "apply", "plan", func(sdk.Plugin) tea.Cmd { return nil })

	if err == nil || !strings.Contains(err.Error(), `plugin "plan" is disabled in tfui.hcl`) {
		t.Errorf("RunPluginFrom error = %v, want the disabled-plugin error for the entry plugin", err)
	}
}

// statusPlugin is a plugin reduced to the lifecycle status outputPortOwner reads.
type statusPlugin struct {
	sdk.Plugin
	status sdk.Status
}

func (p *statusPlugin) Status() sdk.Status { return p.status }

// TestOutputPortOwner covers `tfui apply` entered through plan: when plan
// fails before the hand-over, apply never leaves idle and the plan's error
// and exit code are what the user needs.
func TestOutputPortOwner(t *testing.T) {
	tests := []struct {
		name      string
		verb      sdk.Status
		entry     sdk.Status
		wantEntry bool
	}{
		{name: "entry failed before the hand-over", verb: sdk.StatusIdle, entry: sdk.StatusError, wantEntry: true},
		{name: "verb ran after the hand-over", verb: sdk.StatusError, entry: sdk.StatusDone, wantEntry: false},
		{name: "verb idle, entry done", verb: sdk.StatusIdle, entry: sdk.StatusDone, wantEntry: false},
		{name: "verb done", verb: sdk.StatusDone, entry: sdk.StatusDone, wantEntry: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verb := &statusPlugin{status: tt.verb}
			entry := &statusPlugin{status: tt.entry}
			got := outputPortOwner(verb, entry)
			if (got == sdk.Plugin(entry)) != tt.wantEntry {
				t.Errorf("outputPortOwner() = entry? %v, want %v", got == sdk.Plugin(entry), tt.wantEntry)
			}
		})
	}
}

// exitingPlugin reports a non-zero exit code and records its Shutdown, so a
// test can see whether Shutdown ran before the process exit.
type exitingPlugin struct {
	sdk.Plugin
	events *[]string
	code   int
}

func (p *exitingPlugin) ID() string    { return "plan" }
func (p *exitingPlugin) ExitCode() int { return p.code }
func (p *exitingPlugin) Shutdown()     { *p.events = append(*p.events, "shutdown") }

func TestPumpOutputPort_WhenOwnerExitCodeVaries_ShouldReturnExitError(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		wantCode int
		wantErr  bool
	}{
		{name: "non-zero exit code returns exitError", exitCode: 2, wantCode: 2, wantErr: true},
		{name: "zero exit code returns nil", exitCode: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var events []string
			registry := plugin.NewRegistry()
			registry.RegisterFactory("plan", func(_ sdk.Service) sdk.Plugin {
				return &exitingPlugin{events: &events, code: tt.exitCode}
			}, plugin.PluginMeta{})
			registry.Build(nil, nil)
			session := &Session{effects: Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}}

			err := session.pumpOutputPort(registry, "plan", "plan", nil)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("pumpOutputPort error = %v, want nil", err)
				}
			} else {
				var exitErr *exitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("pumpOutputPort error = %v (%T), want *exitError", err, err)
				}
				if exitErr.code != tt.wantCode {
					t.Errorf("exitError.code = %d, want %d", exitErr.code, tt.wantCode)
				}
			}
			if len(events) != 0 {
				t.Errorf("pumpOutputPort shut plugins down itself: events = %v", events)
			}
		})
	}
}

// reportingPlugin fills every output port, so a test can see which of them
// reach the user.
type reportingPlugin struct {
	sdk.Plugin
	code int
}

func (p *reportingPlugin) ID() string              { return "plan" }
func (p *reportingPlugin) Stdout() ([]byte, error) { return []byte("Plan: 1 to add\n"), nil }
func (p *reportingPlugin) Stderr() []byte          { return []byte("terraform said why\n") }
func (p *reportingPlugin) ExitCode() int           { return p.code }

func TestPumpOutputPort_WhenInterrupted_ShouldSkipStdoutKeepStderrAndExitWithSignalCode(t *testing.T) {
	tests := []struct {
		name     string
		sig      os.Signal
		exitCode int
		wantCode int
	}{
		{name: "SIGINT overrides a zero exit code", sig: syscall.SIGINT, exitCode: 0, wantCode: 130},
		{name: "SIGINT overrides the plugin's exit code", sig: syscall.SIGINT, exitCode: 1, wantCode: 130},
		{name: "SIGTERM overrides the plugin's exit code", sig: syscall.SIGTERM, exitCode: 1, wantCode: 143},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := plugin.NewRegistry()
			registry.RegisterFactory("plan", func(_ sdk.Service) sdk.Plugin {
				return &reportingPlugin{code: tt.exitCode}
			}, plugin.PluginMeta{})
			registry.Build(nil, nil)
			var stdout, stderr bytes.Buffer
			session := &Session{effects: Effects{Stdout: &stdout, Stderr: &stderr}}

			err := session.pumpOutputPort(registry, "plan", "plan", tt.sig)

			var exitErr *exitError
			if !errors.As(err, &exitErr) || exitErr.code != tt.wantCode {
				t.Fatalf("pumpOutputPort error = %v, want exitError with code %d", err, tt.wantCode)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty after a signal", stdout.String())
			}
			if got := stderr.String(); got != "terraform said why\n" {
				t.Errorf("stderr = %q, want the plugin's stderr as is", got)
			}
		})
	}
}

// TestSession_FirstSignalDuringTape_ShouldStopPlaybackAndExit130 covers both
// macro call sites: tfui --macro (Run) and a headless verb with --macro
// (RunPlugin). The tape's first screenshot is a FIFO, so reading its first
// byte proves the guard is watching and playback has reached that command. The
// test opens the FIFO read-write, so neither its open nor the session's ever
// blocks, and closing it releases a pending read. The test
// holds SIGINT for its whole run, so a signal that no guard takes cannot reach
// the default handler. A session that outlives its subtest still watches
// SIGINT and would take the next subtest's signal as its second, which exits
// the binary, so the test stops there.
func TestSession_FirstSignalDuringTape_ShouldStopPlaybackAndExit130(t *testing.T) {
	keepAlive := make(chan os.Signal, 1)
	signal.Notify(keepAlive, syscall.SIGINT)
	defer signal.Stop(keepAlive)

	tests := []struct {
		name string
		run  func(*Session) error
	}{
		{"Run", (*Session).Run},
		{"RunPlugin", func(s *Session) error {
			return runPluginWithCapturedStdout(s, "apply", apply.Input{AutoApprove: true})
		}},
	}
	leaked := ""
	for _, tt := range tests {
		if leaked != "" {
			t.Fatalf("not running %s: the %s session still watches SIGINT", tt.name, leaked)
		}
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0644); err != nil {
				t.Fatalf("write tf: %v", err)
			}
			started := filepath.Join(dir, "started")
			if err := syscall.Mkfifo(started, 0600); err != nil {
				t.Fatalf("mkfifo: %v", err)
			}
			later := filepath.Join(dir, "later")
			session := &Session{
				cfg:          config.Config{Dir: dir},
				ciMode:       true,
				silentStderr: true,
				macro:        MacroSpec{TapeURI: writeTempTape(t, "screenshot "+started+"\nsleep 1h\nscreenshot "+later+"\n")},
				effects:      Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}},
			}
			fifo, err := os.OpenFile(started, os.O_RDWR, 0)
			if err != nil {
				t.Fatalf("open %s: %v", started, err)
			}
			defer func() { _ = fifo.Close() }()
			read := make(chan error, 1)
			go func() {
				_, err := fifo.Read(make([]byte, 1))
				read <- err
			}()
			done := make(chan error, 1)
			go func() { done <- tt.run(session) }()

			select {
			case err := <-read:
				if err != nil {
					t.Fatalf("read %s: %v", started, err)
				}
			case err := <-done:
				t.Fatalf("session returned before its first screenshot: %v", err)
			case <-time.After(10 * time.Second):
				leaked = tt.name
				t.Fatal("session never reached its first screenshot")
			}
			if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
				t.Fatalf("send SIGINT: %v", err)
			}
			select {
			case err := <-done:
				if code, _ := exitStatus(err); code != 130 {
					t.Errorf("exit code = %d (err %v), want 130", code, err)
				}
			case <-time.After(10 * time.Second):
				leaked = tt.name
				t.Fatal("tape kept playing after the first signal")
			}
			if _, err := os.Stat(later); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("stat %s = %v, want the command after the signal not to run", later, err)
			}
		})
	}
}
