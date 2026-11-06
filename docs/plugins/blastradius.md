---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Blast Radius by Module"
id: blastradius
key: B
seo:
  description: "Group Terraform plan changes by module and rank each module by impact score, from its number of changes, their risk levels and any destructive operations."
description: "Describes how the Blast Radius plugin groups changes by module and scores their impact."
audience: "Users weighing a plan's reach, and contributors changing blast radius"
category: analysis
default_enabled: true
---

# Blast Radius

## Overview

The Blast Radius plugin groups plan changes by terraform module and calculates an impact score for each group. Impact is derived from the number of changes, risk levels, and whether destructive operations are involved. Modules are sorted highest-impact first, giving you a quick overview of which parts of your infrastructure are most affected.

## Screenshot

![Blast Radius]({{ site.baseurl }}/assets/demo/blastradius.gif)

## Interactive (TUI)

Press `B` to open the Blast Radius view. It requires a completed plan to analyze.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `j` / `k` | Navigate up/down | List |
| `Enter` / `Space` | Expand/collapse module changes | List |
| `Esc` | Go back | Always |

### Flow

```
Home ──B──→ Blast Radius (loading) ──→ Blast Radius (list)
                                          │
                                          ├── Enter/Space → Expand module changes
                                          └── Esc → Home
```

### Impact Score Calculation

| Score | Criteria |
|-------|----------|
| **critical** | Any change with critical risk |
| **high** | High risk or 3+ destructive operations |
| **moderate** | 3+ changes or medium risk |
| **minimal** | 1-2 changes, all low risk |

## Command Line (CLI)

Blast Radius is an analysis view within the TUI. It does not have a standalone CLI command.

Plan data with module groupings is available in structured form via:

```bash
tfui plan -project ./infra -json
```

The JSON output includes module information on each change for downstream grouping.

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| See module-level impact | `tfui plan -json` (group by module) | Press `B` |
| Identify highest-risk module | Parse JSON output by module | Sorted by impact score |
| Drill into module changes | `tfui plan -json \| jq` | `B` → expand module |

## Configuration

```hcl
# tfui.hcl
plugin "blastradius" {
  enabled = true
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |

## Related

- [Plan](plan.md) -- flat list of all changes
- [Risk Analysis](risk.md) -- changes grouped by risk level
- [Phantom Changes](phantom.md) -- phantom changes flagged in module view
