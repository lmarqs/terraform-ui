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

The Context plugin discovers and lists terraform projects within a monorepo based on glob patterns defined in `tfui.yaml`. You can filter, select, and switch between projects. The active context determines the working directory for all other plugins (plan, apply, state, etc.).

## Usage

Press `c` to open the Context view. It scans for projects matching your configured path patterns.

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | Select project as active |
| Any character | Filter by path/name |
| `Backspace` | Remove last filter character |
| `r` | Re-discover projects |
| `Esc` | Go back |

## Configuration

```yaml
# tfui.yaml
plugins:
  context:
    enabled: true

context:
  paths:
    - "modules/*"
    - "envs/**"
    - "stacks/networking"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |
| `context.paths` | list | `[]` | Glob patterns for project discovery |

## Screenshots/Output

```
Context

* envs/production (production)
  envs/staging (staging)
  modules/networking (networking)
  modules/compute (compute)
  stacks/networking (networking)

5 project(s)

Enter select  / filter  r refresh  Esc back
```

With filter:

```
Context

filter: net

  modules/networking (networking)
  stacks/networking (networking)

2/5 project(s)

Enter select  / filter  r refresh  Esc back
```

## Related

- [Workspaces](workspaces.md) -- manage workspaces within the active context
- [State Browser](state.md) -- browse state for the active context
- [Plan](plan.md) -- plan runs against the active context directory
