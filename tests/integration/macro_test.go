//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMacro(t *testing.T) {
	projectRoot := findProjectRoot()
	planFixture := filepath.Join(projectRoot, "tests", "fixtures", "plan.json")

	tests := []struct {
		name       string
		tape       string
		args       []string
		wantExit   int
		wantStderr string
	}{
		{
			name:     "home menu visible after init",
			tape:     "wait ready\nassert view Plan\nassert view State Browser",
			args:     []string{"--plan", planFixture},
			wantExit: 0,
		},
		{
			name:       "assert failure exits 1",
			tape:       "wait ready\nassert view nonexistent_resource",
			args:       []string{"--plan", planFixture},
			wantExit:   1,
			wantStderr: "assertion failed",
		},
		{
			name:       "syntax error exits 2",
			tape:       "badcmd foo",
			args:       []string{"--plan", planFixture},
			wantExit:   2,
			wantStderr: "unknown command",
		},
		{
			name:     "navigate to plan and verify resources",
			tape:     "wait ready\nkey p\nassert view aws_instance.web\nassert view aws_s3_bucket.data",
			args:     []string{"--plan", planFixture},
			wantExit: 0,
		},
		{
			name:     "screenshot writes file",
			tape:     "wait ready\nscreenshot %s",
			args:     []string{"--plan", planFixture},
			wantExit: 0,
		},
		{
			name:       "macro without plan or state fails",
			tape:       "wait ready",
			args:       nil,
			wantExit:   1,
			wantStderr: "--macro requires --plan or --state",
		},
		{
			name:     "empty tape succeeds",
			tape:     "# just a comment",
			args:     []string{"--plan", planFixture},
			wantExit: 0,
		},
		{
			name:     "resize command",
			tape:     "wait ready\nresize 120 40",
			args:     []string{"--plan", planFixture},
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tape := tt.tape
			var screenshotPath string
			if strings.Contains(tape, "%s") {
				screenshotPath = filepath.Join(t.TempDir(), "screenshot.txt")
				tape = fmt.Sprintf(tape, screenshotPath)
			}

			tapeFile := filepath.Join(t.TempDir(), "test.tape")
			if err := os.WriteFile(tapeFile, []byte(tape), 0644); err != nil {
				t.Fatalf("write tape: %v", err)
			}

			args := append([]string{"--macro", tapeFile}, tt.args...)
			_, stderr, err := runTfui(args...)

			exitCode := 0
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					exitCode = ee.ExitCode()
				} else {
					t.Fatalf("unexpected error type: %v", err)
				}
			}

			if exitCode != tt.wantExit {
				t.Errorf("exit code = %d, want %d\nstderr: %s", exitCode, tt.wantExit, stderr)
			}

			if tt.wantStderr != "" && !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("stderr = %q, want to contain %q", stderr, tt.wantStderr)
			}

			if screenshotPath != "" {
				if _, err := os.Stat(screenshotPath); err != nil {
					t.Errorf("screenshot file not created: %v", err)
				}
			}
		})
	}
}
