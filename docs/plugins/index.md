---
layout: page
title: Plugins
permalink: /plugins/
---

# Plugins

tfui is built around a modular plugin system. Each plugin provides a focused view accessible via a single key press.

## Available Plugins

| ID | Name | Key | Category | Description |
|----|------|-----|----------|-------------|
| [plan](plan.md) | Plan Review | `p` | operations | Review terraform plan changes with expandable attribute diffs |
| [apply](apply.md) | Apply | `a` | operations | Apply terraform changes with confirmation and elapsed time tracking |
| [risk](risk.md) | Risk Analysis | `R` | analysis | Analyze and group planned changes by risk level |
| [phantom](phantom.md) | Phantom Changes | `P` | analysis | Detect and explain phantom (no-op) changes in terraform plans |
| [blastradius](blastradius.md) | Blast Radius | `b` | analysis | Visualize module-grouped changes with impact scores |
| [state](state.md) | State Browser | `s` | navigation | Browse and inspect terraform state resources |
| [workspaces](workspaces.md) | Workspaces | `w` | navigation | Manage terraform workspaces (list, switch, create, delete) |
| [context](context.md) | Context | `c` | navigation | Select terraform project scope |

## Categories

### Operations

Plugins that execute terraform commands and modify infrastructure.

- **[Plan](plan.md)** -- run and review `terraform plan`
- **[Apply](apply.md)** -- execute `terraform apply` with confirmation

### Analysis

Plugins that analyze plan output without modifying infrastructure.

- **[Risk Analysis](risk.md)** -- risk-level grouping and assessment
- **[Phantom Changes](phantom.md)** -- detect cosmetic-only changes
- **[Blast Radius](blastradius.md)** -- module-level impact scoring

### Navigation

Plugins for navigating terraform state, workspaces, and projects.

- **[State Browser](state.md)** -- inspect managed resources
- **[Workspaces](workspaces.md)** -- switch and manage workspaces
- **[Context](context.md)** -- terraform project scope selector

## Enabling/Disabling Plugins

All plugins are enabled by default. To disable one, set `enabled: false` in `tfui.yaml`:

```yaml
plugins:
  phantom:
    enabled: false
  blastradius:
    enabled: false
```

## Creating a Custom Plugin

To create a new plugin:

1. Create a new directory `plugins/<name>/` with a Go file implementing the `Plugin` interface from `internal/plugin/plugin.go`
2. Implement the required methods: `ID()`, `Name()`, `Description()`, `KeyBinding()`, `Init()`, `Update()`, `View()`, `Configure()`, `Ready()`
3. Register the plugin factory in `cmd/tfui/main.go`
4. Add documentation at `docs/plugins/<name>.md`

See the `/add-plugin` slash command for a full step-by-step guide and reference implementation.
