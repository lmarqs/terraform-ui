---
layout: default
parent: Plugins
title: Risk Analysis
id: risk
key: R
description: Analyze and group planned changes by risk level
category: analysis
default_enabled: true
---

## Overview

The Risk Analysis plugin groups plan changes by risk level (critical, high, medium, low, none) and displays an overall risk assessment. It provides a reason for each change's risk classification, such as destructive operations or modifications to critical resources.

## Interactive (TUI)

Press `R` (uppercase) to open the Risk Analysis view. It requires a completed plan -- if no plan has been run, it will prompt you to run one first.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `j` / `k` | Navigate up/down through groups and changes | List |
| `Esc` | Go back | Always |

### Flow

```
Home ──R──→ Risk Analysis (loading) ──→ Risk Analysis (grouped list)
                                           │
                                           ├── j/k → Navigate groups and changes
                                           └── Esc → Home
```

## Configuration

```hcl
# tfui.hcl
plugin "risk" {
  enabled = true
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |

## Screenshots

```
Risk Analysis

!! CRITICAL RISK DETECTED !!

--- CRITICAL (1) ---
   -/+ aws_db_instance.primary    destructive operation
--- HIGH (1) ---
   - aws_s3_bucket.old            destructive operation
--- MEDIUM (1) ---
   ~ aws_security_group.main      modification of critical resource
--- LOW (1) ---
   + aws_instance.web

Total: 4 changes  [critical: 1 | high: 1 | medium: 1 | low: 1]
j/k navigate  Esc back
```

## Related

- [Plan](plan.md) -- view the raw plan changes
- [Blast Radius](blastradius.md) -- module-level impact scoring
- [Phantom Changes](phantom.md) -- filter out cosmetic-only changes
