package main

import (
	"bytes"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	tfexec "github.com/lmarqs/terraform-ui/internal/terraform/exec"
	"github.com/spf13/cobra"
)

func TestBuildService_GivenDryRun_ShouldPickTheRecordingService(t *testing.T) {
	tests := []struct {
		name          string
		session       Session
		wantRecording bool
	}{
		{name: "live run", session: Session{}, wantRecording: false},
		{name: "dry run", session: Session{dryRun: true}, wantRecording: true},
		{name: "macro run", session: Session{macro: MacroSpec{TapeURI: "tape.txt"}}, wantRecording: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.session.cfg = config.Config{Dir: t.TempDir()}
			svc, recorder, err := tt.session.buildService()
			if err != nil {
				t.Fatalf("buildService() error = %v", err)
			}
			if tt.wantRecording {
				if _, ok := svc.(*terraform.MacroService); !ok || recorder == nil {
					t.Fatalf("buildService() = %T, recorder %v; want the MacroService as both", svc, recorder)
				}
				return
			}
			if _, ok := svc.(*tfexec.ExecService); !ok || recorder != nil {
				t.Fatalf("buildService() = %T, recorder %v; want the ExecService and no recorder", svc, recorder)
			}
		})
	}
}

func TestDryRunFlag_GivenRoot_ShouldScopeItToTheRootCommandOnly(t *testing.T) {
	root, session, _ := buildRoot()
	plan := buildPlanCommand(session)
	root.AddCommand(plan)
	if root.LocalNonPersistentFlags().Lookup("dry-run") == nil {
		t.Error("dry-run must be a local flag of the root command")
	}
	if root.PersistentFlags().Lookup("dry-run") != nil {
		t.Error("root must not carry a persistent dry-run flag")
	}
	if plan.Flags().Lookup("dry-run") != nil {
		t.Error("plan must not inherit dry-run")
	}
}

func TestDryRunFlag_GivenPassthroughArgs_ShouldWarnOnlyOnALiveRun(t *testing.T) {
	for _, tc := range []struct {
		name        string
		args        []string
		wantDryRun  bool
		wantWarning string
	}{
		{name: "live run", wantWarning: "warning: ignoring arguments after --: -compact-warnings; only a -macro run records them\n"},
		{name: "dry run", args: []string{"-dry-run"}, wantDryRun: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, session, extraArgs := buildRoot()
			var stdout, stderr bytes.Buffer
			session.effects = Effects{Stdout: &stdout, Stderr: &stderr}
			root.SilenceUsage, root.SilenceErrors = true, true
			root.RunE = func(*cobra.Command, []string) error { return nil }
			*extraArgs = []string{"-compact-warnings"}
			root.SetArgs(normalizeArgs(append([]string{"--project", t.TempDir()}, tc.args...)))
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if session.dryRun != tc.wantDryRun {
				t.Errorf("session.dryRun = %v, want %v", session.dryRun, tc.wantDryRun)
			}
			if got := stderr.String(); got != tc.wantWarning {
				t.Errorf("stderr = %q, want %q", got, tc.wantWarning)
			}
		})
	}
}
