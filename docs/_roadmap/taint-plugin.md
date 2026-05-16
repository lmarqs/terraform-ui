---
title: Taint Plugin (standalone verb)
status: planned
priority: high
created: 2026-05-15
effort: medium
tags: [plugin, ux, workflow]
depends_on: []
---

## Summary

Extract taint from the state plugin into its own action plugin, matching terraform's verb-first CLI design where `terraform taint` is a top-level command.

## Problem

Taint currently lives inside the state plugin as an inline action. This creates workflow confusion:

1. **Wrong mental model**: `terraform taint` is not a `terraform state` sub-command — it's a peer of `plan` and `apply`. Bundling it in state conflates "operates on a resource address" with "is state management."
2. **No workflow continuity**: After tainting, nothing guides the user to plan. The operation succeeds silently and the user stays in state browser. In CLI, the user explicitly runs `terraform plan` next.
3. **Unreachable from plan**: User reviewing plan changes might see a resource they want recreated, but can't taint from plan view — they must navigate to state, find it again, then taint.
4. **No post-action guidance**: Taint's entire purpose is to affect the next plan. The current UX treats it as a terminal action.

## Design

### Plugin Spec

```
ID:          taint
Name:        Taint
Type:        Action (transient — arrive, confirm, execute, return)
Nav:         NavPush (preserves origin, returns on completion/cancel)
Menu:        hidden (no global keybinding)
Reachable:   :taint command, contextual t key in state/plan
```

### States

1. **Confirming** — Shows resource(s) and consequence
2. **Loading** — Executing terraform taint
3. **Done** — Success with navigation hints
4. **Error** — Failure with retry/escape

### Views

**Single resource confirmation:**
```
Taint aws_instance.web?
This resource will be destroyed and recreated on next apply.

[y]es / [n]o
```

**Batch confirmation (multiple pinned):**
```
Taint 3 resources?
  aws_instance.web
  aws_subnet.private
  aws_vpc.main

These resources will be destroyed and recreated on next apply.

[y]es / [n]o
```

**Success:**
```
✓ Tainted aws_instance.web (1.2s)

p plan  Esc back
```

**Error:**
```
✗ Failed to taint aws_instance.web
  Error: resource not found in state

Esc back  ctrl+r retry
```

### Context Passing

Plugin exposes `SetTargets(addresses []string)` called by the app before navigation. Addresses come from:
- State plugin: cursor resource address
- Plan plugin: cursor resource address
- Batch palette: all pinned addresses
- CLI: positional argument (`tfui taint <address>`)

### Events Emitted

- `PlanInvalidatedEvent` — on successful taint (plan plugin auto-replans if active)

### Keybinding Integration

| Context | Key | Behavior |
|---------|-----|----------|
| State list/detail | `t` | Navigate to taint with cursor address |
| Plan list | `t` | Navigate to taint with cursor address |
| Batch palette | `t` | Navigate to taint with all pinned addresses |
| Command mode | `:taint` | Navigate to taint (requires address from context) |

### CLI Surface

```bash
tfui taint <address>           # Direct taint, no TUI
tfui taint <addr1> <addr2>     # Batch taint
```

## Migration

1. Remove `requestTaint` and `batchTaint` from `plugins/state/actions.go`
2. Remove `StateTaintedMsg` handling from state plugin
3. State plugin's `t` key emits `TaintRequestMsg{Address}` instead of inline logic
4. Plan plugin gains `t` key that emits same `TaintRequestMsg{Address}`
5. App handler routes `TaintRequestMsg` → taint plugin (NavPush)

## Future

When `-replace` support is added to the plan plugin (modern workflow), taint becomes a legacy escape hatch but remains available for users on older terraform versions.
