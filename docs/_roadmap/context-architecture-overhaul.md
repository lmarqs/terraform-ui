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

Refactor state ownership so all terraform-affecting state lives in a single atomic Context, replaced on context switch. Implements the data flow model defined in `CONTEXT.md` and the architectural decision in ADR-0018.

## Motivation

Critical safety bug: pinned targets from one context leaked into terraform commands in a different context. User applied with stale targets — TUI showed nothing, terraform received them. Wrong resources modified in the wrong module.

Root cause: terraform-affecting state is scattered across PinService, ResolvedOptions, and plugin-local fields. Context switches patch fields individually; forgotten fields leak silently.

### Bugs to fix

**Bug 1 (CRITICAL): Pinned targets persist across context switches**
- `PinService` is a single shared instance, never cleared on chdir/workspace change
- `internal/ui/app.go:232-245` — ChdirChangedEvent handler does NOT clear pins
- `plugins/apply/apply.go:127-135` — HandleChdirChanged deliberately preserves targets (wrong)

**Bug 2: Parallelism/Lock/LockTimeout never flow from config**
- `config.Resolve()` produces values but `resolveOptions()` only transfers VarFiles and Vars
- `pkg/sdk/options.go` is MISSING these fields
- PlanOptions/ApplyOptions always receive zero/nil for these

**Bug 3: Plan plugin doesn't clear targets on reset**
- `plugins/plan/plan.go:162-182` — `reset()` clears summary/filter/tree but NOT `e.targets`
- Next `Activate()` reads stale targets

**Bug 4: Apply and Plan don't implement WorkspaceHandler**
- Workspace switch within same chdir doesn't trigger any target reset

## Scope

### Phase 1: SDK types + event
- New `TerraformContext` type in `pkg/sdk/` (single source of truth for context-scoped state)
- `ContextChangedEvent` + `ContextChangedHandler` (replaces separate chdir/workspace handlers)
- Rename existing `sdk.Context` (DI container) to `sdk.PluginDeps`

### Phase 2: App owns Context
- Atomic replacement in app on chdir/workspace change
- `resolveOptions()` builds a NEW Context (includes Parallelism/Lock/LockTimeout)
- Context switch uses existing Cancellable/Busy semantics (ADR-0013): cancel if safe, block if holding lock
- Single `ContextChangedEvent` dispatch (replaces ChdirChanged + WorkspaceChanged)

### Phase 3: Migrate plugins
- Replace `e.options`/`e.pins`/`e.targets` with reads from TerraformContext
- Replace HandleChdirChanged + HandleWorkspaceChanged with HandleContextChanged
- Apply: remove `SetTargets()`, receive plan file only (see roadmap: plan-owns-replan)
- All plugins: single handler, full reset on context change

### Phase 4: Cleanup
- Remove ResolvedOptions, PinService, BuildPlanOptions, BuildApplyOptions
- Remove ChdirHandler, WorkspaceHandler, PinsHandler interfaces
- Remove old event types from EventBus

## Files involved

### SDK (`pkg/sdk/`)
- `context.go` — rename struct to `PluginDeps`
- `terraform_context.go` — NEW: single source of truth type
- `events.go` — add `ContextChangedEvent`/`ContextChangedHandler`
- `bus.go` — wire new handler
- `options.go` — eventually delete
- `pin_service.go` — eventually delete

### App (`internal/ui/`)
- `app.go` — own TerraformContext, atomic replacement, block switch while busy

### Plugins (`plugins/`)
- `apply/apply.go` — remove `e.targets`, `SetTargets()`, `StatusReplanning`
- `plan/plan.go` — remove `e.targets`, `e.options`, `e.pins`; read from TerraformContext
- `state/state.go` — remove `e.pins`; read from TerraformContext
- All others implementing `ChdirHandler` — migrate to `ContextChangedHandler`

## Verification

```bash
mise run check:lint && mise run test:unit && mise run check:build && mise run test:macro
```

Key behaviors:
- Context switch replaces all terraform-affecting state — no field survives
- Pins die on context switch
- Parallelism/Lock/LockTimeout flow into terraform commands
- Context switch cancels in-flight operations (or blocks if Busy) before replacing Context
