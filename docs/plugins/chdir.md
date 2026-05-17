---
layout: default
parent: Plugins
title: Chdir Picker
id: chdir
key:
category: navigation
default_enabled: true
description: Select a member directory within a multi-module terraform project
---

## Overview

Select a member directory within a multi-module project. This is an internal plugin activated by the Context plugin (not directly accessible from the home menu). It uses NavPush behavior -- selecting a member or pressing `Esc` returns to the origin plugin.

## Interactive (TUI)

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Select member directory | List |
| `/` | Filter members | List |
| `Esc` | Cancel and return to origin | Always |

### Flow

```
Context ──→ Chdir Picker (list)
               │
               ├── Enter → ChdirChangedEvent → pop back to origin
               ├── / → Filter members
               └── Esc → Cancel → pop back to origin
```

## Command Line (CLI)

Not available as a standalone command. Use `--chdir` flag:

```bash
tfui plan --project ./infra --chdir modules/networking
```

## Configuration

Members are declared in `tfui.hcl`:

```hcl
member "modules/vpc" {}
member "modules/ecs" {}
member "modules/networking" {}
```

## Screenshots

```
Select Member

 > modules/vpc
   modules/ecs
   modules/networking

Enter select  / filter  Esc cancel
```

## Related

- [Context](context.md) -- parent plugin that activates chdir picker
- [Configuration](../guides/configuration.md) -- declaring members in tfui.hcl
