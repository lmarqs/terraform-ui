---
layout: default
parent: Plugins
title: Phantom Changes
id: phantom
key: P
description: Detect and explain phantom (no-op) changes in terraform plans
category: analysis
default_enabled: true
---

## Overview

The Phantom Changes plugin identifies plan changes that are cosmetic only -- they appear in the plan output but result in no actual infrastructure modification. Common causes include JSON field reordering, tag ordering differences, and semantically equivalent value serialization. Each phantom change includes an explanation of why it is a no-op.

## Interactive (TUI)

Press `P` (uppercase) to open the Phantom Changes view. It requires a completed plan to analyze.

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `j` / `k` | Navigate up/down | List |
| `Enter` / `Space` | Expand/collapse phantom details | List |
| `Esc` | Go back | Always |

### Flow

```
Home ──P──→ Phantom Changes (loading) ──→ Phantom Changes (list)
                                             │
                                             ├── Enter/Space → Expand phantom details
                                             └── Esc → Home
```

## Configuration

```hcl
# tfui.hcl
plugin "phantom" {
  enabled = true
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |

## Screenshots

```
Phantom Changes

Detected 2 phantom change(s) out of 5 total
These changes appear in the plan but result in no actual infrastructure modification.

 > ~ aws_iam_policy.deploy      (3 attrs)
   ~ aws_s3_bucket.config       (1 attrs)

j/k navigate  Enter expand  Esc back
```

Expanded detail:

```
 v ~ aws_iam_policy.deploy      (3 attrs)
   Reason: JSON/policy field reordering or whitespace difference
     policy: {"Action":["s3:Get*"... = {"Action":["s3:Get*"...
     tags.managed_by: "terraform" = "terraform"
     tags_all.managed_by: "terraform" = "terraform"
```

## Related

- [Plan](plan.md) -- see all changes including phantoms
- [Risk Analysis](risk.md) -- phantoms are marked as cosmetic in risk view
- [Blast Radius](blastradius.md) -- phantoms are flagged in module groups
