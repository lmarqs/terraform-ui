---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: Taint
id: taint
key: t
category: action
default_enabled: true
description: Mark terraform resources for recreation on next apply
---

# Taint

## Overview

Mark resources for recreation on next apply. Standalone verb plugin -- mirrors `terraform taint` as a top-level command. NavPush behavior: returns to the origin plugin on completion or cancel.

## Screenshot

![Taint]({{ site.baseurl }}/assets/demo/taint.gif)

## Interactive (TUI)

### Entry Points

| Context | Key | Behavior |
|---------|-----|----------|
| State list/detail | `t` | Navigate to taint with cursor address |
| Plan list | `t` | Navigate to taint with cursor address |
| Batch palette (`!`) | `t` | Navigate to taint with all pinned addresses |
| Command mode | `:taint` | Navigate to taint |

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `y` | Confirm taint | Confirming |
| `n` / `Esc` | Cancel and return | Confirming |
| `p` | Navigate to plan | Done |
| `Esc` | Back to previous plugin | Any |
| `ctrl+r` | Retry | Error |

### Flow

```
Idle → Confirming → Loading → Done/Error
```

1. Plugin receives target address(es) via `SetTargets()`
2. Shows confirmation: "Taint \<address\>? (will recreate on next apply)"
3. For batch: shows count and lists all addresses
4. On confirmation, executes `terraform taint` for each address sequentially
5. On success, emits `PlanInvalidatedEvent` (plan auto-replans)
6. NavPush returns user to origin plugin

## Command Line (CLI)

```bash
tfui taint <address>           # Taint single resource
tfui taint <addr1> <addr2>     # Batch taint
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Taint succeeded |
| 1 | Taint failed |

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Taint a resource | `tfui taint <address>` | From state/plan: `t` → `y` |
| Batch taint | `tfui taint <addr1> <addr2>` | Pin multiple → `!` → `t` → `y` |

## Configuration

```hcl
# tfui.hcl
plugin "taint" {
  enabled = true
}
```

## Related

- [Untaint](untaint.md) -- reverse a taint operation
- [State Browser](state.md) -- browse resources and taint from context
- [Plan](plan.md) -- see the effect of tainting (forced replacement)
