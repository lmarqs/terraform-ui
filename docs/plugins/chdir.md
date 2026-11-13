---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Pick a Directory in a Multi-Module Terraform Project"
id: chdir
key: "—"
category: navigation
default_enabled: true
seo:
  description: "Select the member directory tfui runs terraform in, within a multi-module Terraform project. Opened from the Context screen; Esc returns to where you were."
description: "Describes how the Chdir picker selects which member of a project tfui runs terraform in."
audience: "Users of a project with several members, switching which one tfui runs terraform in"
---

# Chdir Picker

## Overview

Select the chdir: the member of the project that tfui runs terraform in. Selecting a member switches the chdir, resets the workspace to `default`, and returns you to Context.

## Screenshot

![Chdir Picker]({{ site.baseurl }}/assets/demo/chdir.gif)

## Interactive (TUI)

The picker has no key on Home: press `Enter` on the Chdir field of [Context](context.md). With a single member, tfui switches to it without showing the list.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `j` / `k` / `↑` / `↓` | Navigate up/down | List |
| `Enter` | Select member | List |
| `Esc` | Cancel and return to Context | Always |

### Flow

```
Context ──→ Chdir Picker (list)
               │
               ├── Enter → switch to that member, with workspace `default` → back to Context
               └── Esc → Cancel → back to Context
```

## Command Line (CLI)

Not available as a standalone command. Use the `-chdir` flag:

```bash
tfui plan -project ./infra -chdir modules/networking
```

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Select member | `-chdir <member>` flag | Context → Chdir → select |

## Related

- [Context](context.md) -- opens the picker, and shows how members are declared
- [Configuration](../guides/configuration.md) -- declaring members in tfui.hcl
