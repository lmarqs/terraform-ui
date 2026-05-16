---
title: Import Plugin (standalone verb)
status: planned
priority: high
created: 2026-05-15
effort: medium
tags: [plugin, ux, workflow]
depends_on: []
---

## Summary

Extract import from the state plugin into its own action plugin. `terraform import` is a top-level verb — it brings existing infrastructure under terraform management, which is conceptually different from state manipulation (rm/mv).

## Problem

1. **Wrong grouping**: Import is not state management. `terraform state rm/mv` reorganize what's already tracked. `terraform import` brings something new in. Different intent, different risk profile.
2. **Incomplete form**: The current inline import via `InputText` only asks for the resource ID. The address is taken from cursor position — but this only works when the address already exists in state (re-import), not for importing new resources.
3. **No post-import guidance**: After import, user should plan to verify the imported resource matches configuration. No hint is shown.
4. **Single entry point**: Only reachable from state plugin. Cannot import from plan view (where "resource not in state" errors appear).

## Design

### Plugin Spec

```
ID:          import
Name:        Import
Type:        Action (transient, with form)
Nav:         NavPush
Menu:        hidden
Reachable:   :import command, contextual n key in state
```

### States

Form → Confirming → Loading → Done/Error

### Views

**Form (fresh, no context):**
```
Import Resource

Address: [                          ]
ID:      [                          ]

Enter submit  Esc cancel
```

**Form (pre-filled from state cursor):**
```
Import Resource

Address: [aws_instance.web         ]
ID:      [                          ]

Enter submit  Esc cancel
```

**Confirmation:**
```
Import i-0abc123def456 as aws_instance.web?

[y]es / [n]o
```

**Success:**
```
✓ Imported aws_instance.web (3.4s)

p plan  Esc back
```

**Error:**
```
✗ Failed to import aws_instance.web
  Error: resource address not found in configuration

Esc back  ctrl+r retry
```

### Context Passing

Plugin exposes:
- `SetAddress(address string)` — pre-fills the address field
- `SetID(id string)` — pre-fills the ID field (optional)

When navigated from state with cursor on a resource, address is pre-filled. When reached via `:import`, form starts empty.

### Events Emitted

- `StateRefreshedEvent` — state has a new resource
- `PlanInvalidatedEvent` — plan should re-run to check config drift

### Keybinding Integration

| Context | Key | Behavior |
|---------|-----|----------|
| State list/detail | `n` | Navigate to import with cursor address pre-filled |
| Command mode | `:import` | Navigate to import (empty form) |

### CLI Surface

```bash
tfui import <address> <id>     # Direct import, no TUI
```

## Migration

1. Remove `requestImport` from `plugins/state/actions.go`
2. Remove `StateImportedMsg` handling from state plugin
3. State plugin's `n` key emits `ImportRequestMsg{Address}`
4. App handler routes `ImportRequestMsg` → import plugin (NavPush)

## Future

Terraform 1.5+ supports `import` blocks in configuration. A future enhancement could offer generating the import block instead of running the imperative command.
