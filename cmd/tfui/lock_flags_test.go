package main

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/plugins/apply"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	"github.com/spf13/cobra"
)

// TestLockFlags_WhenParsed_ShouldReachTypedInput covers the per-invocation lock
// override on both commands that take one: an unpassed --lock must stay
// LockDefault so the value resolved from tfui.hcl survives
// (lmarqs/terraform-ui#58).
func TestLockFlags_WhenParsed_ShouldReachTypedInput(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantLock    sdk.LockMode
		wantTimeout sdk.LockTimeout
	}{
		{name: "no lock flag stays unspecified", args: nil, wantLock: sdk.LockDefault},
		{name: "explicit false disables", args: []string{"--lock=false"}, wantLock: sdk.LockDisabled},
		{name: "explicit true enables", args: []string{"--lock=true"}, wantLock: sdk.LockEnabled},
		{name: "bare flag enables", args: []string{"--lock"}, wantLock: sdk.LockEnabled},
		{
			name:        "timeout reaches input",
			args:        []string{"--lock-timeout=30s"},
			wantLock:    sdk.LockDefault,
			wantTimeout: sdk.LockTimeout("30s"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (plan)", func(t *testing.T) {
			var captured tfuiplan.Input
			c := buildPlanCommand(&Session{cfg: config.Config{}})
			c.RunE = func(cmd *cobra.Command, _ []string) error {
				captured.Lock = lockModeFrom(cmd, mustBool(cmd, "lock"))
				captured.LockTimeout = sdk.LockTimeout(mustString(cmd, "lock-timeout"))
				return nil
			}
			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			assertLock(t, captured.Lock, captured.LockTimeout, tt.wantLock, tt.wantTimeout)
		})

		t.Run(tt.name+" (apply)", func(t *testing.T) {
			var captured apply.Input
			c := buildApplyCommand(&Session{cfg: config.Config{}})
			c.RunE = func(cmd *cobra.Command, _ []string) error {
				captured.Lock = lockModeFrom(cmd, mustBool(cmd, "lock"))
				captured.LockTimeout = sdk.LockTimeout(mustString(cmd, "lock-timeout"))
				return nil
			}
			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			assertLock(t, captured.Lock, captured.LockTimeout, tt.wantLock, tt.wantTimeout)
		})
	}
}

// TestPlanApplyCommands_WhenGivenPositionalArgs_ShouldReject keeps a mistyped
// `-lock false` from being silently read as `--lock` plus an ignored argument.
func TestPlanApplyCommands_WhenGivenPositionalArgs_ShouldReject(t *testing.T) {
	for _, build := range []struct {
		name string
		fn   func(*Session) *cobra.Command
	}{
		{"plan", buildPlanCommand},
		{"apply", buildApplyCommand},
	} {
		t.Run(build.name, func(t *testing.T) {
			c := build.fn(&Session{cfg: config.Config{}})
			c.SetArgs([]string{"--lock", "false"})
			c.SilenceErrors = true
			c.SetOut(nil)
			if err := c.Execute(); err == nil {
				t.Error("Execute() = nil, want an error for the stray positional argument")
			}
		})
	}
}

func assertLock(t *testing.T, gotLock sdk.LockMode, gotTimeout sdk.LockTimeout, wantLock sdk.LockMode, wantTimeout sdk.LockTimeout) {
	t.Helper()
	if gotLock != wantLock {
		t.Errorf("Input.Lock = %v, want %v", gotLock, wantLock)
	}
	if gotTimeout != wantTimeout {
		t.Errorf("Input.LockTimeout = %q, want %q", gotTimeout, wantTimeout)
	}
}

func mustString(c *cobra.Command, name string) string {
	v, err := c.Flags().GetString(name)
	if err != nil {
		panic(err)
	}
	return v
}
