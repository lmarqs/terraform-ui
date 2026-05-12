---
title: CI/CD Pipeline Unification
status: planned
priority: high
created: 2026-05-11
effort: medium
tags: [debt, ci, release]
depends_on: [linting-enforcement]
---

## Summary

Two competing CI workflows (main.yaml path vs legacy go.yaml) cause double semantic-release runs, conflicting coverage thresholds, and unclear release ownership between semantic-release and goreleaser.

## Need

What user pain does this solve? What's the current workaround?

Current state:
- `go.yaml` triggers on same events as `main.yaml` resulting in double runs on push to main
- Coverage threshold conflict: go.yaml requires 100%, test.yaml runs 90%
- Release ownership unclear: semantic-release builds single binary, goreleaser builds cross-platform but never triggers
- OpenTofu compatibility testing lost (only in orphaned go.yaml)
- Coverage report in test.yaml looks for kcov/bats XML but Go produces text format
- Integration and macro tests never run in CI

Workarounds:
- Manual inspection of which workflow actually released
- No enforcement of integration/macro test passing
- OpenTofu compatibility unknown until production use

This section is PERMANENT — it doesn't change as design evolves.

## Expected UX

Single coherent pipeline. PR gets lint + test (2 OS × 2 TF matrix) + coverage + integration. Merge to main tags release. Tag triggers cross-platform goreleaser builds + Homebrew.

## Advantages

Why this is worth doing:
- No more duplicate runs — clearer CI logs, faster feedback
- Clear ownership: semantic-release = version, goreleaser = binaries
- All test layers exercised in CI (unit, golden, integration, macro)
- OpenTofu compatibility verified on every PR
- Coverage enforcement consistent across all workflows
- Release artifacts built once, published to GitHub + Homebrew atomically

## Effort Justification

Medium effort (2-5 days):
- Migration risk: must not break existing release flow
- Cross-workflow orchestration: semantic-release → goreleaser handoff
- Matrix job configuration: 4 OS/TF combos + parallel execution
- Integration with existing .releaserc and .goreleaser.yaml conventions
- Testing: full cycle validation in staging branch before main deployment

## Design

Technical approach:

### Workflow Structure

```
PR → test.yaml (lint + unit + coverage + integration)
main push → version.yaml (semantic-release creates tag)
tag push → release.yaml (goreleaser builds + publishes)
```

### File Changes

1. **Delete** `.github/workflows/go.yaml` (legacy)
2. **Create** `.github/workflows/version.yaml`:
   - Trigger: `push` to `main` branch
   - Jobs: `semantic-release` (creates tag, no build)
3. **Rewrite** `.github/workflows/release.yaml`:
   - Trigger: `push` with tag pattern `v*`
   - Jobs: `goreleaser` (builds all platforms, appends to existing release)
4. **Rewrite** `.github/workflows/test.yaml`:
   - Job 1: `lint` (golangci-lint)
   - Job 2: `unit` (matrix: ubuntu/macos × terraform/tofu)
   - Job 3: `coverage` (Go coverage format, 90% threshold)
   - Job 4: `integration` (fixtures + macro tapes)
5. **Update** `.github/workflows/main.yaml`:
   - Change PR path to call `test.yaml`
   - Change push path to call `version.yaml` instead of `release.yaml`
6. **Update** `.releaserc`:
   - `verifyConditions`: keep default
   - `prepare`: write VERSION file only (no build)
   - `publish`: GitHub release creation only (no asset upload)
7. **Update** `.goreleaser.yaml`:
   - Add `release.mode: append`
   - Add `changelog.disable: true` (semantic-release owns changelog)
   - Keep existing build matrix (cross-platform)

### Dependency Chain

```
linting-enforcement (prerequisite) → ci-pipeline-unification
```

Linting must be stable before CI reorganization to avoid conflating linting rule changes with CI structure changes.

## Open Questions

- Should coverage threshold be 90% or 100%? (Current conflict between workflows)
- Do we need separate coverage reports per OS/TF combo, or single merged report?
- Should integration tests run in matrix (all OS/TF combos) or single ubuntu+terraform?
- Homebrew formula update: automatic via goreleaser, or separate workflow?

## Tasks

- [ ] Delete `.github/workflows/go.yaml`
- [ ] Create `.github/workflows/version.yaml` (semantic-release on push to main)
- [ ] Rewrite `.github/workflows/release.yaml` as tag-triggered goreleaser
- [ ] Rewrite `.github/workflows/test.yaml` with 4 jobs: lint, unit (2×2 matrix), coverage, integration
- [ ] Update `.github/workflows/main.yaml` to call version.yaml instead of release.yaml
- [ ] Update `.releaserc`: prepareCmd writes VERSION only (no build), remove GitHub assets
- [ ] Update `.goreleaser.yaml`: add `release.mode: append`, `changelog.disable: true`
- [ ] Create staging branch to test full cycle end-to-end
- [ ] Verify first semantic-release → goreleaser cycle produces correct artifacts
- [ ] Update CLAUDE.md with new CI workflow documentation
