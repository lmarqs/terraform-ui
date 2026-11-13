---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Blast Radius by Module"
id: blastradius
key: B
seo:
  description: "Group Terraform plan changes by module and rank each module by impact score, from its number of changes, their risk levels and any destructive operations."
description: "Describes how the Blast Radius plugin groups changes by module and scores their impact."
audience: "Users who have run a plan and want to see which modules it touches most, and contributors changing blast radius"
category: analysis
default_enabled: true
---

# Blast Radius

## Overview

The Blast Radius plugin groups plan changes by terraform module and calculates an impact score for each group. Impact is derived from the number of changes, risk levels, and whether destructive operations are involved. Modules are sorted highest-impact first, giving you a quick overview of which parts of your infrastructure are most affected.

## Screenshot

![Blast Radius]({{ site.baseurl }}/assets/demo/blastradius.gif)

## Impact score

| Score | Criteria |
|-------|----------|
| **critical** | Any change with critical risk |
| **high** | A change with high risk, or 3+ changes including at least one delete or replace |
| **moderate** | 3+ changes or medium risk |
| **minimal** | 1-2 changes, all low risk |

## Interactive (TUI)

Press `B` to open the Blast Radius view. It requires a completed plan to analyze.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` | Navigate up/down | List |
| `Enter` / `i` | Expand/collapse module changes | List |
| `q` | Back to Home | Always |

### Flow

```
Home ──B──→ Blast Radius (loading) ──→ Blast Radius (list)
                                          │
                                          ├── Enter/i → Expand module changes
                                          └── q → Home
```

## Command Line (CLI)

Blast Radius is an analysis view within the TUI. It does not have a standalone CLI command.

terraform's `-json` document carries a `module_address` on each resource change inside a module; root-module changes have none. Group by it to rebuild the view:

```bash
tfui plan -project ./infra -json
```

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| See module-level impact | `tfui plan -json` (group by `module_address`) | Press `B` |
| Drill into module changes | `tfui plan -json \| jq` | `B` → expand module |

## Related

- [Plan](plan.md) -- flat list of all changes
- [Risk Analysis](risk.md) -- changes grouped by risk level
- [Phantom Changes](phantom.md) -- phantom changes flagged in module view
