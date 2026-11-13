---
layout: default
title: Plugin Catalog — All Terraform UI Features
nav_order: 4
has_children: true
permalink: /plugins/
seo:
  description: Complete catalog of terraform-ui plugins — plan review, state browser, workspaces, apply, validate, console, and more.
description: "Lists every plugin with its key, category and what it does."
audience: "Users looking for the screen that does a task, or turning a plugin off"
---

# Plugins

Every tfui screen is a plugin. Most plugins open with one key from Home. The rest open from another plugin or from command mode (`:`).

## Available Plugins

| ID | Name | Key | Category | Description |
|----|------|-----|----------|-------------|
| [state](state.md) | State Browser | `s` | navigation | Browse and inspect terraform state resources |
| [plan](plan.md) | Plan Review | `p` | operations | Review terraform plan changes with expandable attribute diffs |
| [apply](apply.md) | Apply | `a` | operations | Apply terraform changes with confirmation and elapsed time tracking |
| [workspace](workspace.md) | Workspace | `w` | navigation | Manage terraform workspaces (list, switch, create, delete) |
| [output](output.md) | Outputs | `o` | navigation | View terraform output values |
| [validate](validate.md) | Validate | `v` | operations | Run terraform validate and show diagnostics |
| [taint](taint.md) | Taint | — (`t` in State/Plan) | action | Mark resources for recreation on next apply |
| [untaint](untaint.md) | Untaint | — (`T` in State/Plan) | action | Remove taint mark from resources |
| [import](import.md) | Import | — (`n` in State) | action | Import existing infrastructure into terraform state |
| [console](console.md) | Console | `~` | operations | Evaluate terraform console expressions |
| [risk](risk.md) | Risk Analysis | `R` | analysis | Analyze and group planned changes by risk level |
| [phantom](phantom.md) | Phantom Changes | `P` | analysis | Detect and explain phantom (no-op) changes in terraform plans |
| [blastradius](blastradius.md) | Blast Radius | `B` | analysis | Visualize module-grouped changes with impact scores |
| [context](context.md) | Context | `C` | navigation | Manage project, chdir, and workspace selection |
| [init](init.md) | Init | `i` | operations | Run terraform init with form-based options |
| [forceunlock](forceunlock.md) | Force Unlock | — | utility | Remove a stale state lock |
| [version](version.md) | Version | — | utility | Show tfui and terraform version information |
| [chdir](chdir.md) | Chdir Picker | — | navigation | Select chdir member (hidden, activated by context plugin) |

## Categories

- **Action**: open from State or Plan on the selected resources, ask for confirmation, run one terraform command; `Esc` then returns you to the screen you came from.
- **Operations**: run a terraform command against the current context.
- **Analysis**: read the last plan and change nothing.
- **Navigation**: browse state, outputs and workspaces, and switch the context.
- **Utility**: show version information or recover from a stale lock.

## Enabling/Disabling Plugins

All plugins are enabled by default. To disable one, add a `plugin` block under `defaults` in the project's root `tfui.hcl`:

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

Plan and Apply take one more setting, `targets`; their pages describe it.

## Adding a Plugin

To add a plugin, use the `add-plugin` skill; [architecture](../development/architecture.md) explains the SDK.
