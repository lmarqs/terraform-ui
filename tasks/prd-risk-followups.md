# PRD: Close the open backlog (remaining stories)

## Overview

This PRD started from a triage of 36 open issues against `main` at `c9a1ae7` on 2026-11-19. Most of its stories have shipped. This file keeps only the stories still open, checked against `main` at `fe36d16` on 2026-11-21. The story IDs keep their original numbers.

| Group | Issues | Stories |
|---|---|---|
| Policy architecture record | #107 | US-004 (last step) |
| Analysis screens receive a plan | #76 | US-013, US-027 |
| Small roadmap items | #32 | US-026 |

## Goals

- Close #76 and #32, each by a merged PR.
- Bring #107's "Decisions so far" up to date, and close #107 only if its own rule allows.
- No fix weakens a check or adds speculative code.

## Quality Gates

These commands must pass for every user story that changes code, tests, tapes or scripts:

- `mise run check:lint`: lint, 0 issues
- `mise run test:unit`: unit tests
- `mise run check:build`: compile
- `mise run test:coverage`: 100% outside `cmd/` and `internal/terraform/exec`
- `flock /tmp/claude-1000/tfui-integration.lock env GOFLAGS=-count=1 mise run test:integration`: integration tests, uncached and serialized

US-013 and US-026 touch rendered screens, so they also need:

- `mise run build && mise run test:macro`: smoke macro tapes against the built binary

US-004 only edits an issue and runs no gates.

## Process rules for every story

- Re-check each story's evidence against the current `main` before writing code. If it no longer holds, stop and report instead of forcing the story.
- Write the failing test first, and see it fail for the stated reason before writing the fix.
- Each PR gets its own worktree from `origin/main`, at `.worktrees/<branch>` (see `.worktrees/README.md`).
- Use conventional commits, ending with the co-author line for the model doing the work. Commit and push at each meaningful step.
- Each PR body ends with `🤖 Generated with [Claude Code](https://claude.com/claude-code)` and carries the issue line from the PR map.
- Once a PR is open and ready, run a Codex adversarial review against `origin/main`. Validate every claim it makes, and fix the real ones before handing over.
- Stop and ask before adding an exported SDK API or another new abstraction that a story does not name.
- Never force-push, use bare `git stash`, merge a PR, delete a branch, worktree or scratch file, or weaken a check.

## PR map

| PR | Branch | Stories | Issue line in the PR body |
|----|--------|---------|---------------------------|
| H | `fix/blastradius-cold-open-main` | US-013 | `Part of #76` |
| U | `docs/regenerate-analysis-gifs` | US-027 | `Closes #76` |
| T | `feat/root-target-pins` | US-026 | `Closes #32` |

**Order and dependencies**

1. US-004 and T are independent of everything else.
2. U starts after H merges, because the Blast Radius GIF needs the cold-open fix to record data.

## User Stories

### US-004: Bring #107's decision record up to date
**Issue:** #107

**Description:** As the owner, I want #107's "Decisions so far" to list every decision already made, so that the wayfinder map shows what is decided.

**Evidence:** #118, the seven policy tickets (#109, #110, #112, #113, #114, #115, #117) and #106 are closed. The design is on `main` as `docs/development/risk-policy-design.md`, and ADR-0026 was folded into ADR-0024 and ADR-0025 (`d85a92c1`). #107's "Decisions so far" still lists only #111 and #108, and #107 has no comments. #116 is still open.

**Acceptance Criteria:**
- [ ] #107's "Decisions so far" gets one linked entry per closed ticket above, in the form the body prescribes, each with a one-line gist.
- [ ] #107 is closed only if its own closing rule allows it with #116 open. Otherwise it stays open, with a comment saying #116 is the one remaining child.

### US-013: Blast Radius opens with data when no plan has run
**Issue:** #76 (Blast Radius part)

**Description:** As a TUI user, I want Blast Radius to request its assessment when I open it cold, so that I see results instead of the empty state.

**Evidence:** On `main`, `Activate` in `plugins/blastradius/assessment.go:39-42` returns nil when `e.summary == nil`. The fix was written and reviewed in PR #171, but #171 merged into `fix/blastradius-esc-done` after that branch had already merged, so it never reached `main`. Its 4 commits (`7283f959`, `714a5cdf`, `879b84a7` and merge `bf3f4d80`) are only on `origin/fix/blastradius-esc-done`. They add `plugins/blastradius/cold_open_test.go`, `tests/fixtures/tapes/macro/blastradius_render.tape` and its wiring in `tests/integration/macro_test.go`.

**Acceptance Criteria:**
- [ ] The three non-merge commits are cherry-picked onto a branch from `origin/main`, with conflicts resolved against the current code.
- [ ] `cold_open_test.go` fails on `main` without the fix. The PR body shows the failure.
- [ ] `Activate` requests an assessment with or without a summary. `HandleRiskAssessed` stores the summary the event carries.
- [ ] `tests/fixtures/tapes/macro/blastradius_render.tape` presses `B` with no plan run first and asserts a resource address from the fixture, and it runs in `tests/integration/macro_test.go`.
- [ ] `tests/fixtures/tapes/smoke/risk_consumers.tape` passes unchanged.
- [ ] `docs/plugins/blastradius.md` describes the cold open.

### US-027: The analysis demo GIFs show real data
**Issue:** #76 (closes the issue)

**Description:** As a reader of the docs, I want the Risk, Phantom and Blast Radius GIFs to show each screen with data, so that the docs show what the feature does.

**Evidence:** `docs/assets/demo/risk-analysis.gif`, `phantom.gif` and `blastradius.gif` were last generated on 2026-11-06 and show the 3-frame empty state. Risk and Phantom now fill when opened cold (#121, #131, #158). Blast Radius does after US-013.

**Acceptance Criteria:**
- [ ] `./demo/generate.sh <tfui>` exits 0 on `main` after US-013 merges.
- [ ] The three GIFs are regenerated, and each shows at least one resource row. The PR body names the frame that shows it.
- [ ] Only these three GIFs change. Any tape change needed to record data is listed in the PR body with the reason.

### US-026: -target on the root command pre-pins resources
**Issue:** #32

**Description:** As a TUI user, I want `tfui -target <address>` to start with those resources pinned, so that I do not pin them by hand each session.

**Evidence:** `-target` exists only on the plan and apply subcommands (`cmd/tfui/plan_command.go`, `apply_command.go`), not on root (`cmd/tfui/root.go` has no `target` flag). `startupContext` in `internal/ui/app.go` starts with no pins. `WithPins` and `TogglePins` exist.

**Acceptance Criteria:**
- [ ] The issue is moved to Todo before the work starts.
- [ ] Tests fail before the change: the root command accepts repeated `-target`, and the startup Context carries those addresses as pins.
- [ ] Pins stay pins. They are not sent as terraform targets unless the user acts on them, which follows the pins-versus-targets rule in `CONTEXT.md` and ADR-0019.
- [ ] A macro tape starts with `-target` and asserts the pinned marker on that row.
- [ ] Every acceptance item in #32 is checked off in the PR body with its test.
- [ ] `docs/reference/cli-reference.md` documents the root flag.

## Functional Requirements

- FR-8: Risk, Blast Radius and Phantom must each show results when opened with no plan run in the session.
- FR-17: `tfui -target <address>` must start the TUI with that address pinned.

## Non-Goals

- **Large roadmap features:** #24, #25, #26, #27, #28, #30, #31.
- **#33 (yank to filter):** its own open questions need a design first.
- **Docs rewrites:** #72, #73, #74.
- **#116:** it stays open until the owner has reviewed the measured costs.
- **#145, the host cancel on `PlanCompletedEvent`:** no path reaches it, and a stale result is already ignored.
- A `tfui phantom` command or per-change engine reasons.

## Technical Considerations

- Plugins import only `pkg/sdk`.
- The macro driver runs Cmds synchronously, so a tape cannot observe a loading state. Loading behavior is tested at unit level.
- `cmd/` is outside the coverage threshold. US-026 still needs the tests its criteria name; its `internal/ui` change counts toward the 100%.

## Success Metrics

- #76 and #32 are closed.
- #107's "Decisions so far" lists every closed child, and #107 is closed if its rule allows.

## Open Questions

- US-004: whether #107 can close while #116 is open depends on #107's own closing rule, which the story checks.
