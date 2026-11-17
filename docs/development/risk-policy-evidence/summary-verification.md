---
layout: default
title: Risk summary synchronization verification
parent: Development
description: "Records US-009b summary synchronization, documentation checks and delivery review boundaries."
audience: "Coordinating contributors verifying the risk implementation handoff"
---

# Risk summary synchronization verification

US-009b synchronizes the CLI skill with the interfaces published by US-009a at
`e099c89`. It changes reference prose only.
Product verification remains recorded in [session integration](session-integration.md) and
[preservation gates](preservation-gates.md); documentation publication alone is not evidence of behavior.

## Summary review

The test-writer reviewed summaries before edits and identified three stale CLI statements: the formerly
ignored `level` setting, the missing Risk-only exit result, and the blanket Terraform-error description.
The CLI skill now uses `threshold`, distinguishes policy diagnostics, and points to the canonical
[Risk I/O contract](../../reference/cli-io-contract.md#risk-command-v1) and
[policy contract](../../reference/risk-policy-contract.md#command-threshold-and-report-v1).
Source-selection guidance links the dedicated selector and preserves Member plugin-block rejection.
No new policy interface or behavior is introduced.

The architecture, conventions and TUI summaries need no risk-specific correction. AGENTS.md's named
commands, paths and rules did not change, so AGENTS.md remains unchanged. No demo media was refreshed.
No product test was added: this task changes no behavior, and the test-writer found no behavioral
regression requiring a failing test.

## Verification applicability

To verify reuse of the US-008b code evidence, run:

```sh
git diff 7a88d0e..e099c89 --name-only
```

The intervening US-009a commit changes only five documentation pages. The current US-009b diff changes
only summaries, evidence and the local progress handoff. The US-008b code/UI results therefore still
apply to the unchanged implementation; they are historical results, not fresh executions in US-009b.
Their exact commands, outcomes and limits remain on the preservation evidence page.

For this prose-only task, each code gate is **not applicable / not run**:

- Quick `mise run check:build`: no compilation input changed.
- Quick affected-package `mise run test:unit`: no Go package changed.
- Done `mise run check:lint`: no executable code changed.
- Done `mise run test:unit`: no product behavior or tests changed.
- Done `mise run check:build`: no compilation input changed.
- `mise run test:coverage`: no production branch changed.
- `mise run test:macro`: no UI implementation changed; no fresh macro binary was built.

Documentation validation is run from the worktree root with `mise run docs:build`, followed by
`mise run docs:check`. Diff whitespace is checked with `git diff --check`.

Observed US-009b results, each exit zero:

| Command | Result |
| --- | --- |
| `mise run docs:build` | Site and search index built. |
| `mise run docs:check` | 84 files and 171 internal links checked after the build. |
| `git diff --check` | No whitespace errors. |

Local `.ralph-tui/docs-build-27.log` and `.ralph-tui/docs-check-27.log` retain output. The commands above
reproduce validation without those ignored logs.

## Delivery boundary

Ralph records the local task commit and Bead completion. The coordinator must still review the diff,
individual checks, task state, commits and runner logs together under
[Agentic development](../agentic.md#review-each-delivery) before closing implementation tracking.
This worker performs no tracking update, commit, push or merge. The PRD checkboxes remain a coordinator
reconciliation task against the completed deliveries; unchecked historical experiments do not override
the linked evidence or justify claiming unverified behavior.
