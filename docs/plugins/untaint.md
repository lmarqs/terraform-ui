---
layout: default
parent: Plugins
title: Untaint
id: untaint
key: T
category: action
default_enabled: true
description: Remove taint mark from terraform resources to prevent forced recreation
---

## Overview

Remove taint mark from resources to prevent forced recreation. Standalone verb plugin -- mirrors `terraform untaint` as a top-level command. NavPush behavior: returns to the origin plugin on completion or cancel.

## Interactive (TUI)

### Entry Points

| Context | Key | Behavior |
|---------|-----|----------|
| State list/detail | `T` | Navigate to untaint with cursor address |
| Plan list | `T` | Navigate to untaint with cursor address |
| Batch palette (`!`) | `T` | Navigate to untaint with all pinned addresses |
| Command mode | `:untaint` | Navigate to untaint |

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `y` | Confirm untaint | Confirming |
| `n` / `Esc` | Cancel and return | Confirming |
| `p` | Navigate to plan | Done |
| `Esc` | Back to previous plugin | Any |
| `ctrl+r` | Retry | Error |

### Flow

```
Idle → Confirming → Loading → Done/Error
```

1. Plugin receives target address(es) via `SetTargets()`
2. Shows confirmation: "Untaint \<address\>?"
3. For batch: shows count and lists all addresses
4. On confirmation, executes `terraform untaint` for each address sequentially
5. On success, emits `PlanInvalidatedEvent` (plan auto-replans)
6. NavPush returns user to origin plugin

## Command Line (CLI)

```bash
tfui untaint <address>           # Untaint single resource
tfui untaint <addr1> <addr2>     # Batch untaint
```

| Code | Meaning |
|------|---------|
| 0 | Untaint succeeded |
| 1 | Untaint failed |

## Configuration

```hcl
# tfui.hcl
plugin "untaint" {
  enabled = true
}
```

## Screenshots

```
Untaint

Untaint aws_instance.web?
This will remove the taint mark (no forced recreation).

[y]es / [n]o
```

## Related

- [Taint](taint.md) -- mark resources for recreation
- [State Browser](state.md) -- browse resources and untaint from context
- [Plan](plan.md) -- see the effect of untainting
