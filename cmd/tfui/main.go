package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/internal/macro"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/scaffold"
	"github.com/lmarqs/terraform-ui/internal/source"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiapply "github.com/lmarqs/terraform-ui/plugins/apply"
	tfuiblastradius "github.com/lmarqs/terraform-ui/plugins/blastradius"
	tfuichdir "github.com/lmarqs/terraform-ui/plugins/chdir"
	tfuiconsole "github.com/lmarqs/terraform-ui/plugins/console"
	tfuicontext "github.com/lmarqs/terraform-ui/plugins/context"
	tfuiforceunlock "github.com/lmarqs/terraform-ui/plugins/forceunlock"
	tfuiimport "github.com/lmarqs/terraform-ui/plugins/import"
	tfuiinit "github.com/lmarqs/terraform-ui/plugins/init"
	tfuioutput "github.com/lmarqs/terraform-ui/plugins/output"
	tfuiphantom "github.com/lmarqs/terraform-ui/plugins/phantom"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
	tfuistate "github.com/lmarqs/terraform-ui/plugins/state"
	tfuitaint "github.com/lmarqs/terraform-ui/plugins/taint"
	tfuiuntaint "github.com/lmarqs/terraform-ui/plugins/untaint"
	tfuivalidate "github.com/lmarqs/terraform-ui/plugins/validate"
	tfuiversion "github.com/lmarqs/terraform-ui/plugins/version"
	tfuiworkspace "github.com/lmarqs/terraform-ui/plugins/workspace"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var version string

func init() {
	if version == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
				return
			}
		}
		version = "0.0.0-SNAPSHOT"
	}
}

func main() {
	var cfg config.Config
	var rootCfg *config.RootConfig
	var configOverrides []string
	var planURI, stateURI, macroURI, recordDir string
	var extraArgs []string
	var ciMode bool
	var jsonStdout bool

	session := &Session{cfg: cfg, rootCfg: rootCfg}

	rootCmd := &cobra.Command{
		Use:          "tfui",
		Short:        "Terminal UI for Terraform operations",
		Long:         "terraform-ui provides animated terminal feedback for terraform plan and apply operations.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg.Dir = resolveProjectDir(cfg.Dir)
			cfg.ApplyOverrides(configOverrides)

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
			logging.Init(recordDir != "", version, cfg.Dir, binary, recordDir)

			session.cfg = cfg
			session.rootCfg = rootCfg
			session.planURI = planURI
			session.stateURI = stateURI
			session.macroURI = macroURI
			session.recordDir = recordDir
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
	rootCmd.PersistentFlags().StringVar(&planURI, "plan", "", "Pre-seed plan data from file (./path, /path, file://) or - for stdin")
	rootCmd.PersistentFlags().StringVar(&stateURI, "state", "", "Pre-seed state data from file (./path, /path, file://) or - for stdin")
	rootCmd.PersistentFlags().StringVar(&macroURI, "macro", "", "Run a macro tape file (headless TUI recording)")
	rootCmd.PersistentFlags().StringVar(&recordDir, "record", "", "Record session frames and tape to directory")
	rootCmd.PersistentFlags().StringVar(&cfg.Chdir, "chdir", "", "Select member directory (validated against member blocks in project mode)")
	rootCmd.PersistentFlags().BoolVar(&ciMode, "ci", false, "Suppress TUI (CI-friendly output)")
	rootCmd.PersistentFlags().BoolVar(&jsonStdout, "json", false, "Output JSON (terraform-compatible)")

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Run terraform plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.ForPlugin("plan").
				WithArgs(args).
				Run()
		},
	}
	planCmd.Flags().StringSliceVar(&cfg.Targets, "target", nil, "Resource targets for plan")

	var scaffoldForce, scaffoldYes bool
	scaffoldCmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Generate tfui.hcl configuration",
		Long:  "Detect terraform project patterns and generate tfui.hcl in the working directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScaffold(cfg, scaffoldForce, scaffoldYes)
		},
	}
	scaffoldCmd.Flags().BoolVar(&scaffoldForce, "force", false, "Overwrite existing tfui.hcl")
	scaffoldCmd.Flags().BoolVar(&scaffoldYes, "yes", false, "Skip prompts, use detected defaults")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.ForPlugin("version").
				WithArgs(args).
				Run()
		},
	}

	var initUpgrade, initReconfigure bool
	var initBackendConfig []string

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Run terraform init",
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.ForPlugin("init").
				WithArgs(buildInitArgs(cmd)).
				Run()
		},
	}
	initCmd.Flags().BoolVar(&initUpgrade, "upgrade", false, "Upgrade modules and plugins")
	initCmd.Flags().BoolVar(&initReconfigure, "reconfigure", false, "Reconfigure backend")
	initCmd.Flags().Bool("backend", true, "Configure backend (--backend=false to skip)")
	initCmd.Flags().StringArrayVar(&initBackendConfig, "backend-config", nil, "Backend configuration values")

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Run terraform validate",
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.ForPlugin("validate").
				WithArgs(args).
				Run()
		},
	}

	outputCmd := &cobra.Command{
		Use:   "output",
		Short: "Show terraform outputs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.ForPlugin("output").
				WithArgs(args).
				Run()
		},
	}

	stateCmd := &cobra.Command{
		Use:   "state",
		Short: "Terraform state operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.ForPlugin("state").
				WithArgs(args).
				Run()
		},
	}

	rootCmd.AddCommand(planCmd, buildApplyCommand(session), buildTaintCommand(session), buildUntaintCommand(session), initCmd, validateCmd, outputCmd, stateCmd, scaffoldCmd, versionCmd)

	// Plugin CLI commands
	for _, cmd := range buildPluginCommands(&cfg) {
		rootCmd.AddCommand(cmd)
	}

	os.Args, extraArgs = splitPassthrough(os.Args)
	os.Args = normalizeArgs(os.Args)
	cfg.ExtraArgs = extraArgs

	if err := rootCmd.Execute(); err != nil {
		if runErr, ok := err.(*macro.RunError); ok {
			fmt.Fprintf(os.Stderr, "macro: %s\n", runErr.Error())
			os.Exit(runErr.Code)
		}
		os.Exit(1)
	}
}

func buildInitArgs(cmd *cobra.Command) []string {
	var args []string
	if cmd.Flags().Changed("upgrade") {
		args = append(args, "--upgrade")
	}
	if cmd.Flags().Changed("reconfigure") {
		args = append(args, "--reconfigure")
	}
	if cmd.Flags().Changed("backend") {
		val, _ := cmd.Flags().GetBool("backend")
		if val {
			args = append(args, "--backend=true")
		} else {
			args = append(args, "--backend=false")
		}
	}
	if cmd.Flags().Changed("backend-config") {
		vals, _ := cmd.Flags().GetStringArray("backend-config")
		for _, v := range vals {
			args = append(args, "--backend-config="+v)
		}
	}
	return args
}

func buildRegistry(svc sdk.Service, cfg config.Config) *plugin.Registry {
	registry := plugin.NewRegistry()
	registry.RegisterFactory("context", tfuicontext.New, plugin.PluginMeta{Keybinding: "C", MenuVisible: true})
	registry.RegisterFactory("chdir", tfuichdir.New, plugin.PluginMeta{Keybinding: "", MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("state", tfuistate.New, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("plan", tfuiplan.New, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("apply", tfuiapply.New, plugin.PluginMeta{Keybinding: "a", MenuVisible: false})
	registry.RegisterFactory("workspace", tfuiworkspace.New, plugin.PluginMeta{Keybinding: "w", MenuVisible: true, Nav: plugin.NavPush})
	registry.RegisterFactory("console", tfuiconsole.New, plugin.PluginMeta{Keybinding: "~", MenuVisible: true})
	registry.RegisterFactory("output", tfuioutput.New, plugin.PluginMeta{Keybinding: "o", MenuVisible: true})
	registry.RegisterFactory("validate", tfuivalidate.New, plugin.PluginMeta{Keybinding: "v", MenuVisible: true})
	registry.RegisterFactory("init", tfuiinit.New, plugin.PluginMeta{Keybinding: "i", MenuVisible: true})
	registry.RegisterFactory("risk", tfuirisk.New, plugin.PluginMeta{Keybinding: "R", MenuVisible: true})
	registry.RegisterFactory("phantom", tfuiphantom.New, plugin.PluginMeta{Keybinding: "P", MenuVisible: true})
	registry.RegisterFactory("blastradius", tfuiblastradius.New, plugin.PluginMeta{Keybinding: "B", MenuVisible: true})
	registry.RegisterFactory("taint", tfuitaint.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("untaint", tfuiuntaint.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("import", tfuiimport.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("forceunlock", tfuiforceunlock.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("version", tfuiversion.New, plugin.PluginMeta{MenuVisible: false, Nav: plugin.NavPush})

	registry.Build(svc, cfg.Plugins)

	var memberPaths []string
	if rootCfg, err := config.LoadRoot(cfg.Dir); err == nil && len(rootCfg.Members) > 0 {
		memberPaths = make([]string, len(rootCfg.Members))
		for i, m := range rootCfg.Members {
			memberPaths[i] = m.Path
		}
	}

	if ctxPlugin, ok := registry.ByID("context"); ok {
		if cp, ok := ctxPlugin.(*tfuicontext.Plugin); ok {
			cp.SetProjectDir(cfg.Dir)
			if len(memberPaths) > 0 {
				cp.SetMembers(memberPaths)
			}
		}
	}
	if chdirPlugin, ok := registry.ByID("chdir"); ok {
		if cp, ok := chdirPlugin.(*tfuichdir.Plugin); ok {
			if len(memberPaths) > 0 {
				cp.SetMembers(memberPaths)
			}
		}
	}
	if versionPlugin, ok := registry.ByID("version"); ok {
		_ = versionPlugin.Configure(map[string]interface{}{"tfui_version": version})
	}

	return registry
}

func seedCache(cache *terraform.ServiceCache, planURI, stateURI string) error {
	if planURI == "-" && stateURI == "-" {
		return fmt.Errorf("stdin (-) can only be used by one flag per invocation; use a file for the other")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	resolver := source.NewResolver(
		&source.LocalProvider{BaseDir: cwd},
		&source.StdinProvider{},
	)
	ctx := context.Background()

	if planURI != "" {
		if planURI == "-" {
			data, resolveErr := resolver.Resolve(ctx, planURI)
			if resolveErr != nil {
				return fmt.Errorf("loading plan: %w", resolveErr)
			}
			if err := cache.SeedPlan("", data); err != nil {
				return fmt.Errorf("parsing plan: %w", err)
			}
		} else {
			planFile, resolveErr := resolveToAbsPath(cwd, planURI)
			if resolveErr != nil {
				return fmt.Errorf("resolving plan path: %w", resolveErr)
			}
			if err := cache.SeedPlan(planFile, nil); err != nil {
				return fmt.Errorf("loading plan: %w", err)
			}
		}
	}

	if stateURI != "" {
		if stateURI == "-" {
			data, resolveErr := resolver.Resolve(ctx, stateURI)
			if resolveErr != nil {
				return fmt.Errorf("loading state: %w", resolveErr)
			}
			if err := cache.SeedState("", data); err != nil {
				return fmt.Errorf("parsing state: %w", err)
			}
		} else {
			stateFile, resolveErr := resolveToAbsPath(cwd, stateURI)
			if resolveErr != nil {
				return fmt.Errorf("resolving state path: %w", resolveErr)
			}
			if err := cache.SeedState(stateFile, nil); err != nil {
				return fmt.Errorf("loading state: %w", err)
			}
		}
	}

	return nil
}

func resolveToAbsPath(baseDir, uri string) (string, error) {
	if filepath.IsAbs(uri) {
		return uri, nil
	}
	clean := uri
	if len(clean) > 2 && clean[:2] == "./" {
		clean = clean[2:]
	}
	if len(clean) > 7 && clean[:7] == "file://" {
		clean = clean[7:]
		if filepath.IsAbs(clean) {
			return clean, nil
		}
	}
	return filepath.Abs(filepath.Join(baseDir, clean))
}

func hasTTY() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

func isStderrTTY() bool {
	return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
}

func effectiveWorkDir(cfg config.Config) string {
	if cfg.Chdir != "" {
		return filepath.Join(cfg.Dir, cfg.Chdir)
	}
	return cfg.WorkingDir()
}

func validateChdir(cfg config.Config) error {
	dir := filepath.Join(cfg.Dir, cfg.Chdir)

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("chdir %q not found (resolved to %s)", cfg.Chdir, dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("chdir %q is not a directory (resolved to %s)", cfg.Chdir, dir)
	}
	if !config.HasTerraformFiles(dir) {
		return fmt.Errorf("chdir %q has no .tf files (resolved to %s)", cfg.Chdir, dir)
	}
	return nil
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
