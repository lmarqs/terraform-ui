//go:build integration

package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestChdir_ImperativeCommands_ShouldRunInTheSelectedMember covers the
// subcommands that bypass the plugin/Session path and build their own terraform
// service: they used to resolve the project root and ignore --chdir entirely
// (lmarqs/terraform-ui#51).
//
// The fixture's root declares an uninitialized s3 backend, so a command that
// runs there reports "Backend initialization required" — the exact symptom from
// the report — while the member is initialized and answers normally.
func TestChdir_ImperativeCommands_ShouldRunInTheSelectedMember(t *testing.T) {
	project := copyFixtureWithoutInit(t, "chdir-project")
	initInDir(t, filepath.Join(project, "envs", "demo"))

	tests := []struct {
		name       string
		args       []string
		wantStdout string
		wantStderr string
	}{
		{
			name:       "workspace list",
			args:       []string{"workspace", "list", "-project", project, "-chdir", "envs/demo"},
			wantStdout: "default",
		},
		{
			name:       "workspace show",
			args:       []string{"workspace", "show", "-project", project, "-chdir", "envs/demo"},
			wantStdout: "default",
		},
		{
			// force-unlock cannot succeed without a held lock; reaching the
			// member's local state at all is the signal. Failing on the root's
			// backend would never get that far.
			name:       "force-unlock",
			args:       []string{"force-unlock", "-force", "00000000-0000-0000-0000-000000000000", "-project", project, "-chdir", "envs/demo"},
			wantStderr: "not locked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runTfui(tt.args...)
			if strings.Contains(stderr, "Backend initialization required") {
				t.Fatalf("command ran in the project root, not the member\nstderr: %s", stderr)
			}
			if tt.wantStdout != "" {
				if err != nil {
					t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
				}
				if !strings.Contains(stdout, tt.wantStdout) {
					t.Errorf("expected %q in stdout, got: %q", tt.wantStdout, stdout)
				}
			}
			if tt.wantStderr != "" && !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("expected %q in stderr, got: %q", tt.wantStderr, stderr)
			}
		})
	}
}

// TestChdir_ImperativeCommands_WhenChdirDoesNotResolve_ShouldFailFast keeps the
// imperative commands aligned with the plugin path, which validates --chdir
// before dispatching: an unresolvable member must be reported as such, not run
// as terraform-in-a-missing-directory.
func TestChdir_ImperativeCommands_WhenChdirDoesNotResolve_ShouldFailFast(t *testing.T) {
	project := copyFixtureWithoutInit(t, "chdir-project")

	_, stderr, err := runTfui("workspace", "list", "-project", project, "-chdir", "envs/nope")
	if err == nil {
		t.Fatalf("expected a failure for an unresolvable --chdir, got success\nstderr: %s", stderr)
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("expected the error to name the unresolved chdir, got: %q", stderr)
	}
}
