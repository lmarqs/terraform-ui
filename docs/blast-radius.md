---
layout: default
title: Blast Radius
description: Visualize the scope and impact of terraform changes
---

# Blast Radius

The blast radius view shows the scope of your planned changes — how many resources are affected, which modules are involved, and the potential impact on your infrastructure.

## What It Shows

- **Module grouping** — Changes organized by terraform module path
- **Action summary** — Count of creates, updates, deletes per module
- **Dependency visualization** — Which modules depend on changed resources
- **Impact score** — Combined risk assessment across all changes

## Module Grouping

Changes are grouped by their terraform module path:

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

## In the TUI

Press `b` from the home screen after running a plan. The blast radius view shows:

1. **Summary bar** — Total changes, overall risk level
2. **Module tree** — Expandable tree grouped by module
3. **Per-module stats** — Add/change/destroy counts with risk indicators

## Module Path Extraction

terraform-ui extracts module paths from resource addresses:

| Address | Module |
|---------|--------|
| `aws_instance.web` | `root` |
| `module.vpc.aws_subnet.private` | `module.vpc` |
| `module.vpc.module.subnets.aws_subnet.a` | `module.vpc.module.subnets` |
