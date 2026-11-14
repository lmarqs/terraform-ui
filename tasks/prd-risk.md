[PRD]
# Finish Risk analysis and project policy evaluation

## Status and source of truth

This is the implementation plan, not the definition of risk. The accepted language, responsibilities and
eligibility are documented; the remaining public contract choices and embedding experiments still block
production implementation. Do not execute dependent stories by choosing unspecified semantics in code.

Definitions and architecture live under `docs/`:

- [Policy contract](../docs/reference/risk-policy-contract.md): purpose, philosophy, terms, eligibility, ownership,
  failure behavior and review presentation.
- [Architecture design](../docs/development/risk-policy-design.md): goals, requirements R1–R8, alternatives,
  component boundaries, evidence and remaining decisions.
- [Runtime research](../docs/development/risk-policy-runtime-research.md): pinned OPA API facts and their limits.
- [ADR-0024](../docs/adr/0024-embed-rego-for-risk-policies.md) and
  [ADR-0025](../docs/adr/0025-classify-resource-mutations-explicitly.md): accepted decisions.
- [Risk plugin guide](../docs/plugins/risk.md) and [fixed classifier](../docs/features/risk-analysis.md): current behavior.

The [architecture map](https://github.com/lmarqs/terraform-ui/issues/107) owns decision progress. This PRD owns
implementation sequencing and verification. Requirements below reference the docs rather than inventing another
policy definition. The architecture map ends at design acceptance; implementing this PRD is subsequent work.

## Delivery boundary

Deliver the policy contract through the Risk view, shared Plan classifications and a dedicated `tfui risk`
command. Preserve the existing Terraform pass-through output and saved-plan lifecycle. Apply behavior is outside
this work. Cross-resource policy analysis, remote distribution, parent-rule mixing, runtime presets, unrelated
plugin repairs and binary-plan support through `-plan` are not part of this delivery.

The working browser mockups are visual references only. Their older Apply interactions are not requirements.
The contract reference owns the intended inline-diff presentation; existing TUI standards own icons and colors.

## Working and verification rules

- Follow `AGENTS.md` and the tfui conventions, architecture, CLI UX and TUI UX skills. Use an isolated worktree
  following `.worktrees/README.md`; do not assume the old `fix/analysis-plugins` worktree still exists or is appropriate.
- Before implementation, the `test-writer` agent writes the failing behavior test. Fixes must have a test that
  fails before the fix. Do not add tests that merely mirror implementation details.
- Plugins import only `pkg/sdk`. OPA and source-loading internals stay behind SDK-facing domain values.
- Check current code before each story. Preserve working cache and event behavior instead of reimplementing it.
- Quick check: `mise run check:build` and `mise run test:unit ./<affected-package>/...`.
- Done check: `mise run check:lint`, `mise run test:unit`, `mise run check:build`.
- CI coverage: `mise run test:coverage`; the repository enforces 100% outside `cmd/` and `internal/terraform/exec`.
- UI stories additionally run `mise run test:macro` against a freshly built binary. `check:build` alone does not
  produce that binary; use `mise run build` or pass the task an explicitly built binary.
- Documentation changes run `mise run docs:build` followed by `mise run docs:check`.
- Report checks individually; never weaken a check. Use conventional commits and follow the active session's
  branch/push authorization. No story grants permission to change Apply behavior.

## Phase 0: resolve architecture and prove feasibility

### D-01: Complete the public contract decisions

**Depends on:** accepted purpose and ownership in ADR-0025; evidence from D-02 and D-03 as needed.

- [ ] Resolve the remaining decisions listed in the architecture design: input/entrypoint/result representation,
  unknown/sensitive handling, trust and capabilities, source selection/reload, multiple-rule resolution,
  tfui aggregation, thresholds and report/version compatibility.
- [ ] Narrow older decision tickets to their remaining questions. Per-resource scope, complete results, custom
  replacement, eligibility and failure behavior are already accepted; do not reintroduce rejected alternatives
  as implementation defaults.
- [ ] Specify the user-facing result for no eligible mutations, incomplete input and partial evaluation failure.
  These cases must preserve the distinctions defined in the contract reference.
- [ ] Decide how the existing Blast Radius consumer of `PlanChange.Risk` handles custom classifications and
  unavailable/partial assessments. Do not silently alter impact semantics or turn failed assessment into a zero level.
- [ ] Decide explicit helper reuse, template testing and built-in-policy upgrade behavior. Review particular
  Lambda classifications without assuming the field report's suggested severities are accepted.
- [ ] Review the architecture with the owner, including the alternatives table and residual risks; update docs
  and the decision map with the accepted outcomes.
- [ ] Specify the approved interface under `docs/reference/`, with exact examples and compatibility behavior,
  before implementing consumers. Replace this phase's gates with links to accepted definitions.

### D-02: Prove input fidelity and Context/source identity

**Dependencies:** can gather code evidence now; final behavior follows D-01. Use disposable prototypes.

- [ ] Compare source plan JSON with candidate per-resource input: large numbers, ordered lists, absent/null,
  nested/whole-object unknowns and sensitivity, resource identity, and both replacement orders.
- [ ] Demonstrate that adding/removing neighboring resources cannot affect a resource's policy input or result.
- [ ] Exercise live and seeded acquisition with call counting. Preserve evidence from the reviewed plan without
  generating a second plan solely for policy data.
- [ ] Exercise phantom counterexamples for array order/null normalization. Document advisory-data limitations;
  this experiment does not override the accepted requirement to assess eligible phantom mutations.
- [ ] Trace real root/Member/workspace source selection, relative path origin, zero-config, absent/empty source,
  invalid/deleted source, source edits, startup Chdir, and switching A to B while A evaluates.
- [ ] Record pass/falsification criteria before execution: silent data loss rejects an input candidate; mixed
  source or stale A results reject a lifecycle candidate. Link reproducible assets and obtain owner review.

### D-03: Validate embedded evaluation and measure costs

**Dependencies:** versioned research is complete; trust/result candidates from D-01 are needed for acceptance.

- [ ] Select and pin a candidate version after reviewing current maintenance. The research pin is evidence, not
  an automatic production dependency choice.
- [ ] Declare fixtures, toolchain, baseline, target matrix and measurement method before running. Use bounded
  workloads and an external harness limit; cooperative context cancellation is not a hard memory/termination bound.
- [ ] Distinguish undefined entrypoint, defined empty value, malformed result, competing complete-rule values,
  compilation failure and evaluated builtin failure. Checking only a returned Go error must not pass undefined output.
- [ ] Compare strict builtin behavior for executed versus unreachable expressions and document the limit.
- [ ] Attempt HTTP, DNS, time, randomness, print, trace and secret interpolation under the candidate capability
  boundary. Capture output/error channels; test against the chosen trust promise rather than claiming generic sandboxing.
- [ ] Build every supported release target and measure baseline/candidate binary size, compilation time,
  cold/warm evaluation and peak memory. Obtain owner judgment on costs; no numerical budget is yet accepted.
- [ ] Record outcomes in docs evidence and the decision ticket. Unsupported builds or failed controls reopen
  the affected architecture choice before production integration.

## Phase 1: implement the accepted contract

### US-001: Preserve policy evidence through plan acquisition

**Dependencies:** D-01, D-02. **Trace:** R1,R2,R5,R6.

- [ ] Retain the approved per-resource evidence separately from display-string diffs in both live and seeded paths.
- [ ] Implement eligibility from the contract reference, including import/move with accompanying mutation and
  explicit treatment of non-assessed changes. Do not map unsupported input silently to `none`.
- [ ] Preserve `ExecService.Plan`'s existing cache short-circuit and its regression tests.
- [ ] Verify seeded analysis through `Session.RunPlugin` with a nonexistent Terraform binary, including the
  initial selected Chdir. Do not rely solely on direct service tests; `WithDir` creates a fresh cache today.
- [ ] Verify no second plan for Risk when the current plan evidence is available. `-plan` remains Terraform show
  JSON input for analysis, not an executable artifact or Apply authorization; preserve `PlanJSON` pass-through.

### US-002: Resolve and load the selected complete policy

**Dependencies:** US-001 and accepted source/trust contracts. **Trace:** R3,R7.

- [ ] Implement approved HCL source selection and path resolution, including built-in selection when no custom
  source is selected. Do not implement a custom HCL matcher or independently inherited rule fragments.
- [ ] Exercise actual configuration parsing/resolution, not just manually built option maps; account for the
  reported ignored member plugin block and per-key merge behavior.
- [ ] Identify the loaded source snapshot consistently for compilation, evaluation, reload and reporting.
- [ ] Report invalid source/configuration explicitly. Design propagation so risk-policy failures preserve Plan
  review with unavailable risk; do not blindly make every Risk Configure error abort the whole application.
- [ ] Cover absent/empty selectors, malformed configuration, unreadable source, Member/workspace changes and
  same-path content edits according to the accepted lifecycle.

### US-003: Evaluate and validate independent classifications

**Dependencies:** US-001, US-002, D-03. **Trace:** R1–R6.

- [ ] Embed the approved evaluator version and capabilities in app internals; expose only the accepted domain
  result through the SDK boundary.
- [ ] Evaluate each eligible mutation against its own permitted data; validate explicit result shape and
  completeness. Cover undefined/missing output, malformed levels/reasons and rule conflicts as specified.
- [ ] Preserve explicit built-in/custom ownership. A custom Lambda rule without a database default cannot obtain
  a hidden built-in database classification.
- [ ] Keep unknown/null/absent distinctions and sensitive-data behavior consistent with the accepted contract.
- [ ] Classify eligible phantom mutations normally. Remove no risk merely because `IsPhantom` is true.
- [ ] Centralize shared assessment use across acquisition/consumers as justified by the accepted design; do not
  create separate policy engines in Plan and Risk or copy the old classification loops into new paths.
- [ ] Verify that policy compilation/evaluation/validation failure never becomes a neutral or successful assessment.

### US-004: Integrate assessment lifecycle and failures

**Dependencies:** US-003. **Trace:** R4,R5,R7.

- [ ] Connect plan completion, invalidation, Context changes and Risk activation to the accepted assessment lifecycle.
- [ ] Reset derived assessment on invalidation and reject late results from a previous Context or policy snapshot.
- [ ] Keep Plan review available on policy failure with unavailable risk and visible error. Preserve the existing
  behavior for actual Terraform planning failures; do not confuse the two failure classes.
- [ ] Verify the accepted compatibility behavior for Blast Radius's shared risk input, including custom results
  and unavailable/partial assessment. Limit changes to this integration boundary; no unrelated plugin redesign.
- [ ] Implement cancellation and terminal statuses using existing operation lifecycle conventions so headless
  sessions terminate rather than hang waiting for `Ready()`.
- [ ] When Risk needs to acquire a plan, use active Context options and a temporary artifact with explicit cleanup;
  an empty plan-file option currently writes `tfplan.out` in the working directory.
- [ ] Preserve Plan's artifact/event ownership under ADR-0020. Risk must not publish itself as the owner of Plan's
  executable saved artifact or alter Apply handoff/confirmation.

### US-005: Render mutations and classifications consistently

**Dependencies:** US-004. **Trace:** R5,R8.

- [ ] Implement the presentation defined in the contract reference using `sdk.ActionSymbol`, shared risk styles
  and a shared attribute-diff renderer extracted from Plan as needed. No plugin-to-plugin imports or copied tables.
- [ ] Show compact attribute changes directly with the resource row, overflow count and full inspection; no
  repeated `Change` label. Keep group headings out of the resource cursor sequence.
- [ ] Preserve action/replacement semantics, unknown notation and sensitive masking in narrow layouts and inspection.
- [ ] Use the standard scroll gutter and cursor/count title; test navigating beyond the first viewport and resizing.
- [ ] Distinguish no eligible mutations, explicit `none`, and unavailable assessment according to the approved
  presentation contract. Phantom-only mutation plans are not automatically neutral.
- [ ] Replace safety claims such as "changes look safe" with the classification and its basis. Add no Apply flow.
- [ ] Run shared renderer tests and macro coverage for actions, both replacement orders, nested unknown/sensitive
  values, long plans, inspection and policy failure.

### US-006: Add risk command reporting and threshold evaluation

**Dependencies:** US-004 and approved report/threshold contract. **Trace:** R2,R4,R5.

- [ ] Register `tfui risk` using existing Standalone/Headless command conventions and cancellation handling.
- [ ] Implement the accepted tfui aggregation and threshold comparison outside policy evaluation.
- [ ] Implement approved text/JSON report, provenance, versioning and exit results. Do not adopt the old PRD's
  `level`, default `high`, exit-code table or JSON fields without the contract decision.
- [ ] Cover threshold boundary equality, explicit overrides, old ignored `level`, empty eligibility, excluded
  changes, explicit defaults and all evaluation failures.
- [ ] Keep `tfui plan -json` Terraform pass-through. Do not add a risk field or change Apply flags/auto-approval.
- [ ] Validate stdout/stderr behavior against the approved CLI contract, including failures and sensitive diagnostics.

### US-007: Provide explicit policy templates and author tests

**Dependencies:** US-003 and approved template/helper decisions. **Trace:** R1–R3,R6.

- [ ] Supply complete, copyable source with an explicit default and no runtime preset selector.
- [ ] Demonstrate the approved author-testing workflow on isolated resource fixtures.
- [ ] Include Lambda code-source versus memory-only examples, unchanged/null/unknown hashes and unknown source
  values. Reasons describe evidence rather than inventing placeholder or external ownership.
- [ ] Apply only reviewed classifications; neither layer severities nor blanket lowering of non-code updates is
  implied by the field report. Verify custom policies never gain built-in fallback through template tooling.

## Phase 2: prove integration and publish implemented interfaces

### US-008: Exercise the real user paths

**Dependencies:** US-005, US-006, US-007.

- [ ] Run seeded Plan and Risk through real session/config wiring with no Terraform binary. Compare classification
  and explanation consistency, not only counts or a summary badge.
- [ ] Exercise current `tests/fixtures/plan.json` and `demo/fixtures/plan-large.json` under the selected policy;
  derive expected outcomes from that policy rather than stale hard-coded default-threshold assumptions.
- [ ] Verify one live plan acquisition for Plan followed by Risk; cover Context/source changes, race rejection,
  policy errors, cancellation, and explicit non-assessed cases.
- [ ] Verify existing Apply behavior and Terraform JSON output remain unchanged using appropriate existing
  regression checks. This is preservation verification, not permission to redesign Apply.
- [ ] Run required unit, coverage, build, lint and UI macro checks. Document actual commands and outcomes.

### US-009: Publish the implemented behavior and complete the PRD

**Dependencies:** US-008.

- [ ] Update the Risk guide's availability, keys, input modes and settings to match verified implementation.
- [ ] Update CLI reference and I/O contract with approved flags, schemas and exit results. Keep one canonical
  definition per interface; link from guides rather than duplicating tables.
- [ ] Update the feature page's built-in rules if their implementation changes; distinguish them from custom policy.
- [ ] Publish only accepted architecture records in the existing format. ADR-0025 is already assigned; discover
  the next free number if report-schema decisions require a separate record.
- [ ] Update relevant skill summaries only when their authoritative docs change. Update AGENTS.md only if a named
  command, path or rule changes. Do not mark design intent as implemented until behavior tests prove it.
- [ ] Build and check the docs site. Refresh demo media only if explicitly included in the implementation scope;
  the existing demo-generation issue does not justify false availability claims.
- [ ] Close implementation tracking only after the required checks pass. Architecture acceptance and documentation
  completion alone do not mean the feature is implemented.
[/PRD]
