---
layout: default
parent: Plugins
title: Import
id: import
key: n
category: action
default_enabled: true
description: Import existing infrastructure into terraform state interactively
---

# Import

## Overview

Import existing infrastructure into terraform state. Standalone verb plugin -- mirrors `terraform import` as a top-level command. NavPush behavior: returns to the origin plugin on completion or cancel.

## Screenshot

![Import]({{ site.baseurl }}/assets/demo/import.gif)

## Interactive (TUI)

### Entry Points

| Context | Key | Behavior |
|---------|-----|----------|
| State list/detail | `n` | Navigate to import with cursor address pre-filled |
| Command mode | `:import` | Navigate to import (empty form) |

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `Enter` | Submit form / confirm | Form / Confirming |
| `Tab` | Next field | Form |
| `Esc` | Cancel and return | Any |
| `p` | Navigate to plan | Done |
| `ctrl+r` | Retry | Error |

### Flow

```
Form → Confirming → Loading → Done/Error
```

1. Plugin shows form: prompts for resource address (pre-filled if from state), then resource ID
2. Shows confirmation: "Import \<id\> as \<address\>?"
3. On confirmation, executes `terraform import <address> <id>`
4. On success, emits `StateRefreshedEvent` + `PlanInvalidatedEvent`
5. NavPush returns user to origin plugin

## Command Line (CLI)

```bash
tfui import <address> <id>     # Direct import, no TUI
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Import succeeded |
| 1 | Import failed |

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Import a resource | `tfui import <address> <id>` | From state: `n` → fill form → `y` |

## Configuration

```hcl
# tfui.hcl
plugin "import" {
  enabled = true
}
```

## Related

- [State Browser](state.md) -- browse resources and import from context
- [Plan](plan.md) -- see the effect of importing (resource now tracked)
