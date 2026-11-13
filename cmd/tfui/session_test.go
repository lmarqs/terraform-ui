package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
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
		effects:      Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Exit: func(int) {}},
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
		effects:      Effects{Stdout: &stdoutBuf, Stderr: &bytes.Buffer{}, Exit: func(int) {}},
	}
	err := runPluginWithCapturedStdout(session, "apply", apply.Input{AutoApprove: true})
	if err != nil {
		t.Fatalf("RunPlugin error = %v", err)
	}
	if !strings.Contains(stdoutBuf.String(), "terraform apply") {
		t.Errorf("stdout missing recorded `terraform apply`, got: %q", stdoutBuf.String())
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
		effects:      Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Exit: func(int) {}},
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
		effects:      Effects{Stdout: &stdoutBuf, Stderr: &bytes.Buffer{}, Exit: func(int) {}},
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

	if !strings.Contains(stdoutBuf.String(), "terraform apply") {
		t.Errorf("stdout missing recorded `terraform apply`, got: %q", stdoutBuf.String())
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
		effects:      Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Exit: func(int) {}},
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
