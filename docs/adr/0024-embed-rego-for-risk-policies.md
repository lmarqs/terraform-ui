---
layout: default
title: "ADR-0024: Embed Rego for project risk policies"
grand_parent: Development
parent: Architecture
nav_order: 0024
description: "Records why tfui uses embedded OPA/Rego for project risk policies instead of a custom matching language."
audience: "Contributors designing risk policy evaluation, configuration and plan data contracts"
---

# Embed Rego for project risk policies

Use Open Policy Agent (OPA), embedded through its Go evaluation API, to evaluate project risk policies written in Rego. HCL selects a local policy source; it does not define a second tfui-specific rule language. A member's selected policy replaces its parent's complete policy; tfui does not implicitly combine ancestor rules. Ship explicit policy templates rather than runtime named presets. The repository owner accepted this language and runtime decision on 2026-11-14. It describes intended implementation, not functionality already shipped.

The decision covers Risk analysis and `tfui risk`. It does not add a gate to `tfui apply`, change confirmation, or change auto-approval behavior. Evaluation uses structured data from the same Terraform plan being reviewed. Evaluation failures and invalid policy results must be reported as errors, not passing assessments.

The current classifier in `internal/terraform/risk.go` uses action and resource type. Its proposed replacement configuration would make tfui maintain a growing matching language, including operators, precedence, diagnostics and policy tests. Rego provides a policy language and test tooling; tfui remains responsible for the plan input and risk result contracts. CEL was the strongest alternative, but would leave the surrounding policy format, ordering and test conventions with tfui. [OPA integration](https://www.openpolicyagent.org/docs/integration), [policy testing](https://www.openpolicyagent.org/docs/policy-testing).

[Field report #106](https://github.com/lmarqs/terraform-ui/issues/106) supplies an attribute-aware example: changing a Lambda package path can overwrite code deployed by another pipeline. The current AWS provider's `needsFunctionCodeUpdate` includes `filename`; its update path reads the file and calls `UpdateFunctionCode`. This supports a reason naming the code-source attribute, even with an unchanged hash. An unchanged or null hash does not by itself establish that the package is a placeholder or that its lifecycle ignores the hash. The report justifies attribute-aware input, not OPA uniquely. [Provider implementation](https://github.com/hashicorp/terraform-provider-aws/blob/main/internal/service/lambda/function.go), inspected 2026-11-14; the report does not identify a provider version for reproduction.

The full analysis is in the [architecture design](../development/risk-policy-design.md). The [contract reference](../reference/risk-policy-contract.md) owns accepted definitions. [ADR-0025](0025-classify-resource-mutations-explicitly.md) subsequently narrows evaluation to independent resource mutations with explicit results and no custom-policy fallback.

## Consequences

- Users write Rego; maintainers own the embedded OPA dependency and its upgrades. Evaluation requires no separate OPA command or daemon. Local evaluation is the initial runtime; remote policy distribution is not part of this decision.
- Preserve structured plan values and unknown/sensitive metadata from the existing live or seeded acquisition path. `PlanSummary`'s display-string diffs are not a sufficient policy input. Do not run a second plan to acquire policy data.
- Keep evaluator dependencies in app internals behind SDK-facing domain types. Plan and Risk use the same validated classifications and reasons; plugins do not import each other or OPA implementation details.
- Source selection is whole-policy replacement. Switching member/workspace must resolve the applicable source and invalidate previous assessments; startup-only configuration is insufficient.
- Rego's declarative semantics do not inherit the earlier proposed first-match-wins behavior. Result shape, multiple-rule resolution, tfui aggregation, threshold configuration, unknown-value handling and migration of the ignored `level` option require a reviewed contract before implementation. ADR-0025 subsequently settles complete custom-policy replacement without fallback and missing-result failure. Selecting the language does not settle those choices.
- Use the Lambda case as an attribute-aware policy fixture and template candidate. Do not automatically lower non-code Lambda changes or assert external deployment ownership from hash equality alone.
- Before production integration, demonstrate policy tests, same-plan evaluation, failure handling and supported builds. Measure binary-size, compilation, evaluation and memory costs against the existing binary. No performance figures or acceptable limits have been measured or agreed. Unacceptable costs reopen this decision before a stable policy interface is released.

## Considered Options

### Keep fixed classification and add a Lambda rule

The smallest fix for #106. Rejected as the project-policy solution because each new policy requires a tfui code release. It remains a possible independently scoped fix; this record does not require the issue to wait for the entire policy integration.

### Define matching rules in HCL, YAML or JSON

Readable for a small lookup table. Rejected because changing serialization does not remove ownership of a custom policy language, diagnostics and tests.

### Embed CEL expressions

A credible fit for per-change conditions with familiar expression syntax. Rejected in favor of Rego's policy authoring and testing model; tfui would still own CEL's surrounding rule document, ordering and testing conventions. [CEL](https://cel.dev/).

### Invoke OPA or Conftest externally

Lets organizations manage their evaluator independently. Rejected for the initial integration because it adds runtime installation, version and process-management requirements. Conftest uses Rego; it is an execution/reporting alternative, not another language. [Conftest](https://www.conftest.dev/).

### Leave policy evaluation entirely in external CI

Requires the least tfui integration and can already use Terraform plan JSON. Rejected for this feature because external-only results do not provide the requested integrated risk classifications and review reasons.

### Merge parent policies or select named presets

Can reduce copied policy source, but adds effective-policy and upgrade semantics. The owner chose independently selected complete policies and explicit templates; composition or extends can be reconsidered separately.
