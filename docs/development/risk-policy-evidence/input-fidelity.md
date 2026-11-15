---
layout: default
title: Risk policy input fidelity evidence
parent: Development
description: "Records reproducible resource-local input fidelity, neighbor independence and phantom normalization experiments."
audience: "Contributors reviewing D-02a and implementing policy evidence acquisition"
---

# Per-resource fidelity and independence

This D-02a experiment tests the [D-01a tagged-value candidate](first-version-candidates.md), using disposable
assets rather than production integration. The inspected baseline is `792cef52cd1df6378794d82d1b63d0779cf70fcb`.
The canonical [contract](../../reference/risk-policy-contract.md) still owns eligibility and failure semantics.

## Criteria recorded before execution

The test-writer writes independent expected-value assertions before the projection prototype. A candidate passes
these bounded fixtures only if source numeric token text, ordered lists, missing versus null, nested/whole-object
unknown and sensitivity masks, identity and both replacement action orders survive projection. Sensitive values
must be absent from projected input. Unknown/sensitive subtree collapse is intentional; all other silent loss
rejects the candidate. Inconsistent mask shapes must fail rather than fabricate evidence.

Adding, removing or reordering neighbors must leave the selected resource's serialized input and deterministic
experimental classification identical. The classifier accepts only the projected envelope; this demonstrates
the structural boundary, not OPA execution, evaluator purity or a production acquisition guarantee.

Array reordering and removal of a null object field must be distinguishable in projected input and usable by
the experimental classifier, even when the current phantom detector labels the mutation phantom. The actual
repository detector must be exercised, rather than accepting a Python imitation as evidence. A phantom marker
cannot exclude a mutation from assessment. These counterexamples establish limitations of the advisory heuristic;
they do not establish provider-specific semantics or reviewed severity assignments.

An additional baseline probe compares the current `LoadPlan` display value with `terraform-json` decoding with
`UseJSONNumber(true)`. A large integer or exponent spelling lost by the display path rejects that path as the
source of policy evidence; it does not reject lossless ingestion from the original plan JSON. This probe does
not repair the loader or authorize a second Terraform plan.

## Prior spike inspection

Read-only inspection of `/home/lmarqs/Work/terraform-ui/.worktrees/feat/risk-plugin/.superpowers/sdd/prd-risk`
returned “No such file or directory”. No evidence from that absent spike is reused or counted as complete.
The current worktree also has no `.superpowers` directory before this experiment. Assets below are fresh.

## Reproduce

Environment: Python 3.13.14; Go 1.25.5 linux/amd64; repository-pinned `terraform-json` v0.27.2. No Terraform
binary, provider, OPA or network service is used. The [source fixture](assets/fidelity/source-plan.json) is
synthetic Terraform-show-shaped JSON, not a newly generated provider plan. All secret markers are synthetic.

Run from this worktree's root. The Go source has a `.txt` suffix to keep it out of production package discovery;
its executable copy stays in the runner's ignored directory inside this worktree. The programs use assertions
and nonzero exit results on a changed baseline or failed candidate assertion.

```sh
python3 -B -m unittest discover -s docs/development/risk-policy-evidence/assets/fidelity -p test_fidelity.py -v
mkdir -p .ralph-tui/d02a
cp docs/development/risk-policy-evidence/assets/fidelity/baseline.go.txt .ralph-tui/d02a/main.go
mise exec -- go run ./.ralph-tui/d02a
```

The [test suite](assets/fidelity/test_fidelity.py) contains hand-derived expected nodes rather than encoding and
decoding through the same implementation. The test-writer first ran all 11 tests without the prototype: all
failed the assertion `candidate projection prototype is missing`. The coordinator then wrote the
[disposable candidate](assets/fidelity/candidate.py), inspected the tests, and reran them: **11 passed**.
The [combined unittest capture](assets/fidelity/tests.txt) records the successful execution. The coordinator
also strengthened neighbor coverage with a neighbor that would receive a different classification, and compared
canonical serialized envelopes as well as structural equality; the same 11 tests passed again.

## Results

| Evidence | Observed result | Conclusion for this candidate |
| --- | --- | --- |
| Large integer `9007199254740993`; exponent `1e3` | Exact number-node strings, including resource index | Lossless token ingestion works for these fixtures; no claim about evaluator arithmetic. |
| Ordered `steps` and mixed null/string/sensitive array | Original order and positions retained | No phantom normalization is used in policy input. |
| Missing object key, known null, absent create-before/delete-after | Missing entry, null node and absent node remain distinct | Action-qualified absent sides do not erase attribute nulls. |
| Nested/mask-only unknown; whole after unknown | Unknown nodes retained, ancestor collapse exposes no children | Unknown values are not reconstructed from display strings. |
| Before/after nested and whole sensitive masks; unknown+sensitive | Opaque sensitive or sensitive_unknown nodes | Synthetic markers and hidden subtree keys absent from input/result capture checked by assertions. |
| Invalid scalar/object and array-mask shapes | Explicit `ValueError` | The tested inconsistent shapes are rejected rather than guessed. |
| Address, previous address, mode, provider, type, name, absent/string/numeric index | Exact expected resource envelope | Resource-local identity survives; no plan-wide metadata is exposed. |
| Both replacement action arrays and replacement paths | Original action/path order and integer path index retained | Neither replacement order becomes a generic unordered flag. |
| Neighbor inserted before/after, then removed | Identical serialized target envelope and classification; source unchanged | This resource-local API is structurally independent for these cases. |

The actual [Go baseline probe](assets/fidelity/baseline.go.txt) produced
[stdout](assets/fidelity/baseline.stdout.txt) below and [empty stderr](assets/fidelity/baseline.stderr.txt), exit 0:

```text
array-order: phantom=true
null-field: phantom=true
LoadPlan: large=9007199254740992 exponent=1000
UseJSONNumber: large=9007199254740993 exponent=1e3
```

`internal/terraform/loader.go` currently decodes without enabling `UseJSONNumber`; `plan_parser.go` converts
attribute values to display strings. The measurement rejects **that display path** as the policy evidence
source: the large integer is rounded and the exponent spelling is lost. Enabling number-preserving decoding in
the disposable probe retains both tokens. This is feasibility evidence, not a change to acquisition or a proof
that the current cache retains the required raw evidence.

## Phantom counterexamples and advisory limits

The Go probe calls the repository's actual `IsPhantomChange` on two update diffs: `["allow","deny"]` to
`["deny","allow"]`, and `{"optional":null}` to `{}`. Both return true. The implementation in
`internal/terraform/phantom.go` sorts array elements and removes null object entries without provider schema
information; it cannot establish that these distinctions are irrelevant to every resource or policy.

The Python suite changes those same fields in the projected resource. Its deliberately synthetic classifier
observes first-element order and the presence of `optional`: baseline gives `low/baseline`, reversed steps gives
`medium/order-sensitive`, and absent optional gives `high/null-sensitive`. These are observability probes,
**not approved resource severities**. Each result uses explicit level/reasons. The Go examples isolate the
normalization behavior on one diff; the richer Python fixture additionally exercises masks and other changes.
No claim is made that the entire rich fixture is marked phantom.

The input has no `IsPhantom` field and projection never skips these mutations. Phantom remains advisory;
eligible phantom mutations must still receive normal policy evaluation under the contract.

## First-version conclusion and remaining boundaries

Retain the D-01a tagged-value candidate for subsequent experiments: the tested permitted evidence can be
represented without float conversion, array sorting or null removal, with intentional sensitive redaction.
Preserve original plan JSON or an equivalent lossless representation before deriving display strings. Do not
try to recover policy evidence from `AttributeDiffs`. No production loader, classifier, Apply path or dependency
is changed by this experiment.

This is a bounded adapter prototype, not a complete schema validator. Importing metadata deliberately raises an
error because this experiment has no acquisition evidence establishing its masking. Unsupported Python value
types and missing required metadata also fail rather than generating neutral classifications. Additional Terraform
formats, import identity/masking, all malformed-mask combinations and provider-generated fixtures remain to be
verified during acquisition implementation; these tests do not claim exhaustive input support. OPA execution,
purity/capability enforcement and large-number arithmetic belong to D-03. Live/seeded call counting, real Session
startup, Context/source selection and stale-result rejection belong to D-02b, not this resource-local experiment.

Delivery review independently reran the 11 tests and Go probe and found no blocking issue. It identified a
specific validation gap: opaque ancestors collapse before descendant-mask validation. For example,
`encode({"x": 1}, {"x": "not-a-mask"}, True)` returns a sensitive node instead of rejecting the invalid mask leaf.
The general inconsistent-mask rejection criterion is therefore **not established** by this prototype; only the
three tested malformed shapes are covered. Production validation must reject invalid masks before projection.
This limits adapter acceptance, while leaving the demonstrated representation fidelity and independence intact.

## Validation boundary

The Python suite (11 tests) and Go probe passed as described above, each with exit 0. `mise run docs:build`
passed; the subsequent `mise run docs:check` passed (75 HTML files). `git diff --check` and additional whitespace
checks over the new evidence files passed. Product Quick/Done checks (`check:build`, affected-package `test:unit`, `check:lint`,
full `test:unit`), `test:coverage` and `test:macro` are not applicable: only documentation and disposable evidence
assets change, with no production Go package or UI behavior changed. The Go probe itself is compiled and run
explicitly by the command above. No unrun product check is counted as passed.
