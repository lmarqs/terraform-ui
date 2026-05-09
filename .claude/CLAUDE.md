# CLAUDE.md

## Overview

terraform-ui is a standalone pure-bash library that provides animated terminal feedback for `terraform plan` and `terraform apply` operations: spinner, elapsed timer, progress bar, and tree-view diff output.

## Architecture

```
lib/tfui.sh              — the library (source this to use)
tests/*.bats             — BATS test suite (split by feature)
tests/helpers/           — shared test utilities
tests/fixtures/          — real terraform projects for integration testing
scripts/                 — install.sh, package.sh
Dockerfile.coverage      — kcov coverage runner
.github/workflows/       — CI (build → test → release)
```

### Design Principles

- Pure bash (3.2+), no compiled dependencies except `jq`
- Cross-platform: macOS and any Linux distro
- Designed to be `source`d into caller scripts, not executed directly
- Terminal UI writes to fd3, keeping stdout/stderr free for data and errors
- All tools managed via mise.toml — no global installs

### Naming Conventions

| Pattern | Role |
|---------|------|
| `tfui_*` | Public API |
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
mise run setup          # Install BATS helper libraries
mise run build          # Syntax check lib/tfui.sh
mise run test:run       # Run test suite (BATS + JUnit XML)
mise run coverage:run   # Run coverage via Docker + kcov
```

Run a single test file: `bats tests/format.bats`

## Development Workflow

1. Edit `lib/tfui.sh`
2. Run `mise run test:run` to verify
3. Keep public API stable (`tfui_init`, `tfui_plan`, `tfui_confirm`, `tfui_apply`)

## Testing

- BATS framework with real terraform fixtures (no mocks)
- Fixtures in `tests/fixtures/<scenario>/` with pre-seeded `terraform.tfstate`
- fd3 conflict: never use `exec 3>` in tests — use `3>/dev/null` or `3>"$file"` on function calls
- Name tests as BDD scenarios: "given X, plan shows Y"
- Use `_fixture_prepare "name"` + `_fixture_plan "msg"` helpers

## CI Pipeline

- `main.yaml` orchestrates: build → test → release (reusable workflow_call)
- Test matrix: ubuntu-latest + macos-latest, `fail-fast: false`
- CI steps call mise tasks — reproducible locally with same commands
- JUnit reports via `dorny/test-reporter@v1` (needs `checks: write`)
- Coverage via Docker (Dockerfile.coverage), uploaded to Codecov
- Only GitHub-specific integrations (test-reporter, codecov) are non-mise steps

## Conventions

- Commits: conventional commits (`feat:`, `fix:`, `test:`, `ci:`, `refactor:`, `docs:`, `chore:`)
- Mise tasks: noun:verb (`test:run`, `coverage:run`), single-word for simple tasks (`setup`, `build`)
- Slash commands: noun-verb matching mise tasks (`/test-run`, `/coverage-run`)
- Terraform: pinned to 1.14, fixtures use `required_version = ">= 1.14"`

## Important Notes

- Never break the public API signature without a major version bump
- All internal functions/vars use `_tfui_` or `_TFUI_` prefix to avoid collisions when sourced
- fd3 is the UI channel; render functions write there, never to stdout/stderr
