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
	"github.com/lmarqs/terraform-ui/plugins/plan"
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

// writeFailingTerraform writes a fake terraform binary that answers the
// version probe and fails every other command with stderrMsg on stderr, the
// way a broken configuration makes `terraform plan` fail.
func writeFailingTerraform(t *testing.T, stderrMsg string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "terraform")
	script := `#!/bin/sh
if [ "$1" = "version" ]; then
  echo '{"terraform_version":"1.14.0","platform":"linux_amd64","provider_selections":{},"terraform_outdated":false}'
  exit 0
fi
echo "` + stderrMsg + `" >&2
exit 1
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake terraform: %v", err)
	}
	return bin
}

// TestRunPluginFrom_WhenEntryPlanFailsBeforeApply_ShouldExitWithPlanError
// covers `tfui apply` entered through plan: the plan fails, apply never
// leaves idle, so the plan's error and exit code reach the user.
func TestRunPluginFrom_WhenEntryPlanFailsBeforeApply_ShouldExitWithPlanError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0644); err != nil {
		t.Fatalf("write tf: %v", err)
	}
	var stderrBuf bytes.Buffer
	exitCode := 0
	session := &Session{
		cfg: config.Config{
			Dir:       dir,
			Terraform: config.TerraformConfig{Bin: writeFailingTerraform(t, "Error: Unsupported block type boom")},
		},
		ciMode:       true,
		silentStderr: true,
		effects:      Effects{Stdout: &bytes.Buffer{}, Stderr: &stderrBuf, Exit: func(code int) { exitCode = code }},
	}

	done := make(chan error, 1)
	go func() {
		done <- session.RunPluginFrom(context.Background(), "apply", "plan", func(p sdk.Plugin) tea.Cmd {
			return p.(*plan.Plugin).Activate(plan.Input{})
		})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunPluginFrom error = %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("RunPluginFrom hung after the entry plan failed")
	}

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (the failed plan's)", exitCode)
	}
	if !strings.Contains(stderrBuf.String(), "boom") {
		t.Errorf("stderr = %q, want the plan's terraform error", stderrBuf.String())
	}
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
