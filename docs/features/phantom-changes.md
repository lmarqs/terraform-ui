---
layout: default
title: Terraform Phantom Change Detection
parent: Features — Terraform Plan Analysis Tools
nav_order: 2
seo:
  description: Detect false-positive terraform plan changes — null vs absent fields and reordered arrays. Stop chasing ghost diffs.
description: "Explains which plan changes tfui treats as phantom, such as null versus absent or reordered arrays, and why."
audience: "Users chasing diffs that change nothing, and contributors changing detection"
---

## Overview

tfui marks an update as phantom when every changed attribute is equal once null fields are dropped and arrays are
sorted. Such a diff appears in the plan but changes nothing, so you can skip it and review the real changes.

## How It Works

A phantom change is an `update` action whose before and after values are equal after normalization. Two causes are
detected:

- **Null vs absent** -- a field inside an object attribute set to `null` on one side and missing on the other
- **Array ordering** -- tags or list attributes serialized in different orders

tfui normalizes both sides of each changed attribute before comparing:

1. Strip null values -- treats an object field set to `null` as equivalent to a missing one
2. Sort arrays -- compares arrays by content regardless of order
3. Recursive normalization -- applies both steps to nested objects and arrays
4. Equality -- compares the normalized values

If every changed attribute is equal after normalization, the change is marked as phantom. Deletes, creates and
replaces are never phantom, and neither is an update with no attribute diff.

## In the TUI

- In the plan review, a phantom change is marked `(phantom)` in faint italic text, and its inspect view says
  `Phantom: yes`.
- In the Risk view, a phantom change keeps the risk level of its action and resource type.

Press `P` from the home screen after running a plan to see only phantom changes with explanations.

## Configuration

```hcl
# tfui.hcl (project root)
defaults {
  plugin "phantom" {
    enabled = false
  }
}
```

The detection rules are fixed; `enabled` is the plugin's only setting. See the
[Phantom Changes plugin](../plugins/phantom.md) for the full TUI reference.
