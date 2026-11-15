---
layout: default
title: Risk integration preservation and gate verification
parent: Development
description: "Records reproducible Apply and Terraform JSON preservation checks and the US-008b integration gates."
audience: "Contributors who know the risk contract and need to reproduce the final integration checks"
---

# Risk integration preservation and gate verification

The US-008b preservation checks and required code/UI gates passed against
`e4c381d5112cb068f4230212acf60c2246ba3bf3` on branch `ralph/risk-plugin`.
This verification adds no production code, tests, fixtures or Apply changes. It complements
[session integration verification](session-integration.md); US-009's guide and reference publication
remains separate.

## Preservation evidence

The test-writer reviewed and ran the existing regressions before any implementation edit.
They passed, so no behavior fix or artificial failing test was introduced. The full unit gate
also passed those regressions. In particular:

- `cmd/tfui/session_plan_test.go` checks exact Terraform show JSON bytes, including surrounding
  whitespace, through `Session.RunPlugin`. No risk field is inserted into that output.
- `plugins/plan/source_test.go` checks that JSON mode skips failed policy loading and preserves
  the original bytes, while actual Terraform failures retain their Terraform error.
- `cmd/tfui/apply_command_test.go` checks typed flags, noninteractive confirmation requirements,
  targets passed to Plan, explicit auto-approval, and lock options.
- Plan and Apply plugin tests check saved-plan handoff, temporary artifact cleanup, confirmation,
  stale replanning, cancellation of pending intent, and terminal success/failure behavior.

The following existing integration tests were additionally run uncached with Terraform available:

```sh
mise exec -- go test -count=1 -tags integration -timeout 180s ./tests/integration -run '^TestCompat_(DestroyModeApplyConfirmation|TargetedPlanEmitsTargetFlag|StandaloneApplyWithTarget)$' -v

mise exec -- go test -count=1 -tags integration -timeout 180s ./tests/integration -run '^Test(Apply_(CreateFixture_SilentMode|BrokenFixture_SilentMode|CreateFixture_AgentMode|Targeted_OnlyAppliesTarget|NoChanges_Succeeds|PartialFailure_ShouldReportWhatGotApplied)|Equivalence_Apply_.*|CLI_PlanJSONValid)$' -v
```

All three compatibility tests passed (6.889 seconds). They check destroy confirmation, targeted
saved-plan handoff without targets on Apply, and standalone target forwarding. All eleven tests
in the second command passed (41.693 seconds): six Apply scenarios, four Apply equivalence
scenarios and valid Plan JSON output. The local-provider fixtures check state/file effects,
target isolation, no-change success, failure and partial-failure reporting, and equivalent state
effects across CLI, CI and JSON modes. Macro-backed compatibility checks inspect recorded
commands; they do not execute those commands against infrastructure.

These checks establish preservation within the tested local-provider and command-recording
scenarios. They do not establish compatibility with every Terraform provider or every interactive
terminal. No cloud infrastructure was used. The integration test runner builds its own fresh
`dist/tfui-integration-test` binary and removes it after execution.

## Required gate results

The run used Go 1.25.5 on Linux/amd64 and Terraform 1.14.9. All commands below exited zero.
Run them from the worktree root. Local `.ralph-tui/*-25.log` files retain this run's output;
the commands and existing tests are sufficient to reproduce it without those ignored logs.

| Command | Observed result | Local log |
| --- | --- | --- |
| `mise run check:build` | Quick build passed. | `build-25.log` |
| `mise run test:unit ./cmd/tfui/...` | 356 tests passed. | `quick-cmd-25.log` |
| `mise run check:lint` | Zero issues. | `lint-25.log` |
| `mise run test:unit` | 5,145 tests passed. | `unit-25.log` |
| `mise run test:coverage` | 100.0% under the repository's configured package exclusions. | `coverage-25.log` |
| `mise run check:build` | Done build passed. | `build-final-25.log` |
| `go build -o .ralph-tui/tfui-25 ./cmd/tfui` | Fresh binary built before macro execution. | `binary-25.log` |
| `mise run test:macro .ralph-tui/tfui-25` | All 22 smoke tapes selected by the existing task passed. | `macro-25.log` |
| `mise run docs:build` | Static site and search index built. | `docs-build-25.log` |
| `mise run docs:check` | 83 files and 166 internal links checked successfully after the build. | `docs-check-25.log` |
| `git diff --check` | Passed. | Terminal output. |

The smoke task's existing `assertion_fail.tape` and `state_tree.tape` exclusions were unchanged.
The fresh binary SHA-256 was
`91584592b35b740789210d18b776946913ccefc926a14d680c5c42683f800a0e`.
The fast `check:build` command was not used as the macro binary producer.

This bead changes only evidence and the run handoff. No production branch was introduced and no
check was weakened. US-009 can publish the implemented interfaces using these results and the
session evidence, while retaining the limits stated above.
