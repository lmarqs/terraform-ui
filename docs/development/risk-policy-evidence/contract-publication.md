---
layout: default
title: Risk policy first-version publication review
parent: Development
description: "Records D-01c contract decisions, evidence applicability, residual risks and publication verification."
audience: "Contributors reviewing the first-version publication and beginning production implementation"
---

# First-version publication review

D-01c publishes the [v1 public contract](../../reference/risk-policy-contract.md) and
[ADR-0026](../../adr/0026-publish-first-version-risk-contract.md) under the owner's delegated authority.
The coordinator selects exact interfaces, reviews the candidates/evidence and owns delivery fixes; the owner
stresses and refines them after merge. This is a planned interface ready for dependent implementation, not
an owner-selected interface, implemented evaluator or available command.

## Candidate disposition and evidence applicability

Publication baseline: `ff2dde6` (D-03c delivery). Current `pkg/sdk/risk.go` still uses a zero-valued integer
severity with no validity, `internal/terraform/risk.go` retains the fixed classifier, Risk still ignores
configuration, and Blast Radius still derives impact from `PlanChange.Risk`. Member plugin blocks remain
explicitly rejected in `internal/config/hcl_child.go`. No production dependency or policy integration changed
between the evidence commits and this documentation-only publication, so the recorded observations remain
applicable at their stated scope. The evidence assets are not product code and are not modified here.

| Candidate | Publication decision | Supporting evidence and limit |
| --- | --- | --- |
| D-01a input | Adopt tagged nodes and source numeric tokens; validate masks before projection. | [D-02a](input-fidelity.md), reproduced by [D-02c](context-source-identity.md), preserves fixture fidelity and independence. Opaque-ancestor malformed-mask validation and import masking are not proven; unsupported evidence fails explicitly. No arbitrary-precision arithmetic claim. |
| D-01a result/trust | Adopt single complete document, strict shape, deterministic positive capabilities, marked-value redaction and fixed diagnostics. | [D-03b](boundary-controls.md) records 40 probes, print erasure/AST rejection and unmarked/authored disclosure. Cardinality defense and all reason/path bounds still need product tests. Exclude four experiment-only arithmetic builtins from the published profile. |
| D-01b source | Adopt dedicated selector, declaration provenance, complete replacement, framed digest, boundary reload and generation guard. | [D-02c](context-source-identity.md) tests a disposable model and actual config/session observations, not production concurrency. Selected-path recheck matters even for identical bytes. Preserve Member plugin-block rejection. |
| Acquisition/lifecycle | Preserve original reviewed evidence and seed short-circuit; repair initial-Chdir seed propagation through real Session tests. | [D-02b](acquisition-counts.md), reproduced by D-02c, rejects initial-Member seeded startup and PlanJSON retrieval as existing solutions. No second plan solely for policy evidence. |
| Runtime/cost | Select measured OPA v1.19.1 as initial integration pin, accept observed costs with a narrow trusted-author promise. | [D-03c](release-costs.md): all four release targets build, 80 timed builds/90 runtime samples. Native non-host runtime, complete-policy costs and hard cancellation bounds remain unproven. [Maintenance review](d03-preparation/maintenance.md) is not current-support/security approval. |
| Aggregation/report/consumers | Adopt complete-only maximum, off-by-default inclusive threshold, versioned successful reports, exit 3 for threshold met and unavailable impact for missing classifications. | D-01a rationale and current Blast Radius arithmetic inspection; CLI, result validation and consumer regressions remain production stories. |
| Helpers/templates/built-in | Adopt explicit vendoring and current fixed severities, with evidence-based Lambda reasons. | D-01b source inspection; author workflow/full templates remain US-007. No new layer severity or blanket lowering is inferred from the field report. |

These are concrete first-version decisions despite bounded or falsified integration assumptions. No unsuccessful
experiment is converted into proof of working production behavior. The design's
[decision map](../risk-policy-design.md#first-version-decision-map) replaces open contract gates with canonical
links, including each original issue number. It is the local outcome map; no remote issue update/closure is claimed.

## Publication refinements

The reference replaces illustrative `local-policy`, `sha256:example` and `reviewed-plan-1` identities with exact
formats and computed examples. Source identity uses D-01b framing; plan identity hashes the original show JSON
bytes. The reference fixes successful text ordering, diagnostic categories, no-op/empty report encoding and
source validation on empty eligibility. Report/input versions remain independent; incompatible-only policy
entrypoints fail undefined rather than silently rewriting sources or querying another major.

The publication uses the measured OPA version/profile as the integration choice, accepting the material binary
and cold-build growth without inferring a numerical budget. Production must review advisories and shared MVS
upgrades and repeat controls if changing the pin. Numeric strings promise ingestion fidelity, not unmeasured
arbitrary-precision operations. The positive profile omits experimental arithmetic controls explicitly.

Risk uses the shared diff renderer, two inline changed paths plus overflow/inspection, resource-only cursor
sequence, standard gutter/count and separate unavailable/none/empty presentation. Blast Radius retains impact
arithmetic and count/action semantics, using explicit validity rather than hidden built-in risk or zero on error.
Thresholds remain tfui interpretation, never policy input or Apply enforcement.

## Residual risks and downstream proof

The [design risk table](../risk-policy-design.md#consequences-and-confirmation) owns mitigations and contingencies.
Retain authored/unmarked disclosure, misleading reasons, provider-version uncertainty, cooperative cancellation,
non-atomic hostile filesystem swaps, older runtime maintenance, dependency upgrades, and unknown full-policy
costs. Bounded evidence supports feasibility; none of these risks is eliminated by publication.

US-001/002/003/004 must prove lossless acquisition, seed propagation, strict real HCL selection, mask/result/path
validation, runtime profile, shared assessment identity and terminal failure/cancellation behavior. US-005/006/007
must prove rendering, report/exit/threshold behavior and author examples. US-008/009 own integration checks and
implemented availability docs. Plan JSON remains Terraform pass-through; saved-artifact ownership and Apply
behavior remain unchanged. The PRD records this sequencing instead of waiting for exact owner selections.

## Review and verification method

The test-writer reviewed the pre-edit docs and recorded eight failing publication checks: input/result,
source/lifecycle, aggregation/states, trust/diagnostics, report/threshold/exits/identities, Blast Radius,
design/PRD gates and first-version availability/authority. This prose-only task adds no product tests.
The coordinator publishes the reference, checks exact digest preimages/examples, reviews every decision-matrix
row against the reference and runs docs build then link validation. A code-checker reviews the final diff,
evidence claims, consistency and downstream gates before completion.

Reproduce documentation validation from the repository root:

```sh
mise run docs:build
mise run docs:check
git diff --check
```

Individual check outcomes are appended to the run's `.ralph-tui/progress.md`. Production Quick/Done build,
affected/full unit, lint, coverage and UI macro gates are not applicable and not run: this task changes only
contract/design/evidence/PRD prose, without Go modules, executable evidence assets or product/UI behavior.
No commit, push, PR, bead command or tracking status change is part of this delivery.

Publication review outcome: the test-writer's eight checks changed from red to green; the code-checker reported
no material findings. Both the coordinator and reviewer recomputed the exact source/plan digests, parsed the
JSON fences and checked report counts; the reviewer also compared Blast Radius examples with current arithmetic.
`docs:build` passed. The first `docs:check` found one historical link to the renamed remaining-decisions heading;
its target was updated to the first-version map. The subsequent build and `docs:check` passed (80 HTML files,
163 internal links). `git diff --check` passed. No production gate is counted as passed or implied by these results.
