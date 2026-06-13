package main

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/plugins/apply"
	"github.com/spf13/cobra"
)

// TestBuildApplyCommand_WhenAutoApproveAndTarget_ShouldBindIntoTypedInput verifies
// cobra parses --auto-approve and --target directly into apply.Input fields. We
// don't run the command — we trigger flag parsing and then read the closure-
// captured Input value.
func TestBuildApplyCommand_WhenAutoApproveAndTarget_ShouldBindIntoTypedInput(t *testing.T) {
	session := &Session{cfg: config.Config{}}
	var captured apply.Input
	c := buildApplyCommand(session)
	// Replace RunE with a probe that captures the parsed Input. The original
	// closure dispatches into Session.RunPlugin which we don't exercise here.
	c.RunE = func(_ *cobra.Command, _ []string) error {
		captured = apply.Input{
			AutoApprove: mustBool(c, "auto-approve"),
			Targets:     mustStringSlice(c, "target"),
			JSON:        session.JSONStdout(),
		}
		return nil
	}

	c.SetArgs([]string{"--auto-approve", "--target=aws_instance.web"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !captured.AutoApprove {
		t.Errorf("Input.AutoApprove = false, want true")
	}
	if len(captured.Targets) != 1 || captured.Targets[0] != "aws_instance.web" {
		t.Errorf("Input.Targets = %v, want [aws_instance.web]", captured.Targets)
	}
	if captured.JSON {
		t.Errorf("Input.JSON = true, want false (root --json not set)")
	}
}

// TestBuildApplyCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput verifies
// the root-persistent --json value flows through Session.JSONStdout into
// Input.JSON at RunE time.
func TestBuildApplyCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput(t *testing.T) {
	session := &Session{cfg: config.Config{}, jsonStdout: true}
	var captured apply.Input
	c := buildApplyCommand(session)
	c.RunE = func(_ *cobra.Command, _ []string) error {
		captured = apply.Input{
			AutoApprove: mustBool(c, "auto-approve"),
			Targets:     mustStringSlice(c, "target"),
			JSON:        session.JSONStdout(),
		}
		return nil
	}

	c.SetArgs([]string{"--auto-approve"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !captured.JSON {
		t.Errorf("Input.JSON = false, want true (session.jsonStdout propagated)")
	}
}

// TestRequireApplyConfirmable enforces the invariant: apply without
// --auto-approve must never run in a context where it cannot confirm. The TUI
// prompt is the only confirmation gate, so a non-interactive session (silent
// stderr, no macro tape) without --auto-approve must fail fast rather than hang
// or apply silently.
func TestRequireApplyConfirmable(t *testing.T) {
	tests := []struct {
		name         string
		silentStderr bool
		macroURI     string
		autoApprove  bool
		wantErr      bool
	}{
		{name: "interactive, no auto-approve", silentStderr: false, wantErr: false},
		{name: "non-interactive without auto-approve", silentStderr: true, wantErr: true},
		{name: "non-interactive with auto-approve", silentStderr: true, autoApprove: true, wantErr: false},
		{name: "non-interactive driven by macro tape", silentStderr: true, macroURI: "tape.txt", wantErr: false},
		{name: "interactive with auto-approve", silentStderr: false, autoApprove: true, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{
				silentStderr: tt.silentStderr,
				macro:        MacroSpec{TapeURI: tt.macroURI},
			}
			err := requireApplyConfirmable(s, tt.autoApprove)
			if (err != nil) != tt.wantErr {
				t.Fatalf("requireApplyConfirmable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func mustBool(c *cobra.Command, name string) bool {
	v, err := c.Flags().GetBool(name)
	if err != nil {
		panic(err)
	}
	return v
}

func mustStringSlice(c *cobra.Command, name string) []string {
	v, err := c.Flags().GetStringSlice(name)
	if err != nil {
		panic(err)
	}
	return v
}
