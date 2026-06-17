## Overview

terraform-ui (tfui) is a k9s-style interactive TUI for Terraform operations. Single Go binary, plugin architecture, BubbleTea framework.

## Quick Reference

```bash
mise run dev              # Run TUI (--project, --plan, --state)
mise run fmt              # Format Go files (accepts file args)
mise run setup            # Bootstrap dependencies
mise run check:lint       # Static analysis (style, correctness, complexity)
mise run check:build      # Compile check (no artifacts)
mise run test:unit        # Unit tests (accepts package arg)
mise run test:coverage    # Coverage (100% threshold)
mise run test:macro       # Macro tapes against binary
mise run test:integration # Integration tests
mise run build            # Cross-platform binaries (goreleaser)
mise run docs:install     # Install Jekyll gems
mise run docs:serve       # Serve docs locally
mise run docs:build       # Build docs for production
mise run demo:generate    # Record demo GIFs
mise run demo:lock        # Lock Python deps
mise run release:run      # Semantic-release (CI only)
```

## Architecture

```
cmd/tfui/     — CLI entry point (cobra, plugin registration, normalizeArgs)
pkg/sdk/      — Public SDK: Plugin, Service, types, UI primitives, frames
internal/     — App internals (config, terraform, source, macro, ui, editor, ai, plugin, logging)
plugins/      — All features as plugins (context, chdir, state, plan, apply, workspace, console, output, validate, risk, phantom, blastradius, forceunlock, version, taint, untaint, import, init)
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

IMPORTANT: "Context" is ONLY the umbrella concept. Code uses "chdir" for member directory selection (never "scope" in new code). Config key: `member "path" {}` (top-level blocks). Event: `ContextChangedEvent` (single event covers chdir + workspace + pin changes — see ADR-0018).

Full domain glossary with relationships and avoid-lists: see `CONTEXT.md`

## Conventions

### Commits
Conventional commits: `feat:`, `fix:`, `test:`, `ci:`, `refactor:`, `docs:`, `chore:`

### Go Package Layout (ADR-0021)

See `docs/adr/0021-plugins-as-use-cases.md` for the full hexagonal architecture rationale. The import rule table:

| Package | Import rule |
|---------|-------------|
| `pkg/sdk` | Plugins import ONLY this |
| `internal/*` | Not importable by plugins |
| `plugins/*` | Import `pkg/sdk` only |
| `cmd/tfui` | Imports everything |

### Naming

- Plugin IDs: lowercase single word (`"state"`, `"plan"`)
- Messages: `{Subject}{Verb}Msg` (e.g., `StateListMsg`)
- Events: `{Subject}{Verb}Event` (e.g., `ContextChangedEvent`)
- Plugin imports use `tfui` prefix: `tfuistate`, `tfuiplan`, `tfuiapply`
- BubbleTea aliased as `tea`

### Testing

- **TDD workflow**: write a failing test first, then implement
- 100% coverage gate on all packages excluding `cmd/` and `internal/terraform/exec` (I/O boundary)
- Table-driven tests preferred
- Use `sdktest.MockService` from `pkg/sdk/sdktest/` — never duplicate mock boilerplate per package
- Use `sdktest.NewDeps(svc)` harness for plugin tests — never construct `PluginDeps` manually or skip `Init()`
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
mise run check:lint && mise run test:unit && mise run check:build
```

For UI changes, also run `mise run test:macro` to verify rendering.

## Deep Dive (read on demand)

- Architecture details: see `.claude/rules/architecture.md` (loaded automatically for `pkg/sdk/` and `internal/` edits)
- TUI UX rules: see `.claude/rules/ux-tui.md` (loaded automatically for `plugins/` and `internal/ui/` edits)
- CLI UX rules: see `.claude/rules/ux-cli.md` (loaded automatically for `cmd/` edits)
- CI/CD pipeline: see `.claude/rules/ci.md` (loaded automatically for `.github/` edits)
- TUI UX spec: `docs/reference/tui-ux.md`
- CLI UX spec: `docs/reference/cli-ux.md`
- Full I/O contract: `docs/reference/cli-io-contract.md`
- CLI reference: `docs/reference/cli-reference.md`
- Architecture overview: `docs/development/architecture.md`
- Macro DSL reference: `docs/reference/macro-language.md`
- Demo pipeline: `demo/README.md`
- Configuration reference: `docs/guides/configuration.md`
- Testing strategy: `docs/development/testing.md`
- Plugin catalog: `docs/plugins/index.md`
- Risk analysis methodology: `docs/features/risk-analysis.md`
- Phantom change detection: `docs/features/phantom-changes.md`
- Blast radius visualization: `docs/features/blast-radius.md`
- Getting started: `docs/guides/getting-started.md`
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
- `mise run fmt` on every `.go` file edit
- Agent checks validate UI consistency, plugin boundaries, CLI contracts, and SDK changes
- Stop hook runs build verification before session ends

## No Speculative Code

CRITICAL: tfui is a UI layer, not a terraform validator. **Never invent behavior that doesn't exist yet.** Every line must serve a real, current use case — not a hypothetical future one.

**What speculative code looks like (DO NOT DO):**

- Validating flag combinations terraform handles itself (e.g., rejecting `--plan` + `--target` — terraform silently ignores targets with a plan file, so should we)
- Defensive nil guards/returns for states that can't happen in production (e.g., `if pinFn == nil` when Init always sets it, or `PinnedAddresses` returning nil instead of empty slice)
- Adding flags nobody asked for "just in case" (e.g., `--outputs`, `--validate-result`, `--workspaces` — speculative cache seeds with no user scenario)
- Mutual exclusivity enforcement at our layer when terraform is the authority (e.g., `if planFile != "" && targets != nil { error }`)
- Building abstractions for hypothetical future requirements (e.g., wrapping a single implementation in an interface "for testability" when the test already works)
- Adding "compat shims" or dead fields to ease a migration that already happened
- Creating error paths for invalid states that the type system or call graph makes unreachable

**The rule:** pass through to terraform, warn when helpful (stderr), never block. If terraform rejects it, the user sees terraform's error — that's fine. Don't duplicate terraform's validation.

**Test corollary:** tests must exercise real behavior through the harness, not bypass Init to test impossible states.

## Learnings

When encountering undocumented patterns or decisions that caused rework, suggest additions to this section.

- 2025-05: Terraform does NOT support `-target` with a saved plan file. In the TUI pipeline, apply consumes only a plan file (ADR-0019). The standalone CLI path passes `-target` directly.
- 2025-05: Apply plugin is NOT on the home menu — only reachable via plan's `a` key. Confirmation is owned by apply (single confirm), not plan.
- 2025-05: `returnTo` is set both by `NavPush` metadata AND workflow transitions (plan→apply). All esc/cancel paths in sub-state plugins must emit `DeactivateMsg`.
- 2026-06: All y/n confirmations share one key set — confirm = `y`/`Y`/`Enter`, cancel = `n`/`N`/`Esc`. Honored uniformly by `ConfirmFrame`, `InputConfirm` (app's `InputRequestBool` handler), and the hand-rolled apply prompt. Keep them in sync when adding new confirm handlers.
- 2026-06: Confirmation is a TUI-only gate (`ExecService.Apply` ignores `AutoApprove` — terraform-exec always applies non-interactively). So non-interactive apply (`-ci`/piped/no TTY) without `-auto-approve` must mirror terraform — reproduce its "Apply not allowed for non-interactive use" error at the CLI boundary, not hang or silently apply. terraform-exec hides terraform's own non-interactive guard by always injecting `-auto-approve`, so we restate it.
