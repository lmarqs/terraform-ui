---
layout: default
title: Risk policy first-version candidates
parent: Development
description: "Records concrete input, result, trust and consumer decisions for the risk policy fidelity and runtime experiments."
audience: "Contributors running D-02 and D-03 and publishing the D-01c contract"
---

# First-version input, result, trust and compatibility candidates

D-01a prepares these decisions for the [architecture design](../risk-policy-design.md#remaining-decisions).
They are contributor-selected first-version choices under the delegated decision authority for this run, not
interfaces selected by the owner or implemented behavior. D-02 and D-03 must test them before D-01c publishes the
production interface in the [contract reference](../../reference/risk-policy-contract.md). A falsified candidate
must be revised there before dependent implementation starts. No production dependency is selected here.

The existing contract still owns independent classification, eligibility, complete custom replacement and
failure behavior. This page owns candidate detail until publication; guides must not advertise these examples
as available. The [D-01b source and template decisions](source-template-decisions.md) own source selection/reload,
snapshot identity, explicit helper reuse and template/upgrade choices; this page does not duplicate those mechanics.

## Current-checkout evidence and reproducibility

Inspected baseline: `a01f2305ebc977df575795df1278c46a0d23f443`. Reproduce the observations from the repository root:

```sh
git show a01f2305ebc977df575795df1278c46a0d23f443:pkg/sdk/change.go
git show a01f2305ebc977df575795df1278c46a0d23f443:pkg/sdk/risk.go
git show a01f2305ebc977df575795df1278c46a0d23f443:internal/terraform/plan_parser.go
git show a01f2305ebc977df575795df1278c46a0d23f443:plugins/blastradius/blastradius.go
git show a01f2305ebc977df575795df1278c46a0d23f443:internal/config/hcl_test.go
```

`PlanChange` retains display-string diffs and an integer `Risk`; its zero value is `RiskNone`.
`OverallRisk` returns the maximum, including `none` for an empty slice. Neither represents assessment validity.
`calculateImpact` combines maximum risk with change counts and destructive actions. These observations justify
separate status and consumer validity checks rather than extending the meaning of a zero integer.

The current `TestLoadChild_WhenPluginBlockPresent_ShouldRejectWithRootDefaultsHint` explicitly expects rejection
of Member plugin blocks, including workspace blocks. Tests constructing options directly and testing per-key
merge do not establish support for those blocks in real HCL. Historical silently-ignored-block reports are not
a current source-selection contract. D-01b/D-02 must exercise actual loading and resolution without restoring that bug.

## Decision matrix

| Boundary | First-version choice | Alternative and reason for declining it | Evidence required |
| --- | --- | --- | --- |
| Input | Versioned envelope containing one mutation and recursively tagged value trees. Numbers use exact decimal strings. | Native untyped JSON is easier to author against but makes accidental float conversion and missing/null sentinels harder to detect. Display diffs already lose evidence. | D-02 lossless source-to-input comparison, except intentional sensitive redaction. |
| Entrypoint | Rego v1 package `tfui.risk.v1`, complete document `classification`; query `data.tfui.risk.v1.classification`. | Finding sets and booleans obscure completeness and independent ownership. | D-03 cardinality, undefined and conflict probes. |
| Result | Exactly one object, `level` and nonempty `reasons`; no extra fields. | Implicit defaults, priority/first-match merging and tfui choosing the greatest returned finding hide policy ownership. | D-03 strict shape validation and competing rules. |
| Unknown/sensitive | Unknown is a node; sensitive subtrees are opaque nodes before OPA sees input. | Giving trusted code plaintext enables attribute tests but also makes routine reason/error output a disclosure channel. | D-02 nested masks; D-03 disclosure probes. |
| Trust | Trusted local project code, reduced deterministic capabilities; no hostile-code sandbox promise. | Arbitrary untrusted policies require isolation/resource guarantees not established by embedded evaluation. | D-03 capability and process-limit experiments. |
| Aggregation | tfui takes maximum only for a nonempty, fully successful assessment. Counts retain all levels. | Weighted scores invent meaning beyond independent classifications. A maximum over partial successes hides failure. | Empty/none/partial/failure matrix below. |
| Threshold | Disabled unless explicitly configured or passed; inclusive comparison in tfui. | A default `high` would silently introduce a caller decision unsupported by the classification contract. | Boundary equality and precedence examples below. |
| Report | Versioned tfui report; successful reports on stdout, operational failures on stderr with empty stdout. | JSON failure reports on stdout would change the existing headless failure contract. | D-01c publishes the schema and CLI contract together. |
| Blast Radius | Use selected-policy classifications with the existing impact formula only when that group's required risk inputs are complete. | Retaining hidden built-in risk contradicts custom ownership; treating missing risk as zero misrepresents failure. | Group-level completeness and unchanged arithmetic regression tests in US-004. |

## Exact experimental input

This envelope is the D-02/D-03 input candidate, independent of Go types. No neighboring resource, plan-wide
variables, configuration, state, policy path, workspace or wall-clock value is exposed. Context and provenance
identify an assessment outside policy input. Resource identity is retained because type/address-aware rules are
in scope; move/import metadata is resource-local evidence, not an eligibility bypass.

```json
{
  "schema_version": "tfui.risk.input/v1",
  "resource": {
    "address": "aws_lambda_function.worker",
    "previous_address": null,
    "mode": "managed",
    "type": "aws_lambda_function",
    "name": "worker",
    "provider_name": "registry.terraform.io/hashicorp/aws",
    "index": {"kind": "absent"}
  },
  "change": {
    "actions": ["update"],
    "importing": null,
    "replace_paths": [],
    "before": {"kind": "object", "entries": {
      "memory_size": {"kind": "number", "value": "128"},
      "source_code_hash": {"kind": "null"},
      "environment": {"kind": "sensitive"}
    }},
    "after": {"kind": "object", "entries": {
      "memory_size": {"kind": "number", "value": "256"},
      "source_code_hash": {"kind": "unknown"},
      "environment": {"kind": "sensitive"}
    }}
  }
}
```

Envelope keys are required. Missing provider/resource/action metadata needed to interpret a mutation is incomplete
input, not an empty plan. `previous_address` is a string or null; `index` is an absent, number or string node.
`importing` is null or an object with `id` as a value node and `unknown` as a boolean, preserving a resource-local
import identifier's sensitivity/uncertainty. If acquisition cannot establish its masking, omit no evidence silently:
fail that input. `replace_paths` is an ordered array of paths, each an array of string keys or integer indexes.
Both action arrays `["delete","create"]` and `["create","delete"]` are retained in their original order.
Do not translate them into an unordered replacement flag. Eligibility is decided before evaluation according to
the contract, including mutations accompanying move/import and eligible phantom mutations.

Value nodes have exactly these forms; inappropriate or additional members fail input validation:

| Source state | Node example | Meaning |
| --- | --- | --- |
| No object key | No entry in `entries` | Absence; querying a missing key in Rego is undefined. |
| Absent whole before/after side | `{"kind":"absent"}` | No prior/new object, distinct from known null. |
| Known null | `{"kind":"null"}` | Explicit null retained. |
| String | `{"kind":"string","value":"artifact.zip"}` | Exact string. |
| Boolean | `{"kind":"boolean","value":false}` | JSON boolean. |
| Number | `{"kind":"number","value":"9007199254740993"}` | Source JSON numeric token as a string, without float conversion. |
| Ordered list | `{"kind":"array","items":[{"kind":"null"},{"kind":"number","value":"2"}]}` | Order, indexes and null entries retained. |
| Object | `{"kind":"object","entries":{"x":{"kind":"null"}}}` | Object key order has no semantic meaning. |
| Unknown subtree | `{"kind":"unknown"}` | Value not known; no invented null value or child structure. |
| Sensitive subtree | `{"kind":"sensitive"}` | No value or child structure exposed. |
| Both unknown and sensitive | `{"kind":"sensitive_unknown"}` | Both facts retained, no content exposed. |

Number strings must be valid JSON numeric tokens. Source spellings such as `1e3` are preserved; authors explicitly
convert with `to_number` when numeric comparison is wanted. String equality compares spellings, not numeric value.
D-02 must establish exact ingestion and D-03 must test numeric operations on large values before this choice is
published; source lexical preservation is not a claim of unbounded evaluator arithmetic precision.

Build object entries from the union of value keys and unknown/sensitivity mask keys: a key present only in an
unknown mask becomes an unknown node, not an absent entry. A mask-only sensitive key whose existence is
established becomes sensitive (or sensitive_unknown); inconsistent masks fail validation. Derive an absent
whole before/after side only from a create/delete action plus its corresponding source null side; a present
attribute with null remains a null node. D-02 must verify these distinctions against actual source JSON.

Apply before/after sensitivity masks separately. A true mask on an ancestor replaces the entire subtree.
Nested object/array masks apply recursively; an ancestor unknown collapses its subtree to unknown, or
`sensitive_unknown` if sensitivity is also established there. An inconsistent mask/shape or unsupported source
format fails evidence validation rather than guessing a value. Unknown/sensitive nodes deliberately disclose
no children; their omission is the stated redaction boundary, not accidental input loss. No phantom normalization
(sorted arrays or deleted null keys) is permitted in this input. `IsPhantom` is not policy input or an exclusion.

## Exact experimental result and rule resolution

The result has exactly `level` and `reasons`. `level` is one of `none`, `low`, `medium`, `high`, `critical`.
`reasons` is an array of 1–16 objects, each with exactly `code`, `message`, `paths`:

```json
{"level":"low","reasons":[{"code":"project-default","message":"Explicit project default for this mutation.","paths":[]}]}
```

`code` matches `[a-z][a-z0-9-]{0,63}`. `message` is 1–512 Unicode code points of nonblank text, with no terminal
control characters. Each `paths` array contains at most 16 paths into before/after values, such as
`["after","memory_size"]`; elements after the initial `before`/`after` are string keys or nonnegative integer
indexes. Paths may name an unknown/sensitive node but cannot traverse beneath it or refer to a nonexistent node.
Messages are trusted author explanations, not verified causal claims. Render them as plain text, never ANSI/markup.
These bounds keep review output usable; they are not resource limits for the evaluator.

A complete minimal experimental module is:

```rego
package tfui.risk.v1

import rego.v1

default classification := {
    "level": "low",
    "reasons": [{
        "code": "project-default",
        "message": "Explicit project default for this mutation.",
        "paths": [],
    }],
}
```

This default severity is a fixture choice, not an accepted built-in rule or a proposed Lambda severity change.
A project must explicitly own any default. The package/query has one complete-document value. Multiple matching
rules returning the identical object are acceptable; differing objects conflict and fail. Authors must make
specialized cases mutually exclusive or resolve them explicitly in Rego. tfui does not impose rule order,
merge reasons or select maximum severity among competing outputs.

| Evaluator observation | tfui candidate treatment |
| --- | --- |
| No result / undefined query | Resource failure `result_undefined`, even without a Go error. |
| Defined `[]`, `{}`, `null`, boolean or string | `result_invalid`, distinct internally from undefined. |
| Missing/unknown level, empty reasons, extra field, invalid reason/path | `result_invalid`. |
| More than one query result/expression value | `result_cardinality`; no arbitrary selection. |
| Competing complete-rule values | `evaluation_conflict`; no fallback. |
| Compilation error | Snapshot unavailable; no resource classifications. |
| Executed builtin error in strict mode | Resource evaluation failure; no default recovery by tfui. |
| Unreachable builtin error expression | Not proof of an evaluated error; test separately. |

Internal error categories support tests and diagnosis; public diagnostics use fixed messages with no raw OPA
error, source excerpt, interpolated value or stack trace. For example: `risk unavailable: result_invalid for
aws_lambda_function.worker`. A runtime/source failure may omit the address. Source-provided resource addresses
are trusted plan metadata and must also be escaped for terminal output.

## Experimental trust and capability profile for D-03

Select **trusted local policy authors with redacted sensitive plan values**. A policy can intentionally encode
constants or infer facts from nonsensitive input. tfui cannot certify that its explanations contain no secrets
or that a malicious policy terminates. Do not run unreviewed project policy as if embedded OPA were an isolation
boundary. Sensitive masking limits direct access to marked values; unmarked secrets and authored text remain a
residual disclosure risk. No claim of general non-disclosure or hostile-code containment is made.

D-03 should use this exact initial positive builtin allowlist (subject to compilation validation against its
chosen version): `eq`, `equal`, `neq`, `lt`, `lte`, `gt`, `gte`, `count`, `is_array`, `is_object`, `is_string`,
`is_number`, `is_boolean`, `is_null`, `object.get`, `object.keys`, `array.concat`, `concat`, `sprintf`, `startswith`,
`endswith`, `contains`, `lower`, `upper`, `to_number`. Language constructs needed by the example modules are
permitted; D-03 must record the exact compiler capabilities artifact, including version-specific internal
operators it requires. Adding a builtin requires a recorded justification, not use of the entire default registry.

Reject HTTP, DNS, clock, randomness, UUID generation and runtime/environment access by absence from capabilities.
No custom builtins, external resolvers, remote schemas or arbitrary bundle data. Use local source snapshots,
strict builtin errors, no query tracers, no print hook and print disabled. Explicitly reject `print` and `trace`
in source rather than relying on erasure or absence of a hook; D-03 must test this including unreachable calls.
Set network allowances empty as defense in depth; the positive builtin set is the primary capability candidate.
Do not pass another resource or global plan into `data`; policy modules/helpers remain selected source only.

The [pinned runtime research](../risk-policy-runtime-research.md) supports testing these controls, not asserting
that they already work. D-03 chooses a maintained candidate version after its own review and runs bounded fixtures
under an external process time/memory limit. Cooperative cancellation alone satisfies no hard resource guarantee.
Compilation/evaluation cost and every supported release target remain acceptance evidence to collect.

## Assessment, aggregation and user-facing states

An assessment retains each eligible resource's `classified` or `unavailable` status and each excluded change's
`not_assessed` status with its eligibility reason. Only a classified resource has a level and reasons.
A plan whose eligible set cannot be determined is unavailable, never empty. Reject an unsupported format/action
rather than silently dropping it. Terraform acquisition failure remains its own error, separate from policy failure.

| Situation | Assessment status | Overall level | Review text and threshold result |
| --- | --- | --- | --- |
| Zero eligible mutations, complete input | `empty` | null | `No eligible mutations (N changes not assessed).` Threshold `not_applicable`; successful command. |
| All eligible resources explicitly `none` | `complete` | `none` | `Risk: none — explicit policy classifications.` Compare only if threshold enabled. No safety claim. |
| All eligible resources classified | `complete` | Maximum ordered level | Counts and attributable reasons; threshold comparison eligible. |
| Input incomplete or policy cannot compile/load | `unavailable` | null | `Risk unavailable: input incomplete.` or fixed source/compile diagnostic. Command error. |
| One resource fails, others succeed | `partial` | null | `Risk unavailable: 1 of 3 eligible mutations failed; 2 classified.` Keep valid resource rows with reasons, mark failed rows unavailable. Command error. |
| All eligible evaluations fail | `unavailable` | null | `Risk unavailable: 3 of 3 eligible mutations failed.` Command error. |
| Cancelled or superseded Context/source | No publishable assessment | No value | Discard late results; cancellation follows existing signal exits. Never reuse the old aggregate. |

Partial evaluation may continue for the remaining finite resource list to aid TUI review; cancellation stops it.
No aggregate, threshold pass or success badge may be derived from a subset. Global input validation failure
invalidates the whole assessment; do not salvage rows from uncertain eligibility. Excluded rows may be listed
but do not contribute to severity counts or the maximum. Unknown values are valid evidence, not incomplete input.

## Command, threshold and report compatibility choices

Candidate invocation: `tfui risk -ci -plan plan.json -json -threshold high`. `-json` controls format independently
of mode. The future Risk setting is `threshold`, with values `off`, `none`, `low`, `medium`, `high`, `critical`;
explicit CLI `-threshold` wins over the resolved Risk setting, then default `off`. These are candidate names for
D-01c, not current flags/settings. No policy-returned threshold or threshold inside the Rego input.

Compare inclusively: with `high`, `medium` passes, `high` and `critical` meet the threshold. With `none`, every
nonempty complete assessment meets the threshold, including explicit `none`. Empty eligibility never matches.
`off` disables comparison; it does not disable assessment or hide errors. Invalid thresholds fail configuration.
The old ignored `level` key is not an alias: report an actionable configuration error (`risk setting level is
unsupported; use threshold`) and unavailable risk while preserving Plan review. D-01c must publish this migration
and add the command-specific exit extension to the CLI I/O contract before implementation.

| Terminal outcome | stdout | stderr after TUI | Exit |
| --- | --- | --- | --- |
| Complete below threshold or comparison off; empty eligible set | Text or JSON report | Empty | 0 |
| Complete and threshold met | Same complete report | `risk threshold met: high >= high` (actual levels substituted) | 3 |
| Source/config/input/evaluation/validation/acquisition failure, including partial | Empty, also in JSON mode | Fixed actionable diagnostic; Terraform acquisition errors follow existing pass-through behavior | 1 |
| Signal interruption | Empty | Existing interrupt diagnostic | 130 / 143 |

Exit 3 distinguishes a completed comparison from an operational failure and preserves plan-only exit 2.
Threshold met is a completed assessment, not the headless operational-failure case. JSON reports on failure
are deliberately deferred to preserve empty stdout on failed runs. The TUI can retain partial rows even though
the command emits no partial report. Text output is deterministic plain text: summary, policy identity, eligible
and excluded counts, then address-sorted classified rows with action order, level and reasons. It uses no claim
that infrastructure is safe. Empty text example: `No eligible mutations (2 changes not assessed).`

Exact successful JSON fixture (the source identity values are illustrative opaque identifiers):

```json
{
  "schema_version": "tfui.risk.report/v1",
  "input_schema_version": "tfui.risk.input/v1",
  "policy": {"kind": "custom", "id": "local-policy", "digest": "sha256:example"},
  "context": {"project": ".", "chdir": ".", "workspace": "default"},
  "plan_id": "reviewed-plan-1",
  "status": "complete",
  "overall_level": "low",
  "counts": {"eligible": 1, "classified": 1, "not_assessed": 0},
  "levels": {"none": 0, "low": 1, "medium": 0, "high": 0, "critical": 0},
  "threshold": {"level": "high", "result": "below"},
  "resources": [{
    "address": "aws_lambda_function.worker", "actions": ["update"], "status": "classified",
    "level": "low", "reasons": [{"code": "project-default", "message": "Explicit project default for this mutation.", "paths": []}]
  }]
}
```

`policy.kind` is `builtin` or `custom`; `id` is a display identity, not source contents. Its digest identifies the
entire compiled source snapshot, including explicitly loaded helpers. D-01b specifies digest construction and
path identity; D-01c replaces these illustrative identity strings with its exact format. `plan_id` identifies the
reviewed evidence, never an executable Apply authorization. No raw values, source text or absolute policy path
is included. Reasons are the validated authored text described above. Context identifiers describe acquisition;
they do not enter the independent policy input. `context.project` is the project root relative to invocation
CWD, or null in zero-config mode. `context.chdir` is relative to that root in project mode, otherwise relative
to invocation CWD; `workspace` is the actual selected workspace. These display paths do not replace internal
Context identity or participate in policy evaluation. JSON object key order is irrelevant; resource rows sort by address.
Excluded rows have address/actions/status `not_assessed` and an `exclusion` code, no level or reasons. Codes are
`read`, `no_op`, `pure_import`, `pure_move`, `output_only`; output-only rows use the output address. In combined
exclusions use `pure_move`, then `pure_import`, then the action code. Exclusion codes describe scope, not severity.
An empty report has `status: "empty"`, `overall_level: null`, zero eligible/classified/level counts, excluded rows
retained, and threshold result `not_applicable`. Otherwise threshold results are `disabled`, `below`, `met`.
A disabled threshold has `level: null`; an empty report preserves the configured level (or null when off).

Input and report versions are independent. A version's field meanings, required fields, levels and exit meanings
are fixed; breaking changes require a new major schema string. Consumers may ignore future additional report
fields, but cannot reinterpret an unknown major version. Policy results reject extra fields to catch author
mistakes. The input version is selected by tfui, not policy-controlled; unsupported policy input majors fail
before evaluation. No automatic rewrite of custom sources or fallback to a different major. A built-in policy
upgrade changes its digest even if the interface stays v1; a copied custom template remains project-owned.
`tfui plan -json` remains Terraform JSON with no risk augmentation. Apply and `-plan` input meanings stay unchanged.

## Blast Radius compatibility decision

For complete inputs, use the selected policy's levels in the existing impact formula: critical maximum gives
critical impact; high maximum or at least three changes including a destructive action gives high; three changes
or medium maximum gives moderate; otherwise minimal. Keep the current action/count semantics. Custom `none`
may therefore still yield moderate/high impact by counts and destructive actions. This is impact, not risk
aggregation; do not change the formula to force matching badges.

A group may use that formula only when every eligible mutation in it has a current classification. Non-assessed
changes still contribute to existing counts/action checks but never supply synthetic risk. For groups without
eligible mutations, retain action/count-only impact and label `Risk: not assessed`; this is distinct from
unavailable input. If eligibility itself is uncertain, or any required classification is missing, mark that
group's impact **unavailable**, retain counts/actions and available per-resource reasons, and suppress its numeric
impact score. Never call the old formula with zero values standing in for missing risk. Other complete groups
can retain their scores, but the overall banner reads `Blast radius unavailable: incomplete risk assessment`
rather than a maximum over completed groups. List unavailable groups first, then existing score order for complete
groups, address order to break ties. A global source/compile/evidence failure makes all affected groups unavailable.
This requires an explicit SDK-facing validity representation in US-004; no evaluator types cross the boundary.

Regression examples: one custom-low database deletion yields minimal impact under existing arithmetic (no hidden
built-in critical fallback); three custom-none mutations including a delete still yield high impact; one failed
classification beside two low ones yields unavailable group impact, never minimal/moderate. Blast Radius changes
are limited to this validity integration and provenance; no new impact algorithm or Apply behavior is authorized.

## Evidence handoff and falsification criteria

The [D-03b prerequisite audit](boundary-controls.md) records the missing D-03a protocol at
the inspected checkout; it contains no executed runtime controls.

D-02 records fixtures, toolchain, baseline and criteria **before running** acquisition/lifecycle experiments.
For every node kind above, compare source evidence and input structurally, including numeric token text, array
order, missing/null/unknown, whole-object masks and both replacement orders. Sensitive content must be absent
from input and capture files. Neighbor insertion/removal must leave the input/result identical. Count live/seeded
plan calls through real Session startup and initial selected Chdir; no second plan for policy evidence. Include
phantom array/null counterexamples as advisory-data limitations. Any silent loss beyond defined redaction rejects
this input; any stale or mixed Context/source snapshot rejects the lifecycle candidate. D-01b supplies path,
reload and source-selection cases, including real current Member-block rejection.

D-03 runs the exact entrypoint/default/result and capability profile above with its selected version and stores
reproducible modules, fixture inputs, harness invocation, raw channel captures with synthetic secrets only, and
measured outcomes. Test undefined, empty/malformed/cardinality/conflict, strict executed/unreachable builtin
errors, numeric operations, and every denied effect in both executed and unreachable code. A synthetic marked
secret must be absent before evaluation and from stdout/stderr/errors; separately demonstrate that an unmarked
or source-literal secret can be returned, documenting the trust limit rather than hiding it. An allowed external
or nondeterministic effect, printed/traced content, or an invalid result accepted as success falsifies the candidate.
Record supported-target builds, baseline/candidate size, cold compilation/evaluation, warm evaluation and peak
memory under an external harness limit. No numerical acceptance budget is invented here; D-01c reviews measured
costs and either records acceptance under delegated authority or revises the candidate before integration.

These are evidence requests, not executed experiments. Runtime, acquisition, configuration and UI behavior tests
belong to the downstream stories once the interface is published. This prose-only preparation adds no product
tests and cannot establish that policy evaluation, reporting or lifecycle behavior already works.

The [D-02a fidelity evidence](input-fidelity.md) now records executed disposable resource-local projection,
neighbor-independence and phantom-counterexample experiments. It does not establish live/seeded acquisition,
Context/source lifecycle or evaluator behavior; those evidence requests remain separate.

The [D-02b acquisition evidence](acquisition-counts.md) records live/seeded call counting and real Session
startup behavior, including rejection of initial-Chdir seed preservation and of `PlanJSON` as a retrieval path
for an already reviewed plan. These are acquisition experiments, not production policy integration.

The [D-02c Context/source evidence](context-source-identity.md) records current HCL/startup/switch behavior,
disposable snapshot/publication experiments and the combined D-02 first-version conclusion.
