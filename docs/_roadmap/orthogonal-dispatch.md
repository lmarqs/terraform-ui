---
title: Orthogonal CLI Dispatch (decompose flag axes)
status: planned
priority: high
created: 2026-05-16
effort: medium
tags: [cli, architecture, refactor]
depends_on: []
---

## Summary

Redesign the CLI execution layer (`cmd/tfui/main.go`) to treat flags as independent, composable axes instead of a tangled priority chain. The current code has three layers of messiness: dispatch (repeated if/else per command), runners (three near-identical functions), and finalization (duplicated output extraction).

## Problem: Three Layers of Mess

### Layer 1: Dispatch (7 copies of the same if/else)

Every subcommand repeats:

```go
if macroURI != "" {
    return runStandaloneMacro(...)
}
mode := resolveMode(ciMode)
if mode == modeCI {
    return runCI(...)
}
return runStandalone(...)
```

This is 7 copies with only `pluginID` and `jsonMode` varying. Adding a new axis (e.g., `--dry-run`) requires editing all 7 commands.

### Layer 2: Runners (three functions sharing 80% code)

```
runStandalone()       = seedCache + buildExecService + buildRegistry + buildApp + tea.Program + emitOutput
runCI()               = seedCache + buildExecService + buildRegistry + buildApp + driver       + emitOutput
runStandaloneMacro()  = seedCache + buildMacroService + buildRegistry + buildApp + driver      + printCommands
```

The differences are:
- Service type: `ExecService` vs `MacroService` (one line)
- Execution: `tea.Program` vs `macro.Driver` (5 lines)
- Finalization: `Output()` + `ExitCode()` vs `Commands()` (5 lines)

Everything else (cache seeding, chdir validation, registry building, app construction) is identical — copy-pasted across three functions.

### Layer 3: Finalization (duplicated output extraction)

Both `runStandalone` and `runCI` have:

```go
if outputter, ok := plugin.(sdk.Outputter); ok {
    data, err := outputter.Output(jsonMode)
    // ...
    os.Stdout.Write(data)
}
if coder, ok := plugin.(sdk.ExitCoder); ok {
    code := coder.ExitCode()
    if code != 0 { os.Exit(code) }
}
```

This is the same 10-line pattern in two places. A third pattern exists for macro mode (print commands to stdout).

### Layer 4: Root command duality

The root command (`tfui` no args) has its own completely separate world:

```go
if macroURI != "" {
    return runMacro(...)  // full TUI macro (yet another runner function)
}
return runTUI(...)        // full TUI (yet another runner function)
```

`runMacro` and `runStandaloneMacro` are 90% identical. `runTUI` and `runStandalone` are 70% identical. Five runner functions total, with massive overlap.

### The Root Cause

Flags represent **orthogonal axes** but the code treats them as a **priority chain**:

```
macro? → macro path
ci?    → ci path
else   → tui path
```

This makes combinations undefined. What does `--macro --ci` mean? What about `--macro --plan file -json`? The code doesn't compose — it picks one winner.

## The Orthogonal Axes

Each flag controls exactly one independent axis:

| Axis | Flag | Values | What it determines |
|------|------|--------|-------------------|
| **Service** | `--macro` | Exec (default) / Macro | What responds to `Plan()`, `Apply()`, etc. |
| **Render** | `--ci` / `CI=1` / no-stderr-TTY | TUI / Headless | Whether the user sees an interactive UI |
| **Format** | `-json` | Human / JSON | What `Output(json)` produces |
| **Source** | `--plan`, `--state` | Live / Pre-seeded | Where data comes from before plugin starts |
| **Scope** | subcommand name / no-args | Standalone plugin / Full TUI | Which plugins are loaded/active |

Every combination of these 5 axes is valid and meaningful:

```
tfui plan --macro tape --ci --plan file -json
  = MacroService + Headless + JSON + Pre-seeded + Standalone(plan)
  = "Drive plan plugin headlessly with tape, using macro service,
     pre-seeded from file, output JSON to stdout, print commands after"
```

## Proposed Design

### Single `run()` function

```go
type RunConfig struct {
    // Scope
    PluginID string   // "" = full TUI
    Args     []string

    // Axes (resolved from flags)
    Service  ServiceAxis  // exec | macro
    Render   RenderAxis   // tui | headless
    Format   FormatAxis   // human | json
    Source   SourceAxis   // planURI, stateURI (empty = live)
    MacroURI string       // tape file (empty = no tape, only for macro service)
}

func run(cfg config.Config, rootCfg *config.RootConfig, rc RunConfig) error {
    // 1. Source axis: build cache, seed if needed
    cache := terraform.NewServiceCache()
    if rc.Source.HasPreseeded() {
        cfg.PreloadedData = true
        if err := seedCache(cache, rc.Source.PlanURI, rc.Source.StateURI); err != nil {
            return err
        }
    }

    // 2. Service axis: build the appropriate service
    svc, macroSvc := buildService(cfg, cache, rc.Service)

    // 3. Scope axis: build app (standalone or full)
    registry := buildRegistry(svc, cfg)
    app := buildApp(cfg, svc, registry, rootCfg, rc)

    // 4. If macro service with tape: parse and prepare runner
    var tapeCommands []macro.Command
    if rc.MacroURI != "" {
        var err error
        tapeCommands, err = loadTape(rc.MacroURI)
        if err != nil { return err }
    }

    // 5. Render axis: run TUI or headless
    plugin := execute(app, registry, rc, tapeCommands)

    // 6. Finalization: output + exit code + macro commands
    return finalize(plugin, rc.Format, macroSvc)
}
```

### Each subcommand becomes a one-liner

```go
planCmd.RunE = func(cmd *cobra.Command, args []string) error {
    return run(cfg, rootCfg, RunConfig{
        PluginID: "plan",
        Args:     args,
        Service:  serviceFromFlag(macroURI),
        Render:   renderFromFlags(ciMode),
        Format:   formatFromFlag(jsonMode),
        Source:   SourceAxis{PlanURI: planURI, StateURI: stateURI},
        MacroURI: macroURI,
    })
}
```

### Root command also uses `run()`

```go
rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
    return run(cfg, rootCfg, RunConfig{
        PluginID: "", // empty = full TUI
        Service:  serviceFromFlag(macroURI),
        Render:   renderFromFlags(false), // root always renders TUI
        Source:   SourceAxis{PlanURI: planURI, StateURI: stateURI},
        MacroURI: macroURI,
    })
}
```

### Helper functions (composed, not branched)

```go
func buildService(cfg, cache, axis) (sdk.Service, *MacroService) {
    // One switch, returns the right service
}

func buildApp(cfg, svc, registry, rootCfg, rc) ui.App {
    // If PluginID != "" → standalone config, else full TUI
}

func execute(app, registry, rc, tapeCommands) sdk.Plugin {
    // If tapeCommands != nil → drive with tape runner
    // Else if Render == TUI → tea.NewProgram
    // Else → headless driver waiting for Ready
}

func finalize(plugin, format, macroSvc) error {
    // Always: emit Output() if Outputter
    // Always: check ExitCode() if ExitCoder
    // If macroSvc: print recorded commands
}
```

## What This Fixes

| Current problem | After refactor |
|----------------|----------------|
| 7 copies of dispatch if/else | 7 one-liner `run()` calls |
| 5 runner functions (80% identical) | 1 `run()` function |
| Duplicated output extraction | 1 `finalize()` function |
| Undefined flag combinations (`--macro --ci`) | All combinations valid by construction |
| Adding a new axis requires editing 7+ places | Adding a new axis = add field to RunConfig + one helper |
| Root command is a separate world | Root command uses same `run()` with `PluginID: ""` |

## What Gets Deleted

- `runStandalone()` (~30 lines)
- `runCI()` (~30 lines)
- `runStandaloneMacro()` (~45 lines)
- `runMacro()` (~50 lines)
- `runTUI()` (~30 lines)
- `resolveMode()` (~10 lines)
- 7 × repeated if/else blocks (~50 lines)
- **Total: ~245 lines deleted**

## What Gets Added

- `RunConfig` struct (~15 lines)
- `run()` orchestrator (~40 lines)
- `buildService()` (~10 lines)
- `buildApp()` (~10 lines)
- `execute()` (~25 lines)
- `finalize()` (~15 lines)
- `loadTape()` (~15 lines)
- Axis resolution helpers (~15 lines)
- **Total: ~145 lines added**

**Net: ~100 lines deleted, and 1 code path instead of 5.**

## Verification

```bash
# All axis combinations work:
tfui plan                                           # exec + tui + human + live
tfui plan --ci                                      # exec + headless + human + live
tfui plan -json                                     # exec + tui + json + live
tfui plan --ci -json                                # exec + headless + json + live
tfui plan --plan file                               # exec + tui + human + pre-seeded
tfui plan --ci --plan file                          # exec + headless + human + pre-seeded
tfui plan --macro tape                              # macro + tui + human + live
tfui plan --macro tape --ci                         # macro + headless + human + live
tfui plan --macro tape --plan file                  # macro + tui + human + pre-seeded
tfui plan --macro tape --ci --plan file -json       # macro + headless + json + pre-seeded
tfui                                                # full TUI (PluginID="")
tfui --macro tape --plan file                       # full TUI macro

# Tests still pass:
mise run test:unit
mise run test:macro
```

## Risk

Low. Pure refactor of `cmd/tfui/main.go`. No plugin, SDK, or app changes. The behavioral contract (what each flag combination produces) is unchanged — only the internal wiring is cleaned up.

## Design Patterns

Three named patterns compose the solution:

1. **Builder** (`RunConfig`) — each flag contributes one field to a config struct. The struct IS the composition. No if/else decides what to build — flags declare it.

2. **Strategy** (per-axis behavior) — `Service`, `Render`, `Format` are swappable behaviors, not switch branches. `execute()` receives an `Executor` interface, not a mode enum.

3. **Pipeline** (execution flow) — `seed → build → execute → finalize` is a linear chain where each stage's output feeds the next. Stages are testable independently.

Compositional sketch:

```go
NewRunner().
    WithService(macroURI).         // strategy: exec or macro
    WithRender(ciMode).            // strategy: tui or headless
    WithSource(planURI, stateURI). // builder: data seeding
    WithFormat(jsonMode).          // builder: output format
    Run("plan", args)              // pipeline: seed → build → execute → finalize
```

## Related Documentation

- [CLI I/O Contract](../cli-io-contract.md) — defines what each flag combination produces on stdout/stderr
- [CLI UX Guidelines](../cli-ux.md) — documents the two-mode model (TUI/CI) and flag scoping rules
- [CLI Reference](../cli-reference.md) — user-facing flag descriptions and examples
- [Architecture Overview](../architecture.md) — app model, plugin routing, service layer
- [Macro Language](../macro-language.md) — tape DSL that drives the macro service axis
- [Testing Strategy](../testing.md) — macro tapes as integration tests (affected by standalone macro)
- [ADR-0006: Orthogonal Income/Outcome](../adr/0006-orthogonal-income-outcome.md) — prior decision on separating input sources from output formats
- [Roadmap: Interactive Command Recording](interactive-command-recording.md) — related: `--dry-run` would be a 6th axis (execution intent)

### Source files

- `cmd/tfui/main.go` — the dispatch layer being refactored
- `cmd/tfui/cli.go` — imperative commands (workspace, force-unlock) that bypass dispatch
- `internal/ui/app.go` — `StandaloneConfig` and the app model's dual behavior
- `internal/macro/driver.go` — headless execution driver (used by both CI and macro)
- `internal/terraform/service.go` — `ExecService` (exec axis)
- `internal/terraform/macro_service.go` — `MacroService` (macro axis)
- `pkg/sdk/plugin.go` — `Outputter`, `ExitCoder`, `ActivateWithArgs` interfaces (finalization contracts)

### Agent rules

- `.claude/rules/ux-cli.md` — flag scoping rules, must be updated when dispatch changes
- `.claude/rules/architecture.md` — service layer docs, macro engine docs

## Dependencies

None. This is entirely within `cmd/tfui/main.go`.
