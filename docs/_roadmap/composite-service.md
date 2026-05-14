---
title: Composite Service (Hybrid Read/Write Mode)
status: planned
priority: high
created: 2026-05-11
effort: large
tags: [source, service, ux]
depends_on: []
---

## Summary

Make `--plan` and `--state` behave as data source overrides, not as a read-only lockout. Each flag simply tells tfui WHERE to get the data — everything else works normally.

## Need

Today when a user passes `--plan ./plan.json` or `--state ./state.json`, the entire TUI becomes read-only. No mutations, no refresh, no live operations. This is wrong because:

1. **A user with `--state ./state.json` expects taint/mv/rm to work** — terraform itself supports `terraform state mv -state=./file.json`. The file is a valid state backend.
2. **A user who presses `r` (refresh) expects fresh data** — the file may have been updated by another process (CI pipeline, teammate, terraform running in parallel).
3. **A user with `--plan ./plan.json` but no `--state` expects state to load from terraform** — they just want to review a pre-generated plan, not lose access to state browsing.
4. **A user with `--state ./state.json` but no `--plan` expects plan to run live** — they pointed at a state file, they didn't ask to disable planning.

## Expected UX

```bash
# Review and apply a CI-generated plan (binary)
tfui --plan ./tfplan.out --chdir modules/global

# Review a JSON export (apply runs fresh)
terraform show -json tfplan.out | tfui --plan -

# Browse state from a pulled file, plan runs live
tfui --state ./terraform.tfstate --chdir modules/global

# Fully file-based (both overridden)
tfui --plan ./tfplan.out --state ./terraform.tfstate

# Refresh re-reads the file (catches external changes)
# Mutations on state file work via terraform -state= flag
```

**In the TUI:**

| Action | `--plan` set (file) | `--plan -` (stdin) | `--state` set |
|--------|--------------------|--------------------|---------------|
| View plan | `terraform show -json <file>` | Parse cached JSON | Runs live terraform |
| View state | Runs live terraform | Runs live terraform | Re-reads file |
| Refresh plan | Re-run `show -json <file>` | Returns cached data | Re-runs terraform plan |
| Refresh state | Re-runs terraform show | Re-runs terraform show | Re-reads file |
| Apply | `terraform apply <file>` | Runs fresh live plan+apply | Runs live terraform |
| Taint/Untaint | N/A (state op) | N/A (state op) | `terraform taint -state=<file>` |
| State mv/rm | N/A (state op) | N/A (state op) | `terraform state mv -state=<file>` |

**No `[read-only]` badge.** Nothing is read-only. The flags just change the data source.

## Design Decisions

### Plan file format: binary only

`--plan` accepts **binary plan files** (output of `terraform plan -out=`). This matches terraform's native workflow exactly:

```bash
tfui plan                          # produces tfplan.out (same as terraform plan -out=)
tfui --plan ./tfplan.out           # review AND apply that exact plan
terraform plan -out=tfplan.out && tfui --plan ./tfplan.out  # same thing, explicit
```

- `Plan()` reads the binary via `terraform show -json <file>` to parse changes for display
- `Apply()` runs `terraform apply <file>` directly — exact reproducibility, no re-plan
- One flag, one format, one mental model. No format detection needed.

Stdin (`terraform show -json tfplan.out | tfui --plan -`) remains supported for piped review — inherently view-only since you can't apply stdin.

### Refresh when state is a file?

**Re-read the file from disk.** Rationale:
1. `terraform refresh -state=<file>` would mutate the file to match real infrastructure, which is surprising when the user just piped a state snapshot.
2. The primary use-case is "file was updated externally" (CI pipeline, teammate). Re-reading catches those updates.
3. If the user wants to sync the file with real infra, they can run `terraform refresh -state=<file>` externally and then press `r` again.

Exception: when no state file is specified, `Refresh()` delegates to live terraform as today.

### Stdin sources

When the URI is `-` (stdin), data is consumed once at startup and cached. These bytes are non-refreshable — `Refresh()` returns the same cached data (cannot re-read stdin).

### Macros are command generators, never executors

Macros record what terraform *would* run and output commands to stdout. They never execute real terraform operations. The user decides whether to pipe to `sh`:

```bash
tfui --macro tape.txt --plan ./tfplan.out          # prints commands to stdout
tfui --macro tape.txt --plan ./tfplan.out | sh     # user opts in to execution
```

`RecordingService` stays as-is — wraps the composite service, captures commands, no mutations. This is a safety boundary: macros are deterministic and inspectable before execution.

## Architecture

### CompositeService

```go
type CompositeService struct {
    live       *TerraformService  // always available for live ops
    planFile   string             // absolute path to binary/JSON plan; "" = use live
    stateFile  string             // absolute path to state file; "" = use live
    stdinPlan  []byte             // cached stdin plan data (non-refreshable)
    stdinState []byte             // cached stdin state data (non-refreshable)
}
```

### Method Routing Table

| Method | `planFile` set | `stateFile` set | Neither |
|--------|---------------|-----------------|---------|
| `Plan()` | `terraform show -json <file>` → parse | Delegate to `live` | Delegate to `live` |
| `Apply()` | `terraform apply <file>` | Delegate to `live` | Delegate to `live` |
| `StateList()` | Delegate to `live` | Re-read file, parse | Delegate to `live` |
| `Show()` | Delegate to `live` | Re-read file, find resource | Delegate to `live` |
| `StateRm()` | Delegate to `live` | `live` with `-state=<path>` | Delegate to `live` |
| `StateMove()` | Delegate to `live` | `live` with `-state=<path>` | Delegate to `live` |
| `Import()` | Delegate to `live` | `live` with `-state=<path>` | Delegate to `live` |
| `Taint()` | Delegate to `live` | `live` with `-state=<path>` | Delegate to `live` |
| `Untaint()` | Delegate to `live` | `live` with `-state=<path>` | Delegate to `live` |
| `Refresh()` | Re-parse plan file | Re-read state file | Delegate to `live` |
| `Output()` | Delegate to `live` | `live` with `-state=<path>` | Delegate to `live` |
| `Validate()` | Delegate to `live` | Delegate to `live` | Delegate to `live` |
| `Init()` | Delegate to `live` | Delegate to `live` | Delegate to `live` |
| `Workspace*()` | Delegate to `live` | Delegate to `live` | Delegate to `live` |
| `ForceUnlock()` | Delegate to `live` | Delegate to `live` | Delegate to `live` |
| `WithDir()` | New composite with updated `live` | Same | Delegate |

### TerraformService: statePath support

Add optional `statePath string` field. When set, state-mutating operations pass `tfexec.State(statePath)`. terraform-exec supports this option for: `StateRm`, `StateMv`, `Import`, `Taint`, `Untaint`, `Output`.

```go
type TerraformService struct {
    workingDir string
    binaryPath string
    statePath  string  // NEW: optional -state= path for mutations
}

func NewServiceWithState(workingDir, binaryPath, statePath string) *TerraformService
```

`WithDir()` propagates `statePath` to the new instance. File paths are absolute, so they remain valid across chdir changes.

## Implementation Phases

### Phase 1: Foundation (no behavior change)

**Step 1 — TerraformService.statePath**

File: `internal/terraform/service.go`

- Add `statePath string` field
- Add `NewServiceWithState()` constructor
- Modify `StateRm`, `StateMove`, `Import`, `Taint`, `Untaint`, `Output` to pass `tfexec.State(s.statePath)` when non-empty
- Modify `WithDir` to propagate `statePath`

**Step 2 — Create CompositeService**

New file: `internal/terraform/composite_service.go`

- Implement `sdk.Service` interface
- Route methods per the routing table above
- Use `internal/terraform/loader.go` (`LoadPlan`, `LoadState`) for file parsing
- Resolve all paths to absolute at construction time

**Step 3 — Tests (TDD: written BEFORE step 2)**

New file: `internal/terraform/composite_service_test.go`

| Test | Scenario |
|------|----------|
| `TestComposite_Plan_FromFile` | planFile set → reads file, returns parsed summary |
| `TestComposite_Plan_FromStdin` | stdinPlan set → returns parsed data from cache |
| `TestComposite_Plan_LiveFallback` | no planFile → delegates to live |
| `TestComposite_StateList_FromFile` | stateFile set → reads file, returns resources |
| `TestComposite_StateList_LiveFallback` | no stateFile → delegates to live |
| `TestComposite_Show_FromFile` | stateFile set → reads file, finds resource |
| `TestComposite_Show_NotFound` | stateFile set, address missing → error |
| `TestComposite_StateRm_WithStatePath` | stateFile set → delegates to live (live has statePath) |
| `TestComposite_StateRm_LiveFallback` | no stateFile → delegates to live (no statePath) |
| `TestComposite_Apply_AlwaysLive` | regardless of flags → delegates to live |
| `TestComposite_Refresh_ReReadsFile` | write file, refresh, verify new content |
| `TestComposite_Refresh_StdinCached` | stdin source → returns same bytes |
| `TestComposite_Refresh_LiveFallback` | no files → delegates to live.Refresh |
| `TestComposite_WithDir_PropagatesLive` | WithDir updates live, preserves file paths |
| `TestComposite_Output_WithStatePath` | stateFile set → delegates with statePath |
| `TestComposite_FileNotFound` | file deleted → clear error with path |

### Phase 2: Wire up (behavior changes)

**Step 4 — Update cmd/tfui/main.go**

- New function: `buildCompositeService(cfg, planURI, stateURI) (sdk.Service, error)`
- Resolves URIs to absolute paths (or stdin bytes) via existing source abstraction
- Creates `TerraformService` (with optional `statePath`)
- Creates `CompositeService` wrapping live + file paths
- Remove `cfg.ReadOnly = true` assignments
- Update `runTUI`, `runMacro`, `runStaticNonInteractive`

**Step 5 — Remove ReadOnly from config**

File: `internal/config/config.go`

- Delete `ReadOnly` field from struct
- Compile errors guide cleanup

**Step 6 — Remove ReadOnly check in app**

File: `internal/ui/app.go`

- Remove `cfg.ReadOnly` condition (currently used to skip context picker)
- Replace with check for data override presence (plan/state flags set)

**Step 7 — Remove ErrReadOnly**

File: `pkg/sdk/errors.go`

- Remove `ErrReadOnly` variable
- Remove any references in production code

### Phase 3: Macro and Non-Interactive

**Step 8 — Macro mode**

- `RecordingService` wraps `CompositeService` transparently (implements `sdk.Service`)
- Ensure `runMacro` builds composite, then wraps with recorder

**Step 9 — Non-interactive mode**

- Use composite service: `Plan()` re-reads file, `StateList()` re-reads file
- Stdin sources work via cached bytes

### Phase 4: Context picker skip logic

**Step 10 — Fix context picker**

- Add `DataOverride bool` to config (or equivalent mechanism)
- Check `cfg.DataOverride` instead of `cfg.ReadOnly` in app.go

### Phase 5: Cleanup

**Step 11 — Remove StaticService**

- Delete `internal/terraform/static_service.go` and its test
- Or keep only for stdin-piped data if needed (but composite handles stdin via cached bytes, so likely deletable)

**Step 12 — Update integration tests**

- Tests referencing "readonly" workspace need updating
- New integration test: `--state ./file.json` + macro that does state operations

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Workspace calls when no terraform dir | `Workspace()` fails with no `.terraform/` | Catch error, fallback to `"default"` |
| File deleted between reads | `Refresh()` or `Plan()` returns confusing error | Return clear error including file path |
| `WithDir()` + relative file paths | State file path becomes invalid | Resolve all paths to absolute at construction |
| Binary plan staleness | User applies a plan generated hours ago; infra drifted | Terraform itself validates plan freshness — it will error if state serial changed |
| State mutations without `terraform init` | `state rm -state=<file>` needs providers | Let terraform's error propagate naturally |
| Stdin non-refreshable | User presses `r`, nothing changes | Acceptable — matches expectations; log debug message |
| Recording service compatibility | `RecordingService` wraps transparently | Macros never execute — they only emit commands to stdout |

## Files Changed

| File | Action |
|------|--------|
| `internal/terraform/composite_service.go` | **Create** |
| `internal/terraform/composite_service_test.go` | **Create** |
| `internal/terraform/service.go` | Modify (add statePath) |
| `cmd/tfui/main.go` | Modify (replace buildStaticService) |
| `internal/config/config.go` | Modify (remove ReadOnly) |
| `internal/ui/app.go` | Modify (remove ReadOnly check) |
| `pkg/sdk/errors.go` | Modify (remove ErrReadOnly) |
| `internal/terraform/static_service.go` | Delete (Phase 5) |
| `internal/terraform/static_service_test.go` | Delete (Phase 5) |
| `tests/integration/compat_test.go` | Modify (update assertions) |

## References

- Terraform `-state` flag documentation
- `tfexec.State(string)` option in terraform-exec
- Current implementation: `internal/terraform/static_service.go`
- Decorator pattern reference: `internal/terraform/recording_service.go`
- Source abstraction: `internal/source/source.go`
