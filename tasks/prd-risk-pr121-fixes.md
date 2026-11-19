[PRD]
# PRD: Fix PR #121 risk-assessment findings before merge

## Overview
PR #121 (`agent/risk-plugin-pr`) adds policy-based risk assessment to tfui: an embedded Rego evaluator, policy input built from the plan JSON, Risk, Plan and Blast Radius views, and the `tfui risk` command. Two Codex reviews, an xhigh `/code-review` and a code-quality review found two kinds of problems.

Correctness bugs, reproduced with hand-written plan JSON:
- Any plan containing a `delete` makes the whole plan's policy input `input_invalid`.
- `replace_paths` is dropped before the display parse, so "(forces new)" never appears.
- A deposed object and the current object at the same address get each other's levels and reasons.
- The Terraform `forget` action fails the whole plan.
- An import with a known ID fails the whole plan.
- In the Plan view, toggling a pin resets the cursor and tree, and re-entering the view shows a fake plan run.

Structural duplication:
- Plan, Risk and Blast Radius each run their own copy of the assessment lifecycle.
- There is a second `tfui.hcl` parser.
- `PolicyDiagnostic` does synchronous disk I/O on the UI thread.
- Statuses and error codes are plain strings.
- Each plan is decoded, re-encoded and decoded again.

This PRD fixes all of them on the PR #121 branch before it merges. Decisions already made:
- The deposed key is exposed to policy authors and in the JSON report.
- `forget` is excluded as "not assessed".
- A known import ID is passed to policies as a redacted `sensitive` value.
- The CLI flag `-threshold` is renamed to `-risk-threshold`.

## Goals
- Every plan in the regression fixtures (delete, replace, deposed, forget, import with known ID) produces a usable assessment, with no whole-plan failure.
- Each resource instance (address + deposed key) gets its own level, reasons and failure in Plan, Risk, Blast Radius and the JSON report.
- One host-owned assessment lifecycle replaces the three plugin copies. Plugins only render `(summary, assessment)`.
- No synchronous risk disk I/O in any plugin `Update`, `Activate`, `View`, `Stdout` or `Stderr` path.
- Typed statuses and codes, with byte-identical string values.
- One `tfui.hcl` parser, still reloaded at every assessment boundary as the contract requires.
- The display parse never re-encodes the plan JSON.
- Every behavior currently pinned by these tests keeps passing, except the threshold flag's name and scope:
  - `cmd/tfui/{risk_command,risk_report,session_risk,session_plan}_test.go`
  - the five risk macro tapes

## Quality Gates

These commands must pass for every user story:
- `mise run check:lint` - Lint (golangci-lint)
- `mise run test:unit` - Unit tests
- `mise run check:build` - Compile

UI stories touch `View()`, hints, frames, key handling or rendered text in `plugins/plan`, `plugins/risk` or `plugins/blastradius`. They must also pass:
- `mise run build && mise run test:macro` - Macro tapes against the built binary

## Assumptions (unverified; a story that depends on one must check it first)
- **A-1:** Terraform emits `"after": null` with `"after_unknown": {}` for deletes. Only checked with hand-written JSON. US-018 confirms it with real `terraform show -json`.
- **A-2:** Some valid plan JSON fails a direct `tfjson.Plan` decode, which is why the PR strips policy-only fields before the display parse. Not reproduced. US-002 checks it.
- **A-3:** Existing tests do not pin "Risk activation with a broken policy never runs Terraform". US-009 adds the test either way.

## User Stories

Rules for every story:
- Work in the ralph-tui worktree on a branch off `agent/risk-plugin-pr`.
- Write the failing test first, as AGENTS.md requires: the `test-writer` agent produces it before any implementation edit.
- A bug fix includes a test that fails before the fix.
- Use conventional commits.
- Change only what the story needs. Keep existing behavior and on-screen strings unless the story says otherwise.
- Remove only code that the story's own change leaves unused.

### US-001: Delete changes produce valid policy input
**Description:** As a user reviewing a plan that destroys resources, I want risk assessed so that a delete does not make the whole plan `input_invalid`.

**Acceptance Criteria:**
- [ ] A new test calls `terraform.LoadPlan` on a plan with a `["delete"]` change shaped `"after": null`, `"after_unknown": {}`, `"after_sensitive": false`. It asserts that `PolicyEvidence.ErrorCode == ""` and that the deleted resource has an `Input` with `After.Kind == "absent"`. It fails before the fix.
- [ ] `projectPolicyValue` (`internal/terraform/policy_evidence.go`) accepts an empty object or empty array mask on a null or scalar value as "not marked". A non-empty mask on a scalar is still rejected as `input_invalid` (test).
- [ ] `TestCompat_DestroyModeRiskClassification` (`tests/integration/compat_test.go`) gets a fixture with `"mode":"managed"` and passes.

### US-002: Replacement paths survive plan loading
**Description:** As a user inspecting a replacement, I want to see which attributes force the replacement so that I understand why a resource is destroyed.

**Acceptance Criteria:**
- [ ] A new test calls `LoadPlan` on a `["delete","create"]` change with `replace_paths: [["ami"]]`. It asserts that the `ami` `AttributeDiff` has `ForcesNew == true`, and fails before the fix.
- [ ] Check A-2 first. Try to write a test with plan JSON that a direct `json.Unmarshal` into `tfjson.Plan` rejects but that has usable display changes, for example a malformed `importing` or `deferred_changes`.
  - **If such input exists:** `LoadPlan` decodes the original bytes into `tfjson.Plan` directly. Only if that fails does it decode a copy with `replace_paths`, `importing` and `deferred_changes` removed. The test proves the fallback still yields a display summary with evidence `ErrorCode == "input_invalid"`.
  - **If no such input exists:** `LoadPlan` decodes the original bytes directly, with no fallback and no field removal. The story's commit message records the inputs that were tried.
- [ ] The plan JSON is never re-marshalled.
- [ ] Evidence is still built from the `UseNumber` map decode of the original bytes.
- [ ] `PlanID` is still the `sha256` of the original bytes, and `cmd/tfui/session_plan_test.go` passes unchanged.

### US-003: `forget` action is excluded, not invalid
**Description:** As a user with a `removed { lifecycle { destroy = false } }` block, I want that resource marked "not assessed" so that the rest of the plan is still classified.

**Acceptance Criteria:**
- [ ] A test calls `LoadPlan` on a plan with one `["forget"]` change and one `["update"]` change, and fails before the fix. It asserts:
  - the evidence `ErrorCode` is empty
  - the forget row has `Exclusion == "forget"` and no `Input`
  - the update row has an `Input`
- [ ] `policyActions` and `policyResourceEvidence` (`internal/terraform/policy_evidence.go`) recognize `forget`. `sdk.ActionFromStrings` maps `forget` to `ActionNoOp` today (`pkg/sdk/plan_json_test.go`), so the check must use the raw action string.
- [ ] `docs/reference/risk-policy-contract.md` lists `forget` among the exclusions.
- [ ] The `tfui risk` text output shows `not assessed (forget)` for the forget row.

### US-004: Known import ID is passed redacted
**Description:** As a user importing a resource with a known ID while changing it, I want it classified so that one import doesn't make risk unavailable for the whole plan.

**Acceptance Criteria:**
- [ ] A test calls `LoadPlan` on a change with `"importing": {"id": "i-123"}` and actions `["update"]`, and fails before the fix. It asserts that the evidence `ErrorCode` is empty and that `Input.Change.Importing.ID.Kind == "sensitive"` with no value.
- [ ] The string `i-123` appears nowhere in the marshalled `PolicyInput` (asserted).
- [ ] `docs/reference/risk-policy-contract.md` says a known import ID is passed as kind `sensitive`.

### US-005: Deposed key flows into the plan model and policy input
**Description:** As a policy author, I want to tell a leftover deposed object apart from the current object at the same address so that I can classify its cleanup separately.

**Acceptance Criteria:**
- [ ] `sdk.Resource` (`pkg/sdk/resource.go`) has `Deposed string`, which `ParsePlan` sets from `resource_changes[].deposed`.
- [ ] `sdk.PolicyResource` has `Deposed *string` with JSON key `deposed`: null for the current object, the key for a deposed one.
- [ ] `PolicyResourceEvidence` carries the same key.
- [ ] A test calls `LoadPlan` on a plan with `a.b` (update) and `a.b` deposed `00000001` (delete). It asserts two display changes and two evidence rows that differ by deposed key.
- [ ] `docs/reference/risk-policy-contract.md` documents `resource.deposed` in `tfui.risk.input/v1`.
- [ ] `examples/risk-policy/lambda/fixtures.json` includes `deposed: null` wherever the schema requires it.

### US-006: Typed status, exclusion, code and kind constants
**Description:** As a maintainer, I want typed constants in `pkg/sdk` so that plugins stop comparing raw strings.

**Acceptance Criteria:**
- [ ] `pkg/sdk` defines typed string constants for:
  - assessment status: `complete`, `empty`, `partial`, `unavailable`, `cancelled`
  - resource status: `classified`, `not_assessed`, `unavailable`
  - exclusions, including `forget`
  - error codes
  - policy value kinds
- [ ] Every value is byte-identical to today's string (a table test asserts each constant's string).
- [ ] `PolicyAssessment`, `PolicyResourceAssessment`, `PolicyResourceEvidence` and `PolicyValue` fields use these types, and all producers and consumers compile.
- [ ] `internal/riskpolicy` returns sentinel errors that carry an error code. No caller turns `err.Error()` into a code.
- [ ] `grep -rnE '"(unavailable|classified|not_assessed|complete|partial)"' plugins internal cmd pkg --include='*.go' | grep -v _test.go` matches only the constant definitions.

### US-007: Shared assessment helpers in the SDK
**Description:** As a maintainer, I want one lookup, one level parser and one projection so that plugins stop re-implementing them.

**Acceptance Criteria:**
- [ ] `sdk.ParseRiskLevel(string) (RiskLevel, bool)` replaces the level parsing in `internal/riskpolicy/result.go`, in `plugins/risk/report.go` (the threshold map) and in `pkg/sdk/risk_settings.go`.
- [ ] `PolicyAssessment.Usable() bool` returns true for `complete` and `empty`, and replaces every `Status != complete && Status != empty` check.
- [ ] `sdk.InstanceKey{Address, Deposed}` exists. `PolicyAssessment.Resource(InstanceKey) (PolicyResourceAssessment, bool)` is the only per-resource lookup.
- [ ] `sdk.ApplyAssessment(*PlanSummary, *PolicyAssessment) *PlanSummary` returns a copy with each change's `Risk` set from its classified row, matched by instance key. Unclassified rows get `RiskNone`. The input summary is not mutated (asserted).
- [ ] Unit tests cover each helper, including two rows that share an address and differ by deposed key.

### US-008: All consumers match rows by instance key
**Description:** As a user with a deposed object in my plan, I want each instance's own level, reasons and failure shown in every view and in the report.

**Acceptance Criteria:**
- [ ] Every per-resource lookup goes through `PolicyAssessment.Resource(InstanceKey)` or `sdk.ApplyAssessment`. This covers `plugins/plan/assessment.go`, `plugins/risk/{risk,lifecycle,frames,report}.go` and `plugins/blastradius/assessment.go`. No `row.Address ==` comparison is left.
- [ ] JSON report rows (`plugins/risk/report.go`) include `deposed` (string or null). Rows sort by address, then deposed key. The text output shows the deposed key next to the address.
- [ ] A fixture-driven test has an update and a deposed delete at the same address, with policy levels `low` and `high`. It asserts:
  - Plan shows both badges correctly.
  - Risk groups each instance under its own level with its own reasons.
  - Blast Radius lists each with its own status.
  - `tfui risk -json` has two distinguishable rows.
- [ ] `docs/reference/risk-policy-contract.md` documents the report's `deposed` field.

### US-009: Host-owned assessment request and event
**Description:** As a maintainer, I want one assessment lifecycle in the host so that plugins don't each manage plan fetching, generations, cancellation and evaluation.

**Acceptance Criteria:**
- [ ] `pkg/sdk` adds:
  - `PluginDeps.RequestAssessment func(summary *PlanSummary) tea.Cmd`
  - `RiskAssessedEvent{Summary *PlanSummary, Assessment PolicyAssessment, Err error}`
  - a `RiskAssessedHandler` interface
  - dispatch of the event through the existing `EventBus` (`pkg/sdk/bus.go`)
- [ ] `internal/ui/risk_policy.go` implements the request, entirely inside the returned Cmd (off the UI thread):
  1. Capture the current Context.
  2. Resolve the selection and load the source. On failure, return an unavailable assessment with that code and do not call `Service.Plan`.
  3. If `summary` is nil, get the plan through the captured Context's `Service.Plan` with a temp plan file. This logic moves from `plugins/risk/lifecycle.go`. On plan failure, set `Err` and skip evaluation.
  4. Evaluate. Re-check the selection and source before publishing, keeping the existing `source_changed` behavior.
  5. Return a message that the App dispatches as `RiskAssessedEvent`.
- [ ] The App keeps one generation counter and one cancel function behind the existing pointer-shared state pattern, and reuses `riskAssessmentCache`.
- [ ] `ContextChangedEvent` and `PlanInvalidatedEvent` advance the generation. A result from an older generation is dropped (test).
- [ ] Two tests pin the request behavior:
  - A broken policy with a nil summary never calls `Service.Plan`. The fake service fails the test if it is called.
  - A valid policy with a nil summary calls `Service.Plan` exactly once, and the event carries that summary.
- [ ] `pkg/sdk/sdktest` provides a fake `RequestAssessment` that returns a canned event, for plugin tests.
- [ ] `internal/ui/risk_assessment_test.go`, `risk_assessment_race_test.go` and `risk_policy_test.go` are rewritten against the request API. They keep every behavior they pinned: the captured Context, mid-session `tfui.hcl` edits and the race.

### US-010: Plan plugin renders from the host assessment
**Description:** As a user in the Plan view, I want risk badges to update without the cursor jumping, without false "running plan" states and without lost results.

**Acceptance Criteria:**
- [ ] `plugins/plan` no longer has:
  - the fields `assess`, the assessment `generation`, `rawSummary` and `riskDiagnostic`
  - `planAssessedMsg`, `reassess`, `invalidateAssessment` and `PlanResultMsg.Assessment`
- [ ] The Plan plugin requests assessments through `RequestAssessment` and handles `RiskAssessedEvent`.
- [ ] One method, for example `riskCell(change)`, produces the badge or diagnostic for a row. The flat rows, the tree leaf, the inspect content and the overall line all use it.
- [ ] Regression tests that fail on the current branch:
  - (a) Toggling a pin on row 3 keeps the cursor on row 3 and the tree expansion unchanged. No row shows `risk unavailable: input_incomplete` while the reassessment is pending.
  - (b) Re-entering a finished Plan view never shows `Running terraform plan`. Esc during reassessment leaves the plan Done, and the next entry does not call `Service.Plan` again.
  - (c) `PlanInvalidatedEvent` while Idle or Error does not change the status to Done.
- [ ] `Busy()` stays true until the assessment for the current plan arrives, so headless `tfui plan` cannot print `Stdout`/`Stderr` before it (test through the macro driver).
- [ ] Update `cmd/tfui/session_risk_test.go` (which reads `PlanResultMsg` at lines 180 and 217) and `cmd/tfui/session_plan_test.go` (line 78) to read the assessment from `RiskAssessedEvent` or the plugin. Each test keeps the behavior it asserts.
- [ ] `plugins/plan/plan.go` stays under its current 1,175 lines.

### US-011: Risk plugin renders from the host assessment
**Description:** As a user of the Risk view and of `tfui risk`, I want the same results as today, with no plan fetching or lifecycle code in the plugin.

**Acceptance Criteria:**
- [ ] `plugins/risk/lifecycle.go` no longer calls `Service.Plan`, creates a temp dir, or keeps its own evaluation `cancel` or `generation`. Activation calls `RequestAssessment(retainedSummary)`, where the summary may be nil.
- [ ] Results arrive through `RiskAssessedEvent`. `acceptAssessment` uses `sdk.ApplyAssessment`. A plan failure (`Err`) shows as it does today.
- [ ] `checkPolicySource` and every `PolicyDiagnostic()` call are removed from `plugins/risk`. A policy failure on activation with no plan shows `risk unavailable: <code>`, and the existing `cmd/tfui/risk_command_test.go` case passes.
- [ ] The macro tapes `tests/fixtures/tapes/macro/{risk-policy-unavailable,risk_empty,risk_none,risk_partial,risk_render}.tape` pass with unchanged assertions.

### US-012: Blast Radius renders from the host assessment
**Description:** As a user of Blast Radius, I want it to show the shared assessment with no lifecycle code of its own.

**Acceptance Criteria:**
- [ ] `plugins/blastradius/assessment.go` no longer defines `blastRadiusAssessedMsg`, `assess`, `generation` or `cancel`.
- [ ] Blast Radius requests through `RequestAssessment` and handles `RiskAssessedEvent`, using `sdk.ApplyAssessment` and `PolicyAssessment.Resource`.
- [ ] Unavailable modules still sort first. Module rows still show `[unavailable]` or `Risk: not assessed` exactly as today.
- [ ] The rendered-text assertions in `plugins/blastradius/assessment_test.go` and `assessment_lifecycle_test.go` are unchanged.
- [ ] The smoke tapes `tests/fixtures/tapes/smoke/risk_consumers.tape` and `risk_lifecycle.tape` pass.

### US-013: Remove the synchronous policy diagnostic from the SDK
**Description:** As a maintainer, I want no risk disk I/O on the UI thread and no duplicate diagnostic channel.

**Acceptance Criteria:**
- [ ] Delete `PluginDeps.RiskPolicyDiagnostic`, `PluginDeps.AssessRisk`, `PluginBase.PolicyDiagnostic`, `App.riskPolicyDiagnostic` and `pkg/sdk/plugin_base_policy_test.go`.
- [ ] Plan's headless `Stderr` prints `risk unavailable: <code>` from the assessment's error code. The "fixed diagnostic" and "policy source failure keeps plan review" cases in `cmd/tfui/session_plan_test.go` pass.
- [ ] `grep -rn 'PolicyDiagnostic\|AssessRisk' --include='*.go' .` returns nothing.

### US-014: One `tfui.hcl` parser for `risk_policy`
**Description:** As a maintainer, I want `risk_policy` parsed by the existing config loader so that `tfui.hcl` has one parser, still reloaded at each assessment boundary.

**Acceptance Criteria:**
- [ ] `DefaultsConfig`, `ChildConfig` and `WorkspaceConfig` (`internal/config/hcl_types.go`) gain `RiskPolicy *PolicySelector{Value string; Origin string; Invalid bool}`, which `LoadRoot` and `LoadChild` populate.
- [ ] These values set `Invalid: true` and **never make `LoadRoot` or `LoadChild` return an error**: null, non-string, unknown, URI (`://`), whitespace-only and duplicate. `cmd/tfui/root.go:42-48` aborts startup on any `LoadRoot` error. A test checks that a root `tfui.hcl` with `defaults { risk_policy = null }` still starts `tfui plan` and that Plan review works.
- [ ] `config.Resolve` layers `RiskPolicy` from defaults, then member, then workspace, using `matchWorkspace` (`internal/config/hcl_resolve.go`).
- [ ] `ResolveRiskPolicy` becomes thin: it loads root and child from disk, resolves, applies the root-member rule, and maps an `Invalid` selector or a load failure to `configuration_invalid`.
- [ ] `policyConfigBody` and `policySelector` are deleted.
- [ ] Every case in `internal/config/risk_policy_test.go` passes with unchanged expected values:
  - layering, glob precedence, the `""` reset, literal spaces and absolute paths
  - the invalid-input table, directory-as-file, a missing member and a deleted directory
  - var blocks and zero-config
- [ ] A malformed member `tfui.hcl` still never blocks Plan; only risk reports `configuration_invalid` (existing test).

### US-015: Evidence and parser cleanups
**Description:** As a maintainer, I want the duplicated and misnamed helpers this PR introduced removed.

**Acceptance Criteria:**
- [ ] `markedPolicyMask` (`internal/terraform/policy_evidence.go`) is deleted. Its callers use `markedValue` (`internal/terraform/plan_parser.go`).
- [ ] The unknown-value check in `parseAttributeDiffs` uses a helper whose name doesn't say "sensitive", for example `maskMarked`. `isKeySensitive` is used only for sensitivity.
- [ ] The unreachable `source_invalid` branch in `prepare` (`internal/riskpolicy/evaluator.go`) is removed. Coverage for the package stays at 100%.
- [ ] A test asserts that the type lists in `internal/riskpolicy/builtin.rego` match `criticalTypes`, `highRiskTypes` and `mediumRiskTypes` in `internal/terraform/risk.go`.

### US-016: `-risk-threshold` flag scoped to `tfui risk`
**Description:** As a CLI user, I want the threshold flag named `-risk-threshold`, available only where it matters, with correct help text.

**Acceptance Criteria:**
- [ ] The flag is registered as `risk-threshold`, as a local flag on the `risk` command (`cmd/tfui/risk_command.go`) instead of a root persistent flag (`cmd/tfui/root.go:88`). `Session.threshold` is removed.
- [ ] Remove `TestRoot_GivenThresholdFlag_ShouldExposeItForEveryCommand` (`cmd/tfui/risk_command_test.go:83`). It pins the persistent scope, which this story deliberately changes, so removing it does not weaken a check. Its replacement asserts:
  - `risk-threshold` is a local flag of `risk`.
  - The root has no `threshold` or `risk-threshold` persistent flag.
- [ ] `tfui risk -threshold high` and `tfui plan -threshold high` are both rejected as unknown flags (test).
- [ ] `tfui risk -risk-threshold high` and `tfui risk --risk-threshold high` behave exactly as `-threshold high` does today. `cmd/tfui/normalize.go` already turns single-dash multi-letter flags into double-dash. Both forms keep the same comparison, the same exit code 3 when met and the same `risk threshold met: <level> >= <threshold>` stderr line.
- [ ] `tfui risk --help` lists `--risk-threshold` with the values `off, none, low, medium, high, critical`, and shows the default `off` exactly once.
- [ ] `risk.Input` uses `Threshold *string`, and `ThresholdSet` is removed. A set flag overrides the `tfui.hcl` `plugin "risk" { threshold = ... }` value, as today. The config key name `threshold` is unchanged.
- [ ] Every current mention of the flag is renamed:
  - `cmd/tfui/risk_command_test.go`
  - `cmd/tfui/risk_report_test.go`
  - `docs/plugins/risk.md`
  - `docs/reference/cli-reference.md`
  - `docs/reference/risk-policy-contract.md`
  - Docs keep their single-dash style (`-risk-threshold`).
- [ ] `grep -rnE '(^|[^a-zA-Z-])--?threshold\b' --exclude-dir=.git --exclude-dir=.worktrees --exclude-dir=tasks .` returns nothing. The config key `threshold = ...` and the prose "risk threshold met" do not match this pattern.
- [ ] The literal `9` in `RenderCompactAttributeDiffs` (`pkg/sdk/attribute_diff.go:44`) becomes a named constant that says what it accounts for. Rendering output is unchanged (existing tests).

### US-017: Documentation and example housekeeping
**Description:** As a maintainer, I want the PR's docs to follow the project's rule against documenting history.

**Acceptance Criteria:**
- [ ] Two lines describe how the `level` key used to work. Reword each to state only the current behavior: `level` is rejected with an error that names `threshold`.
  - `.agents/skills/tfui-cli-ux/SKILL.md:120` ("migration from the formerly ignored `level` key")
  - `docs/plugins/risk.md:98` ("The former ignored `level` setting now produces a migration error")
- [ ] `examples/risk-policy/lambda/main.rego` has no trailing blank line, and `git diff --check origin/main...HEAD` reports nothing.
- [ ] `AGENTS.md` is updated if any command, path or rule it names changed.

### US-018: Final verification with real Terraform fixtures
**Description:** As the PR owner, I want proof against real Terraform output that the whole branch is healthy before merge.

**Acceptance Criteria:**
- [ ] The delete, forget and known-ID-import fixtures under `tests/fixtures/` come from real `terraform show -json` output:
  - Produced with the repo's terraform 1.14 (`mise.toml`) on `terraform_data` resources, with local state only.
  - The generation steps are recorded next to the fixtures.
  - If the real output contradicts A-1, fix the US-001 tests to match the real shape and note it in the PR comment.
- [ ] The deposed fixture is hand-written, and the PR comment says so.
- [ ] `tfui risk -ci -json -plan <fixture>` exits 0 or 3, never 1, for each of the four fixtures, with the expected rows (integration test).
- [ ] These all pass:
  - `mise run test:coverage` (at 100%)
  - `mise run test:integration`
  - `mise run build && mise run test:macro`
- [ ] The `arch-checker`, `tui-checker` and `cli-checker` agents report no violations for the changed files.
- [ ] A comment on PR #121 lists each finding (US-001 to US-017) with the commit that fixed it, plus every gate's result.

## Functional Requirements
- **FR-1:** `LoadPlan` must produce policy evidence without a whole-plan error for any plan containing only these actions: no-op, create, read, update, delete, delete-then-create, create-then-delete and forget.
- **FR-2:** A `delete` change's `after` must appear in the policy input as kind `absent`.
- **FR-3:** For plans loaded through `LoadPlan`, `AttributeDiff.ForcesNew` must be true for every attribute named as the first element of a `replace_paths` entry.
- **FR-4:** A `forget` change must be reported as not assessed, with exclusion `forget`.
- **FR-5:** A known import ID must reach policies only as kind `sensitive`, with no value.
- **FR-6:** Resource identity across evidence, assessment, views and the report must be the pair (address, deposed key).
- **FR-7:** The policy input must include `resource.deposed` (string or null), and report rows must include `deposed`.
- **FR-8:** Only the host may evaluate policies or fetch a plan for assessment. Plugins receive results only through `RiskAssessedEvent`.
- **FR-9:** A result must be discarded if a Context change or a plan invalidation happened after its request.
- **FR-10:** The policy configuration and source must be re-read at every assessment boundary: the initial assessment, a Context change, plan completion, a refresh and Risk activation. They must be re-checked before a result is published (contract `risk-policy-contract.md:395-413`).
- **FR-11:** No plugin `Update`, `Activate`, `View`, `Stdout` or `Stderr` may read policy configuration or source from disk.
- **FR-12:** Every status, exclusion and error-code string shown on screen or in JSON must stay byte-identical to the current branch.
- **FR-13:** A policy or configuration failure, including an invalid `risk_policy` value in any `tfui.hcl`, must leave Plan review available, with risk marked unavailable and the code visible.
- **FR-14:** When the selected policy cannot be loaded, an assessment request must not run Terraform.
- **FR-15:** `-risk-threshold` is the only CLI flag that sets the risk threshold, and only `tfui risk` accepts it. Its value set and default (`off`) are unchanged.

## Non-Goals
- Fixing `childCfg` going stale when switching back to the root chdir (`internal/ui/app.go:298-304`). This predates the PR; file a GitHub issue.
- Relaxing the contract rule that rejects symlinked policy paths or parent directories. File a GitHub issue for discussion.
- Removing the double read of the policy source or the re-check before publishing; the contract requires both.
- Changing the built-in policy severities or the fixed classifier.
- Changing `tfui plan -json` output, which stays raw Terraform JSON.
- Renaming the `threshold` key inside the `plugin "risk"` block of `tfui.hcl`.
- Keeping `-threshold` as a deprecated alias; it has never been released.
- Refactoring code this PR did not add or change.
- Benchmarking or any performance work beyond removing the re-encode.

## Technical Considerations
- Headless runs use `macro.Driver`, which runs Cmds synchronously and depth-first (`internal/macro/driver.go:82-100`). The host event must be dispatched in time for `WaitUntil(Ready || terminal)` (`cmd/tfui/session.go:296-305`). The plan plugin's `Busy()` must cover a pending assessment.
- Pre-seeded `-plan` sessions and headless `tfui risk` never publish `PlanCompletedEvent`. Assessment is triggered by `RequestAssessment`, never by that event.
- `cmd/tfui/root.go:42-48` aborts on any `LoadRoot` error other than not-found. This is why US-014 must record an invalid `risk_policy` as data instead of returning an error.
- `cmd/tfui/normalize.go` rewrites single-dash multi-letter flags to double-dash before Cobra parses them. No change is needed there for `-risk-threshold`.
- The App uses value receivers. Mutable host state must live behind a pointer, as `contextHolder` and `riskAssessmentCache` already do.
- `ExecService.WithDir` creates an empty cache, so a seeded plan is visible only through the startup service (`app.go:853-862`). The host's plan fetch in US-009 uses the captured Context's `Service`, which preserves this.
- CI enforces 100% coverage outside `cmd/` and `internal/terraform/exec`, so every new branch needs a test.
- Plugins import only `pkg/sdk`; `internal/` belongs to the app.
- About 20 test files fake `AssessRisk` or `PolicyDiagnostic` today. They should use the `sdktest` fake from US-009 instead of inline closures.

## Success Metrics
- Zero whole-plan `input_invalid` or `input_incomplete` results on the four fixture plans, three of them generated by real Terraform.
- Three plugin lifecycles reduced to one. No plugin defines an assessment message type or calls `Service.Plan` for assessment.
- No string-literal status comparisons outside the constant definitions.
- All existing risk end-to-end tests and macro tapes pass with unchanged assertions, except the flag rename and the flag-scope test.
- The total non-test line count under `plugins/{plan,risk,blastradius}` decreases compared with the current branch.

## Open Questions
- Should the deposed key also appear in Plan list rows, for example `aws_instance.web (deposed 00000001)`, or only in inspect views and the report?
- Should `forget` rows count toward the Risk view's "not assessed" total, like `read` and `no_op`, or be hidden?
- If `tfui plan` later reports risk against a threshold, should it accept the same `-risk-threshold` flag? This PRD keeps the flag on `tfui risk` only.
[/PRD]
