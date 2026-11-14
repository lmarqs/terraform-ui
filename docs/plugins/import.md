---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Import from a Terminal UI"
id: import
key: "—"
category: action
default_enabled: true
seo:
  description: "Import existing infrastructure into Terraform state from tfui. The Import screen mirrors terraform import and returns to where you started when done."
description: "Describes how the Import plugin brings existing infrastructure into state."
audience: "Users adopting a resource created outside terraform into state"
---

# Import

## Overview

Imports existing infrastructure into terraform state, like `terraform import`. When it finishes or you cancel, `Esc` returns you to the screen you came from.

## Screenshot

![Import]({{ site.baseurl }}/assets/demo/import.gif)

## Interactive (TUI)

Import has no key on Home. Press `n` in State to import with the address under the cursor filled in.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `Enter` | Submit the address, then the ID; an empty answer cancels | Prompt |
| `y` / `Y` / `Enter` | Confirm import | Confirming |
| `n` / `N` / `Esc` | Cancel | Confirming |
| `p` | Navigate to plan | Done |
| `ctrl+r` | Retry | Error |
| `Esc` | Back to the screen you came from | Idle, Done, Error |

### Flow

```
Prompts → Confirming → Loading → Done/Error
```

1. tfui asks "Resource address:" (filled in when you came from State), then "Resource ID:".
2. It asks "Import \<id\> as \<address\>?"
3. On `y`, it runs `terraform import <address> <id>`.
4. When it finishes, the plan is marked stale: Plan re-plans the next time you open it. `Esc` returns you to where you started; `p` opens Plan instead.

## Command Line (CLI)

```bash
tfui import <address> <id>
```

With a terminal attached, tfui asks for confirmation; a [headless run](../reference/cli-io-contract.md#mode-resolution) imports without asking.

Exit codes: see [CLI I/O contract](../reference/cli-io-contract.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Import a resource | `tfui import <address> <id>` | From state: `n` → `Enter` → type ID → `Enter` → `y` |

## Related

- [State Browser](state.md) -- browse resources and import from context
- [Plan](plan.md) -- see the effect of importing (resource now tracked)
