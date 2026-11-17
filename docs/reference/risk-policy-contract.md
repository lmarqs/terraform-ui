---
layout: default
title: Risk policy contract
parent: Reference
description: "Defines the accepted purpose, responsibilities, eligibility and failure semantics of the planned risk policy contract."
audience: "Policy authors and contributors designing risk classification and its consumers"
---

# Risk policy contract

This page defines the accepted behavior of the **planned** policy interface. Custom policy evaluation and
`tfui risk` are not implemented in the current checkout. This is a semantic contract, not a published Rego
entrypoint or JSON schema. The [architecture design](https://github.com/lmarqs/terraform-ui/blob/research/risk-plugin/docs/development/risk-policy-design.md) explains the choices
and identifies the remaining decisions. The [Risk plugin guide](../plugins/risk.md) describes current availability.

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
| Policy source | The local source selected through configuration for a custom policy; precise file/directory and path-resolution rules are not yet specified. |
| Classification | A risk level and its explanation for one resource mutation. |
| Assessment | The result of classifying the eligible mutations in a supplied plan, or an explicit failure to do so. It makes no claim about infrastructure outside that plan. |
| Not assessed | A change outside this version's assessment eligibility; not a severity and not an implicit classification of `none`. |
| Unavailable | An assessment could not be completed because of an evaluation or validation failure; not a severity. |
| Unknown value | A value not known in the plan; distinct from a known null, an absent field, and an assessment failure. Its policy representation is still to be specified. |
| Threshold | A comparison used by tfui to interpret classifications for the risk command. The policy does not choose the caller's response. |
| Template | Explicit policy source copied and owned by a project, rather than a named runtime preset. |

The existing ordered severity vocabulary is `none`, `low`, `medium`, `high`, `critical`. In the planned contract,
`none` is an explicit policy classification. It must not stand in for missing results, excluded changes, failures,
or a general guarantee of safety. The precise encoded result and versioning rules remain undecided.

## Responsibilities

| Owner | Responsibility | Boundary |
| --- | --- | --- |
| Policy | Classify and explain each submitted resource mutation using its own change data. | No cross-resource references, plan-wide findings, threshold enforcement or Apply decisions. |
| tfui | Select the policy, preserve the reviewed plan's evidence, determine eligibility, validate classifications, aggregate them, render the report and perform threshold comparison. | No hidden fallback for incomplete custom-policy output. Aggregation formula and threshold settings are not yet chosen. |
| Caller, human or automation | Decide what to do with the report and command result. | This feature does not block or modify `tfui apply`. |

Independent classification is a semantic boundary: a resource's classification cannot depend on other resources
in the plan. Whether the implementation batches calls internally is not a promise to policy authors. Unknown and
sensitive metadata must not be reconstructed from rendered diff strings; the exact permitted input remains an
architecture decision.

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
explicitly classified as `none`; the command/report representation of that case remains to be decided.

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
second rule language. Explicit templates replace named runtime presets. Exact source-loading, explicit reuse of
helpers, reload and configuration-key semantics remain to be specified.

## Failure behavior

When risk evaluation fails, Plan remains reviewable with risk clearly marked **unavailable** and the error
visible. `tfui risk` reports an error result, never a passing assessment. Apply behavior remains unchanged.

The architecture still needs to decide how partial classifications are presented, how diagnostics avoid sensitive
values, and the exact command exit codes and output schema. None of those details may disguise failure as `none`,
reuse a previous assessment as if it were current, or introduce an implicit fallback policy.

## Review presentation

The intended Risk view shows the mutation itself alongside its classification: action symbol, resource address,
and compact attribute diff. It does not repeat a `Change` label on each row. Inspection exposes the full diff and
classification explanation. Plan and Risk share action/risk styles and attribute-diff rendering; plugin-local
copies of symbol or color tables would contradict that design.

The visual conventions belong to [TUI UX](tui-ux.md). Unknown and sensitive notation, viewport navigation and
classification failures must remain understandable in narrow layouts. Risk does not add an Apply confirmation
flow. Older browser mockups are visual experiments, not authority for Apply behavior or policy semantics.

## Remaining interface decisions

The accepted semantics above do not choose an entrypoint, input envelope, result field names, diagnostic format,
multiple-rule resolution, aggregation formula, threshold default/override syntax, source paths, reload timing,
trust/capability boundary, version migration, or the effect of custom/unavailable assessments on existing Blast
Radius impact calculations. Those decisions are tracked in the
[architecture map](https://github.com/lmarqs/terraform-ui/issues/107) and analyzed in the
[design](https://github.com/lmarqs/terraform-ui/blob/research/risk-plugin/docs/development/risk-policy-design.md#remaining-decisions). No executable configuration or schema example
is published here until those choices are accepted.

## Decision records

- [Embed Rego for project risk policies](../adr/0024-embed-rego-for-risk-policies.md).
- [Classify each resource mutation explicitly](../adr/0025-classify-resource-mutations-explicitly.md).
