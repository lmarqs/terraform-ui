---
title: Service Layer Simplification
status: planned
priority: high
created: 2026-05-14
effort: medium
tags: [architecture, refactor, service, macro]
depends_on: []
---

## Summary

Eliminate `CompositeService` and simplify the service layer to two implementations: `ExecService` (live mode) and `MacroService` (macro mode). Pre-loaded data (`--plan`, `--state`) becomes a cache-seeding concern at the app layer, not a service variant.

## Problem

The current service layer conflates two orthogonal concerns:

1. **Mode of operation**: execute terraform for real vs record commands to stdout
2. **Data source**: pre-loaded from files vs fetched from terraform

This produced three service types stacked together:

```
Current architecture (3 layers, mixed concerns):

  RecordingService (decorator)
      ↓ delegates reads + records mutations
  CompositeService (hybrid)
      ↓ reads from files OR delegates to live
  TerraformService (executor)
      ↓ runs terraform binary
```

Problems:
- `CompositeService` exists solely to say "if I have file data, use it; otherwise call live." That's a caching concern, not a service concern.
- Pre-loaded data cannot be reused across service types — `CompositeService` is hardcoded to wrap `TerraformService`, so `MacroService` must also wrap `CompositeService` to access file data.
- Adding a new mode (e.g., `--dry-run` without macro) requires yet another wrapper.
- `CompositeService.Apply()` delegates to live — meaning pre-loaded mode is NOT read-only for mutations, which is a semantic surprise.
- The data-loading logic (reading plan.json, parsing state.json) is buried inside `CompositeService` and `buildCompositeService()` in main.go rather than being a reusable data pipeline.

## Target Architecture

```
Target architecture (2 services, data pre-seeded):

  App startup:
    1. Load phase (optional):
       --plan file.json  →  parse → *PlanSummary
       --state file.json →  parse → []Resource
    2. Seed cache with pre-loaded data (if any)
    3. Create service (one or the other):
       ExecService(binary, dir)    — live mode
       MacroService(binary, dir)   — macro mode

  Runtime:
    Plugin calls svc.Plan() →
      - If cache has PlanSummary: return cached (skip terraform call)
      - If cache is empty: ExecService calls terraform, MacroService returns empty
    Plugin calls svc.StateList() →
      - If cache has resources: return cached
      - If cache is empty: ExecService calls terraform, MacroService returns empty
    Plugin calls svc.Apply() →
      - ExecService: executes real terraform apply
      - MacroService: records command, returns nil
```

The key insight: **pre-loaded data is just pre-seeding the cache.** The service doesn't need to know WHERE the data came from. It only knows how to fetch fresh data (ExecService) or record operations (MacroService). Whether it ever needs to fetch depends on whether the cache is already warm.

## Design Details

### Phase 1: Rename existing types

| Current | New | File rename |
|---------|-----|-------------|
| `TerraformService` | `ExecService` | `service.go` → `exec_service.go` |
| `RecordingService` | `MacroService` | `recording_service.go` → `macro_service.go` |

### Phase 2: Move pre-loaded data out of CompositeService

The loading logic currently in `CompositeService` and `buildCompositeService()`:

```go
// cmd/tfui/main.go — current
func buildCompositeService(cfg, planURI, stateURI) (*CompositeService, error) {
    // reads files, creates CompositeService
}
```

Becomes:

```go
// cmd/tfui/main.go — target
func loadPreloadedData(planURI, stateURI) (*sdk.PlanSummary, []sdk.Resource, error) {
    // reads files, parses, returns domain objects
}
```

The parsed data is passed to a cache or directly to the app context.

### Phase 3: Add cache awareness to ExecService

```go
type ExecService struct {
    binary string
    dir    string
    cache  *ServiceCache  // pre-seeded with loaded data, or empty
}

func (s *ExecService) Plan(ctx context.Context, opts PlanOptions) (*PlanSummary, error) {
    if cached := s.cache.Plan(); cached != nil {
        return cached, nil
    }
    // call terraform binary
}

func (s *ExecService) StateList(ctx context.Context) ([]Resource, error) {
    if cached := s.cache.State(); cached != nil {
        return cached, nil
    }
    // call terraform binary
}
```

### Phase 4: MacroService uses same cache

```go
type MacroService struct {
    binary string
    dir    string
    cache  *ServiceCache
    store  *commandStore
}

func (s *MacroService) Plan(ctx context.Context, opts PlanOptions) (*PlanSummary, error) {
    s.record("plan", nil, buildPlanFlags(opts))
    if cached := s.cache.Plan(); cached != nil {
        return cached, nil
    }
    return &PlanSummary{}, nil  // no data available, return empty
}

func (s *MacroService) Apply(_ context.Context, opts ApplyOptions) error {
    s.record("apply", nil, buildApplyFlags(opts))
    return nil  // never executes
}
```

### Phase 5: Eliminate CompositeService

Delete `composite_service.go` and `composite_service_test.go`. All their tests migrate to `exec_service_test.go` (cache hit/miss behavior).

### Phase 6: Simplify main.go

```go
// cmd/tfui/main.go — target
func runTUI(cfg, planURI, stateURI) error {
    cache := terraform.NewServiceCache()
    if planURI != "" || stateURI != "" {
        plan, state, err := loadPreloadedData(planURI, stateURI)
        if err != nil { return err }
        cache.SeedPlan(plan)
        cache.SeedState(state)
    }
    svc := terraform.NewExecService(cfg.TerraformBinary(), effectiveWorkDir(cfg), cache)
    // ...
}

func runMacro(cfg, macroURI, planURI, stateURI) error {
    cache := terraform.NewServiceCache()
    if planURI != "" || stateURI != "" {
        plan, state, err := loadPreloadedData(planURI, stateURI)
        if err != nil { return err }
        cache.SeedPlan(plan)
        cache.SeedState(state)
    }
    svc := terraform.NewMacroService(cfg.TerraformBinary(), effectiveWorkDir(cfg), cache)
    // ...
}
```

## Behavioral Rules (invariants)

1. `ExecService` with empty cache → calls terraform binary (current behavior)
2. `ExecService` with seeded cache → returns cached data on first read, then follows normal cache lifecycle
3. `MacroService` with empty cache → returns empty data for reads, records mutations
4. `MacroService` with seeded cache → returns cached data for reads, records mutations
5. Mutations NEVER use cache (Apply always executes or records, never returns cached)
6. Explicit refresh (`svc.Refresh()`, user presses `r`) invalidates cache → next read re-fetches
7. User-initiated Plan (`p` key) always executes terraform and updates cache with fresh result
8. `--plan`/`--state` is just "seed the cache at startup" — not a mode, just a warm start
9. After first read consumes seeded data, the cache follows normal lifecycle (invalidate, re-fetch, update)

## Files Affected

| File | Change |
|------|--------|
| `internal/terraform/service.go` | Rename to ExecService, add cache field |
| `internal/terraform/state_ops.go` | Update receiver type name |
| `internal/terraform/workspace_ops.go` | Update receiver type name |
| `internal/terraform/recording_service.go` | Rename to MacroService, inline cache logic |
| `internal/terraform/composite_service.go` | **Delete** |
| `internal/terraform/composite_service_test.go` | **Delete** (tests migrate) |
| `cmd/tfui/main.go` | Simplify service creation, extract `loadPreloadedData()` |
| `internal/terraform/cache.go` | **New** — ServiceCache type |

## Migration Strategy

This can be done incrementally:
1. Rename types first (mechanical, no logic change)
2. Introduce `ServiceCache` alongside existing code
3. Wire cache into `ExecService`, keep `CompositeService` as fallback
4. Once all tests pass with cache, delete `CompositeService`
5. Wire cache into `MacroService`

## Test Plan

- Existing unit tests for `CompositeService` become cache-behavior tests for `ExecService`
- Existing `RecordingService` tests become `MacroService` tests (already renamed)
- Integration tests unchanged (they test the binary, not internal types)
- Macro tapes unchanged (they test end-to-end behavior)
- New tests: `ServiceCache` unit tests (seed, hit, miss, invalidate)

## Unlocked Feature: `--dry-run`

Once the service layer is simplified, `--dry-run` becomes trivial — same TUI, keyboard input, alt-screen, but `MacroService` instead of `ExecService`. When the user quits, print recorded commands to stdout.

The input method (tape vs keyboard) and the service mode (exec vs macro) are independent axes:

| Input | Service | Flag |
|-------|---------|------|
| Keyboard | ExecService | `tfui` (normal) |
| Keyboard | MacroService | `tfui --dry-run` |
| Tape | MacroService | `tfui --macro tape.tape` |

Implementation (~5 lines in `runTUI`):

```go
func runTUI(cfg, planURI, stateURI) error {
    cache := loadCache(planURI, stateURI)
    var svc sdk.Service
    if cfg.DryRun {
        svc = terraform.NewMacroService(binary, dir, cache)
    } else {
        svc = terraform.NewExecService(binary, dir, cache)
    }
    app := ui.NewApp(cfg, svc, registry)
    p := tea.NewProgram(app, tea.WithAltScreen())
    p.Run()
    if cfg.DryRun {
        for _, cmd := range svc.(*terraform.MacroService).Commands() {
            fmt.Println(cmd.String())
        }
    }
    return nil
}
```

User experience:
```bash
# Normal interactive session
tfui

# Same session, but nothing executes — commands printed on exit
tfui --dry-run
# user navigates, plans, applies, quits
# stdout:
# terraform plan
# terraform apply -target=aws_instance.web

# Pipe to sh if happy
tfui --dry-run | sh
```

This validates the architecture: if `--dry-run` is hard to add, the service layer is coupled wrong. If it's 5 lines, the separation is clean.

## Open Questions

- Should `ServiceCache` be in `internal/terraform/` or `pkg/sdk/`? Plugins don't interact with it directly, so likely internal.
- Should the cache support partial seeding (plan only, state only)? Yes — `--plan` without `--state` is valid.
- Does `WithDir()` need to clear the cache? Probably yes — changing dir means different terraform state.
