# PRD: Fix the findings from the review of PRs #121, #129 and #130

## Overview

After PRs #121, #129 and #130 merged, a `/code-review` and a Codex adversarial review found ten issues. A
code-quality review of the Risk plugin then found that it models one assessment twice and derives its outcome in
several places.

Only #121 reached `main`. #129 and #130 merged into their stacked base branches, so their code lives only on
`refactor/assessment-coordinator`, which PR #131 lands on `main`.

This PRD fixes every finding in place. Mistakes are corrected where they were made: no ADR, no superseding
document, no compatibility shim and no dead code left behind.

Work happens in two places:

- **`main`:** fixes to code that is already on `main` (US-001 to US-003). Each is pushed directly to `main`.
- **`refactor/assessment-coordinator`:** fixes to the code PR #131 introduces (US-005 to US-009) and the Risk plugin
  restructuring (US-010 to US-012). The branch first takes in `main` (US-004). The run merges #131 at the end
  (US-013).

## Goals

- A cancelled or replaced `tfui plan --json` run can no longer overwrite the current run's output.
- A partly unknown map or object attribute shows its known keys, as `terraform plan` does.
- A custom risk policy loads from any path.
- One plugin's cancel never drops or cancels another plugin's assessment.
- Leaving a plugin stops its policy evaluation, not only the `terraform plan` the host runs for it.
- A cached assessment returns without compiling the policy.
- Each assessment's rows are indexed once.
- No production code exists only for tests, and no method always returns a constant only because tests call it.
- The Risk plugin builds its view from the assessment once and decides its outcome in one place, with unchanged
  output.
- PR #131 lands on `main`, which removes the navigation block while Plan's assessment is pending.

## Quality Gates

These commands must pass for every user story:

- `mise run check:lint`: lint, 0 issues
- `mise run test:unit`: unit tests
- `mise run check:build`: compile
- `mise run test:coverage`: 100% coverage outside `cmd/` and `internal/terraform/exec`
- `GOFLAGS=-count=1 mise run test:integration`: integration tests, uncached
- `mise run build && mise run test:macro`: cross-platform build and macro tapes

## Process rules for every story

These come from `AGENTS.md` and bind every story:

- The `test-writer` agent writes the failing test before any implementation code. A bug fix ships with a test that
  failed before the fix.
- Never make a check pass by weakening it: no skipped tests, no suppressed warnings, no loosened assertions.
- Plugins import only `pkg/sdk`.
- Use conventional commits, small. Each message ends with a blank line, then
  `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Push after each commit.
- Never use bare `git stash`. Never force-push.
- Comments record only reasons the code cannot show.
- Report each quality gate's result in the story's final report.

## User Stories

### US-001: Ignore a stale `plan --json` result (`main`)

**Description:** As a `tfui plan --json` user, I want a result from a cancelled or replaced run ignored, so that the
output I get always belongs to the run I asked for. No key path reaches this today: Refresh is allowed only in Done or
Error, and Cancel there quits. This is hardening against a latent overwrite.

**Acceptance Criteria:**

- [ ] `planJSONMsg` (`plugins/plan/plan.go:47`, built at `:332`, handled at `:399`) carries the same run identity
      `PlanResultMsg` carries.
- [ ] The handler drops a `planJSONMsg` whose run identity is not the current run, using the guard #121 added for
      `PlanResultMsg` (`:430`).
- [ ] Two tests are written first and fail before the fix:
  - [ ] After Cancel, a late `planJSONMsg` from the cancelled run leaves `status`, `jsonBytes` and `errMsg` unchanged.
  - [ ] After a new run starts, the old run's `planJSONMsg` is ignored and the new run's is accepted. The test starts
        the new run by calling `Plugin.Refresh()` directly while Loading, because no key path allows it.
- [ ] One `fix(plan):` commit, pushed directly to `main`.

### US-002: Show the known keys of a partly unknown map (`main`)

**Description:** As a reviewer, I want a map or object attribute with some unknown keys to show its known changes, so
that an unknown sibling does not hide a real change.

**Acceptance Criteria:**

- [ ] Split the unknown check from the sensitive check. Today `isKeySensitive` (`internal/terraform/plan_parser.go:151`)
      and the unknown check at `:104` share `maskMarked`/`markedValue`, which report a whole value as marked when any
      nested leaf is. The sensitive path keeps that whole-value behavior, and so does
      `internal/terraform/policy_evidence.go:244`.
- [ ] Keep one `AttributeDiff` per top-level key. When only some leaves are unknown, render `after` with each unknown
      leaf replaced by `(known after apply)`. Do not split the attribute into per-key rows, which would change phantom
      detection and compact counts.
- [ ] Wholly unknown values keep showing `(known after apply)` alone.
- [ ] `RenderCompactAttributeDiffs` (`pkg/sdk/attribute_diff.go:49-56`) stops replacing a whole value that contains
      `(known after apply)` with that string alone; it truncates the substituted value like any other.
- [ ] Tests are written first and fail before the fix:
  - [ ] `tags = {env: "prod"→"stage", id: <unknown>}` renders the `env` change, with `id` as `(known after apply)`.
  - [ ] Nested objects, and lists of objects with a partly unknown element, render the same way.
  - [ ] A partly sensitive map stays wholly masked.
  - [ ] Plan inspect and Risk's compact rows both show the known keys.
- [ ] One `fix(plan):` commit, pushed directly to `main`.

### US-003: Remove symlink handling from the policy loader (`main`)

**Description:** As a user, I want a custom risk policy to load from any path, so that Risk and Plan never report
`source_invalid` because of a symlink.

**Acceptance Criteria:**

- [ ] Delete `rejectSymlinks` and every symlink check in `internal/riskpolicy/source.go`, including the directory
      entry check. Replace `os.Lstat` with `os.Stat` (`:44`, and the `lstat` uses at `:111` and `:150`), so a linked
      file or directory counts as the regular file or directory it points to.
- [ ] Remove every mention of policy-path symlinks from `docs/reference/risk-policy-contract.md` (`:367-369`, including
      "Resolve lexical absolute paths without following symlinks") and any other risk doc. Fix the text in place; add
      no note about the removal.
- [ ] The three rejection cases in `TestLoadSource_GivenInvalidSelection_ShouldFailWithoutBuiltinFallback`
      (`source_test.go:145-170`) become positive tests, written first and failing before the fix: a linked selected
      file, a linked ancestor directory and a linked module entry each load.
- [ ] The digest, `policy.id` and the reports stay unchanged.
- [ ] One `fix(risk):` commit, pushed directly to `main`.

### US-004: Bring `main` into the #131 branch

**Description:** As the maintainer, I want `refactor/assessment-coordinator` to contain `main`'s fixes, so that #131
merges without conflicts and the later stories build on the current code.

**Acceptance Criteria:**

- [ ] Merge `origin/main` into `refactor/assessment-coordinator`. Do not rebase: the branch has merged PRs on it.
- [ ] Resolve any conflicts so that both the US-001 to US-003 fixes and the #129/#130 code survive.
- [ ] All quality gates pass on the merge result before pushing.

### US-005: Cancel only a plugin's own assessment, including its policy evaluation

**Description:** As a user with a large plan and a custom policy, I want leaving a plugin to stop its own pending
evaluation and nothing else, so that moving back and forth does not run duplicate OPA evaluations. Today
`CancelPlans` cancels only the plan context (`internal/riskassess/coordinator.go:97-101`), so a summary request's
evaluation keeps running, and it cancels every plugin's host-run plan. Its global `cancelledThrough` and
`supersededThrough` counters can also drop another plugin's result; no key path shows that today, because
`cancelPlugin` only targets the active plugin.

**Acceptance Criteria:**

- [ ] The coordinator keeps one cancel per requester, replaced on each new request from that requester. It derives
      from the generation context and covers both the host-run `terraform plan` and the policy evaluation. The Cmd
      releases its own cancel when it finishes.
- [ ] The requester is known by building `RequestAssessment` per plugin: copy the shared `deps` in the Init loop
      (`internal/ui/app.go:157-173`) with a closure that captures `p.ID()`. No new public SDK method.
- [ ] `App.cancelPlugin(p)` cancels only `p`'s request. Other plugins' requests keep running.
- [ ] Remove `cancelledThrough`, `supersededThrough` and the supersede logic. `Current` checks only the generation;
      each plugin keeps only its own request by `AssessmentID`.
- [ ] `Advance` (Context change or plan invalidation) still cancels and drops every request.
- [ ] `planForAssessment` (`internal/riskassess/coordinator.go`) returns `ctx.Err()` before calling `Service.Plan`
      when the request is already cancelled, so a cancelled request never starts Terraform.
- [ ] The doc on `sdk.PolicyAssessment` (`pkg/sdk/policy_assessment.go:6`) matches the code: either the host never
      dispatches a cancelled result, or the doc says it may and that the requester ignores it by `AssessmentID`.
- [ ] Tests are written first and fail before the fix:
  - [ ] A request cancelled before its Cmd runs never calls `MockService.Plan`.
  - [ ] In `internal/ui/risk_pending_guard_test.go`, after navigation proceeds during a policy-only assessment, the
        withheld `RiskAssessedEvent` is delivered and the plugin does not accept it.
  - [ ] In package `riskassess`, through the in-package evaluator seam: cancelling requester A cancels A's blocking
        evaluation; requester B's concurrent request completes.
  - [ ] At App level, with no evaluator injection: a nil-summary request whose `MockService.Plan` blocks on its
        context is cancelled by `cancelPlugin` on its plugin, and not by `cancelPlugin` on another plugin. The test
        calls `cancelPlugin` on the non-active plugin directly, since no key path does.
- [ ] Tests that assert the removed supersede behavior are rewritten to the ID-matching behavior. The report says why
      for each.
- [ ] Update the `requireIdle` comment in `internal/ui/app.go` and `docs/development/architecture.md:193-195`, which
      describe the supersede rule and say a supplied summary keeps evaluating.

### US-006: Return a cached assessment without compiling the policy

**Description:** As a user returning to Plan on an already-assessed plan, I want the cached result without a full
policy compile, so that navigation stays fast.

**Acceptance Criteria:**

- [ ] In `run()` (`internal/riskassess/coordinator.go:145`; `Compile` at `:153`), with a non-nil summary, check the cache after
      `loadPolicy` and before `riskpolicy.Compile`.
- [ ] `loadPolicy` still runs first, because the cache key needs the digest of the source as read now. This keeps the
      contract's reload at every assessment boundary.
- [ ] A cache hit returns the cached result and never calls Compile.
- [ ] A nil-summary request still compiles before planning, so a broken policy never runs Terraform.
- [ ] A test is written first and fails before the fix: a second identical request returns the cached result, and an
      injected compile counter stays at 1. Use a seam inside the package, not an exported one.

### US-007: Index assessment rows once per result

**Description:** As a user with a large plan, I want each assessment indexed once, so that applying a result does not
build the same map twice.

**Acceptance Criteria:**

- [ ] `sdk.ApplyAssessment` (`pkg/sdk/policy_assessment.go:69`) uses an `AssessmentRows` index the caller passes in,
      instead of calling `assessment.Rows()` again (`:76`).
- [ ] Blast Radius (`plugins/blastradius/assessment.go:84`) passes the rows its `setAssessment` already built. Plan
      does not call `ApplyAssessment`. Risk stops calling it in US-010, so leave its call site as is here.
- [ ] Update `BenchmarkApplyAssessment` to the new signature. Report its ns/op and allocs/op before and after.
- [ ] Existing ApplyAssessment tests keep their assertions; only the call signature changes.

### US-008: Remove code that exists only for tests

**Description:** As a maintainer, I want every production line to serve a current use case, so that the code does not
carry test-only APIs or constant methods.

**Acceptance Criteria:**

- [ ] Delete the package-level `riskpolicy.Assess` (`internal/riskpolicy/evaluator.go:36`). Its tests call
      `Compile(...).Assess(...)` instead, with assertions unchanged.
- [ ] Remove the exported `Coordinator.RequestWith`. Its callers are `riskassess/harness_test.go:47`,
      `internal/ui/risk_assessment_race_test.go:18` and `internal/ui/risk_request_test.go:164,193,241`. Tests that
      inject an evaluator move into package `riskassess` and use an unexported in-package seam. App-level tests that
      stay in `internal/ui` drive a nil-summary request through `MockService.Plan` instead. Assertions are unchanged.
- [ ] Delete Blast Radius's `Busy()` (`plugins/blastradius/assessment.go:10`), which always returns false. Tests that
      called it assert instead that the plugin does not implement `sdk.Busy`, or drop a check that is now always true.
      `plugins/blastradius/busy_test.go` tests only the constant and is deleted; `internal/ui/risk_pending_guard_test.go`
      already covers the App behavior. The report names each change.
- [ ] Coverage stays 100%.

### US-009: Regression test for stale-pin cleanup during a pending assessment

**Description:** As a user, I want stale pins removed after a plan even while Plan's assessment is pending, so that
later plans do not run with unintended `-target` arguments.

**Acceptance Criteria:**

- [ ] Add an App-level test with this scenario:
  1. A plan completes without a pinned address.
  2. Plan's assessment is held pending.
  3. The `PinToggleRequestMsg` from `pruneStalePins` is processed.
  4. The pin is removed.
  5. The pin change replaces the Context, which calls `Advance` (`internal/ui/app.go:906`): Plan is stale, has no
     pending assessment and is not stuck in Loading. The old assessment's late result is dropped.
- [ ] Check out `8e7fe5f`, the #129 head without the #130 fix, and confirm the test fails there: the pin toggle is
      refused by `requireIdle("pin")` every time. Record that in the report.
- [ ] No production change is expected. If the test fails on the current branch, stop and report; reordering inside
      `tea.Batch` would not change the outcome.

### US-010: Build the Risk view from the assessment once

**Description:** As a maintainer, I want the Risk plugin to group its rows straight from the assessment, so that it
stops copying the summary, filtering it and looking each row up again.

**Acceptance Criteria:**

- [ ] `acceptAssessment` (`plugins/risk/lifecycle.go`) builds the groups in one pass over the summary's changes joined
      to their assessment rows. Each group item holds the change and its row.
- [ ] Remove from `plugins/risk/`: the `sdk.ApplyAssessment` call and the filter loop after it, the second
      unavailable check in `Analyze`, every `resourceAssessment` lookup in `renderChangeRow` and `inspectSelected`, the
      `rows` field and `setAssessment`.
- [ ] The overall level comes from `assessment.OverallLevel`; delete the `sdk.OverallRisk` call in `risk.go`.
- [ ] Delete `riskReason` and its three tests (`risk_test.go`: `TestRiskReason_*`). Its value is always overwritten,
      because every rendered change has a row.
- [ ] Selection uses one flat item list. `totalItems`, the `starts`/`ends` lists in `renderAnalysis` and the group walk
      in `inspectSelected` become one index into that list.
- [ ] Unexport `Analyze`, `RiskGroup`, `Overall`, `MoveUp` and `MoveDown`; nothing outside `plugins/risk` calls them.
- [ ] Behavior is unchanged: golden tests, macro tapes and every behavior test keep their assertions. Rewrite only
      tests that reach removed internals, keeping what they assert, and list each in the report.

### US-011: Decide the Risk outcome in one place and parse the threshold once

**Description:** As a maintainer, I want one value to say how a Risk run ended, so that View, Stderr and ExitCode stop
deriving it separately from four fields.

**Acceptance Criteria:**

- [ ] Replace `riskDiagnostic`, `planError` and the meanings of `StatusError` with one outcome value, set only in
      `Activate` and `acceptAssessment`. It distinguishes: invalid threshold, plan failed, assessment unusable, done.
      It carries its diagnostic line, computed once, including the first row error code that `Stderr` now finds by
      re-sorting rows.
- [ ] `View`, `Stderr` and `ExitCode` switch on the outcome.
- [ ] Parse the threshold once, in `Configure` and `Activate(input)`, into `*sdk.RiskLevel` (nil means off). Remove
      the `"off"` sentinel, the repeated `ParseRiskLevel` call and the unchecked `value.(string)` in `configureReport`.
      `thresholdResult` is computed once per result.
- [ ] Remove `settingsInvalid` and `overrideError` if the outcome makes them redundant. Invalid settings still report
      the host's diagnostic with its hint, as today.
- [ ] The unusable-assessment diagnostic never prints a status (`partial`, `unavailable`) as if it were an error
      code. Today `acceptAssessment` falls back to `string(assessment.Status)`. Map each unusable status to a code
      listed in `docs/reference/risk-policy-contract.md`, or prove the fallback unreachable and delete it. A test,
      written first, covers each unusable status's stderr line.
- [ ] Risk enters through `Activate(input Input) tea.Cmd`, like Plan, Apply and State (ADR-0021), routed by the typed
      dispatch in `internal/ui/app.go`. Delete `ActivateInput`; `cmd/tfui/risk_command.go` uses the typed path. The
      no-input `Activate()` for TUI navigation stays. Closes #122.
- [ ] `resetDerived` no longer calls `Cancel`, whose effect it overwrites.
- [ ] `requestAssessment` records the raw Context; `report.go` resolves the report context and its chdir fallback.
- [ ] Stdout, Stderr, exit codes and the JSON report are byte-identical for every existing test, including
      `cmd/tfui/{risk_command,risk_report,session_risk}_test.go`.

### US-012: Share instance ordering and row counts in the SDK

**Description:** As a maintainer, I want instance ordering and assessment counts defined once, so that Risk's view,
report and stderr cannot drift apart.

**Acceptance Criteria:**

- [ ] Add `sdk.InstanceKey.Compare` (address, then deposed key). The test-writer writes its test first.
- [ ] Use it for the three address-then-deposed sorts in `plugins/risk` (`compareChanges`, the report sort and any
      sort left in `Stderr` after US-011). Delete `deposedKey` if nothing else needs it.
- [ ] Add one counts method on `sdk.PolicyAssessment` returning classified, unavailable and not-assessed counts. The
      test-writer writes its test first.
- [ ] Use it in `renderOverallBanner`, `excludedCount` (delete it) and the counts in `report()`.
- [ ] Blast Radius and Plan adopt either helper only where they repeat the same loop; the report lists each call site
      changed.
- [ ] Output is unchanged: the same tests as US-011 keep their assertions.

### US-013: Land PR #131

**Description:** As the maintainer, I want #131 merged once it is clean, so that `main` gets #129, #130 and US-004 to
US-012.

**Acceptance Criteria:**

- [ ] Update the PR #131 description: list US-004 to US-012 with their commits and each quality gate's result.
- [ ] Before merging, a review agent reviews the whole #131 diff against `main`, with no Critical or Important
      findings open.
- [ ] CI on #131 is green.
- [ ] Merge #131 into `main` with a merge commit, matching #121.
- [ ] After the merge, `origin/main` contains `d812051` (the #130 merge) and `8e7fe5f` (#129's head). `dc3abec`,
      #129's merge commit, lives only on `agent/risk-plugin-pr` and is not expected on `main`.
- [ ] Issues #122, #127 and #128 are closed. Close them by hand if the closing keywords did not fire.
- [ ] Close #126, resolved by US-003, and the epic #120, which #121 delivered, each with a comment naming the
      commit or PR.
- [ ] Edit #124 to remove items 2 and 4, fixed by US-011 and US-005, and leave its other items open.
- [ ] Do not delete branches or worktrees; cleanup waits for the maintainer.

## Functional Requirements

- FR-1: A `planJSONMsg` from a run that is not current must not change Plan's state.
- FR-2: A partly unknown map or object attribute must render its known keys, with only unknown leaves shown as
  `(known after apply)`, in both the full and the compact diff. Sensitive masking stays whole-value.
- FR-3: The policy loader must not inspect or reject symlinks.
- FR-4: The risk-policy contract must not mention symlinks.
- FR-5: Cancelling one plugin must not drop, delay or cancel another plugin's pending assessment.
- FR-6: Cancelling a plugin must cancel its own pending assessment's context, both the host-run plan and the policy
  evaluation.
- FR-7: A Context change or plan invalidation must cancel and drop every pending assessment.
- FR-8: With a non-nil summary, a cache hit must return without compiling the policy. Config and source must still be
  reloaded at every assessment boundary.
- FR-9: A broken policy must never run `terraform plan`.
- FR-10: Applying an assessment must build its instance index once.
- FR-11: No exported or package-level production function may exist only for tests, and no method may return a
  constant only because tests call it. Unexported seams inside a package, used by that package's tests, are allowed.
- FR-12: Stale-pin cleanup after a plan must apply while Plan's assessment is pending.
- FR-13: The Risk plugin must look up each assessment row once per result and must not copy the plan summary to
  render it.
- FR-14: The Risk plugin's outcome must be set in one place; View, Stderr and ExitCode must read it, not derive it.
- FR-15: The Risk restructuring must not change any TUI frame, stdout, stderr, exit code or JSON report, except the
  unusable-assessment diagnostic, which must name a code from the contract's list.
- FR-16: A request cancelled before it runs must not start `terraform plan`.

## Non-Goals

- No ADR, superseding note or "changed from" history in docs. Mistakes are fixed in place.
- No in-flight deduplication of identical evaluations (sharing one running evaluation between requests).
- No change to built-in policy severities, the fixed classifier, the report schema or `policy.id`.
- No change to `tfui plan --json` output, which stays raw Terraform JSON.
- No deletion of branches, worktrees, backup branches or scratch files.
- No fix for `childCfg` going stale on a return to the root chdir. It predates these PRs; it is filed separately if it
  is not filed already.

## Technical Considerations

- **Branches:**
  - US-001 to US-003 run on a `main` checkout and push straight to `origin/main`. Each `fix:` commit makes
    semantic-release cut a patch release.
  - US-004 to US-013 run on `.worktrees/refactor/assessment-coordinator`. Pull `origin/refactor/assessment-coordinator`
    first; it must contain the #130 merge `d812051`.
- **Order:** stories run one after another, in ID order. US-005 and US-006 tests use the in-package seam, not
  `RequestWith`, which US-008 removes. US-010 to US-012 run after US-007, so Risk's `ApplyAssessment` call is removed
  once, not changed and then removed.
- **Relevant files:**
  - `internal/riskassess/coordinator.go` (#129) and `internal/ui/app.go`, with `cancelPlugin` and `requireIdle`.
  - `pkg/sdk/plugin_deps.go`, for the `RequestAssessment` contract.
  - `plugins/{plan,risk,blastradius}/assessment.go` and `lifecycle.go`.
  - `pkg/sdk/sdktest/testdeps.go`, the shared fake.
- **Contract:** `docs/reference/risk-policy-contract.md` still binds the loader, the policy reload and the
  re-verification before publishing.

## Success Metrics

- All ten review findings and the Risk plugin findings are resolved.
- Every quality gate passes on `main` after US-001 to US-003, and on #131 before it merges.
- CI on `main` is green after #131 merges.
- `BenchmarkApplyAssessment` reports fewer allocs/op than before US-007.

## Open Questions

None. Every decision was settled in the review grilling.
