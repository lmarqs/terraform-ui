---
title: State Plugin Slimming (verb extraction)
status: planned
priority: high
created: 2026-05-15
effort: small
tags: [plugin, ux, refactor]
depends_on: [taint-plugin, untaint-plugin, import-plugin]
---

## Summary

After extracting taint, untaint, and import into standalone plugins, slim the state plugin to only handle genuine `terraform state` sub-commands: `list`, `show`, `rm`, and `mv`.

## Problem

The state plugin currently bundles 6 different terraform verbs:
- `terraform state list` (browse) ✓ belongs here
- `terraform state rm` (delete) ✓ belongs here
- `terraform state mv` (move) ✓ belongs here
- `terraform taint` ✗ top-level verb, not a state sub-command
- `terraform untaint` ✗ top-level verb, not a state sub-command
- `terraform import` ✗ top-level verb, not a state sub-command

After extraction, state becomes a focused browser for terraform's state management.

## Design

### What Stays

| Key | Action | Rationale |
|-----|--------|-----------|
| `d` | Delete (state rm) | Genuine `terraform state rm` |
| `m` | Move (state mv) | Genuine `terraform state mv` |
| `e` | Edit ($EDITOR) | Opens .tf file at resource definition |
| `/` | Filter | Browse/search |
| `Space` | Pin | Target scoping |
| `Enter` | Inspect | Detail view |
| `!` | Batch palette | Batch operations on pinned |

### What Changes (keys become navigation triggers)

| Key | Before | After |
|-----|--------|-------|
| `t` | Inline taint + confirmation | Emit `TaintRequestMsg` → navigate to taint plugin |
| `T` | Inline untaint + confirmation | Emit `UntaintRequestMsg` → navigate to untaint plugin |
| `n` | Inline import + input | Emit `ImportRequestMsg` → navigate to import plugin |

### Batch Palette (revised)

When pins > 0, `!` shows:
```
[d] delete  [t] taint  [T] untaint
```

Batch taint/untaint navigate to their respective plugins with all pinned addresses.
Batch delete stays inline (it's a genuine state operation).

### Post-Mutation Hints

After returning from taint/untaint/import (via `PlanInvalidatedEvent`):
- State plugin refreshes its resource list
- Hint bar shows `p plan` to encourage reviewing impact

### State Plugin Responsibilities (final)

1. **Browse** terraform state (list, filter, tree, inspect)
2. **Pin** resources for targeting
3. **Delete** resources from state (inline, with confirmation)
4. **Move** resources in state (inline, with input + confirmation)
5. **Navigate** to verb plugins for other operations (taint, untaint, import)

## Migration

1. Remove `requestTaint`, `batchTaint`, `StateTaintedMsg` handling
2. Remove `requestUntaint`, `batchUntaint`, `StateUntaintedMsg` handling
3. Remove `requestImport`, `StateImportedMsg` handling
4. Replace key handlers with navigation messages
5. Update batch palette to navigate for taint/untaint
6. Add `PlanInvalidatedEvent` listener for post-mutation hint

## Code Reduction

Estimated removal from `plugins/state/actions.go`:
- ~60 lines (taint/untaint functions + batch variants)
- ~30 lines (import function)
- ~20 lines (message handlers in Update)

Net result: state plugin focuses on what it owns (browse + rm + mv).
