---
layout: default
parent: Plugins
title: Workspace
id: workspace
key: w
description: Manage terraform workspace (list, switch, create, delete)
category: navigation
default_enabled: true
---

## Overview

The Workspaces plugin lists all terraform workspaces, highlights the current one, and lets you switch between them, create new workspaces, or delete unused ones. The current workspace is marked with an asterisk.

## Usage

Press `w` to open the Workspaces view. It loads the workspace list from the current terraform configuration.

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | Switch to selected workspace |
| `n` | Create a new workspace |
| `d` | Delete selected workspace |
| `r` | Refresh workspace list |
| `Esc` | Go back / cancel create |

When creating a new workspace, type the name and press `Enter` to confirm or `Esc` to cancel.

## Configuration

```hcl
# tfui.hcl
defaults {
  plugin "workspace" {
    enabled = true
  }
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |

## Screenshots/Output

```
Workspaces

* production
  staging
  development
  default (default)

3 workspace(s)  Current: production

Enter switch  n new  d delete  ^r refresh  Esc back
```

Creating a new workspace:

```
Workspaces

New workspace: feature-branch_

Enter confirm  Esc cancel
```

## CLI

All workspace operations are also available as non-interactive CLI subcommands:

```bash
tfui workspace show                    # print current workspace name
tfui workspace list                    # list all workspaces (current marked with *)
tfui workspace select <name>           # switch to workspace
tfui workspace new <name>              # create and switch to workspace
tfui workspace delete <name>           # delete workspace
```

### Flags

| Flag | Applies to | Description |
|------|-----------|-------------|
| `-lock` | `new`, `delete` | Lock state during operation (default: true) |
| `-lock-timeout` | `new`, `delete` | Duration to wait for a state lock |
| `-force` | `delete` | Force deletion of a non-empty workspace |

### Examples

```bash
tfui workspace new feature-branch -lock=false
tfui workspace delete old-branch -force
tfui workspace list | grep staging
```

## Related

- [State Browser](state.md) — browse state for the selected workspace
- [Context](context.md) — manage project/chdir/workspace selection
