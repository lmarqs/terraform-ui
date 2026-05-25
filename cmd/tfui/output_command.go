package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/plugins/output"
	"github.com/spf13/cobra"
)

// buildOutputCommand wires `tfui output` to the output plugin's typed Input.
// The root-persistent --json value flows through Session.JSONStdout into
// Input.JSON at RunE time.
func buildOutputCommand(s *Session) *cobra.Command {
	var input output.Input
	c := &cobra.Command{
		Use:   "output",
		Short: "Show terraform outputs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			input.JSON = s.JSONStdout()
			return s.RunPlugin(cmd.Context(), "output", func(p sdk.Plugin) tea.Cmd {
				return p.(*output.Plugin).Activate(input)
			})
		},
	}
	return c
}
