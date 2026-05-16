---
title: Untaint Plugin (standalone verb)
status: planned
priority: high
created: 2026-05-15
effort: medium
tags: [plugin, ux, workflow]
depends_on: []
---

## Summary

Extract untaint from the state plugin into its own action plugin, mirroring the taint plugin design. `terraform untaint` is a top-level verb, not a state sub-command.

## Problem

Same issues as taint:
1. Wrong location — untaint is not state management, it's reversing a recreation decision
2. Only reachable from state — user reviewing a plan showing unexpected recreation can't untaint from plan view
3. No post-action guidance — after untaint, user should re-plan to verify the recreation is gone

## Design

### Plugin Spec

```
ID:          untaint
Name:        Untaint
Type:        Action (transient)
Nav:         NavPush
Menu:        hidden
Reachable:   :untaint command, contextual T key in state/plan
```

### States

Confirming → Loading → Done/Error (identical lifecycle to taint)

### Views

**Confirmation:**
```
Untaint aws_instance.web?
This resource will no longer be recreated on next apply.

[y]es / [n]o
```

**Batch confirmation:**
```
Untaint 3 resources?
  aws_instance.web
  aws_subnet.private
  aws_vpc.main

These resources will no longer be recreated on next apply.

[y]es / [n]o
```

**Success:**
```
✓ Untainted aws_instance.web (0.8s)

p plan  Esc back
```

**Error:**
```
✗ Failed to untaint aws_instance.web
  Error: resource is not currently tainted

Esc back  ctrl+r retry
```

### Context Passing

Plugin exposes `SetTargets(addresses []string)` — same pattern as taint.

### Events Emitted

- `PlanInvalidatedEvent` — on successful untaint

### Keybinding Integration

| Context | Key | Behavior |
|---------|-----|----------|
| State list/detail | `T` | Navigate to untaint with cursor address |
| Plan list | `T` | Navigate to untaint with cursor address |
| Batch palette | `T` | Navigate to untaint with all pinned addresses |
| Command mode | `:untaint` | Navigate to untaint |

### CLI Surface

```bash
tfui untaint <address>
tfui untaint <addr1> <addr2>
```

## Migration

1. Remove `requestUntaint` and `batchUntaint` from `plugins/state/actions.go`
2. Remove `StateUntaintedMsg` handling from state plugin
3. State plugin's `T` key emits `UntaintRequestMsg{Address}`
4. Plan plugin gains `T` key that emits same `UntaintRequestMsg{Address}`
5. App handler routes `UntaintRequestMsg` → untaint plugin (NavPush)
