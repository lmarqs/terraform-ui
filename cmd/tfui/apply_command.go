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

// requireApplyConfirmable rejects an apply that has no way to confirm. Apply
// without --auto-approve relies on the TUI confirmation prompt; when stderr is
// not a terminal (CI / piped) and no macro tape is driving the session, there
// is no prompt to answer. Rather than hang or apply silently, fail fast and
// direct the user to --auto-approve.
func requireApplyConfirmable(s *Session, autoApprove bool) error {
	if autoApprove || s.macro.Active() || !s.SilentStderr() {
		return nil
	}
	return fmt.Errorf("apply needs confirmation but stderr is not a terminal; re-run with --auto-approve for non-interactive apply")
}
