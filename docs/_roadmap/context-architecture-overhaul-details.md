# Context Architecture Overhaul

## Trigger: Critical Safety Bug

A user applied terraform in one context (chdir A), switched to another context (chdir B), and applied again. The first apply had pinned targets; the second had none. BUT the second apply silently carried the targets from the first. The TUI showed no targets — terraform received them anyway.

**Impact**: wrong resources modified in the wrong module. Silent data corruption.

---

## Root Cause Analysis

### Bug 1: Pinned targets persist across context switches (CRITICAL)

**Where**: `PinService` (`pkg/sdk/pin_service.go`) is a single shared instance. Never cleared on chdir or workspace change.

**How it leaks**: When user presses `a` to apply, the app reads `a.pins.All()` at `internal/ui/app.go:404` and pushes targets into apply via `applyPlugin.SetTargets(pinned)`. If pins from a previous context still exist in PinService, they flow into apply.

**Why it wasn't caught**: Apply's `HandleChdirChanged` (line 128-135) has an explicit comment: "Apply intentionally preserves targets/confirmed/totalResources across scope changes." This was a deliberate design choice that turned out to be wrong.

**Relevant code**:
- `pkg/sdk/pin_service.go` — shared instance, no context awareness
- `internal/ui/app.go:84` — single `pins := sdk.NewPinService()` for the whole app
- `internal/ui/app.go:232-245` — `ChdirChangedEvent` handler: does NOT clear pins
- `internal/ui/app.go:256-262` — `WorkspaceChangedEvent` handler: does NOT clear pins
- `internal/ui/app.go:401-412` — `ApplyRequestMsg` handler: reads `a.pins.All()` → `SetTargets`
- `plugins/apply/apply.go:127-135` — `HandleChdirChanged`: deliberately preserves targets
- `plugins/apply/apply.go:175-181` — `RequestApply`: uses `e.targets` to decide if replan needed

### Bug 2: Parallelism/Lock/LockTimeout never flow from config (SILENT)

**Where**: `config.Resolve()` produces a `ResolvedConfig` with `Parallelism()`, `Lock()`, `LockTimeout()`. But `resolveOptions()` in `internal/ui/app.go:771-778` only transfers `VarFiles` and `Vars` to `ResolvedOptions`. The three execution-affecting fields are resolved but never delivered.

**How it leaks**: `PlanOptions` and `ApplyOptions` structs have `Parallelism`, `Lock`, `LockTimeout` fields. They're always zero/nil because `BuildPlanOptions`/`BuildApplyOptions` read from `ResolvedOptions` which doesn't have them.

**Relevant code**:
- `internal/config/hcl_types.go:80-93` — `ResolvedConfig` has the fields + getters
- `pkg/sdk/options.go:5-9` — `ResolvedOptions` is MISSING these fields
- `pkg/sdk/options.go:12-33` — `BuildPlanOptions`/`BuildApplyOptions` can't propagate what doesn't exist
- `internal/ui/app.go:771-778` — `resolveOptions()` only sets VarFiles and Vars

### Bug 3: Plan plugin doesn't clear targets on reset

**Where**: `plugins/plan/plan.go:162-182` — `reset()` clears summary, filter, tree, stream, etc. but NOT `e.targets` (line 49) or `e.pinnedOnly` (line 53).

**How it leaks**: `HandleChdirChanged` calls `reset()`, which leaves stale targets. Next `Activate()` (line 184) reads `e.targets` at line 189 and passes them to `BuildPlanOptions`.

### Bug 4: Apply and Plan don't implement WorkspaceHandler

**Where**: Neither plugin handles `WorkspaceChangedEvent`. Only `ChdirChangedEvent` is handled. A workspace switch within the same chdir doesn't trigger any reset of targets.

**How it leaks**: User pins resources in workspace "dev", switches to workspace "prod", applies. The targets from "dev" context are still active.

### Bug 5: State shared across contexts via mutable pointer

**Where**: `internal/ui/app.go:85` creates `opts := &sdk.ResolvedOptions{...}`. This pointer is passed to all plugins at Init. When `resolveOptions()` is called (line 771), it mutates the same pointer in place.

**How it leaks (conceptually)**: Not a race condition (BubbleTea is single-threaded), but the architectural problem is clear: there's one mutable bag of state, multiple consumers, and manual field-by-field updates that can go out of sync.

---

## The Architectural Problem (not just the bugs)

These bugs are symptoms of one structural defect: **terraform-affecting state is scattered across multiple owners with no mechanism to enforce consistency.**

Current state ownership:
```
App (internal/ui/app.go):
  - activeChdir, activeWorkspace (correctly scoped)
  - *ResolvedOptions (shared mutable pointer — VarFiles, Vars, ExtraArgs)
  - *PinService (shared instance — targets)

Plan plugin:
  - e.targets []string (LOCAL COPY — goes stale)
  - e.options *ResolvedOptions (pointer to shared — mutated behind its back)
  - e.pins *PinService (pointer to shared — never cleared)

Apply plugin:
  - e.targets []string (PUSHED by app on ApplyRequestMsg — stale after context switch)
  - e.options *ResolvedOptions (pointer to shared)

State plugin:
  - e.pins *PinService (pointer to shared — state plugin is the only one that clears it on chdir)
```

**On context switch (ChdirChangedEvent)**, the app:
1. Updates `activeChdir` ✓
2. Reloads child config ✓
3. Calls `resolveOptions()` which updates VarFiles/Vars on the shared pointer ✓
4. Does NOT clear pins ✗
5. Does NOT update Parallelism/Lock/LockTimeout ✗
6. Dispatches event to plugins (each plugin must handle its own cleanup) ✗ fragile

**On workspace switch (WorkspaceChangedEvent)**, the app:
1. Updates `activeWorkspace` ✓
2. Calls `resolveOptions()` ✓
3. Does NOT clear pins ✗
4. Dispatches event — but most plugins don't implement WorkspaceHandler ✗

The pattern "dispatch event, hope each plugin cleans up correctly" is structurally unsound. It's opt-in, partial, and has no compile-time enforcement.

---

## Decided Architecture

### Two Laws

**Law 1: Context is Atomic.** All terraform-affecting state forms a single indivisible unit. On context switch, the unit is **replaced** — never patched field-by-field. You can't forget to clear a field in a struct that no longer exists.

**Law 2: Single Source of Truth.** No component may maintain its own copy of terraform-affecting state. If it flows into a terraform CLI argument, it lives in exactly one place. Plugins read from it; they don't cache it.

### Data Flow Model

Data flows downstream. Invalidation flows upstream. Each node in the chain:
1. Receives input from its parent
2. Derives its own state from that input
3. Produces output for its children
4. Resets when parent signals change

```
App Context ──────→ Plan Plugin ──────→ Apply Plugin
(replaced            (+ pins,            (plan file,
 atomically           plan file)          confirm & run)
 on switch)
```

Three layers:
- **App Context**: working dir, workspace, var-files, vars, parallelism, lock, lock-timeout, scoped service. Owned by app. Replaced atomically on context switch.
- **Plugin-derived state**: pins, filter, tree mode, scroll position. Derived from or layered on the App Context. Invalidated (full reset) when parent changes.
- **Operation artifact**: plan file. Produced by plan. Consumed by apply. Immutable once created.

### Context Switch = One Event, One Behavior

Today: `ChdirChangedEvent` + `WorkspaceChangedEvent` + `PinsChangedEvent` as separate optional handlers.

After: **one** `ContextChangedEvent`. One handler interface. One contract. Plugins that need to react implement one method. The behavior is always: full reset. "Everything I knew is gone."

This eliminates:
- The "forgot to implement WorkspaceHandler" class of bugs
- The "partially handled chdir but not workspace" class of bugs  
- The need to remember which events to handle

### Pins

- Scoped to the current Context — they die on context switch because they're plugin-derived state (they live in the plugin layer, not the app Context layer)
- Shared across plugins within that Context (pin in state → visible in plan)
- Mutations go through messages: plugin emits `PinToggleMsg` → app/owning-plugin mutates the shared pin state → next View() render picks it up naturally
- NOT part of the app Context — they're the plan/state plugin's enrichment of it
- Because BubbleTea re-renders after every message, no event dispatch is needed for pin changes — View() always reads current state

### Plan Owns All Planning

Domain rule: **Plan = preparation. Apply = confirmation of what was already prepared.**

- If pins exist, plan runs a targeted plan (`terraform plan -target=X -target=Y`)
- This is equivalent to `ctrl+r` refresh but with targets — just a replan
- Plan never hands off to apply until the plan file matches the user's current intent (including targets)
- Apply receives a plan file and executes `terraform apply planfile.out` — no targets, no options, no re-planning

This means the "replan inside apply" (`StatusReplanning` in `plugins/apply/apply.go:179-181`) moves to plan.

### Apply Entry Points

- **TUI flow (from plan)**: apply receives a plan file reference. Shows summary for confirmation. Executes `terraform apply planfile.out`.
- **CLI flow (`tfui apply --target=X`)**: maps directly to `terraform apply -target=X` — this is terraform's own plan+apply-in-one-shot mode. No replan needed because terraform does it internally.

Both are explicit inputs at activation time. Neither reads from shared mutable state.

### Concurrency Safety

Context switches are **blocked** while a terraform operation is in-flight. User must cancel the operation or wait for it to complete. This prevents results from one context arriving in another context.

### The Global vs Context-Scoped Boundary

```
SESSION-GLOBAL (app lifetime)        CONTEXT-SCOPED (replaced on switch)
─────────────────────────────        ─────────────────────────────────────
Binary path                          Working directory
Extra args (CLI --)                  Workspace
Root config (tfui.hcl)               Var files (resolved from config)
Plugin registry                      Vars (resolved from config)
Logger                               Parallelism (resolved from config)
                                     Lock / Lock timeout (resolved from config)
                                     Scoped service (WithDir)
```

Decision rule: "Does this value have different correct values in different contexts?" If yes → context-scoped. If no → global.

- `ExtraArgs`: global (user typed `--` args once for the session)
- `VarFiles`: context-scoped (resolved per chdir + workspace from config)
- `Targets/Pins`: plugin-derived (die with context, not even part of the app Context)

### What's NOT Part of Context (and doesn't need to be)

- `treeMode`, `pinnedOnly`, filter text — pure UI state. Plugin-derived. Reset on context change, but not terraform inputs.
- Plan/state cache — derived output, already invalidated via `WithDir()` creating fresh cache
- Lock info — informational display, doesn't affect command construction
- Plan file reference — this is already scoped correctly via `filepath.Join(s.workingDir, planFileName)` after `WithDir()`

### Naming Decision

The domain term **Context** (in CONTEXT.md) is expanded to include resolved execution parameters. The init-time DI container currently named `sdk.Context` must be renamed to something like `sdk.PluginDeps` or `sdk.InitContext`. The domain concept reclaims the name.

---

## Implementation Plan

### Phase 1: New SDK type + event

1. Create `sdk.TerraformContext` (or just `sdk.Context` if renaming the DI container first) — single struct holding all context-scoped values, with methods: `PlanOptions()`, `ApplyOptions()`, `TogglePin()`, `IsPinned()`, `Targets()`, `SetPins()`, `ClearPins()`, `Dir()`, `Workspace()`, `Service()`
2. Add `ContextChangedEvent` and `ContextChangedHandler` interface to `pkg/sdk/events.go`
3. Wire `ContextChangedHandler` into `EventBus`
4. Rename existing `sdk.Context` (DI container) to `sdk.PluginDeps`
5. Update `internal/plugin/context.go` type alias accordingly

### Phase 2: App owns Context, atomic replacement

1. In `internal/ui/app.go`: create `TerraformContext` at startup, replace it atomically in chdir/workspace handlers
2. In `resolveOptions()`: build a NEW `TerraformContext` from resolved config + session globals (ExtraArgs), replacing the old one
3. Dispatch `ContextChangedEvent` instead of separate `ChdirChangedEvent`/`WorkspaceChangedEvent`
4. Block context switch when any plugin reports `Busy()`

### Phase 3: Migrate plugins

For each plugin:
1. Replace `e.options *sdk.ResolvedOptions` + `e.pins *sdk.PinService` with a reference to `TerraformContext`
2. Replace `HandleChdirChanged` + `HandleWorkspaceChanged` with `HandleContextChanged` (full reset)
3. Remove any local `e.targets` field — read from context at operation time
4. Plan plugin: move replan logic from apply into plan (before handing off to apply)
5. Apply plugin: remove `SetTargets()`, receive plan file on activation only

### Phase 4: Cleanup

1. Remove `ResolvedOptions`, `BuildPlanOptions`, `BuildApplyOptions` from `pkg/sdk/options.go`
2. Remove `PinService` from `pkg/sdk/pin_service.go` (or keep as internal helper if needed)
3. Remove `ChdirHandler`, `WorkspaceHandler`, `PinsHandler` interfaces (replaced by `ContextChangedHandler`)
4. Remove `ChdirChangedEvent`, `WorkspaceChangedEvent`, `PinsChangedEvent` types
5. Update `EventBus` to remove old dispatch paths

---

## Files Involved (by area)

### SDK (`pkg/sdk/`)
- `context.go` — rename struct to `PluginDeps`
- `terraform_context.go` — NEW: the single source of truth type
- `events.go` — add `ContextChangedEvent`/`ContextChangedHandler`, deprecate old events
- `bus.go` — wire new handler, eventually remove old dispatch
- `options.go` — eventually delete (replaced by TerraformContext methods)
- `pin_service.go` — eventually delete (replaced by TerraformContext methods)

### App (`internal/ui/`)
- `app.go` — own TerraformContext, atomic replacement on switch, block switch while busy

### Config (`internal/config/`)
- `hcl_resolve.go` — no change needed (already produces correct values)

### Plugins (all in `plugins/`)
- `apply/apply.go` — remove `e.targets`, `SetTargets()`, `StatusReplanning`. Receive plan file only.
- `plan/plan.go` — remove `e.targets`, `e.options`, `e.pins`. Read from TerraformContext. Own all replan.
- `state/state.go` — remove `e.pins`. Read from TerraformContext.
- All others implementing `ChdirHandler` — migrate to `ContextChangedHandler`

### Internal plugin bridge
- `internal/plugin/context.go` — update type alias

---

## ADR-0018 (to rewrite)

The current file at `docs/adr/0018-context-as-single-source-of-truth.md` is bloated and mixes architecture with implementation. Rewrite to:

```markdown
# Context is the single source of truth for terraform inputs

All state that flows into a terraform CLI argument (targets, var-files, vars, parallelism, lock, lock-timeout) lives in a single atomic Context owned by the app. On context switch the entire Context is replaced — never patched field-by-field. Plugins read from it but never store local copies.

We chose this over a tactical fix (clear pins at app level, patch individual handlers) because the field-by-field approach is structurally unsound: any new plugin or new field that forgets to clear on context switch silently leaks into the next terraform command. Atomic replacement eliminates the category of bug entirely.

## Considered Options

### Tactical fix: clear state in individual handlers

Cheaper to implement but leaves the structural defect intact. Every future plugin author must remember to implement cleanup for every context-change event. Rejected — treats symptoms.
```

---

## Dirty Working Tree State

Before starting implementation, clean up:
- `pkg/sdk/terraform_context.go` — DELETE (premature implementation, written before architecture was resolved)
- `docs/adr/0018-context-as-single-source-of-truth.md` — REWRITE (current content is bloated)
- `CONTEXT.md` — already updated correctly (keep as-is)

---

## Verification

```bash
mise run check:lint && mise run test:unit && mise run check:build && mise run test:macro
```

Key behaviors to verify:
- Context switch replaces all terraform-affecting state atomically — no field survives
- No targets survive context switch (pins are plugin-derived, die with reset)
- Parallelism/Lock/LockTimeout flow from config into terraform commands
- Plan owns replan; apply only receives and executes plan files (TUI flow)
- `tfui apply --target=X` still works (terraform's own plan+apply mode)
- Context switch blocked while terraform operation is in-flight
