---
layout: default
title: Risk policy acquisition call-count evidence
parent: Development
description: "Records live and seeded acquisition counts, startup seed loss and same-artifact evidence reuse."
audience: "Contributors reviewing D-02b and preserving plan acquisition in US-001 and US-004"
---

# Live and seeded acquisition

This D-02b experiment inspects baseline `8f56f88e03618d47677b012548c5003a57d7d2ab` on
`ralph/risk-plugin`. It uses disposable overlay tests against the actual `ExecService` and
`Session.RunPlugin`; it changes no product code. The [contract](../../reference/risk-policy-contract.md)
owns eligibility and failure behavior, and [D-02a](input-fidelity.md) owns representation fidelity.

[US-008a session verification](session-integration.md) records the later production integration tests and
repository fixture corrections. The baseline failures below remain historical evidence.

## Criteria declared before execution

The [protocol](assets/acquisition/protocol.md) was written before the first run. A seeded reviewed plan
must be available through real Session startup, including the initial selected Chdir, without a new plan.
A missing Terraform binary distinguishes a seed hit from fallback acquisition. Root startup is the control.
Failure at initial Chdir rejects the current lifecycle as sufficient for future policy acquisition; observing
that failure successfully does not mean the candidate passed.

Count Terraform `plan`, `show`, `version` and workspace calls separately. Reject calling `PlanJSON` to
retrieve an already reviewed plan's policy evidence if it generates another plan. A same-artifact experiment
must preserve exact source JSON bytes and their SHA-256 after the original source changes, with no additional
plan call. Reconstructing original evidence from display-string summaries is rejected by the reproduced
[D-02a baseline](input-fidelity.md#results); these counts do not repair that loss.

## Reproduction and limits

The [test source](assets/acquisition/acquisition_test.go.txt) is injected into `cmd/tfui` using a Go
overlay, so it can call the real Session without adding tests or branches to production packages. The
[runner](assets/acquisition/run.sh) keeps generated files, temporary artifacts and Go build caches inside
this worktree. Run from the worktree root with the repository Go toolchain:

```sh
mise exec -- bash docs/development/risk-policy-evidence/assets/acquisition/run.sh
ACQUISITION_CANDIDATE=1 mise exec -- bash docs/development/risk-policy-evidence/assets/acquisition/run.sh
```

The second command intentionally exits nonzero while initial-Chdir seed preservation remains absent. Its
assertion is retained separately from passing baseline observations; no product check is weakened to hide it.
Environment: Go 1.25.5 linux/amd64 and Python 3.13.14. No Terraform provider, real Terraform process, OPA or
policy engine is used. The counted [fake Terraform](assets/acquisition/fake_terraform.py) implements only
version, workspace show, plan and show JSON. It stores synthetic JSON in the saved artifact and reads it back;
that artifact is **not a Terraform binary plan**. This proves the acquisition call boundary and snapshot
mechanism with controlled bytes, not provider fidelity or real Terraform performance. Every unexpected command
fails. Any real Terraform binary-plan/show equivalence still requires downstream integration verification.

The first test-writer run used the existing `tests/fixtures/plan.json` and a nonexistent binary. The root
returned both expected resource rows, empty stderr and exit 2; the Member run returned empty stdout,
missing-binary stderr and exit 1. The [first red capture](assets/acquisition/candidate-red.txt) preserves this
failure before the fake fixture was implemented. Exit 2 is existing Plan behavior, not a proposed risk-command
threshold or exit contract.

## Observed counts

The [baseline capture](assets/acquisition/baseline.txt) records five passing experiment functions, including
file/byte seed variants and four live/seeded Session startup variants. Counts below are cumulative within a
scenario; each Session row starts fresh. `version` and `workspace` are subprocesses too, even when no plan runs.

| Scenario | plan | show | version | workspace |
| --- | ---: | ---: | ---: | ---: |
| Direct seeded Plan, twice, file or bytes | 0 | 0 | 0 | 0 |
| Direct live Plan | 1 | 1 | 1 | 0 |
| Then initialize Risk and Analyze that summary | 1 | 1 | 1 | 0 |
| Then read raw JSON from the same artifact after source edit | 1 | 2 | 1 | 0 |
| Then call live Plan again | 2 | 3 | 2 | 0 |
| Then call PlanJSON | 3 | 4 | 2 | 0 |
| Fresh seeded service, call PlanJSON | 1 | 1 | 0 | 0 |
| Live Session, root | 1 | 1 | 2 | 1 |
| Live Session, initial Member | 1 | 1 | 2 | 1 |
| Seeded Session, root | 0 | 0 | 1 | 1 |
| Seeded Session, initial Member | 1 | 1 | 2 | 1 |

The Session variants assert Plan's existing exit 2, expected resource summary and empty stderr, and verify live
plan subprocesses run in the selected directory. The normalized process logs in the capture retain command
arguments and working directories. They distinguish workspace discovery from plan execution in the initial directory.
The Member seeded case only appears to succeed with the fake binary because it silently reacquires a plan;
its call count rejects reuse. With a nonexistent binary it fails, as the first red run demonstrates.

Same-artifact retrieval retains **978 bytes**, SHA-256
`bd37a13a379c79e11e5a2dd5aa0f07927d4ede03af4bf5eb3132bc231cc224ae`, equal byte for byte to the
tracked [plan fixture](https://github.com/lmarqs/terraform-ui/blob/8f56f88e03618d47677b012548c5003a57d7d2ab/tests/fixtures/plan.json). The source copy is replaced with an empty plan
before retrieval, so reading the current source instead of the saved artifact would fail the assertion.
No extra plan is generated in this step. The initial typed `Plan` already did a show; this experiment's second
show demonstrates a recovery mechanism. Production should retain evidence at acquisition rather than require
another show solely for policy data when those bytes are already available.

## Current code and first-version conclusion

`internal/terraform/exec/service.go` short-circuits `Plan` on `cache.GetPlan()`, preserving the existing seeded
regression behavior. A live Plan returns a summary without `SetPlan`; calling Plan again reacquires it.
`PlanJSON` has no seeded cache shortcut and always runs plan then raw show. Therefore it is not an accessor for
an existing reviewed plan's JSON, despite preserving bytes for the new plan it generates.

`ServiceCache.SeedPlan` produces the parsed display summary; file seeds retain a pathname and byte seeds retain
input data privately. Neither exposes lossless policy evidence through the current SDK. Retaining the original
source bytes before display conversion is feasible without another Terraform plan, but is still US-001 work.
File content must be captured as part of that reviewed acquisition, rather than reread later and assumed identical.

`Session.RunPlugin` seeds the base service before `ui.App.Init`. `startupContext` uses `WithDir` when Chdir is
set; `ExecService.WithDir` creates a new empty cache. This rejects the existing scoped-seed lifecycle as a
sufficient implementation. Preserve the direct `Plan` cache short-circuit while fixing startup propagation
with Context identity tests in US-001; indiscriminately sharing seeds across all future directories would not
establish correct identity. No such fix is included here.

Risk's current `Analyze(summary)` consumes shared classifications with zero acquisition calls. The probe initializes
it using `sdktest.NewDeps` and calls Analyze explicitly; the registry does not wire this call automatically.
This is evidence that the existing consumer can reuse a reviewed summary, **not** a working Plan-to-Risk session
flow, policy input fidelity or evaluated reasons. Do not claim a `tfui risk` command or production policy path
exists on the strength of these experiments.

Retain acquisition-time lossless evidence and its plan/Context identity as the first-version direction. Reject
both a second `PlanJSON` call and the assumption that current initial-Chdir cache propagation already works.
No production integration, Apply change, configuration change, policy decision or dependency is added. Actual
source selection, workspace transitions, reload, A-to-B race rejection and cancellation remain separate lifecycle
experiments; this bounded call-count task does not establish them. Real binary-plan fidelity and provider behavior
remain downstream verification.

## Delivery validation

The coordinator inspected the complete assets and current acquisition code. The code-checker independently
reran the corrected baseline: all five experiment functions passed, with no outstanding findings. Review caught
and fixed the fake's initial omission of `workspace show -no-color`; the final baseline uses that supported
invocation. Captures replace the absolute checkout prefix with `WORKTREE`; Session process logs additionally
replace each fixture directory with `WORK`. The first red capture retains its historical `.tmp` subdirectory;
the current runner uses ignored `.ralph-tui/d02b` instead.

- Disposable acquisition overlay: **PASS**; five experiment functions, including live/seeded Session cases.
- Initial-Chdir preservation candidate: **REJECTED**; assertion failure retained, reproduced on the final harness.
- `mise run docs:build`: **PASS**.
- `mise run docs:check`: **PASS**; 76 HTML files checked after the build.
- `git diff --check` and new-file whitespace checks: **PASS**.
- `mise run check:build`: **NOT APPLICABLE / not run**; no production Go package changed. The overlay is compiled.
- `mise run test:unit ./<affected-package>/...`: **NOT APPLICABLE / not run**; no affected production package.
- `mise run check:lint`: **NOT APPLICABLE / not run**; documentation and explicitly run disposable assets only.
- `mise run test:unit`: **NOT APPLICABLE / not run**; no product behavior changed.
- `mise run test:coverage`: **NOT APPLICABLE / not run**; no production executable branches added.
- `mise run test:macro`: **NOT APPLICABLE / not run**; no UI change.

Passing observations of a rejected candidate do not certify that the future production lifecycle works.
