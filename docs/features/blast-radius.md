---
layout: default
title: Terraform Blast Radius Visualization
parent: Features — Terraform Plan Analysis Tools
nav_order: 3
seo:
  description: See module-grouped impact of terraform changes before applying. Understand which modules are affected and how deeply.
description: "Explains how blast radius groups a plan's changes by module and scores how far they reach."
audience: "Users weighing a plan's reach, and contributors changing the scoring"
---

## Overview

The blast radius view shows the scope of your planned changes -- how many resources are affected, which modules are involved, and an impact score per module from its changes' risk levels, their count and whether any deletes or replaces. Modules are sorted highest-impact first.

## How It Works

### Module Grouping

Changes are grouped by their terraform module path. Each group shows add/change/destroy counts and an impact score derived from risk levels.

```
module.vpc (1 to add, 1 to change)
  + module.vpc.aws_subnet.new
  ~ module.vpc.aws_vpc.main

module.ecs (2 to destroy)
  - module.ecs.aws_ecs_service.api
  - module.ecs.aws_ecs_task_definition.api

root (1 to add)
  + aws_cloudwatch_log_group.app
```

### Module Path Extraction

tfui extracts module paths from resource addresses:

| Address | Module |
|---------|--------|
| `aws_instance.web` | `root` |
| `module.vpc.aws_subnet.private` | `module.vpc` |
| `module.vpc.module.subnets.aws_subnet.a` | `module.vpc.module.subnets` |

### Impact Score

| Score | Criteria |
|-------|----------|
| **critical** | Any change with critical risk |
| **high** | Any change with high risk, or 3+ changes in the module with at least one delete or replace |
| **moderate** | 3+ changes, or any change with medium risk |
| **minimal** | Fewer than 3 changes, none above low risk |

## In the TUI

Press `B` from the home screen after running a plan. The blast radius view shows:

1. **Summary line** -- the highest module impact, the number of modules affected and the total changes
2. **Module rows** -- one per module, each with its change count, impact badge and add/change/destroy/replace counts
3. **Expanded module** -- `Enter` lists the module's changes with their risk badges

## Configuration

```hcl
# tfui.hcl (project root)
defaults {
  plugin "blastradius" {
    enabled = false
  }
}
```

`enabled` is the plugin's only setting.

See the [Blast Radius plugin](../plugins/blastradius.md) for the full TUI reference.
