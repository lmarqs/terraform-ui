# ADR-0021: Plugins as Use Cases (Hexagonal Architecture)

**Status:** Accepted (2026-05)

## Context

The coding rule "plugins import only `pkg/sdk`, never `internal/`" was framed as a convention without naming the underlying architecture. This allowed two drifts:

1. **OneShot** — a `Run(ctx context.Context, args []string) Result` interface that put CLI parsing inside the plugin. cobra parsed typed flags → re-stringified into `[]string` → plugin reparsed. Two grammars, two parsers, and the plugin knew about CLI grammar (a boundary violation the import rule didn't catch).

2. **cmdadapter** — an `internal/cmdadapter` package with `Deps` interface and `RunFn` type that made plugins import both the adapter and cobra. The plugin now imported its driving adapter — the same hexagonal violation in a different shape.

Both were rolled back. This ADR names the principle so the pattern doesn't repeat.

## Decision

tfui follows **hexagonal architecture** (ports and adapters):

### Inner hexagon: plugins are use cases

Plugins live in `plugins/*`. Their job is domain logic: the BubbleTea model, the lifecycle, and calls to the Service interface. They don't know whether they're driven by cobra, a macro tape, or a unit test.

### Port boundary: `pkg/sdk`

Everything a use case depends on from the outside world — `Service`, `Context`, events, logger, UI primitives — is exposed as an `sdk.*` interface or type. Plugins import this and only this from outside their own package.

### Adapters live outside the hexagon

| Adapter | Direction | Location |
|---------|-----------|----------|
| cobra commands | Driving | `cmd/tfui/*_command.go` |
| BubbleTea program | Driving | `cmd/tfui/session.go` |
| Macro driver/runner | Driving | `cmd/tfui/session.go` |
| ExecService | Driven | `internal/terraform/exec/` |
| MacroService | Driven | `internal/terraform/` |

### Input port: typed Input + Activate

Plugins reachable via CLI verbs receive parsed input through a typed `Input` struct and an `Activate(input Input) tea.Cmd` method. cobra wiring in `cmd/tfui/` parses flags into the Input and calls `Session.RunPlugin(ctx, pluginID, activate)`.

`--json` flows as `Input.JSON bool`. The SDK does nothing with it; the plugin decides what (if anything) to do.

### Output port: channel-specific emitter interfaces

Plugins implement only the channels they have content for:

- `StdoutEmitter.Stdout() ([]byte, error)` — bytes for stdout
- `StderrEmitter.Stderr() []byte` — post-quit stderr (warnings, summaries)
- `ExitCoder.ExitCode() int` — process exit code

The framework (`Session.RunPlugin`) pumps each emitter to the matching sink after the model completes.

### Three independent flags, three channels

| Flag | Channel | Effect |
|------|---------|--------|
| `--ci` | stderr | Suppress rich TUI (macro driver runs model headlessly) |
| `--json` | stdout (indirectly) | Boolean handed to plugin via `Input.JSON`; plugin decides |
| `--macro` | input + backend | Tape drives input; MacroService records terraform calls |

The flags compose independently. No enum, no axes pair, no precedence. Each is root-persistent.

### Uniform execution

Every per-plugin cobra command calls `Session.RunPlugin(ctx, pluginID, activate)`. The plugin's model always runs the same way regardless of which adapter drives it — the difference is only which sinks receive output and which source provides input.

## Lessons

### a. OneShot's `args []string` = stringly-typed boundary

Putting `[]string` parsing in the plugin made the plugin know about CLI grammar. The typed Input eliminates this — cobra owns parsing, plugin owns domain.

### b. cmdadapter violated the hexagonal boundary

Making plugins import `internal/cmdadapter` (or cobra) means the use case depends on its driving adapter. The fix: plugins expose `Activate(input)`, cmd-side files call it.

### c. `Outputter.Output(json bool)` coupled SDK to format decision

The `json bool` parameter forced every implementer to branch on a framework-level concept. Moving `--json` to `Input.JSON` delegates fully — the plugin reads it in Activate and decides what `Stdout()` returns. No double terraform calls needed for passthrough.


## Consequences

### Import rule (enforced)

| Package | Import rule | Reason |
|---------|-------------|--------|
| `pkg/sdk` | use cases import this | the port |
| `plugins/*` | import only `pkg/sdk` and stdlib | use cases must not depend on adapters |
| `internal/*` | adapter-side; cmd/tfui imports as needed | adapters live outside the hexagon |
| `cmd/tfui` | imports everything | composition root |
| external (cobra, BubbleTea direct ref) | NOT imported by plugins | adapters belong in cmd or internal |

**CRITICAL:** If you find yourself wanting to import cobra or any cmd-side type from a plugin, stop — the use case receives parsed inputs, not parses them. Add the cobra wiring to `cmd/tfui/<plugin>_command.go` instead.

### CI termination invariant

`--ci` runs the same plugin model as the TUI path. For it to terminate, the plugin's typed Input must fully parameterize the lifecycle (e.g., `AutoApprove` skips prompts). Plugins that don't expose a non-interactive path will hang — enforced by per-plugin CI-termination tests.
