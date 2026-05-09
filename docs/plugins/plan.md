---
layout: plugin
title: Plan Review
id: plan
key: p
description: Review terraform plan changes with expandable attribute diffs
category: operations
default_enabled: true
---

## Overview

The Plan plugin runs `terraform plan` and presents the results in an interactive tree view. Each resource change is displayed with its action type, risk level, and phantom status. You can expand individual changes to inspect attribute-level diffs.

## Usage

Press `p` to open the Plan view. The plugin immediately runs a plan against the current working directory (or targeted resources if configured).

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` / `Space` | Expand/collapse attribute diffs |
| `g` / `G` | Jump to first/last change |
| `r` | Re-run plan |
| `a` | Switch to Apply |
| `Esc` | Go back |

## Configuration

```yaml
# tfui.yaml
plugins:
  plan:
    enabled: true
    targets:
      - "module.networking"
      - "aws_instance.web"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |
| `targets` | list | `[]` | Resource targets passed to `terraform plan -target` |

## Screenshots/Output

```
Plan Review

 > + aws_instance.web                          [low]
   ~ aws_security_group.main                   [medium]
   - aws_s3_bucket.old                         [HIGH]
   -/+ aws_db_instance.primary                 [CRITICAL]

Plan: 1 to add, 1 to change, 1 to destroy, 1 to replace
Overall risk: CRITICAL

j/k navigate  Enter expand  r refresh  a apply  Esc back
```

Expanded attribute diff:

```
 v ~ aws_security_group.main                   [medium]
    ingress: "0.0.0.0/0" -> "10.0.0.0/8"
    tags.env: "staging" -> "production"
```

## Related

- [Apply](apply.md) -- apply the planned changes
- [Risk Analysis](risk.md) -- risk breakdown by severity
- [Blast Radius](blastradius.md) -- module-grouped impact view
