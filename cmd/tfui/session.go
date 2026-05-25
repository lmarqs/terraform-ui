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
	"github.com/lmarqs/terraform-ui/internal/terraform"
	tfexec "github.com/lmarqs/terraform-ui/internal/terraform/exec"
	"github.com/lmarqs/terraform-ui/internal/ui"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Session is the cmd-side coordinator. It holds parsed persistent flags and
// exposes RunPlugin (per-plugin execution) and Run (app-mode execution).
type Session struct {
	cfg          config.Config
	rootCfg      *config.RootConfig
	sources      PreseedSources
	macro        MacroSpec
	effects      Effects
	jsonStdout   bool
	ciMode       bool
	silentStderr bool // resolved at PersistentPreRunE: --ci || CI=1 || !isStderrTTY
}

func (s *Session) Run() error {
	if err := s.validate(); err != nil {
		return err
	}
	tape, err := s.macro.LoadTape()
	if err != nil {
		return err
	}
	svc, recorder, err := s.buildService()
	if err != nil {
		return err
	}
	registry := buildRegistry(svc, s.cfg, s.rootCfg)
	app := s.buildApp(svc, registry)
	_, err = s.present(app, tape)
	if err != nil {
		return err
	}
	return s.emitRecorded(recorder)
}

func (s *Session) validate() error {
	if s.cfg.Chdir != "" {
		if err := validateChdir(s.cfg); err != nil {
			return err
		}
	}
	if !s.macro.Active() && !hasTTY() {
		return fmt.Errorf("no TTY detected (terminal required for interactive mode)\n\nFor non-interactive use:\n  tfui plan --ci            (CI mode, no TUI)\n  CI=1 tfui plan            (same via env var)")
	}
	return nil
}

func (s *Session) buildService() (sdk.Service, *terraform.MacroService, error) {
	cache := terraform.NewServiceCache()
	if !s.sources.Empty() {
		if err := s.sources.Seed(cache); err != nil {
			return nil, nil, err
		}
	}
	if s.macro.Active() {
		logging.Logger().Debug("session.backend", "type", "recording", "reason", "macro")
		svc := terraform.NewMacroService(s.cfg.TerraformBinary(), cache)
		return svc, svc, nil
	}
	logging.Logger().Debug("session.backend", "type", "exec")
	svc := tfexec.NewExecService(effectiveWorkDir(s.cfg), s.cfg.TerraformBinary(), cache)
	return svc, nil, nil
}

func (s *Session) buildApp(svc sdk.Service, registry *plugin.Registry) ui.App {
	return ui.NewApp(s.cfg, svc, registry, s.rootCfg)
}

func (s *Session) present(app ui.App, tape []macro.Command) (ui.App, error) {
	if s.macro.Active() {
		driver := macro.NewDriver(app, 80, 24)
		if tape != nil {
			runner := macro.NewRunner(driver)
			if s.macro.RecordDir != "" {
				rec := macro.NewRecorder(nil, s.macro.RecordDir, 80, 24)
				runner.WithRecorder(rec)
			}
			return app, runner.Execute(tape)
		}
		return app, nil
	}

	opts := []tea.ProgramOption{tea.WithAltScreen()}

	if s.macro.RecordDir != "" {
		rec := macro.NewRecorder(app, s.macro.RecordDir, 80, 24)
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
}

func (s *Session) emitRecorded(recorder *terraform.MacroService) error {
	if recorder == nil {
		return nil
	}
	s.effects.WriteRecordedCommands(recorder.Commands(), s.jsonStdout)
	return nil
}

// SilentStderr reports whether stderr should be silenced (no rich TUI). Set
// from --ci, CI=1, or non-TTY stderr at PersistentPreRunE time.
func (s *Session) SilentStderr() bool {
	return s.silentStderr
}

// JSONStdout reports whether the caller asked for JSON-shaped stdout. cmd-side
// per-plugin command builders copy this into each plugin's Input.JSON.
func (s *Session) JSONStdout() bool {
	return s.jsonStdout
}

// resolveSilentStderr derives the stderr-silence boolean from --ci, CI=1, and
// stderr TTY status. Called once at PersistentPreRunE time; the result is
// cached on s.silentStderr and read by every dispatch path.
func (s *Session) resolveSilentStderr() bool {
	if s.ciMode {
		return true
	}
	if os.Getenv("CI") == "1" {
		return true
	}
	return !isStderrTTY()
}

// RunPlugin is the uniform per-plugin execution helper. Every per-plugin cobra
// command calls this once with a typed-Input activator. The function:
//  1. Resolves the service backend (ExecService normally; MacroService when
//     --macro is set).
//  2. Builds the plugin registry and standalone App.
//  3. Calls activate(plugin) to apply the typed Input to plugin state and
//     capture the initial tea.Cmd. The Cmd is plumbed into the App via
//     StandaloneConfig so the TUI processes it identically to the user-driven
//     path.
//  4. Runs the model headlessly (silent stderr) or under a real BubbleTea
//     program on stderr (rich interface).
//  5. Pumps the output port: MacroService recorder → stdout (cmd-formatted per
//     --json); else StdoutEmitter.Stdout() → stdout, StderrEmitter.Stderr() →
//     stderr; ExitCoder.ExitCode() → process exit code.
func (s *Session) RunPlugin(_ context.Context, pluginID string, activate func(sdk.Plugin) tea.Cmd) error {
	if err := s.validatePlugin(pluginID); err != nil {
		return err
	}
	tape, err := s.macro.LoadTape()
	if err != nil {
		return err
	}

	cache := terraform.NewServiceCache()
	if !s.sources.Empty() {
		if err := s.sources.Seed(cache); err != nil {
			return err
		}
	}
	var svc sdk.Service
	var recorder *terraform.MacroService
	if s.macro.Active() {
		recorder = terraform.NewMacroService(s.cfg.TerraformBinary(), cache)
		svc = recorder
	} else {
		svc = tfexec.NewExecService(effectiveWorkDir(s.cfg), s.cfg.TerraformBinary(), cache)
	}

	registry := buildRegistry(svc, s.cfg, s.rootCfg)
	standalone := &ui.StandaloneConfig{
		PluginID: pluginID,
		Activate: activate,
	}
	app := ui.NewApp(s.cfg, svc, registry, s.rootCfg, standalone)

	if s.silentStderr {
		driver := macro.NewDriver(app, 80, 24)
		if tape != nil {
			runner := macro.NewRunner(driver)
			if s.macro.RecordDir != "" {
				rec := macro.NewRecorder(nil, s.macro.RecordDir, 80, 24)
				runner.WithRecorder(rec)
			}
			if err := runner.Execute(tape); err != nil {
				return err
			}
		} else {
			driver.Init()
			if err := driver.WaitUntil(func(_ string) bool {
				if p, ok := registry.ByID(pluginID); ok {
					return p.Ready() || terminalStatus(p)
				}
				return false
			}, 10*time.Minute); err != nil {
				return err
			}
		}
	} else {
		opts := []tea.ProgramOption{tea.WithAltScreen(), tea.WithOutput(s.effects.Stderr)}
		if s.macro.RecordDir != "" {
			rec := macro.NewRecorder(app, s.macro.RecordDir, 80, 24)
			_, runErr := tea.NewProgram(rec, opts...).Run()
			_ = rec.Finalize()
			if runErr != nil {
				return runErr
			}
		} else {
			if _, err := tea.NewProgram(app, opts...).Run(); err != nil {
				return err
			}
		}
	}

	// Pump output port.
	if recorder != nil {
		s.effects.WriteRecordedCommands(recorder.Commands(), s.jsonStdout)
		return nil
	}
	p, ok := registry.ByID(pluginID)
	if !ok {
		return nil
	}
	if emitter, ok := p.(sdk.StdoutEmitter); ok {
		data, err := emitter.Stdout()
		if err != nil {
			return err
		}
		s.effects.WriteStdout(data)
	}
	if emitter, ok := p.(sdk.StderrEmitter); ok {
		s.effects.WriteStderr(emitter.Stderr())
	}
	if coder, ok := p.(sdk.ExitCoder); ok {
		s.effects.ExitWithCode(coder.ExitCode())
	}
	return nil
}

// validatePlugin runs the lightweight pre-dispatch checks shared with the
// legacy Run path (chdir validation, TTY check). Per-plugin commands hand
// control here from cobra's RunE.
func (s *Session) validatePlugin(_ string) error {
	if s.cfg.Chdir != "" {
		if err := validateChdir(s.cfg); err != nil {
			return err
		}
	}
	return nil
}

// terminalStatus reports whether a plugin has reached a terminal lifecycle
// state — used to short-circuit the macro driver's WaitUntil when the plugin
// has no Ready() bridge for terminal Done/Error states.
func terminalStatus(p sdk.Plugin) bool {
	if sr, ok := p.(sdk.Statusable); ok {
		st := sr.Status()
		return st == sdk.StatusDone || st == sdk.StatusError
	}
	return false
}
