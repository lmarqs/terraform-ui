# PRD: Close the open backlog

## Overview

The repository has 36 open issues. All were triaged against `main` at `c9a1ae7` on 2026-11-19. This PRD takes on every issue that can be closed with no work, with a recorded decision, or with a small or medium fix. It closes 23 existing issues, and a 24th (#107) if its own rules allow.

The work started as the follow-ups to #141 (issue #142 plus the #141 review notes) and grew to the whole backlog. The file name keeps the original scope.

| Group | Issues closed | Stories |
|---|---|---|
| Already resolved, close with evidence | #61, #106, #109, #110, #112, #113, #114, #115, #117 | US-001 to US-003 |
| Policy architecture record | #118, and #107 if allowed | US-004 |
| Risk and Blast Radius follow-ups | #142, plus two issues this PRD files | US-005 to US-012 |
| Analysis screens receive a plan | #76 | US-013, US-014 |
| Lists through `ui.ContentPanel` | #97 | US-015, US-016 |
| Demo | #77, #87 | US-017, US-018 |
| Apply options | #64 | US-019 |
| Signals | #102, #104, #105 | US-020 to US-022 |
| Standard `ctrl+r` | #90 | US-023 |
| Small roadmap items | #23, #29, #32 | US-024 to US-026 |

Twelve or thirteen existing issues stay open, as Non-Goals lists.

## Goals

- Close 23 existing issues, each by a merged PR or by a comment that cites evidence on `main`.
- Every story names the issue it closes or advances.
- No fix weakens a check, adds speculative code or changes the v1 Risk CLI contract.

## Quality Gates

These commands must pass for every user story that changes code, tests, tapes or scripts:

- `mise run check:lint`: lint, 0 issues
- `mise run test:unit`: unit tests
- `mise run check:build`: compile
- `mise run test:coverage`: 100% outside `cmd/` and `internal/terraform/exec`
- `flock /tmp/claude-1000/tfui-integration.lock env GOFLAGS=-count=1 mise run test:integration`: integration tests, uncached and serialized

Stories that touch `View()`, hints, frames, key handling or rendered text (US-005, US-006, US-008, US-010, US-013 to US-016, US-023, US-024, US-026) also need:

- `mise run build && mise run test:macro`: smoke macro tapes against the built binary

Stories that only comment on or close issues (US-000 to US-003) run no gates.

## Process rules for every story

- The evidence in each story comes from triage at `c9a1ae7`. Re-check it against the current `main` before writing code. If it no longer holds, stop and report instead of forcing the story.
- Write the failing test first, and see it fail for the stated reason before writing the fix.
- Each PR gets its own worktree from `origin/main`, at `.worktrees/<branch>` (see `.worktrees/README.md`).
- Use conventional commits, ending with `Co-Authored-By: Claude Fable 5.2 <noreply@anthropic.com>`. Commit and push at each meaningful step.
- Each PR body ends with `🤖 Generated with [Claude Code](https://claude.com/claude-code)` and carries the issue line from the PR map.
- Once a PR is open and ready, run a Codex adversarial review against `origin/main`. Validate every claim it makes, and fix the real ones before handing over.
- Stop and ask before adding an exported SDK API or another new abstraction that a story does not name.
- A story that closes an issue by comment posts the evidence first, then closes it. The user approved these closes on 2026-11-19.
- Never force-push, use bare `git stash`, merge a PR, delete a branch, worktree or scratch file, or weaken a check.

## PR map

| PR | Branch | Stories | Issue line in the PR body |
|----|--------|---------|---------------------------|
| A | `docs/risk-policy-architecture-record` | US-004 | `Closes #118` |
| B | `fix/blastradius-esc-done` | US-005 | `Closes #<I-1>` |
| C | `fix/content-panel-wrap-cursor` | US-006 | `Part of #142`, `Refs #97` |
| D | `fix/risk-retry-reevaluates` | US-007 | `Part of #142` |
| E | `fix/risk-deposed-identity` | US-008 | `Part of #142` |
| F | `refactor/risk-report-boundary` | US-009, US-010 | `Closes #142` |
| G | `test/risk-followup-hygiene` | US-011, US-012 | `Closes #<I-2>` |
| H | `fix/blastradius-cold-open` | US-013 | `Part of #76` |
| I | `fix/phantom-receives-plan` | US-014 | `Closes #76` |
| J | `fix/analysis-lists-content-panel` | US-015 | `Part of #97` |
| K | `fix/validate-output-content-panel` | US-016 | `Closes #97` |
| L | `fix/demo-generate-fails-loudly` | US-017 | `Closes #77` |
| M | `chore/remove-unused-seeds` | US-018 | `Closes #87` |
| N | `fix/apply-forwards-options` | US-019 | `Closes #64` |
| O | `fix/interrupt-guard-shutdown` | US-020, US-021 | `Closes #102`, `Closes #104` |
| P | `fix/macro-playback-stops-on-signal` | US-022 | `Closes #105` |
| Q | `fix/standard-refresh-key` | US-023 | `Closes #90` |
| R | `feat/plan-filter-by-action` | US-024 | `Closes #23` |
| S | `feat/dry-run-recording` | US-025 | `Closes #29` |
| T | `feat/root-target-pins` | US-026 | `Closes #32` |

`<I-1>`, `<I-2>` and `<I-3>` are the issues US-000 files. Replace the placeholders in this file once they have numbers.

**Order and dependencies**

1. US-000 to US-003 first. They need no code.
2. A, B, E, G, L, M, N, R, S and T are independent.
3. F starts after E merges, because both change how Risk labels instances.
4. J starts after C and I merge. K starts after C merges.
5. P starts after O merges, because both change the interrupt guard.
6. L is checked again after H and I merge, because the demo GIFs for Phantom and Blast Radius depend on them.

A PR that says `Closes` for an issue spanning several PRs (#142, #76, #97) must be the last of its group to be opened. If it is not, it says `Part of`, and the last one closes the issue.

## User Stories

### US-000: File the missing issues
**Issue:** creates `<I-1>`, `<I-2>` and `<I-3>`.

**Description:** As a maintainer, I want every story tied to a GitHub issue, so that each PR closes or advances a known issue.

**Acceptance Criteria:**
- [ ] `<I-1>`: Blast Radius ignores Esc on the result view, with the evidence from US-005.
- [ ] `<I-2>`: Risk test hygiene, covering the weak tape assertion and the 1-second tick waits, with the evidence from US-011 and US-012.
- [ ] `<I-3>`: `HandlePlanCompleted` does not cancel the host risk request. It states that no path reaches it today and proposes the one-line fix: call `a.risk.Advance()` on `sdk.PlanCompletedEvent` in `internal/ui/app.go`, matching `PlanInvalidatedEvent`. No story implements it.
- [ ] Each issue is labelled as `docs/development/backlog.md` says.
- [ ] A comment on #142 links this PRD's story list and notes that the empty diagnostic (US-010) ships with the reporting change.
- [ ] The placeholders in this file are replaced with the issue numbers, and the change is committed.

### US-001: Close #61 as already fixed
**Issue:** #61

**Description:** As a maintainer, I want #61 closed with evidence, so that the backlog does not list a fixed bug.

**Evidence:** #61 expects that "either the flags exist, or they stop being advertised". `docs/reference/cli-reference.md:45-52` now lists only `-json`, `-ci`, `-target`, `-lock` and `-lock-timeout` for plan. `docs/guides/configuration.md:118-124` mentions only `-lock` and `-lock-timeout`.

**Acceptance Criteria:**
- [ ] `grep -rn -- '-var-file\|-destroy\|-refresh-only\|-replace\|-parallelism\|-out' docs/ --include=*.md` finds no line that offers these as `tfui plan` flags. The command output goes in the comment.
- [ ] A comment on #61 cites the two doc locations and the commit checked, then the issue is closed.
- [ ] If the grep finds an advertised flag, fix the doc in a one-file PR that says `Closes #61` instead of closing by comment.

### US-002: Close the seven answered policy tickets
**Issue:** #109, #110, #112, #113, #114, #115, #117

**Description:** As the owner, I want the policy tickets that `main` already answers closed, so that the wayfinder map shows what is decided.

**Evidence:** The v1 contract (`docs/reference/risk-policy-contract.md`), ADR-0024 and ADR-0025 answer each ticket. The contract says its details were chosen under delegated authority. The owner accepted those delegated choices on 2026-11-19.

| Ticket | Answered by |
|---|---|
| #109 input fidelity | Contract §Input v1, §Reload |
| #110 selection and identity across Context | Contract §Source selection, §Loading and snapshot identity, §Reload; `internal/riskassess/assessment_race_test.go` |
| #112 trust and sensitive-data boundary | Contract §Runtime and trust boundary; `internal/riskpolicy/capabilities.go` |
| #113 phantom suppression | ADR-0025: annotations stay advisory |
| #114 completeness and aggregation | ADR-0025; contract §Result v1, §Assessment |
| #115 built-in and custom ownership | ADR-0024, ADR-0025; contract §Helpers, templates and upgrades |
| #117 threshold, reporting, compatibility | Contract §Command, threshold and report v1; `cmd/tfui/risk_command.go` |

**Acceptance Criteria:**
- [ ] For each ticket, read its completion criteria and check each one against the cited sections. Any criterion that is not met is listed in the comment, and that ticket stays open.
- [ ] Each ticket that passes gets a resolution comment in the form #107 prescribes. It names the sections above and states that the owner accepted the delegated choices on 2026-11-19. Then it is closed.
- [ ] #116 gets no comment and stays open.

### US-003: Close #106 as resolved at the current levels
**Issue:** #106

**Description:** As the owner, I want #106 closed without changing severities, so that the built-in policy stays consistent with the contract.

**Evidence:** The built-in policy flags Lambda code-source replacement (`internal/riskpolicy/builtin.rego:91,150-163`, codes `lambda-code-source-change` and `lambda-source-uncertain`; `docs/features/risk-analysis.md:102-116`). The contract's Lambda section keeps all function updates at `high` and declines the `critical` level. The owner chose on 2026-11-19 to keep the current levels.

**Acceptance Criteria:**
- [ ] A comment on #106 cites the policy lines, the docs section and the contract section. It states that the proposed `critical` and `medium` levels were declined for consistency, and that a custom policy can return `critical`.
- [ ] #106 is closed. No code changes.

### US-004: Bring the policy architecture record to main
**Issue:** #118, #107

**Description:** As the owner, I want the design doc and ADR-0026 on `main`, so that merging the PR records my acceptance of the architecture.

**Evidence:** The design doc and ADR-0026 exist only on `origin/research/risk-plugin` (moved there by 5a441c9 and 73a7beb). `main` links to them by blob URL. `docs/adr/` ADR-0024 (line 13) and ADR-0025 (line 17) still say the design is intended and not shipped. #118 requires that "the owner accepts the architecture design and its repository ADR".

**Acceptance Criteria:**
- [ ] ADR-0026 and the design doc are on `main` under the paths `docs/README.md` maps them to, with content taken from `origin/research/risk-plugin` unchanged except for link fixes.
- [ ] Blob-URL links on `main` to those two files point to the in-repo paths.
- [ ] The status lines of ADR-0024 and ADR-0025 state what shipped, naming PRs #131 to #141.
- [ ] `docs/adr/index.md` lists ADR-0026.
- [ ] The research evidence files (`risk-policy-evidence/*`) stay on the research branch. The PR body says so.
- [ ] The PR body states that merging it is the owner's acceptance for #118, and lists each #118 criterion with where it is met.
- [ ] After the merge, #107's "Decisions so far" is updated with the tickets closed by US-002, US-003 and #118. #107 is closed only if its own closing rule allows it with #116 open. Otherwise it stays open, with a comment saying #116 is the one remaining child.

### US-005: Esc leaves Blast Radius from the result view
**Issue:** `<I-1>`

**Description:** As a TUI user, I want Esc to leave Blast Radius after the assessment completes, so that I am not stuck pressing `q`.

**Evidence:** `plugins/blastradius/blastradius.go:119` only lets Esc through while loading or after a failure. `handleKey` has no `esc` case, and the app has no fallback.

**Acceptance Criteria:**
- [ ] A new test, `TestUpdate_GivenEscInDoneState_ShouldRequestDeactivation`, in `plugins/blastradius/retry_test.go`, fails before the fix. It reaches the done state through the existing `runBlastCmd`/`deliverBlast` helpers and asserts the returned cmd yields `sdk.DeactivateMsg`.
- [ ] The Esc guard returns `DeactivateMsg` in every state.
- [ ] The existing loading and error-state Esc tests pass unchanged.
- [ ] The result-view hints include `Esc back` if the `tfui-tui-ux` hint rules require it. The `tui-checker` agent's verdict is quoted in the PR body.
- [ ] `docs/plugins/blastradius.md` says Esc leaves from every state.

### US-006: With wrap on, the selected row stays visible
**Issue:** #142 (wrapped cursor disappears); related to #97

**Description:** As a TUI user, I want the selected mutation to stay on screen when wrap is on, so that `G` and `j` never move the cursor out of view and Enter opens the row I can see.

**Evidence:** Risk's `renderAnalysis` (`plugins/risk/risk.go:234-260`) picks its window in logical rows and passes `rows[offset:last]`. In wrap mode, `ContentPanel.Render` (`pkg/sdk/ui/content_panel.go:115-130`) fills `Height` with wrapped lines from the top of that slice and clips the rest. #142 reproduces it at 32×8 with three mutations and a long first row: `Ctrl+w`, `G`.

**Acceptance Criteria:**
- [ ] In `plugins/risk/navigation_test.go`, a regression test reproduces the #142 case: a 32×8 view, three mutations, a long first row, `ctrl+w`, then `G`. It fails before the fix and asserts two things:
  - `View()` contains the third mutation's address;
  - Enter opens the inspector for that third mutation.
- [ ] In `pkg/sdk/ui/content_panel_test.go`, a test fails before the fix. Wrap is on, each row wraps to 3 or more lines, the cursor is on the last row, and `Height` cannot fit every wrapped row. It asserts the cursor row's first wrapped line is rendered with the selected style.
- [ ] One width-aware computation in `pkg/sdk/ui` decides which rows are drawn, where they are clipped and where the gutter thumb sits. Risk uses it instead of its own row arithmetic.
- [ ] A test asserts that with wrap on and the cursor on the last row, the gutter thumb is at the bottom of the track.
- [ ] With wrap off, `ContentPanel.Render` output is unchanged, and the existing ContentPanel and Risk navigation tests pass unchanged.
- [ ] A check by hand is recorded in the PR body: `mise run dev` with long addresses, a narrow terminal, then `R`, `ctrl+w`, `G`, Enter.

### US-007: Retry reevaluates a failed assessment
**Issue:** #142 (retry can replay a cached failure)

**Description:** As a TUI user, I want `Ctrl+r` after a failed assessment to evaluate again, so that a retry can recover instead of showing the same cached failure.

**Evidence:** `retry()` (`plugins/risk/lifecycle.go:94-100`) issues the same request. `evaluate` (`internal/riskassess/coordinator.go:223-244`) writes every result to the cache, including partial and unavailable ones.

**Acceptance Criteria:**
- [ ] A coordinator test in `internal/riskassess` fails before the fix. Two requests with the same Context, reviewed plan and policy, where the first evaluation returns a partial assessment, must run two evaluations.
- [ ] The coordinator caches only assessments for which `Usable()` is true. A comment on the cache write states that rule.
- [ ] A test asserts that two requests returning a complete assessment still run one evaluation.
- [ ] A test asserts that the retry evaluates against the same reviewed plan and does not run `terraform plan` again when a reviewed plan exists.
- [ ] `tests/fixtures/tapes/macro/risk_retry.tape` passes unchanged.

### US-008: The Risk list and inspector show the deposed identity
**Issue:** #142 (the TUI loses deposed identity)

**Description:** As a TUI user, I want each deposed object labelled with its deposed key, so that two deposed deletes at the same address are not shown as the same row.

**Evidence:** `renderChangeRow` (`plugins/risk/risk.go:300-316`) and `inspectSelected` (`plugins/risk/frames.go:49-64`) print only the address. The CLI output already uses `instanceLabel(address, deposed)` (`plugins/risk/report.go:230`).

**Acceptance Criteria:**
- [ ] A test fails before the fix. Given two deposed deletes at the same address with different deposed keys, `View()` contains both `(deposed <key>)` labels.
- [ ] A test fails before the fix. Opening the inspector on each of those rows shows that row's deposed key in the `Address:` line and in the frame's address.
- [ ] The list and the inspector both use the existing `instanceLabel`. No second formatter is added.
- [ ] Rows that are not deposed render exactly as before.

### US-009: Move CLI reporting out of the assessment lifecycle
**Issue:** #142 (CLI reporting leaks into assessment completion)

**Description:** As a maintainer, I want threshold results and CLI text derived at the report boundary, so that the lifecycle holds only what was assessed and whether the run failed.

**Evidence:** `acceptAssessment` (`plugins/risk/lifecycle.go:113-130`) stores the wire-format threshold string (`"met"`) and formats the stderr line. Text `Stdout` (`plugins/risk/report.go:183-220`) builds the JSON DTO first and reads its optional pointer fields.

**Acceptance Criteria:**
- [ ] The lifecycle outcome holds the assessment and the failure kind. It holds no wire-format threshold string and no preformatted threshold text.
- [ ] A typed threshold result is derived where the report is produced. `Stdout`, `Stderr` and `ExitCode` use it.
- [ ] Text `Stdout` renders from the assessment rows directly. Only the JSON path builds the DTO.
- [ ] The existing Risk CLI output and exit-code tests pass with no change to their expected bytes or codes.
- [ ] No new exported identifier is added to `pkg/sdk`.

### US-010: Risk never renders an empty diagnostic
**Issue:** #142 (ships with the reporting change)

**Description:** As a TUI or CLI user, I want an unusable assessment to always name a reason, so that `risk unavailable:` is never followed by nothing.

**Evidence:** `failureDiagnostic` (`plugins/risk/report.go:65-82`) returns `""` when the assessment has no `ErrorCode` and no row has one. The test `report_test.go:350` pins the empty result.

**Acceptance Criteria:**
- [ ] Replace `TestFailureDiagnostic_GivenNoGlobalCodeAndNoFailedRow_ShouldBeEmpty` with a test asserting the fallback reason: `string(a.Status)` when the status is non-empty, and the fixed text `assessment unusable` when it is empty.
- [ ] A View test and a Stderr test for that assessment each assert the output contains the fallback reason and has no `risk unavailable: ` at the end of a line.
- [ ] Assessments that have a global code or a failed row keep their current text.
- [ ] This story is built on US-009 in the same PR.

### US-011: risk-policy-unavailable.tape proves Risk is on screen
**Issue:** `<I-2>`

**Description:** As a maintainer, I want the tape to fail if Risk is not shown, so that it cannot pass on Plan's copy of the same diagnostic.

**Evidence:** The tape's last assertion, `risk unavailable: source_unreadable`, is a substring that Plan also renders (`plugins/plan/plan.go:705`, `:978`). `assert view` is a plain substring check (`internal/macro/driver.go:77`).

**Acceptance Criteria:**
- [ ] After `key R` / `wait ready`, `tests/fixtures/tapes/macro/risk-policy-unavailable.tape` asserts text that only Risk's error view renders, for example `^r retry`.
- [ ] To show the new assertion works, temporarily replace `key R` with a key that leaves Plan on screen, and confirm the integration case `risk selected policy unavailable` fails. Record this in the PR body, and revert the swap.
- [ ] Every existing assertion in the tape stays.

### US-012: The identity tests stop waiting on real timer ticks
**Issue:** `<I-2>`

**Description:** As a maintainer, I want the assessment identity tests to run without real 1-second ticks, so that `t.Parallel()` stops hiding a 1–2s cost for each subtest.

**Evidence:** `plugins/risk/assessment_identity_test.go` and `plugins/blastradius/assessment_identity_test.go` run `Activate` through `runLifecycleCmd`/`runBlastCmd`. Those helpers execute the batched `timer.Start()`, which is `tea.Tick(time.Second)` (`pkg/sdk/ui/timer.go:92`). The request is issued synchronously inside `Activate`.

**Acceptance Criteria:**
- [ ] The identity tests call `p.Activate(...)` directly and discard the returned cmd. Every existing assertion stays, word for word.
- [ ] `go test -count=1 -run AssessmentIdentity -v ./plugins/risk/ ./plugins/blastradius/` reports each subtest under 0.1s. The PR body shows the before and after times.
- [ ] The PR body lists any other test in these packages that still pays a real tick.

### US-013: Blast Radius opens with data when no plan has run
**Issue:** #76 (Blast Radius part)

**Description:** As a TUI user, I want Blast Radius to request its assessment when I open it cold, so that I see results instead of the empty state.

**Evidence:** `Activate` in `plugins/blastradius/assessment.go:33-49` returns nil when `e.summary == nil`. Risk requests with a nil summary, and the host then runs the plan (`pkg/sdk/plugin_deps.go:32-38`). `demo/tapes/blastradius.tape` opens Blast Radius cold and records the empty state.

**Acceptance Criteria:**
- [ ] A unit test fails before the fix: `Activate` with no summary returns a cmd that requests an assessment.
- [ ] `Activate` requests an assessment with or without a summary. `HandleRiskAssessed` stores the summary the event carries.
- [ ] A new tape, `tests/fixtures/tapes/macro/blastradius_render.tape`, presses `B` with no plan run first and asserts a resource address from the fixture. It is wired into `tests/integration/macro_test.go`.
- [ ] `tests/fixtures/tapes/smoke/risk_consumers.tape` passes unchanged.

### US-014: Phantom receives a plan
**Issue:** #76 (Phantom part; closes the issue)

**Description:** As a TUI user, I want Phantom to analyze the plan, so that its screen shows phantom changes instead of the empty state.

**Evidence:** `plugins/phantom/phantom.go` has no `Activate` and no `HandlePlanCompleted`. `Analyze` (`:72`) has no caller outside tests. No tape covers the screen.

**Acceptance Criteria:**
- [ ] A unit test fails before the fix: delivering `sdk.PlanCompletedEvent` fills the Phantom list.
- [ ] A unit test fails before the fix: `Activate` with no plan returns a cmd that obtains one, using the same SDK mechanism Risk and Blast Radius use.
- [ ] Phantom implements `HandlePlanCompleted` and `Activate` with those behaviors, importing only `pkg/sdk`.
- [ ] A new macro tape presses `P` cold against a fixture that has phantom changes and asserts one of them. It is wired into `tests/integration/macro_test.go`.
- [ ] The `tfui phantom` command and per-change engine reasons mentioned in #76's comments are out of scope. The PR body says so.
- [ ] `docs/plugins/` describes when Phantom's screen fills.

### US-015: Blast Radius and Phantom lists render through ContentPanel
**Issue:** #97 (two of the four remaining lists)

**Description:** As a TUI user, I want the Blast Radius and Phantom lists to scroll, wrap and show a gutter like every other list.

**Evidence:** `plugins/blastradius/blastradius.go:202` uses `maxVisible := height - 10`. `plugins/phantom/phantom.go:177` uses `height - 10`. Risk already renders through `e.panel.Render(ui.RenderParams{...})` (`plugins/risk/risk.go:260`).

**Acceptance Criteria:**
- [ ] Both lists render through `ui.ContentPanel`, using the shared computation from US-006. Neither file keeps a hand-rolled `height - N` window.
- [ ] For each list, a test asserts that with more rows than fit, moving to the last row keeps it visible and shows the gutter.
- [ ] For each list, a test asserts `ctrl+w` wraps long rows and the hints show the wrap state, as Risk's do.
- [ ] Existing tests and tapes for both plugins pass. Any expected-output change is listed in the PR body with the reason.

### US-016: Validate and Output lists render through ContentPanel
**Issue:** #97 (the last two lists; closes the issue)

**Description:** As a TUI user, I want the Validate and Output lists to scroll, wrap and show a gutter like every other list.

**Evidence:** `plugins/validate/validate.go:280` uses `height - 6`. `plugins/output/output.go:290` uses `height - 5`.

**Acceptance Criteria:**
- [ ] Both lists render through `ui.ContentPanel`. Neither file keeps a hand-rolled `height - N` window.
- [ ] For each list, a test asserts that with more rows than fit, moving to the last row keeps it visible and shows the gutter.
- [ ] For each list, a test asserts `ctrl+w` wraps long rows and the hints show the wrap state.
- [ ] `grep -rn 'height - [0-9]' plugins/` finds no list-window arithmetic in the five plugins #97 names. The output goes in the PR body.

### US-017: demo/generate.sh fails when a recording fails
**Issue:** #77

**Description:** As a maintainer, I want the demo generator to exit non-zero when a recording fails, so that a broken tape cannot ship a stale GIF unnoticed.

**Evidence:** `demo/generate.sh:48-54` passes only flags tfui has, so the original flag failure is fixed. Line 54 still swallows failures with `|| echo "  WARNING: ..."`.

**Acceptance Criteria:**
- [ ] A failed recording makes the script exit non-zero after it names every tape that failed.
- [ ] `./demo/generate.sh <tfui>` exits 0 on `main` after US-018 merges. If a tape fails, fix the tape in this PR.
- [ ] To show it works, break one tape temporarily, confirm the non-zero exit, record it in the PR body and revert.
- [ ] `demo/README.md` states the exit behavior.

### US-018: Remove the unused seed functions and fixtures
**Issue:** #87

**Description:** As a maintainer, I want the dead seed path removed, so that every line serves a current use case and the demo docs stop describing a workaround.

**Evidence:** `ServiceCache.SeedOutputs`, `SeedDiagnostics` and `SeedWorkspaces` (`internal/terraform/service_cache.go:169,219,269`) have no caller outside tests. `demo/fixtures/outputs.json`, `validate.json` and `workspaces.json` are unused. The owner chose option 2 of #87 on 2026-11-19.

**Acceptance Criteria:**
- [ ] The three functions, their tests and the three fixtures are deleted. Nothing else in `ServiceCache` changes.
- [ ] The `output`, `validate` and `workspace` demo tapes and their GIFs in `docs/assets/demo/` are removed, and the plugin pages no longer embed them.
- [ ] `demo/README.md` no longer documents the workaround.
- [ ] `grep -rn 'SeedOutputs\|SeedDiagnostics\|SeedWorkspaces' .` finds nothing.

### US-019: Apply forwards lock and parallelism options
**Issue:** #64

**Description:** As a CLI user, I want `tfui apply` to pass my lock and parallelism settings to terraform, so that apply behaves like plan.

**Evidence:** `internal/terraform/exec/service.go:170-182` builds apply options from only `PlanFile`, `Targets`, `VarFiles` and `Vars`. Plan maps `Parallelism`, `Lock` and `LockTimeout` (`:450-460`). The test named in #64 only asserts the macro adapter.

**Acceptance Criteria:**
- [ ] A test that runs the exec adapter against a stub terraform binary, without `-macro`, fails before the fix. It asserts the apply command line carries `-lock`, `-lock-timeout` and `-parallelism` when they are set.
- [ ] Apply forwards those three options. Plan and Apply share one translation of the lock options.
- [ ] Apply treats `ExtraArgs` exactly as Plan does. If neither forwards them, the existing stderr warning covers both, and the PR body cites it.
- [ ] The test #64 names is fixed or replaced so that it guards the exec adapter.

### US-020: A signal during final cleanup exits 130 or 143
**Issue:** #102

**Description:** As a CLI user, I want a signal that arrives during plugin cleanup to produce the signal's exit code, so that scripts see an interrupted run as interrupted.

**Evidence:** The stop func of `startInterruptGuard` in `cmd/tfui/interrupt.go` runs `signal.Stop` and `close(signals)` without waiting for the watcher goroutine. `Watch` has no `done` channel.

**Acceptance Criteria:**
- [ ] A test in `cmd/tfui/interrupt_test.go` fails before the fix. A signal delivered just before stop must be reflected in the exit code after stop returns.
- [ ] Stop waits for the watcher to finish, as #102's body proposes.
- [ ] The test passes 100 times in a row under `-race -count=100`.

### US-021: A signal during the editor handoff cannot block the forced exit
**Issue:** #104

**Description:** As a CLI user, I want a second signal to force the exit even while an editor is open, so that tfui never hangs on Ctrl+C.

**Evidence:** `Watch` in `cmd/tfui/interrupt.go` calls `endRun()` synchronously on the first signal, and `cmd/tfui/session.go:141` passes `program.Quit`. That `program.Quit` blocks while the loop runs the editor is an inference from BubbleTea's design, so the first criterion proves it.

**Acceptance Criteria:**
- [ ] A test in `cmd/tfui/interrupt_test.go` fails before the fix. With an `endRun` that blocks, a second signal must still trigger the forced exit.
- [ ] The watcher no longer blocks on `endRun`.
- [ ] US-020 and US-021 ship in one PR, and both tests pass under `-race -count=100`.

### US-022: The first signal stops macro playback
**Issue:** #105

**Description:** As a CLI user, I want Ctrl+C to stop a running macro, so that a long tape does not have to finish before tfui exits.

**Evidence:** `cmd/tfui/session.go:101` calls `guard.Watch(func() {})`. The `tape != nil` branch in `runStandalone` (`:287-294`) returns `runner.Execute(tape)` without checking for an interrupt. `internal/macro/runner.go:128` sleeps with `time.Sleep(dur)`.

**Acceptance Criteria:**
- [ ] A test in `internal/macro/runner_test.go` fails before the fix. Cancelling during a `sleep` or `wait` command returns within 100ms and runs no later command.
- [ ] `Runner` accepts a `context.Context` and checks it between commands and inside sleep and wait.
- [ ] A test in `cmd/tfui/session_test.go` asserts that the first signal cancels playback in both call sites and that the exit code is 130.
- [ ] Tapes that run to the end behave as before, and the integration suite passes unchanged.

### US-023: ctrl+r follows one standard in every mode
**Issue:** #90

**Description:** As a TUI user, I want `ctrl+r` to do what its hint says on every screen, in the full TUI and in standalone commands, so that no screen shows a key that does nothing.

**Evidence:** `plugins/apply/apply.go:326-327` sends `NavigateMsg{PluginID:"plan"}` on `ctrl+r` and shows the hint (`:115`). In standalone `tfui apply`, `internal/ui/app.go:351` drops that message. The owner requires the result to follow the standard UX.

**Acceptance Criteria:**
- [ ] The PR body quotes the `ctrl+r` rule from `docs/reference/tui-ux.md` (the `tfui-tui-ux` spec). If the spec has no rule that covers standalone mode, the PR adds one sentence that does, and the fix follows it.
- [ ] An audit lists every plugin that shows a refresh or retry hint, and what `ctrl+r` does there in the full TUI and standalone. The table goes in the PR body.
- [ ] A test in `internal/ui` fails before the fix. In standalone apply on the result screen, either `ctrl+r` performs the standard action, or the hint is absent and the key is a no-op. The spec decides which.
- [ ] Every row of the audit where a shown hint does nothing is fixed the same way. No plugin gets a one-off rule.
- [ ] The fix adds no exported SDK field unless the spec's rule cannot be met without one. If it needs one, stop and ask first.
- [ ] The `tui-checker` agent's verdict on the changed screens is quoted in the PR body.

### US-024: Filter plan changes by action symbol
**Issue:** #23

**Description:** As a TUI user, I want to type `+`, `~`, `-` or `-/+` in the Plan filter, so that I see only changes of that action.

**Evidence:** `SetFilter` in `plugins/plan/plan.go:565` always fuzzy-matches on the address (`:126`). It has no case for the action symbols.

**Acceptance Criteria:**
- [ ] The issue is moved to Todo as `docs/development/backlog.md` requires before the work starts.
- [ ] Table-driven tests fail before the change: each of the four symbols, typed as the whole filter, keeps only changes of that action.
- [ ] Any other filter text behaves as before.
- [ ] Every acceptance item in #23 is checked off in the PR body with its test.
- [ ] `docs/plugins/` documents the symbols for the Plan filter.

### US-025: Record commands interactively with -dry-run
**Issue:** #29

**Description:** As a user, I want `tfui -dry-run` to run the real TUI without touching terraform state and print the commands it would have run, so that I can review them first.

**Evidence:** No dry-run flag exists. The backend is chosen only by `s.macro.Active()` (`cmd/tfui/session.go:83`, `:234`). `emitRecorded` (`:145`) and `WriteRecordedCommands` (`cmd/tfui/effects.go:33`) already print recorded commands.

**Acceptance Criteria:**
- [ ] The issue is moved to Todo before the work starts.
- [ ] Tests fail before the change: with `-dry-run`, the session uses the recording service, runs the interactive program (not the macro driver), and prints the recorded commands on exit.
- [ ] Without the flag, behavior is unchanged.
- [ ] The flag follows the conventions in the `tfui-cli-ux` skill. The `cli-checker` agent's verdict is quoted in the PR body.
- [ ] Every acceptance item in #29 is checked off in the PR body with its test.
- [ ] `docs/reference/cli-reference.md` documents the flag.

### US-026: -target on the root command pre-pins resources
**Issue:** #32

**Description:** As a TUI user, I want `tfui -target <address>` to start with those resources pinned, so that I do not pin them by hand each session.

**Evidence:** `-target` exists only on the plan and apply subcommands (`cmd/tfui/plan_command.go:30`, `apply_command.go:44`), not on root (`root.go:79-88`). `startupContext` (`internal/ui/app.go:861`) starts with no pins. `WithPins` and `TogglePins` exist.

**Acceptance Criteria:**
- [ ] The issue is moved to Todo before the work starts.
- [ ] Tests fail before the change: the root command accepts repeated `-target`, and the startup Context carries those addresses as pins.
- [ ] Pins stay pins. They are not sent as terraform targets unless the user acts on them, which follows the pins-versus-targets rule in `CONTEXT.md` and ADR-0019.
- [ ] A macro tape starts with `-target` and asserts the pinned marker on that row.
- [ ] Every acceptance item in #32 is checked off in the PR body with its test.
- [ ] `docs/reference/cli-reference.md` documents the root flag.

## Functional Requirements

- FR-1: Pressing Esc in Blast Radius must return `sdk.DeactivateMsg` whatever the status.
- FR-2: When wrap is on, a `ContentPanel` list must draw the cursor row's first line whenever the cursor is within the rows and `Height >= 1`.
- FR-3: When wrap is off, `ContentPanel.Render` output must not change.
- FR-4: The coordinator must not serve a cached assessment for which `Usable()` is false, and must run one evaluation for repeated requests that produce a usable one.
- FR-5: The Risk list and inspector must label a deposed object with its deposed key.
- FR-6: The v1 JSON report, text stdout, stderr and exit codes of the Risk CLI must stay byte-for-byte what they are at `c9a1ae7`, except for the fallback reason in FR-7.
- FR-7: `failureDiagnostic` must never return an empty string for an assessment that cannot be used.
- FR-8: Risk, Blast Radius and Phantom must each show results when opened with no plan run in the session.
- FR-9: The Risk, Blast Radius, Phantom, Validate and Output lists must render through `ui.ContentPanel`.
- FR-10: `demo/generate.sh` must exit non-zero when any recording fails.
- FR-11: `tfui apply` must forward `-lock`, `-lock-timeout` and `-parallelism` to terraform.
- FR-12: A signal must produce exit code 130 or 143 whenever it arrives before the process exits, and a second signal must always force the exit.
- FR-13: The first signal must stop macro playback.
- FR-14: No screen may show a `ctrl+r` hint for a key that does nothing.
- FR-15: The Plan filter must treat `+`, `~`, `-` and `-/+` as action filters.
- FR-16: `tfui -dry-run` must run interactively against the recording service and print the recorded commands on exit.
- FR-17: `tfui -target <address>` must start the TUI with that address pinned.

## Non-Goals

- **Large roadmap features:** #24 (split the App struct), #25 (Infracost), #26 (drift detection), #27 (remove ExtraArgs passthrough), #28 (external plugins over gRPC), #30 (binary `.tfplan`), #31 (remote sources). Each is medium to extra large.
- **#33 (yank to filter):** its own open questions need a design first.
- **Docs rewrites:** #72 (Diátaxis regrouping), #73 (one voice), #74 (diagrams).
- **#116:** it stays open until the owner has reviewed the measured costs.
- **`<I-3>`, the host cancel on `PlanCompletedEvent`:** no path reaches it, and a stale result is already ignored. The issue tracks it, with no code.
- Changing the built-in Lambda severities (#106 closes at the current levels).
- A `tfui phantom` command or per-change engine reasons.
- Bringing back seed flags for outputs, diagnostics or workspaces.
- Injecting a clock into `ui.Timer`.
- The plan flags #61 listed (`-var`, `-var-file`, `-destroy` and the rest). The docs no longer advertise them.

## Technical Considerations

- Plugins import only `pkg/sdk`. The shared window computation from US-006 lives in `pkg/sdk/ui`, and US-015 and US-016 reuse it.
- US-007 changes the cache for every coordinator client, so Blast Radius retries get the same behavior.
- US-009 and US-010 touch `lifecycle.go:113-130` and `report.go` together, which is why they share PR F.
- US-013 and US-014 make the Phantom and Blast Radius demo tapes record real data. Regenerating the GIFs is part of US-017's check, not a separate story.
- US-018 deletes demo tapes that US-017's script would otherwise run, so M merges before L is verified.
- The macro driver runs Cmds synchronously, so a tape cannot observe a loading state. Loading behavior is tested at unit level.
- `cmd/` and `internal/terraform/exec` are outside the coverage threshold. US-019 to US-022 still need the tests their criteria name.
- Coverage is 100% elsewhere, so every new branch needs a test.

## Success Metrics

- 23 existing issues are closed: #23, #29, #32, #61, #64, #76, #77, #87, #90, #97, #102, #104, #105, #106, #109, #110, #112, #113, #114, #115, #117, #118 and #142. #107 makes 24 if its rule allows. The two issues this PRD files for its own PRs, `<I-1>` and `<I-2>`, also close.
- The open count goes from 36 to 14, or 13 if #107 closes: the eleven Non-Goal issues, #116, #107 and the newly filed `<I-3>`.
- 20 PRs are open or merged, each with every gate passing and a Codex adversarial review validated.

## Open Questions

- US-023 depends on what `docs/reference/tui-ux.md` says about `ctrl+r` in standalone mode. If the spec is silent, the story adds the rule, and the owner reviews that sentence in the PR.
- US-004: whether #107 can close while #116 is open depends on #107's own closing rule, which the story checks.
