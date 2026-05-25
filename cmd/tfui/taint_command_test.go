package main

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/plugins/taint"
	"github.com/spf13/cobra"
)

// TestBuildTaintCommand_WhenAddressesGiven_ShouldBindIntoTypedInput verifies
// cobra hands positional args directly into taint.Input.Addrs at RunE time.
func TestBuildTaintCommand_WhenAddressesGiven_ShouldBindIntoTypedInput(t *testing.T) {
	session := &Session{cfg: config.Config{}}
	var captured taint.Input
	c := buildTaintCommand(session)
	c.RunE = func(_ *cobra.Command, args []string) error {
		captured = taint.Input{Addrs: args, JSON: session.JSONStdout()}
		return nil
	}

	c.SetArgs([]string{"aws_instance.web", "aws_instance.db"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(captured.Addrs) != 2 || captured.Addrs[0] != "aws_instance.web" {
		t.Errorf("Input.Addrs = %v, want [aws_instance.web aws_instance.db]", captured.Addrs)
	}
	if captured.JSON {
		t.Errorf("Input.JSON = true, want false (root --json not set)")
	}
}

// TestBuildTaintCommand_WhenNoArgs_ShouldFailArity verifies cobra enforces
// MinimumNArgs(1). The taint verb is meaningless without addresses; we let
// cobra produce the error rather than re-implementing arity checks.
func TestBuildTaintCommand_WhenNoArgs_ShouldFailArity(t *testing.T) {
	session := &Session{cfg: config.Config{}}
	c := buildTaintCommand(session)
	c.SilenceErrors = true
	c.SilenceUsage = true

	c.SetArgs([]string{})
	if err := c.Execute(); err == nil {
		t.Fatal("Execute() with zero args should fail (MinimumNArgs(1))")
	}
}

// TestBuildTaintCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput verifies
// the root-persistent --json value flows through Session.JSONStdout into
// Input.JSON at RunE time.
func TestBuildTaintCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput(t *testing.T) {
	session := &Session{cfg: config.Config{}, jsonStdout: true}
	var captured taint.Input
	c := buildTaintCommand(session)
	c.RunE = func(_ *cobra.Command, args []string) error {
		captured = taint.Input{Addrs: args, JSON: session.JSONStdout()}
		return nil
	}

	c.SetArgs([]string{"aws_instance.web"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !captured.JSON {
		t.Errorf("Input.JSON = false, want true (session.jsonStdout propagated)")
	}
}
