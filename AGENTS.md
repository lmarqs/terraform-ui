# terraform-ui (tfui)

A k9s-style interactive TUI and headless CLI for Terraform operations: one Go binary where every feature is a
plugin. Users and features are in the [README](README.md).

Stack: Go 1.25 with BubbleTea, Lipgloss and Cobra (`go.mod`); golangci-lint 2.12, goreleaser 2.15 and terraform
1.14 (`mise.toml`).

## Quick reference

- Install or add a dependency: `mise install && mise run setup`; `go get <module>@<version>`
- Run: `mise run dev` (accepts `--project`, `--plan`, `--state`)
- Build: `mise run check:build` to compile; `mise run build` for the cross-platform binaries (goreleaser)
- Test one file: `mise run test:unit ./<package>/...` (Go tests run per package)
- Lint or format one file: `mise run fmt <file.go>`; lint runs repo-wide, `mise run check:lint`
- Generate code: none
- Local services: none
<!-- cook:begin io.github.lmarqs.cookbook:git#worktrees/worktrees -->
<!-- io.github.lmarqs.cookbook:git#worktrees --instructions=AGENTS.md -->
- Worktrees: branch `<branch>` is checked out at `.worktrees/<branch>`; `.worktrees/README.md` says how to
  create one ready to run and remove it.
<!-- cook:end io.github.lmarqs.cookbook:git#worktrees/worktrees -->
- Quick check, seconds on what changed: `mise run check:build && mise run test:unit ./<package>/...`
- Done check, all that must pass before a change is done: `mise run check:lint && mise run test:unit && mise run check:build`,
  about 50 seconds, no services; for UI changes also `mise run test:macro`.
- Every other task (coverage, integration, docs site, demo GIFs): `mise tasks` lists each with when to run it.

## Boundaries

**Always**

- Run the Quick check before moving on. A change is done when the Done check passes.
- Report each check's result, or that it did not run.
- Update this file in the change that alters a command, path or rule it names.
- Keep this file under 800 words; past 1,200, move a section into a skill or memory file before adding. All
  `AGENTS.md` files together stay under 16 KiB.
- Write the failing test first: the `test-writer` agent produces it before any implementation code.

**Never**

- Commit secrets or credentials, or put personal data in logs, fixtures or commits.
- Edit generated files by hand: `CHANGELOG.md` and `VERSION`, which semantic-release writes.
- Make a check pass by weakening it: skipping a test, suppressing a warning, loosening an assertion.

**Stop and ask** before a change spanning 3+ files or adding a new abstraction (share a brief plan first), and when
a check fails again after a fix, a command is refused, or a decision is left open.

Enforced by none.

## Where things go

- New code: `cmd/tfui/` CLI wiring, `pkg/sdk/` the public SDK, `internal/` app internals, `plugins/<id>/` each
  feature; the `tfui-conventions` skill has the import rules, and `docs/development/architecture.md` the design.
- Tests: `_test.go` beside the code, table-driven; a plugin test builds its deps with `sdktest.NewDeps(svc)` and
  calls `Init()`, as `plugins/version/version_test.go` does; integration tests in `tests/integration/`.
- Settings: `tfui.hcl` per project ([configuration](docs/guides/configuration.md)).
- Decisions and domain rules: `docs/adr/` and the glossary in `CONTEXT.md`.
- A procedure an agent repeats: a skill in `.agents/skills/`.
- A trap in one area: `.agents/memory/`, listed under Deep dive.

## Rules

- A behavior change ships with a test; a bug fix with one that failed before the fix.
- Comments record only reasons the code cannot show.
- Change only what the task needs, matching the surrounding code.
- Plugins import only `pkg/sdk`; `internal/` belongs to the app.
- tfui is a pass-through layer: every line serves a current use case, terraform validates its own inputs, tfui warns
  on stderr when helpful, and terraform's error reaches the user as is. The `tfui-conventions` skill lists what
  speculative code looks like.
- Commits: conventional commits (`feat:`, `fix:`, `test:`, `ci:`, `refactor:`, `docs:`, `chore:`); break large
  changes into reviewable chunks; work is tracked in GitHub issues as `docs/development/backlog.md` says.
- Deploy: a push to `main` runs semantic-release, which tags, writes `CHANGELOG.md` and runs goreleaser; the
  `tfui-ci` skill has the pipeline.

## Gotchas

- "Context" is only the umbrella of Project + Chdir + Workspace; code says "chdir" for the member directory, never
  "scope", and `ContextChangedEvent` covers chdir, workspace and pin changes (ADR-0018).
- Terraform rejects `-target` with a saved plan file: targets go on plan, never on apply (ADR-0019).
- CI runs `mise run test:coverage` at a 100% threshold: an untested branch outside `cmd/` and
  `internal/terraform/exec` turns it red.

## Deep dive (read on demand)

The `tfui-*` skills hold the conventions, the SDK architecture, the TUI and CLI UX rules and the CI pipeline, each
with its spec in `docs/`. Beyond them:

- Choosing what to test, and at which level: `docs/development/testing.md`
- Filing, labelling or moving an issue: `docs/development/backlog.md`
- A plugin's behavior: `docs/plugins/index.md`; risk, phantom changes, blast radius: `docs/features/`
- Demo fixtures, tapes or GIFs: `demo/README.md`
