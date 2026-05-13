---
layout: plugin
title: Context
id: context
key: c
description: Select terraform project scope
category: navigation
default_enabled: true
---

## Overview

The Context plugin discovers and lists terraform scopes within a monorepo based on glob patterns defined in `tfui.hcl`. You can filter, select, and switch between scopes. The active scope determines the working directory for all other plugins (plan, apply, state, etc.).

## Usage

Press `c` to open the Context view. It scans for scopes matching your configured path patterns.

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | Select scope as active |
| Any character | Filter by path/name |
| `Backspace` | Remove last filter character |
| `r` | Re-discover scopes |
| `Esc` | Go back |

## Configuration

```yaml
# tfui.hcl
plugins:
  context:
    enabled: true

scope:
  paths:
    - "modules/*"
    - "envs/**"
    - "stacks/networking"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |
| `scope.paths` | list | `[]` | Glob patterns for scope discovery |

## Screenshots/Output

```
Context

* envs/production (production)
  envs/staging (staging)
  modules/networking (networking)
  modules/compute (compute)
  stacks/networking (networking)

5 scope(s)

Enter select  / filter  r refresh  Esc back
```

With filter:

```
Context

filter: net

  modules/networking (networking)
  stacks/networking (networking)

2/5 scope(s)

Enter select  / filter  r refresh  Esc back
```

## Related

- [Workspaces](workspaces.md) -- manage workspaces within the active scope
- [State Browser](state.md) -- browse state for the active scope
- [Plan](plan.md) -- plan runs against the active scope directory
