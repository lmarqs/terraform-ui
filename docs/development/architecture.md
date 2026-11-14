---
layout: default
title: Architecture
parent: Development
nav_order: 1
has_children: true
seo:
  description: Internal architecture of terraform-ui
description: "Explains how the SDK, app internals and plugins fit together, and which way dependencies point."
audience: "Contributors before a change that crosses packages"
---

# Architecture

terraform-ui is a Go application built with [Bubbletea](https://github.com/charmbracelet/bubbletea) (TUI framework) and [terraform-exec](https://github.com/hashicorp/terraform-exec) (Go library that runs the terraform binary). Features are modular plugins that depend only on a public SDK package.

This page maps each part to its directory and the decision behind it. How to test each part is in
[Testing](testing.md); TUI and CLI rules are in [TUI UX](../reference/tui-ux.md) and [CLI UX](../reference/cli-ux.md);
decisions are in [the ADRs](../adr/index.md).

## Dependency Direction

```
plugins/*  ───→  pkg/sdk  ←───  internal/*
   │                                 │
   │  (public contract only)         │  (implements sdk interfaces)
   │                                 │
   └── never imports internal/ ──────┘
```

Plugins depend exclusively on `pkg/sdk/` ([ADR-0001](../adr/0001-sdk-isolation-boundary.md)). Internal packages implement the interfaces defined there.

## Project Structure

```
cmd/tfui/                  — CLI entry point (thin glue)
  main.go                  — Cobra command tree (rootCmd.AddCommand)
  root.go                  — Root command and persistent flags
  registry.go              — buildRegistry(): every plugin factory with its PluginMeta
  session.go               — Session: Run (app mode) and RunPlugin (one plugin)
  <name>_command.go        — One file per subcommand; most parse flags into a plugin's Input
  normalize.go             — Terraform flag compatibility (normalizeArgs, splitPassthrough)
pkg/sdk/                   — Public SDK (the only dependency for plugins; key files below)
  plugin.go                — Plugin interface + optional interfaces
  plugin_deps.go           — PluginDeps, handed to Init()
  context.go, pins.go      — Immutable Context snapshot and its Pins
  service.go               — Service interface (terraform operations)
  command.go               — Command: one terraform invocation
  bus.go, events.go        — EventBus typed dispatch, events and handler interfaces
  action_runner.go         — ActionRunner: the action-mode lifecycle
  frame.go, stack.go       — Frame stack navigation primitives
  overlay.go, input.go     — Modal overlays and input system
  styles.go                — Style constants for consistent rendering
  frames/                  — Reusable frames (filter, inspect, confirm, action, form, stream)
  ui/                      — Widgets (cursor, filter, timer, tree, actions bar, content panel)
  sdktest/                 — Test harness, MockService, golden helpers
internal/
  config/                  — HCL config loading, project discovery, Resolve
  plugin/registry.go       — Plugin registry (host-side only)
  terraform/
    exec/
      service.go           — ExecService (implements sdk.Service via terraform-exec)
      state_ops.go         — State operations (rm, mv, import, taint, untaint)
      workspace_ops.go     — Workspace operations
      dir_lock.go          — Non-reentrant per-directory lock (ADR-0016)
    macro_service.go       — MacroService (records commands, reads from cache)
    service_cache.go       — ServiceCache (typed, source-aware data cache)
    loader.go              — LoadPlan / LoadState from raw bytes
    plan_parser.go         — Plan JSON parsing
    risk.go, phantom.go, grouping.go — Analysis over plan and state
  source/                  — Source layer (URI → bytes)
  macro/                   — Macro engine (driver, runner, tape parser, recorder)
  ui/
    app.go                 — Root Bubbletea model: routing, navigation, Context ownership
    components/            — Header, statusbar, command bar, content border
  editor/                  — Editor integration ($EDITOR at file:line)
  logging/                 — Structured logger setup
  scaffold/                — Project scaffolding (tfui.hcl generation)
  compliance/              — Plugin lifecycle compliance tests
plugins/
  context/                 — Context dashboard: Project + Chdir + Workspace
  chdir/                   — Chdir picker: select member from explicit list
  state/                   — State browser (list, inspect, pin, delete, move)
  plan/                    — Plan review (diff view, expand attributes, risk)
  apply/                   — Apply executor (the plan file from plan, or a one-shot apply)
  taint/                   — Taint (Action Mode): mark for recreation
  untaint/                 — Untaint (Action Mode): remove taint mark
  import/                  — Import (Action Mode): bring existing infra
  workspace/               — Workspace management
  console/                 — Terraform console (REPL)
  output/                  — Terraform outputs viewer
  validate/                — Terraform validate with diagnostics
  init/                    — Terraform init with form-based options
  risk/                    — Risk classification (decorates plan)
  phantom/                 — Phantom change detection (decorates plan)
  blastradius/             — Blast radius visualization
  forceunlock/             — Force unlock stale state locks
  version/                 — Version information display
tests/
  integration/             — CLI integration tests (require terraform)
  fixtures/                — Real terraform projects and config fixtures
```

## Plugin System

Every feature is a plugin implementing `sdk.Plugin` (`pkg/sdk/plugin.go`):

```go
type Plugin interface {
    ID() string
    Name() string
    Description() string
    Init(deps *PluginDeps) tea.Cmd
    Update(msg tea.Msg) (Plugin, tea.Cmd)
    View(width, height int) string
    Configure(cfg map[string]interface{}) error
    Ready() bool
}
```

Optional interfaces add capabilities the app detects by type assertion: `Activatable`, `Busy`, `Cancellable`,
`Countable`, `Hintable`, `KeyCapturer`, `Pinnable`, `Positionable`, `Stackable` (`frame.go`), `StdoutEmitter`,
`StderrEmitter`, `ExitCoder`, `Statusable`, `Shutdowner` (called on every registered plugin when the program ends).

A plugin invoked from the CLI is a use case with two ports ([ADR-0021](../adr/0021-plugins-as-use-cases.md)):

- **Input port:** a typed `Input` DTO and the plugin's own `Activate(input Input) tea.Cmd` method, distinct from
  `sdk.Activatable`, whose `Activate()` takes no input. The subcommand in `cmd/tfui/<name>_command.go` parses flags
  into the Input.
- **Output port:** `StdoutEmitter.Stdout() ([]byte, error)`, `StderrEmitter.Stderr() []byte` and
  `ExitCoder.ExitCode() int`; the framework pumps each to its sink.

Plugins are:
- **Self-contained** — own view logic, types, messages
- **Configurable** — per-plugin options in `tfui.hcl`
- **Independent** — import only `pkg/sdk`, never `internal/`, and never reference each other
- **Testable** — built with the test harness and `sdktest.MockService` ([Testing](testing.md#plugin-test-harness))

### PluginDeps and Context

`Init` receives `*sdk.PluginDeps` (`pkg/sdk/plugin_deps.go`), fixed for the process lifetime:

| Field | Purpose |
|-------|---------|
| `Logger` | Structured logger |
| `Service` | Unscoped terraform service, a fallback until the first `ContextChangedEvent` |
| `Context` | `func() *Context` returning the current snapshot |
| `Pin` | Toggles a group of pin addresses; triggers a Context replacement |
| `ClearPins` | Removes every pin |

`sdk.Context` (`pkg/sdk/context.go`) is an immutable snapshot of everything that shapes a terraform command: `Chdir`,
`WorkingDir`, `Workspace`, a `Service` scoped to the working directory, `Pins`, `VarFiles`, `Vars`, `ExtraArgs`,
`Parallelism`, `Lock` and `LockTimeout`. The app replaces it whole on every chdir, workspace or pin change
([ADR-0018](../adr/0018-atomic-context-replacement.md)); plugins never mutate it and react through
`ContextChangedHandler`. `WithPins` and `TogglePins` return a fresh snapshot.

Pins are the UI selection; targets are terraform's `-target=` flags. The boundary is `Context.PlanOptions()`, which
sets `Targets` from `Pins`. `Context.ApplyOptions()` carries no targets, because apply runs a saved plan file
([ADR-0019](../adr/0019-unidirectional-data-flow.md)). Pins are always resource addresses: a tree row resolves to them
through `Tree.PinAddresses(node)` in `pkg/sdk/ui/tree/` (a leaf gives its own address, a branch every descendant leaf), and the tree keeps no
pin state of its own, so a row cannot disagree with the Context.

### Event Bus

`pkg/sdk/bus.go` dispatches typed events (`pkg/sdk/events.go`) to every plugin that implements the matching handler
interface ([ADR-0003](../adr/0003-typed-event-bus.md)):

| Event | Handler |
|-------|---------|
| `ContextChangedEvent` | `ContextChangedHandler` |
| `PlanCompletedEvent` | `PlanCompletedHandler` |
| `PlanInvalidatedEvent` | `PlanInvalidatedHandler` |
| `LockDetectedEvent` | `LockDetectedHandler` |
| `LockClearedEvent` | `LockClearedHandler` |
| `StateRefreshedEvent` | `StateRefreshedHandler` |

`ContextChangedEvent` carries `Prev`, `Next` and a `Reason`; `OnlyPinsChanged()` lets plan tell a pin toggle (keep UI
state, mark stale) from a chdir or workspace switch (full reset).

## Plugin Loading

Plugins are compiled into the binary. `buildRegistry()` in `cmd/tfui/registry.go` is the one place every plugin is
registered and handed its `tfui.hcl` settings. It registers each factory with a `plugin.PluginMeta` that holds its
routing, so plugins stay invocation-agnostic ([ADR-0006](../adr/0006-orthogonal-income-outcome.md)):

| Field | Meaning |
|-------|---------|
| `Keybinding` | Key that opens it from home |
| `MenuVisible` | Listed on the home menu |
| `Nav` | `NavReplace` (lateral switch, history cleared) or `NavPush` (pushed onto the return stack) |

## Navigation (`internal/ui/app.go`)

The app keeps the active plugin and a `navStack` of plugins to return to
([ADR-0004](../adr/0004-navigation-model.md)):

- `navigateTo(p)` passes the busy-guard, cancels the active plugin if it is `Cancellable`, then pushes onto `navStack`
  for `NavPush` or clears it for `NavReplace`.
- `navigateBack()` pops `navStack`, or returns home when it is empty.
- `popIfPushed(cmd)` pops a `NavPush` plugin after an event it raised, such as a context switch.
- Workflow transitions push at runtime regardless of metadata: `ApplyRequestMsg` from plan pushes plan and activates
  apply.
- Plugins request navigation with `NavigateMsg{PluginID}` and leave with `DeactivateMsg`; they never hold references to
  each other.

**Busy-guard.** Every action that may start a terraform call or change its inputs goes through `requireIdle(reason)`:
while any plugin is `Busy()` (holding the directory lock, [ADR-0016](../adr/0016-non-reentrant-dir-lock.md)), the
action is refused with a message pointing to `:q!`. The chokepoints are `ContextSwitchRequestMsg`, `navigateTo` and
`:q`. UI-only actions such as scrolling, filtering and esc bypass it. `:q!` skips the guard and cancels every
cancellable plugin.

**View delegation.** The app calls `plugin.View(width, height)` directly. The frame stack routes input
(`Stack.Update()`) but not rendering, so a `Stackable` plugin's `View()` delegates to its top frame itself.

### Action Mode

A plugin is in action mode while it executes and reports: run, result, render, `ctrl+r` retry, success events, with a
cancellable context and an elapsed timer. The embeddable `sdk.ActionRunner`, configured per operation with an
`sdk.ActionSpec`, provides it ([`CONTEXT.md`](https://github.com/lmarqs/terraform-ui/blob/main/CONTEXT.md)). Taint,
untaint, import and forceunlock use it; they are `NavPush`, hidden from the menu, and reached through contextual keys
in state and plan (`TaintRequestMsg`, `UntaintRequestMsg`, `ImportRequestMsg`) or `:command`.

| Operation | Events on success |
|-----------|-------------------|
| Taint, Untaint | `PlanInvalidatedEvent` |
| Import | `StateRefreshedEvent`, `PlanInvalidatedEvent` |
| ForceUnlock | `LockClearedEvent`, `PlanInvalidatedEvent` |

### Overlays, Input and Frames

`sdk.Overlay` (`pkg/sdk/overlay.go`) is a modal with its own `Update`, `View` and `Hints`. A plugin asks for input with
`RequestInputMsg` wrapping an `InputRequest` (Text, Bool, Select or Filter mode) and receives `InputResponseMsg`; the
helpers are `InputConfirm`, `InputText` and `InputSelect` (`pkg/sdk/input.go`). `pkg/sdk/frames/` holds the reusable
frames: `FilterFrame`, `InspectFrame`, `ConfirmFrame`, `ActionFrame` (a running operation with cancel), `FormFrame`
(fields with validation) and `StreamFrame` (streaming output of plan, apply and init).

## Terraform Integration

Two sibling implementations of `sdk.Service` exist as a strategy pattern, selected once at startup; plugins never know
which one runs ([ADR-0002](../adr/0002-dual-runtime-strategy.md)).
A new `Service` method is implemented on `ExecService`, `MacroService` and `sdktest.MockService`.

### ExecService (live execution)

`internal/terraform/exec/service.go` shells out to the terraform (or tofu) binary via `hashicorp/terraform-exec`.
Every call is serialized per working directory by a non-reentrant `DirLock`
([ADR-0016](../adr/0016-non-reentrant-dir-lock.md)). `WithDir()` returns a service scoped to another directory with a
fresh cache. Reads go through the `ServiceCache`; when it is pre-seeded, reads are served without shelling out.

### MacroService (recording)

`internal/terraform/macro_service.go` records every mutation as an `sdk.Command` without executing anything and serves
reads from the same `ServiceCache`. After macro playback the recorded commands are printed to stdout. Macros are
deterministic and safe: the UI renders real data from the cache, and mutations never touch infrastructure.

### ServiceCache (shared state)

`internal/terraform/service_cache.go` is shared by both strategies:

- Pre-seeded at startup from `-plan`/`-state` ([ADR-0005](../adr/0005-cache-preseeding.md)): `PreseedSources.Seed`
  (`cmd/tfui/sources.go`) fills the cache before the service that wraps it is built
- Three source kinds: `file` (re-reads on invalidate), `stdin` (immutable), `exec` (cleared on invalidate)
- State-mutating operations (`StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`) invalidate the state cache
- `Refresh()`, `Init()` and `WorkspaceSelect()` invalidate everything
- Plugins bypass it with `StateList(ctx, sdk.SkipCache())` for an explicit refresh (ctrl+r); the variadic option keeps
  cache semantics out of the interface ([ADR-0007](../adr/0007-no-cache-semantics-in-sdk.md))

### Source layer (`internal/source/`)

Resolves a URI to raw bytes; `internal/terraform/loader.go` parses them with `LoadPlan` and `LoadState`.

| URI | Resolves to |
|-----|-------------|
| `-` | stdin |
| `/path`, `./path`, `../path` | local file |
| `file://...` | local file |
| `scheme://...` | the `Provider` registered for that scheme |
| anything else | an error with a suggestion |

### Binary resolution

OpenTofu is supported via explicit `terraform.bin = "tofu"` in HCL config, or by letting terraform-exec resolve the binary from PATH. There is no auto-detection logic.

## Entry Point (`cmd/tfui/`)

Most subcommands are `build<Plugin>Command(s *Session)` in their own file. Their `RunE` fills the plugin's Input from
flags and calls `Session.RunPlugin(ctx, "<id>", activate)`. `workspace` and `force-unlock` take the config instead and
run terraform directly through `imperativeService(cfg)`. The bare `tfui` command calls `Session.Run()`.

### Session pipeline (`session.go`)

`Session.Run()` executes `validate → LoadTape → buildService → buildRegistry → buildApp → present → emitRecorded`.
`RunPlugin` runs the same pipeline with the app in standalone mode for one plugin.

Two orthogonal axes are resolved from flags, and every combination is valid
([ADR-0006](../adr/0006-orthogonal-income-outcome.md)):

| Axis | Values | Determines |
|------|--------|-----------|
| **Presentation** | `Interactive` / `Headless` | A `tea.Program` on the alt screen, or the macro driver: for bare `tfui` under `--macro`; for a command when the [mode resolution](../reference/cli-io-contract.md#mode-resolution) picks headless |
| **Backend** | `Exec` / `Recording` | Terraform runs live (`ExecService`) or commands are recorded (`MacroService`, under `--macro`) |

### Version

`main.version` comes from the release build's `-ldflags -X main.version=…` (goreleaser), else from
`debug.ReadBuildInfo()` for `go install`, else `0.0.0-SNAPSHOT`.

## Config Resolution & Propagation

Config is resolved from `tfui.hcl` at three levels: root defaults → member (`config.LoadChild`) → workspace-specific overrides,
by `config.Resolve(root, child, workspace)`.

```
ContextSwitchRequestMsg{Chdir, Workspace}     (emitted by the chdir and workspace plugins)
  → requireIdle("context-switch")
  → config.LoadChild(member dir)              → childCfg
  → rebuildContext(chdir, absChdir, workspace)
      Service.WithDir(absChdir)
      config.Resolve(rootCfg, childCfg, workspace)
        → VarFiles, Vars, Parallelism, Lock, LockTimeout
      ExtraArgs from the CLI -- passthrough
      Pins empty: a switch ends the targeting selection
  → replaceContext(next, reason)              → ContextChangedEvent{Prev, Next, Reason}
```

`ExtraArgs` (from CLI `--` passthrough) is set once at boot and carried into every snapshot.

## Bubbletea Model/Update/View

```
┌────────────────────────────────────────┐
│  App (root model)                      │
│  - routes key presses to active plugin │
│  - handles global keys (q, Esc)        │
│  - manages window resize               │
├────────────────────────────────────────┤
│  Plugin (active)                       │
│  - receives Update() delegation        │
│  - returns View() for content area     │
│  - Init() for async data loading       │
├────────────────────────────────────────┤
│  Components (header, statusbar)        │
│  - shared UI chrome                    │
│  - rendered by App around plugin view  │
└────────────────────────────────────────┘
```

## Macro Engine (`internal/macro/`)

A synchronous driver for the app model plus the tape language ([Macro language](../reference/macro-language.md)),
used for headless runs, tests and demo recording.

| File | Role |
|------|------|
| `driver.go` | Drives the model without a terminal: `NewDriver(app, w, h)`, `SendKey`, `WaitUntil`, `View` |
| `runner.go` | Executes parsed tape commands against the driver |
| `tape.go` | Tape parser |
| `recorder.go` | Wraps a `tea.Model` to capture ANSI frames and generate a tape (`--record`); headless, the runner feeds it with `CaptureView()` |
| `key_string.go` | Maps `tea.KeyMsg` back to tape syntax |

## Testing

How each layer is tested, and the coverage gate, are in [Testing](testing.md).

## Command Type (`pkg/sdk/command.go`)

Every service operation maps to a `sdk.Command`:

```go
type Command struct {
    Binary string   // "terraform" or "tofu"
    Verb   string   // "plan", "apply", "state rm", etc.
    Args   []string // positional (addresses, IDs)
    Flags  []string // flags like "-target=X"
    Dir    string   // working directory
}
```

Every operation is one terraform command: **tfui is a command builder with a visual interface**. The TUI helps you construct the right terraform command. The CLI gives you a concise alternative. Both produce the same command, same outcome.

In macro mode, mutating operations are recorded as commands — the user sees what they'd need to run:
```
terraform state rm aws_instance.old
terraform plan -out=<planfile> -target=aws_instance.web
```

## Data Flow: Plan → Pin → Apply

Targets go on plan, never on apply: terraform ignores `-target` on a saved plan file and applies the whole plan
([ADR-0019](../adr/0019-unidirectional-data-flow.md)).

```
User presses 'p' → Plan plugin activates
  → Context().Service.Plan(ctx, Context().PlanOptions())
  → Shows changes with risk badges, plan saved to a plan file

User presses 'space' on resources → deps.Pin(...) → Context replaced with the new Pins
  → ContextChangedEvent (OnlyPinsChanged) → plan keeps its view and marks itself stale
  → the next plan runs with -target for each pin

User presses 'a' (or 'A' to auto-approve) → Plan emits ApplyRequestMsg{PlanFile, AutoApprove}
  → App pushes plan onto navStack and activates Apply with the plan file
  → Apply shows the confirmation (skipped on AutoApprove)

User confirms → terraform apply <planfile>
  → Shows elapsed time → success/error
  → PlanInvalidatedEvent on success
  → esc returns to plan through DeactivateMsg
```
