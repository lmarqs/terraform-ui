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

The Risk Analysis plugin groups plan changes by risk level (critical, high, medium, low, none) and displays an overall risk assessment. It provides a reason for each change's risk classification, such as destructive operations or modifications to critical resources.

## Screenshot

![Risk Analysis]({{ site.baseurl }}/assets/demo/risk-analysis.gif)

## Interactive (TUI)

Press `R` (uppercase) to open the Risk Analysis view. It requires a completed plan; without one, it shows "Run a plan first".

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` | Navigate up/down through groups and changes | List |
| `q` | Back to Home | Always |

### Flow

```
Home ──R──→ Risk Analysis (loading) ──→ Risk Analysis (grouped list)
                                           │
                                           ├── j/k → Navigate groups and changes
                                           └── q → Home
```

## Command Line (CLI)

Risk Analysis is an analysis view within the TUI. It does not have a standalone CLI command.

Risk is not in `-json` output. `tfui plan -ci` prints the overall level as its last line, for example `Risk: critical`.

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| See risk breakdown | — | Press `R` |
| Get overall risk level | `tfui plan -ci` (last line, e.g. `Risk: critical`) | Overall risk line in Plan |

## Related

- [Plan](plan.md) -- view the raw plan changes
- [Blast Radius](blastradius.md) -- module-level impact scoring
- [Phantom Changes](phantom.md) -- filter out cosmetic-only changes
