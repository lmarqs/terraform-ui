---
name: tfui-conventions
description: "tfui coding conventions: package layout and import rules, domain terms, naming of plugins, messages and events, testing rules, plugin implementation rules, the No Speculative Code rule with examples, and the Claude Code hooks that run on edits. Use before writing, reviewing or planning Go code in cmd/, pkg/, internal/ or plugins/."
---

# tfui conventions

## Package layout

```
cmd/tfui/     — CLI entry point (cobra, plugin registration, normalizeArgs)
pkg/sdk/      — Public SDK: Plugin, Service, types, UI primitives, frames
internal/     — App internals (config, terraform, source, macro, ui, editor, ai, plugin, logging)
plugins/      — All features as plugins (context, chdir, state, plan, apply, workspace, console, output, validate, risk, phantom, blastradius, forceunlock, version, taint, untaint, import, init)
tests/        — Integration tests + fixtures
demo/         — Demo pipeline: fixtures, macro tapes, GIF generation scripts
```

Import rules (ADR-0021, `docs/adr/0021-plugins-as-use-cases.md` gives the hexagonal rationale):

| Package | Import rule |
|---------|-------------|
| `pkg/sdk` | Plugins import ONLY this |
| `internal/*` | Not importable by plugins |
| `plugins/*` | Import `pkg/sdk` only |
| `cmd/tfui` | Imports everything |

## Terminology

| Term | Definition |
|------|-----------|
| **Project** | Root directory where `tfui.hcl` lives |
| **Chdir** | Selected member directory within a project |
| **Workspace** | Terraform workspace within a chdir |
| **Context** | Umbrella concept: Project + Chdir + Workspace combined |

"Context" is ONLY the umbrella concept. Code uses "chdir" for member directory selection (never "scope" in new
code). Config key: `member "path" {}` (top-level blocks). Event: `ContextChangedEvent` (single event covers chdir +
workspace + pin changes — see ADR-0018). `CONTEXT.md` holds the full glossary with relationships and avoid-lists.

## Naming

- Plugin IDs: lowercase single word (`"state"`, `"plan"`)
- Messages: `{Subject}{Verb}Msg` (e.g., `StateListMsg`)
- Events: `{Subject}{Verb}Event` (e.g., `ContextChangedEvent`)
- Plugin imports use `tfui` prefix: `tfuistate`, `tfuiplan`, `tfuiapply`
- BubbleTea aliased as `tea`

## Testing

- TDD: the `test-writer` agent writes a failing test before any implementation code.
- 100% coverage gate on all packages excluding `cmd/` and `internal/terraform/exec` (I/O boundary)
- Table-driven tests preferred
- Use `sdktest.MockService` from `pkg/sdk/sdktest/` — never duplicate mock boilerplate per package
- Use `sdktest.NewDeps(svc)` harness for plugin tests — never construct `PluginDeps` manually or skip `Init()`
- Integration tests in `tests/integration/`

## Plugin rules

- Inter-plugin communication uses typed events — no stringly-typed state sharing
- Destructive ops require staleness check + user confirmation
- AI features check `ctx.AI != nil` before offering
- Config getters ALWAYS take a default value — no nil panics
- Editor integration uses `tea.ExecProcess` for proper terminal handoff

## Key dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `github.com/hashicorp/terraform-exec` | Terraform CLI wrapper |
| `github.com/hashicorp/terraform-json` | Terraform JSON types |
| `github.com/junegunn/fzf` | Fuzzy search algorithm |
| `github.com/spf13/cobra` | CLI framework |

## No Speculative Code

tfui is a UI layer, not a terraform validator. **Never invent behavior that doesn't exist yet.** Every line must
serve a real, current use case — not a hypothetical future one.

**What speculative code looks like (DO NOT DO):**

- Validating flag combinations terraform handles itself (e.g., rejecting `--plan` + `--target` — terraform silently ignores targets with a plan file, so should we)
- Defensive nil guards/returns for states that can't happen in production (e.g., `if pinFn == nil` when Init always sets it, or `PinnedAddresses` returning nil instead of empty slice)
- Adding flags nobody asked for "just in case" (e.g., `--outputs`, `--validate-result`, `--workspaces` — speculative cache seeds with no user scenario)
- Mutual exclusivity enforcement at our layer when terraform is the authority (e.g., `if planFile != "" && targets != nil { error }`)
- Building abstractions for hypothetical future requirements (e.g., wrapping a single implementation in an interface "for testability" when the test already works)
- Adding "compat shims" or dead fields to ease a migration that already happened
- Creating error paths for invalid states that the type system or call graph makes unreachable

**The rule:** pass through to terraform, warn when helpful (stderr), never block. If terraform rejects it, the user
sees terraform's error — that's fine. Don't duplicate terraform's validation.

**Test corollary:** tests must exercise real behavior through the harness, not bypass Init to test impossible states.

## Claude Code hooks

`PostToolUse` hooks in `.claude/settings.json` run on every edit:

- `mise run fmt` on every `.go` file
- Agent checks on `internal/ui/`, `plugins/`, `cmd/tfui/` and `pkg/sdk/` edits: UI consistency, plugin boundaries,
  CLI contracts and SDK changes
