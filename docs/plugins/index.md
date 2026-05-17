---
layout: default
title: Plugins
nav_order: 4
has_children: true
permalink: /plugins/
description: Modular plugin catalog for terraform-ui with keybindings and categories
---

# Plugins

tfui is built around a modular plugin system. Each plugin provides a focused view accessible via a single key press.

## Available Plugins

| ID | Name | Key | Category | Description |
|----|------|-----|----------|-------------|
| [state](state.md) | State Browser | `s` | navigation | Browse and inspect terraform state resources |
| [plan](plan.md) | Plan Review | `p` | operations | Review terraform plan changes with expandable attribute diffs |
| [apply](apply.md) | Apply | `a` | operations | Apply terraform changes with confirmation and elapsed time tracking |
| [workspace](workspace.md) | Workspace | `w` | navigation | Manage terraform workspaces (list, switch, create, delete) |
| [output](output.md) | Outputs | `o` | navigation | View terraform output values |
| [validate](validate.md) | Validate | `v` | operations | Run terraform validate and show diagnostics |
| [taint](taint.md) | Taint | `t` | action | Mark resources for recreation on next apply |
| [untaint](untaint.md) | Untaint | `T` | action | Remove taint mark from resources |
| [import](import.md) | Import | `n` | action | Import existing infrastructure into terraform state |
| [console](console.md) | Console | `~` | operations | Interactive terraform console (REPL) |
| [risk](risk.md) | Risk Analysis | `R` | analysis | Analyze and group planned changes by risk level |
| [phantom](phantom.md) | Phantom Changes | `P` | analysis | Detect and explain phantom (no-op) changes in terraform plans |
| [blastradius](blastradius.md) | Blast Radius | `B` | analysis | Visualize module-grouped changes with impact scores |
| [context](context.md) | Context | `C` | navigation | Manage project, chdir, and workspace selection |
| [init](init.md) | Init | `i` | operations | Run terraform init with form-based options |
| [forceunlock](forceunlock.md) | Force Unlock | — | utility | Remove a stale state lock |
| [version](version.md) | Version | — | utility | Show tfui and terraform version information |
| [chdir](chdir.md) | Chdir Picker | — | internal | Select chdir member (hidden, activated by context plugin) |

## Categories

### Action (Verb Plugins)

Transient plugins for terraform top-level verbs. Arrive with context, confirm, execute, return.

- **[Taint](taint.md)** — mark resources for recreation (`terraform taint`)
- **[Untaint](untaint.md)** — remove taint mark (`terraform untaint`)
- **[Import](import.md)** — import existing infrastructure (`terraform import`)

### Operations

Plugins that execute terraform commands or modify infrastructure.

- **[Plan](plan.md)** — run and review `terraform plan`
- **[Apply](apply.md)** — execute `terraform apply` with replan + confirmation
- **[Init](init.md)** — run `terraform init` with form-based options
- **[Validate](validate.md)** — run `terraform validate`
- **[Console](console.md)** — interactive `terraform console`

### Analysis

Plugins that analyze plan output without modifying infrastructure.

- **[Risk Analysis](risk.md)** — risk-level grouping and assessment
- **[Phantom Changes](phantom.md)** — detect cosmetic-only changes
- **[Blast Radius](blastradius.md)** — module-level impact scoring

### Navigation

Plugins for navigating terraform state, workspaces, and project context.

- **[State Browser](state.md)** — inspect managed resources
- **[Workspace](workspace.md)** — switch and manage workspace
- **[Outputs](output.md)** — view output values
- **[Context](context.md)** — project/chdir/workspace selector

### Utility

Informational and recovery plugins.

- **[Force Unlock](forceunlock.md)** — remove stale state locks
- **[Version](version.md)** — show tfui and terraform version info

### CLI Commands (not plugins)

- **[Scaffold](scaffold.md)** — configuration file generator (`tfui scaffold`, CLI-only — no plugin directory)

## Enabling/Disabling Plugins

All plugins are enabled by default. To disable one, configure it in `tfui.hcl`:

```hcl
defaults {
  plugin "phantom" {
    enabled = false
  }
  plugin "blastradius" {
    enabled = false
  }
}
```

## Creating a Custom Plugin

To create a new plugin:

1. Create a new directory `plugins/<name>/` with a Go file implementing the `Plugin` interface from `pkg/sdk/plugin.go`
2. Implement the required methods: `ID()`, `Name()`, `Description()`, `Init()`, `Update()`, `View()`, `Configure()`, `Ready()`
3. Register the plugin factory in `cmd/tfui/main.go` with `PluginMeta` (keybinding, menu visibility)
4. Add documentation at `docs/plugins/<name>.md`

See the `/add-plugin` slash command for a full step-by-step guide and reference implementation.
