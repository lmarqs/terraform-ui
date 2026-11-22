package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/plugins/apply"
)

func TestRunPlugin_GivenRecordingRunInAnotherDirectory_ShouldRecordTheEffectiveChdir(t *testing.T) {
	tests := []struct {
		name      string
		chdir     string
		baseDir   string
		wantChdir string
	}{
		{name: "chdir member", chdir: "modules/app", wantChdir: "modules/app"},
		{name: "project only", wantChdir: "."},
		{name: "basedir member", baseDir: "modules/app", wantChdir: "modules/app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			member := filepath.Join(project, "modules", "app")
			if err := os.MkdirAll(member, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, dir := range []string{project, member} {
				if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want := filepath.Join(project, tt.wantChdir)
			var stdout bytes.Buffer
			session := &Session{
				cfg:          config.Config{Dir: project, Chdir: tt.chdir, BaseDir: tt.baseDir},
				ciMode:       true,
				silentStderr: true,
				macro:        MacroSpec{TapeURI: writeTempTape(t, "")},
				effects:      Effects{Stdout: &stdout, Stderr: &bytes.Buffer{}},
			}
			if err := runPluginWithCapturedStdout(session, "apply", apply.Input{AutoApprove: true}); err != nil {
				t.Fatalf("RunPlugin error = %v", err)
			}
			if !strings.Contains(stdout.String(), "-chdir="+want) {
				t.Errorf("stdout = %q, want a command with -chdir=%s", stdout.String(), want)
			}
		})
	}
}
