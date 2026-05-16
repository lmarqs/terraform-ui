---
layout: default
parent: Plugins
title: Import
id: import
key: n
category: action
---

# Import

Import existing infrastructure into terraform state. Standalone verb plugin — mirrors `terraform import` as a top-level command.

## Interactive (TUI)

### Entry Points

| Context | Key | Behavior |
|---------|-----|----------|
| State list/detail | `n` | Navigate to import with cursor address pre-filled |
| Command mode | `:import` | Navigate to import (empty form) |

### Flow

1. Plugin shows form: prompts for resource address (pre-filled if from state), then resource ID
2. Shows confirmation: "Import <id> as <address>?"
3. On confirmation, executes `terraform import <address> <id>`
4. On success, emits `StateRefreshedEvent` + `PlanInvalidatedEvent`
5. NavPush returns user to origin plugin

### States

```
Form → Confirming → Loading → Done/Error
```

### Keybindings

| Key | State | Action |
|-----|-------|--------|
| `p` | Done | Navigate to plan |
| `Esc` | Any | Back to previous plugin |
| `ctrl+r` | Error | Retry |

## CLI

```bash
tfui import <address> <id>     # Direct import, no TUI
```

## Navigation

- **Nav behavior**: NavPush (preserves origin, returns on completion/cancel)
- **Menu visible**: No (hidden — reached via contextual keys or command mode)
- **Events emitted**: `StateRefreshedEvent` + `PlanInvalidatedEvent` on success
