# AGENTS.md

## Overview

terraform-ui (tfui) is a k9s-style interactive TUI for Terraform operations. Single Go binary, plugin architecture, BubbleTea framework.

## Quick Reference

```bash
mise run dev              # Run TUI in development mode
mise run fmt              # Format (gofmt)
mise run check:lint       # Lint (golangci-lint v2)
mise run check:vet        # Quick go vet
mise run test:unit        # Unit tests
mise run test:coverage    # Coverage (100% threshold)
mise run build            # Cross-platform binaries
mise run test:macro       # Macro tapes against binary
mise run 'test:integration:*'  # Integration tests (terraform/tofu/terragrunt)
mise run demo:generate    # Record demo GIFs from macro tapes
mise run demo:lock        # Lock Python deps from pyproject.toml
```

## Architecture

```
cmd/tfui/     — CLI entry point (cobra, plugin registration, normalizeArgs)
pkg/sdk/      — Public SDK: Plugin, Service, types, UI primitives, frames
internal/     — App internals (config, terraform, source, macro, ui, editor, ai, plugin, logging)
plugins/      — All features as plugins (context, chdir, state, plan, apply, workspace, console, output, validate, risk, phantom, blastradius, forceunlock, version)
tests/        — Integration tests + fixtures
demo/         — Demo pipeline: fixtures, macro tapes, GIF generation scripts
```

## Terminology

| Term | Definition |
|------|-----------|
| **Project** | Root directory where `tfui.hcl` lives |
| **Chdir** | Selected member directory within a project |
| **Workspace** | Terraform workspace within a chdir |
| **Context** | Umbrella concept: Project + Chdir + Workspace combined |

IMPORTANT: "Context" is ONLY the umbrella concept. Code uses "chdir" for member directory selection (never "scope" in new code). Config key: `member "path" {}` (top-level blocks). Event: `ChdirChangedEvent`.

Full domain glossary with relationships and avoid-lists: see `CONTEXT.md`

## Conventions

### Commits
Conventional commits: `feat:`, `fix:`, `test:`, `ci:`, `refactor:`, `docs:`, `chore:`

### Go Package Layout

| Package | Import rule |
|---------|-------------|
| `pkg/sdk` | Plugins import ONLY this |
| `internal/*` | Not importable by plugins |
| `plugins/*` | Import `pkg/sdk` only |
| `cmd/tfui` | Imports everything |

### Naming

- Plugin IDs: lowercase single word (`"state"`, `"plan"`)
- Messages: `{Subject}{Verb}Msg` (e.g., `StateListMsg`)
- Events: `{Subject}{Verb}Event` (e.g., `ChdirChangedEvent`)
- Plugin imports use `tfui` prefix: `tfuistate`, `tfuiplan`, `tfuiapply`
- BubbleTea aliased as `tea`

### Testing

- **TDD workflow**: write a failing test first, then implement
- 100% coverage gate on all packages excluding `cmd/` glue
- Table-driven tests preferred
- Mock services implement `sdk.Service` with no-op methods
- Integration tests in `tests/integration/`

### Roadmap
- Items in `docs/_roadmap/` as individual markdown files
- Delete immediately once completed — don't mark "done"

## Workflow

- For changes spanning 3+ files or new abstractions: produce a brief plan before coding
- For single-file changes: proceed directly
- Break large changes into reviewable chunks

## Verification

IMPORTANT: Before considering work complete, run:
```bash
mise run check:vet && mise run check:lint && mise run test:unit && mise run build
```

For UI changes, also run `mise run test:macro` to verify rendering.

## Deep Dive (read on demand)

- Architecture details: see `.claude/rules/architecture.md` (loaded automatically for `pkg/sdk/` and `internal/` edits)
- TUI UX rules: see `.claude/rules/ux-tui.md` (loaded automatically for `plugins/` and `internal/ui/` edits)
- CLI UX rules: see `.claude/rules/ux-cli.md` (loaded automatically for `cmd/` edits)
- CI/CD pipeline: see `.claude/rules/ci.md` (loaded automatically for `.github/` edits)
- TUI UX spec: `docs/tui-ux.md`
- CLI UX spec: `docs/cli-ux.md`
- Full I/O contract: `docs/cli-io-contract.md`
- CLI reference: `docs/cli-reference.md`
- Architecture overview: `docs/architecture.md`
- Macro DSL reference: `docs/macro-language.md`
- Demo pipeline: `demo/README.md`
- Demo landing page: `docs/demo.md`
- Configuration reference: `docs/configuration.md`
- Testing strategy: `docs/testing.md`
- Plugin catalog: `docs/plugins/index.md`
- Risk analysis methodology: `docs/risk-analysis.md`
- Phantom change detection: `docs/phantom-changes.md`
- Blast radius visualization: `docs/blast-radius.md`
- Domain glossary: `CONTEXT.md`
- Architecture Decision Records: `docs/adr/`

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `github.com/hashicorp/terraform-exec` | Terraform CLI wrapper |
| `github.com/hashicorp/terraform-json` | Terraform JSON types |
| `github.com/junegunn/fzf` | Fuzzy search algorithm |
| `github.com/spf13/cobra` | CLI framework |

## Important Rules

CRITICAL: **TDD is non-negotiable** — spawn `test-writer` agent to produce a failing test BEFORE writing implementation code.

CRITICAL: **Plugins import ONLY `pkg/sdk`** — never `internal/`. This is the architectural boundary.

- Inter-plugin communication uses typed events — no stringly-typed state sharing
- Destructive ops require staleness check + user confirmation
- AI features check `ctx.AI != nil` before offering
- Config getters ALWAYS take a default value — no nil panics
- Editor integration uses `tea.ExecProcess` for proper terminal handoff

## Automation

PostToolUse hooks run automatically (configured in `.claude/settings.json`):
- `gofmt` on every `.go` file edit
- Agent checks validate UI consistency, plugin boundaries, CLI contracts, and SDK changes
- Stop hook runs build verification before session ends

## Learnings

When encountering undocumented patterns or decisions that caused rework, suggest additions to this section.

- 2025-05: Terraform does NOT support `-target` with a saved plan file. Apply handles this via replan (see `.claude/rules/architecture.md`).
- 2025-05: Apply plugin is NOT on the home menu — only reachable via plan's `a` key. Confirmation is owned by apply (single confirm), not plan.
- 2025-05: `returnTo` is set both by `NavPush` metadata AND workflow transitions (plan→apply). All esc/cancel paths in sub-state plugins must emit `DeactivateMsg`.
