package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/plugins/apply"
	"github.com/spf13/cobra"
)

// buildApplyCommand wires `tfui apply` to the apply plugin's typed Input.
// cobra binds --auto-approve and --target directly into apply.Input fields;
// the root-persistent --json value is copied into Input.JSON at RunE time.
func buildApplyCommand(s *Session) *cobra.Command {
	var input apply.Input
	c := &cobra.Command{
		Use:   "apply",
		Short: "Run terraform apply (with plan file or directly with targets)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			input.JSON = s.JSONStdout()
			return s.RunPlugin(cmd.Context(), "apply", func(p sdk.Plugin) tea.Cmd {
				return p.(*apply.Plugin).Activate(input)
			})
		},
	}
	c.Flags().BoolVar(&input.AutoApprove, "auto-approve", false, "Skip confirmation prompt")
	c.Flags().StringSliceVar(&input.Targets, "target", nil, "Resource targets (plans+applies in one shot)")
	return c
}
