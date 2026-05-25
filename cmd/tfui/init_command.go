package main

import (
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiinit "github.com/lmarqs/terraform-ui/plugins/init"
	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
)

func buildInitCommand(s *Session) *cobra.Command {
	var input tfuiinit.Input
	var backend bool

	c := &cobra.Command{
		Use:   "init",
		Short: "Run terraform init",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("backend") {
				input.Backend = &backend
			}
			return s.RunPlugin(cmd.Context(), "init", func(p sdk.Plugin) tea.Cmd {
				return p.(*tfuiinit.Plugin).Activate(input)
			})
		},
	}
	c.Flags().BoolVar(&input.Upgrade, "upgrade", false, "Upgrade modules and plugins")
	c.Flags().BoolVar(&input.Reconfigure, "reconfigure", false, "Reconfigure backend")
	c.Flags().BoolVar(&backend, "backend", true, "Configure backend (--backend=false to skip)")
	c.Flags().StringArrayVar(&input.BackendConfig, "backend-config", nil, "Backend configuration values")
	return c
}
