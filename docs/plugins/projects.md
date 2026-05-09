---
layout: plugin
title: Projects
id: projects
key: m
description: Navigate terraform projects in a monorepo
category: navigation
default_enabled: true
---

## Overview

The Projects plugin discovers and lists terraform projects within a monorepo based on glob patterns defined in `tfui.yaml`. You can filter, select, and switch between projects. The active project determines the working directory for all other plugins (plan, apply, state, etc.).

## Usage

Press `m` to open the Projects view. It scans for projects matching your configured path patterns.

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
  projects:
    enabled: true

projects:
  paths:
    - "modules/*"
    - "envs/**"
    - "stacks/networking"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |
| `projects.paths` | list | `[]` | Glob patterns for project discovery |

## Screenshots/Output

```
Projects

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
Projects

filter: net

  modules/networking (networking)
  stacks/networking (networking)

2/5 project(s)

Enter select  / filter  r refresh  Esc back
```

## Related

- [Workspaces](workspaces.md) -- manage workspaces within the active project
- [State Browser](state.md) -- browse state for the active project
- [Plan](plan.md) -- plan runs against the active project directory
