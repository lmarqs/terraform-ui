---
layout: default
title: Blast Radius
parent: Features
nav_order: 3
description: Visualize the scope and impact of terraform changes
---

## Overview

The blast radius view shows the scope of your planned changes -- how many resources are affected, which modules are involved, and the potential impact on your infrastructure. Modules are sorted highest-impact first.

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

terraform-ui extracts module paths from resource addresses:

| Address | Module |
|---------|--------|
| `aws_instance.web` | `root` |
| `module.vpc.aws_subnet.private` | `module.vpc` |
| `module.vpc.module.subnets.aws_subnet.a` | `module.vpc.module.subnets` |

### Impact Score

| Score | Criteria |
|-------|----------|
| **critical** | Any change with critical risk |
| **high** | High risk or 3+ destructive operations |
| **moderate** | 3+ changes or medium risk |
| **minimal** | 1-2 changes, all low risk |

## In the TUI

Press `b` from the home screen after running a plan. The blast radius view shows:

1. **Summary bar** -- total changes, overall risk level
2. **Module tree** -- expandable tree grouped by module
3. **Per-module stats** -- add/change/destroy counts with risk indicators

## Configuration

```hcl
# tfui.hcl
plugin "blastradius" {
  enabled = true
}
```

See the [Blast Radius plugin](../plugins/blastradius.md) for the full TUI reference.
