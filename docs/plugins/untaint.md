---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Untaint: Cancel Forced Recreation"
id: untaint
key: "—"
category: action
default_enabled: true
seo:
  description: "Remove the taint mark from Terraform resources so the next apply no longer forces their recreation. The Untaint screen mirrors terraform untaint."
description: "Describes how the Untaint plugin clears a resource's taint mark."
audience: "Users who tainted a resource and no longer want it replaced"
---

# Untaint

## Overview

Removes the taint mark from resources so the next apply no longer recreates them, like `terraform untaint`. When it finishes or you cancel, `Esc` returns you to the screen you came from.

## Screenshot

![Untaint]({{ site.baseurl }}/assets/demo/untaint.gif)

## Interactive (TUI)

Untaint has no key on Home. Press `T` in State or Plan to untaint the resource under the cursor, or press `!` → `T` to untaint every pinned resource.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `y` / `Y` / `Enter` | Confirm untaint | Confirming |
| `n` / `N` / `Esc` | Cancel | Confirming |
| `p` | Navigate to plan | Done |
| `ctrl+r` | Retry | Error |
| `Esc` | Back to the screen you came from | Idle, Done, Error |

### Flow

```
Idle → Confirming → Loading → Done/Error
```

1. tfui asks "Untaint \<address\>?", or lists every address for a batch.
2. On `y`, it runs `terraform untaint` for each address in turn.
3. When it finishes, the plan is marked stale: Plan re-plans the next time you open it. `Esc` returns you to where you started; `p` opens Plan instead.

## Command Line (CLI)

```bash
tfui untaint <address>           # Untaint single resource
tfui untaint <addr1> <addr2>     # Batch untaint
```

With a terminal attached, tfui asks for confirmation; a [headless run](../reference/cli-io-contract.md#mode-resolution) untaints without asking.

Exit codes: see [CLI I/O contract](../reference/cli-io-contract.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Untaint a resource | `tfui untaint <address>` | From state/plan: `T` → `y` |
| Batch untaint | `tfui untaint <addr1> <addr2>` | Pin multiple → `!` → `T` → `y` |

## Related

- [Taint](taint.md) -- mark resources for recreation
- [State Browser](state.md) -- browse resources and untaint from context
- [Plan](plan.md) -- see the effect of untainting
