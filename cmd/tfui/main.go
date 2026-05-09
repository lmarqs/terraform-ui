package main

import (
	"fmt"
	"os"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var version = "1.0.0-dev"

func main() {
	var cfg config.Config

	rootCmd := &cobra.Command{
		Use:   "tfui",
		Short: "Terminal UI for Terraform operations",
		Long:  "terraform-ui provides animated terminal feedback for terraform plan and apply operations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(cfg)
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfg.Dir, "dir", ".", "Working directory for terraform operations")
	rootCmd.PersistentFlags().StringVar(&cfg.TerraformBinary, "terraform-bin", "terraform", "Path to terraform binary")

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Run terraform plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlan(cfg)
		},
	}
	planCmd.Flags().StringVar(&cfg.Mode, "mode", "progress", "UI mode: silent, spinner, progress, agent")
	planCmd.Flags().StringSliceVar(&cfg.Targets, "target", nil, "Resource targets for plan")

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Run terraform apply",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cfg)
		},
	}
	applyCmd.Flags().StringVar(&cfg.Mode, "mode", "progress", "UI mode: silent, spinner, progress, agent")
	applyCmd.Flags().StringSliceVar(&cfg.Targets, "target", nil, "Resource targets for apply")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("tfui %s\n", version)
		},
	}

	rootCmd.AddCommand(planCmd, applyCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI(cfg config.Config) error {
	app := ui.NewApp(cfg)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func runPlan(cfg config.Config) error {
	fmt.Printf("Running plan in %s mode for directory: %s\n", cfg.Mode, cfg.Dir)
	if len(cfg.Targets) > 0 {
		fmt.Printf("Targets: %v\n", cfg.Targets)
	}
	return nil
}

func runApply(cfg config.Config) error {
	fmt.Printf("Running apply in %s mode for directory: %s\n", cfg.Mode, cfg.Dir)
	if len(cfg.Targets) > 0 {
		fmt.Printf("Targets: %v\n", cfg.Targets)
	}
	return nil
}
