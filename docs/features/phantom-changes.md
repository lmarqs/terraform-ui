---
layout: default
title: Phantom Changes
parent: Features
nav_order: 2
description: Detect and filter false-positive terraform plan changes
---

# Phantom Change Detection

Terraform sometimes reports changes that won't actually modify infrastructure. These "phantom" changes appear because of serialization differences, provider normalization, or state representation quirks.

## What Are Phantom Changes?

A phantom change is an `update` action where the before and after values are semantically identical. Common causes:

- **Null vs absent**: A field stored as `null` in state but omitted in the plan
- **Array ordering**: Tags or list attributes serialized in different orders
- **Provider normalization**: Trailing slashes, case differences, whitespace
- **Computed attribute refresh**: Values that get re-read but haven't changed

## How Detection Works

terraform-ui normalizes both sides of a change before comparing:

1. **Strip null values** — treats `null` as equivalent to missing
2. **Sort arrays** — compares arrays by content regardless of order
3. **Recursive normalization** — handles nested objects and arrays
4. **Deep equality** — compares the normalized structures

If the normalized before and after are identical, the change is marked as phantom.

## In the TUI

Phantom changes are:
- Visually dimmed in the plan review
- Shown with a `👻` indicator
- Collapsible/hideable via filter
- Excluded from risk analysis by default

## In Non-Interactive Mode

```bash
# Phantom detection via enrichment command
tfui show -json tfplan.out | tfui phantom --json | jq '.phantom_changes'
```

Output:
```json
{
  "phantom_changes": 2,
  "real_changes": 5,
  "phantom_resources": [
    "aws_security_group.web",
    "aws_iam_role.lambda_exec"
  ]
}
```
