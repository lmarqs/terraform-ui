package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/plugins/untaint"
	"github.com/spf13/cobra"
)

// buildUntaintCommand wires `tfui untaint <addr>...` to the untaint plugin's
// typed Input. Positional args become Input.Addrs; the root-persistent --json
// value is copied into Input.JSON at RunE time.
func buildUntaintCommand(s *Session) *cobra.Command {
	var input untaint.Input
	c := &cobra.Command{
		Use:   "untaint <address>...",
		Short: "Remove taint mark from resources",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input.Addrs = args
			input.JSON = s.JSONStdout()
			return s.RunPlugin(cmd.Context(), "untaint", func(p sdk.Plugin) tea.Cmd {
				return p.(*untaint.Plugin).Activate(input)
			})
		},
	}
	return c
}
