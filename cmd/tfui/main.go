package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/internal/macro"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/scaffold"
	"github.com/lmarqs/terraform-ui/internal/source"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiapply "github.com/lmarqs/terraform-ui/plugins/apply"
	tfuiblastradius "github.com/lmarqs/terraform-ui/plugins/blastradius"
	tfuichdir "github.com/lmarqs/terraform-ui/plugins/chdir"
	tfuicontext "github.com/lmarqs/terraform-ui/plugins/context"
	tfuioutput "github.com/lmarqs/terraform-ui/plugins/output"
	tfuiphantom "github.com/lmarqs/terraform-ui/plugins/phantom"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	tfuirepl "github.com/lmarqs/terraform-ui/plugins/repl"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
	tfuistate "github.com/lmarqs/terraform-ui/plugins/state"
	tfuivalidate "github.com/lmarqs/terraform-ui/plugins/validate"
	tfuiworkspaces "github.com/lmarqs/terraform-ui/plugins/workspaces"
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
	var debug bool
	var configOverrides []string
	var planURI, stateURI, macroURI string
	var extraArgs []string

	rootCmd := &cobra.Command{
		Use:          "tfui",
		Short:        "Terminal UI for Terraform operations",
		Long:         "terraform-ui provides animated terminal feedback for terraform plan and apply operations.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg.Dir = resolveProjectDir(cfg.Dir)
			cfg.ApplyOverrides(configOverrides)

			rootCfg, err := config.LoadRoot(cfg.Dir)
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
			logging.Init(debug, version, cfg.Dir, binary, cfg.LogDir())
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if macroURI != "" {
				return runMacro(cfg, macroURI, planURI, stateURI)
			}
			return runTUI(cfg, planURI, stateURI)
		},
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&cfg.Dir, "project", ".", "Project root directory (where tfui.hcl lives)")
	rootCmd.PersistentFlags().StringVar(&cfg.Terraform.Bin, "terraform-bin", "", "Path to terraform/tofu binary")
	rootCmd.PersistentFlags().StringArrayVar(&configOverrides, "config", nil, "Override config values (key=value, e.g. --config logger.dir=/tmp/logs --config terraform.bin=tofu)")
	rootCmd.Flags().StringVar(&planURI, "plan", "", "Load plan JSON from file (./path, /path, file://) or - for stdin")
	rootCmd.Flags().StringVar(&stateURI, "state", "", "Load state JSON from file (./path, /path, file://) or - for stdin")
	rootCmd.Flags().StringVar(&macroURI, "macro", "", "Run a macro tape file")
	rootCmd.PersistentFlags().StringVar(&cfg.ActiveScope, "chdir", "", "Select member directory (validated against member blocks in project mode)")

	var ciMode bool
	var jsonMode bool

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Run terraform plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlan(cfg, ciMode, jsonMode)
		},
	}
	planCmd.Flags().BoolVar(&ciMode, "ci", false, "Suppress spinner (CI-friendly)")
	planCmd.Flags().BoolVar(&jsonMode, "json", false, "Output JSON (terraform-compatible)")
	planCmd.Flags().StringSliceVar(&cfg.Targets, "target", nil, "Resource targets for plan")

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Run terraform apply",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cfg, ciMode, jsonMode)
		},
	}
	applyCmd.Flags().BoolVar(&ciMode, "ci", false, "Suppress spinner (CI-friendly)")
	applyCmd.Flags().BoolVar(&jsonMode, "json", false, "Output JSON (terraform-compatible)")
	applyCmd.Flags().StringSliceVar(&cfg.Targets, "target", nil, "Resource targets for apply")

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
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("tfui %s\n", version)
		},
	}

	rootCmd.AddCommand(planCmd, applyCmd, scaffoldCmd, versionCmd)

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

func runTUI(cfg config.Config, planURI, stateURI string) error {
	if !hasTTY() {
		if planURI != "" || stateURI != "" {
			return runStaticNonInteractive(cfg, planURI, stateURI)
		}
		return fmt.Errorf("no TTY detected (terminal required for interactive mode)\n\nFor non-interactive use:\n  tfui plan -json           (JSON output)\n  tfui plan --ci            (tree output, no spinner)\n  tfui --plan ./file.json   (auto-renders without TTY)")
	}

	if cfg.ActiveScope != "" {
		if err := validateScope(cfg); err != nil {
			return err
		}
	}

	cache := terraform.NewServiceCache()
	if planURI != "" || stateURI != "" {
		cfg.PreloadedData = true
		if err := seedCache(cache, planURI, stateURI); err != nil {
			return err
		}
	}
	svc := terraform.NewExecService(effectiveWorkDir(cfg), cfg.TerraformBinary(), cache)

	registry := buildRegistry(svc, cfg)

	app := ui.NewApp(cfg, svc, registry)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func buildRegistry(svc sdk.Service, cfg config.Config) *plugin.Registry {
	registry := plugin.NewRegistry()
	registry.RegisterFactory("context", tfuicontext.New, plugin.PluginMeta{Keybinding: "C", MenuVisible: true})
	registry.RegisterFactory("chdir", tfuichdir.New, plugin.PluginMeta{Keybinding: "", MenuVisible: false, Nav: plugin.NavPush})
	registry.RegisterFactory("state", tfuistate.New, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("plan", tfuiplan.New, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("apply", tfuiapply.New, plugin.PluginMeta{Keybinding: "a", MenuVisible: true})
	registry.RegisterFactory("workspaces", tfuiworkspaces.New, plugin.PluginMeta{Keybinding: "w", MenuVisible: true, Nav: plugin.NavPush})
	registry.RegisterFactory("repl", tfuirepl.New, plugin.PluginMeta{Keybinding: "t", MenuVisible: true})
	registry.RegisterFactory("output", tfuioutput.New, plugin.PluginMeta{Keybinding: "o", MenuVisible: true})
	registry.RegisterFactory("validate", tfuivalidate.New, plugin.PluginMeta{Keybinding: "v", MenuVisible: true})
	registry.RegisterFactory("risk", tfuirisk.New, plugin.PluginMeta{Keybinding: "R", MenuVisible: true})
	registry.RegisterFactory("phantom", tfuiphantom.New, plugin.PluginMeta{Keybinding: "P", MenuVisible: true})
	registry.RegisterFactory("blastradius", tfuiblastradius.New, plugin.PluginMeta{Keybinding: "B", MenuVisible: true})

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
			cp.SetConfig(cfg)
			if len(memberPaths) > 0 {
				cp.SetMembers(memberPaths, cfg.Dir)
			}
		}
	}
	if chdirPlugin, ok := registry.ByID("chdir"); ok {
		if cp, ok := chdirPlugin.(*tfuichdir.Plugin); ok {
			if len(memberPaths) > 0 {
				cp.SetMembers(memberPaths, cfg.Dir)
			}
		}
	}

	return registry
}

func runMacro(cfg config.Config, macroURI, planURI, stateURI string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	resolver := source.NewResolver(
		&source.LocalProvider{BaseDir: cwd},
		&source.StdinProvider{},
	)
	ctx := context.Background()

	tapeData, err := resolver.Resolve(ctx, macroURI)
	if err != nil {
		return fmt.Errorf("loading macro tape: %w", err)
	}

	commands, err := macro.ParseTape(tapeData)
	if err != nil {
		return &macro.RunError{Code: macro.ExitSyntaxError, Message: err.Error()}
	}

	if len(commands) == 0 {
		return nil
	}

	cache := terraform.NewServiceCache()
	if planURI != "" || stateURI != "" {
		cfg.PreloadedData = true
		if err := seedCache(cache, planURI, stateURI); err != nil {
			return err
		}
	}

	svc := terraform.NewMacroService(cfg.TerraformBinary(), cache)
	registry := buildRegistry(svc, cfg)
	app := ui.NewApp(cfg, svc, registry)

	driver := macro.NewDriver(app, 80, 24)
	runner := macro.NewRunner(driver)

	if err := runner.Execute(commands); err != nil {
		return err
	}

	for _, cmd := range svc.Commands() {
		fmt.Println(cmd.String())
	}
	return nil
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

func runStaticNonInteractive(cfg config.Config, planURI, stateURI string) error {
	cache := terraform.NewServiceCache()
	if err := seedCache(cache, planURI, stateURI); err != nil {
		return err
	}

	ctx := context.Background()

	if planURI != "" {
		plan, ok := cache.GetPlan()
		if !ok {
			svc := terraform.NewExecService(effectiveWorkDir(cfg), cfg.TerraformBinary(), cache)
			var err error
			plan, err = svc.Plan(ctx, sdk.PlanOptions{})
			if err != nil {
				return err
			}
		}
		printTreeView(plan)
	}

	if stateURI != "" {
		resources, ok := cache.GetResources()
		if !ok {
			svc := terraform.NewExecService(effectiveWorkDir(cfg), cfg.TerraformBinary(), cache)
			var err error
			resources, err = svc.StateList(ctx)
			if err != nil {
				return err
			}
		}
		if planURI != "" {
			fmt.Println()
		}
		fmt.Printf("State: %d resources\n", len(resources))
		for _, r := range resources {
			fmt.Printf("  %s\n", r.Address)
		}
	}

	return nil
}

func hasTTY() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

func isStderrTTY() bool {
	return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
}

func effectiveWorkDir(cfg config.Config) string {
	if cfg.ActiveScope != "" {
		return filepath.Join(cfg.Dir, cfg.ActiveScope)
	}
	return cfg.WorkingDir()
}

func validateScope(cfg config.Config) error {
	scopeDir := filepath.Join(cfg.Dir, cfg.ActiveScope)

	info, err := os.Stat(scopeDir)
	if err != nil {
		return fmt.Errorf("scope %q not found (resolved to %s)", cfg.ActiveScope, scopeDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("scope %q is not a directory (resolved to %s)", cfg.ActiveScope, scopeDir)
	}
	if !config.HasTerraformFiles(scopeDir) {
		return fmt.Errorf("scope %q has no .tf files (resolved to %s)", cfg.ActiveScope, scopeDir)
	}
	return nil
}

// spinnerFrames are the braille spinner characters.
var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// spinner manages an animated spinner on stderr.
type spinner struct {
	mu          sync.Mutex
	stop        chan struct{}
	done        chan struct{}
	message     string
	start       time.Time
	showElapsed bool
}

func newSpinner(message string, showElapsed bool) *spinner {
	return &spinner{
		message:     message,
		showElapsed: showElapsed,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
		start:       time.Now(),
	}
}

func (s *spinner) run() {
	go func() {
		defer close(s.done)
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			case <-ticker.C:
				s.mu.Lock()
				frame := spinnerFrames[i%len(spinnerFrames)]
				if s.showElapsed {
					elapsed := time.Since(s.start).Truncate(time.Second)
					fmt.Fprintf(os.Stderr, "\r\033[K%c %s (%s)", frame, s.message, elapsed)
				} else {
					fmt.Fprintf(os.Stderr, "\r\033[K%c %s", frame, s.message)
				}
				s.mu.Unlock()
				i++
			}
		}
	}()
}

func (s *spinner) halt() {
	close(s.stop)
	<-s.done
}

// actionSymbol returns the tree-view prefix for a given action.
func actionSymbol(action terraform.Action) string {
	switch action {
	case terraform.ActionCreate:
		return "+"
	case terraform.ActionUpdate:
		return "~"
	case terraform.ActionDelete:
		return "-"
	case terraform.ActionDeleteThenCreate, terraform.ActionCreateThenDelete:
		return "-/+"
	case terraform.ActionRead:
		return "<="
	default:
		return " "
	}
}

// printTreeView prints the plan tree view to stdout.
func printTreeView(summary *terraform.PlanSummary) {
	for _, change := range summary.Changes {
		sym := actionSymbol(change.Action)
		fmt.Printf("%s %s\n", sym, change.Resource.Address)
	}
	fmt.Println()
	fmt.Printf("Plan: %d to add, %d to change, %d to destroy.\n",
		summary.ToCreate, summary.ToUpdate+summary.ToReplace, summary.ToDelete)

	risk := terraform.OverallRisk(summary.Changes)
	if risk > terraform.RiskNone {
		fmt.Printf("Risk: %s\n", risk)
	}
}

// agentOutput is the JSON structure for agent mode.
type agentOutput struct {
	Changes          []agentChange `json:"changes"`
	Summary          agentSummary  `json:"summary"`
	Risk             string        `json:"risk"`
	PhantomChanges   int           `json:"phantom_changes"`
	PhantomResources []string      `json:"phantom_resources"`
}

type agentChange struct {
	Address string `json:"address"`
	Action  string `json:"action"`
	Risk    string `json:"risk"`
	Phantom bool   `json:"phantom,omitempty"`
}

type agentSummary struct {
	Add     int `json:"add"`
	Change  int `json:"change"`
	Destroy int `json:"destroy"`
}

// printAgentJSON outputs structured JSON for agent mode.
func printAgentJSON(summary *terraform.PlanSummary) error {
	phantomResult := terraform.DetectPhantomChanges(summary.Changes)

	changes := make([]agentChange, 0, len(summary.Changes))
	for _, c := range summary.Changes {
		changes = append(changes, agentChange{
			Address: c.Resource.Address,
			Action:  string(c.Action),
			Risk:    c.Risk.String(),
			Phantom: c.IsPhantom,
		})
	}

	output := agentOutput{
		Changes: changes,
		Summary: agentSummary{
			Add:     summary.ToCreate,
			Change:  summary.ToUpdate + summary.ToReplace,
			Destroy: summary.ToDelete,
		},
		Risk:             terraform.OverallRisk(summary.Changes).String(),
		PhantomChanges:   phantomResult.PhantomCount,
		PhantomResources: phantomResult.PhantomAddresses,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
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

func runPlan(cfg config.Config, ci bool, jsonOutput bool) error {
	if cfg.ActiveScope != "" {
		if err := validateScope(cfg); err != nil {
			return err
		}
	}

	binary := cfg.TerraformBinary()
	svc := terraform.NewExecService(effectiveWorkDir(cfg), binary, nil)
	ctx := context.Background()

	showSpinner := !ci && !jsonOutput && isStderrTTY()
	var s *spinner
	if showSpinner {
		s = newSpinner("Running terraform plan...", true)
		s.run()
	}

	summary, err := svc.Plan(ctx, sdk.PlanOptions{Targets: cfg.Targets, ExtraArgs: cfg.ExtraArgs})

	if showSpinner {
		s.halt()
	}
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
	}

	if jsonOutput {
		return printAgentJSON(summary)
	}
	printTreeView(summary)
	return nil
}

func runApply(cfg config.Config, ci bool, jsonOutput bool) error {
	if cfg.ActiveScope != "" {
		if err := validateScope(cfg); err != nil {
			return err
		}
	}

	binary := cfg.TerraformBinary()
	svc := terraform.NewExecService(effectiveWorkDir(cfg), binary, nil)
	ctx := context.Background()

	showSpinner := !ci && !jsonOutput && isStderrTTY()
	var s *spinner
	if showSpinner {
		s = newSpinner("Running terraform apply...", true)
		s.run()
	}

	err := svc.Apply(ctx, sdk.ApplyOptions{Targets: cfg.Targets, ExtraArgs: cfg.ExtraArgs})

	if showSpinner {
		s.halt()
	}
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	if jsonOutput {
		result := map[string]interface{}{"status": "complete"}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	fmt.Println("Apply complete.")
	return nil
}
