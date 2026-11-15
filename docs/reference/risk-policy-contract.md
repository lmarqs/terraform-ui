---
layout: default
title: Risk policy contract
parent: Reference
description: "Specifies first-version risk policy input, results, source lifecycle, reporting and consumer compatibility."
audience: "Policy authors and contributors implementing the first-version risk interface"
---

# Risk policy contract

This is the **implemented first-version public contract**, published under delegated contributor decision authority
on 2026-11-15. The coordinator selected the exact interfaces from D-01a/b and reviewed D-02/D-03 evidence;
the owner did not select these details. Stress and refine this version after merge through explicit revisions.

Plan and Risk share selected-policy classifications; `tfui risk` provides the report and comparison described
below. The [session verification](../development/risk-policy-evidence/session-integration.md) and
[preservation gates](../development/risk-policy-evidence/preservation-gates.md) record the tested production paths
and their limits. The [Risk guide](../plugins/risk.md) explains use. [ADR-0026](../adr/0026-publish-first-version-risk-contract.md)
records the original publication decision; the [design](../development/risk-policy-design.md) records alternatives
and residual risks. Historical disposable evidence establishes feasibility within its stated bounds, not broader
production guarantees.

## Purpose and philosophy

Risk classifies each planned resource mutation independently and explains its classification. It helps a reviewer
understand the change; it does not decide whether someone should apply it, certify infrastructure as safe, or
perform general compliance analysis.

**Make consequential behavior explicit.** A classification, default, policy selection, exclusion or failure must
have an identifiable meaning. Missing output is not a default. An unavailable assessment is not `none`. A custom
policy does not silently inherit built-in classifications. This principle favors predictable behavior without
requiring configuration for every implementation detail.

## Terms

Project, Member, Chdir and Workspace retain the meanings in the repository's
[domain glossary](https://github.com/lmarqs/terraform-ui/blob/main/CONTEXT.md). The risk terms below are owned by
this page.

| Term | Meaning |
| --- | --- |
| Resource mutation | A planned create, update, delete or replacement of one resource. |
| Policy | The complete selected classification logic, responsible for an explicit result for every submitted resource mutation. |
| Policy source | The complete local file or directory snapshot selected by `risk_policy`, as specified below. |
| Classification | A risk level and its explanation for one resource mutation. |
| Assessment | The result of classifying the eligible mutations in a supplied plan, or an explicit failure to do so. It makes no claim about infrastructure outside that plan. |
| Not assessed | A change outside this version's assessment eligibility; not a severity and not an implicit classification of `none`. |
| Unavailable | An assessment could not be completed because of an evaluation or validation failure; not a severity. |
| Unknown value | A value not known in the plan; distinct from a known null, an absent field, and an assessment failure. Represented by an opaque tagged node, as specified below. |
| Threshold | A comparison used by tfui to interpret classifications for the risk command. The policy does not choose the caller's response. |
| Template | Explicit policy source copied and owned by a project, rather than a named runtime preset. |

The existing ordered severity vocabulary is `none`, `low`, `medium`, `high`, `critical`. In this contract,
`none` is an explicit policy classification. It must not stand in for missing results, excluded changes, failures,
or a general guarantee of safety. The v1 result and compatibility rules are specified below.

## Responsibilities

| Owner | Responsibility | Boundary |
| --- | --- | --- |
| Policy | Classify and explain each submitted resource mutation using its own change data. | No cross-resource references, plan-wide findings, threshold enforcement or Apply decisions. |
| tfui | Select the policy, preserve the reviewed plan's evidence, determine eligibility, validate classifications, aggregate them, render the report and perform threshold comparison. | No hidden fallback for incomplete custom-policy output. Use the complete-assessment maximum and explicit threshold below. |
| Caller, human or automation | Decide what to do with the report and command result. | This feature does not block or modify `tfui apply`. |

Independent classification is a semantic boundary: a resource's classification cannot depend on other resources
in the plan. Whether the implementation batches calls internally is not a promise to policy authors. Unknown and
sensitive metadata must not be reconstructed from rendered diff strings; the permitted input is the resource-local envelope below.

## Which changes are assessed

| Planned change | Treatment |
| --- | --- |
| Create, update or delete | Submit for explicit classification. |
| Replacement, in either ordering | Submit for explicit classification; preserve the replacement action. |
| Mutation labeled phantom by tfui | Submit normally. The label does not bypass assessment or force `none`. |
| Read or no-op | Not assessed. |
| Pure import or pure address move | Not assessed. |
| Output-only change | Not assessed. |
| Import or move accompanied by a mutation | Assess the create/update/delete/replacement mutation. |

These exclusions describe scope, not proof that excluded operations are harmless. A targeted plan is assessed
only for the eligible mutations it contains. An empty eligible set must be distinguishable from resources
explicitly classified as `none`; the `empty` report state below represents that case.

## Explicit results and defaults

Every submitted mutation requires an explicit classification. A policy may contain an explicit default case for
mutations without a specialized rule. Missing or undefined output is an assessment error. Invalid output also
fails validation; it cannot be interpreted as successful evaluation merely because the evaluator returned no Go
error.

With no custom policy selected, tfui uses its built-in classifier. Selecting a custom policy replaces that
classifier completely. For example, a custom policy specializing in Lambda updates must also classify a database
deletion in the same plan, perhaps through its own default. tfui does not fill the gap from built-in rules.

This ownership rule is separate from configuration inheritance. A Member's selected policy replaces the complete
inherited policy; tfui does not combine ancestor rules. HCL selects local policy source rather than defining a
second rule language. Explicit templates replace named runtime presets. Source loading, explicit helper reuse, reload and configuration keys are specified below.

## Failure behavior

When risk evaluation fails, Plan remains reviewable with risk clearly marked **unavailable** and the error
visible. `tfui risk` reports an error result, never a passing assessment. Apply behavior remains unchanged.

The partial states, fixed diagnostics and command outcomes below must not disguise failure as `none`,
reuse a previous assessment as if it were current, or introduce an implicit fallback policy.

## Review presentation

The first-version Risk view shows the mutation itself alongside its classification: action symbol, resource address,
and compact attribute diff. It does not repeat a `Change` label on each row. Inspection exposes the full diff and
classification explanation. Plan and Risk share action/risk styles and attribute-diff rendering; plugin-local
copies of symbol or color tables would contradict that design.

The visual conventions belong to [TUI UX](tui-ux.md). Unknown and sensitive notation, viewport navigation and
classification failures must remain understandable in narrow layouts. Risk does not add an Apply confirmation
flow. Older browser mockups are visual experiments, not authority for Apply behavior or policy semantics.

Group headings are not navigable resources. Show the standard scroll gutter and `[cursor/navigable]` title
counter. Show at most two changed attribute paths under each resource, with `+N more` for overflow; inspection
shows every diff and validated reason. Collapse long values to fit, preserving action/replacement order,
`(known after apply)` and `(sensitive)` notation. Group unavailable resources first, then critical through none;
within groups sort by address. Explicit `none`, `No eligible mutations`, and `Risk unavailable` have distinct
headings. Narrow layouts may wrap or truncate display values, never replace an unavailable badge with a level.

## Input v1

This required envelope is independent of Go types. No neighboring resource, plan-wide
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

Envelope/resource/change keys shown above are required and no other keys are exposed. Address, type, name and
provider_name are nonempty strings; mode is `managed` or `data`. Actions must be exactly `["create"]`,
`["update"]`, `["delete"]`, `["delete","create"]` or `["create","delete"]` after eligibility.
Missing provider/resource/action metadata needed to interpret a mutation is incomplete
input, not an empty plan. `previous_address` is a string or null; `index` is an absent, number or string node.
`importing` is null or an object with `id` as a value node and `unknown` as a boolean, preserving a resource-local
import identifier's sensitivity/uncertainty. If acquisition cannot establish its masking, omit no evidence silently:
fail that input. `replace_paths` is an ordered array of paths, each an array of string keys or nonnegative integer indexes
(never booleans). An empty replacement path denotes the whole object; paths preserve Terraform evidence
even when a target subtree is opaque. A resource index is a nonnegative integer number node or a string node;
no fractional/negative numeric index is accepted.
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
The fidelity experiment establishes exact ingestion for its fixtures, not evaluator arithmetic precision.
V1 promises exact token strings, not arbitrary-precision numeric operations. Authors converting with `to_number`
own conversion semantics and must test their numeric ranges; production comparison tests cover large-number cases.

Build object entries from the union of value keys and unknown/sensitivity mask keys: a key present only in an
unknown mask becomes an unknown node, not an absent entry. A mask-only sensitive key whose existence is
established becomes sensitive (or sensitive_unknown); inconsistent masks fail validation. Derive an absent
whole before/after side only from a create/delete action plus its corresponding source null side; a present
attribute with null remains a null node. Production acquisition must validate these distinctions against source JSON before projection, including
descendant masks beneath an opaque ancestor; the disposable adapter did not establish that validation.

Apply before/after sensitivity masks separately. Validate every mask leaf as boolean and every container against its source/unknown shape before redaction;
array masks must match established list length and false/missing mask entries mean unmarked.
A true sensitivity mask on an ancestor replaces the entire subtree with `sensitive`.
Nested object/array masks apply recursively; an ancestor unknown collapses its subtree to unknown, or
`sensitive_unknown` if any true sensitivity mask exists at or below that unknown ancestor. A true sensitivity
mask on a known ancestor hides descendant uncertainty as well as values; only an unknown ancestor whose
subtree is also marked sensitive exposes both facts. This follows the tested projector precedence. An inconsistent mask/shape or unsupported source
format fails evidence validation rather than guessing a value. Unknown/sensitive nodes deliberately disclose
no children; their omission is the stated redaction boundary, not accidental input loss. No phantom normalization
(sorted arrays or deleted null keys) is permitted in this input. `IsPhantom` is not policy input or an exclusion.

## Result v1 and rule resolution

Use Rego v1 package `tfui.risk.v1`, complete document `classification`, queried exactly as
`data.tfui.risk.v1.classification`. Require one result with one expression value; no variable bindings
or finding sets are part of this query. The result has exactly `level` and `reasons`. `level` is one of `none`, `low`, `medium`, `high`, `critical`.
`reasons` is an array of 1–16 objects, each with exactly `code`, `message`, `paths`:

```json
{"level":"low","reasons":[{"code":"project-default","message":"Explicit project default for this mutation.","paths":[]}]}
```

`code` matches `[a-z][a-z0-9-]{0,63}` over the entire string. `message` is 1–512 Unicode code points of nonblank text, with no terminal
control characters (Unicode general category Cc, including newline, tab and escape). Each `paths` array contains at most 16 paths into before/after values, such as
`["after","memory_size"]`; elements after the initial `before`/`after` are string keys or nonnegative integer
indexes; booleans are not indexes. Empty individual paths are invalid; use `paths: []` for a general reason. Paths may name an unknown/sensitive node but cannot traverse beneath it or refer to a nonexistent node.
Messages are trusted author explanations, not verified causal claims. Render them as plain text, never ANSI/markup.
These bounds keep review output usable; they are not resource limits for the evaluator.

A complete minimal custom module is:

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

| Evaluator observation | tfui treatment |
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

## Runtime and trust boundary

Use embedded OPA **v1.19.1**, revision `54896f9f28515d14b31a93e6ef5737bb85866e30`, as the first integration
pin with the tested Go1.25.5 toolchain. This selects the measured version, not a claim of current upstream support
or security approval. Production integration must review advisories and regression-test the resolved dependency
upgrades; any necessary runtime change must rerun controls/builds and record compatibility rather than silently
changing the profile. No host execution service is exposed to policy modules.

Select **trusted local policy authors with redacted sensitive plan values**. A policy can intentionally encode
constants or infer facts from nonsensitive input. tfui cannot certify explanations contain no secrets or that
malicious code terminates. Pre-evaluation masking limits direct access to marked values; unmarked secrets,
resource identity, authored constants and reasons remain disclosure risks. Embedded OPA is not a hostile-code
sandbox. Production uses cooperative cancellation and generation rejection, without a hard memory/deadline
promise; the experiments' external process guards are not product controls.

The positive builtin allowlist is exactly: `eq`, `equal`, `neq`, `lt`, `lte`, `gt`, `gte`, `count`, `is_array`,
`is_object`, `is_string`, `is_number`, `is_boolean`, `is_null`, `object.get`, `object.keys`, `array.concat`,
`concat`, `sprintf`, `startswith`, `endswith`, `contains`, `lower`, `upper`, `to_number`.
Use the declarations, `future_keywords` and `features` from the
[frozen capability artifact](../development/risk-policy-evidence/assets/boundary/output/capabilities.json),
removing the experimental `plus`, `minus`, `mul`, `div` controls. Those arithmetic operators are not v1 author
capabilities. Compute the capability fingerprint as `sha256:` plus SHA-256 of UTF-8 JSON with recursively sorted
object keys, no insignificant whitespace, literal Unicode and array order retained, over that filtered artifact.
Network allowances are an explicitly empty slice in the evaluator; absence in serialized JSON is not permission.

No custom builtins, external resolvers, remote schemas, arbitrary bundle data or neighboring-resource data.
Modules/helpers are the only supplied policy data. HTTP, DNS, time, randomness, UUID and runtime/environment
access are denied by omission. Use strict executed builtin errors, no query tracer or print hook, and print
disabled. Reject `print` and `trace` calls by parsed AST inspection in **every** selected module, including
unqueried code; print disabling alone can erase the call. A strict error in an expression that never executes
is not detectable as an executed failure. Never expose raw OPA errors, module excerpts or stack traces.

## Source selection

Choose a dedicated optional string attribute `risk_policy` in root `defaults`, Member top level, and matching
Member workspace blocks. It selects shared assessment logic, independently of whether the Risk view is enabled.
Keep generic `plugin` blocks root-only. Decline adding Member plugin overrides: it expands unrelated settings and
invites per-key policy mixing. Decline placing source only in root plugin options: it cannot express complete
Member/workspace replacement through the currently supported HCL.

Example v1 configuration files:

```hcl
# /project/tfui.hcl
member "services/api" {}
defaults {
  risk_policy = "policies/team"
}
```

```hcl
# /project/services/api/tfui.hcl
risk_policy = "policies/api.rego"
workspace "prod" {
  risk_policy = "policies/prod"
}
workspace "dev-*" {
  risk_policy = ""
}
```

| Selection case | V1 behavior |
| --- | --- |
| No configuration or no selector anywhere | Embedded complete built-in policy. Never auto-discover policy files. |
| Selector omitted at a later level | Inherit the complete earlier selection, including its declaring directory. |
| Selector exactly `""` | Explicit reset to built-in; suppress inherited custom selection. |
| Nonempty string | Select exactly that local file or directory, replacing all inherited modules. Do not trim it. |
| Whitespace-only string | Invalid selector; no built-in fallback. |
| Null, nonstring, duplicate attribute or expression without a known literal string value | Configuration failure, never treated as omission. |
| Selected source absent, unreadable, deleted, empty or invalid | Assessment unavailable; preserve Plan and show a fixed diagnostic. No inherited/built-in fallback. |

Apply root defaults → Member → one matching workspace, with existing exact-name-before-first-glob matching.
Relative paths originate in the directory containing the declaration that won, not invocation CWD or active
Chdir. Thus the inherited root path is `/project/policies/team`; Member default is
`/project/services/api/policies/api.rego`; prod is `/project/services/api/policies/prod`; dev selects built-in.
Absolute local paths are allowed; no URI, environment expansion, glob or remote fetch. Spaces inside a path are
literal. When Member is the project root, parse the root file only as root configuration; do not parse it again
as a child. In zero-config startup with `-chdir`, use built-in unless an actually loaded configuration selects
custom source; do not search parents or infer a source from files in Chdir.

The new selector needs strict HCL parsing with declaration provenance. Existing fallback on invalid child
configuration must not hide a malformed selected policy or resurrect the root policy. Isolate policy diagnostics
from usable execution settings where possible so Plan remains reviewable. If the effective selection cannot be
established, mark assessment unavailable. This task does not change global handling of unrelated invalid HCL.

## Loading and snapshot identity

Accept a regular `.rego` file as one module, or a directory's immediate regular `.rego` files sorted by filename.
Exclude `*_test.rego`; ignore other regular files and subdirectories. No recursion, bundles, JSON/YAML data or
implicit import lookup. Reject a selected test file, a directory with no production modules, and symlinks at the
selected path, any ancestor path component, or an eligible module entry. This deliberately modest local-loader
boundary is not a hostile-filesystem sandbox. Resolve lexical absolute paths without following symlinks.

A file's logical module name is its basename; directory modules use immediate filenames. Read exact bytes into
an immutable snapshot before compiling. Imports resolve only against that snapshot. Missing helper imports,
duplicate module identities, conflicting definitions or undefined entrypoints cannot trigger loading elsewhere. Compilation/result
validation, rather than filenames, enforces the complete entrypoint contract above.

Choose source digest `sha256:` plus lowercase hexadecimal SHA-256 over this byte sequence:

1. ASCII `tfui-risk-source-v1` followed by one zero byte.
2. Module count as unsigned 64-bit big-endian.
3. For each module in UTF-8 filename byte order: name byte length (unsigned 64-bit big-endian), UTF-8 name bytes,
   source byte length (same integer encoding), exact source bytes.

Reject non-UTF-8 filenames. Do not normalize line endings or whitespace. Embedded modules use the same framing.
Renaming or editing a module changes the digest; moving the same named modules to another directory does not.
For example `main.rego` containing ASCII `package x\n` contributes name length 9 and source length 10.
Tests, unrelated files, mtimes and absolute paths do not enter the digest. This is source identity, not a signature.
The compilation-cache key additionally includes evaluator version, input/result major and the capability-set
fingerprint; a runtime upgrade must not reuse a prepared query just because source bytes match.

Use report `policy.id` = `builtin/v1` for embedded v1, or `custom/` followed by the full source digest for custom.
Always include `policy.kind` and `policy.digest` separately in the report below. No absolute source path in reports.
Same source bytes can have the same digest across Contexts; internal publication identity must still include
canonical absolute Project/Chdir, actual Workspace, reviewed plan evidence identity, selection origin/path,
source digest and an assessment generation. None of these Context values becomes policy input.

## Reload and publication

Choose reload at assessment boundaries, without a watcher or a new keybinding. At initial assessment, Context
change, new plan completion, explicit existing refresh, and Risk activation, reread effective configuration and
selected modules. Compare snapshots before reusing a cached assessment. Risk activation after Plan with unchanged
evidence/source reuses the assessment and performs no second Terraform plan. Pin-only Context changes invalidate
publication according to existing Context events; acquisition retains existing target/cache rules.

| Event | Required result |
| --- | --- |
| Same path, changed module bytes or eligible file added/removed | New snapshot and assessment; invalidate previous derived risk immediately. |
| Same source, only mtime/test/unrelated file changed | Same digest; reuse is permitted for identical plan and Context. |
| Source deleted or malformed on next boundary | Unavailable risk; never continue displaying last successful risk as current. |
| A evaluates while switching to B | Cancel A; generation check rejects late A results even if cancellation is ignored. |
| Source changes during evaluation | Evaluate immutable bytes; reread selection/source before publication. If changed, discard result and mark it superseded. |
| Source changes while view is idle | No immediate detection guarantee; next boundary detects it. |

After a detected mid-evaluation edit, leave assessment unavailable with a source-changed diagnostic until the
next assessment boundary; do not spin indefinitely retrying. Directory acquisition records names and content,
then verifies the listing/content again before accepting the snapshot. Any observed inconsistency rejects that
attempt. Recheck before publication as above. This detects ordinary concurrent edits, not adversarial filesystem
swaps between reads. No mixture of observed generations may be published. A same-byte restoration is the same
source, while the Context generation still prevents stale A publication. Cancellation must terminate headless
operation with the existing lifecycle contract rather than leave `Ready()` pending.

## Helpers, templates and upgrades

Choose explicit vendoring: a project copies `main.rego`, optional helper modules and tests into its selected
directory. A file source is self-contained; a directory source may import another selected module. No hidden
`data.tfui.builtin`, installed helper library, parent modules, import-driven fetch or fallback classifier exists.
Decline a separately versioned automatic helper package for v1: its upgrades would silently change project logic.
Pure helpers may return constructed reasons or inspect the one input; selecting them does not authorize plan-wide data.

Exact illustrative complete template and helper, both project-owned (the level is a project default example,
not a changed built-in classification):

```rego
# main.rego
package tfui.risk.v1
import rego.v1
import data.project.reason

classification := {"level": "low", "reasons": [reason.default_reason]}
```

```rego
# reason.rego
package project.reason
import rego.v1

default_reason := {
    "code": "project-default",
    "message": "Explicit project default for this mutation.",
    "paths": [],
}
```

This unconditional rule explicitly classifies every submitted mutation. A Lambda-specialized template must use
an explicit default plus mutually exclusive specialized rules, and include a database deletion fixture proving
the project's own default, never an invisible built-in database result. Templates are complete source copies,
not runtime preset names. The [authoring guide](../guides/risk-policy-authoring.md) supplies a complete
attribute-aware Lambda example, isolated full-input fixtures and executable author tests using this contract.

Choose ordinary Rego author tests alongside isolated resource input fixtures, using OPA v1.19.1 and input v1. Minimal author test for the template above:

```rego
# main_test.rego
package tfui.risk.v1_test
import rego.v1
import data.tfui.risk.v1 as risk

test_explicit_default if {
    result := risk.classification with input as {"change": {"actions": ["delete"]}}
    result == {"level": "low", "reasons": [{
        "code": "project-default",
        "message": "Explicit project default for this mutation.",
        "paths": [],
    }]}
}
```

Author command: `opa test main.rego reason.rego main_test.rego --fail-on-empty -v`. This minimal input
tests policy logic only; it deliberately is not a valid tfui envelope. Production author-test fixtures use full v1
envelopes and verify tfui validation/capabilities too. `opa test` alone proves neither eligibility nor the embedded
capability boundary nor valid tfui results. OPA documents `with input`, test discovery and `--fail-on-empty` in its
[testing guide](https://www.openpolicyagent.org/docs/policy-testing). The complete
[Lambda example](../guides/risk-policy-authoring.md) is also exercised through tfui's production loader,
capabilities and result validator. The standalone CLI does not enforce tfui result validation.

Built-in policy ships as embedded complete source, with the current fixed action/type severities preserved for
v1; the eligibility contract supersedes the old classifier's unhandled-action `none`. Add evidence-based reasons,
not invented severity changes. New built-in bytes change the digest and invalidate assessments. Publish rule
changes in release notes and the canonical feature page with regression fixtures; schema compatibility alone
does not imply identical classifications. Authors needing unchanged decisions copy and pin source in their project.
tfui upgrades never rewrite copied templates/helpers or silently append new built-in rules. Test copies against
the selected runtime before upgrading; incompatible input/result majors fail explicitly under the compatibility rules below, without fallback.

## Lambda first-version choices

Retain the fixed classifier's levels, including for templates advertised as built-in-equivalent. Attribute-aware
reasons explain evidence without implying external deployment ownership. Decline automatic lowering of memory-only
or other non-code updates: the field report establishes a code-update concern, not a reviewed new severity scale.
Decline raising every layer mutation without independent justification. These are first-version contributor decisions, not owner-selected severities or a provider-version reproduction.

| Mutation/evidence | v1 level | Reason basis |
| --- | --- | --- |
| Lambda function create | medium | Function creation; source availability does not change action classification. |
| Function delete or either replacement order | critical | Destructive function action, retaining the original action order. |
| Update with known changed `filename`, `s3_bucket`, `s3_key`, `s3_object_version`, `image_uri` or `source_code_hash` | high | Name changed code-source attributes and their paths; no claim about bytes or deployment ownership. |
| `filename` changes with hash unchanged, null, absent or unknown | high | Filename evidence remains; hash state does not negate it or establish placeholder contents. |
| Only memory 128 → 256, all code-source fields known equal | high | Memory changed; retained built-in function-update classification. No claim of a code upload. |
| Hash alone becomes unknown or a source field is sensitive/unknown | high | Code-source evidence is unavailable/uncertain; do not claim known changed content. |
| Memory changes while any source field is unknown | high | Explain observed memory change and uncertainty; do not call it memory-only. |
| Other function update, including `layers` | high | Name available changed paths; retained function-update classification. |
| Layer-version create / update / delete or replacement | low / low / medium | Unlisted-type action defaults retained; no new layer-specific severity. |

Order reason selection explicitly: destructive/create action first, then known code-source change, then uncertain
source evidence, then observed configuration update, then type/action default. Return known changed code-source paths in the field order listed above, with before then after paths for each
field when those nodes exist, in one reason; do not let competing complete-rule objects resolve by source order.
Determine known equality only for known nodes; absent/null/unknown/sensitive are distinct. A hash unchanged at a
known value is evidence only about that field. A null-to-null hash proves no code equality. For a field absent
on both sides, report no evidence for that field, not known source equality. Eligible phantom mutations retain
these same rules. No provider schemas, package filesystem reads or external pipeline facts enter evaluation.

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

Evaluate the remaining finite resource list after a resource failure to aid TUI review; cancellation stops it.
Validate input/eligibility and load/compile the selected source even for empty eligibility: invalid selection
is unavailable, never an empty success. Only complete validated input and a valid source permit `empty`. Pure read/no-op changes need only enough source
metadata to establish exclusion, not a fabricated mutation envelope. A present output change is an excluded
row with actions from its output change; use address `output.<name>` to distinguish it from resource addresses.
No aggregate, threshold pass or success badge may be derived from a subset. Global input validation failure
invalidates the whole assessment; do not salvage rows from uncertain eligibility. Excluded rows are retained in reports and may be shown in review; they do not contribute to severity counts or the maximum. Unknown values are valid evidence, not incomplete input.

## Command, threshold and report v1

Invocation: `tfui risk -ci -plan plan.json -json -threshold high`. `-json` controls format independently
of mode. The Risk setting is `threshold`, with values `off`, `none`, `low`, `medium`, `high`, `critical`;
explicit CLI `-threshold` wins over the resolved Risk setting, then default `off`. No policy-returned threshold or threshold inside the Rego input.

Compare inclusively: with `high`, `medium` passes, `high` and `critical` meet the threshold. With `none`, every
nonempty complete assessment meets the threshold, including explicit `none`. Empty eligibility never matches.
`off` disables comparison; it does not disable assessment or hide errors. Invalid thresholds fail configuration.
The old ignored `level` key is not an alias: report an actionable configuration error (`risk setting level is
unsupported; use threshold`) and unavailable risk while preserving Plan review. The [CLI I/O contract](cli-io-contract.md#risk-command-v1) owns the command channel/exit extension.

Exit 3 distinguishes a completed comparison from an operational failure and preserves plan-only exit 2.
Threshold met is a completed assessment, not the headless operational-failure case. JSON reports on failure
are deliberately deferred to preserve empty stdout on failed runs. The TUI can retain partial rows even though
the command emits no partial report. Text output is deterministic plain text: summary, policy identity, eligible
and excluded counts, then address-sorted classified rows with action order, level and reasons. It uses no claim
that infrastructure is safe. Empty text example: `No eligible mutations (2 changes not assessed).`

Exact successful JSON fixture, with threshold `high`, root project, workspace `default`, and the minimal
`main.rego` above. Its synthetic show JSON preimage is this single UTF-8 line **without a trailing newline**:

```json
{"format_version":"1.2","resource_changes":[{"address":"aws_lambda_function.worker","mode":"managed","type":"aws_lambda_function","name":"worker","provider_name":"registry.terraform.io/hashicorp/aws","change":{"actions":["update"],"before":{"memory_size":128},"after":{"memory_size":256},"after_unknown":{},"before_sensitive":{},"after_sensitive":{}}}]}
```

Report:

```json
{
  "schema_version": "tfui.risk.report/v1",
  "input_schema_version": "tfui.risk.input/v1",
  "policy": {"kind": "custom", "id": "custom/sha256:47f4de18e2ee23e5fb200f7a111012c543db320571ba33c984c4c0d4f9cf0f25", "digest": "sha256:47f4de18e2ee23e5fb200f7a111012c543db320571ba33c984c4c0d4f9cf0f25"},
  "context": {"project": ".", "chdir": ".", "workspace": "default"},
  "plan_id": "sha256:d24e2217367ec17a80f3ace738aee3ca16ae8035185b3fddcf3799b65b2c71e6",
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
entire selected source snapshot, including explicitly loaded helpers, using the framing above. The fixture
uses `main.rego` with exactly the minimal module above (one trailing LF); custom identity is `custom/` plus
that digest. `plan_id` is `sha256:` plus lowercase hexadecimal SHA-256 over the original UTF-8 Terraform show
JSON bytes, without whitespace normalization, before redaction or display parsing. Live acquisition retains
the same show bytes; seeded input retains the supplied bytes. It identifies evidence, never Apply authorization. Report serialization includes no raw value trees, source text or absolute policy path. Reasons are validated
authored text and can still contain unmarked or authored secrets under the stated trust boundary. Context
identifiers describe acquisition;
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

All shown report fields are required. `status` is `complete` or `empty`; `overall_level` is a level or null.
Counts and level counts are nonnegative integers. `resources` contains every eligible/excluded change; rows
have exactly the status-specific fields above. Reasons follow result v1. `plan_id`, policy digest and Context
describe the same acquisition/snapshot used for all rows. A report digest is provenance, not a signature; hashes
of sensitive evidence can still disclose equality across reports.

Input and report versions are independent. A version's field meanings, required fields, levels and exit meanings
are fixed; breaking changes require a new major schema string. Consumers may ignore future additional report
fields, but cannot reinterpret an unknown major version. Policy results reject extra fields to catch author
mistakes. The input version is selected by tfui, not policy-controlled; only the v1 entrypoint is queried. A policy offering only another major fails
with `result_undefined`, never an implicit query of that other major. No automatic rewrite of custom sources or fallback to a different major. A built-in policy
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
SDK-facing assessment values carry validity explicitly; no evaluator types cross the boundary.

Regression examples: one custom-low database deletion yields minimal impact under existing arithmetic (no hidden
built-in critical fallback); three custom-none mutations including a delete still yield high impact; one failed
classification beside two low ones yields unavailable group impact, never minimal/moderate. Blast Radius changes
are limited to this validity integration and provenance; no new impact algorithm or Apply behavior is authorized.

Threshold configuration remains root-only generic Risk settings, independent of the dedicated source selector:

```hcl
defaults {
  risk_policy = "policies/team"
  plugin "risk" { threshold = "high" }
}
```

The selector is usable even if the Risk view is disabled. A CLI threshold override affects command comparison,
not classification or policy source. Register `--threshold` as a persistent tfui flag (single-dash normalization
also accepted), usable with all commands; it has comparison effects only for Risk and never changes Plan/Apply.
`-json` and `-ci` retain their independent format/mode meanings; `-plan` accepts show JSON, including stdin `-`,
never a binary saved plan. Without seeded evidence Risk uses active Context plan options and a temporary saved
artifact, explicitly cleaned up, and never claims Plan's executable artifact ownership.

### Text format and diagnostics

Successful text uses LF-terminated lines, no ANSI, and this field order. Within each classified/excluded text section, rows sort by address;
reason rows retain policy order. Action labels join the original actions with `/`. Counts are decimal.
For the JSON example above, the exact text is:

```text
Risk: low
Policy: custom/sha256:47f4de18e2ee23e5fb200f7a111012c543db320571ba33c984c4c0d4f9cf0f25
Mutations: 1 eligible, 1 classified; 0 changes not assessed
Threshold: high (below)
update aws_lambda_function.worker: low
  project-default: Explicit project default for this mutation.
```

Reasons with
paths append ` [paths: <compact JSON paths array>]`. Excluded rows follow classified rows in address order:
`<actions> <address>: not assessed (<exclusion>)`. An empty assessment replaces the first line with
`No eligible mutations (N changes not assessed).`, retains policy/count/threshold lines, then excluded rows.
Threshold text is `off (disabled)` when off for nonempty reports and `<level-or-off> (not_applicable)` for empty.
A complete explicit-none report starts `Risk: none`; it does not imply safety. End every report with one LF.

Policy/config/input failures use fixed lowercase diagnostics `risk unavailable: <code>` with an optional
` for <escaped-address>`. Codes: `configuration_invalid`, `source_unreadable`, `source_invalid`,
`source_changed`, `input_incomplete`, `input_invalid`, `compilation_failed`, `result_undefined`,
`result_invalid`, `result_cardinality`, `evaluation_conflict`, `evaluation_failed`.
Use `configuration_invalid` for the unsupported `level` key and append
`\n\nhint: risk setting level is unsupported; use threshold`. For unavailable/partial command results choose
the first failure by address (global failures take priority), with no partial stdout. TUI retains every valid row
and a summary count. Escape terminal controls in source metadata; public diagnostics never interpolate raw values,
source paths or compiler messages. Terraform acquisition errors retain their existing pass-through error behavior.

### Empty report example

For the same policy identity, no eligible resources and one no-op change, the variable part of a report is:

```json
{
  "status": "empty", "overall_level": null,
  "counts": {"eligible": 0, "classified": 0, "not_assessed": 1},
  "levels": {"none": 0, "low": 0, "medium": 0, "high": 0, "critical": 0},
  "threshold": {"level": "high", "result": "not_applicable"},
  "resources": [{"address": "aws_s3_bucket.logs", "actions": ["no-op"],
    "status": "not_assessed", "exclusion": "no_op"}]
}
```

The omitted schema, policy, Context and plan identity fields are still required in the full report. Failure and
partial assessments have no command JSON encoding in v1: stdout is empty, exit 1. Only the TUI retains partial rows.

## Decision records and evidence

- [ADR-0024: embedded Rego](../adr/0024-embed-rego-for-risk-policies.md).
- [ADR-0025: explicit independent classification](../adr/0025-classify-resource-mutations-explicitly.md).
- [ADR-0026: first-version publication](../adr/0026-publish-first-version-risk-contract.md).
- [Publication review and evidence limits](../development/risk-policy-evidence/contract-publication.md).
