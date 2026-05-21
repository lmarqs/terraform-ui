package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/internal/macro"
	"github.com/lmarqs/terraform-ui/internal/plugin"
	"github.com/lmarqs/terraform-ui/internal/source"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	tfexec "github.com/lmarqs/terraform-ui/internal/terraform/exec"
	"github.com/lmarqs/terraform-ui/internal/ui"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type Presentation int

const (
	Interactive Presentation = iota
	Headless
)

type Backend int

const (
	Exec Backend = iota
	Recording
)

type axes struct {
	presentation Presentation
	backend      Backend
}

type Session struct {
	cfg               config.Config
	rootCfg           *config.RootConfig
	pluginID          string
	args              []string
	jsonMode          bool
	planURI           string
	stateURI          string
	outputsURI        string
	validateResultURI string
	workspacesURI     string
	macroURI          string
	recordDir         string
	ciMode            bool
}

func NewSession(cfg config.Config, rootCfg *config.RootConfig) *Session {
	return &Session{cfg: cfg, rootCfg: rootCfg}
}

func (s *Session) ForPlugin(id string) *Session {
	s.pluginID = id
	return s
}

func (s *Session) WithArgs(args []string) *Session {
	s.args = args
	return s
}

func (s *Session) WithJSON(on bool) *Session {
	s.jsonMode = on
	return s
}

func (s *Session) WithSeeds(plan, state string) *Session {
	s.planURI = plan
	s.stateURI = state
	return s
}

func (s *Session) WithExtraSeeds(outputs, validateResult, workspaces string) *Session {
	s.outputsURI = outputs
	s.validateResultURI = validateResult
	s.workspacesURI = workspaces
	return s
}

func (s *Session) WithMacro(uri string) *Session {
	s.macroURI = uri
	return s
}

func (s *Session) WithRecord(dir string) *Session {
	s.recordDir = dir
	return s
}

func (s *Session) WithCI(flag bool) *Session {
	s.ciMode = flag
	return s
}

func (s *Session) Run() error {
	if err := s.validate(); err != nil {
		return err
	}
	ax := s.resolveAxes()
	tape, err := s.loadTape()
	if err != nil {
		return err
	}
	svc, recorder, err := s.buildService(ax)
	if err != nil {
		return err
	}
	registry := buildRegistry(svc, s.cfg)
	app := s.buildApp(svc, registry)
	result, err := s.present(app, registry, ax, tape)
	if err != nil {
		return err
	}
	return s.emit(result, registry, recorder)
}

func (s *Session) validate() error {
	if s.cfg.Chdir != "" {
		if err := validateChdir(s.cfg); err != nil {
			return err
		}
	}
	if s.pluginID == "" && s.macroURI == "" && !hasTTY() {
		return fmt.Errorf("no TTY detected (terminal required for interactive mode)\n\nFor non-interactive use:\n  tfui plan --ci            (CI mode, no TUI)\n  CI=1 tfui plan            (same via env var)")
	}
	return nil
}

func (s *Session) resolveAxes() axes {
	if s.macroURI != "" {
		logging.Logger().Debug("session.axes", "presentation", "headless", "backend", "recording", "reason", "macro")
		return axes{Headless, Recording}
	}
	if s.pluginID == "" {
		logging.Logger().Debug("session.axes", "presentation", "interactive", "backend", "exec", "reason", "root")
		return axes{Interactive, Exec}
	}
	if s.ciMode {
		logging.Logger().Debug("session.axes", "presentation", "headless", "backend", "exec", "reason", "ci-flag")
		return axes{Headless, Exec}
	}
	if os.Getenv("CI") == "1" {
		logging.Logger().Debug("session.axes", "presentation", "headless", "backend", "exec", "reason", "ci-env")
		return axes{Headless, Exec}
	}
	if !isStderrTTY() {
		logging.Logger().Debug("session.axes", "presentation", "headless", "backend", "exec", "reason", "no-tty")
		return axes{Headless, Exec}
	}
	logging.Logger().Debug("session.axes", "presentation", "interactive", "backend", "exec", "reason", "tty")
	return axes{Interactive, Exec}
}

func (s *Session) loadTape() ([]macro.Command, error) {
	if s.macroURI == "" {
		return nil, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	resolver := source.NewResolver(
		&source.LocalProvider{BaseDir: cwd},
		&source.StdinProvider{},
	)
	tapeData, err := resolver.Resolve(context.Background(), s.macroURI)
	if err != nil {
		return nil, fmt.Errorf("loading macro tape: %w", err)
	}
	commands, err := macro.ParseTape(tapeData)
	if err != nil {
		return nil, &macro.RunError{Code: macro.ExitSyntaxError, Message: err.Error()}
	}
	if len(commands) == 0 {
		return nil, nil
	}
	return commands, nil
}

func (s *Session) buildService(ax axes) (sdk.Service, *terraform.MacroService, error) {
	cache := terraform.NewServiceCache()
	if s.planURI != "" || s.stateURI != "" || s.outputsURI != "" || s.validateResultURI != "" || s.workspacesURI != "" {
		s.cfg.PreloadedData = true
		if err := seedCache(cache, s.planURI, s.stateURI, s.outputsURI, s.validateResultURI, s.workspacesURI); err != nil {
			return nil, nil, err
		}
	}
	switch ax.backend {
	case Recording:
		svc := terraform.NewMacroService(s.cfg.TerraformBinary(), cache)
		return svc, svc, nil
	default:
		svc := tfexec.NewExecService(effectiveWorkDir(s.cfg), s.cfg.TerraformBinary(), cache)
		return svc, nil, nil
	}
}

func (s *Session) buildApp(svc sdk.Service, registry *plugin.Registry) ui.App {
	if s.pluginID == "" {
		return ui.NewApp(s.cfg, svc, registry, s.rootCfg)
	}
	standalone := &ui.StandaloneConfig{
		PluginID: s.pluginID,
		Args:     s.args,
		JSONMode: s.jsonMode,
	}
	return ui.NewApp(s.cfg, svc, registry, s.rootCfg, standalone)
}

func (s *Session) present(app ui.App, registry *plugin.Registry, ax axes, tape []macro.Command) (ui.App, error) {
	switch ax.presentation {
	case Interactive:
		opts := []tea.ProgramOption{tea.WithAltScreen()}
		if s.pluginID != "" {
			opts = append(opts, tea.WithOutput(os.Stderr))
		}

		if s.recordDir != "" {
			rec := macro.NewRecorder(app, s.recordDir, 80, 24)
			_, err := tea.NewProgram(rec, opts...).Run()
			_ = rec.Finalize()
			if err != nil {
				return app, err
			}
			if inner, ok := rec.Inner().(ui.App); ok {
				return inner, nil
			}
			return app, nil
		}

		model, err := tea.NewProgram(app, opts...).Run()
		if err != nil {
			return app, err
		}
		if a, ok := model.(ui.App); ok {
			return a, nil
		}
		return app, nil

	case Headless:
		driver := macro.NewDriver(app, 80, 24)
		if tape != nil {
			runner := macro.NewRunner(driver)
			if s.recordDir != "" {
				rec := macro.NewRecorder(nil, s.recordDir, 80, 24)
				runner.WithRecorder(rec)
			}
			return app, runner.Execute(tape)
		}
		driver.Init()
		pluginID := s.pluginID
		return app, driver.WaitUntil(func(view string) bool {
			if p, ok := registry.ByID(pluginID); ok {
				return p.Ready()
			}
			return false
		}, 10*time.Minute)
	}
	return app, nil
}

func (s *Session) emit(app ui.App, registry *plugin.Registry, recorder *terraform.MacroService) error {
	if recorder != nil {
		for _, cmd := range recorder.Commands() {
			fmt.Println(cmd.String())
		}
		return nil
	}
	if s.pluginID == "" {
		return nil
	}

	var p sdk.Plugin
	if active := app.ActivePlugin(); active != nil {
		p = active
	} else if found, ok := registry.ByID(s.pluginID); ok {
		p = found
	}
	if p == nil {
		return nil
	}

	if outputter, ok := p.(sdk.Outputter); ok {
		data, err := outputter.Output(s.jsonMode)
		if err != nil {
			return err
		}
		_, _ = os.Stdout.Write(data)
	}
	if coder, ok := p.(sdk.ExitCoder); ok {
		if code := coder.ExitCode(); code != 0 {
			os.Exit(code)
		}
	}
	return nil
}
