package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiimport "github.com/lmarqs/terraform-ui/plugins/import"
	"github.com/spf13/cobra"
)

// buildImportCommand wires `tfui import <addr> <id>` to the import plugin's
// typed Input. Cobra's ExactArgs(2) validates the positional arity; both args
// are copied into Input.Addr / Input.ID at RunE time. The root-persistent
// --json value flows through Session.JSONStdout into Input.JSON.
func buildImportCommand(s *Session) *cobra.Command {
	var input tfuiimport.Input
	c := &cobra.Command{
		Use:   "import <address> <id>",
		Short: "Import existing infrastructure into terraform state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input.Addr = args[0]
			input.ID = args[1]
			input.JSON = s.JSONStdout()
			return s.RunPlugin(cmd.Context(), "import", func(p sdk.Plugin) tea.Cmd {
				return p.(*tfuiimport.Plugin).Activate(input)
			})
		},
	}
	return c
}
