[PRD]
# PRD: Close the risk-assessment review findings (PRs #121, #129, #130)

## Overview

This PRD replaces `tasks/prd-risk-pr121-fixes.md` and `tasks/prd-review-fixes.md`. Every claim in both was checked
against the code on 2026-11-19: `origin/main` at `f29a3a1` (v1.32.0, #121 merged) and
`refactor/assessment-coordinator` at `d144305` (main plus #129 and #130, open as PR #131).

**The PR #121 PRD is done.** All 18 stories shipped in #121 and #129. `go test ./...` passes, coverage is 100%, the
risk integration tests, the risk macro tapes and the 22 smoke tapes pass. These deviations from its wording are accepted
and need no work:

- `LoadPlan`'s fallback decode re-marshals the plan when `tfjson` rejects it (`internal/terraform/loader.go:64`). The
  normal path decodes the original bytes.
- The status-string grep also matches the JSON tags `json:"classified"` and `json:"not_assessed"`
  (`plugins/risk/report.go:103-104`). They are wire names, not comparisons.
- `PolicyAssessment.Resource(InstanceKey)` became `Rows() AssessmentRows` (47139cd).
- The `prepare` guard in `internal/riskpolicy/evaluator.go:111-113` is reachable, so it stays and returns
  `compilation_failed`.
- The Plan "Busy until assessed" rule became `Ready()` in #130; Busy now means "Terraform is running".
- Two Blast Radius assertions changed to follow #130 and 9afc537.

**What remains** is every finding in the review-fixes PRD. A Codex adversarial review of PR #131 independently
confirmed the three worst: evaluation keeps running after navigation, a partly unknown map hides known changes, and
symlinked policy paths fail. Verification found these errors in that PRD, corrected below:

- Risk cannot keep a no-input `Activate()` beside `Activate(input)`, because Go allows one method per name.
- `risk.Analyze` is called outside `plugins/risk`, at `internal/terraform/exec/policy_evidence_test.go:49`.
- Deleting `riskReason` changes two golden files unless their setup supplies rows.
- Changing `ApplyAssessment`'s signature breaks Risk's call, so that call cannot stay untouched.
- The symlink removal breaks `TestLoadSource_GivenFilesystemFailure_*`, which counts the extra stat calls.
- The unusable-status fallback lives at `plugins/risk/lifecycle.go:114` and never reaches output.
- Several line citations were off.

Work happens in two places:

- **`main`:** US-001 to US-003, each pushed directly to `main`.
- **`refactor/assessment-coordinator`:** US-004 to US-012. US-013 merges #131.

## Goals

- A cancelled or replaced `tfui plan --json` run can no longer overwrite the current run's output.
- A partly unknown map or object attribute shows its known keys, as `terraform plan` does.
- A custom risk policy loads through symlinks.
- Leaving a plugin cancels its own pending assessment, both plan and policy evaluation, and nothing else.
- A cached assessment returns without compiling the policy.
- Each assessment's rows are indexed once.
- No production code exists only for tests.
- The Risk plugin builds its view from the assessment once and decides its outcome in one place, with unchanged
  output.
- PR #131 lands on `main`, and issues #120, #122, #126, #127 and #128 close.

## Quality Gates

These commands must pass for every user story:

- `mise run check:lint`: lint, 0 issues
- `mise run test:unit`: unit tests
- `mise run check:build`: compile
- `mise run test:coverage`: 100% outside `cmd/` and `internal/terraform/exec`
- `GOFLAGS=-count=1 mise run test:integration`: integration tests, uncached

For stories that touch `View()`, hints, frames or rendered text (US-002, US-010, US-011, US-012), also:

- `mise run build && mise run test:macro`: macro tapes against the built binary

## Process rules for every story

From `AGENTS.md`:

- The `test-writer` agent writes the failing test before any implementation code. A bug fix ships with a test that
  failed before the fix; the report shows that failure.
- Never make a check pass by weakening it: no skipped tests, no suppressed warnings, no loosened assertions.
- Plugins import only `pkg/sdk`. Comments record only reasons the code cannot show.
- Small conventional commits, each ending with a blank line and
  `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Push after each commit.
- Never use bare `git stash`. Never force-push.
- Report each quality gate's result in the story's final report.

## User Stories

### US-001: Ignore a stale `plan --json` result (`main`)

**Description:** As a `tfui plan --json` user, I want a result from a cancelled or replaced run ignored, so that the
output always belongs to the run I asked for. No key path reaches this today: ctrl+r works only in Done or Error
(`plugins/plan/frames.go:114`), and Cancel reaches a Loading run only through `:q!` or a standalone deactivate, both
of which quit. This is hardening.

**Acceptance Criteria:**

- [ ] `planJSONMsg` (`plugins/plan/plan.go:47`, built at `:332`, handled at `:399`) carries the run identity
      `PlanResultMsg` carries (`run context.Context`, `:43`).
- [ ] The handler drops a `planJSONMsg` whose run is not `e.run`, as the `PlanResultMsg` guard at `:430` does.
- [ ] Tests: the two tests already exist, uncommitted, in the main checkout's `plugins/plan/plan_test.go`
      (`TestUpdate_GivenCancelledJSONRun_WhenItsResultArrivesLate_ShouldIgnoreIt` and
      `TestUpdate_GivenReplacedJSONRun_WhenBothResultsArrive_ShouldApplyOnlyTheCurrentRun`). Both fail today. Commit
      them with the fix; do not rewrite or discard them.
- [ ] One `fix(plan):` commit, pushed directly to `main`.

### US-002: Show the known keys of a partly unknown map (`main`)

**Description:** As a reviewer, I want a map or object attribute with some unknown keys to show its known changes,
so that an unknown sibling does not hide a real change. Reproduced: before `{env: prod, id: a}`, after `{env: stage}`,
after_unknown `{id: true}` renders `new=(known after apply)`, and the `env` change disappears.

**Acceptance Criteria:**

- [ ] Split the unknown check from the sensitive check. Today both `isKeySensitive`
      (`internal/terraform/plan_parser.go:151`) and the unknown check at `:104` call `maskMarked` (`:155`), which
      calls `markedValue` (`:168-185`); `markedValue` reports the whole value as marked when any nested leaf is. The
      sensitive path keeps that whole-value behavior, and so does `internal/terraform/policy_evidence.go:244`.
- [ ] Keep one `AttributeDiff` per top-level key, so compact row counts (`pkg/sdk/attribute_diff.go:39,72`) stay
      the same. When only some leaves are unknown, render `after` with each unknown leaf shown as
      `(known after apply)`. Terraform omits unknown keys from `after` (inference; confirm with a real
      `terraform show -json` fixture), so the unknown leaves are added from `after_unknown`.
- [ ] A wholly unknown value still shows `(known after apply)` alone.
- [ ] `RenderCompactAttributeDiffs` stops replacing a whole value that contains `(known after apply)` with that
      string (`pkg/sdk/attribute_diff.go:51-58`); it truncates the value like any other.
- [ ] Tests, written first and failing before the fix:
  - [ ] `tags = {env: "prod"→"stage", id: <unknown>}` renders the `env` change, with `id` as `(known after apply)`.
  - [ ] Nested objects, and lists of objects with a partly unknown element, render the same way.
  - [ ] A partly sensitive map stays wholly masked.
  - [ ] Plan inspect and Risk's compact rows both show the known keys.
- [ ] One `fix(plan):` commit, pushed directly to `main`.

### US-003: Remove symlink handling from the policy loader (`main`)

**Description:** As a user, I want a custom risk policy to load through a symlinked file or directory, so that Risk
and Plan never report `source_invalid` because of a link. This reverses the earlier non-goal in the PR #121 PRD and
answers issue #126; the review grilling decided it.

**Acceptance Criteria:**

- [ ] Delete `rejectSymlinks` (`internal/riskpolicy/source.go:164-180`) and the module-entry symlink check (`:137`).
      Replace `os.Lstat` with `os.Stat` at `:44` and in the `lstat` uses at `:111` and `:150`, so a linked file or
      directory counts as what it points to. Rename the `lstat` seam to match.
- [ ] The three rejection cases in `TestLoadSource_GivenInvalidSelection_ShouldFailWithoutBuiltinFallback`
      (`internal/riskpolicy/source_test.go:145-169`) become positive tests, written first and failing before the
      fix: a linked selected file, a linked ancestor directory and a linked module entry each load.
- [ ] `TestLoadSource_GivenFilesystemFailure_ShouldReturnFixedUnavailableDiagnostic` (`source_test.go:264-300`)
      counts the stat calls `rejectSymlinks` makes (`calls == 2`, `calls == 3`, "ancestor stat denied"). Rewrite it
      to the new call sequence, keeping one case per stat failure point, and say in the report what each rewritten
      case now covers. Update the other `lstat` seam uses (`source_test.go:232,252,272-288,316,340`,
      `error_code_test.go:37`).
- [ ] Remove the symlink text from `docs/reference/risk-policy-contract.md:367-369`, including "Resolve lexical
      absolute paths without following symlinks". These are the only symlink mentions in `docs/`. Fix the text in
      place; add no note about the removal.
- [ ] The digest, `policy.id` and the reports stay unchanged (existing tests).
- [ ] One `fix(risk):` commit, pushed directly to `main`, with `Closes #126` in the body.

### US-004: Bring `main` into the #131 branch

**Description:** As the maintainer, I want `refactor/assessment-coordinator` to contain `main`'s fixes, so that #131
merges cleanly and later stories build on current code.

**Acceptance Criteria:**

- [ ] Merge `origin/main` into `refactor/assessment-coordinator`. Do not rebase: the branch holds merged PRs.
- [ ] Resolve any conflicts so that both the US-001 to US-003 fixes and the #129/#130 code survive. As of
      `f29a3a1`, `git merge-tree` reports no conflicts.
- [ ] All quality gates pass on the merge result before pushing.

### US-005: Cancel only a plugin's own assessment, including its policy evaluation

**Description:** As a user with a large plan and a custom policy, I want leaving a plugin to stop its own pending
evaluation and nothing else, so that moving back and forth does not pile up OPA evaluations. Today:

- `CancelPlans` (`internal/riskassess/coordinator.go:96-102`) cancels only the shared `planCtx`, so a summary
  request, which runs on `c.ctx` (`:131-135`), keeps evaluating.
- The shared `planCtx` (`:86-92`) means one cancel stops every plugin's host-run plan.
- `cancelledThrough`/`supersededThrough` (`:81`, `:97`, `:118`) let one plugin's cancel drop another plugin's later
  result. No key path shows that today: `cancelPlugin` targets the active plugin (`internal/ui/app.go:357,699,765`);
  only `:q!` (`:1022-1023`) cancels all, and it quits.

**Acceptance Criteria:**

- [ ] The coordinator keeps one `context.CancelFunc` per requester, replaced on each new request from that requester
      (the old one is called). The context derives from the generation context and covers both the host-run
      `terraform plan` and the policy evaluation.
- [ ] The Cmd calls its own `CancelFunc` when it finishes. It never touches the coordinator's per-requester map: only
      the Update loop mutates coordinator state (`coordinator.go:31-33`).
- [ ] The requester is known by building `RequestAssessment` per plugin: copy the shared `deps` in the Init loop
      (`internal/ui/app.go:157-176`) with a closure capturing `p.ID()`. Plugins already copy
      `deps.RequestAssessment` at Init (`plugins/plan/plan.go:165`, `plugins/risk/risk.go:77`,
      `plugins/blastradius/blastradius.go:99`). No new public SDK method.
- [ ] `App.cancelPlugin(p)` cancels only `p`'s request.
- [ ] Remove `cancelledThrough`, `supersededThrough` and the supersede logic. `Current` checks only the generation;
      each plugin keeps only its own request by `AssessmentID`.
- [ ] `Advance` (Context change or plan invalidation) still cancels and drops every request.
- [ ] `planForAssessment` returns `ctx.Err()` before `os.MkdirTemp` and `Service.Plan` when the request is already
      cancelled (`coordinator.go:168-181`).
- [ ] Add the unexported in-package evaluator seam in `internal/riskassess` that US-005, US-006 and US-008 tests use.
- [ ] The doc on `sdk.PolicyAssessment` (`pkg/sdk/policy_assessment.go:6`) states the behavior: the host may dispatch
      a cancelled request's result, and the requester ignores it by `AssessmentID`. Plan, Risk and Blast Radius
      already clear their pending ID on Cancel (`plugins/plan/plan.go:281`, `plugins/risk/lifecycle.go:30-32`,
      `plugins/blastradius/assessment.go:13-16`).
- [ ] Tests, written first and failing before the fix:
  - [ ] A request cancelled before its Cmd runs never calls `MockService.Plan`.
  - [ ] In package `riskassess`, through the seam: cancelling requester A cancels A's blocking evaluation, and
        requester B's concurrent request completes.
  - [ ] At App level, with no evaluator injection: a nil-summary request whose `MockService.Plan` blocks on its
        context is cancelled by `cancelPlugin` on its plugin and not by `cancelPlugin` on another plugin. The test
        calls `cancelPlugin` on the non-active plugin directly.
  - [ ] In `internal/ui/risk_pending_guard_test.go`, after navigation proceeds during a policy-only assessment, the
        late `RiskAssessedEvent` is delivered and the plugin does not accept it.
- [ ] Tests that assert the removed behavior, including `TestCancelPlans_GivenSuppliedSummary_ShouldKeepEvaluatingIt`,
      are rewritten to the new behavior. The report says why for each.
- [ ] Rewrite `docs/development/architecture.md:193-195`, which states the supersede rule and that a supplied summary
      keeps evaluating. Re-read the `requireIdle` comment (`internal/ui/app.go:991-1002`); it becomes accurate after
      this story, so change it only if it is not.
- [ ] Include `Refs #124` in the commit body; the story fixes its item 4.

### US-006: Return a cached assessment without compiling the policy

**Description:** As a user returning to Plan on an assessed plan, I want the cached result without a policy compile,
so that navigation stays fast.

**Acceptance Criteria:**

- [ ] In `run()` (`internal/riskassess/coordinator.go:145`), with a non-nil summary, check the cache after
      `loadPolicy` and before `riskpolicy.Compile` (`:153`). The key (`:62-67`) needs only `snapshot.Digest()`,
      which exists before Compile.
- [ ] `loadPolicy` still runs first, so the source is re-read at every assessment boundary as the contract requires.
- [ ] A cache hit returns the cached result and never calls Compile.
- [ ] A nil-summary request still compiles before planning, so a broken policy never runs Terraform.
- [ ] A test, written first and failing before the fix: a second identical request returns the cached result, and a
      compile counter injected through an unexported in-package seam stays at 1.

### US-007: Index assessment rows once per result

**Description:** As a user with a large plan, I want each assessment indexed once, so that applying a result does not
build the same map twice.

**Acceptance Criteria:**

- [ ] `sdk.ApplyAssessment` (`pkg/sdk/policy_assessment.go:69`) takes an `AssessmentRows` argument instead of
      calling `assessment.Rows()` (`:78`).
- [ ] Blast Radius passes the rows `setAssessment` built (`plugins/blastradius/assessment.go:66-71`) at its call
      (`:84`).
- [ ] Risk's call (`plugins/risk/lifecycle.go:101`) changes only mechanically, passing `e.assessment.Rows()`; US-010
      removes it. Plan does not call `ApplyAssessment`.
- [ ] `BenchmarkApplyAssessment` (`pkg/sdk/assessment_helpers_test.go:263`) moves to the new signature. Report its
      ns/op and allocs/op before and after.
- [ ] Existing `ApplyAssessment` tests keep their assertions; only the call changes.

### US-008: Remove code that exists only for tests

**Description:** As a maintainer, I want every production line to serve a current use case.

**Acceptance Criteria:**

- [ ] Delete the package-level `riskpolicy.Assess` (`internal/riskpolicy/evaluator.go:36`). Its callers, all
      `_test.go` files in package `riskpolicy`, call `Compile(...).Assess(...)` with unchanged assertions.
- [ ] Unexport `Coordinator.RequestWith`; production `Request` (`coordinator.go:124`) keeps calling it as the seam.
      Test callers: `internal/riskassess/harness_test.go:47` uses the seam directly. Tests in
      `internal/ui/risk_assessment_race_test.go:18` and `internal/ui/risk_request_test.go:164,193,241` either move
      into package `riskassess` or drive a nil-summary request through `MockService.Plan`. Assertions are unchanged.
- [ ] Delete Blast Radius's `Busy()` (`plugins/blastradius/assessment.go:10`), which always returns false. Delete
      `plugins/blastradius/busy_test.go`, which tests only the constant. Its other callers
      (`assessment_lifecycle_test.go:19,42,59`, `assessment_identity_test.go:52`,
      `assessment_event_test.go:41,57,100`) assert that the plugin does not implement `sdk.Busy`, or drop a check
      that is now always true. The report names each change.

### US-009: Regression test for stale-pin cleanup during a pending assessment

**Description:** As a user, I want stale pins removed after a plan even while Plan's assessment is pending, so that
later plans do not run with unintended `-target` arguments. #130 fixed this; no test pins it.

**Acceptance Criteria:**

- [ ] Add an App-level test with this scenario:
  1. A plan completes without a pinned address.
  2. Plan's assessment is held pending.
  3. The `PinToggleRequestMsg` from `pruneStalePins` (`plugins/plan/plan.go:638`, emitted at `:456`) is processed.
  4. The pin is removed (`internal/ui/app.go:249-250` passes `requireIdle("pin")`).
  5. The pin change replaces the Context, which calls `Advance` (`app.go:906`): Plan is stale, has no pending
     assessment and is not stuck in Loading. The old assessment's late result is dropped.
- [ ] Confirm the test fails at `8e7fe5f` (#129's head, before #130), where Plan's Busy included `assessing`, so
      `requireIdle("pin")` refused the toggle. Use a separate scratch worktree and remove it after; record the
      failure in the report.
- [ ] No production change is expected. If the test fails on the current branch, stop and report.

### US-010: Build the Risk view from the assessment once

**Description:** As a maintainer, I want the Risk plugin to group its rows straight from the assessment, so that it
stops copying the summary, filtering it and looking each row up again.

**Acceptance Criteria:**

- [ ] `acceptAssessment` (`plugins/risk/lifecycle.go`) builds the groups in one pass over the summary's changes
      joined to their assessment rows. Each group item holds the change and its row.
- [ ] Remove from `plugins/risk/`:
  - the `sdk.ApplyAssessment` call and the filter loop after it (`lifecycle.go:101-110`)
  - the second unavailable check in `Analyze` (`risk.go:129`)
  - the `resourceAssessment` lookups (`risk.go:317`, `frames.go:302,308`)
  - the `rows` field (`risk.go:36`) and `setAssessment` (`lifecycle.go:83-88`)
- [ ] The overall level comes from `assessment.OverallLevel`; delete the `sdk.OverallRisk` call (`risk.go:115`).
      `OverallLevel` is nil unless the status is complete. Before changing code, write a test that pins today's
      overall banner for complete, partial, unavailable and empty assessments; the new code keeps each.
- [ ] Delete `riskReason` (`risk.go:349`) and its three tests (`risk_test.go:400,526,534`). The golden tests
      (`golden_test.go:31-75`) set `p.groups` with no assessment, and two golden files show `destructive operation`
      from `riskReason`:
      `TestView_Given_MixedRisk_ShouldRender_GroupedChanges.txt` and
      `TestView_Given_CriticalRisk_ShouldRender_CriticalBanner.txt`. Rebuild those tests' setup from an assessment
      whose rows carry that reason, so both golden files stay byte-identical.
- [ ] Selection uses one flat item list. `totalItems` (`risk.go:195`, also used at `frames.go:29` and `risk.go:207`),
      the `starts`/`ends` lists in `renderAnalysis` (`risk.go:243-256`) and the group walk in `inspectSelected`
      become one index into that list.
- [ ] Unexport `RiskGroup`, `Overall`, `MoveUp` and `MoveDown`. Unexport `Analyze` and rewrite
      `internal/terraform/exec/policy_evidence_test.go:49` to deliver the assessment through `HandleRiskAssessed`,
      keeping what it asserts.
- [ ] Behavior is unchanged: goldens, macro tapes and behavior tests keep their assertions. Tests that reach removed
      internals are rewritten, keeping what they assert, and listed in the report.

### US-011: Decide the Risk outcome in one place and parse the threshold once

**Description:** As a maintainer, I want one value to say how a Risk run ended, so that View, Stderr and ExitCode stop
deriving it from four fields.

**Acceptance Criteria:**

- [ ] Replace `riskDiagnostic`, `planError` (`risk.go:33,39`) and the meanings of `StatusError` (set at `risk.go:89`,
      `lifecycle.go:31,92,113`) with one outcome value, set only on activation and in `acceptAssessment`. It
      distinguishes: invalid threshold, plan failed, assessment unusable, done. It carries its diagnostic line,
      computed once, including the first row error code that `Stderr` finds today by re-sorting rows
      (`report.go:226-236`).
- [ ] `View`, `Stderr` and `ExitCode` switch on the outcome.
- [ ] Parse the threshold once, in `Configure` and activation, into `*sdk.RiskLevel` (nil means off). Remove the
      `"off"` sentinel (`report.go:31,47,57,123`), the repeated `thresholdResult` evaluation
      (`report.go:126,240,250`) and the unchecked `value.(string)` (`report.go:37`).
- [ ] Remove `settingsInvalid` and `overrideError` (`risk.go:42,98-103`) if the outcome makes them redundant.
      Invalid settings still report the host's diagnostic with its hint.
- [ ] Delete the `string(assessment.Status)` fallback (`lifecycle.go:114`). It runs only for a partial or unavailable
      assessment with no global code, whose rows all carry codes, so its text never reaches stderr or the view. A
      test, written first, pins the current stderr line for a partial and an unavailable assessment: the first row's
      error code. Include `Refs #124`; this fixes its item 2.
- [ ] Risk enters through `Activate(input Input) tea.Cmd`, like Plan and State (ADR-0021):
  - Delete `ActivateInput` (`report.go:25-28`) and the no-input `Activate()` (`risk.go:83`).
  - Add `*tfuirisk.Plugin` to the typed switch in `internal/ui/app.go:959-976`, with `tfuirisk.Input{}` for TUI
    navigation.
  - `cmd/tfui/risk_command.go:43` calls `Activate(input)`.
  - Update the callers of `risk.Activate()`: `cmd/tfui/session_risk_test.go:51`,
    `cmd/tfui/session_plan_test.go:59,213` and `cmd/tfui/risk_command_test.go:35`.
  - `Closes #122` in the commit body.
- [ ] `resetDerived` (`lifecycle.go:16`) no longer calls `Cancel`, whose effect it overwrites at `:20` and `:25`.
- [ ] `requestAssessment` records the raw Context; `report.go` resolves the report context and its chdir fallback
      (today `lifecycle.go:57-66`).
- [ ] Stdout, Stderr, exit codes and the JSON report are byte-identical for every existing test, including
      `cmd/tfui/{risk_command,risk_report,session_risk}_test.go`.

### US-012: Share instance ordering and keep Risk's counts in one place

**Description:** As a maintainer, I want instance ordering defined once in the SDK and Risk's row counts computed
once, so that Risk's view, report and stderr cannot drift apart.

**Acceptance Criteria:**

- [ ] Add `sdk.InstanceKey.Compare` (address, then deposed key), test first.
- [ ] Use it for the address-then-deposed sorts: `plugins/risk/risk.go:373-375`, `report.go:152-154`, any sort left
      in `Stderr` after US-011 (`report.go:228-230` today) and `plugins/plan/assessment.go:120`.
- [ ] `deposedKey` (`report.go:198`) stays while `instanceLabel` uses it (`report.go:179,192`).
- [ ] Add one unexported counts helper in `plugins/risk`, test first, returning classified, unavailable and
      not-assessed counts. Only Risk counts rows, so it does not go in the SDK. Use it in `renderOverallBanner`
      (`risk.go:273-291`), in place of `excludedCount` (`risk.go:377-387`, deleted) and in `report()`
      (`report.go:132-149`).
- [ ] Output is unchanged: the US-011 tests keep their assertions.

### US-013: Land PR #131 and close the issues

**Description:** As the maintainer, I want #131 merged once it is clean, so that `main` gets #129, #130 and US-004 to
US-012.

**Acceptance Criteria:**

- [ ] Update the #131 description: list US-004 to US-012 with their commits and each gate's result.
- [ ] Before merging, a review agent reviews the whole #131 diff against `main`, and a Codex adversarial review
      (`/codex:adversarial-review --base origin/main`) reruns. No Critical or Important finding is left open, and the
      three Codex findings of 2026-11-19 (cancellation, partly unknown maps, symlinks) no longer reproduce.
- [ ] CI on #131 is green.
- [ ] Merge #131 into `main` with a merge commit, matching #121.
- [ ] `origin/main` then contains `d812051` (#130's merge) and `8e7fe5f` (#129's head). `dc3abec` (#129's merge)
      stays only on `agent/risk-plugin-pr`.
- [ ] #122, #126, #127 and #128 are closed; close any by hand whose keyword did not fire. Close the epic #120, which
      #121 delivered, with a comment naming the PR.
- [ ] Edit #124: remove items 2 and 4 (fixed by US-011 and US-005) and leave items 1 and 3 open.
- [ ] Do not delete branches or worktrees.

## Functional Requirements

- FR-1: A `planJSONMsg` from a run that is not current must not change Plan's state.
- FR-2: A partly unknown map or object attribute must render its known keys, with only unknown leaves shown as
  `(known after apply)`, in the full and the compact diff. Sensitive masking stays whole-value.
- FR-3: The policy loader must follow symlinks and treat a linked file or directory as its target.
- FR-4: The risk-policy contract must not mention symlinks.
- FR-5: Cancelling one plugin must not drop, delay or cancel another plugin's pending assessment.
- FR-6: Cancelling a plugin must cancel its own pending assessment's context, both the host-run plan and the policy
  evaluation.
- FR-7: A Context change or plan invalidation must cancel and drop every pending assessment.
- FR-8: With a non-nil summary, a cache hit must return without compiling the policy. Config and source are still
  re-read at every assessment boundary.
- FR-9: A broken policy must never run `terraform plan`.
- FR-10: Applying an assessment must build its instance index once.
- FR-11: No exported or package-level production function may exist only for tests, and no method may return a
  constant only because tests call it. Unexported in-package seams used by that package's tests are allowed.
- FR-12: Stale-pin cleanup after a plan must apply while Plan's assessment is pending.
- FR-13: The Risk plugin must look up each assessment row once per result and must not copy the plan summary to
  render it.
- FR-14: The Risk plugin's outcome must be set in one place; View, Stderr and ExitCode read it.
- FR-15: The Risk restructuring must not change any TUI frame, stdout, stderr, exit code or JSON report.
- FR-16: A request cancelled before it runs must not start `terraform plan`.

## Non-Goals

- No ADR, superseding note or "changed from" history in docs. Mistakes are fixed in place.
- No in-flight deduplication of identical evaluations.
- No change to built-in policy severities, the fixed classifier, the report schema or `policy.id`.
- No change to `tfui plan --json` output, which stays raw Terraform JSON.
- No rework of the accepted deviations from the PR #121 PRD listed in the Overview.
- No fix for #125 (`childCfg` stale on return to the root chdir) or #123 (Risk and Blast Radius loading-state UX),
  and no work on #124 items 1 and 3.
- No change to `cmd/tfui/zz_repro_test.go` (`TestRepro_StandaloneApplyTTY`, from f38a939), which is unrelated.
- No deletion of branches, worktrees, backup branches or scratch files.

## Technical Considerations

- **Branches:** US-001 to US-003 run on the main checkout (`/home/lmarqs/Work/terraform-ui`) and push to
  `origin/main`; each `fix:` commit cuts a patch release. US-004 to US-013 run in
  `.worktrees/refactor/assessment-coordinator`; pull first, and confirm it contains `d812051`.
- **Order:** stories run in ID order. US-005 creates the in-package seam that US-006 and US-008 use. US-007 changes
  Risk's `ApplyAssessment` call mechanically; US-010 removes it. US-011 runs after US-010 and US-012 after US-011,
  because each edits the same Risk files.
- **Concurrency:** coordinator state is touched only from the Update loop. A Cmd may call its own `CancelFunc` but
  never mutates the coordinator.
- **Headless:** `macro.Driver` runs Cmds synchronously and depth-first (`internal/macro/driver.go:82-100`); headless
  completion waits on `Ready()`, not `Busy()`, since #130.
- **Coverage:** CI enforces 100% outside `cmd/` and `internal/terraform/exec`, so every new branch needs a test.
- **Contract:** `docs/reference/risk-policy-contract.md` still binds the loader, the policy reload at every boundary
  and the re-check before publishing.

## Success Metrics

- The three Codex findings no longer reproduce, and every review-fixes finding is closed by a commit.
- Every quality gate passes on `main` after US-001 to US-003, and on #131 before it merges.
- CI on `main` is green after #131 merges.
- `BenchmarkApplyAssessment` reports fewer allocs/op than before US-007.
- Issues #120, #122, #126, #127 and #128 are closed; #124 keeps only items 1 and 3.

## Open Questions

None.
[/PRD]
