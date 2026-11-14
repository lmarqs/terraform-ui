package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/plugins/apply"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	"github.com/spf13/cobra"
)

// buildApplyCommand wires `tfui apply`. With --auto-approve it activates the
// apply plugin with its typed Input (cobra binds --auto-approve and --target;
// the root-persistent --json value is copied into Input.JSON at RunE time).
// Without it, the run enters through the plan plugin, which receives the
// targets and lock overrides and hands the saved plan over to apply.
func buildApplyCommand(s *Session) *cobra.Command {
	var input apply.Input
	var lock bool
	c := &cobra.Command{
		Use:   "apply",
		Short: "Run terraform apply (with plan file or directly with targets)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			input.JSON = s.JSONStdout()
			input.Lock = lockModeFrom(cmd, lock)
			if err := requireApplyConfirmable(s, input.AutoApprove); err != nil {
				return err
			}
			if input.AutoApprove {
				return s.RunPlugin(cmd.Context(), "apply", func(p sdk.Plugin) tea.Cmd {
					return p.(*apply.Plugin).Activate(applyInputForAutoApprove(input, s.cfg))
				})
			}
			planInput := planInputForApply(input)
			return s.RunPluginFrom(cmd.Context(), "apply", "plan", func(p sdk.Plugin) tea.Cmd {
				return p.(*tfuiplan.Plugin).Activate(planInput)
			})
		},
	}
	c.Flags().BoolVar(&input.AutoApprove, "auto-approve", false, "Skip confirmation prompt")
	c.Flags().StringArrayVar(&input.Targets, "target", nil, "Resource target, repeatable and taken verbatim (plans+applies in one shot)")
	bindLockFlags(c, &lock, (*string)(&input.LockTimeout))
	return c
}

// planInputForApply maps the interactive `tfui apply` flags onto the plan run
// that produces the plan file apply consumes. Without --target, plan applies
// its own configured `targets`.
func planInputForApply(input apply.Input) tfuiplan.Input {
	return tfuiplan.Input{Targets: input.Targets, Lock: input.Lock, LockTimeout: input.LockTimeout}
}

// applyInputForAutoApprove gives a one-shot apply, which has no plan to
// review, the scope a reviewed apply would have: --target, else plan's
// configured `targets`. One list sets the default scope (ADR-0019).
func applyInputForAutoApprove(input apply.Input, cfg config.Config) apply.Input {
	if len(input.Targets) == 0 {
		input.Targets = sdk.ConfigStrings(cfg.Plugins["plan"].Options, "targets", nil)
	}
	return input
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
