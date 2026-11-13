---
layout: default
title: "ADR-0021: Plugins as use cases (hexagonal architecture)"
grand_parent: Development
parent: Architecture
nav_order: 0021
seo:
  description: Decision to treat plugins as use cases behind typed input and output ports, with CLI wiring outside the plugin
description: "Records why plugins are use cases behind typed input and output ports, with CLI wiring outside them."
audience: "Contributors adding a plugin or a CLI command"
---

# Plugins as use cases (hexagonal architecture)

tfui follows **hexagonal architecture** (ports and adapters). Plugins are the use cases, the inner hexagon: they live in `plugins/*`, and their job is domain logic: the BubbleTea model, the lifecycle, and calls to the Service interface. They don't know whether they're driven by cobra, a macro tape, or a unit test.

`pkg/sdk` is the port boundary. Everything a use case depends on from the outside world — `Service`, `Context`, events, logger, UI primitives — is exposed as an `sdk.*` interface or type. From outside their own package, plugins import `pkg/sdk`, BubbleTea for their model and commands, and the standard library.

ADR-0001 named the architecture, but the rule "plugins import only `pkg/sdk`, never `internal/`" was applied as an import check only, which allowed two drifts:

1. **OneShot** — a `Run(ctx context.Context, args []string) Result` interface that put CLI parsing inside the plugin. cobra parsed typed flags → re-stringified into `[]string` → plugin reparsed. Two grammars, two parsers, and the plugin knew about CLI grammar (a boundary violation the import rule didn't catch).

2. **cmdadapter** — an `internal/cmdadapter` package with `Deps` interface and `RunFn` type that made plugins import both the adapter and cobra. The plugin now imported its driving adapter — the same hexagonal violation in a different shape.

Both were rolled back. This record names the principle so the pattern doesn't repeat.

## Consequences

### Adapters live outside the hexagon

| Adapter | Direction | Location |
|---------|-----------|----------|
| cobra commands | Driving | `cmd/tfui/*_command.go` |
| BubbleTea program | Driving | `cmd/tfui/session.go` |
| Macro driver/runner | Driving | `internal/macro/` (wired in `cmd/tfui/session.go`) |
| ExecService | Driven | `internal/terraform/exec/` |
| MacroService | Driven | `internal/terraform/` |

### Input port: typed Input + Activate

Plugins reachable via CLI verbs receive parsed input through a typed `Input` struct and an `Activate(input Input) tea.Cmd` method. cobra wiring in `cmd/tfui/` parses flags into the Input and calls `Session.RunPlugin(ctx, pluginID, activate)`.

`-json` flows as `Input.JSON bool`. The SDK does nothing with it; the plugin decides what (if anything) to do.

### Output port: channel-specific emitter interfaces

Plugins implement only the channels they have content for:

- `StdoutEmitter.Stdout() ([]byte, error)` — bytes for stdout
- `StderrEmitter.Stderr() []byte` — post-quit stderr (warnings, summaries)
- `ExitCoder.ExitCode() int` — process exit code

The framework (`Session.RunPlugin`) pumps each emitter to the matching sink after the model completes.

### Three independent flags, three channels

| Flag | Channel | Effect |
|------|---------|--------|
| `-ci` | stderr | Suppress the TUI; the model runs headless under the macro driver (`internal/macro`) |
| `-json` | stdout (indirectly) | Boolean handed to plugin via `Input.JSON`; plugin decides |
| `-macro` | input + backend | Tape drives input; MacroService records terraform calls |

The flags compose independently. No enum, no axes pair, no precedence. Each is root-persistent.

### Uniform execution

Every per-plugin cobra command calls `Session.RunPlugin(ctx, pluginID, activate)`, or `Session.RunPluginFrom` when the run enters through another plugin (`tfui apply` without `-auto-approve` enters through plan). The plugin's model always runs the same way regardless of which adapter drives it — the difference is only which sinks receive output and which source provides input.

### Import rule (checked by hook and review)

| Package | Import rule | Reason |
|---------|-------------|--------|
| `pkg/sdk` | use cases import this | the port |
| `plugins/*` | import `pkg/sdk`, BubbleTea and stdlib | use cases must not depend on adapters |
| `internal/*` | adapter-side; cmd/tfui imports as needed | adapters live outside the hexagon |
| `cmd/tfui` | imports everything | composition root |
| cobra and `internal/*` | NOT imported by plugins | adapters belong in cmd or internal |

Cobra wiring for a plugin goes in `cmd/tfui/<plugin>_command.go`; the plugin receives parsed input and never parses it.

### CI termination invariant

`-ci` runs the same plugin model as the TUI path. For it to terminate, the plugin's typed Input must fully parameterize the lifecycle (e.g., `AutoApprove` skips prompts). Plugins that don't expose a non-interactive path will hang. CI-termination tests in `cmd/tfui/session_test.go` cover apply and taint.

## Considered Options

### OneShot: `Run(ctx, args []string)` (tried, rolled back)

A stringly-typed boundary: putting `[]string` parsing in the plugin made the plugin know about CLI grammar. The typed Input eliminates this — cobra owns parsing, plugin owns domain.

### cmdadapter (tried, rolled back)

Violated the hexagonal boundary: making plugins import `internal/cmdadapter` (or cobra) means the use case depends on its driving adapter. The fix: plugins expose `Activate(input)`, cmd-side files call it.

### `Outputter.Output(json bool)` (replaced)

Coupled the SDK to the format decision: the `json bool` parameter forced every implementer to branch on a framework-level concept. Moving `-json` to `Input.JSON` delegates fully — the plugin reads it in Activate and decides what `Stdout()` returns. No double terraform calls needed for passthrough.
