# CLAUDE.md

## Overview

terraform-ui is a standalone pure-bash tool that provides animated terminal feedback for `terraform plan` and `terraform apply` operations: spinner, elapsed timer, progress bar, and tree-view diff output.

## Architecture

```
bin/tfui                 — CLI entry point (executable)
lib/tfui.sh              — the library (source this to embed)
tests/*.bats             — BATS test suite (split by feature)
tests/helpers/           — shared test utilities
tests/fixtures/          — real terraform projects for integration testing
scripts/                 — install.sh, package.sh
Dockerfile.coverage      — kcov coverage runner
.github/workflows/       — CI (build → test → release)
```

### Layers

```
┌─────────────────────────────────────────────┐
│  CLI  (bin/tfui)                            │
│  Argument parsing, fd3 setup, dispatch      │
├─────────────────────────────────────────────┤
│  Public API  (tfui_*)                       │
│  init, plan, confirm, apply                 │
├─────────────────────────────────────────────┤
│  Orchestration  (_tfui_run*)                │
│  Strategy resolution, phase sequencing      │
├─────────────────────────────────────────────┤
│  Strategies  (_tfui_strategy_*)             │
│  silent, spinner, progress                  │
├──────────────────────┬──────────────────────┤
│  UI Engine           │  Execution           │
│  (_tfui_ui_*)        │  (_tfui_exec)        │
│  Animation, layout,  │  Working dir, output │
│  render, format      │  capture, exit codes │
├──────────────────────┴──────────────────────┤
│  Renderer  (_tfui_render_*)                 │
│  Plan tree view (jq)                        │
├─────────────────────────────────────────────┤
│  State  (_tfui_state_*)                     │
│  Message, timer, output file                │
├─────────────────────────────────────────────┤
│  Lifecycle  (_tfui_lifecycle_*)             │
│  Exit trap, die, cleanup                    │
└─────────────────────────────────────────────┘
```

### Layer Responsibilities

| Layer | Prefix | Responsibility |
|-------|--------|----------------|
| CLI | `_tfui_cli_*` | Parse args, validate deps, open fd3, dispatch to public API |
| Public API | `tfui_*` | User-facing contract; composes orchestration and rendering |
| Orchestration | `_tfui_run*` | Prepare state, resolve strategy, handle errors |
| Strategies | `_tfui_strategy_*` | Execute commands with a specific UI treatment |
| UI Engine | `_tfui_ui_*` | Cursor management, animation loop, render to fd3 |
| Formatters | `_tfui_ui_format_*` | Pure functions returning formatted strings (no side effects) |
| Renderers | `_tfui_ui_render_*` | Write formatted content at current cursor position (fd3) |
| Execution | `_tfui_exec` | Run shell commands in working dir, capture output |
| Output Renderer | `_tfui_render_*` | Parse plan JSON into tree view (stdout) |
| State | `_tfui_state_*` | Mutate internal variables (_TFUI_*) |
| Lifecycle | `_tfui_lifecycle_*` | EXIT trap, error handling, temp file cleanup |

### Design Principles

- Pure bash (3.2+), no compiled dependencies except `jq`
- Cross-platform: macOS and any Linux distro
- Two usage modes: CLI (`tfui plan`) or library (`source lib/tfui.sh`)
- Terminal UI writes to fd3, keeping stdout/stderr free for data and errors
- CLI pre-opens fd3; library detects and skips if already open
- All tools managed via mise.toml — no global installs

### Naming Conventions

| Pattern | Role |
|---------|------|
| `tfui_*` | Public API |
| `_tfui_cli_*` | CLI internals |
| `_tfui_run*` | Orchestration |
| `_tfui_state_*` | State mutators |
| `_tfui_lifecycle_*` | Process lifecycle |
| `_tfui_ui_render_*` | Render at cursor position (fd3) |
| `_tfui_ui_format_*` | Pure formatters (return string via stdout) |
| `_tfui_ui_*` | Layout and animation control |
| `_tfui_strategy_*` | Execution strategies |
| `_tfui_render_*` | Output formatting (tree view) |
| `_TFUI_*` | Internal variables |

### Strategies

- `silent` — no UI (plain mode)
- `spinner` — one-line animated spinner (simple mode)
- `progress` — two-line: spinner + progress bar (rich mode)

## Mise Tasks

```bash
mise run setup               # Install BATS helper libraries
mise run build [version]     # Package dist/ (default: reads VERSION file)
mise run test:run            # Syntax check + run test suite
mise run coverage:run        # Run coverage via Docker + kcov
mise run changelog           # Generate CHANGELOG.md from conventional commits
```

Run a single test file: `bats tests/format.bats`

## Development Workflow

1. Edit `lib/tfui.sh` or `bin/tfui`
2. Run `mise run test:run` to syntax-check and test
3. Run `mise run build` to produce dist/
4. Keep public API stable (`tfui_init`, `tfui_plan`, `tfui_confirm`, `tfui_apply`)

## Testing

- BATS framework with real terraform fixtures (no mocks)
- Fixtures in `tests/fixtures/<scenario>/` with pre-seeded `terraform.tfstate`
- fd3 conflict: never use `exec 3>` in tests — use `3>/dev/null` or `3>"$file"` on function calls
- Name tests as BDD scenarios: "given X, plan shows Y"
- Use `_fixture_prepare "name"` + `_fixture_plan "msg"` helpers

## CI Pipeline

```
main.yaml (orchestrator)
  │
  ├─ build.yaml
  │    ├─ Checkout with full history (fetch-depth: 0)
  │    ├─ Compute next version (git-cliff --bumped-version)
  │    ├─ mise run build (packages dist/ + tarball)
  │    ├─ mise run changelog (generates CHANGELOG.md via git-cliff)
  │    └─ Upload artifacts (dist/ + tarball + CHANGELOG.md)
  │
  ├─ test.yaml (needs: build)
  │    ├─ Matrix: ubuntu-latest + macos-latest, fail-fast: false
  │    ├─ mise run test:run (syntax check + tests)
  │    ├─ JUnit report via dorny/test-reporter@v1
  │    └─ Coverage via Docker + kcov
  │
  └─ release.yaml (needs: test)
       ├─ Download all artifacts (incl. pre-built tarball)
       ├─ Read VERSION from artifact to resolve tag
       │    push to main → vX.Y.Z (official)
       │    PR           → vX.Y.Z-rc.<timestamp> (prerelease)
       ├─ Create git tag via GitHub API
       ├─ Create GitHub release (tarball + CHANGELOG.md + test reports)
       └─ Push only: commit CHANGELOG.md + VERSION back to main (via GitHub API)
```

### Versioning

- Git tags are the source of truth for released versions
- git-cliff computes the next version from conventional commits since last tag
  - `feat:` → minor bump, `fix:` → patch bump, `BREAKING CHANGE` → major bump
- Build writes the computed version into VERSION and embeds it in the artifact
- Release commits VERSION back to main so the repo always shows the latest released version
- CLI reads VERSION relative to its install path; local dev defaults to "dev"
- Tags follow `vX.Y.Z` format, created by release via GitHub API
- Release commit uses `[skip ci]` to prevent infinite CI loops

### Changelog

- Generated by git-cliff from conventional commits (`cliff.toml` config)
- Build is the only pipeline with deep fetch (`fetch-depth: 0`) — it owns history access
- Changelog is uploaded as a separate artifact; downstream steps consume it without checkout
- Release commits `CHANGELOG.md` back to main via the GitHub Contents API (no checkout needed)
- Available locally: `mise run changelog`

### Design Decisions

- Build is a mise task — runs identically locally and in CI
- Only build checks out with full history — test uses shallow checkout, release has no checkout
- Build produces the release tarball (preserving permissions) — release only copies it to assets
- Release resolves official vs prerelease from event type, not from build
- Release creates tags and commits files via GitHub API (no git clone needed)
- Test reports are release assets for traceability

## Conventions

- Commits: conventional commits (`feat:`, `fix:`, `test:`, `ci:`, `refactor:`, `docs:`, `chore:`)
- Mise tasks: noun:verb (`test:run`, `coverage:run`), single-word for simple tasks (`setup`, `build`)
- Slash commands: noun-verb matching mise tasks (`/test-run`, `/coverage-run`)
- Terraform: pinned to 1.14, fixtures use `required_version = ">= 1.14"`

## Important Notes

- Never break the public API signature without a major version bump
- All internal functions/vars use `_tfui_` or `_TFUI_` prefix to avoid collisions when sourced
- fd3 is the UI channel; render functions write there, never to stdout/stderr
