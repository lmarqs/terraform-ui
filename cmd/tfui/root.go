package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/spf13/cobra"
)

func buildRoot() (*cobra.Command, *Session, *[]string) {
	var cfg config.Config
	var configOverrides []string
	var sources PreseedSources
	var macroSpec MacroSpec
	var ciMode bool
	var jsonStdout bool
	var extraArgs []string

	session := &Session{effects: DefaultEffects()}

	rootCmd := &cobra.Command{
		Use:          "tfui",
		Short:        "Terminal UI for Terraform operations",
		Long:         "terraform-ui provides animated terminal feedback for terraform plan and apply operations.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg.Dir = resolveProjectDir(cfg.Dir)
			if info, err := os.Stat(cfg.Dir); err != nil {
				return fmt.Errorf("project directory not found: %s", cfg.Dir)
			} else if !info.IsDir() {
				return fmt.Errorf("project path is not a directory: %s", cfg.Dir)
			}
			cfg.ApplyOverrides(configOverrides)

			var rootCfg *config.RootConfig
			var err error
			rootCfg, err = config.LoadRoot(cfg.Dir)
			if err != nil {
				var notFound *config.ConfigNotFoundError
				if !errors.As(err, &notFound) {
					return fmt.Errorf("%w\n\nhint: check HCL syntax in tfui.hcl", err)
				}
			}
			if rootCfg != nil {
				if rootCfg.Terraform.Bin != "" && cfg.Terraform.Bin == "" {
					cfg.Terraform.Bin = rootCfg.Terraform.Bin
				}
				resolved := config.Resolve(rootCfg, nil, cfg.Workspace)
				cfg.VarFiles = resolved.VarFiles()
				cfg.Vars = resolved.Vars()
			}

			binary := cfg.TerraformBinary()
			logging.Init(macroSpec.RecordDir != "", version, cfg.Dir, binary, macroSpec.RecordDir)

			cfg.ExtraArgs = extraArgs
			cfg.PreloadedData = !sources.Empty()
			session.cfg = cfg
			session.rootCfg = rootCfg
			session.sources = sources
			session.macro = macroSpec
			session.ciMode = ciMode
			session.jsonStdout = jsonStdout
			session.silentStderr = session.resolveSilentStderr()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.Run()
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfg.Dir, "project", ".", "Project root directory (where tfui.hcl lives)")
	rootCmd.PersistentFlags().StringVar(&cfg.Terraform.Bin, "terraform-bin", "", "Path to terraform/tofu binary")
	rootCmd.PersistentFlags().StringArrayVar(&configOverrides, "config", nil, "Override config values (key=value, e.g. --config logger.dir=/tmp/logs --config terraform.bin=tofu)")
	rootCmd.PersistentFlags().StringVar(&sources.Plan, "plan", "", "Pre-seed plan data from file (./path, /path, file://) or - for stdin")
	rootCmd.PersistentFlags().StringVar(&sources.State, "state", "", "Pre-seed state data from file (./path, /path, file://) or - for stdin")
	rootCmd.PersistentFlags().StringVar(&macroSpec.TapeURI, "macro", "", "Run a macro tape file (headless TUI recording)")
	rootCmd.PersistentFlags().StringVar(&macroSpec.RecordDir, "record", "", "Record session frames and tape to directory")
	rootCmd.PersistentFlags().StringVar(&cfg.Chdir, "chdir", "", "Select member directory (validated against member blocks in project mode)")
	rootCmd.PersistentFlags().BoolVar(&ciMode, "ci", false, "Suppress TUI (CI-friendly output)")
	rootCmd.PersistentFlags().BoolVar(&jsonStdout, "json", false, "Output JSON (terraform-compatible)")

	return rootCmd, session, &extraArgs
}
