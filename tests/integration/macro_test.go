//go:build integration

package integration

import (
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
	tapeDir := filepath.Join(projectRoot, "tests", "fixtures", "tapes", "macro")

	tape := func(name string) string { return filepath.Join(tapeDir, name) }

	tests := []struct {
		name       string
		tapeFile   string
		args       []string
		setup      func(t *testing.T) []string
		cwdFunc    func(t *testing.T) string
		assertFile string // when set, asserts the file exists relative to cwd after run
		wantExit   int
		wantStderr string
		wantStdout string
	}{
		{
			name:     "home menu visible after init",
			tapeFile: tape("home_menu.tape"),
			args:     []string{"-plan", planFixture},
			wantExit: 0,
		},
		{
			name:       "assert failure exits 1",
			tapeFile:   tape("assert_failure.tape"),
			args:       []string{"-plan", planFixture},
			wantExit:   1,
			wantStderr: "assertion failed",
		},
		{
			name:       "syntax error exits 2",
			tapeFile:   tape("syntax_error.tape"),
			args:       []string{"-plan", planFixture},
			wantExit:   2,
			wantStderr: "unknown command",
		},
		{
			name:     "navigate to plan and verify resources",
			tapeFile: tape("plan_resources.tape"),
			args:     []string{"-plan", planFixture},
			wantExit: 0,
		},
		{
			name:       "screenshot writes file",
			tapeFile:   tape("screenshot.tape"),
			args:       []string{"-plan", planFixture},
			cwdFunc:    func(t *testing.T) string { return t.TempDir() },
			assertFile: "screenshot.txt",
			wantExit:   0,
		},
		{
			name:     "macro without plan or state navigates to init",
			tapeFile: tape("init_navigate.tape"),
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
				if err := os.WriteFile(filepath.Join(dir, "tfui.hcl"), []byte("member \"modules/vpc\" {}\n"), 0644); err != nil {
					t.Fatalf("write tfui.hcl: %v", err)
				}
				return []string{"-project", dir}
			},
			wantExit: 0,
		},
		{
			name:     "empty tape succeeds",
			tapeFile: tape("empty.tape"),
			args:     []string{"-plan", planFixture},
			wantExit: 0,
		},
		{
			name:     "resize command",
			tapeFile: tape("resize.tape"),
			args:     []string{"-plan", planFixture},
			wantExit: 0,
		},
		{
			name:       "apply outputs command to stdout",
			tapeFile:   tape("apply_record.tape"),
			args:       []string{"-plan", planFixture, "-state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform apply ",
		},
		{
			// ADR-0019: targets belong on plan, never on apply. Scoping a
			// targeted apply means generating a targeted plan file first; the
			// apply command itself only references the resulting plan file.
			name:       "targeted plan records plan command with targets",
			tapeFile:   tape("apply_targeted.tape"),
			args:       []string{"-plan", planFixture, "-state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform plan -out=",
		},
		{
			// Plan owns all replanning: pinning AFTER an initial plan must
			// trigger a second targeted plan before apply runs. This proves
			// the deferred-apply path; without it the user would apply the
			// stale (un-targeted) plan file.
			name:       "pin after plan records targeted replan before apply",
			tapeFile:   tape("apply_after_pin_replans.tape"),
			args:       []string{"-plan", planFixture, "-state", stateFixture},
			wantExit:   0,
			wantStdout: "-target=aws_instance.web",
		},
		{
			name:       "plan records plan command",
			tapeFile:   tape("plan_record.tape"),
			args:       []string{"-plan", planFixture, "-state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform plan -out=",
		},
		{
			name:     "state list is a data fetch not recorded",
			tapeFile: tape("state_browse.tape"),
			args:     []string{"-plan", planFixture, "-state", stateFixture},
			wantExit: 0,
		},
		{
			name:       "state delete outputs state rm command",
			tapeFile:   tape("state_delete.tape"),
			args:       []string{"-plan", planFixture, "-state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform state rm aws_instance.web",
		},
		{
			name:       "state taint outputs taint command",
			tapeFile:   tape("state_taint.tape"),
			args:       []string{"-plan", planFixture, "-state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform taint aws_instance.web",
		},
		{
			name:       "state untaint outputs untaint command",
			tapeFile:   tape("state_untaint.tape"),
			args:       []string{"-plan", planFixture, "-state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform untaint aws_instance.web",
		},
		{
			name:       "state import outputs import command",
			tapeFile:   tape("state_import.tape"),
			args:       []string{"-plan", planFixture, "-state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform import aws_instance.web i-123",
		},
		{
			name:       "state move outputs state mv command",
			tapeFile:   tape("state_move.tape"),
			args:       []string{"-plan", planFixture, "-state", stateFixture},
			wantExit:   0,
			wantStdout: "terraform state mv aws_instance.web aws_instance.new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"-macro", tt.tapeFile}, tt.args...)
			if tt.setup != nil {
				args = append(args, tt.setup(t)...)
			}

			cwd := ""
			if tt.cwdFunc != nil {
				cwd = tt.cwdFunc(t)
			}

			stdout, stderr, err := runTfuiIn(cwd, args...)

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

			if tt.assertFile != "" {
				if _, err := os.Stat(filepath.Join(cwd, tt.assertFile)); err != nil {
					t.Errorf("expected file %q in cwd not created: %v", tt.assertFile, err)
				}
			}
		})
	}
}
