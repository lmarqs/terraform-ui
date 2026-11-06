[PRD]
# PRD: Finish the Blast Radius plugin (run third of three)

## Overview

`plugins/blastradius` (key `B`) groups a plan's changes by module and scores each module's impact,
but it has never shown a result outside its tests. Its `Analyze(*sdk.PlanSummary)` has no
production caller, so `B` always shows "Run a plan first". GitHub issue #76 records this.

It has no headless side, and its docs disagree with the code:
- They say "high" means "3+ destructive operations". `calculateImpact` means any destructive change in a module with 3 or more changes.
- The feature page says key `b`. The registry binds `B`.
- The README promises "resource dependencies", but no dependency graph exists.
- The pages list `Space` to expand, but only `enter` and `i` are bound.

This PRD depends on the risk PRD (`tasks/prd-risk.md`) and the phantom PRD (`tasks/prd-phantom.md`),
which must be complete first. They deliver:
- the exec service serving a pre-seeded `-plan`
- `AnnotatePlan`
- `cmd/tfui/analysis_session_test.go`
- ADR-0023 for tfui-native JSON
- the lifecycle pattern this PRD copies

## Goals

- `B` shows a filled view after `p`, or with a pre-seeded `-plan`.
- `tfui blastradius` follows the CLI I/O contract.
- Blast radius docs and the README match the code.

## Quality Gates

These commands must pass for every user story:
- `mise run check:lint` - golangci-lint, repo-wide
- `mise run test:unit` - Go unit tests
- `mise run check:build` - compile check
- `mise run test:coverage` - CI enforces 100% coverage outside `cmd/` and `internal/terraform/exec`

For UI stories (US-001, US-003), also include:
- `mise run test:macro` - macro tape tests verifying TUI rendering

## Working rules for every story

- Work in the worktree `.worktrees/fix/analysis-plugins` on branch `fix/analysis-plugins`.
- Follow `AGENTS.md` and the `tfui-conventions`, `tfui-architecture`, `tfui-cli-ux` and `tfui-tui-ux` skills in `.agents/skills/`.
- Write the failing test first, then the implementation.
- Plugins import only `pkg/sdk`. Plugin tests use `sdktest.NewDeps(svc)` and call `Init()`.
- Mirror the finished risk and phantom plugins for structure, names and test layout.
- Commit each story as a conventional commit and push it to `origin fix/analysis-plugins`.
- Never weaken a check to make it pass.

## User Stories

### US-001: Blast radius view receives the plan
**Description:** As a user, I want `B` to show which modules my plan touches, highest impact first, whether I ran `p` first or opened tfui with `-plan`.

**Acceptance Criteria:**
- [ ] `plugins/blastradius/input.go` defines `Input{JSON bool}`.
- [ ] `Activate(Input)`, `HandlePlanCompleted`, `HandlePlanInvalidated`, `HandleContextChanged`, `Busy` and `Cancel` behave exactly as the risk plugin's do, with temp plan file prefix `tfui-blastradius-`.
- [ ] While the plan runs, the view shows `Loading plan…`.
- [ ] The idle view reads `No plan yet. Press p to run one.`, and the idle test near `plugins/blastradius/blastradius_test.go:328` matches.
- [ ] Scoring in `calculateImpact` is unchanged.
- [ ] Keys stay `j`/`k`/arrows, `enter`/`i`, `esc`/`q`.
- [ ] `internal/ui/app.go` `activate()` gains a blast radius case, and `TestApp_Activate_TypedSwitch_DispatchesPerPluginType` gains a `blastradius` row.

### US-002: Blast radius command output
**Description:** As a script author, I want `tfui blastradius` to print each touched module with its impact and change counts.

**Acceptance Criteria:**
- [ ] Text `Stdout()` prints one line per module, highest impact first, as `<module>  <impact>  +<add> ~<change> -<destroy> ±<replace>`, then `Modules: <n>, changes: <m>`.
- [ ] JSON `Stdout()` prints `{"total_changes": m, "modules": [{"module", "impact", "add", "change", "destroy", "replace", "changes": [{"address", "action", "risk", "phantom"}]}]}`. `impact` is `ImpactScore.String()` and `action` is the `sdk.Action` string.
- [ ] `Stderr()` returns the plan error plus newline on `StatusError`, and nil otherwise. `Stdout()` returns nothing on error.
- [ ] `ExitCode()` is 1 on `StatusError` and 0 otherwise.
- [ ] `cmd/tfui/blastradius_command.go` (`Use: "blastradius"`) follows `cmd/tfui/risk_command.go`, is registered in `cmd/tfui/main.go`, and has `blastradius_command_test.go`.
- [ ] Golden or table tests cover text and JSON output, a root-only plan, and each exit code.

### US-003: End-to-end proof without terraform
**Description:** As a maintainer, I want `tfui blastradius` exercised headlessly from a seeded plan so that the path cannot regress.

**Acceptance Criteria:**
- [ ] `cmd/tfui/analysis_session_test.go` gains blast radius cases against `demo/fixtures/plan-large.json` with a terraform binary path that does not exist. They assert text stdout, the JSON shape, the first module's impact and exit code 0.
- [ ] `tests/fixtures/tapes/smoke/blastradius.tape` presses `B` after `wait ready` and asserts text seen only in a filled view, such as `module(s) affected`.
- [ ] `mise run test:macro` passes.

### US-004: Blast radius docs and README match the code
**Description:** As a reader, I want the blast radius pages to describe what `B` and `tfui blastradius` actually do.

**Acceptance Criteria:**
- [ ] `docs/plugins/blastradius.md` is rewritten on the existing plugin template, covering:
  - keys as bound, with no `Space`
  - a CLI section with `tfui blastradius`, `-ci`, `-json` and `-plan -`
  - the equivalence table
  - the JSON schema field by field
  - the impact rule as `calculateImpact` implements it
- [ ] `docs/features/blast-radius.md` uses key `B`, states the impact rule exactly and makes no dependency claim.
- [ ] The README blast radius bullet drops "resource dependencies".
- [ ] `docs/reference/cli-reference.md` gains a `tfui blastradius` section.
- [ ] `docs/reference/cli-io-contract.md` gains `tfui blastradius` rows in the Standalone, CI, Failure and Pre-seeded tables, and `blastradius` joins the `-json` flag scope.
- [ ] `demo/tapes/blastradius.tape` comments drop "dependency graph".
- [ ] `docs/plugins/index.md` and the README keys table still list `R`, `P` and `B` correctly.

## Functional Requirements

- FR-1: When the plan plugin finishes a plan, the blast radius plugin must analyze the summary carried by `sdk.PlanCompletedEvent`.
- FR-2: Activating it with no analyzed summary must obtain one through `Service.Plan` with a temporary plan file. With one already analyzed, it must not run terraform.
- FR-3: `tfui blastradius` must exit 1 when the plan fails and 0 otherwise. On failure it must write terraform's error to stderr and nothing to stdout.
- FR-4: `-json` must switch stdout to the documented schema.

## Non-Goals

- No dependency graph and no resource-level blast radius. Blast radius stays module grouping.
- No change to the impact scoring rule.
- No exit-code gate on impact.
- Risk and phantom work belong to their own PRDs.
- No GIF regeneration. That is blocked on #77.

## Technical Considerations

- Module grouping comes from `sdk.GroupByModule` (`pkg/sdk/grouping.go`), which sorts alphabetically and names changes without a module `root`. The plugin then sorts by impact.
- Phantom changes already carry risk `none` from the risk PRD's `AnnotatePlan`, which lowers some module scores. Tests should reflect that.

## Success Metrics

- `tfui -plan demo/fixtures/plan-large.json` followed by `B` shows a filled view, proven by `blastradius.tape`.
- `tfui blastradius -ci -json -plan demo/fixtures/plan-large.json` returns modules sorted by impact with no terraform binary present.
- `mise run test:coverage` stays at 100%.
- After this PRD, issue #76 can be closed.

## Open Questions

- None blocking.
[/PRD]
