---
layout: default
title: Author a risk policy
parent: Guides
description: "Shows how to copy, test and select a complete local risk policy with isolated resource fixtures."
audience: "Policy authors who know Rego and want to customize tfui classifications"
---

# Author a risk policy

Copy the [Lambda policy example](https://github.com/lmarqs/terraform-ui/tree/main/examples/risk-policy/lambda)
into your project, run its author tests, then select its directory with `risk_policy`.
The example is complete project-owned source with an explicit default. It demonstrates
attribute evidence while retaining the reviewed Lambda levels. The
[policy contract](../reference/risk-policy-contract.md) owns input, result, source and trust semantics.

## Copy the complete source

From a tfui checkout, copy the example into your project's `policies/lambda` directory:

```sh
mkdir -p /path/to/project/policies
cp -R examples/risk-policy/lambda /path/to/project/policies/lambda
```

`main.rego` contains the complete classifier. `main_test.rego` contains ordinary Rego
author tests; `fixtures.json` contains named, isolated `tfui.risk.input/v1` envelopes.
Each fixture describes exactly one resource mutation. These tagged inputs are policy
test data, not Terraform show JSON accepted by `tfui --plan`.

The example specializes Lambda functions and layer versions. All other resources use
its explicit **low** project default, including the database deletion fixture. That
default demonstrates project ownership; review it for your project before adoption.
Selecting this source replaces the entire built-in classifier. No database rules,
installed helpers or parent rule fragments are added implicitly.

## Run isolated author tests

Use OPA **v1.19.1**, the embedded runtime version for this contract:

```sh
opa version
opa test /path/to/project/policies/lambda --fail-on-empty -v
```

The tests substitute one full fixture with `with input as ...` and check the returned
level and reason. The fixture names describe the evidence being varied:

- `filename-*-hash` changes the filename while the hash is unchanged, null, absent or
  unknown. Filename evidence still yields high; the reason does not infer deployed bytes.
- `hash-unknown`, `sensitive-source` and `memory-unknown-source` yield high with an
  uncertainty reason. Memory changes alongside unknown source evidence do not establish
  a memory-only change.
- `memory-known-sources` changes only memory with all listed code-source fields known
  equal. It remains high and explains configuration evidence without claiming a code upload.
- Function create, delete and both replacement orders preserve the reviewed action levels.
  Layer fixtures preserve the unlisted-type action defaults.
- `database-delete` checks the explicit project default and guards against hidden fallback.

When changing a rule, add an isolated fixture and a Rego assertion for the intended
classification and explanation. Keep absent entries, null, unknown and sensitive nodes
distinct. Never put real secrets in test fixtures or reason messages. The
[Lambda decision table](../reference/risk-policy-contract.md#lambda-first-version-choices)
owns the reviewed levels and reason priority.

## Check embedded capabilities and result validation

In a tfui checkout, run the same committed fixtures and author tests through the pinned
OPA dependency, plus the production source loader and evaluator:

```sh
mise run test:unit ./internal/riskpolicy/...
```

The template acceptance tests validate classifications, reason paths and source ownership
through `Assess`, using tfui's restricted capabilities and result validation. They also
execute the Rego author tests and fail if no tests are discovered. An unrestricted
`opa test` success alone does not prove compatibility with tfui's capability boundary or
result contract. After modifying your project copy, verify it with a real plan through
`tfui risk` as described below; the checkout tests exercise the committed example.

## Select and review your policy

In your project's `tfui.hcl`, select the copied directory in the root defaults:

```hcl
defaults {
  risk_policy = "policies/lambda"
}
```

Only the directory's immediate production `.rego` modules are loaded. Test modules,
fixture JSON and subdirectories are excluded. Imports must resolve within that selected
source. Relative paths originate at the configuration declaration; see
[source selection](../reference/risk-policy-contract.md#source-selection).

Review a Terraform show JSON plan using the selected policy:

```sh
tfui risk -ci -project /path/to/project -plan /path/to/plan.json -json
```

Check the report's custom policy identity, resource classifications and reasons. Its
[report and failure contract](../reference/risk-policy-contract.md#command-threshold-and-report-v1)
defines channels and exits. A malformed source or result makes risk unavailable; it never
falls back to the built-in classifier. Copied source stays yours across tfui upgrades:
retain your tests and rerun them against the new runtime before upgrading.
