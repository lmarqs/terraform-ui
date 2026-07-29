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
	// StringArrayVar, not StringSliceVar: a slice flag parses each value as a CSV
	// record, which rejects every quoted index key (module.a["x y"]) and leaves an
	// address containing a comma inexpressible. Repeat the flag per address, the
	// way terraform's own -target works.
	c.Flags().StringArrayVar(&input.Targets, "target", nil, "Resource target for plan (repeatable, taken verbatim)")
	return c
}
