---
title: Plan Plugin Contextual Verb Keys
status: planned
priority: high
created: 2026-05-15
effort: small
tags: [plugin, ux, workflow]
depends_on: [taint-plugin, untaint-plugin]
---

## Summary

Add `t` (taint) and `T` (untaint) contextual keys to the plan plugin, allowing users to taint/untaint resources directly from the plan view without navigating to state first.

## Problem

A common terraform workflow:

1. User runs plan, sees unexpected drift on a resource
2. Decides "I want this resource recreated"
3. **Today**: must leave plan → navigate to state → find same resource → taint → go back to plan
4. **Expected**: press `t` on the resource in plan → taint → plan auto-refreshes

The plan view shows resources with addresses. The user already has context. Forcing them to leave, find, and act is unnecessary friction.

## Design

### New Keys in Plan List Frame

| Key | Action | Behavior |
|-----|--------|----------|
| `t` | Taint | Emit `TaintRequestMsg{Address}` for cursor resource |
| `T` | Untaint | Emit `UntaintRequestMsg{Address}` for cursor resource |

These keys navigate to the taint/untaint plugins (NavPush, returnTo=plan).

### Flow

```
Plan (shows changes) → user cursors to resource → t
  → Navigate to taint plugin with address
  → Taint plugin: confirm → execute → success
  → User presses Esc (or p)
  → Returns to plan
  → Plan receives PlanInvalidatedEvent → auto-replans
  → Plan shows updated changes (resource now marked for recreation)
```

### Hint Bar Update

Plan list frame hints (when status is Done):
```
a apply  t taint  T untaint  / filter  Space pin  Enter inspect  q back
```

### Edge Cases

- **Resource only in plan, not in state** (e.g., a "create" action): `t` is a no-op or shows "Cannot taint — resource does not exist in state yet." The taint plugin handles this error gracefully.
- **Resource being destroyed**: Taint on a resource planned for destruction is valid (forces destroy+recreate instead of just destroy). Let terraform handle the semantics.

### Auto-Replan After Return

Plan plugin already listens to `PlanInvalidatedEvent`. When the user returns from taint/untaint:
1. Event fires
2. Plan marks itself stale
3. Plan auto-replans
4. User sees updated results

No new code needed for this — existing event handling covers it.

## Migration

1. Add `t` and `T` key handlers to `plugins/plan/frames.go` listFrame
2. Extract resource address from cursor position in plan tree/list
3. Emit request messages (same type state plugin will emit)
4. Add hints for `t` and `T` to plan's hint list
