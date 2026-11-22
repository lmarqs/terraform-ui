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
	dryRun       bool
	silentStderr bool // resolved at PersistentPreRunE: --ci || CI=1 || agent env || !isStderrTTY

	guard *interruptGuard // the last run's; exitCode asks it for a late signal
}

// exitCode is the process exit code for a run that ended with code. The run's
// guard keeps the signals until the process exits, so one that arrived after
// the run returned still decides the code.
func (s *Session) exitCode(code int) int {
	if s.guard == nil {
		return code
	}
	if sig := s.guard.Late(); sig != nil {
		return interruptExitCode(sig)
	}
	return code
}

func (s *Session) Run() error {
	if err := s.validate(); err != nil {
		return err
	}
	return s.runApp()
}

// runApp runs the full TUI once validate has passed.
func (s *Session) runApp() error {
	tape, err := s.macro.LoadTape()
	if err != nil {
		return err
	}
	svc, recorder, err := s.buildService()
	if err != nil {
		return err
	}
	registry := s.buildRegistry(svc)
	guard, shutdown := startInterruptGuard(svc, s.effects.Stderr, registry.Shutdown)
	s.guard = guard
	defer shutdown()
	app := s.buildApp(svc, registry)
	_, err = s.present(app, tape, guard)
	sig, cutShort := guard.Finish()
	if cutShort {
		sig = os.Interrupt
	}
	if sig != nil {
		return &exitError{code: interruptExitCode(sig)}
	}
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
	if s.dryRun && !s.macro.Active() && !isTerminal(s.effects.Stderr) {
		return fmt.Errorf("no TTY on stderr: -dry-run draws the TUI on stderr and prints the commands on stdout\n\nKeep stderr on the terminal; redirect stdout only:\n  tfui -dry-run > commands.txt")
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
	if s.recordsCommands() {
		logging.Logger().Debug("session.backend", "type", "recording", "macro", s.macro.Active(), "dry_run", s.dryRun)
		svc := terraform.NewMacroService(s.cfg.TerraformBinary(), cache, effectiveWorkDir(s.cfg))
		return svc, svc, nil
	}
	logging.Logger().Debug("session.backend", "type", "exec")
	svc := tfexec.NewExecService(effectiveWorkDir(s.cfg), s.cfg.TerraformBinary(), cache)
	return svc, nil, nil
}

func (s *Session) recordsCommands() bool {
	return s.macro.Active() || s.dryRun
}

// buildRegistry leaves Console out of a dry run: it runs `terraform console`
// itself, outside the service, so the recording service cannot stop it.
func (s *Session) buildRegistry(svc sdk.Service) *plugin.Registry {
	if !s.dryRun {
		return buildRegistry(svc, s.cfg, s.rootCfg)
	}
	cfg := s.cfg
	cfg.Plugins = make(map[string]config.PluginConfig, len(s.cfg.Plugins)+1)
	for id, pc := range s.cfg.Plugins {
		cfg.Plugins[id] = pc
	}
	disabled := false
	console := cfg.Plugins["console"]
	console.Enabled = &disabled
	cfg.Plugins["console"] = console
	return buildRegistry(svc, cfg, s.rootCfg)
}

func (s *Session) buildApp(svc sdk.Service, registry *plugin.Registry) ui.App {
	return ui.NewApp(s.cfg, svc, registry, s.rootCfg)
}

func (s *Session) present(app ui.App, tape []macro.Command, guard *interruptGuard) (ui.App, error) {
	if s.macro.Active() {
		playback, stopPlayback := context.WithCancel(context.Background())
		defer stopPlayback()
		guard.Watch(stopPlayback)
		driver := macro.NewDriver(app, 80, 24)
		if tape != nil {
			runner := macro.NewRunner(driver)
			if s.macro.RecordDir != "" {
				rec := macro.NewRecorder(nil, s.macro.RecordDir, 80, 24)
				runner.WithRecorder(rec)
			}
			return app, runner.Execute(playback, tape)
		}
		return app, nil
	}

	opts := []tea.ProgramOption{tea.WithAltScreen(), tea.WithoutSignalHandler()}
	if s.effects.Stdin != nil {
		opts = append(opts, tea.WithInput(s.effects.Stdin))
	}
	if s.dryRun {
		// stdout carries only the recorded commands, so it stays clean in a pipe.
		opts = append(opts, tea.WithOutput(s.effects.Stderr))
	}

	if s.macro.RecordDir != "" {
		rec := macro.NewRecorder(app, s.macro.RecordDir, 80, 24)
		_, err := runGuarded(tea.NewProgram(rec, opts...), guard)
		if finalErr := rec.Finalize(); err == nil {
			err = finalErr
		}
		if err != nil {
			return app, err
		}
		if inner, ok := rec.Inner().(ui.App); ok {
			return inner, nil
		}
		return app, nil
	}

	model, err := runGuarded(tea.NewProgram(app, opts...), guard)
	if err != nil {
		return app, err
	}
	if a, ok := model.(ui.App); ok {
		return a, nil
	}
	return app, nil
}

// runGuarded runs program with the first signal quitting it.
func runGuarded(program *tea.Program, guard *interruptGuard) (tea.Model, error) {
	guard.Watch(program.Quit)
	return program.Run()
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

// resolveSilentStderr derives the stderr-silence boolean from --ci, headless
// environment variables, and stderr TTY status. Called once at
// PersistentPreRunE time; the result is cached on s.silentStderr and read by
// every dispatch path.
func (s *Session) resolveSilentStderr() bool {
	return resolveSilentStderrFrom(s.ciMode, os.Getenv, isStderrTTY)
}

// resolveSilentStderrFrom is the pure mode-resolution rule behind
// resolveSilentStderr. Headless is chosen for --ci, CI=1, agent environments
// (CLAUDECODE / AI_AGENT — coding agents attach a PTY, so TTY detection alone
// would launch a TUI their session can never drive), or a non-TTY stderr.
func resolveSilentStderrFrom(ciMode bool, getenv func(string) string, stderrIsTTY func() bool) bool {
	if ciMode {
		return true
	}
	if getenv("CI") == "1" {
		return true
	}
	if getenv("CLAUDECODE") != "" || getenv("AI_AGENT") != "" {
		return true
	}
	return !stderrIsTTY()
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
func (s *Session) RunPlugin(ctx context.Context, pluginID string, activate func(sdk.Plugin) tea.Cmd) error {
	return s.RunPluginFrom(ctx, pluginID, pluginID, activate)
}

// RunPluginFrom is RunPlugin for a verb whose run starts in another plugin:
// activate receives the entry plugin, which hands over to the verb's plugin
// (pluginID). The output port belongs to the verb's plugin; when the entry
// plugin fails before the hand-over, its stderr and exit code are pumped
// instead (outputPortOwner).
func (s *Session) RunPluginFrom(_ context.Context, pluginID, entryID string, activate func(sdk.Plugin) tea.Cmd) error {
	if err := s.validatePlugin(pluginID); err != nil {
		return err
	}
	if entryID != pluginID {
		if err := s.validatePlugin(entryID); err != nil {
			return err
		}
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
	guard, shutdown := startInterruptGuard(svc, s.effects.Stderr, registry.Shutdown)
	s.guard = guard
	defer shutdown()
	standalone := &ui.StandaloneConfig{
		PluginID: pluginID,
		EntryID:  entryID,
		Activate: activate,
	}
	app := ui.NewApp(s.cfg, svc, registry, s.rootCfg, standalone)

	runErr := s.runStandalone(app, registry, pluginID, tape, guard)
	sig, cutShort := guard.Finish()
	if sig == nil && runErr != nil {
		return runErr
	}
	if cutShort {
		sig = os.Interrupt
	}

	if recorder != nil && sig == nil {
		s.effects.WriteRecordedCommands(recorder.Commands(), s.jsonStdout)
		return nil
	}
	return s.pumpOutputPort(registry, pluginID, entryID, sig)
}

// runStandalone runs the verb's App headlessly (silent stderr) or under a
// BubbleTea program on stderr, until the verb finishes or a signal ends it.
func (s *Session) runStandalone(app ui.App, registry *plugin.Registry, pluginID string,
	tape []macro.Command, guard *interruptGuard) error {
	if !s.silentStderr {
		opts := []tea.ProgramOption{tea.WithAltScreen(), tea.WithOutput(s.effects.Stderr), tea.WithoutSignalHandler()}
		if s.macro.RecordDir != "" {
			rec := macro.NewRecorder(app, s.macro.RecordDir, 80, 24)
			_, err := runGuarded(tea.NewProgram(rec, opts...), guard)
			if finalErr := rec.Finalize(); err == nil {
				err = finalErr
			}
			return err
		}
		_, err := runGuarded(tea.NewProgram(app, opts...), guard)
		return err
	}

	run, endRun := context.WithCancel(context.Background())
	defer endRun()
	guard.Watch(endRun)
	driver := macro.NewDriver(app, 80, 24)
	if tape != nil {
		runner := macro.NewRunner(driver)
		if s.macro.RecordDir != "" {
			rec := macro.NewRecorder(nil, s.macro.RecordDir, 80, 24)
			runner.WithRecorder(rec)
		}
		return runner.Execute(run, tape)
	}
	driver.Init()
	return driver.WaitUntil(run, func(_ string) bool {
		if p, ok := registry.ByID(pluginID); ok {
			return p.Ready() || terminalStatus(p)
		}
		return false
	}, 10*time.Minute)
}

// pumpOutputPort writes the verb's stdout, then the stderr and exit code of
// the plugin outputPortOwner picks; a non-zero exit code comes back as an
// exitError. After a signal (interrupted non-nil) it writes no stdout and the
// signal's exit code replaces the plugin's.
func (s *Session) pumpOutputPort(registry *plugin.Registry, pluginID, entryID string, interrupted os.Signal) error {
	p, ok := registry.ByID(pluginID)
	if !ok {
		return nil
	}
	entry, _ := registry.ByID(entryID)
	failure := outputPortOwner(p, entry)
	if interrupted != nil {
		if emitter, ok := failure.(sdk.StderrEmitter); ok {
			s.effects.WriteStderr(emitter.Stderr())
		}
		return &exitError{code: interruptExitCode(interrupted)}
	}
	if emitter, ok := p.(sdk.StdoutEmitter); ok {
		data, err := emitter.Stdout()
		if err != nil {
			return err
		}
		s.effects.WriteStdout(data)
	}
	if emitter, ok := failure.(sdk.StderrEmitter); ok {
		s.effects.WriteStderr(emitter.Stderr())
	}
	if coder, ok := failure.(sdk.ExitCoder); ok && coder.ExitCode() != 0 {
		return &exitError{code: coder.ExitCode()}
	}
	return nil
}

// outputPortOwner picks the plugin whose stderr and exit code report the run:
// the verb's plugin, unless it never left idle because the entry plugin failed
// before handing over.
func outputPortOwner(verb, entry sdk.Plugin) sdk.Plugin {
	if statusOf(verb) == sdk.StatusIdle && statusOf(entry) == sdk.StatusError {
		return entry
	}
	return verb
}

// validatePlugin runs the lightweight pre-dispatch checks shared with the
// legacy Run path (chdir validation, TTY check). Per-plugin commands hand
// control here from cobra's RunE.
func (s *Session) validatePlugin(pluginID string) error {
	if s.cfg.Chdir != "" {
		if err := validateChdir(s.cfg); err != nil {
			return err
		}
	}
	// A disabled plugin never reaches the registry, so there is nothing for the
	// dispatch path to activate — saying so beats exiting 0 having done nothing.
	if pc, ok := s.cfg.Plugins[pluginID]; ok && !pc.IsEnabled() {
		return fmt.Errorf("plugin %q is disabled in tfui.hcl\n\nRemove `enabled = false` from its plugin block, or drop the block entirely", pluginID)
	}
	return nil
}

// terminalStatus reports whether a plugin has reached a terminal lifecycle
// state — used to short-circuit the macro driver's WaitUntil when the plugin
// has no Ready() bridge for terminal Done/Error states.
func terminalStatus(p sdk.Plugin) bool {
	st := statusOf(p)
	return st == sdk.StatusDone || st == sdk.StatusError
}

// statusOf returns the plugin's lifecycle status, or StatusIdle when it does
// not report one.
func statusOf(p sdk.Plugin) sdk.Status {
	if sr, ok := p.(sdk.Statusable); ok {
		return sr.Status()
	}
	return sdk.StatusIdle
}
