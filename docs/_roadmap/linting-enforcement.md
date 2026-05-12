---
title: Linting Enforcement
status: planned
priority: high
created: 2026-05-11
effort: small
tags: [debt, tooling, dx]
depends_on: []
---

## Summary

`.golangci.yaml` configures 6 linters (importas, goimports, govet, errcheck, staticcheck, unused) but `mise run lint` only runs `go vet ./...`. golangci-lint is never installed or invoked — import alias rules, errcheck, staticcheck, and unused checks are dead config.

## Need

What user pain does this solve? What's the current workaround?

- Import alias enforcement (tfui* prefix) documented in CLAUDE.md but never validated
- errcheck violations likely exist undetected (unchecked error returns)
- staticcheck would catch subtle bugs
- unused code may be accumulating
- Developers think they're linted when running `mise run lint` but only get go vet
- Note: golangci-lint 1.64 (last v1 release) required because `.golangci.yaml` uses v1 config format. Go 1.25 requires golangci-lint 2.x which needs config migration.

**Current workaround:** Manual review catches some issues during PR, but no automated enforcement. Import alias violations and subtle bugs slip through.

## Expected UX

How the user interacts with this feature:

```bash
# Local development
mise run lint              # runs full golangci-lint suite
mise run vet               # quick go vet alternative

# Slash command in Claude Code
/lint                      # invokes real linter via mise

# CI behavior
# Pull requests automatically block on lint failures
# Same tool version as local (pinned via mise.toml)
```

Error output shows actionable lint violations with file:line and suggested fixes. `--fix` flag available for auto-fixable issues.

## Advantages

Why this is worth doing:

- **Import aliases actually enforced**: prevents silent drift from tfui* naming convention
- **Early detection**: unchecked errors, unused code, subtle bugs caught before review
- **Local/CI parity**: same tool, same version, same config everywhere
- **Developer confidence**: `mise run lint` does what you think it does
- **Code quality**: staticcheck catches issues go vet misses
- **Velocity**: auto-fix handles formatting, reduces review cycles

## Effort Justification

Why **small** (< 1 day):

- Tool installation: add 1 line to `mise.toml`
- Config migration: straightforward if needed (golangci-lint has migration docs)
- Task update: change 1 line in `mise.toml [tasks.lint]`
- Fix violations: `--fix` handles most issues automatically
- CI integration: copy existing Go job pattern, add lint step

Risk: unknown number of existing violations. Mitigation: fix in bulk with `--fix`, then address remainder.

## Design

Technical approach:

### Tool Setup

```toml
# mise.toml
[tools]
go = "1.25"
golangci-lint = "2"  # or 1.64 if Go 1.25 incompatible
```

Run `mise install` to download pinned version.

### Config Migration

If using golangci-lint 2.x, migrate `.golangci.yaml`:

```bash
golangci-lint config migrate --config .golangci.yaml
```

Review diff, commit migrated config. Key change: v1 → v2 format syntax.

### Task Definition

```toml
# mise.toml
[tasks.lint]
description = "Run golangci-lint"
run = "golangci-lint run ./..."

[tasks.vet]
description = "Quick go vet check"
run = "go vet ./..."
```

Update `.claude/commands/lint.md` to allow `golangci-lint` binary.

### Fix Existing Violations

```bash
# Auto-fix formatting/imports
mise run lint --fix

# Review remaining violations
mise run lint

# Fix errcheck/staticcheck/unused manually
# (likely small count, address one by one)
```

Commit fixes before enforcement.

### CI Integration

```yaml
# .github/workflows/test.yml
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jdx/mise-action@v2
      - run: mise run lint
```

No OS matrix needed — linting is deterministic, ubuntu-only sufficient.

## Open Questions

- Should we pin golangci-lint 1.64 (stable, no config migration) or upgrade to 2.x (requires migration, future-proof)?
- Do we want `--fast` mode for local lint (skip slow analyzers)?
- Should CI cache golangci-lint install, or is mise caching sufficient?

## Tasks

- [ ] Add `golangci-lint` to `mise.toml [tools]` with pinned version
- [ ] Migrate `.golangci.yaml` to v2 format (if using golangci-lint 2.x)
- [ ] Update `[tasks.lint]` to run `golangci-lint run ./...`
- [ ] Add `[tasks.vet]` task for quick go vet alternative
- [ ] Update `.claude/commands/lint.md` allowed-tools list
- [ ] Run `golangci-lint run --fix` to auto-fix violations
- [ ] Fix remaining errcheck/staticcheck/unused violations manually
- [ ] Add lint job to `.github/workflows/test.yml`
- [ ] Update CLAUDE.md Development Workflow section to mention lint task
- [ ] Document `--fix` flag usage for future contributors
