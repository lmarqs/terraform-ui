---
title: RecordingService Decorator (Command Recording Extraction)
status: planned
priority: high
created: 2026-05-12
effort: medium
tags: [architecture, macros, service]
depends_on: []
---

## Summary

Extract command recording from `StaticService` into a `RecordingService` decorator that wraps any `sdk.Service`. Eliminates hardcoded verb strings from `StaticService`, separates concerns, and enables recording on live services too.

## Need

`StaticService` currently has three responsibilities:
1. Serving pre-loaded plan/state data
2. Building CLI command strings (hardcoded verbs like `"state rm"`, `"taint"`, etc.)
3. Collecting commands for macro stdout output

This violates single responsibility, duplicates knowledge (verb strings exist both here and implicitly in `TerraformService`), and prevents recording on live services.

## Expected UX

Macro output behavior changes:
- Default: show only mutations (apply, taint, state rm) — clean pipeline for `| sh`
- `--macro-verbose`: include all commands (plan, state list, workspace show)

No change to plugin or TUI behavior. Internal architecture only.

## Advantages

- **Single responsibility**: StaticService serves data, RecordingService records
- **Composable**: Works with future CompositeService (wraps it naturally)
- **Reusable**: Can wrap live `TerraformService` for audit logging
- **Testable**: Each layer tested in isolation
- **Eliminates duplication**: Command verbs defined once

## Effort Justification

Medium effort:
- New service decorator with thread-safe shared store (1 day)
- StaticService simplification removes 100+ lines (0.5 days)
- Command filtering logic + flag (0.5 days)
- Integration test updates for new default (0.5 days)

Core abstraction is clean decorator pattern. Risk is minimal — wraps existing contracts.

## Design

### RecordingService (new file: `internal/terraform/recording_service.go`)

A decorator implementing `sdk.Service` that:
- Wraps any inner `sdk.Service`
- Records every operation as an `sdk.Command` (verb + args + flags)
- Delegates to the wrapped service for actual execution/data
- Shares a `*commandStore` across `WithDir()` copies for accumulation
- Thread-safe via mutex (plugins fire commands in goroutines)

```go
type commandStore struct {
    mu       sync.Mutex
    commands []sdk.Command
}

type RecordingService struct {
    inner  sdk.Service
    binary string
    store  *commandStore
}
```

### Simplified StaticService

After extraction:
- No more `commandErr()`, `record()`, `Commands()`, `binary`, or `commands` fields
- Mutating methods return `nil` (simulating success for macro playback)
- Pure data server for pre-loaded plan/state

### Wiring (`cmd/tfui/main.go`)

```go
recorder := terraform.NewRecordingService(staticSvc, cfg.TerraformBinary())
registry := buildRegistry(recorder, cfg)
```

### Command Filtering

Classification:
- **Read** (suppress by default): Plan, StateList, Show, Workspace, WorkspaceList, Validate, Output
- **Mutate** (always record): Apply, StateRm, StateMove, Import, Taint, Untaint, WorkspaceSelect, WorkspaceNew, WorkspaceDelete, Refresh, Init, ForceUnlock

Add `--macro-verbose` flag for full trace output.

## Open Questions

- Should RecordingService also track timing/duration for each command?
- Do we want structured output format (JSON) as an alternative to shell script?

## Interaction with Other Roadmap Items

| Item | Interaction |
|------|-------------|
| Composite Service | RecordingService wraps CompositeService in the decorator stack |
| Terraform Flag Passthrough | When service methods gain options structs, only RecordingService serialization changes |
| Plugin Refactoring | No impact — plugins are unaffected by the decorator |

## Tasks

- [ ] Create `internal/terraform/recording_service.go` with `commandStore` shared pattern
- [ ] Create `internal/terraform/recording_service_test.go` (table-driven, concurrency, WithDir)
- [ ] Wire `RecordingService` into `runMacro()`
- [ ] Strip command logic from `StaticService` (return nil for mutations)
- [ ] Add `CommandFilter` enum and `--macro-verbose` flag
- [ ] Update integration tests for mutations-only default

## References

- Current: `internal/terraform/static_service.go`
- Service interface: `pkg/sdk/service.go`
- Command type: `pkg/sdk/command.go`
- Macro wiring: `cmd/tfui/main.go` (`runMacro()`)
