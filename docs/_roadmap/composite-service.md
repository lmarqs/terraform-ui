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
# Review a CI-generated plan, state comes live from terraform
tfui --plan ./plan.json --chdir modules/global

# Browse state from a pulled file, plan runs live
tfui --state ./state.json --chdir modules/global

# Fully file-based review (both overridden)
tfui --plan ./plan.json --state ./state.json

# Refresh re-reads the file (catches external changes)
# Mutations on state file work via terraform -state= flag
```

**In the TUI:**

| Action | `--plan` set | `--state` set |
|--------|-------------|---------------|
| View plan | Shows file contents | Runs live terraform |
| View state | Runs live terraform | Shows file contents |
| Refresh plan | Re-reads file from disk | Re-runs terraform plan |
| Refresh state | Re-runs terraform show | Re-reads file from disk |
| Taint/Untaint | N/A (state op) | `terraform taint -state=<file>` |
| State mv/rm | N/A (state op) | `terraform state mv -state=<file>` |
| Apply | Runs live terraform | Runs live terraform |

**No `[read-only]` badge.** Nothing is read-only. The flags just change the data source.

## Advantages

- **Matches terraform's own model** — `-state=<file>` is a first-class terraform feature, not a hack
- **File refresh catches external changes** — CI updates the plan, user presses `r`, sees new data
- **No lost functionality** — every feature works regardless of which flags are set
- **Enables offline-first workflows** — pull state once, browse/mutate locally without network

## Effort Justification

**Large** because:
- Replaces `StaticService` entirely with new `CompositeService`
- Service interface stays the same, but internal routing changes per-method
- terraform-exec integration needs `-state=` plumbing on 6 methods (StateRm, StateMove, Import, Taint, Untaint, Refresh)
- Non-interactive fallback needs update (re-read file instead of cached data)
- Tests need updating (mock behaviors change)

## Design

```go
type CompositeService struct {
    live       *TerraformService  // always available for live ops
    planFile   string             // if set, Plan() reads this file
    stateFile  string             // if set, StateList()/Show() read this file
}
```

Each method decides its source:
- `Plan()`: if `planFile` set → re-read + parse; else → `live.Plan()`
- `StateList()`: if `stateFile` set → re-read + parse; else → `live.StateList()`
- `StateRm()`: if `stateFile` set → `terraform state rm -state=<file>`; else → `live.StateRm()`
- `Apply()`: always → `live.Apply()` (apply is a live operation)

## Open Questions

- Should `Apply` work when plan comes from a JSON file? (terraform needs binary plan for apply, not JSON)
- When state is a file and user does `Refresh()`, should it call `terraform refresh -state=./file` (updates the file with real infra) or just re-read the file as-is?

## Tasks

- [ ] Create `CompositeService` implementing `sdk.Service`
- [ ] Plan source: re-read + parse file on each `Plan()` call
- [ ] State source: re-read + parse file on each `StateList()`/`Show()` call
- [ ] State mutations: pass `-state=<path>` to terraform-exec (6 methods)
- [ ] Remove `StaticService` (or keep only for stdin-piped data where no file path exists)
- [ ] Update `runTUI` to build CompositeService
- [ ] Remove `ReadOnly` flag and `[read-only]` badge
- [ ] Update non-interactive fallback to re-read file

## References

- Terraform `-state` flag documentation
- `tfexec.State(string)` option in terraform-exec
- Current implementation: `internal/terraform/static_service.go`
