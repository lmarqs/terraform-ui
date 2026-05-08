# CLAUDE.md

## Overview

terraform-ui is a standalone pure-bash library that provides animated terminal feedback for `terraform plan` and `terraform apply` operations: spinner, elapsed timer, progress bar, and tree-view diff output.

## Architecture

```
lib/tfui.sh       — the library (source this to use)
tests/tfui-test.sh — BDD-style test suite
install.sh        — curl-pipe-bash installer
package.sh        — basher metadata
Formula/          — homebrew formula
```

### Design Principles

- Pure bash (3.2+), no compiled dependencies except `jq`
- Cross-platform: macOS and any Linux distro
- Designed to be `source`d into caller scripts, not executed directly
- Terminal UI writes to fd3, keeping stdout/stderr free for data and errors

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

## Commands

```bash
# Syntax check
bash -n lib/tfui.sh

# Run tests
bash tests/tfui-test.sh
```

## Development Workflow

1. Edit `lib/tfui.sh`
2. Run `bash tests/tfui-test.sh` to verify
3. Keep public API stable (`tfui_init`, `tfui_plan`, `tfui_confirm`, `tfui_apply`)

## Important Notes

- Never break the public API signature without a major version bump
- All internal functions/vars use `_tfui_` or `_TFUI_` prefix to avoid collisions when sourced
- The test suite uses mocked terraform — no real infra calls
- fd3 is the UI channel; render functions write there, never to stdout/stderr
