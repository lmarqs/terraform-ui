---
layout: default
title: "ADR-0025: Classify each resource mutation explicitly"
grand_parent: Development
parent: Architecture
nav_order: 0025
description: "Records why risk policies independently classify mutations with explicit results and no hidden fallback."
audience: "Contributors changing policy scope, classification ownership or assessment failure behavior"
---

# Classify each resource mutation explicitly

Risk policies independently classify planned resource mutations and explain their risk. They do not inspect
other resources, issue plan-wide findings, or decide what to apply. tfui owns aggregation and threshold comparison;
the caller decides what to do with the information. The repository owner accepted this boundary in the
[assessment-purpose discussion](https://github.com/lmarqs/terraform-ui/issues/108). It records intended behavior,
not an implemented policy API.

The owner chose independent resource classification as sufficient for the initial plugin. A broader plan-wide
policy program would make a resource's level depend on other changes and expand the feature into decisions the
owner does not want the policy to make. The guiding principle is to make consequential behavior explicit.

Each submitted create, update, delete or replacement requires an explicit classification. Missing or undefined
output is an assessment error. Custom policy selection fully replaces the built-in classifier; no custom source
means built-in classification, while a custom source is responsible for its own default cases. There is no hidden
built-in fallback or implicit ancestor-rule mixing.

Phantom labels do not suppress eligible mutations or force risk to `none`. Reads, no-ops, pure imports, pure moves
and output-only changes are explicitly not assessed. An import or move accompanied by a mutation is assessed for
that mutation. These exclusions are not safety claims.

Policy failure leaves Plan review available with risk marked unavailable and the error visible. The risk command
returns an error outcome. Apply behavior does not change. The [contract reference](../reference/risk-policy-contract.md)
owns these definitions; the [architecture design](../development/risk-policy-design.md) holds the alternatives,
evidence, remaining decisions and confirmation criteria.

## Consequences

- Classification can be explained and tested from one resource's change data. Cross-resource outage and compliance
  rules are outside this contract; adding them would require an explicit scope decision.
- Authors must cover every submitted mutation, including generic cases. Templates help authors start with a
  complete policy, but tfui cannot silently complete a partial custom policy.
- `none`, not assessed and unavailable have different meanings. Consumers must not collapse them into one value.
- Existing phantom detection cannot establish a bypass: it normalizes array order and null-valued object fields.
  The implementation must preserve the classification boundary even when presentation labels a mutation phantom.
- tfui must distinguish evaluation success from a valid, complete classification. OPA can return an undefined
  query without a Go error; the [runtime research](../development/risk-policy-runtime-research.md) explains this.
- Input/output representation, sensitive-data access, capabilities, multiple-rule resolution, aggregation,
  threshold settings and policy loading remain separate decisions. This ADR accepts none of their draft defaults.

## Considered Options

### Keep the fixed classifier

The smallest implementation and no policy authoring cost. It cannot express project-specific attribute rules
without a tfui release; the owner already chose embedded Rego in ADR-0024.

### Let policies judge the entire plan

Supports combined-change rules, such as detecting deletion of every replica represented in a plan. Rejected for
the initial contract because the owner wants independent risk indication, without cross-resource references or
policy-owned responses to the assessment.

### Accept sparse custom results and fill gaps from built-in rules

Lets authors write only special cases. Rejected because an omitted classification would acquire a hidden meaning
and the effective policy would depend on two sources. An explicit default in the selected policy serves the
simple-authoring case without implicit fallback.

### Suppress phantom mutations automatically

Reduces noise but delegates a classification decision to normalization heuristics. Rejected: eligible mutations
still require policy output, and meaningful array/null changes can be lost by existing normalization.

### Stop Plan review or treat evaluation failures as neutral

Stopping review makes a risk-policy failure disable the underlying change inspection; treating failure as neutral
misrepresents the evidence. Keep review available and make unavailability explicit instead.
