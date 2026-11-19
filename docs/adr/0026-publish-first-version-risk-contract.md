---
layout: default
title: "ADR-0026: Publish the first-version risk contract"
grand_parent: Development
parent: Architecture
nav_order: 0026
description: "Records delegated first-version choices for risk policy interfaces after bounded fidelity and runtime evidence."
audience: "Contributors changing the risk authoring contract, reports or consumer compatibility"
---

# Publish the first-version risk contract

Publish the [v1 contract](../reference/risk-policy-contract.md) for dependent implementation, with tagged
single-mutation input, a complete Rego document, explicit source replacement and validated classifications.
The coordinator makes this first-version choice under the owner's explicit delegation on 2026-11-15; the exact
interfaces are not owner-selected. Stress and refine the contract after merge through explicit revisions.
This accepts a planned interface, not implemented policy evaluation or an available `tfui risk` command.

ADR-0024 chose embedded Rego and ADR-0025 chose independent explicit classification. Those boundaries alone
left authors and consumers unable to implement interoperable policies. D-01a/b supplied candidates; D-02
reproduced numeric/display loss, same-plan acquisition and source/Context identity limits. D-03 reproduced
result/capability controls and built all four release targets in a combined disposable tfui/OPA binary.
The [design](../development/risk-policy-design.md#first-version-decision-map) records each outcome and alternative;
the [publication review](https://github.com/lmarqs/terraform-ui/blob/research/risk-plugin/docs/development/risk-policy-evidence/contract-publication.md) preserves evidence limits.

Choose OPA v1.19.1 as the measured initial integration pin and accept its observed binary/build costs for the
single-binary approach. This is neither current-support nor security approval. Trusted local authors,
pre-evaluation sensitive redaction, deterministic positive capabilities and fixed diagnostics define the trust
promise. Cooperative cancellation is not a hard termination/memory boundary.

## Consequences

- Tagged nodes retain numbers as exact token strings and distinguish absence, null, unknown and redacted values.
  Authors pay an ergonomics cost; production validates source masks before subtree collapse.
- A custom source owns every eligible classification; helpers/templates are explicitly vendored copies.
  Configuration selects one declaration-relative file/directory snapshot, never mixed parent rules.
- tfui computes a maximum only over a complete nonempty assessment. Thresholds default to off and compare
  inclusively when explicitly selected. Empty, none, partial and unavailable remain distinct.
- Report v1 carries source/plan/Context provenance. Operational failure emits no command stdout; threshold met
  emits the completed report and exits 3. Terraform Plan JSON and Apply behavior remain unchanged.
- Blast Radius uses selected-policy levels with unchanged impact arithmetic only for complete required inputs.
  Missing risk makes affected impact unavailable rather than a zero severity.
- Snapshot/generation checks must reject late results and detect source changes at assessment boundaries.
  Existing acquisition/configuration paths need implementation fixes; experiments did not repair them.
- Binary growth of 15.0–16.2MiB and cold build growth of roughly 12.6–13.0 seconds are accepted measured costs,
  not budgets. Native non-host runtime, full-policy cost and hard isolation are not established.
- US production tests must verify session seed propagation, mask validation, strict config/source selection,
  full result/path validation, cancellation, race rejection and dependency upgrades before availability claims.

## Considered Options

### Retain fixed classification or postpone publication

The smallest change avoids authoring compatibility commitments, but leaves project customization and downstream
stories blocked. Prior evidence supports a workable first version, and the owner delegated its selection.
Keep the fixed severities as the built-in baseline without postponing the public interface.

### Native JSON values, sparse findings or implicit fallback

Simpler source is attractive, but display/float conversion loses evidence and sparse results obscure ownership.
Choose exact tagged values and a complete result with an explicit policy default. Undefined and conflicts fail.

### Runtime presets, inherited helper fragments or filesystem watching

These reduce copying or reload latency, but add hidden upgrade/ownership behavior and lifecycle complexity.
Choose local complete replacement, vendored helpers and boundary reload with explicit stale-result rejection.

### Arbitrary untrusted policy isolation

A separate process could support stronger resource containment, at installation and lifecycle cost. Embedded
controls establish no hostile-code sandbox. Keep trusted authors and a narrow capability profile; revisit
isolation only with a separate design and evidence, never infer it from cooperative cancellation.

### Weighted aggregation, implicit high threshold or partial-success reports

These can produce convenient gates but invent risk meaning or hide missing classifications. Choose a complete
maximum, explicit inclusive threshold and operational failure for partial assessment. Preserve empty stdout
on failure under the existing CLI contract; structured failure reporting needs a future versioned choice.
