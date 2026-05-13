---
title: Event Bus (replace session keys)
status: done
priority: high
created: 2026-05-12
completed: 2026-05-12
effort: medium
tags: [architecture, extensibility]
depends_on: []
---

# Event Bus: Replace Session Keys

## Status: Done

Implemented a typed event bus (`pkg/sdk/bus.go`, `pkg/sdk/events.go`) that replaces stringly-typed session key communication with compile-time-safe handler interfaces dispatched through BubbleTea's message loop.

## What Was Done

1. **Infrastructure**: `EventBus` type with typed dispatch table, `Event` marker interface, 5 event types, 5 handler interfaces
2. **ChdirChanged migration**: All 8 plugins implement `ChdirHandler` — `ChdirGuard` deleted entirely
3. **PlanCompleted migration**: Plan plugin publishes event, Apply plugin subscribes
4. **WorkspaceChanged migration**: Workspaces plugin publishes event, Context plugin subscribes, App updates header
5. **PlanInvalidated migration**: App publishes on editor close, Plan plugin subscribes to reset

## Remaining Work

- `PinsChangedEvent`: deferred — PinService operates synchronously during key handling, would need API redesign
- Session keys `config.var_files`, `config.vars`, `config.extra_args` remain — these are write-once boot config, not reactive
- `syncActiveScope()` remains as safety net during migration; can be removed once validated

## Architecture

```
Plugin returns tea.Cmd → produces Event msg
    ↓
App.Update() switch case → app-level reaction + session dual-write
    ↓
bus.Dispatch(msg) → fan-out to all handler implementations
    ↓
Handlers return tea.Cmd → batched back to BubbleTea
```

Subscription is interface-based: plugins implement `ChdirHandler`, `WorkspaceHandler`, etc. The bus discovers handlers via type assertion at init time (zero registration boilerplate).
