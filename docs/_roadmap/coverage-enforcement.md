---
title: Per-Layer Coverage Enforcement
status: planned
priority: high
created: 2026-05-11
effort: medium
tags: [debt, testing, ci]
depends_on: [ci-pipeline-unification]
---

## Summary

CLAUDE.md states "100% coverage on all packages excluding cmd/ glue" but the mise task enforces 90% flat, also excluding `internal/terraform`. Legacy go.yaml enforces 100% on everything including cmd/. No per-layer differentiation exists.

## Need

What user pain does this solve? What's the current workaround?

- Coverage threshold conflicts across documentation, mise task, and CI workflows create confusion about actual requirements
- Single flat threshold doesn't match the architecture's layer guarantees — can hide poor plugin coverage behind strong SDK coverage
- `internal/terraform` excluded from coverage but contains parseable pure-logic code (risk.go, phantom.go, grouping.go) that should be tested
- No visibility into which layer is dragging coverage down when CI fails
- Developers lack clear guidance on where to focus test-writing efforts

Current workaround: manually inspect coverage reports to identify weak areas, rely on code review to catch untested logic.

## Expected UX

How the user interacts with this feature:

```bash
$ mise run coverage:run
Testing pkg/sdk/... coverage: 100.0% (✓)
Testing internal/... coverage: 98.5% (✗ requires 100%)
  Missing coverage in:
    internal/config/loader.go:45-52
Testing plugins/... coverage: 100.0% (✓)
Skipping cmd/ (excluded)
Skipping internal/terraform/ (I/O boundary)

FAILED: 1 of 3 layers below threshold
```

CI shows exactly which layer fails if coverage drops, with file:line references for missing coverage.

## Advantages

Why this is worth doing:

- Clear per-layer reporting — can't hide poor plugin coverage behind strong SDK coverage
- Architecture layers match testing guarantees (public SDK = 100%, CLI glue = excluded)
- Documentation matches reality — no more threshold conflicts
- Targeted feedback — developers know exactly where to add tests
- Encourages test-writing at the right layer (catch bugs in pure logic, not I/O glue)

## Effort Justification

Why medium effort:

- Requires rewriting mise coverage task with per-layer check function (2-3 hours)
- Each layer needs its own coverage profile file generation
- Combined `coverage.out` must still be produced for CI artifact upload
- Documentation updates across CLAUDE.md and `.claude/commands/coverage.md`
- Likely need to fix existing test gaps to reach 100% per layer (biggest unknown — could be 1 day or 3 days depending on current state)
- Testing the coverage enforcement logic itself

Not large because the coverage tooling already exists — this is orchestration, not new infrastructure.

## Design

Technical approach:

### Thresholds

| Layer | Threshold | Rationale |
|-------|-----------|-----------|
| `pkg/sdk/` | 100% | Public API contract for plugins |
| `internal/` (excl. `internal/terraform`) | 100% | Core application logic |
| `plugins/` | 100% | Feature implementations |
| `cmd/` | excluded | CLI glue, integration-level testing only |
| `internal/terraform` | excluded | I/O boundary (service.go wraps terraform-exec) |

### Implementation

1. Rewrite `mise.toml [tasks."coverage:run"]` to:
   - Generate per-layer coverage profiles: `coverage-sdk.out`, `coverage-internal.out`, `coverage-plugins.out`
   - Run `go test -coverprofile` with package filters for each layer
   - Parse each profile with `go tool cover -func` to extract percentage
   - Compare against layer-specific threshold
   - Combine all profiles into `coverage.out` for CI upload
   - Exit with failure if any layer below threshold

2. Update `.claude/commands/coverage.md`:
   - Document per-layer thresholds table
   - Explain exclusion rationale
   - Add examples of passing/failing output

3. Update CLAUDE.md Testing section:
   - Replace "100% coverage on all packages excluding cmd/ glue" with layer-specific table
   - Remove mention of 90% threshold

4. Fix existing coverage gaps:
   - Run new coverage task to identify gaps
   - Add table-driven tests for uncovered branches
   - Focus on `internal/` and `plugins/` — SDK likely already at 100%

### Open Questions

- Should `internal/terraform` exclusion only apply to `service.go`, or entire package? (Risk/phantom/grouping logic is pure and testable)
- Do we need a `coverage:fix` task that auto-generates test stubs for uncovered functions?

## Tasks

- [ ] Add per-layer coverage check function to `mise.toml`
- [ ] Generate separate coverage profiles per layer
- [ ] Combine profiles into single `coverage.out` for CI
- [ ] Update `.claude/commands/coverage.md` with layer thresholds
- [ ] Update CLAUDE.md Testing section
- [ ] Run coverage task to identify gaps
- [ ] Write tests to fill coverage gaps (per layer)
- [ ] Test coverage enforcement on intentionally broken code
- [ ] Update CI workflow to parse per-layer output
