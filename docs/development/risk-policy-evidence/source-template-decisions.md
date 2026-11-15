---
layout: default
title: Risk policy source and template decisions
parent: Development
description: "Records first-version source selection, reload, helper, template and Lambda choices for contract publication."
audience: "Contributors proving D-02 source identity and publishing D-01c before implementation"
---

# First-version source, helper and template decisions

D-01b selects these first-version candidates under delegated contributor authority. They are concrete choices
for experiments and D-01c publication, not owner-selected interfaces or implemented settings. The
[contract](../../reference/risk-policy-contract.md) still owns independent mutation classification, complete
custom replacement, eligibility and failure behavior. The [D-01a candidates](first-version-candidates.md) own
input/result/trust/report detail. D-02 must falsify or substantiate this lifecycle before dependent implementation.
No production code, Apply behavior or runtime dependency changes here.

## Reproducible baseline and limits

Inspected checkout: `7db0658724060509a1b0d5c8a1555f8ac3c431a8`. From this worktree's root:

```sh
git show 7db0658724060509a1b0d5c8a1555f8ac3c431a8:internal/config/hcl_child.go
git show 7db0658724060509a1b0d5c8a1555f8ac3c431a8:internal/config/hcl_root.go
git show 7db0658724060509a1b0d5c8a1555f8ac3c431a8:internal/config/hcl_resolve.go
git show 7db0658724060509a1b0d5c8a1555f8ac3c431a8:internal/config/hcl_test.go
git show 7db0658724060509a1b0d5c8a1555f8ac3c431a8:internal/terraform/risk.go
```

`LoadChild` rejects `plugin` blocks at Member top level and inside workspaces. Preserve that rejection and its
`TestLoadChild_WhenPluginBlockPresent_ShouldRejectWithRootDefaultsHint` regression. `mergePlugins` merges
option keys in manually constructed settings; that is not evidence of supported Member HCL. `extractPluginOptions`
can skip invalid expressions and unsupported values; the policy selector must not use that lossy route.
The existing classifier assigns Lambda function create medium, update high, delete/both replacements critical;
`aws_lambda_layer_version` is unlisted. These are observations of source, not new runtime test results.

Issue read attempts on 2026-11-15: `gh issue view` for #107/#110/#114/#115 could not authenticate; public web fetches
also failed. No live issue body or state was verified or edited. The ticket narrowing below is a local replacement
scope prepared from the checked-in design's decision map, not a claim that remote tickets were updated.

## Source selection

Choose a dedicated optional string attribute `risk_policy` in root `defaults`, Member top level, and matching
Member workspace blocks. It selects shared assessment logic, independently of whether the Risk view is enabled.
Keep generic `plugin` blocks root-only. Decline adding Member plugin overrides: it expands unrelated settings and
invites per-key policy mixing. Decline placing source only in root plugin options: it cannot express complete
Member/workspace replacement through the currently supported HCL.

Exact proposed files (unsupported until implementation):

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

| Selection case | First-version behavior |
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
duplicate/conflicting packages or undefined entrypoints cannot trigger loading elsewhere. Compilation/result
validation, rather than filenames, enforces the complete entrypoint contract from D-01a.

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
Always include `policy.kind` and `policy.digest` separately as in D-01a. No absolute source path in reports.
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
not runtime preset names. US-007 supplies tested full attribute-aware examples after input/runtime evidence.

Choose ordinary Rego author tests alongside isolated resource input fixtures, using the same OPA release chosen
by D-03 and the final published input version. Minimal author test for the template above:

```rego
# main_test.rego
package tfui.risk.v1_test
import rego.v1
import data.tfui.risk.v1

test_explicit_default if {
    result := v1.classification with input as {"change": {"actions": ["delete"]}}
    result == {"level": "low", "reasons": [{
        "code": "project-default",
        "message": "Explicit project default for this mutation.",
        "paths": [],
    }]}
}
```

Proposed author command: `opa test main.rego reason.rego main_test.rego --fail-on-empty -v`. This minimal input
tests policy logic only; it deliberately is not a valid tfui envelope. US-007's fixtures must use full D-01c
envelopes and verify tfui validation/capabilities too. `opa test` alone proves neither eligibility nor the embedded
capability boundary nor valid tfui results. OPA documents `with input`, test discovery and `--fail-on-empty` in its
[testing guide](https://www.openpolicyagent.org/docs/policy-testing). No OPA tests were executed in this bead;
these are author-workflow examples for later verification, not evidence of an accepted runtime pin.

Built-in policy ships as embedded complete source, with the current fixed action/type severities preserved for
v1; the eligibility contract supersedes the old classifier's unhandled-action `none`. Add evidence-based reasons,
not invented severity changes. New built-in bytes change the digest and invalidate assessments. Publish rule
changes in release notes and the canonical feature page with regression fixtures; schema compatibility alone
does not imply identical classifications. Authors needing unchanged decisions copy and pin source in their project.
tfui upgrades never rewrite copied templates/helpers or silently append new built-in rules. Test copies against
the selected runtime before upgrading; incompatible input/result majors fail explicitly per D-01a, without fallback.

## Lambda first-version choices

Retain the fixed classifier's levels, including for templates advertised as built-in-equivalent. Attribute-aware
reasons explain evidence without implying external deployment ownership. Decline automatic lowering of memory-only
or other non-code updates: the field report establishes a code-update concern, not a reviewed new severity scale.
Decline raising every layer mutation without independent justification. These are contributor choices, not owner
approval of the field report's suggested rules or a provider-version reproduction.

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
source evidence, then observed configuration update, then type/action default. Return the ordered set of known
changed code-source paths in one reason; do not let competing complete-rule objects resolve by source order.
Determine known equality only for known nodes; absent/null/unknown/sensitive are distinct. A hash unchanged at a
known value is evidence only about that field. A null-to-null hash proves no code equality. For a field absent
on both sides, report no evidence for that field, not known source equality. Eligible phantom mutations retain
these same rules. No provider schemas, package filesystem reads or external pipeline facts enter evaluation.

## Narrowed decision-ticket scopes

These are replacement scopes for older decision tickets, ready for later tracker publication. All inherit
ADR-0025's settled independent per-resource scope, complete results, custom replacement, eligibility and failure
behavior; none may reopen those questions as defaults. The design map links here so local preparation is discoverable.

| Ticket | Remaining question/evidence only | Already settled; remove from its decision scope |
| --- | --- | --- |
| #109 input fidelity | Prove D-01a tagged values preserve permitted evidence in live/seeded acquisition. | Whether input is plan-wide or per-resource. |
| #110 source identity | Prove this selector's provenance, precedence, strict failures, digest/reload and A→B rejection through real HCL/session wiring. | Parent-rule mixing, automatic fallback, silently ignored Member plugin blocks. |
| #112 trust | Test D-01a redaction, deterministic capabilities and diagnostic channels under the declared trust promise. | Whether evaluation failure can pass. |
| #114 result | Prove exact entrypoint/cardinality/schema, rule conflicts, empty/partial/unavailable presentation against D-01a. | Sparse results with built-in fallback; eligibility; independent versus combined findings. |
| #115 packaging | Verify explicit local helpers, complete copied templates, author workflow, upgrade isolation and the conservative Lambda table above. | Who owns a custom classification, hidden built-in defaults, runtime presets or parent composition. |
| #116 runtime | Pin/measure candidate builds, limits and failure controls under D-03's predeclared protocol. | Language selection unless experiment evidence reopens ADR-0024. |
| #117 reporting | Publish/test D-01a aggregation, thresholds, report/version migration and Blast Radius validity integration. | Policy enforcing caller thresholds or altering Apply. |
| #118 architecture review/consumers | Review D-01a selected-policy Blast Radius inputs and unavailable/partial validity handling with D-02/D-03 evidence before D-01c publication. Preserve existing impact arithmetic. | Reopening ownership, independent scope or eligibility; waiting for owner selection of exact first-version interfaces under this run's delegated authority. |
| #107 architecture map | Link reviewed evidence and publish the first-version contract in D-01c before implementation. | Treating design acceptance as a completed feature. |

## Experiment handoff and rejection criteria

D-02 must record fixtures, toolchain, baseline and protocol before execution. Use the exact HCL examples above
with root, inherited Member, Member override, exact/glob workspace, explicit empty, omitted and malformed selectors.
Exercise startup Chdir, root-as-Member and zero-config with nonexistent Terraform for seeded paths. Assert loaded
module bytes and declaring origin, not just option-map contents. Include disabled Risk view with Plan assessment.
Verify rejected Member plugin blocks remain rejected. Read root/child parse failures through actual session wiring;
any silent selection fallback rejects this candidate.

For the digest, compare identical modules in two directories, reordered enumeration, rename, byte/line-ending
edit, helper edit, test-only edit, added/deleted module, missing/empty/unreadable source and symlink cases. Before
runtime publication, block A with a controllable evaluator, switch to B, then release A; B must not display A's
result. Edit source while acquisition/evaluation is paused. Any observed mixed snapshot or late A result rejects
the lifecycle. Count Terraform calls: reopening Risk for identical evidence/source must not generate a new plan.

US-007 later verifies full fixtures for every Lambda row, both replacement orders, every hash state, unknown
source values, helper omission, explicit custom database defaults and source upgrades. D-03 proves the selected
runtime accepts author modules under the permitted capabilities. This bead records source inspection and decision
preparation only; no acquisition, evaluator, race, provider or author-test experiment has been run here.
