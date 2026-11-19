package main

import (
	"bytes"
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootTarget_GivenRepeatedFlags_ShouldStartTheSessionWithThosePins(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{name: "no target", want: nil},
		{name: "single dash with a space", args: []string{"-target", "aws_instance.web"}, want: []string{"aws_instance.web"}},
		{
			name: "repeated, with equals, taken verbatim",
			args: []string{"-target=aws_instance.web", "--target", `module.a["x,y"].aws_s3_bucket.old`, "-dry-run"},
			want: []string{"aws_instance.web", `module.a["x,y"].aws_s3_bucket.old`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, session, _ := buildRoot()
			var stderr bytes.Buffer
			session.effects = Effects{Stdout: &bytes.Buffer{}, Stderr: &stderr}
			root.RunE = func(*cobra.Command, []string) error { return nil }
			root.SetArgs(normalizeArgs(append([]string{"--project", t.TempDir()}, tc.args...)))
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !slices.Equal(session.cfg.Pins, tc.want) {
				t.Errorf("session.cfg.Pins = %q, want %q", session.cfg.Pins, tc.want)
			}
		})
	}
}

func TestRootTarget_GivenPlanAndApply_ShouldStayLocalToEachCommand(t *testing.T) {
	root, session, _ := buildRoot()
	plan := buildPlanCommand(session)
	root.AddCommand(plan)
	if root.LocalNonPersistentFlags().Lookup("target") == nil {
		t.Error("target must be a local flag of the root command")
	}
	if root.PersistentFlags().Lookup("target") != nil {
		t.Error("root must not carry a persistent target flag")
	}
	if plan.InheritedFlags().Lookup("target") != nil {
		t.Error("plan must keep its own target flag, not inherit root's")
	}
}

func TestRootTarget_GivenPlanSubcommand_ShouldLeaveTheSessionPinsEmpty(t *testing.T) {
	root, session, _ := buildRoot()
	plan := buildPlanCommand(session)
	plan.RunE = func(*cobra.Command, []string) error { return nil }
	root.AddCommand(plan)
	session.effects = Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	root.SetArgs(normalizeArgs([]string{"--project", t.TempDir(), "plan", "-target", "aws_instance.web"}))
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if session.cfg.Pins != nil {
		t.Errorf("session.cfg.Pins = %q, want none: plan's -target is a plan target, not a pin", session.cfg.Pins)
	}
}

func TestRootTarget_GivenAModulePath_ShouldRejectItAndPointAtPlanTarget(t *testing.T) {
	for _, tc := range []struct {
		target     string
		wantReject bool
	}{
		{target: "module.networking", wantReject: true},
		{target: `module.app["blue"]`, wantReject: true},
		{target: "module.app[0]", wantReject: true},
		{target: "module.a.module.b", wantReject: true},
		{target: `module.a["x.y"].module.b`, wantReject: true},
		{target: "module.networking.aws_vpc.main"},
		{target: "module.a.data.aws_ami.x"},
		{target: `module.app["module.x"].aws_instance.api`},
		{target: "aws_instance.web"},
		{target: "aws_instance.web[0]"},
		{target: "data.aws_ami.module"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			root, session, _ := buildRoot()
			session.effects = Effects{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
			root.RunE = func(*cobra.Command, []string) error { return nil }
			root.SetArgs(normalizeArgs([]string{"--project", t.TempDir(), "-target", "aws_s3_bucket.ok", "-target", tc.target}))
			err := root.Execute()
			if !tc.wantReject {
				if err != nil {
					t.Fatalf("Execute() error = %v, want the resource address accepted", err)
				}
				return
			}
			want := "-target " + tc.target + " is a module path; a pin is a resource address\n\n" +
				"Target a module on the plan command instead:\n  tfui plan -target '" + tc.target + "'"
			if err == nil || err.Error() != want {
				t.Fatalf("Execute() error = %v, want %q", err, want)
			}
			if code, _ := exitStatus(err); code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
		})
	}
}
