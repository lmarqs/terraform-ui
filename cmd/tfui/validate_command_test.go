package main

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/plugins/validate"
	"github.com/spf13/cobra"
)

// TestBuildValidateCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput
// verifies the root-persistent --json value flows into Input.JSON at RunE time.
func TestBuildValidateCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput(t *testing.T) {
	tests := []struct {
		name       string
		jsonStdout bool
		want       bool
	}{
		{"NoJSON", false, false},
		{"WithJSON", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{cfg: config.Config{}, jsonStdout: tt.jsonStdout}
			var captured validate.Input
			c := buildValidateCommand(session)
			c.RunE = func(_ *cobra.Command, _ []string) error {
				captured = validate.Input{JSON: session.JSONStdout()}
				return nil
			}
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if captured.JSON != tt.want {
				t.Errorf("Input.JSON = %v, want %v", captured.JSON, tt.want)
			}
		})
	}
}
