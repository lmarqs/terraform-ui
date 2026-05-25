package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/scaffold"
	"github.com/spf13/cobra"
)

func buildScaffoldCommand(session *Session) *cobra.Command {
	var force, yes bool
	c := &cobra.Command{
		Use:   "scaffold",
		Short: "Generate tfui.hcl configuration",
		Long:  "Detect terraform project patterns and generate tfui.hcl in the working directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScaffold(session.cfg, force, yes)
		},
	}
	c.Flags().BoolVar(&force, "force", false, "Overwrite existing tfui.hcl")
	c.Flags().BoolVar(&yes, "yes", false, "Skip prompts, use detected defaults")
	return c
}

func runScaffold(cfg config.Config, force, yes bool) error {
	outPath := filepath.Join(cfg.Dir, config.HCLConfigFileName)
	if !force {
		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", outPath)
		}
	}

	var content string

	if yes || !hasTTY() {
		var err error
		content, err = scaffold.GenerateConfig(cfg.Dir)
		if err != nil {
			return fmt.Errorf("scaffold failed: %w", err)
		}
	} else {
		wizard := newScaffoldWizard(cfg.Dir)
		p := tea.NewProgram(wizard)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("scaffold failed: %w", err)
		}
		r := wizard.result()
		if r.Aborted {
			return nil
		}
		content = r.Content
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "Wrote %s\n", outPath)
	fmt.Print(content)
	return nil
}
