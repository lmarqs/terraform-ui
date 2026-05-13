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
    service.go             — TerraformService (implements sdk.Service via terraform-exec)
    risk.go                — Risk classification (shared logic)
    phantom.go             — Phantom change detection (shared logic)
    grouping.go            — Module-level change grouping (shared logic)
  ui/
    app.go                 — Root Bubbletea model, plugin routing
    styles/theme.go        — Lipgloss style definitions
    views/home.go          — Home screen (auto-generated from registry)
    components/            — Header, statusbar
plugins/
  plan/                    — Plan review plugin
  risk/                    — Risk analysis plugin
  phantom/                 — Phantom change detection plugin
  blastradius/             — Blast radius visualization plugin
  state/                   — State browser plugin
  apply/                   — Apply with confirmation plugin
  workspaces/              — Workspace management plugin
  context/                 — Context manager (scope picker, monorepo support)
tests/
  integration/             — CLI integration tests (require terraform)
  fixtures/                — Real terraform projects for testing
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
- **Configurable** — per-plugin options in `tfui.yaml`
- **Independent** — import only `pkg/sdk`, never `internal/`
- **Testable** — mock the `sdk.Service` interface

## Plugin Loading

Currently: built-in (compiled into the binary). Factories are registered in `cmd/tfui/main.go` and built via the registry with config.

Future (v1.1+): hashicorp/go-plugin for external plugins over gRPC. Third-party plugins as separate binaries communicating via the same `sdk.Plugin` contract serialized over proto.

## Terraform Integration

`internal/terraform/service.go` implements `sdk.Service` using `hashicorp/terraform-exec`:

- `Plan()` — terraform plan → parse JSON → classify risk → detect phantoms
- `Apply()` — terraform apply on saved plan file
- `StateList()` — parse state JSON for resource list
- `Show()` — terraform state show for resource detail
- `Workspace()` / `WorkspaceList()` — workspace management

OpenTofu is supported via explicit `terraform.bin = "tofu"` in HCL config, or by letting terraform-exec resolve the binary from PATH. There is no auto-detection logic.

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

Coverage enforcement: 100% on all packages except `cmd/` (glue layer) and terraform-exec I/O calls. Pipeline fails if coverage drops.

## Development

```bash
mise run dev              # Launch TUI in dev mode
mise run build            # Cross-platform binaries
mise run fmt              # Format source files
mise run check:lint       # Lint (golangci-lint v2)
mise run test:unit        # Unit tests
mise run test:coverage    # Coverage enforcement (90%)
mise run test:integration # Integration tests
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

### UX

| Decision | Rationale |
|----------|-----------|
| Plan and Apply are separate screens | Different cognitive purposes (review vs execute). Apply can run standalone. Can take 10+ minutes. |
| Pins = targets | Visual selection (space) is the TUI equivalent of `--target`. Pin then apply = apply only pinned. |
| `a` from Plan = apply with pins | Seamless flow: review → select → execute. No address typing. |
| StaticService returns commands, not errors | Read-only mode becomes a command builder. User sees the terraform command to run manually. |
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
