---
layout: plugin
title: Apply
id: apply
key: a
description: Apply terraform changes with confirmation and elapsed time tracking
category: operations
default_enabled: true
---

## Overview

The Apply plugin executes `terraform apply` with an interactive confirmation prompt. It tracks elapsed time during the apply and reports success or failure with duration. Targets can be scoped to specific resources.

## Usage

Press `a` from the Plan view (or any view) to start an apply. You will be prompted for confirmation before any changes are made.

| Key | Action |
|-----|--------|
| `y` / `Enter` | Confirm apply |
| `n` / `Esc` | Cancel apply |
| `r` | Retry after failure |

## Configuration

```yaml
# tfui.yaml
plugins:
  apply:
    enabled: true
    targets:
      - "module.networking"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |
| `targets` | list | `[]` | Resource targets passed to `terraform apply -target` |

## Screenshots/Output

Confirmation prompt:

```
Apply

Are you sure you want to apply these changes?
This will modify your infrastructure.

[y]es / [n]o
```

Running state:

```
Apply

>>> Applying changes... 12s
```

Success state:

```
Apply

Apply complete! Resources are up-to-date.
Duration: 45s

Press Esc to go back
```

## Related

- [Plan](plan.md) -- review changes before applying
- [Risk Analysis](risk.md) -- assess risk before applying
