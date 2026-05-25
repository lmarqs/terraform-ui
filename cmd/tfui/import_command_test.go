package main

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	tfuiimport "github.com/lmarqs/terraform-ui/plugins/import"
	"github.com/spf13/cobra"
)

// TestBuildImportCommand_WhenAddrAndIDGiven_ShouldBindIntoTypedInput verifies
// cobra hands the two positional args directly into tfuiimport.Input.Addr / ID.
func TestBuildImportCommand_WhenAddrAndIDGiven_ShouldBindIntoTypedInput(t *testing.T) {
	session := &Session{cfg: config.Config{}}
	var captured tfuiimport.Input
	c := buildImportCommand(session)
	c.RunE = func(_ *cobra.Command, args []string) error {
		captured = tfuiimport.Input{Addr: args[0], ID: args[1], JSON: session.JSONStdout()}
		return nil
	}

	c.SetArgs([]string{"aws_instance.web", "i-1234567890"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if captured.Addr != "aws_instance.web" {
		t.Errorf("Input.Addr = %q, want aws_instance.web", captured.Addr)
	}
	if captured.ID != "i-1234567890" {
		t.Errorf("Input.ID = %q, want i-1234567890", captured.ID)
	}
	if captured.JSON {
		t.Errorf("Input.JSON = true, want false (root --json not set)")
	}
}

// TestBuildImportCommand_WhenWrongArity_ShouldFail verifies cobra enforces
// ExactArgs(2). One or three args must reject before RunE fires.
func TestBuildImportCommand_WhenWrongArity_ShouldFail(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"ZeroArgs", []string{}},
		{"OneArg", []string{"aws_instance.web"}},
		{"ThreeArgs", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{cfg: config.Config{}}
			c := buildImportCommand(session)
			c.SilenceErrors = true
			c.SilenceUsage = true
			c.SetArgs(tt.args)
			if err := c.Execute(); err == nil {
				t.Fatal("Execute() with wrong arity should fail")
			}
		})
	}
}

// TestBuildImportCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput
// verifies the root-persistent --json value flows into Input.JSON.
func TestBuildImportCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput(t *testing.T) {
	session := &Session{cfg: config.Config{}, jsonStdout: true}
	var captured tfuiimport.Input
	c := buildImportCommand(session)
	c.RunE = func(_ *cobra.Command, args []string) error {
		captured = tfuiimport.Input{Addr: args[0], ID: args[1], JSON: session.JSONStdout()}
		return nil
	}

	c.SetArgs([]string{"aws_instance.web", "i-abc"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !captured.JSON {
		t.Errorf("Input.JSON = false, want true (session.jsonStdout propagated)")
	}
}
