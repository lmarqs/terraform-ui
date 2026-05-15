package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/spf13/cobra"
)

// buildPluginCommands creates cobra subcommands for plugin actions.
func buildPluginCommands(cfg *config.Config) []*cobra.Command {
	return []*cobra.Command{
		buildStateCommands(cfg),
		buildWorkspaceCommands(cfg),
		buildValidateCommand(cfg),
		buildOutputCommand(cfg),
		buildForceUnlockCommand(cfg),
	}
}

func buildStateCommands(cfg *config.Config) *cobra.Command {
	stateCmd := &cobra.Command{
		Use:   "state",
		Short: "Terraform state operations",
	}

	rmCmd := &cobra.Command{
		Use:   "rm <address>",
		Short: "Remove a resource from state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			binary := cfg.TerraformBinary()
			svc := terraform.NewExecService(cfg.WorkingDir(), binary, nil)
			address := args[0]
			fmt.Fprintf(os.Stderr, "Removing %s from state...\n", address)
			if err := svc.StateRm(context.Background(), address); err != nil {
				return fmt.Errorf("state rm failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Removed %s\n", address)
			return nil
		},
	}

	mvCmd := &cobra.Command{
		Use:   "mv <source> <destination>",
		Short: "Move a resource in state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			binary := cfg.TerraformBinary()
			svc := terraform.NewExecService(cfg.WorkingDir(), binary, nil)
			fmt.Fprintf(os.Stderr, "Moving %s → %s...\n", args[0], args[1])
			if err := svc.StateMove(context.Background(), args[0], args[1]); err != nil {
				return fmt.Errorf("state mv failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Moved successfully\n")
			return nil
		},
	}

	importCmd := &cobra.Command{
		Use:   "import <address> <id>",
		Short: "Import an existing resource into state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			binary := cfg.TerraformBinary()
			svc := terraform.NewExecService(cfg.WorkingDir(), binary, nil)
			fmt.Fprintf(os.Stderr, "Importing %s (id: %s)...\n", args[0], args[1])
			if err := svc.Import(context.Background(), args[0], args[1]); err != nil {
				return fmt.Errorf("import failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Imported successfully\n")
			return nil
		},
	}

	taintCmd := &cobra.Command{
		Use:   "taint <address>",
		Short: "Mark a resource for recreation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			binary := cfg.TerraformBinary()
			svc := terraform.NewExecService(cfg.WorkingDir(), binary, nil)
			if err := svc.Taint(context.Background(), args[0]); err != nil {
				return fmt.Errorf("taint failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Tainted %s\n", args[0])
			return nil
		},
	}

	untaintCmd := &cobra.Command{
		Use:   "untaint <address>",
		Short: "Remove taint from a resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			binary := cfg.TerraformBinary()
			svc := terraform.NewExecService(cfg.WorkingDir(), binary, nil)
			if err := svc.Untaint(context.Background(), args[0]); err != nil {
				return fmt.Errorf("untaint failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Untainted %s\n", args[0])
			return nil
		},
	}

	stateCmd.AddCommand(rmCmd, mvCmd, importCmd, taintCmd, untaintCmd)
	return stateCmd
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

func buildValidateCommand(cfg *config.Config) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Run terraform validate",
		RunE: func(cmd *cobra.Command, args []string) error {
			binary := cfg.TerraformBinary()
			svc := terraform.NewExecService(cfg.WorkingDir(), binary, nil)
			diags, err := svc.Validate(context.Background())
			if err != nil {
				return fmt.Errorf("validate failed: %w", err)
			}
			if jsonOutput {
				return printValidateJSON(diags)
			}
			if len(diags) == 0 {
				fmt.Println("✓ Configuration is valid")
				return nil
			}
			for _, d := range diags {
				icon := "✗"
				if d.Severity == "warning" {
					icon = "⚠"
				}
				fmt.Printf("%s %s", icon, d.Summary)
				if d.File != "" {
					fmt.Printf(" (%s", d.File)
					if d.Line > 0 {
						fmt.Printf(":%d", d.Line)
					}
					fmt.Printf(")")
				}
				fmt.Println()
				if d.Detail != "" {
					fmt.Printf("  %s\n", d.Detail)
				}
			}
			if hasErrors(diags) {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func buildOutputCommand(cfg *config.Config) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "output [name]",
		Short: "Show terraform outputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			binary := cfg.TerraformBinary()
			svc := terraform.NewExecService(cfg.WorkingDir(), binary, nil)
			outputs, err := svc.Output(context.Background())
			if err != nil {
				return fmt.Errorf("output failed: %w", err)
			}
			if len(args) == 1 {
				name := args[0]
				o, ok := outputs[name]
				if !ok {
					return fmt.Errorf("output %q not found", name)
				}
				if o.Sensitive {
					fmt.Println("(sensitive)")
				} else {
					fmt.Printf("%v\n", o.Value)
				}
				return nil
			}
			for name, o := range outputs {
				val := "(sensitive)"
				if !o.Sensitive {
					val = fmt.Sprintf("%v", o.Value)
				}
				if jsonOutput {
					fmt.Printf("{\"name\":%q,\"value\":%q,\"sensitive\":%v}\n", name, val, o.Sensitive)
				} else {
					fmt.Printf("%s = %s\n", name, val)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func hasErrors(diags []sdk.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == "error" {
			return true
		}
	}
	return false
}

func printValidateJSON(diags []sdk.Diagnostic) error {
	errorCount := 0
	warningCount := 0
	for _, d := range diags {
		if d.Severity == "error" {
			errorCount++
		} else {
			warningCount++
		}
	}

	type diagJSON struct {
		Severity string `json:"severity"`
		Summary  string `json:"summary"`
		Detail   string `json:"detail,omitempty"`
		File     string `json:"file,omitempty"`
		Line     int    `json:"line,omitempty"`
	}

	type validateJSON struct {
		Valid        bool       `json:"valid"`
		ErrorCount   int        `json:"error_count"`
		WarningCount int        `json:"warning_count"`
		Diagnostics  []diagJSON `json:"diagnostics"`
	}

	out := validateJSON{
		Valid:        errorCount == 0,
		ErrorCount:   errorCount,
		WarningCount: warningCount,
		Diagnostics:  make([]diagJSON, 0, len(diags)),
	}
	for _, d := range diags {
		out.Diagnostics = append(out.Diagnostics, diagJSON{
			Severity: d.Severity,
			Summary:  d.Summary,
			Detail:   d.Detail,
			File:     d.File,
			Line:     d.Line,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encoding validate JSON: %w", err)
	}
	if errorCount > 0 {
		os.Exit(1)
	}
	return nil
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
