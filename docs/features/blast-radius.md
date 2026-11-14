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

Changes are grouped by their terraform module path. Each group shows add/change/destroy/replace counts and its impact score.

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

The [Blast Radius plugin](../plugins/blastradius.md#impact-score) scores each module group from its changes'
[risk levels](risk-analysis.md) and states the thresholds.
