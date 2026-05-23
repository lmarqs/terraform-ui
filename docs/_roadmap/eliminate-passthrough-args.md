---
title: Eliminate ExtraArgs Passthrough — Parse All Terraform Flags
status: planned
priority: high
created: 2026-05-23
effort: medium
tags: [arch, cli, correctness]
depends_on: []
---

## Summary

The `--` passthrough mechanism (`ExtraArgs`) is architecturally broken. ExecService uses terraform-exec's typed API which cannot forward arbitrary strings, so passthrough args are silently dropped during real execution. Only MacroService (recording-only) appends them. Users believe their flags reach terraform — they don't.

## Problem

1. **Silent data loss**: `tfui plan -- -no-color -compact-warnings` records the flags in MacroService but ExecService drops them. No error, no warning. The user thinks terraform received them.

2. **Inconsistent behavior across services**: MacroService includes ExtraArgs in recorded commands; ExecService ignores them. Same options struct, different semantics depending on which service implementation runs.

3. **Naming confusion**: `ExtraArgs` is vague. The init plugin has a UI form field called `extraArgs` that's a completely separate concept (user-typed init flags in the TUI). Same name, unrelated mechanisms.

4. **Phantom field propagation**: `ExtraArgs` lives on `Context`, `PlanOptions`, `ApplyOptions`, `InitOptions` — carried through the entire data model — yet has zero effect on actual terraform execution.

5. **False sense of completeness**: The config resolution chain documents `-- passthrough` as the final override, implying it works. Users who rely on it are silently getting incorrect behavior.

## Root Cause

`hashicorp/terraform-exec` exposes a typed API: `tfexec.PlanOption`, `tfexec.ApplyOption`, etc. Each supported flag has a constructor (`tfexec.Target(...)`, `tfexec.VarFile(...)`, `tfexec.Out(...)`). There is no `tfexec.RawArg(string)` escape hatch. Flags not in the typed API cannot be forwarded.

Terraform flags that fall through the gap (not in tfexec's typed API):
- `-compact-warnings`
- `-no-color`
- `-input=false`
- `-json` (partial — only on some commands)
- Provider-specific flags on `init`

## Design

Replace the blind passthrough with explicit typed support for every terraform flag tfui interacts with.

### Phase 1: Audit and catalog

Map every terraform flag across `plan`, `apply`, `init`, `refresh`, `import`, `taint`, `untaint`, `state rm`, `state mv`, `workspace` commands. For each, classify:

- **Already typed**: has a field on the Options struct and tfexec supports it (e.g., `-target`, `-var-file`, `-parallelism`)
- **Typed but not wired**: tfexec supports it but tfui doesn't expose it yet
- **Not in tfexec**: needs custom handling (exec the binary directly or contribute upstream)

### Phase 2: Extend Options structs

For flags that are actionable (user would realistically pass them), add proper typed fields:

```go
type PlanOptions struct {
    // ... existing ...
    NoColor          bool
    CompactWarnings  bool
    Input            *bool
}
```

Wire them through ExecService using either:
- tfexec constructors (if available)
- Direct binary invocation fallback for flags tfexec doesn't support

### Phase 3: CLI flag promotion

Promote commonly-used passthrough flags to proper tfui CLI flags:

```bash
tfui plan --no-color --compact-warnings
```

These parse into typed fields on the config, flow through Context, and reach ExecService properly.

### Phase 4: Delete ExtraArgs

- Remove `ExtraArgs` from `Context`, `PlanOptions`, `ApplyOptions`, `InitOptions`
- Remove `splitPassthrough()` from `normalize.go`
- Remove the `-- ` handling from `cmd/tfui/main.go`
- Remove MacroService's `flags = append(flags, opts.ExtraArgs...)` lines
- Remove init plugin's `editExtraArgs` UI form (replace with typed fields)
- If a user passes `--` with unknown flags, emit a clear error: "unknown flag: -foo (use tfui --help to see supported flags)"

### Phase 5: Upstream contribution (optional)

For flags terraform supports but tfexec doesn't, contribute PRs to `hashicorp/terraform-exec` to extend the typed API. This is the cleanest long-term solution.

## Alternatives Considered

**A. Keep passthrough, fix ExecService to forward raw args**

Rejected. terraform-exec's typed API is intentional — it validates flags and prevents injection. Bypassing it with raw string forwarding undermines safety. Also requires forking or wrapping tfexec.

**B. Shell out directly instead of using terraform-exec**

Rejected. Loses structured output, error typing, and the safety guarantees terraform-exec provides (context cancellation, environment isolation, etc.).

**C. Only fix the naming (ExtraArgs → PassthroughArgs) and document the limitation**

Rejected. Documenting broken behavior doesn't fix it. Users will still be surprised that their flags are silently dropped.

## Verification

- `grep -r "ExtraArgs\|splitPassthrough\|extraArgs" pkg/ internal/ plugins/ cmd/` returns nothing
- Every terraform flag used in macro tapes and integration tests reaches the actual binary
- `tfui plan -- -unknown-flag` produces a clear error, not silent success
- MacroService and ExecService produce identical flag sets (no divergence)

## Effort Estimate

Medium (~2-3 days):
- Phase 1 (audit): 2h — mechanical grep + terraform docs comparison
- Phase 2 (extend structs): 4h — straightforward field additions + wiring
- Phase 3 (CLI flags): 4h — cobra flag registration + tests
- Phase 4 (deletion): 2h — mechanical removal + verify tests pass
- Phase 5 (upstream): unbounded — separate effort, not blocking
