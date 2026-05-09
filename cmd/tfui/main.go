package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/internal/ui"
	"github.com/lmarqs/terraform-ui/plugins/apply"
	"github.com/lmarqs/terraform-ui/plugins/blastradius"
	tfuiinit "github.com/lmarqs/terraform-ui/plugins/init"
	"github.com/lmarqs/terraform-ui/plugins/phantom"
	"github.com/lmarqs/terraform-ui/plugins/plan"
	"github.com/lmarqs/terraform-ui/plugins/projects"
	"github.com/lmarqs/terraform-ui/plugins/risk"
	"github.com/lmarqs/terraform-ui/plugins/state"
	"github.com/lmarqs/terraform-ui/plugins/workspaces"
	"github.com/spf13/cobra"
)

var version = "1.0.0-dev"

func main() {
	var cfg config.Config
	var debug bool

	rootCmd := &cobra.Command{
		Use:   "tfui",
		Short: "Terminal UI for Terraform operations",
		Long:  "terraform-ui provides animated terminal feedback for terraform plan and apply operations.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			binary := config.DetectBinary(cfg.TerraformBinary)
			logging.Init(debug, version, cfg.Dir, binary)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(cfg)
		},
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging to ~/.tfui/debug.log")
	rootCmd.PersistentFlags().StringVar(&cfg.Dir, "dir", ".", "Working directory for terraform operations")
	rootCmd.PersistentFlags().StringVar(&cfg.TerraformBinary, "terraform-bin", "", "Path to terraform/tofu binary (auto-detects if empty)")

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

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Generate tfui.yaml configuration",
		Long:  "Detect terraform project patterns and generate a tfui.yaml config file in the working directory.",
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

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI(cfg config.Config) error {
	cfg.TerraformBinary = config.DetectBinary(cfg.TerraformBinary)
	svc := terraform.NewService(cfg.Dir, cfg.TerraformBinary)

	// Create and populate the plugin registry
	registry := plugin.NewRegistry()
	registry.RegisterFactory("plan", plan.New)
	registry.RegisterFactory("risk", risk.New)
	registry.RegisterFactory("phantom", phantom.New)
	registry.RegisterFactory("state", state.New)
	registry.RegisterFactory("apply", apply.New)
	registry.RegisterFactory("workspaces", workspaces.New)
	registry.RegisterFactory("projects", projects.New)
	registry.RegisterFactory("blastradius", blastradius.New)
	registry.RegisterFactory("init", tfuiinit.New)

	// Build plugins from config
	registry.Build(svc, cfg.Plugins)

	app := ui.NewApp(cfg, svc, registry)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
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

	outPath := filepath.Join(cfg.Dir, "tfui.yaml")
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "Wrote %s\n", outPath)
	fmt.Print(content)
	return nil
}

func runPlan(cfg config.Config) error {
	cfg.TerraformBinary = config.DetectBinary(cfg.TerraformBinary)
	svc := terraform.NewService(cfg.Dir, cfg.TerraformBinary)
	ctx := context.Background()

	switch cfg.Mode {
	case "silent":
		summary, err := svc.Plan(ctx, cfg.Targets)
		if err != nil {
			return fmt.Errorf("plan failed: %w", err)
		}
		printTreeView(summary)

	case "spinner":
		s := newSpinner("Running terraform plan...", false)
		s.run()
		summary, err := svc.Plan(ctx, cfg.Targets)
		s.halt()
		if err != nil {
			return fmt.Errorf("plan failed: %w", err)
		}
		printTreeView(summary)

	case "progress":
		s := newSpinner("Running terraform plan...", true)
		s.run()
		summary, err := svc.Plan(ctx, cfg.Targets)
		s.halt()
		if err != nil {
			return fmt.Errorf("plan failed: %w", err)
		}
		printTreeView(summary)

	case "agent":
		summary, err := svc.Plan(ctx, cfg.Targets)
		if err != nil {
			return fmt.Errorf("plan failed: %w", err)
		}
		return printAgentJSON(summary)

	default:
		return fmt.Errorf("unknown mode: %s", cfg.Mode)
	}

	return nil
}

func runApply(cfg config.Config) error {
	cfg.TerraformBinary = config.DetectBinary(cfg.TerraformBinary)
	svc := terraform.NewService(cfg.Dir, cfg.TerraformBinary)
	ctx := context.Background()

	switch cfg.Mode {
	case "silent":
		err := svc.Apply(ctx, cfg.Targets)
		if err != nil {
			return fmt.Errorf("apply failed: %w", err)
		}
		fmt.Println("Apply complete.")

	case "spinner":
		s := newSpinner("Running terraform apply...", false)
		s.run()
		err := svc.Apply(ctx, cfg.Targets)
		s.halt()
		if err != nil {
			return fmt.Errorf("apply failed: %w", err)
		}
		fmt.Println("Apply complete.")

	case "progress":
		s := newSpinner("Running terraform apply...", true)
		s.run()
		err := svc.Apply(ctx, cfg.Targets)
		s.halt()
		if err != nil {
			return fmt.Errorf("apply failed: %w", err)
		}
		fmt.Println("Apply complete.")

	case "agent":
		err := svc.Apply(ctx, cfg.Targets)
		if err != nil {
			return fmt.Errorf("apply failed: %w", err)
		}
		output := map[string]interface{}{
			"status": "complete",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)

	default:
		return fmt.Errorf("unknown mode: %s", cfg.Mode)
	}

	return nil
}
