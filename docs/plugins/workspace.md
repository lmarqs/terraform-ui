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

## Interactive (TUI)

Press `w` to open the Workspaces view. It loads the workspace list from the current terraform configuration.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `j` / `k` | Navigate up/down | List |
| `Enter` | Switch to selected workspace | List |
| `n` | Create a new workspace | List |
| `d` | Delete selected workspace | List |
| `r` | Refresh workspace list | List |
| `Esc` | Go back / cancel create | Always |

### Flow

```
Home ──w──→ Workspaces (list)
               │
               ├── Enter → WorkspaceChangedEvent → pop back to origin
               ├── n → Input name → Enter → WorkspaceCreatedEvent
               ├── d → Confirm delete → deleted, refresh
               └── Esc → Cancel → pop back to origin
```

When creating a new workspace, type the name and press `Enter` to confirm or `Esc` to cancel.

## Command Line (CLI)

```bash
tfui workspace show                    # Print current workspace name
tfui workspace list                    # List all workspaces (current marked with *)
tfui workspace select <name>           # Switch to workspace
tfui workspace new <name>              # Create and switch to workspace
tfui workspace delete <name>           # Delete workspace
```

| Flag | Applies to | Description |
|------|-----------|-------------|
| `-lock` | `new`, `delete` | Lock state during operation (default: true) |
| `-lock-timeout` | `new`, `delete` | Duration to wait for a state lock |
| `-force` | `delete` | Force deletion of a non-empty workspace |

| Code | Meaning |
|------|---------|
| 0 | Operation succeeded |
| 1 | Operation failed |

## Configuration

```hcl
# tfui.hcl
plugin "workspace" {
  enabled = true
}
```

## Screenshots

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

- [State Browser](state.md) -- browse state for the selected workspace
- [Context](context.md) -- manage project/chdir/workspace selection
