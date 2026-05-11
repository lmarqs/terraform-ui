---
title: Composite Service (Hybrid Read/Write Mode)
status: planned
priority: high
created: 2026-05-11
effort: large
tags: [source, service, ux]
depends_on: [source-abstraction]
---

## Summary

Replace `StaticService` with a `CompositeService` that sources plan and state independently. Each axis can be either a local file or live terraform — they're not coupled.

## Problem

Currently `--plan`/`--state` creates a fully read-only `StaticService`. But:

- Users expect `--state ./file.json` to support mutations (terraform's `-state=` flag does this)
- Refresh should re-read the file from disk (external process may have updated it)
- When only `--plan` is given, state should come from live terraform
- When only `--state` is given, plan should run live terraform

## Design

The service composes two independent sources:

| Source | Read ops | Mutations | Refresh |
|--------|----------|-----------|---------|
| `--plan ./file` | Return file contents | N/A (plan is a snapshot) | Re-read file from disk |
| `--state ./file` | Return file contents | Delegate to `terraform -state=./file` | Re-read file from disk |
| Not specified | Live terraform | Live terraform | Live terraform |

**terraform-exec integration for state mutations:**

```go
tf.StateMv(ctx, src, dst, tfexec.State("./state.json"))
tf.StateRm(ctx, addr, tfexec.State("./state.json"))
tf.Import(ctx, addr, id, tfexec.State("./state.json"))
tf.Taint(ctx, addr, tfexec.State("./state.json"))
```

**Key changes:**
- Replace `StaticService` with `CompositeService`
- `CompositeService` holds: `planSource` (file path or nil) + `stateSource` (file path or nil) + `TerraformService` (for live ops)
- When `planSource` is set: `Plan()` re-reads file, `Apply()` still runs live terraform
- When `stateSource` is set: `StateList()` re-reads file, mutations pass `-state=` to terraform
- Remove `ErrReadOnly` — nothing is read-only anymore

## Open Questions

- Should `Apply` work when plan comes from a file? (terraform can `apply <planfile>` but the file must be binary, not JSON)
- When state is a file, should `Refresh()` call `terraform refresh -state=./file` or just re-read the JSON?

## Tasks

- [ ] Create `CompositeService` implementing `sdk.Service`
- [ ] Plan source: re-read file on each `Plan()` call
- [ ] State source: re-read file on each `StateList()` call
- [ ] State mutations: pass `-state=<path>` to terraform-exec
- [ ] Remove `StaticService` (or keep as fallback for stdin-loaded data)
- [ ] Update `buildStaticService` in main.go to build CompositeService
- [ ] Update non-interactive fallback to re-read file

## References

- Terraform `-state` flag: all state subcommands support it
- terraform-exec: `tfexec.State(string)` option function
- Current StaticService: `internal/terraform/static_service.go`
