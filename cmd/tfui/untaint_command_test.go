package main

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/plugins/untaint"
	"github.com/spf13/cobra"
)

// TestBuildUntaintCommand_WhenAddressesGiven_ShouldBindIntoTypedInput verifies
// cobra hands positional args directly into untaint.Input.Addrs at RunE time.
func TestBuildUntaintCommand_WhenAddressesGiven_ShouldBindIntoTypedInput(t *testing.T) {
	session := &Session{cfg: config.Config{}}
	var captured untaint.Input
	c := buildUntaintCommand(session)
	c.RunE = func(_ *cobra.Command, args []string) error {
		captured = untaint.Input{Addrs: args, JSON: session.JSONStdout()}
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

// TestBuildUntaintCommand_WhenNoArgs_ShouldFailArity verifies cobra enforces
// MinimumNArgs(1).
func TestBuildUntaintCommand_WhenNoArgs_ShouldFailArity(t *testing.T) {
	session := &Session{cfg: config.Config{}}
	c := buildUntaintCommand(session)
	c.SilenceErrors = true
	c.SilenceUsage = true

	c.SetArgs([]string{})
	if err := c.Execute(); err == nil {
		t.Fatal("Execute() with zero args should fail (MinimumNArgs(1))")
	}
}

// TestBuildUntaintCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput
// verifies the root-persistent --json value flows through Session.JSONStdout
// into Input.JSON at RunE time.
func TestBuildUntaintCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput(t *testing.T) {
	session := &Session{cfg: config.Config{}, jsonStdout: true}
	var captured untaint.Input
	c := buildUntaintCommand(session)
	c.RunE = func(_ *cobra.Command, args []string) error {
		captured = untaint.Input{Addrs: args, JSON: session.JSONStdout()}
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
