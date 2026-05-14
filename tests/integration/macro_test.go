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

	stateFixture := filepath.Join(projectRoot, "tests", "fixtures", "state.json")

	tests := []struct {
		name       string
		tape       string
		args       []string
		setup      func(t *testing.T) []string // returns extra args
		wantExit   int
		wantStderr string
		wantStdout string
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
			name: "macro without plan or state navigates to init",
			tape: "wait ready\nkey i\nwait view Init",
			args: nil,
			setup: func(t *testing.T) []string {
				t.Helper()
				dir := t.TempDir()
				vpcDir := filepath.Join(dir, "modules", "vpc")
				if err := os.MkdirAll(vpcDir, 0755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(vpcDir, "main.tf"), []byte("resource \"aws_vpc\" \"main\" {}\n"), 0644); err != nil {
					t.Fatalf("write main.tf: %v", err)
				}
				return []string{"--project", dir}
			},
			wantExit: 0,
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
		{
			name:       "apply outputs command to stdout",
			tape:       "wait ready\nkey p\nwait view aws_instance\nkey a\nwait view Apply plan\nkey y\nwait view Are you sure\nkey y",
			args:       []string{"--plan", planFixture, "--state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform apply\n",
		},
		{
			name:       "targeted apply outputs command with targets",
			tape:       "wait ready\nkey p\nwait view aws_instance\nkey space\nkey a\nwait view targeted resource\nkey y\nwait view Are you sure\nkey y",
			args:       []string{"--plan", planFixture, "--state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform apply -target=",
		},
		{
			name:       "plan records plan command",
			tape:       "wait ready\nkey p\nwait view aws_instance",
			args:       []string{"--plan", planFixture, "--state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform plan\n",
		},
		{
			name:     "state list is a data fetch not recorded",
			tape:     "wait ready\nkey s\nwait view aws_instance.web",
			args:     []string{"--plan", planFixture, "--state", stateFixture},
			wantExit: 0,
		},
		{
			name:       "state delete outputs state rm command",
			tape:       "wait ready\nkey s\nwait view aws_instance.web\nkey d\nwait view Remove\nkey y",
			args:       []string{"--plan", planFixture, "--state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform state rm aws_instance.web",
		},
		{
			name:       "state taint outputs taint command",
			tape:       "wait ready\nkey s\nwait view aws_instance.web\nkey t\nwait view Taint\nkey y",
			args:       []string{"--plan", planFixture, "--state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform taint aws_instance.web",
		},
		{
			name:       "state untaint outputs untaint command",
			tape:       "wait ready\nkey s\nwait view aws_instance.web\nkey T\nwait view Untaint\nkey y",
			args:       []string{"--plan", planFixture, "--state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform untaint aws_instance.web",
		},
		{
			name:       "state import outputs import command",
			tape:       "wait ready\nkey s\nwait view aws_instance.web\nkey n\nwait view Resource ID\nkey i\nkey -\nkey 1\nkey 2\nkey 3\nkey enter\nwait view Import\nkey y",
			args:       []string{"--plan", planFixture, "--state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform import aws_instance.web i-123",
		},
		{
			name:       "state move outputs state mv command",
			tape:       "wait ready\nkey s\nwait view aws_instance.web\nkey m\nwait view Move to\nkey backspace\nkey backspace\nkey backspace\nkey n\nkey e\nkey w\nkey enter\nwait view Move\nkey y",
			args:       []string{"--plan", planFixture, "--state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform state mv aws_instance.web aws_instance.new",
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
			if tt.setup != nil {
				args = append(args, tt.setup(t)...)
			}
			stdout, stderr, err := runTfui(args...)

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

			if tt.wantStdout != "" && !strings.Contains(stdout, tt.wantStdout) {
				t.Errorf("stdout = %q, want to contain %q", stdout, tt.wantStdout)
			}

			if screenshotPath != "" {
				if _, err := os.Stat(screenshotPath); err != nil {
					t.Errorf("screenshot file not created: %v", err)
				}
			}
		})
	}
}
