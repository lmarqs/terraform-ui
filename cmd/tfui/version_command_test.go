package main

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	tfuiversion "github.com/lmarqs/terraform-ui/plugins/version"
	"github.com/spf13/cobra"
)

// TestBuildVersionCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput
// verifies the root-persistent --json value flows into Input.JSON.
func TestBuildVersionCommand_WhenSessionJSONStdout_ShouldPropagateIntoInput(t *testing.T) {
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
			var captured tfuiversion.Input
			c := buildVersionCommand(session)
			c.RunE = func(_ *cobra.Command, _ []string) error {
				captured = tfuiversion.Input{JSON: session.JSONStdout()}
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
