---
layout: default
parent: Plugins
title: Blast Radius
id: blastradius
key: b
description: Visualize module-grouped changes with impact scores
category: analysis
default_enabled: true
---

## Overview

The Blast Radius plugin groups plan changes by terraform module and calculates an impact score for each group. Impact is derived from the number of changes, risk levels, and whether destructive operations are involved. Modules are sorted highest-impact first, giving you a quick overview of which parts of your infrastructure are most affected.

## Usage

Press `b` to open the Blast Radius view. It requires a completed plan to analyze.

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` / `Space` | Expand/collapse module changes |
| `Esc` | Go back |

## Configuration

```yaml
# tfui.hcl
plugins:
  blastradius:
    enabled: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |

## Screenshots/Output

```
Blast Radius

CRITICAL BLAST RADIUS  3 module(s) affected, 7 total change(s)

 > module.database (3 changes)  [CRITICAL]  ~1 -1 -/+1
   module.networking (2 changes)  [moderate]  ~2
   root (2 changes)  [minimal]  +2

j/k navigate  Enter expand  Esc back
```

Expanded module:

```
 v module.database (3 changes)  [CRITICAL]  ~1 -1 -/+1
     ~ aws_rds_cluster.main              [HIGH]
     - aws_rds_cluster_instance.reader   [CRITICAL]
     -/+ aws_rds_cluster_instance.writer [CRITICAL] (phantom)
```

### Impact Score Calculation

| Score | Criteria |
|-------|----------|
| **critical** | Any change with critical risk |
| **high** | High risk or 3+ destructive operations |
| **moderate** | 3+ changes or medium risk |
| **minimal** | 1-2 changes, all low risk |

## Related

- [Plan](plan.md) -- flat list of all changes
- [Risk Analysis](risk.md) -- changes grouped by risk level
- [Phantom Changes](phantom.md) -- phantom changes flagged in module view
