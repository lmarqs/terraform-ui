---
layout: default
parent: Plugins
title: Untaint
id: untaint
key: T
category: action
---

# Untaint

Remove taint mark from resources. Standalone verb plugin — mirrors `terraform untaint` as a top-level command.

## Interactive (TUI)

### Entry Points

| Context | Key | Behavior |
|---------|-----|----------|
| State list/detail | `T` | Navigate to untaint with cursor address |
| Plan list | `T` | Navigate to untaint with cursor address |
| Batch palette (`!`) | `T` | Navigate to untaint with all pinned addresses |
| Command mode | `:untaint` | Navigate to untaint |

### Flow

1. Plugin receives target address(es) via `SetTargets()`
2. Shows confirmation prompt: "Untaint <address>?"
3. For batch: shows count and lists all addresses
4. On confirmation, executes `terraform untaint` for each address sequentially
5. On success, emits `PlanInvalidatedEvent` (plan auto-replans)
6. NavPush returns user to origin plugin

### States

```
Idle → Confirming → Loading → Done/Error
```

### Keybindings

| Key | State | Action |
|-----|-------|--------|
| `p` | Done | Navigate to plan |
| `Esc` | Any | Back to previous plugin |
| `ctrl+r` | Error | Retry |

## CLI

```bash
tfui untaint <address>           # Untaint single resource
tfui untaint <addr1> <addr2>     # Batch untaint
```

## Navigation

- **Nav behavior**: NavPush (preserves origin, returns on completion/cancel)
- **Menu visible**: No (hidden — reached via contextual keys or command mode)
- **Events emitted**: `PlanInvalidatedEvent` on success
