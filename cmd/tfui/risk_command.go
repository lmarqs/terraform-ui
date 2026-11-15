package main

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
	"github.com/spf13/cobra"
)

func buildRiskCommand(s *Session) *cobra.Command {
	return &cobra.Command{
		Use:   "risk",
		Short: "Assess planned resource mutations with the selected risk policy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			input := tfuirisk.Input{JSON: s.JSONStdout(), ThresholdSet: cmd.Flags().Changed("threshold")}
			if input.ThresholdSet {
				input.Threshold = s.threshold
			}
			origin := cwd
			if s.rootCfg != nil {
				project, err := filepath.Rel(cwd, s.cfg.Dir)
				if err != nil {
					return err
				}
				input.Project = &project
				origin = s.cfg.Dir
			}
			input.Chdir, err = filepath.Rel(origin, effectiveWorkDir(s.cfg))
			if err != nil {
				return err
			}
			return s.RunPlugin(cmd.Context(), "risk", func(p sdk.Plugin) tea.Cmd { return p.(*tfuirisk.Plugin).ActivateInput(input) })
		},
	}
}
