---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Workspace Manager in the Terminal"
id: workspace
key: w
seo:
  description: "List, switch, create and delete Terraform workspaces from a terminal UI, with the current workspace marked by an asterisk."
description: "Describes how the Workspace plugin lists, switches, creates and deletes workspaces."
audience: "Users switching, creating or deleting terraform workspaces"
category: navigation
default_enabled: true
---

# Workspace

## Overview

The Workspace plugin lists all terraform workspaces, highlights the current one, and lets you switch between them, create new workspaces, or delete unused ones. The current workspace is marked with an asterisk.

## Screenshot

![Workspace]({{ site.baseurl }}/assets/demo/workspace.gif)

## Interactive (TUI)

Press `w`, or pick Workspace in [Context](context.md), to open the Workspace view. It loads the workspace list from the current terraform configuration.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` | Navigate up/down | List |
| `Enter` | Switch to selected workspace and go back | List |
| `s` | Switch to selected workspace and stay in the list | List |
| `n` | Create a new workspace | List |
| `d` | Delete selected workspace (not the current or `default` workspace) | List |
| `ctrl+r` | Refresh workspace list | List |
| `Esc` | Go back / cancel create | Always |

### Flow

```
Home ──w──→ Workspace (list)
               │
               ├── Enter → workspace switches → back to where you came from
               ├── s → workspace switches → stay in the list
               ├── n → Input name → Enter → workspace created
               ├── d → Confirm delete → deleted, refresh
               └── Esc → Cancel → back to where you came from
```

## Command Line (CLI)

```bash
tfui workspace show                    # Print current workspace name
tfui workspace list                    # List all workspaces (current marked with *)
tfui workspace select <name>           # Switch to workspace
tfui workspace new <name>              # Create and switch to workspace
tfui workspace delete <name>           # Delete workspace
```

Flags (`-lock`, `-lock-timeout`, `-force`): see [`tfui workspace`](../reference/cli-reference.md#tfui-workspace).
Exit codes: see [CLI reference](../reference/cli-reference.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| List workspaces | `tfui workspace list` | Press `w` |
| Switch workspace | `tfui workspace select <name>` | `w` → navigate → `Enter` |
| Create workspace | `tfui workspace new <name>` | `w` → `n` → type name → `Enter` |
| Delete workspace | `tfui workspace delete <name>` | `w` → navigate → `d` → confirm |

## Related

- [State Browser](state.md) -- browse state for the selected workspace
- [Context](context.md) -- manage project/chdir/workspace selection
