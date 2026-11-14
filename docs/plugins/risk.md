---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Plan Changes Grouped by Risk Level"
id: risk
key: R
seo:
  description: "Group Terraform plan changes by risk level, critical, high, medium, low or none, with an overall assessment and the reason behind each classification."
description: "Describes how the Risk Analysis plugin groups planned changes by risk level."
audience: "Users who have run a plan and want its riskiest changes first"
category: analysis
default_enabled: true
---

# Risk Analysis

## Overview

The Risk Analysis plugin contains a view that groups a supplied plan summary by risk level. In the current
checkout, production activation and plan-completion events do not supply that summary, so opening the view stays
idle even after Plan completes. This is tracked in [Risk, Phantom and Blast Radius screens never receive a plan](https://github.com/lmarqs/terraform-ui/issues/76).

Plan's risk badges and headless text summary work independently of this view. Their fixed classification rules
are documented in [Risk analysis](../features/risk-analysis.md). Custom policies and `tfui risk` are planned, not
available configuration or commands; their accepted definitions are in the [policy contract](../reference/risk-policy-contract.md).

## Screenshot

![Risk Analysis]({{ site.baseurl }}/assets/demo/risk-analysis.gif)

## Interactive (TUI)

Press `R` (uppercase) to open the Risk Analysis view. The current idle message is "Run a plan first to analyze risk...". The missing plan-delivery wiring means running a plan does not currently populate this view.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` / arrows | Navigate groups and changes | Supplied-summary list |
| `q` | Back to Home | Always |

### Supplied-summary rendering

The view can render an overall level and groups from critical to none when supplied a summary directly. Reasons
are heuristic strings for some actions, not policy-authored explanations. The screenshot illustrates that
rendering; it is not evidence that the production plan-delivery path works.

## Command Line (CLI)

Risk Analysis is an analysis view within the TUI. It has no CLI command of its own.

Risk is not in `-json` output. `tfui plan -ci` prints the overall level as its last line, for example `Risk: critical`.

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| See risk breakdown | — | `R` view exists; plan-delivery wiring is incomplete |
| Get overall risk level | `tfui plan -ci` (last line, e.g. `Risk: critical`) | Overall risk line in Plan |

## Configuration

Risk currently reads no plugin-specific options: `Configure` ignores them. In particular, an option named `level`
does not configure a working threshold. The generic `enabled` setting is handled by the application; see
[configuration](../guides/configuration.md). Custom source selection and templates belong to the planned contract,
not the current HCL interface.

## Related

- [Plan](plan.md) -- view the raw plan changes
- [Blast Radius](blastradius.md) -- module-level impact scoring
- [Phantom Changes](phantom.md) -- filter out cosmetic-only changes
