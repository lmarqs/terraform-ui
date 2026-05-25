package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiversion "github.com/lmarqs/terraform-ui/plugins/version"
	"github.com/spf13/cobra"
)

// buildVersionCommand wires `tfui version` to the version plugin's typed
// Input. The root-persistent --json value flows through Session.JSONStdout
// into Input.JSON at RunE time.
func buildVersionCommand(s *Session) *cobra.Command {
	var input tfuiversion.Input
	c := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			input.JSON = s.JSONStdout()
			return s.RunPlugin(cmd.Context(), "version", func(p sdk.Plugin) tea.Cmd {
				return p.(*tfuiversion.Plugin).Activate(input)
			})
		},
	}
	return c
}
