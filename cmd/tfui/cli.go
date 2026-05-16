package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/spf13/cobra"
)

// buildPluginCommands creates cobra subcommands for imperative operations
// that remain as direct CLI commands (not standalone TUI).
func buildPluginCommands(cfg *config.Config) []*cobra.Command {
	return []*cobra.Command{
		buildWorkspaceCommands(cfg),
		buildForceUnlockCommand(cfg),
	}
}


func buildWorkspaceCommands(cfg *config.Config) *cobra.Command {
	wsCmd := &cobra.Command{
		Use:   "workspace",
		Short: "Terraform workspace operations",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current workspace name",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := terraform.NewExecService(cfg.WorkingDir(), cfg.TerraformBinary(), nil)
			name, err := svc.Workspace(context.Background())
			if err != nil {
				return fmt.Errorf("workspace show failed: %w", err)
			}
			fmt.Println(name)
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := terraform.NewExecService(cfg.WorkingDir(), cfg.TerraformBinary(), nil)
			workspaces, err := svc.WorkspaceList(context.Background())
			if err != nil {
				return fmt.Errorf("workspace list failed: %w", err)
			}
			current, err := svc.Workspace(context.Background())
			if err != nil {
				return fmt.Errorf("workspace list failed: %w", err)
			}
			for _, ws := range workspaces {
				if ws == current {
					fmt.Printf("* %s\n", ws)
				} else {
					fmt.Printf("  %s\n", ws)
				}
			}
			return nil
		},
	}

	selectCmd := &cobra.Command{
		Use:   "select <name>",
		Short: "Select a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := terraform.NewExecService(cfg.WorkingDir(), cfg.TerraformBinary(), nil)
			if err := svc.WorkspaceSelect(context.Background(), args[0]); err != nil {
				return fmt.Errorf("workspace select failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Switched to workspace %q.\n", args[0])
			return nil
		},
	}

	var newLock *bool
	var newLockTimeout string
	newCmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := terraform.NewExecService(cfg.WorkingDir(), cfg.TerraformBinary(), nil)
			opts := sdk.WorkspaceNewOptions{LockTimeout: newLockTimeout}
			if cmd.Flags().Changed("lock") {
				opts.Lock = newLock
			}
			if err := svc.WorkspaceNew(context.Background(), args[0], opts); err != nil {
				return fmt.Errorf("workspace new failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Created and switched to workspace %q!\n", args[0])
			return nil
		},
	}
	newLock = newCmd.Flags().Bool("lock", true, "Lock state during operation")
	newCmd.Flags().StringVar(&newLockTimeout, "lock-timeout", "", "Duration to wait for a state lock")

	var deleteForce bool
	var deleteLock *bool
	var deleteLockTimeout string
	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := terraform.NewExecService(cfg.WorkingDir(), cfg.TerraformBinary(), nil)
			opts := sdk.WorkspaceDeleteOptions{
				Force:       deleteForce,
				LockTimeout: deleteLockTimeout,
			}
			if cmd.Flags().Changed("lock") {
				opts.Lock = deleteLock
			}
			if err := svc.WorkspaceDelete(context.Background(), args[0], opts); err != nil {
				return fmt.Errorf("workspace delete failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Deleted workspace %q!\n", args[0])
			return nil
		},
	}
	deleteCmd.Flags().BoolVar(&deleteForce, "force", false, "Force deletion of a non-empty workspace")
	deleteLock = deleteCmd.Flags().Bool("lock", true, "Lock state during operation")
	deleteCmd.Flags().StringVar(&deleteLockTimeout, "lock-timeout", "", "Duration to wait for a state lock")

	wsCmd.AddCommand(showCmd, listCmd, selectCmd, newCmd, deleteCmd)
	return wsCmd
}


func buildForceUnlockCommand(cfg *config.Config) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "force-unlock <lock-id>",
		Short: "Remove a terraform state lock",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lockID := args[0]
			svc := terraform.NewExecService(cfg.WorkingDir(), cfg.TerraformBinary(), nil)

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

// resolveProjectDir resolves the --project flag value to an absolute directory path.
// Accepts:
//   - A directory path: resolved to absolute
//   - A path to tfui.yaml: uses its parent directory
//   - A path ending in tfui.yaml that doesn't exist yet: uses parent directory
func resolveProjectDir(project string) string {
	dir := project

	if dir == "" || dir == "." {
		dir = "."
	} else {
		// If it points to a file (or looks like it points to tfui.yaml), use parent dir
		base := filepath.Base(dir)
		if strings.EqualFold(base, config.HCLConfigFileName) {
			dir = filepath.Dir(dir)
		} else if info, err := os.Stat(dir); err == nil && !info.IsDir() {
			// If it's an existing file (not a directory), use parent
			dir = filepath.Dir(dir)
		}
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}
