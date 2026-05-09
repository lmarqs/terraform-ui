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
- `agent` — no UI, structured JSON output (agent mode)

## Mise Tasks

```bash
mise run setup               # Install npm + BATS dependencies
mise run build [version]     # Package dist/ + tarball (default: reads VERSION file)
mise run test:run            # Syntax check + run test suite
mise run coverage:run        # Run coverage via Docker + kcov
mise run release             # Run semantic-release (CI only)
```

Run a single test file: `bats tests/format.bats`

## Development Workflow

1. Edit `lib/tfui.sh` or `bin/tfui`
2. Run `mise run test:run` to syntax-check and test
3. Run `mise run build` to produce dist/
4. Keep public API stable (`tfui_init`, `tfui_plan`, `tfui_confirm`, `tfui_apply`)
5. Test agent mode output: `tfui plan --dir tests/fixtures/<name> --mode agent | jq .`

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
  ├─ test.yaml
  │    ├─ Matrix: ubuntu-latest + macos-latest, fail-fast: false
  │    ├─ mise run test:run (syntax check + tests)
  │    ├─ JUnit report via dorny/test-reporter@v1
  │    └─ Coverage via Docker + kcov
  │
  └─ release.yaml (needs: test, push only)
       ├─ Checkout with full history (fetch-depth: 0)
       ├─ npm ci + npx semantic-release
       ├─ Computes version from conventional commits
       ├─ Builds tarball via @semantic-release/exec
       ├─ Creates GitHub release with tarball asset
       └─ Commits CHANGELOG.md + VERSION back to main
```

### Versioning & Release

- Powered by [semantic-release](https://github.com/semantic-release/semantic-release) (`.releaserc`)
- Version determined from conventional commits since last tag
  - `feat:` → minor bump, `fix:` → patch bump, `BREAKING CHANGE` → major bump
- `@semantic-release/exec` runs `mise run build` with the computed version
- `@semantic-release/github` uploads the tarball to GitHub Releases
- `@semantic-release/git` commits CHANGELOG.md + VERSION back to main
- Release commit uses `[skip ci]` to prevent infinite CI loops
- CLI reads VERSION relative to its install path
- Tags follow `vX.Y.Z` format

### Design Decisions

- Build is a mise task — runs identically locally and in CI
- Test uses shallow checkout; release checks out with full history
- semantic-release owns the full release lifecycle (version, tag, changelog, assets, commit-back)
- No manual version management needed — commit messages drive everything

## Conventions

- Commits: conventional commits (`feat:`, `fix:`, `test:`, `ci:`, `refactor:`, `docs:`, `chore:`)
- Mise tasks: noun:verb (`test:run`, `coverage:run`), single-word for simple tasks (`setup`, `build`)
- Slash commands: noun-verb matching mise tasks (`/test-run`, `/coverage-run`)
- Terraform: pinned to 1.14, fixtures use `required_version = ">= 1.14"`

## Architecture Decisions

### Go Rewrite

k9s-style interactive TUI for Terraform. Plugin architecture. Single binary (no jq dependency).

- All features are plugins under `plugins/`
- Shared logic in `internal/terraform/`
- Config: `tfui.yaml` with `plugins:` map
- Coverage must be 100% excluding `cmd/` glue layer

### Fuzzy Search (State Plugin)

Uses `github.com/junegunn/fzf/src/algo.FuzzyMatchV2` directly on `r.Address`. No wrappers, no helpers, no custom logic.

- Match against the full address — what the user sees is what they search
- Pattern lowercased before matching (`caseSensitive=false` requires this per fzf docs)
- Space-separated terms = AND logic (each term scored independently)
- Results sorted by fzf score descending, threshold: `score > 0`
- DO NOT add buildSearchTexts, stripSeparators, segmentMatch, or any string manipulation
- DO NOT add score thresholds beyond `> 0`
- If results seem "too many" — that's fine, best match is at the top

### UX Model (k9s-inspired)

- **`:` command mode**: type plugin name to switch views. Tab autocomplete, prefix matching.
- **`/` filter mode** in state browser: auto-focused on entry. `esc` exits. Typing = filter, arrows = navigate, enter = inspect.
- **`ctrl+w` wrap toggle**: global across all modes. `w` also works in normal/detail mode.
- **`←→` pan**: works in list and detail views. Capped at content width.
- `esc` exits current stack level (filter → normal → home).
- `q` exits to home (app level). `:` opens command input (app level).

### Detail View

- Enter = instant inspect (state cached from initial load).
- Arrows scroll vertically, ←→ pan horizontally (10 chars/press).
- `w` toggles wrap. Fixed header stays visible.
- Scroll indicator `[n/total]` when content overflows.

### State Caching

- `TerraformService.stateCache` holds parsed state from `StateList()`.
- `Show(address)` reuses cache instead of re-pulling (~35s saved).
- Cache invalidated on `StateList()` call (refresh).

### Context Auto-Discovery

- No `context.paths` in `tfui.yaml` → recursively walks subdirs for terraform projects.
- Skips hidden dirs (`.terraform`), stops descending when dir has `.tf` files.
- 2+ projects found → offered as selectable contexts.

## Important Notes

- Never break the public API signature without a major version bump
- All internal functions/vars use `_tfui_` or `_TFUI_` prefix to avoid collisions when sourced
- fd3 is the UI channel; render functions write there, never to stdout/stderr
