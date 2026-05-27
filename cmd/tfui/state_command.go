package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuistate "github.com/lmarqs/terraform-ui/plugins/state"
	"github.com/spf13/cobra"
)

func buildStateCommand(s *Session) *cobra.Command {
	c := &cobra.Command{
		Use:   "state [rm|mv|taint|untaint] [args...]",
		Short: "Terraform state operations",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			input := tfuistate.Input{JSON: s.JSONStdout()}
			if len(args) > 0 {
				input.Subcommand = args[0]
				input.Targets = args[1:]
			}
			return s.RunPlugin(cmd.Context(), "state", func(p sdk.Plugin) tea.Cmd {
				return p.(*tfuistate.Plugin).Activate(input)
			})
		},
	}
	return c
}
