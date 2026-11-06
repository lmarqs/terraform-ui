[PRD]
# PRD: Finish the Phantom plugin (run second of three)

## Overview

`plugins/phantom` (key `P`) lists plan changes that do nothing, but it has never shown a result
outside its tests. Its `Analyze(*sdk.PlanSummary)` has no production caller, so `P` always shows
"Run a plan first". GitHub issue #76 records this.

Its explanations are guesses from the attribute name. `explainPhantom`
(`plugins/phantom/phantom.go:244-269`) says "JSON/policy reordering" for any key containing
`json` or `policy`, whatever the engine actually normalized. It has no headless side. Its docs promise
a `tfui phantom` stdin filter, a `tfui show` command and `phantom` fields in `tfui plan -json`.
None of those exists.

This PRD depends on the risk PRD (`tasks/prd-risk.md`), which must be complete first. The risk PRD
delivers:
- `AnnotatePlan`
- phantom changes carrying risk `none`
- the exec service serving a pre-seeded `-plan`
- plugin `Configure` errors
- `cmd/tfui/analysis_session_test.go`
- ADR-0023 for tfui-native JSON
- the risk plugin, whose lifecycle this PRD copies

## Goals

- Each phantom change carries the real normalization that made it a no-op.
- `P` shows a filled view after `p`, or with a pre-seeded `-plan`.
- `tfui phantom` follows the CLI I/O contract.
- Phantom docs match the code.

## Quality Gates

These commands must pass for every user story:
- `mise run check:lint` - golangci-lint, repo-wide
- `mise run test:unit` - Go unit tests
- `mise run check:build` - compile check
- `mise run test:coverage` - CI enforces 100% coverage outside `cmd/` and `internal/terraform/exec`

For UI stories (US-002, US-004), also include:
- `mise run test:macro` - macro tape tests verifying TUI rendering

## Working rules for every story

- Work in the worktree `.worktrees/fix/analysis-plugins` on branch `fix/analysis-plugins`.
- Follow `AGENTS.md` and the `tfui-conventions`, `tfui-architecture`, `tfui-cli-ux` and `tfui-tui-ux` skills in `.agents/skills/`.
- Write the failing test first, then the implementation.
- Plugins import only `pkg/sdk`. Plugin tests use `sdktest.NewDeps(svc)` and call `Init()`.
- Mirror the finished risk plugin (`plugins/risk/`) for structure, names and test layout.
- Commit each story as a conventional commit and push it to `origin fix/analysis-plugins`.
- Never weaken a check to make it pass.

## User Stories

### US-001: Record why a change is phantom
**Description:** As a user reading the phantom view, I want the real reason a change is a no-op so that I can trust it.

**Acceptance Criteria:**
- [ ] `pkg/sdk/change.go` adds `PhantomReason string` to `PlanChange`.
- [ ] `internal/terraform/phantom.go` tests each attribute diff progressively and records the first normalization under which old equals new:
  - raw strings equal: `identical`
  - equal as parsed JSON: `json formatting`
  - equal after dropping null map entries: `null vs absent`
  - equal after sorting lists: `list order`
- [ ] A change with several diffs takes the reason of its first diff that is not `identical`. A change whose diffs are all `identical` gets `identical`.
- [ ] `DetectPhantomChanges` sets `PhantomReason` on every change it marks `IsPhantom`, and leaves it empty on real changes.
- [ ] `internal/terraform/phantom_test.go` has a case per reason, a multi-diff case and a real-change case.
- [ ] The non-JSON equal-values case in `internal/terraform/service_test.go` (near line 584) asserts `identical`.

### US-002: Phantom view receives the plan
**Description:** As a user, I want `P` to list phantom changes, whether I ran `p` first or opened tfui with `-plan`.

**Acceptance Criteria:**
- [ ] `plugins/phantom/input.go` defines `Input{JSON bool}`.
- [ ] `Activate(Input)`, `HandlePlanCompleted`, `HandlePlanInvalidated`, `HandleContextChanged`, `Busy` and `Cancel` behave exactly as the risk plugin's do, with temp plan file prefix `tfui-phantom-`.
- [ ] While the plan runs, the view shows `Loading plan…`.
- [ ] The idle view reads `No plan yet. Press p to run one.`, and the idle test near `plugins/phantom/phantom_test.go:283` matches.
- [ ] `explainPhantom` and `PhantomChange.Explanation` are removed. The expanded row shows `change.PhantomReason`.
- [ ] Tests near `plugins/phantom/phantom_test.go:71-104` and `:427` are updated or removed to match.
- [ ] Keys stay `j`/`k`/arrows, `enter`/`i`, `esc`/`q`.
- [ ] `internal/ui/app.go` `activate()` gains a phantom case, and `TestApp_Activate_TypedSwitch_DispatchesPerPluginType` gains a `phantom` row.

### US-003: Phantom command output
**Description:** As a script author, I want `tfui phantom` to print the phantom changes and their reasons.

**Acceptance Criteria:**
- [ ] Text `Stdout()` prints `<address>  <reason>` per phantom, then `Phantom: <n> of <total> changes`.
- [ ] JSON `Stdout()` prints `{"phantom_changes": n, "real_changes": m, "phantom_resources": [{"address", "reason"}]}`.
- [ ] `Stderr()` returns the plan error plus newline on `StatusError`, and nil otherwise. `Stdout()` returns nothing on error.
- [ ] `ExitCode()` is 1 on `StatusError` and 0 otherwise.
- [ ] `cmd/tfui/phantom_command.go` (`Use: "phantom"`) follows `cmd/tfui/risk_command.go`, is registered in `cmd/tfui/main.go`, and has `phantom_command_test.go`.
- [ ] Golden or table tests cover text and JSON output, a plan with no phantoms, and each exit code.

### US-004: End-to-end proof without terraform
**Description:** As a maintainer, I want `tfui phantom` exercised headlessly from a seeded plan so that the path cannot regress.

**Acceptance Criteria:**
- [ ] `cmd/tfui/analysis_session_test.go` gains phantom cases against `demo/fixtures/plan-large.json` with a terraform binary path that does not exist. They assert text stdout, the JSON shape and exit code 0.
- [ ] `tests/fixtures/tapes/smoke/phantom.tape` presses `P` after `wait ready` and asserts text seen only in a filled view. On `tests/fixtures/plan.json` that may be `No phantom changes detected.`.
- [ ] `mise run test:macro` passes.

### US-005: Phantom docs match the code
**Description:** As a reader, I want the phantom pages to describe what `P` and `tfui phantom` actually do.

**Acceptance Criteria:**
- [ ] `docs/plugins/phantom.md` is rewritten on the existing plugin template, covering:
  - keys as bound, with no `Space`
  - a CLI section with `tfui phantom`, `-ci`, `-json` and `-plan -`
  - the equivalence table
  - the JSON schema field by field
- [ ] No page claims `tfui plan -json` carries a `phantom` field or `phantom_resources`, or that `tfui plan -ci` shows a phantom count.
- [ ] `docs/features/phantom-changes.md`:
  - lists the four reasons with an example each
  - says phantom changes carry risk `none`
  - drops `tfui show` and the stdin-only example
  - drops the dimming and filter claims about the plan view
- [ ] `docs/reference/cli-reference.md` gains a `tfui phantom` section.
- [ ] `docs/reference/cli-io-contract.md` gains `tfui phantom` rows in the Standalone, CI, Failure and Pre-seeded tables, and `phantom` joins the `-json` flag scope.
- [ ] `demo/tapes/phantom.tape` still presses `P` directly.

## Functional Requirements

- FR-1: Each phantom change must carry exactly one of `identical`, `json formatting`, `null vs absent` or `list order`.
- FR-2: When the plan plugin finishes a plan, the phantom plugin must analyze the summary carried by `sdk.PlanCompletedEvent`.
- FR-3: Activating it with no analyzed summary must obtain one through `Service.Plan` with a temporary plan file. With one already analyzed, it must not run terraform.
- FR-4: `tfui phantom` must exit 1 when the plan fails and 0 otherwise. On failure it must write terraform's error to stderr and nothing to stdout.
- FR-5: `-json` must switch stdout to the documented schema.

## Non-Goals

- No phantom filter, dimming or hiding in the plan view.
- No new normalizations beyond the four that exist.
- No `phantom` field in `tfui plan -json`.
- Risk and blast radius work belong to their own PRDs.

## Technical Considerations

- `IsPhantomChange` compares normalized strings (`internal/terraform/phantom.go:41-47`). It cannot tell which normalization made them equal, so the reason requires testing each stage in turn.
- `internal/terraform` sits inside the 100% coverage gate, so every reason branch needs a test.
- Every `PlanChange` literal in the repo uses keyed fields, so adding a field compiles everywhere.

## Success Metrics

- `tfui -plan demo/fixtures/plan-large.json` followed by `P` shows a filled view, proven by `phantom.tape`.
- Every phantom row shows one of the four engine reasons.
- `mise run test:coverage` stays at 100%.

## Open Questions

- None blocking.
[/PRD]
