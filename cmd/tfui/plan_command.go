package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	"github.com/spf13/cobra"
)

func buildPlanCommand(s *Session) *cobra.Command {
	var input tfuiplan.Input

	c := &cobra.Command{
		Use:   "plan",
		Short: "Run terraform plan",
		RunE: func(cmd *cobra.Command, _ []string) error {
			input.JSON = s.JSONStdout()
			return s.RunPlugin(cmd.Context(), "plan", func(p sdk.Plugin) tea.Cmd {
				return p.(*tfuiplan.Plugin).Activate(input)
			})
		},
	}
	c.Flags().StringSliceVar(&input.Targets, "target", nil, "Resource targets for plan")
	return c
}
