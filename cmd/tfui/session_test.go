package main

import (
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
		macroURI:     writeTempTape(t, ""),
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
	session := &Session{
		cfg:          config.Config{Dir: dir},
		ciMode:       true,
		silentStderr: true,
		macroURI:     writeTempTape(t, ""),
	}
	stdout, err := captureStdoutDuring(func() error {
		return runPluginWithCapturedStdout(session, "apply", apply.Input{AutoApprove: true})
	})
	if err != nil {
		t.Fatalf("RunPlugin error = %v", err)
	}
	if !strings.Contains(stdout, "terraform apply") {
		t.Errorf("stdout missing recorded `terraform apply`, got: %q", stdout)
	}
}

// runPluginWithCapturedStdout is a tiny adapter that calls Session.RunPlugin
// with apply's typed Input — keeps the test bodies linear.
func runPluginWithCapturedStdout(session *Session, id string, input apply.Input) error {
	return session.RunPlugin(context.Background(), id, func(p sdk.Plugin) tea.Cmd {
		return p.(*apply.Plugin).Activate(input)
	})
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

// captureStdoutDuring redirects os.Stdout for the duration of fn and returns
// everything written to it.
func captureStdoutDuring(fn func() error) (string, error) {
	r, w, perr := os.Pipe()
	if perr != nil {
		return "", perr
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		var out []byte
		for {
			n, err := r.Read(buf)
			if n > 0 {
				out = append(out, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
		done <- string(out)
	}()
	err := fn()
	os.Stdout = orig
	_ = w.Close()
	captured := <-done
	_ = r.Close()
	return captured, err
}
