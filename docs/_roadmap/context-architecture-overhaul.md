---
title: Context Architecture Overhaul
status: planned
priority: critical
created: 2026-05-22
effort: large
tags: [architecture, safety, context, data-flow]
depends_on: []
---

## Summary

Redesign state ownership so all terraform-affecting state lives in a single atomic Context, replaced on context switch. Eliminates a class of bugs where stale state leaks across contexts.

## Problem

Critical safety bug: pinned targets from one context (chdir/workspace) leaked into terraform commands in a different context. User applied with stale targets — TUI showed nothing, terraform received them.

Root cause is architectural: terraform-affecting state is scattered across PinService, ResolvedOptions, and plugin-local fields (`e.targets`). Context switches patch fields individually; any forgotten field leaks silently. The pattern "dispatch event, hope each plugin cleans up" is structurally unsound.

Additional bugs found during analysis:
- Parallelism/Lock/LockTimeout resolved from config but never delivered to terraform commands
- Plan plugin doesn't clear `e.targets` on reset
- Apply/Plan don't implement WorkspaceHandler (workspace switch doesn't reset targets)
- Apply explicitly preserves targets across chdir changes (intentional but wrong)

## Architecture Decisions

See ADR-0018 (atomic Context) and ADR-0019 (plan owns replan).

Core laws:
1. **Context is Atomic** — replaced entirely on switch, never patched field-by-field
2. **Single Source of Truth** — no component stores local copies of terraform-affecting state
3. **Data flows downstream** — Context → Plan → Apply. Each node derives from parent, resets on parent change
4. **One event, one behavior** — single ContextChangedEvent replaces ChdirChanged + WorkspaceChanged. Full reset always.

## Scope

### Phase 1: SDK types + event
- New `TerraformContext` type in `pkg/sdk/` (single source of truth)
- `ContextChangedEvent` + `ContextChangedHandler` (replaces separate chdir/workspace handlers)
- Rename existing `sdk.Context` (DI container) to `sdk.PluginDeps`

### Phase 2: App owns Context
- Atomic replacement in app on chdir/workspace change
- `resolveOptions()` builds a NEW Context (includes Parallelism/Lock/LockTimeout)
- Block context switch while any plugin reports Busy()

### Phase 3: Migrate plugins
- Replace `e.options`/`e.pins`/`e.targets` with reads from TerraformContext
- Replace HandleChdirChanged + HandleWorkspaceChanged with HandleContextChanged
- Move replan from apply into plan (ADR-0019)
- Apply receives plan file only; remove SetTargets()

### Phase 4: Cleanup
- Remove ResolvedOptions, PinService, BuildPlanOptions, BuildApplyOptions
- Remove ChdirHandler, WorkspaceHandler, PinsHandler interfaces
- Remove old event types from EventBus

## Verification

```bash
mise run check:lint && mise run test:unit && mise run check:build && mise run test:macro
```

Key behaviors:
- Context switch replaces all terraform-affecting state — no field survives
- Pins die on context switch (plugin-derived state)
- Parallelism/Lock/LockTimeout flow into terraform commands
- Plan owns replan; apply only receives plan files (TUI flow)
- `tfui apply --target=X` works (terraform's plan+apply mode)
- Context switch blocked while terraform operation is in-flight
