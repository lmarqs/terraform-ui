---
layout: default
parent: Plugins
title: Context
id: context
key: C
description: View and manage working context (project, chdir, workspace)
category: navigation
default_enabled: true
---

## Overview

The Context plugin displays the current working context as a form dashboard — Project directory, Chdir member, and Workspace. Selectable fields navigate to their respective picker plugins (chdir, workspace) for selection.

Changing context invalidates all plugin state. This is a full view switch (`NavReplace`), not a push — after a context change, plugins reload with fresh data.

## Usage

Press `C` from any screen or type `:context` to open the Context dashboard.

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate between fields |
| `Enter` | Open picker for selected field |
| `Esc` | Go back |
| `q` | Go home |

### Fields

| Field | Selectable | Action |
|-------|-----------|--------|
| Project | No | Displays the project root directory (read-only) |
| Chdir | Only if members configured | Opens the chdir picker |
| Workspace | Yes | Opens the workspace picker |

## Configuration

Members are declared as top-level `member "path" {}` blocks in `tfui.hcl`. When no members are configured, the Chdir field is non-selectable.

```hcl
# tfui.hcl
member "modules/*" {}
member "envs/**" {}
member "stacks/networking" {}
```

## Screenshots/Output

With members configured (cursor on Chdir):

```
▸ Chdir        modules/vpc  ▸
  Workspace    default      ▸
  Project      /my/project

↑↓ navigate  Enter select  Esc cancel
```

Without members (cursor on Workspace):

```
  Project      .
  Chdir        -
▸ Workspace    default  ▸

↑↓ navigate  Enter select  Esc cancel
```

## Navigation Flow

```
C (from any screen) → Context (NavReplace)
  ├── Enter on Chdir → Chdir picker (NavPush) → select → ChdirChangedEvent → return to Context
  ├── Enter on Workspace → Workspace picker (NavPush) → select → WorkspaceChangedEvent → return to Context
  └── Esc / q → home
```

## Related

- [Workspace](workspace.md) — manage workspace within the active chdir
- [State Browser](state.md) — browse state for the active chdir
- [Plan](plan.md) — plan runs against the active chdir directory
