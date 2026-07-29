package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/spf13/cobra"
)

func buildWorkspaceCommand(cfg *config.Config) *cobra.Command {
	wsCmd := &cobra.Command{
		Use:   "workspace",
		Short: "Terraform workspace operations",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current workspace name",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			svc, err := imperativeService(cfg)
			if err != nil {
				return err
			}
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
		RunE: func(_ *cobra.Command, _ []string) error {
			svc, err := imperativeService(cfg)
			if err != nil {
				return err
			}
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
		RunE: func(_ *cobra.Command, args []string) error {
			svc, err := imperativeService(cfg)
			if err != nil {
				return err
			}
			if err := svc.WorkspaceSelect(context.Background(), args[0]); err != nil {
				return fmt.Errorf("workspace select failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Switched to workspace %q.\n", args[0])
			return nil
		},
	}

	var newLock bool
	var newLockTimeout string
	newCmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := imperativeService(cfg)
			if err != nil {
				return err
			}
			opts := sdk.WorkspaceNewOptions{
				Lock:        lockModeFrom(cmd, newLock),
				LockTimeout: sdk.LockTimeout(newLockTimeout),
			}
			if err := svc.WorkspaceNew(context.Background(), args[0], opts); err != nil {
				return fmt.Errorf("workspace new failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Created and switched to workspace %q!\n", args[0])
			return nil
		},
	}
	bindLockFlags(newCmd, &newLock, &newLockTimeout)

	var deleteForce bool
	var deleteLock bool
	var deleteLockTimeout string
	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := imperativeService(cfg)
			if err != nil {
				return err
			}
			opts := sdk.WorkspaceDeleteOptions{
				Force:       deleteForce,
				Lock:        lockModeFrom(cmd, deleteLock),
				LockTimeout: sdk.LockTimeout(deleteLockTimeout),
			}
			if err := svc.WorkspaceDelete(context.Background(), args[0], opts); err != nil {
				return fmt.Errorf("workspace delete failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Deleted workspace %q!\n", args[0])
			return nil
		},
	}
	deleteCmd.Flags().BoolVar(&deleteForce, "force", false, "Force deletion of a non-empty workspace")
	bindLockFlags(deleteCmd, &deleteLock, &deleteLockTimeout)

	wsCmd.AddCommand(showCmd, listCmd, selectCmd, newCmd, deleteCmd)
	return wsCmd
}
