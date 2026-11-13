---
layout: default
parent: Plugin Catalog — All Terraform UI Features
title: "Terraform Force Unlock for Stale State Locks"
id: forceunlock
key: "—"
category: utility
default_enabled: true
seo:
  description: "Remove a stale Terraform state lock from tfui. Press u on any screen showing a lock error and the lock ID is already filled in, no manual entry needed."
description: "Describes how the Force Unlock plugin removes a stale state lock."
audience: "Users whose state is stuck behind a lock"
---

# Force Unlock

## Overview

Remove a stale terraform state lock. When a plugin hits a lock error, tfui keeps the lock ID, so `u` opens this screen with it filled in.

## Screenshot

![Force Unlock]({{ site.baseurl }}/assets/demo/forceunlock.gif)

## Interactive (TUI)

Press `u` on a lock error in State or Plan, or type `:forceunlock` in command mode.

### Keybindings

| Key | Action | When |
|-----|--------|------|
| `y` / `Y` / `Enter` | Confirm unlock | Confirming |
| `n` / `N` / `Esc` | Cancel | Confirming |
| `ctrl+r` | Retry | Error |
| `q` / `Esc` | Back to previous plugin | Always |

### Flow

```
Idle → Confirming → Loading → Done/Error
```

1. If tfui knows the lock from an earlier lock error, it asks to confirm unlocking that lock ID
2. Otherwise it asks whether to enter a lock ID by hand, then asks for it
3. On confirmation, executes `terraform force-unlock`
4. On success the lock badge clears and the plan is marked stale
5. `q` or `Esc` returns you to the previous plugin

## Command Line (CLI)

```bash
tfui force-unlock <lock-id>          # Asks y/n on the terminal
tfui force-unlock -force <lock-id>  # Skip confirmation (CI)
```

Exit codes: see [CLI reference](../reference/cli-reference.md#exit-codes).

## Equivalence

| Goal | CLI | TUI |
|------|-----|-----|
| Remove stale lock | `tfui force-unlock <lock-id>` | `u` from lock error |
| Skip confirmation | `tfui force-unlock -force <lock-id>` | N/A (TUI always confirms) |

## Related

- [State Browser](state.md) -- press `u` from lock error to reach this plugin
- [Plan](plan.md) -- press `u` from lock error during planning
