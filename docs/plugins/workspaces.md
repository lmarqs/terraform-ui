---
layout: plugin
title: Workspaces
id: workspaces
key: w
description: Manage terraform workspaces (list, switch, create, delete)
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
  plugin "workspaces" {
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

## Related

- [State Browser](state.md) — browse state for the selected workspace
- [Context](context.md) — manage project/chdir/workspace selection
