//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lmarqs/terraform-ui/internal/config"
	"golang.org/x/sys/unix"
)

// lockedBuffer lets the test read what the terminal shows while the program writes it.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// openPTY returns both ends of an 80x24 pseudo-terminal. The program only
// sends its first WindowSizeMsg, and so leaves "Loading...", on a terminal.
func openPTY(t *testing.T) (controller, terminal *os.File) {
	t.Helper()
	controller, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal: %v", err)
	}
	fd := int(controller.Fd())
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	n, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err = os.OpenFile("/dev/pts/"+strconv.Itoa(n), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.IoctlSetWinsize(int(terminal.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 24, Col: 80}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = terminal.Close()
		_ = controller.Close()
	})
	return controller, terminal
}

// The macro driver draws nothing, so a TUI on stderr proves the interactive
// program ran; the keys reach it through the terminal, as a user's would.
func TestRunApp_GivenDryRun_ShouldRunTheProgramOnStderrAndPrintRecordedCommandsOnExit(t *testing.T) {
	for _, tc := range []struct {
		name     string
		json     bool
		record   bool
		commands func(t *testing.T, stdout string) []string
	}{
		{name: "record", record: true, commands: func(_ *testing.T, stdout string) []string {
			return strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
		}},
		{name: "text", commands: func(_ *testing.T, stdout string) []string {
			return strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
		}},
		{name: "json", json: true, commands: func(t *testing.T, stdout string) []string {
			var commands []string
			if err := json.Unmarshal([]byte(stdout), &commands); err != nil {
				t.Fatalf("stdout is not a JSON array: %v\n%s", err, stdout)
			}
			return commands
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixtures, err := filepath.Abs(filepath.Join("..", "..", "tests", "fixtures"))
			if err != nil {
				t.Fatal(err)
			}
			controller, terminal := openPTY(t)
			screen := &lockedBuffer{}
			go func() { _, _ = io.Copy(screen, controller) }()
			var stdout bytes.Buffer
			session := &Session{
				cfg:        config.Config{Dir: t.TempDir(), PreloadedData: true},
				sources:    PreseedSources{Plan: filepath.Join(fixtures, "plan.json"), State: filepath.Join(fixtures, "state.json")},
				dryRun:     true,
				jsonStdout: tc.json,
				effects:    Effects{Stdin: terminal, Stdout: &stdout, Stderr: terminal},
			}
			if tc.record {
				session.macro.RecordDir = t.TempDir()
			}
			done := make(chan error, 1)
			go func() { done <- session.runApp() }()

			waitFor(t, screen, "Plan")
			if strings.Contains(screen.String(), "Console") {
				t.Error("home menu offers Console, which runs terraform itself")
			}
			if _, err := controller.Write([]byte("p")); err != nil {
				t.Fatal(err)
			}
			waitFor(t, screen, "aws_instance.web")
			if _, err := controller.Write([]byte{0x03}); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("runApp() error = %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("runApp() did not exit after ctrl+c")
			}

			commands := tc.commands(t, stdout.String())
			if len(commands) != 1 || !strings.HasPrefix(commands[0], "terraform plan -out=") {
				t.Errorf("recorded commands = %q, want the one terraform plan the p key ran", commands)
			}
			if tc.record {
				if _, err := os.Stat(filepath.Join(session.macro.RecordDir, "manifest.json")); err != nil {
					t.Errorf("-record wrote no manifest: %v", err)
				}
			}
		})
	}
}

func waitFor(t *testing.T, screen *lockedBuffer, text string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(screen.String(), text) {
		if time.Now().After(deadline) {
			t.Fatalf("terminal never showed %q; got:\n%q", text, screen.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
