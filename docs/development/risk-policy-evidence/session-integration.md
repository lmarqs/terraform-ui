---
layout: default
title: Risk policy session integration verification
parent: Development
description: "Records reproducible seeded and live acquisition checks for selected-policy Plan and Risk review."
audience: "Contributors who know the risk contract and need to reproduce US-008a integration verification"
---

# Risk policy session integration verification

The US-008a tests exercise the real Session, configuration, registry, app and policy evaluator. They compare
resource classifications and explanations between Plan and Risk, and count Terraform acquisition across
navigation and source reload. This bounded verification complements the earlier
[disposable acquisition experiments](acquisition-counts.md); it does not certify cloud-provider execution,
hard evaluator termination limits or the remaining US-008/US-009 delivery work.

## Fixture failures and corrections

The test-writer first ran `TestSession_GivenSeededRepositoryFixtures` against the unchanged repository
fixtures. Both returned `input_incomplete`: every resource row lacked `mode`. Adding explicit
`"mode": "managed"` exposed `input_invalid` in the large fixture. Its two database sensitivity masks named
`password` fields absent from their value objects; its deleted database paired `after: null` with
`after_unknown: {}`. The corrected fixture retains the sensitivity masks, includes masked `password: null`
fields, and uses `after_unknown: false` for the deletion. Actions, addresses and the other attribute values
remain unchanged. Missing and malformed input still fail production validation; no parser fallback was added.

Before correction, the fixture SHA-256 values were:

- `tests/fixtures/plan.json`: `bd37a13a379c79e11e5a2dd5aa0f07927d4ede03af4bf5eb3132bc231cc224ae`.
- `demo/fixtures/plan-large.json`: `29bc0446b08bd86e30b2809ec0a252243177ebc66c401c72b1ebae0a69734a45`.

The failing assertions are retained in the run's `.ralph-tui/red-24.log` and intermediate
`.ralph-tui/quick-cmd-24.log`. Those local logs are not required to reproduce the tests: the fixture diff
identifies the original malformed inputs, and `TestLoadPlan_GivenInvalidPolicyEvidence_ShouldKeepReviewAndMarkUnavailable`
continues to exercise missing metadata and malformed evidence rejection.

The corrected fixtures also exposed two obsolete smoke tapes: `risk_consumers.tape` and
`risk_lifecycle.tape` expected the missing-mode failure. Their replacement assertions require the small
fixture's built-in high bucket update and low instance creation, Plan's high aggregate, and high Blast Radius
with both exact policy explanations. Separate missing-source and partial-assessment macro scenarios retain
failure presentation coverage. The first updated consumer assertion used lowercase `[high]`; the SDK renders
`[HIGH]`, so that assertion was corrected to the exact badge. No check or parser rule was relaxed.

## Selected-policy expectations

`cmd/tfui/session_risk_test.go` creates an isolated project with a complete custom Rego policy and loads its
selector through actual HCL parsing. The policy explicitly classifies eligible mutations as low with reason
code `selected-project-policy` and message `Reviewed with the selected project policy.` The expected level
and explanation come from that source; they are independent of built-in type severities or thresholds.

The small fixture has two eligible mutations. The large fixture has fourteen eligible mutations and one
excluded no-op. Tests compare the complete Plan and Risk assessments, independently check every reviewed
address/action pair, check exact levels and reasons, and preserve the reviewed JSON digest. Plan inspection
also displays the selected classification and explanation. Seeded `Session.RunPlugin` uses a nonexistent
Terraform binary and creates no default `tfplan.out` artifact.

The bounded live path uses Session's ExecService and registry with a local shell executable implementing
only the Terraform responses needed by the test. The app navigates Plan to Risk, edits the selected source
at the same path, then revisits Plan and Risk. The assessment changes from low to high with a new reason
and source digest while retaining the reviewed plan digest. Deleting the source preserves Plan review with
unavailable risk and gives Risk a terminal source error. The process call log must contain exactly one
`plan` and one `show` across these operations. This tests the real execution/cache/event boundary; the
shell executable does not run a provider or create real infrastructure.

The same app then switches to a Member and the `dev` workspace through `ContextSwitchRequestMsg`.
The Member's HCL selects a missing default source and overrides it with `workspace.rego` for `dev`.
The test requires the previous assessment to clear, verifies Chdir/Workspace/WorkingDir, and compares
Plan and Risk against the workspace policy's new explanation. The new Context has its own single
`plan`/`show` pair; the previous Context's acquisition is not reused.

## Reproduction and complementary lifecycle checks

Run the new session scenarios from the repository root:

```sh
go test ./cmd/tfui -run '^TestSession_' -count=1
```

The existing deterministic suites cover timing cases without relying on scheduler luck:

```sh
go test ./internal/ui -run 'TestAssessRisk_|TestRiskPolicyDiagnostic_' -count=1
go test ./plugins/risk -run 'TestAssessment_|TestCancel_|TestActivate_|TestPlanInvalidated_' -count=1
```

`internal/ui/risk_assessment_race_test.go` changes source bytes, selector, configuration or source existence
inside the evaluation boundary and requires `source_changed` with no publishable aggregate. The Risk lifecycle
tests deliver a late old-Context assessment after the new assessment, and cancel an in-flight acquisition
before delivering its result. They require cancellation of the old operation, rejection of stale rows and a
terminal state for headless execution. These are controlled boundary tests; they do not claim a hard memory
or forced-termination limit for arbitrary Rego.

The source/configuration suites also exercise actual root, Member and workspace resolution, including explicit
built-in reset and Member plugin-block rejection. The policy-failure Session tests preserve Plan review,
fixed diagnostics and empty Risk stdout. Successful fixture review does not supersede those failure cases.

## Verification results

The final code and UI checks passed individually:

| Command | Result |
| --- | --- |
| `mise run check:build` | Passed both Quick and Done builds. |
| `mise run test:unit ./cmd/tfui/...` | Passed, 356 tests. |
| `mise run check:lint` | Passed, zero issues. |
| `mise run test:unit` | Passed, 5,145 tests, including existing race/cancellation and failure checks. |
| `mise run test:coverage` | Passed, 100.0%. |
| `go build -o .ralph-tui/tfui-24 ./cmd/tfui` | Passed; fresh binary used by the smoke task. |
| `mise run test:macro .ralph-tui/tfui-24` | Passed after the two obsolete smoke expectations were corrected. |
| `go test -tags integration ./tests/integration -run 'TestMacro/risk' -count=1` | Passed; the integration runner builds its own fresh binary. |
| `git diff --check` | Passed. |

Independent code-checker review found no findings in the session tests, fixture corrections, evidence or
updated smoke assertions. Production Go code and Apply behavior were not changed.

The next integration bead should run its preservation checks against these corrected fixtures and retain
the distinction between excluded changes, explicit classifications and unavailable assessment.
