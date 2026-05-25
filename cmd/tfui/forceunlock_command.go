package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lmarqs/terraform-ui/internal/config"
	tfexec "github.com/lmarqs/terraform-ui/internal/terraform/exec"
	"github.com/spf13/cobra"
)

func buildForceUnlockCommand(cfg *config.Config) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "force-unlock <lock-id>",
		Short: "Remove a terraform state lock",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			lockID := args[0]
			svc := tfexec.NewExecService(cfg.WorkingDir(), cfg.TerraformBinary(), nil)

			if !force && hasTTY() {
				fmt.Fprintf(os.Stderr, "Terraform state has been locked. Force-unlock will remove the lock.\n")
				fmt.Fprintf(os.Stderr, "This may cause data corruption if another operation is in progress.\n\n")
				fmt.Fprintf(os.Stderr, "  Lock ID: %s\n\n", lockID)
				fmt.Fprintf(os.Stderr, "Do you want to continue? (y/n): ")
				var answer string
				_, _ = fmt.Scanln(&answer)
				if answer != "y" && answer != "yes" {
					return fmt.Errorf("force-unlock cancelled")
				}
			}

			fmt.Fprintf(os.Stderr, "Removing lock %s...\n", lockID)
			if err := svc.ForceUnlock(context.Background(), lockID); err != nil {
				return fmt.Errorf("force-unlock failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Lock %s removed successfully.\n", lockID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
