package main

import (
	"fmt"

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
			if err := requireApplyConfirmable(s, input.AutoApprove); err != nil {
				return err
			}
			return s.RunPlugin(cmd.Context(), "apply", func(p sdk.Plugin) tea.Cmd {
				return p.(*apply.Plugin).Activate(input)
			})
		},
	}
	c.Flags().BoolVar(&input.AutoApprove, "auto-approve", false, "Skip confirmation prompt")
	c.Flags().StringSliceVar(&input.Targets, "target", nil, "Resource targets (plans+applies in one shot)")
	return c
}

// requireApplyConfirmable mirrors terraform's own behavior: `terraform apply`
// with no plan file and no -auto-approve requires interactive approval, and
// errors out ("Apply not allowed for non-interactive use") when there is no
// terminal to approve from. terraform-exec hides this by always injecting
// -auto-approve, so we reproduce terraform's error at our boundary instead of
// silently applying (or hanging on a prompt that can never be answered).
func requireApplyConfirmable(s *Session, autoApprove bool) error {
	if autoApprove || s.macro.Active() || !s.SilentStderr() {
		return nil
	}
	return fmt.Errorf("apply not allowed for non-interactive use\n\nThe apply command requires interactive approval of the plan. To run apply in an automated environment, use the -auto-approve flag")
}
