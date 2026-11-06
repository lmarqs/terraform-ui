[PRD]
# PRD: Finish the Risk plugin (run first of three)

## Overview

`plugins/risk` (key `R`) groups a plan's changes by risk level, but it has never shown a result
outside its tests. Its `Analyze(*sdk.PlanSummary)` has no production caller. The plan plugin
publishes `sdk.PlanCompletedEvent` (`plugins/plan/plan.go:421-428`), and no plugin implements
`sdk.PlanCompletedHandler`. `app.activate()` (`internal/ui/app.go:897-923`) has no case for risk.
So `R` always shows the idle "Run a plan first" text. GitHub issue #76 records this.

Risk also lacks the headless side other read plugins have: `Input`, `Activate(Input)`, `Stdout`,
`Stderr`, `ExitCode` and a `cmd/tfui/<id>_command.go`. `Configure` ignores the
`plugin "risk" { level = ... }` option that `tests/integration/config_test.go:140` expects. Its docs
promise `risk` fields in `tfui plan -json` that do not exist, and they state rules that differ from
`internal/terraform/risk.go`.

This is the first of three PRDs, run in this order: risk, phantom, blast radius. It also carries the
groundwork the other two depend on:
- the exec service serves a pre-seeded `-plan`
- plugin `Configure` errors stop tfui
- one engine function annotates a plan
- the ADR for tfui-native JSON

## Goals

- `R` shows a filled risk view after `p`, or with a pre-seeded `-plan`, without a second terraform run when a plan was already shared.
- `tfui risk` follows the CLI I/O contract, and `tfui risk -ci` exits 2 when overall risk is at or above `level`, which defaults to `high`.
- `tfui <cmd> -plan file` reads the seeded plan without running terraform.
- Phantom changes carry risk `none`, so a no-op never trips the gate.
- Risk docs match the code.

## Quality Gates

These commands must pass for every user story:
- `mise run check:lint` - golangci-lint, repo-wide
- `mise run test:unit` - Go unit tests
- `mise run check:build` - compile check
- `mise run test:coverage` - CI enforces 100% coverage outside `cmd/` and `internal/terraform/exec`

For UI stories (US-006, US-008, US-009), also include:
- `mise run test:macro` - macro tape tests verifying TUI rendering

## Working rules for every story

- Work in the worktree `.worktrees/fix/analysis-plugins` on branch `fix/analysis-plugins`.
- Follow `AGENTS.md` and the `tfui-conventions`, `tfui-architecture`, `tfui-cli-ux` and `tfui-tui-ux` skills in `.agents/skills/`.
- Write the failing test first, then the implementation. A bug fix ships with a test that failed before the fix.
- Plugins import only `pkg/sdk`. Plugin tests build deps with `sdktest.NewDeps(svc)` and call `Init()`, as `plugins/version/version_test.go` does.
- Tests are table-driven and sit beside the code. Golden files live in `testdata/golden/` and are refreshed through `sdktest.AssertGolden`'s update flag.
- Commit each story as a conventional commit and push it to `origin fix/analysis-plugins`.
- Never weaken a check to make it pass.

## User Stories

### US-001: Parse a risk level from text
**Description:** As the risk plugin, I want to turn a config string into an `sdk.RiskLevel` so that `level` can set the CI threshold.

**Acceptance Criteria:**
- [ ] `pkg/sdk/risk.go` exports `ParseRiskLevel(s string) (RiskLevel, error)` next to `RiskLevel.String()`.
- [ ] `"none"`, `"low"`, `"medium"`, `"high"` and `"critical"` each map to their constant and round-trip through `String()`.
- [ ] Any other value, including `""` and `"HIGH"`, returns an error that quotes the value and lists the accepted ones.
- [ ] A table test in `pkg/sdk/risk_test.go` covers each accepted value and at least two rejected ones.

### US-002: Annotate a plan in one place, with phantom changes carrying no risk
**Description:** As a user, I want a phantom change to carry risk `none` so that a no-op never raises the overall risk or trips the CI gate.

**Acceptance Criteria:**
- [ ] `internal/terraform` exports `AnnotatePlan(summary *PlanSummary)`. It runs `ClassifyRisk` on every change, then `DetectPhantomChanges`, then sets `Risk = RiskNone` on every change with `IsPhantom`.
- [ ] `LoadPlan` (`internal/terraform/loader.go`) and `ExecService.Plan` (`internal/terraform/exec/service.go`, around lines 121-127) call `AnnotatePlan` instead of their inline loops.
- [ ] A test proves a phantom update of a critical type, such as `aws_db_instance`, ends with `RiskNone`, and a real update of the same type keeps its classified risk.
- [ ] `internal/terraform/loader_test.go` asserts the exact risk for a phantom change instead of `Risk >= 0`.
- [ ] Plan plugin tests and golden files that pair a phantom update with a risk badge other than `none` are updated.

### US-003: Serve a pre-seeded plan from the exec service
**Description:** As a user running `tfui plan -ci -plan plan.json`, I want tfui to read my file instead of running terraform, as `docs/reference/cli-io-contract.md` promises.

**Acceptance Criteria:**
- [ ] `ExecService.Plan` returns `cache.GetPlan()` when the cache holds a plan, without running terraform or taking the dir lock. This mirrors `loadState` (`internal/terraform/exec/service.go:55-58`).
- [ ] `ExecService.PlanJSON` is unchanged.
- [ ] A test in `internal/terraform/exec` seeds a `ServiceCache` with `SeedPlan` and sets a terraform binary path that does not exist. `Plan` must return the seeded summary with no error.
- [ ] Without a seeded plan, `Plan` behaves as before.
- [ ] The commit message says it restores the pre-seed short-circuit that commit `c1c92df` removed.

### US-004: Surface plugin Configure errors
**Description:** As a user with `level = "hgih"` in `tfui.hcl`, I want tfui to fail and name the plugin so that a broken gate never passes silently.

**Acceptance Criteria:**
- [ ] `internal/plugin/registry.go` `Build` returns `error`, wrapping a non-nil `Configure` error as `plugin %q: %w`. Today it discards that error at line 76.
- [ ] `buildRegistry` (`cmd/tfui/registry.go`) returns `(*plugin.Registry, error)`.
- [ ] `Session.Run` and `Session.RunPlugin` (`cmd/tfui/session.go`) return the error before starting any UI.
- [ ] `internal/plugin/registry_test.go` has a failing-`Configure` test that asserts the message names the plugin ID.
- [ ] Existing `Build` call sites in tests still compile.

### US-005: Risk plugin input and threshold
**Description:** As a CI author, I want to set the risk gate with `plugin "risk" { level = "critical" }` so that I choose what blocks a pipeline.

**Acceptance Criteria:**
- [ ] `plugins/risk/input.go` defines `Input{JSON bool}`.
- [ ] `New` sets the threshold to `sdk.RiskHigh`. `Registry.Build` skips `Configure` when a block has no options, so the default must live in `New`.
- [ ] `Configure` reads `level` with `sdk.ParseRiskLevel` and returns its error unchanged.
- [ ] A missing `level` keeps the default.
- [ ] Tests cover the default, each valid level and an invalid level.

### US-006: Risk view receives the plan
**Description:** As a user, I want `R` to show my plan grouped by risk, whether I ran `p` first or opened tfui with `-plan`.

**Acceptance Criteria:**
- [ ] `Activate(Input) tea.Cmd` runs a plan only when no summary has been analyzed. It follows `plugins/apply/apply.go:214-263`:
  - options come from `GetCtx().PlanOptions()`
  - the plan file is `sdk.NewTempPlanFile` under `os.TempDir()` with prefix `tfui-risk-`, removed once the summary is read
  - no `Writer` is set
- [ ] When a summary is already analyzed, `Activate` returns nil.
- [ ] While the plan runs, the view shows `Loading plan…` and status is `sdk.StatusLoading`. The word `Loading` is what macro `wait ready` blocks on.
- [ ] A successful result calls `Analyze` and sets `StatusDone`. An error sets `StatusError` and keeps the message.
- [ ] `HandlePlanCompleted` calls `Analyze(ev.Summary)` and sets `StatusDone`.
- [ ] `HandlePlanInvalidated` resets to idle.
- [ ] `HandleContextChanged` calls `ReactToContext(ev, nil, e.reset)`, as `plugins/apply/apply.go:171-181` does.
- [ ] `Busy()` reports `StatusLoading`. `Cancel()` aborts the in-flight plan's context.
- [ ] The plugin never emits `sdk.PlanCompletedEvent`. The plan plugin owns the shared plan file (ADR-0020).
- [ ] The idle view reads `No plan yet. Press p to run one.` The golden file `plugins/risk/testdata/golden/TestView_Given_Idle_ShouldRender_NoPlanMessage.txt` and the idle test near `plugins/risk/risk_test.go:255` match it.
- [ ] `Hints()` has no dead branch.
- [ ] `internal/ui/app.go` `activate()` gains `case *tfuirisk.Plugin: return tp.Activate(tfuirisk.Input{})`.
- [ ] `TestApp_Activate_TypedSwitch_DispatchesPerPluginType` (`internal/ui/app_test.go`, near line 5527) has a `risk` row.
- [ ] Tests drive `Activate`, the result message, each handler, `Busy` and `Cancel` through `sdktest.NewDeps` with a `MockService`.

### US-007: Risk command output and CI gate
**Description:** As a CI pipeline, I want `tfui risk -ci` to print a risk report and exit 2 at or above my threshold.

**Acceptance Criteria:**
- [ ] `ExitCode()` returns 1 on `StatusError`, 2 when `sdk.OverallRisk` is at or above the threshold, and 0 otherwise.
- [ ] Text `Stdout()` prints one line per change as `<level>  <address>  <reason>`, groups highest first, then `Risk: <overall>`. Levels use lowercase `RiskLevel.String()`, the same word `tfui plan -ci` prints.
- [ ] JSON `Stdout()` prints `{"overall": "...", "threshold": "...", "counts": {"<level>": n}, "changes": [{"address", "action", "risk", "reason", "phantom"}]}`. `action` is the `sdk.Action` string, and `counts` lists only levels that are present.
- [ ] `Stderr()` returns the plan error plus newline on `StatusError`, and nil otherwise. `Stdout()` returns nothing on error.
- [ ] `cmd/tfui/risk_command.go` follows `cmd/tfui/output_command.go` (`Use: "risk"`) and is registered in `rootCmd.AddCommand` (`cmd/tfui/main.go:28`).
- [ ] `cmd/tfui/risk_command_test.go` copies `cmd/tfui/output_command_test.go`.
- [ ] Golden or table tests cover text and JSON output and each exit code.

### US-008: End-to-end proof without terraform
**Description:** As a maintainer, I want tests that run `tfui risk` and `tfui plan` headlessly from a seeded plan so that the pre-seed path and the gate cannot regress silently.

**Acceptance Criteria:**
- [ ] A new test file `cmd/tfui/analysis_session_test.go` drives `Session.RunPlugin`, following `cmd/tfui/session_test.go:26-121`. It uses `PreseedSources{Plan: ...}`, `silentStderr: true`, a captured `Effects` and a terraform binary path that does not exist.
- [ ] `demo/fixtures/plan-large.json` deletes an `aws_db_instance`, so its overall risk is `critical`. The test asserts the default threshold exits 2 and asserts the text and JSON stdout.
- [ ] `tests/fixtures/plan.json` holds an `aws_instance` create and an `aws_s3_bucket` update. The test first asserts its overall risk, `high` unless the update is phantom, then asserts threshold `critical` exits 0.
- [ ] The same file asserts `tfui plan -ci` from a seeded file prints the tree with no terraform run.
- [ ] `tests/fixtures/tapes/smoke/risk.tape` presses `R` after `wait ready` and asserts text that appears only in a filled view.
- [ ] `mise run test:macro` passes.

### US-009: Risk docs, CLI reference and the JSON ADR
**Description:** As a reader or script author, I want the risk pages and CLI reference to describe what `R` and `tfui risk` actually do.

**Acceptance Criteria:**
- [ ] `docs/plugins/risk.md` is rewritten on the existing plugin template, covering:
  - keys as bound: `j`/`k`/arrows, `esc`/`q`
  - a CLI section with `tfui risk`, `-ci`, `-json` and `-plan -`
  - the equivalence table
  - the JSON schema field by field
  - the `level` option with its default and the exit codes
- [ ] No page claims `tfui plan -json` carries a `risk` field.
- [ ] `docs/features/risk-analysis.md` states the rules in `internal/terraform/risk.go` exactly:
  - delete of an unlisted type is medium
  - update of an unlisted type is low
  - the full type sets from the code
  - phantom changes carry risk `none`
- [ ] `docs/reference/cli-reference.md` gains a `tfui risk` section. Its exit-code table says `2` means plan has changes (`plan`) or risk at or above the threshold (`risk`).
- [ ] `docs/reference/cli-io-contract.md` changes:
  - it gains `tfui risk` rows in the Standalone, CI, Failure and Pre-seeded tables
  - `risk` joins the `-json` flag scope
  - `tfui show -json tfplan.out | tfui risk` becomes `terraform show -json tfplan.out | tfui risk -plan -`
  - the pre-seeded rows say no terraform runs
- [ ] `docs/reference/cli-ux.md` (around lines 122-126 and 165-171) no longer describes stdin-only analysis filters with no TUI.
- [ ] New `docs/adr/0023-analysis-commands-emit-tfui-json.md` follows the format of the other ADRs and is listed in `docs/adr/index.md`. It records the decision: analysis commands emit tfui's own JSON because terraform has no equivalent, while `plan`, `validate`, `output` and `version` `-json` stay terraform pass-through (ADR-0006).
- [ ] `.agents/skills/tfui-cli-ux/SKILL.md` lists exit code 2 for risk. `AGENTS.md` changes only if a command, path or rule it names changed, and it stays under 800 words.

## Functional Requirements

- FR-1: When the plan plugin finishes a plan, the risk plugin must analyze the summary carried by `sdk.PlanCompletedEvent`.
- FR-2: Activating the risk plugin with no analyzed summary must obtain one through `Service.Plan` with the active Context's plan options and a temporary plan file it deletes afterwards.
- FR-3: Activating it with an analyzed summary must not run terraform.
- FR-4: `sdk.PlanInvalidatedEvent` and a chdir or workspace change must return it to idle.
- FR-5: `ExecService.Plan` must return a pre-seeded plan from the cache without running terraform.
- FR-6: One engine function must classify risk, detect phantoms and set risk `none` on phantom changes, for both plan entry points.
- FR-7: `tfui risk` must exit 1 when the plan fails, 2 when overall risk is at or above `level`, and 0 otherwise.
- FR-8: On failure `tfui risk` must write terraform's error to stderr and nothing to stdout.
- FR-9: An invalid plugin option value must stop tfui with an error naming the plugin.

## Non-Goals

- Phantom reasons and the phantom plugin belong to the phantom PRD.
- Blast radius belongs to the blast radius PRD.
- No change to the risk type lists or the classification rules.
- No `risk` field in `tfui plan -json`, which stays terraform's schema.
- No `PlanJSON` cache support.
- No fix for `ExecService.WithDir` dropping the seeded cache, binary plan files in `-plan`, or apply from idle without a plan file. Each is filed as its own issue outside this run.
- No GIF regeneration. That is blocked on #77.

## Technical Considerations

- The event bus binds handlers once when the app is built (`pkg/sdk/bus.go:14-37`), after `Configure`. Adding handler methods is enough wiring.
- In CI mode `Session.RunPlugin` waits for `Ready()` or a terminal status (`cmd/tfui/session.go:229-234`). A failed plan must reach `StatusError`, or the command hangs for 10 minutes.
- An empty `PlanOptions.PlanFile` makes the exec service write `tfplan.out` into the working directory. Always pass a temp path.
- `tests/integration` exits early without terraform (`tests/integration/main_test.go:19-22`). Terraform-free proof belongs in `cmd/tfui`.
- Standalone mode builds the bus too, so `tfui plan` will feed the risk plugin in memory. Its output must not change.

## Success Metrics

- `tfui -plan demo/fixtures/plan-large.json` followed by `R` shows a filled view, proven by `risk.tape`.
- `tfui risk -ci -plan demo/fixtures/plan-large.json` exits 2 with no terraform binary present.
- A live session where the user runs `p` then `R` logs one `terraform.exec` plan.
- `mise run test:coverage` stays at 100%.

## Open Questions

- None blocking. The reviewer can revisit these planning calls on the pull request:
  - a view runs its own plan when none was shared
  - phantom changes carry risk `none`
  - the JSON field names
  - the default threshold `high`
[/PRD]
