package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/plugins/validate"
	"github.com/spf13/cobra"
)

// buildValidateCommand wires `tfui validate` to the validate plugin's typed
// Input. The root-persistent --json value flows through Session.JSONStdout
// into Input.JSON at RunE time.
func buildValidateCommand(s *Session) *cobra.Command {
	var input validate.Input
	c := &cobra.Command{
		Use:   "validate",
		Short: "Run terraform validate",
		RunE: func(cmd *cobra.Command, _ []string) error {
			input.JSON = s.JSONStdout()
			return s.RunPlugin(cmd.Context(), "validate", func(p sdk.Plugin) tea.Cmd {
				return p.(*validate.Plugin).Activate(input)
			})
		},
	}
	return c
}
