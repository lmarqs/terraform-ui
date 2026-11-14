---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Taint: Force Resource Recreation"
id: taint
key: "—"
category: action
default_enabled: true
seo:
  description: "Mark Terraform resources for recreation on the next apply from tfui. The Taint screen mirrors terraform taint and returns to where you started when done."
description: "Describes how the Taint plugin marks resources for recreation on the next apply."
audience: "Users who need a resource replaced on the next apply"
---

# Taint

## Overview

Marks resources for recreation on the next apply, like `terraform taint`. When it finishes or you cancel, `Esc` returns you to the screen you came from.

## Screenshot

![Taint]({{ site.baseurl }}/assets/demo/taint.gif)

## Interactive (TUI)

Taint has no key on Home. Press `t` in State or Plan to taint the resource under the cursor, or press `!` → `t` to taint every pinned resource.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `y` / `Y` / `Enter` | Confirm taint | Confirming |
| `n` / `N` / `Esc` | Cancel | Confirming |
| `p` | Navigate to plan | Done |
| `ctrl+r` | Retry | Error |
| `Esc` | Back to the screen you came from | Idle, Done, Error |

### Flow

```
Idle → Confirming → Loading → Done/Error
```

1. tfui asks "Taint \<address\>? (will recreate on next apply)", or lists every address for a batch.
2. On `y`, it runs `terraform taint` for each address in turn.
3. When it finishes, the plan is marked stale: Plan re-plans the next time you open it. `Esc` returns you to where you started; `p` opens Plan instead.

## Command Line (CLI)

```bash
tfui taint <address>           # Taint single resource
tfui taint <addr1> <addr2>     # Batch taint
```

With a terminal attached, tfui asks for confirmation; a [headless run](../reference/cli-io-contract.md#mode-resolution) taints without asking.

Exit codes: see [CLI I/O contract](../reference/cli-io-contract.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Taint a resource | `tfui taint <address>` | From state/plan: `t` → `y` |
| Batch taint | `tfui taint <addr1> <addr2>` | Pin multiple → `!` → `t` → `y` |

## Related

- [Untaint](untaint.md) -- reverse a taint operation
- [State Browser](state.md) -- browse resources and taint from context
- [Plan](plan.md) -- see the effect of tainting (forced replacement)
