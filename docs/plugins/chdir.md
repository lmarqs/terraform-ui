---
layout: default
parent: Plugins
title: Chdir Picker
id: chdir
category: internal
description: Select a member directory within a multi-module terraform project
---

# Chdir Picker

Select a member directory within a multi-module project.

## Behavior

This is an internal plugin activated by the Context plugin (not directly accessible from the home menu). It uses `NavPush` behavior — selecting a member or pressing `Esc` returns to the origin plugin.

### Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Select member directory |
| `/` | Filter members |
| `Esc` | Cancel and return to origin |

### Events

- Emits `ChdirChangedEvent` when a member is selected
- The app pops back to the `returnTo` plugin after selection

### Configuration

Members are declared in `tfui.hcl`:

```hcl
member "modules/vpc" {}
member "modules/ecs" {}
```
