---
title: Test Layer Classification and CI Coverage
status: planned
priority: medium
created: 2026-05-11
effort: small
tags: [debt, testing, ci, docs]
depends_on: [ci-pipeline-unification]
---

## Summary

The project has 4 test layers (unit, golden, integration, macro/UI) but only unit+golden run in CI. Test types are not documented. Developers may not know integration and macro tests exist or how to run them.

## Need

What user pain does this solve? What's the current workaround?

- 64 unit test files run in CI, but 8 integration tests and 11 macro tapes are local-only
- No documentation of what each test layer covers and when to use it
- Missing `mise run test:all` task for running everything locally
- CI doesn't exercise real terraform operations or TUI interactions
- New contributors don't know about golden file tests (`-update` flag) or tape DSL
- Currently: run `go test ./...` and hope you know about build tags and tape files

## Expected UX

How the user interacts with this feature.

**CLI commands**:
```bash
mise run test              # fast unit+golden tests (default, ~5s)
mise run test:integration  # requires terraform on PATH (~30s)
mise run test:all          # runs all layers with appropriate timeouts (~45s)
```

**Developer workflow**:
1. Write code change
2. Run `mise run test` for quick feedback
3. Run `mise run test:all` before committing
4. CI fails if any layer breaks

**CI behavior**:
- Pull requests run all test layers automatically
- Integration tests run in parallel with unit tests (separate job)
- Clear failure messages indicate which layer broke

**Documentation**:
- CLAUDE.md Testing section explains when to use each layer
- Golden file workflow documented (update flag, assertion helper)
- Tape DSL syntax and macro test guidelines included

## Advantages

Why this is worth doing.

- **Catch real bugs**: Integration tests prevent regressions in terraform CLI interaction
- **UI confidence**: Macro tapes validate full TUI workflows (navigation, key handling)
- **Contributor clarity**: New developers know which test type to write
- **Golden workflow**: Documented update process reduces confusion around rendering changes
- **CI completeness**: All layers exercised automatically reduces "works on my machine"
- **Low effort**: Mostly docs + CI config, test infrastructure already exists

## Effort Justification

Why the effort estimate is small (< 1 day).

- Test infrastructure already exists (no new harness needed)
- Integration tests already tagged and runnable
- Macro tapes already work via Driver API
- Mostly writing docs + adding mise task + updating CI workflow
- No new code, just orchestration and documentation

## Test Inventory

Current test coverage breakdown:

| Layer | Count | Tag | Location | CI Run | Purpose |
|-------|-------|-----|----------|--------|---------|
| Unit | 64 files | none | `*_test.go` in-package | ✓ (matrix) | Logic correctness |
| Golden | 50 files | none | `testdata/golden/` | ✓ (same) | UI rendering snapshots |
| Integration | 8 files | `integration` | `tests/integration/` | ✗ → ✓ | Real terraform ops |
| Macro/UI | 11 tapes | `integration` | `tests/fixtures/tapes/` | ✗ → ✓ | TUI interaction sequences |

**Additional**:
- 1 BATS test (`tests/blast-radius.bats`) — external dependency test (consider for CI?)

## Design

Technical approach.

### 1. Add mise tasks

```toml
# mise.toml
[tasks."test:integration"]
description = "Run integration tests (requires terraform)"
run = "go test -tags=integration -timeout=60s ./tests/integration/..."

[tasks."test:all"]
description = "Run all test layers"
depends = ["test", "test:integration"]
```

### 2. Update CI workflow

Add job to `.github/workflows/ci.yaml`:

```yaml
test-integration:
  needs: lint
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: hashicorp/setup-terraform@v3
      with:
        terraform_version: "1.10.5"
    - uses: actions/setup-go@v5
      with:
        go-version-file: go.mod
    - run: go test -tags=integration -timeout=60s ./tests/integration/...
```

### 3. Document test layers in CLAUDE.md

Add subsection to Testing section:

**Test Layers**:

| Layer | When to use | Run command | Fixture/helper |
|-------|-------------|-------------|----------------|
| Unit | Pure logic, no I/O | `go test ./pkg/...` | Mock services |
| Golden | UI rendering, diff detection | `go test -update` | `sdktest.AssertGolden()` |
| Integration | Real terraform CLI interaction | `mise run test:integration` | `tests/fixtures/` |
| Macro/UI | Full TUI workflows | `mise run test:integration` | Tape DSL parser |

**Golden file workflow**:
1. Write test using `sdktest.AssertGolden(t, got, name)`
2. Run `go test -update` to generate `.golden` file
3. Commit golden file with test
4. Future runs diff against committed golden

**Tape DSL**:
```
key p
wait view Plan
assert view create
screenshot /tmp/plan.txt
```

See `tests/fixtures/tapes/README.md` for full syntax.

### 4. Document macro test guidelines

Add to CLAUDE.md:

**When to write macro tests**:
- Multi-step workflows (home → plugin → detail → action)
- Navigation stack behavior (push/pop frames)
- Key binding conflicts (ensure no leakage)
- Error recovery flows (service failure → user action)

**Not needed for**:
- Single-frame rendering (use golden tests)
- Pure logic (use unit tests)
- Terraform CLI calls (use integration tests)

## Open Questions

- Should BATS test run in CI? (requires Python + graphviz)
- Should macro tests run in CI or stay local-only? (may be flaky)
- Do we need separate `test:unit` and `test:golden` tasks or keep combined?

## Tasks

- [ ] Add `test:integration` and `test:all` tasks to `mise.toml`
- [ ] Add integration test job to CI workflow (after ci-pipeline-unification)
- [ ] Document test layers table in CLAUDE.md Testing section
- [ ] Document golden file workflow (`-update` flag, `AssertGolden()`)
- [ ] Document tape DSL syntax and macro test guidelines
- [ ] Add `tests/fixtures/tapes/README.md` with tape DSL reference
- [ ] Decide: include BATS test in CI or keep local-only?
- [ ] Update test-writer agent to know about all 4 layers
