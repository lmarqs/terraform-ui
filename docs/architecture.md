---
layout: default
title: Architecture
description: Internal architecture of terraform-ui
---

# Architecture

terraform-ui is a Go application built with [Bubbletea](https://github.com/charmbracelet/bubbletea) (TUI framework) and [terraform-exec](https://github.com/hashicorp/terraform-exec) (terraform SDK). Features are modular plugins that depend only on a public SDK package.

## Dependency Direction

```
plugins/*  ───→  pkg/sdk  ←───  internal/*
   │                                 │
   │  (public contract only)         │  (implements sdk interfaces)
   │                                 │
   └── never imports internal/ ──────┘
```

Plugins depend exclusively on `pkg/sdk/`. Internal packages implement the interfaces defined there. This enables future extraction of plugins to separate repositories or gRPC-based external processes.

## Project Structure

```
cmd/tfui/main.go           — CLI entry point, plugin registration (thin glue)
pkg/sdk/                   — Public SDK (the only dependency for plugins)
  plugin.go                — Plugin interface
  context.go               — Shared context (service, dir, workspace)
  types.go                 — Domain types (Action, RiskLevel, Resource, PlanChange)
  service.go               — Service interface (terraform operations)
  bus.go                   — EventBus typed dispatch
  events.go                — Event types and handler interfaces
  options.go               — ResolvedOptions + BuildPlanOptions/BuildApplyOptions
  styles.go                — Style constants for consistent rendering
internal/
  config/config.go         — HCL config loading, project discovery
  plugin/registry.go       — Plugin registry (host-side only)
  terraform/
    service.go             — ExecService (implements sdk.Service via terraform-exec)
    macro_service.go       — MacroService (records commands, reads from cache)
    service_cache.go       — ServiceCache (typed, source-aware data cache)
    state_ops.go           — State operations (rm, mv, import, taint, untaint)
    workspace_ops.go       — Workspace operations
    plan_parser.go         — Plan JSON parsing
  source/                  — Universal source abstraction (URI resolution, providers)
  macro/                   — Macro engine (Driver, tape DSL parser)
  ui/
    app.go                 — Root Bubbletea model, plugin routing
    components/            — Header, statusbar
  editor/                  — Editor integration ($EDITOR at file:line)
  ai/                      — AI provider (Claude via Bedrock, auto-detection)
  logging/                 — Structured logger setup
plugins/
  context/                 — Context dashboard: Project + Chdir + Workspace
  chdir/                   — Chdir picker: select member from explicit list
  state/                   — State browser (list, inspect, pin, delete, move, edit)
  plan/                    — Plan review (diff view, expand attributes, risk)
  apply/                   — Apply executor with confirmation
  workspace/               — Workspace management
  repl/                    — Terraform console (REPL)
  output/                  — Terraform outputs viewer
  validate/                — Terraform validate with diagnostics
  risk/                    — Risk classification (decorates plan)
  phantom/                 — Phantom change detection (decorates plan)
  blastradius/             — Blast radius visualization
  forceunlock/             — Force unlock stale state locks
  version/                 — Version information display
tests/
  integration/             — CLI integration tests (require terraform/tofu/terragrunt)
  fixtures/                — Real terraform projects and config fixtures
```

## Plugin System

Every feature is a plugin implementing `pkg/sdk.Plugin`:

```go
type Plugin interface {
    ID() string
    Name() string
    Description() string
    Init(ctx *Context) tea.Cmd
    Update(msg tea.Msg) (Plugin, tea.Cmd)
    View(width, height int) string
    Configure(cfg map[string]interface{}) error
    Ready() bool
}
```

Plugins are registered with external metadata (`PluginMeta`) that controls keybinding and menu visibility — plugins themselves are invocation-agnostic.

Plugins can also implement handler interfaces (`ChdirHandler`, `WorkspaceHandler`, etc.) to receive typed events from other plugins via the EventBus. This enables reactive data loading without polling or stringly-typed session keys.

Plugins are:
- **Self-contained** — own view logic, types, messages
- **Configurable** — per-plugin options in `tfui.hcl`
- **Independent** — import only `pkg/sdk`, never `internal/`
- **Testable** — mock the `sdk.Service` interface

## Plugin Loading

Currently: built-in (compiled into the binary). Factories are registered in `cmd/tfui/main.go` and built via the registry with config.

Future (v1.1+): hashicorp/go-plugin for external plugins over gRPC. Third-party plugins as separate binaries communicating via the same `sdk.Plugin` contract serialized over proto.

## Terraform Integration

Two sibling implementations of `sdk.Service` exist as a strategy pattern — selected at startup based on execution mode:

### ExecService (live execution)

`internal/terraform/service.go` — shells out to the terraform (or tofu) binary via `hashicorp/terraform-exec`:

- `Plan()` — terraform plan → parse JSON → classify risk → detect phantoms
- `Apply()` — terraform apply on saved plan file
- `StateList(opts ...StateListOption)` — parse state JSON for resource list (supports `SkipCache()` to force re-read)
- `Show()` — terraform state show for resource detail
- `Workspace*()` — workspace list, select, create, delete
- `WithDir()` — returns a new ExecService scoped to a different directory (fresh cache)

Reads go through a `ServiceCache` (typed, source-aware). If the cache is pre-seeded (via `--plan`/`--state` flags), reads are served from cache without shelling out — this enables read-only mode.

### MacroService (recording)

`internal/terraform/macro_service.go` — records every operation as an `sdk.Command` without executing anything:

- Mutating calls (`Plan`, `Apply`, `StateRm`, etc.) → record the command, return empty/cached data
- Read calls (`StateList`, `Show`, `Workspace`) → serve from the same `ServiceCache`
- After macro playback, all recorded commands are printed to stdout

This makes macros deterministic and safe: the UI renders real data (from cache) but mutations never touch infrastructure.

### ServiceCache (shared state)

`internal/terraform/service_cache.go` — both strategies share this typed cache:

- Pre-seeded at startup from `--plan`/`--state` file/stdin
- Three source kinds: `file` (re-reads on invalidate), `stdin` (immutable), `exec` (cleared on invalidate)
- ExecService populates it after live calls; MacroService only reads from it
- State-mutating operations (`StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`) auto-invalidate state cache
- Plugins can bypass cache via `StateList(ctx, sdk.SkipCache())` for explicit refresh (ctrl+r)

### Binary resolution

OpenTofu is supported via explicit `terraform.bin = "tofu"` in HCL config, or by letting terraform-exec resolve the binary from PATH. There is no auto-detection logic.

## Entry Point (`cmd/tfui/main.go`)

main.go is the composition root — it wires everything together but contains no domain logic:

```
main.go
├── rootCmd (cobra)           → runTUI() or runMacro()
├── planCmd, applyCmd         → runPlan(), runApply()
├── scaffoldCmd               → runScaffold()
├── versionCmd
└── plugin CLI commands       → buildPluginCommands()
```

### Architectural decisions in main.go

| Decision | Rationale |
|----------|-----------|
| Two orthogonal axes: income × outcome | Income = how the user drives (TUI or CLI). Outcome = what happens (ExecService executes live, MacroService records). These are independent — macro is not bound to TUI or CLI; any income can pair with any outcome. |
| `buildRegistry()` is axis-agnostic | Plugins don't know their income or outcome. Same registry regardless of which axis combination is active |
| ServiceCache pre-seeded before service creation | `seedCache()` runs first, then the service wraps it. Enables read-only mode transparently |
| `buildRegistry()` as single composition point | All 12 plugins registered with metadata in one place. Config injection happens here, not scattered |
| TTY detection gates interactive mode | No TTY + `--plan`/`--state` → auto-renders non-interactively. No TTY without data → actionable error |
| `splitPassthrough()` + `normalizeArgs()` | Terraform flag compatibility: `--` separates tfui flags from terraform extras; short flags normalized |
| Version resolution chain | `ldflags` (CI) → `ReadBuildInfo` (go install) → `"0.0.0-SNAPSHOT"` (dev) |
| Spinner on stderr, data on stdout | CLI commands respect Unix conventions: pipe-safe stdout, human feedback on stderr (suppressed with `--ci`) |

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

## Testing Strategy

| Layer | Approach | Coverage Target |
|-------|----------|-----------------|
| `pkg/sdk` | Type definitions, no logic | N/A |
| `internal/terraform` (parsers) | Unit tests, mock data | 100% |
| `internal/terraform` (service calls) | Integration tests (need terraform binary) | Excluded from unit coverage |
| `internal/plugin` | Unit tests, mock plugins | 100% |
| `internal/ui` | Unit tests, mock registry | 100% |
| `plugins/*` | Unit tests, mock service | 100% |
| `cmd/tfui` | Excluded (thin glue) | N/A |
| `tests/integration` | Integration (build tag, need terraform) | Behavioral coverage |

Coverage enforcement: 100% gate on all packages except `cmd/` (glue layer) and terraform-exec I/O calls. Pipeline fails if coverage drops.

## Development

```bash
mise run dev              # Launch TUI in dev mode
mise run build            # Cross-platform binaries
mise run fmt              # Format source files
mise run check:lint       # Lint (golangci-lint v2)
mise run test:unit        # Unit tests
mise run test:coverage    # Coverage enforcement (100%)
mise run 'test:integration:*'  # Integration tests (terraform/tofu/terragrunt)
mise run test:macro       # Macro tapes
```

## Key Design Decisions

### Architecture

| Decision | Rationale |
|----------|-----------|
| `pkg/sdk` as public contract | Plugins never import internal/. Enables future extraction and gRPC plugins. |
| Plugin = feature | Every capability is a plugin. Core app is just routing. |
| Dependency injection | Service interface enables mocking. Plugins receive context, not concrete types. |
| Plugins invocation-agnostic | Same plugin works via keybinding, command bar, CLI, macro. No coupling to navigation. |
| hashicorp/go-plugin (future) | Third-party plugins as separate binaries over gRPC. Same interface, different transport. |
| 100% coverage (excl cmd/) | Pipeline enforced. Untestable layers kept minimal via DI. |
| OpenTofu first-class | Configurable via `terraform.bin` in HCL config, or let terraform-exec resolve. No auto-detection. Test matrix includes both. |
| `EventBus` typed pub/sub | Plugins react to state changes (chdir, workspace, plan) via handler interfaces. No polling, no stringly-typed keys. Compile-time safe. |
| ExecService + MacroService as sibling strategies | Same interface, different behavior. ExecService shells out for live ops; MacroService records commands without executing. Selected at startup, plugins are unaware. |
| ServiceCache as shared read layer | Both strategies read from the same cache. Pre-seeded from `--plan`/`--state` at startup, populated by ExecService after live calls. Enables read-only mode transparently. |

### UX

| Decision | Rationale |
|----------|-----------|
| Plan and Apply are separate screens | Different cognitive purposes (review vs execute). Apply can run standalone. Can take 10+ minutes. |
| Pins = targets | Visual selection (space) is the TUI equivalent of `--target`. Pin then apply = apply only pinned. |
| `a` from Plan = apply with pins | Seamless flow: review → select → execute. No address typing. |
| ExecService executes live, user waits for real results | Spinners, elapsed time, real errors — the UX communicates "this is happening now" with progress feedback and retry on failure. |
| MacroService returns commands, not errors | Macro playback becomes a command builder. User sees the terraform commands that would run. |
| CLI and TUI produce identical state | Testable invariant. The equivalence guarantee. |
| Macros test UI, integration tests outcomes | Different concerns, different tools. Macros are fast/deterministic. Integration tests are authoritative. |

### Command Type (`pkg/sdk/command.go`)

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

This makes the tool's relationship to terraform explicit: **tfui is a command builder with a visual interface**. The TUI helps you construct the right terraform command. The CLI gives you a concise alternative. Both produce the same command, same outcome.

In read-only mode, mutating operations return `CommandErr` — the user sees what they'd need to run:
```
terraform state rm aws_instance.old
terraform apply -target=aws_instance.web
```

### Data Flow: Plan → Pin → Apply

```
User presses 'p' → Plan plugin activates
  → svc.Plan(ctx, nil) → terraform plan
  → Shows changes with risk badges

User presses 'space' on resources → pins via shared PinService

User presses 'a' → Plan emits ApplyRequestMsg
  → plan emits PlanCompletedEvent
  → App reads pins from PinService
  → Activates Apply plugin with pins as targets
  → Re-plans with --target (only pinned)
  → Shows confirmation: "Apply 2 of 12 changes?"

User presses 'y' → svc.Apply(ctx, targets)
  → terraform apply (targeted plan)
  → Shows elapsed time → success/error
```
