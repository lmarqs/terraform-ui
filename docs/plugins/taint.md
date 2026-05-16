---
layout: default
parent: Plugins
title: Taint
id: taint
key: t
category: action
---

# Taint

Mark resources for recreation on next apply. Standalone verb plugin — mirrors `terraform taint` as a top-level command.

## Interactive (TUI)

### Entry Points

| Context | Key | Behavior |
|---------|-----|----------|
| State list/detail | `t` | Navigate to taint with cursor address |
| Plan list | `t` | Navigate to taint with cursor address |
| Batch palette (`!`) | `t` | Navigate to taint with all pinned addresses |
| Command mode | `:taint` | Navigate to taint |

### Flow

1. Plugin receives target address(es) via `SetTargets()`
2. Shows confirmation prompt: "Taint <address>? (will recreate on next apply)"
3. For batch: shows count and lists all addresses
4. On confirmation, executes `terraform taint` for each address sequentially
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
tfui taint <address>           # Taint single resource
tfui taint <addr1> <addr2>     # Batch taint
```

## Navigation

- **Nav behavior**: NavPush (preserves origin, returns on completion/cancel)
- **Menu visible**: No (hidden — reached via contextual keys or command mode)
- **Events emitted**: `PlanInvalidatedEvent` on success
