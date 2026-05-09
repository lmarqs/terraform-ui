package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/spf13/cobra"
)

// buildPluginCommands creates cobra subcommands for plugin actions.
func buildPluginCommands(cfg *config.Config) []*cobra.Command {
	return []*cobra.Command{
		buildStateCommands(cfg),
		buildValidateCommand(cfg),
		buildOutputCommand(cfg),
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
			svc := terraform.NewService(cfg.Dir, binary)
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
			svc := terraform.NewService(cfg.Dir, binary)
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
			svc := terraform.NewService(cfg.Dir, binary)
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
			svc := terraform.NewService(cfg.Dir, binary)
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
			svc := terraform.NewService(cfg.Dir, binary)
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

func buildValidateCommand(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Run terraform validate",
		RunE: func(cmd *cobra.Command, args []string) error {
			binary := cfg.TerraformBinary()
			svc := terraform.NewService(cfg.Dir, binary)
			diags, err := svc.Validate(context.Background())
			if err != nil {
				return fmt.Errorf("validate failed: %w", err)
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
}

func buildOutputCommand(cfg *config.Config) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "output [name]",
		Short: "Show terraform outputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			binary := cfg.TerraformBinary()
			svc := terraform.NewService(cfg.Dir, binary)
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
