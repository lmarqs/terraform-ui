---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Switch Terraform Project, Directory and Workspace"
id: context
key: C
seo:
  description: "See and change the working context in tfui: the Terraform project directory, the member directory and the workspace. Every screen reloads with fresh data."
description: "Describes how the Context plugin shows and switches project, chdir and workspace."
audience: "Users who want to see or switch the project member and workspace tfui runs against"
category: navigation
default_enabled: true
---

# Context

## Overview

The Context plugin displays the current working context as a form dashboard -- Project directory, Chdir member, and Workspace. Selectable fields navigate to their respective picker plugins (chdir, workspace) for selection.

Changing context invalidates all plugin state. After a switch, every screen reloads for the new context.

## Screenshot

![Context]({{ site.baseurl }}/assets/demo/context.gif)

## Interactive (TUI)

Press `C` from any screen or type `:context` to open the Context dashboard.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` / `↑` / `↓` | Navigate between fields | Dashboard |
| `Enter` | Open picker for selected field | Dashboard |
| `Esc` | Go back | Always |
| `q` | Go home | Always |

### Fields

| Field | Selectable | Action |
|-------|-----------|--------|
| Project | No | Displays the project root directory (read-only) |
| Chdir | Only when the project has a `tfui.hcl` | Opens the chdir picker |
| Workspace | Yes | Opens the workspace picker |

### Flow

```
C (from any screen) → Context
  ├── Enter on Chdir → Chdir picker → select → chdir switches (workspace resets to `default`) → back here
  ├── Enter on Workspace → Workspace picker → select → workspace switches → back here
  └── Esc / q → Home
```

## Command Line (CLI)

Has no command of its own. Context is set via flags:

```bash
tfui plan -project ./infra -chdir modules/networking
```

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Select chdir | `-chdir <member>` flag | `C` → Chdir → select |
| Set workspace | `tfui workspace select <name>` | `C` → Workspace → select |
| View current context | Implicit from flags | `C` (dashboard) |

## Members

Members are declared as top-level `member "path" {}` blocks in the project's root `tfui.hcl`, one block per path; each label is used as written. A `tfui.hcl` without `member` blocks has one member, the project root (`.`), and selecting Chdir switches to it without showing a list. Without a `tfui.hcl`, the Chdir field is not selectable.

```hcl
# tfui.hcl
member "modules/vpc" {}
member "envs/prod" {}
member "stacks/networking" {}
```

The [configuration guide](../guides/configuration.md) covers the rest of `tfui.hcl`.

## Related

- [Workspace](workspace.md) -- manage workspace within the active chdir
- [Chdir Picker](chdir.md) -- internal picker for member selection
- [State Browser](state.md) -- browse state for the active chdir
- [Plan](plan.md) -- plan runs against the active chdir directory
