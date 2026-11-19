---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Plan Changes Grouped by Risk Level"
id: risk
key: R
seo:
  description: "Group Terraform plan changes by risk level, critical, high, medium, low or none, with an overall assessment and the reason behind each classification."
description: "Describes how the Risk Analysis plugin groups planned changes by risk level."
audience: "Users who know Terraform plans and want to review or report their risk classifications"
category: analysis
default_enabled: true
---

# Risk Analysis

## Overview

Risk Analysis groups eligible resource mutations from the reviewed plan by the selected policy's classification,
from critical to none, with the policy's explanation beside each resource. Plan uses the same assessment for its
risk badges; Blast Radius uses those classifications in its impact calculation. The built-in rules are described
in [Risk analysis](../features/risk-analysis.md). A custom policy completely replaces them.

Open Risk after Plan to reuse the reviewed evidence without generating a second plan. Without a current plan,
tfui acquires one for Risk using the active Context's plan options and cleans up its temporary saved artifact.
Risk provides review and reporting; it adds no Apply action.

## Interactive (TUI)

Press `R` (uppercase) to open Risk Analysis. Resource rows show the original action symbol, address, policy
explanation and compact attribute changes. Additional changes have an overflow count; inspect a resource for the
full diff and reasons. Unknown values keep their unknown notation and sensitive values stay masked.

Only resources are cursor items; group headings are skipped. The title shows the cursor/count, and a scroll
gutter appears when the list exceeds the viewport.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` / `↑` / `↓` | Navigate resources | Mutation list |
| `g` / `G` | Jump to the first / last mutation | Mutation list |
| `←` / `→` | Scroll the list horizontally when wrapping is off | Mutation list |
| `Ctrl+w` | Toggle wrapping | Mutation list |
| `Enter` | Inspect the selected mutation's full diff and reasons | Mutation list |
| `↑` / `↓`, `g` / `G` | Scroll detail; jump to start/end | Inspection |
| `←` / `→` | Scroll horizontally when wrapping is off | Inspection |
| `Ctrl+w` | Toggle wrapping | Inspection |
| `Ctrl+r` | Request the assessment again for the same plan | Plan failed or assessment unusable (not an invalid `-risk-threshold`, which a retry cannot change) |
| `Esc` | Close inspection; leave Risk from the list | Inspection or list |
| `q` | Close inspection first; from the list, return Home (quit in standalone mode) | Inspection or list |

### Assessment states

A complete nonempty assessment shows its maximum level and groups the classified mutations. An explicit
`none` is a policy classification, not a safety guarantee. With no eligible mutations, the view instead shows
`No eligible mutations (N changes not assessed).` Reads, no-ops and pure moves/imports are excluded; a phantom
label does not exclude a mutation.

Policy or input failure shows unavailable risk. A partial assessment retains classified resources and failed
rows for inspection, but supplies no overall classification. Plan review remains available on policy failure;
Terraform planning failures still report the Terraform error.

Source and configuration are rechecked at assessment boundaries, including Risk activation and Context changes.
Edits are detected at the next boundary; there is no background watcher. See the
[source lifecycle](../reference/risk-policy-contract.md#reload-and-publication).

## Command Line (CLI)

```bash
tfui risk -ci -plan ./plan.json
tfui risk -ci -plan ./plan.json -json -risk-threshold high
terraform show -json tfplan.out | tfui risk -ci -plan -
```

`-plan` accepts Terraform show JSON from a file or stdin, so seeded analysis needs no Terraform binary.
Without it, tfui acquires a live plan for Risk. `-json` selects the versioned risk report; `tfui plan -json` continues to
emit Terraform JSON without risk fields. Show JSON is analysis input, not an executable saved plan.

The [CLI reference](../reference/cli-reference.md#tfui-risk) lists the command and flags. The
[policy contract](../reference/risk-policy-contract.md#command-threshold-and-report-v1) owns report schemas and
threshold comparison; the [I/O contract](../reference/cli-io-contract.md#risk-command-v1) owns channels and exits.

## Configuration

With no policy selector, Risk uses the embedded complete built-in policy. Select a local complete policy with
`risk_policy`; the [source selection contract](../reference/risk-policy-contract.md#source-selection) defines
root, Member and workspace precedence, declaration-relative paths and explicit reset to built-in.

```hcl
# Project tfui.hcl
defaults {
  risk_policy = "policies/team"
  plugin "risk" {
    threshold = "high"
  }
}
```

`threshold` controls command comparison and defaults to `off`; explicit `-risk-threshold` overrides it. Accepted
values and inclusive comparison are defined in the [threshold contract](../reference/risk-policy-contract.md#command-threshold-and-report-v1).
The `level` setting is rejected with a configuration error that names `threshold`.

Generic plugin settings such as `enabled` remain root-only; Member plugin blocks are rejected. Disabling the
Risk view does not disable the shared policy selector used by Plan. See [configuration](../guides/configuration.md)
for plugin enablement and the [authoring guide](../guides/risk-policy-authoring.md) for complete copyable policies
and isolated author tests.

## Related

- [Plan](plan.md) — review the plan tree and shared risk badges
- [Blast Radius](blastradius.md) — module-level impact from selected-policy classifications
- [Phantom Changes](phantom.md) — review advisory phantom labels
