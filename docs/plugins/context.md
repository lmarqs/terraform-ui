---
layout: plugin
title: Context
id: context
key: C
description: Select terraform project chdir member
category: navigation
default_enabled: true
---

## Overview

The Context plugin discovers and lists chdir members within a monorepo based on `member` blocks defined in `tfui.hcl`. You can filter, select, and switch between members. The active chdir determines the working directory for all other plugins (plan, apply, state, etc.).

## Usage

Press `C` to open the Context view. It scans for members matching your configured paths.

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | Select member as active chdir |
| Any character | Filter by path/name |
| `Backspace` | Remove last filter character |
| `r` | Re-discover members |
| `Esc` | Go back |

## Configuration

```hcl
# tfui.hcl
member "modules/*" {}
member "envs/**" {}
member "stacks/networking" {}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable the plugin |

Members are declared as top-level `member "path" {}` blocks in `tfui.hcl`.

## Screenshots/Output

```
Context

* envs/production (production)
  envs/staging (staging)
  modules/networking (networking)
  modules/compute (compute)
  stacks/networking (networking)

5 member(s)

Enter select  / filter  r refresh  Esc back
```

With filter:

```
Context

filter: net

  modules/networking (networking)
  stacks/networking (networking)

2/5 member(s)

Enter select  / filter  r refresh  Esc back
```

## Related

- [Workspaces](workspaces.md) -- manage workspaces within the active chdir
- [State Browser](state.md) -- browse state for the active chdir
- [Plan](plan.md) -- plan runs against the active chdir directory
