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
	"github.com/lmarqs/terraform-ui/internal/source"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	tfuiapply "github.com/lmarqs/terraform-ui/plugins/apply"
	tfuiblast "github.com/lmarqs/terraform-ui/plugins/blastradius"
	tfuichdir "github.com/lmarqs/terraform-ui/plugins/chdir"
	tfuicontext "github.com/lmarqs/terraform-ui/plugins/context"
	tfuiinit "github.com/lmarqs/terraform-ui/plugins/init"
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
	rootCmd.Flags().StringVar(&macroURI, "macro", "", "Run a macro tape file (requires --plan or --state)")
	rootCmd.PersistentFlags().StringVar(&cfg.ActiveScope, "chdir", "", "Select chdir member (validated against chdir.members in project mode)")

	var ciMode bool
	var outputFormat string

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Run terraform plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlan(cfg, ciMode, outputFormat)
		},
	}
	planCmd.Flags().BoolVar(&ciMode, "ci", false, "Suppress spinner (CI-friendly)")
	planCmd.Flags().StringVar(&outputFormat, "output", "text", "Output format: text, json")
	planCmd.Flags().StringSliceVar(&cfg.Targets, "target", nil, "Resource targets for plan")

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Run terraform apply",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cfg, ciMode, outputFormat)
		},
	}
	applyCmd.Flags().BoolVar(&ciMode, "ci", false, "Suppress spinner (CI-friendly)")
	applyCmd.Flags().StringVar(&outputFormat, "output", "text", "Output format: text, json")
	applyCmd.Flags().StringSliceVar(&cfg.Targets, "target", nil, "Resource targets for apply")

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Generate tfui.hcl configuration",
		Long:  "Detect terraform project patterns and generate tfui.hcl in the working directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cfg)
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("tfui %s\n", version)
		},
	}

	rootCmd.AddCommand(planCmd, applyCmd, initCmd, versionCmd)

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
		return fmt.Errorf("no TTY detected (terminal required for interactive mode)\n\nFor non-interactive use:\n  tfui plan --output json   (JSON output)\n  tfui plan --ci            (tree output, no spinner)\n  tfui --plan ./file.json   (auto-renders without TTY)")
	}

	if cfg.ActiveScope != "" {
		if err := validateScope(cfg); err != nil {
			return err
		}
	}

	var svc sdk.Service

	if planURI != "" || stateURI != "" {
		compositeSvc, err := buildCompositeService(cfg, planURI, stateURI)
		if err != nil {
			return err
		}
		svc = compositeSvc
	} else {
		binary := cfg.TerraformBinary()
		svc = terraform.NewService(effectiveWorkDir(cfg), binary)
	}

	registry := buildRegistry(svc, cfg)

	app := ui.NewApp(cfg, svc, registry)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func buildRegistry(svc sdk.Service, cfg config.Config) *plugin.Registry {
	registry := plugin.NewRegistry()
	registry.RegisterFactory("context", tfuicontext.New, plugin.PluginMeta{Keybinding: "C", MenuVisible: true})
	registry.RegisterFactory("chdir", tfuichdir.New, plugin.PluginMeta{Keybinding: "", MenuVisible: false})
	registry.RegisterFactory("state", tfuistate.New, plugin.PluginMeta{Keybinding: "s", MenuVisible: true})
	registry.RegisterFactory("plan", tfuiplan.New, plugin.PluginMeta{Keybinding: "p", MenuVisible: true})
	registry.RegisterFactory("apply", tfuiapply.New, plugin.PluginMeta{Keybinding: "a", MenuVisible: true})
	registry.RegisterFactory("workspaces", tfuiworkspaces.New, plugin.PluginMeta{Keybinding: "w", MenuVisible: true})
	registry.RegisterFactory("repl", tfuirepl.New, plugin.PluginMeta{Keybinding: "t", MenuVisible: true})
	registry.RegisterFactory("output", tfuioutput.New, plugin.PluginMeta{Keybinding: "o", MenuVisible: true})
	registry.RegisterFactory("validate", tfuivalidate.New, plugin.PluginMeta{Keybinding: "v", MenuVisible: true})
	registry.RegisterFactory("risk", tfuirisk.New, plugin.PluginMeta{Keybinding: "R", MenuVisible: true})
	registry.RegisterFactory("phantom", tfuiphantom.New, plugin.PluginMeta{Keybinding: "P", MenuVisible: true})
	registry.RegisterFactory("blastradius", tfuiblast.New, plugin.PluginMeta{Keybinding: "B", MenuVisible: true})
	registry.RegisterFactory("init", tfuiinit.New, plugin.PluginMeta{Keybinding: "i", MenuVisible: true})

	registry.Build(svc, cfg.Plugins)

	if ctxPlugin, ok := registry.ByID("context"); ok {
		if cp, ok := ctxPlugin.(*tfuicontext.Plugin); ok {
			cp.SetConfig(cfg)
		}
	}
	if chdirPlugin, ok := registry.ByID("chdir"); ok {
		if cp, ok := chdirPlugin.(*tfuichdir.Plugin); ok {
			if rootCfg, err := config.LoadRoot(cfg.Dir); err == nil && len(rootCfg.Chdir.Members) > 0 {
				cp.SetMembers(rootCfg.Chdir.Members, cfg.Dir)
			}
		}
	}

	return registry
}

func runMacro(cfg config.Config, macroURI, planURI, stateURI string) error {
	if planURI == "" && stateURI == "" {
		return fmt.Errorf("--macro requires --plan or --state (read-only data source)")
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

	compositeSvc, err := buildCompositeService(cfg, planURI, stateURI)
	if err != nil {
		return err
	}

	recorder := terraform.NewRecordingService(compositeSvc, cfg.TerraformBinary())
	registry := buildRegistry(recorder, cfg)
	app := ui.NewApp(cfg, recorder, registry)

	driver := macro.NewDriver(app, 80, 24)
	runner := macro.NewRunner(driver)

	if err := runner.Execute(commands); err != nil {
		return err
	}

	for _, cmd := range recorder.Commands() {
		fmt.Println(cmd.String())
	}
	return nil
}

func buildCompositeService(cfg config.Config, planURI, stateURI string) (*terraform.CompositeService, error) {
	if planURI == "-" && stateURI == "-" {
		return nil, fmt.Errorf("stdin (-) can only be used by one flag per invocation; use a file for the other")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	resolver := source.NewResolver(
		&source.LocalProvider{BaseDir: cwd},
		&source.StdinProvider{},
	)
	ctx := context.Background()

	var planFile string
	var stdinPlan []byte
	if planURI != "" {
		if planURI == "-" {
			data, resolveErr := resolver.Resolve(ctx, planURI)
			if resolveErr != nil {
				return nil, fmt.Errorf("loading plan: %w", resolveErr)
			}
			stdinPlan = data
		} else {
			planFile, err = resolveToAbsPath(cwd, planURI)
			if err != nil {
				return nil, fmt.Errorf("resolving plan path: %w", err)
			}
		}
	}

	var stateFile string
	var stdinState []byte
	if stateURI != "" {
		if stateURI == "-" {
			data, resolveErr := resolver.Resolve(ctx, stateURI)
			if resolveErr != nil {
				return nil, fmt.Errorf("loading state: %w", resolveErr)
			}
			stdinState = data
		} else {
			stateFile, err = resolveToAbsPath(cwd, stateURI)
			if err != nil {
				return nil, fmt.Errorf("resolving state path: %w", err)
			}
		}
	}

	binary := cfg.TerraformBinary()
	var live sdk.Service
	if stateFile != "" {
		live = terraform.NewServiceWithState(effectiveWorkDir(cfg), binary, stateFile)
	} else {
		live = terraform.NewService(effectiveWorkDir(cfg), binary)
	}

	return terraform.NewCompositeService(live, planFile, stateFile, stdinPlan, stdinState), nil
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
	compositeSvc, err := buildCompositeService(cfg, planURI, stateURI)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if planURI != "" {
		summary, err := compositeSvc.Plan(ctx, sdk.PlanOptions{})
		if err != nil {
			return err
		}
		printTreeView(summary)
	}

	if stateURI != "" {
		resources, err := compositeSvc.StateList(ctx)
		if err != nil {
			return err
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

func runInit(cfg config.Config) error {
	content, err := tfuiinit.GenerateConfig(cfg.Dir)
	if err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	outPath := filepath.Join(cfg.Dir, config.HCLConfigFileName)
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "Wrote %s\n", outPath)
	fmt.Print(content)
	return nil
}

func runPlan(cfg config.Config, ci bool, output string) error {
	if output != "text" && output != "json" {
		return fmt.Errorf("unknown output format: %s (valid: text, json)", output)
	}

	if cfg.ActiveScope != "" {
		if err := validateScope(cfg); err != nil {
			return err
		}
	}

	binary := cfg.TerraformBinary()
	svc := terraform.NewService(effectiveWorkDir(cfg), binary)
	ctx := context.Background()

	showSpinner := !ci && isStderrTTY()
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

	switch output {
	case "json":
		return printAgentJSON(summary)
	default:
		printTreeView(summary)
		return nil
	}
}

func runApply(cfg config.Config, ci bool, output string) error {
	if output != "text" && output != "json" {
		return fmt.Errorf("unknown output format: %s (valid: text, json)", output)
	}

	if cfg.ActiveScope != "" {
		if err := validateScope(cfg); err != nil {
			return err
		}
	}

	binary := cfg.TerraformBinary()
	svc := terraform.NewService(effectiveWorkDir(cfg), binary)
	ctx := context.Background()

	showSpinner := !ci && isStderrTTY()
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

	switch output {
	case "json":
		result := map[string]interface{}{"status": "complete"}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	default:
		fmt.Println("Apply complete.")
		return nil
	}
}
