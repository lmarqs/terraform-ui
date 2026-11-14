---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Phantom Changes: Spot No-Op Diffs"
id: phantom
key: P
seo:
  description: "Find Terraform plan changes that modify no infrastructure, such as JSON field reordering or tag order differences, and see why each one is cosmetic."
description: "Describes how the Phantom Changes plugin detects and explains no-op changes in a plan."
audience: "Users reviewing a plan full of diffs that change nothing"
category: analysis
default_enabled: true
---

# Phantom Changes

## Overview

The Phantom Changes plugin identifies plan changes that are cosmetic only -- they appear in the plan output but result in no actual infrastructure modification. Each phantom change includes an explanation of why it is a no-op. [What counts as phantom](../features/phantom-changes.md) lists the causes tfui detects.

## Screenshot

![Phantom Changes]({{ site.baseurl }}/assets/demo/phantom.gif)

## Interactive (TUI)

Press `P` (uppercase) to open the Phantom Changes view. It requires a completed plan to analyze.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` | Navigate up/down | List |
| `Enter` / `i` | Expand/collapse phantom details | List |
| `q` | Back to Home | Always |

### Flow

```
Home ──P──→ Phantom Changes (loading) ──→ Phantom Changes (list)
                                             │
                                             ├── Enter/i → Expand phantom details
                                             └── q → Home
```

## Command Line (CLI)

Phantom Changes is an analysis view within the TUI. It has no CLI command of its own, and phantom analysis is available only in the TUI.

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Identify phantom changes | — | Press `P` |
| Spot phantoms while reviewing | — | `(phantom)` mark on the change in Plan |
| See phantom explanations | — | `P` → expand change |

## Related

- [Plan](plan.md) -- see all changes including phantoms
- [Risk Analysis](risk.md) -- phantoms are marked as cosmetic in risk view
- [Blast Radius](blastradius.md) -- phantoms are flagged in module groups
