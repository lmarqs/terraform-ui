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
  styles.go                — Style constants for consistent rendering
internal/
  config/config.go         — tfui.yaml loading, project discovery, OpenTofu detection
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

OpenTofu is supported via auto-detection (`tofu` preferred if on PATH) or explicit `terraform_binary: tofu` in config.

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
mise install              # Install Go, terraform, node
mise run build            # Build (runs fmt + lint first)
mise run test             # Unit tests
mise run coverage         # Coverage with 100% enforcement
mise run test:integration # Integration tests (need terraform)
mise run run              # Launch TUI in dev mode
```

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| `pkg/sdk` as public contract | Plugins never import internal/. Enables future extraction and gRPC plugins. |
| Plugin = feature | Every capability is a plugin. Core app is just routing. |
| Dependency injection | Service interface enables mocking. Plugins receive context, not concrete types. |
| hashicorp/go-plugin (future) | Third-party plugins as separate binaries over gRPC. Same interface, different transport. |
| semantic-release + goreleaser | Automatic versioning from commits + cross-platform binary distribution. |
| 100% coverage (excl cmd/) | Pipeline enforced. Untestable layers kept minimal via DI. |
| OpenTofu first-class | Auto-detect, configurable. Test matrix includes both. |
