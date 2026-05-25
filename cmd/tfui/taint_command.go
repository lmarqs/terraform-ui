package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/plugins/taint"
	"github.com/spf13/cobra"
)

// buildTaintCommand wires `tfui taint <addr>...` to the taint plugin's typed
// Input. Positional args become Input.Addrs; the root-persistent --json value
// is copied into Input.JSON at RunE time.
func buildTaintCommand(s *Session) *cobra.Command {
	var input taint.Input
	c := &cobra.Command{
		Use:   "taint <address>...",
		Short: "Mark resources for recreation on next apply",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input.Addrs = args
			input.JSON = s.JSONStdout()
			return s.RunPlugin(cmd.Context(), "taint", func(p sdk.Plugin) tea.Cmd {
				return p.(*taint.Plugin).Activate(input)
			})
		},
	}
	return c
}
