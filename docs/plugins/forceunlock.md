---
layout: default
parent: Plugins
title: Force Unlock
id: forceunlock
key:
category: utility
default_enabled: true
description: Remove a stale terraform state lock safely from the TUI
---

## Overview

Remove a stale terraform state lock. Access via `:forceunlock` from the command bar, or press `u` from any plugin showing a lock error. The plugin subscribes to `LockDetectedEvent` -- when any plugin encounters a lock error, the lock ID is already known (no manual entry needed).

## Interactive (TUI)

### Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `y` | Confirm unlock | Confirming |
| `n` / `Esc` | Cancel and return | Confirming |
| `ctrl+r` | Retry | Error |
| `q` / `Esc` | Back to previous plugin | Always |

### Flow

```
Idle → Confirming → Loading → Done/Error
```

1. If a lock was detected (via state/plan error), shows lock details and prompts for confirmation
2. If no lock is detected, offers manual lock ID entry
3. On confirmation, executes `terraform force-unlock`
4. On success, emits `LockClearedEvent` (clears header badge) + `PlanInvalidatedEvent` (triggers refresh)
5. NavPush returns user to previous plugin

## Command Line (CLI)

```bash
tfui force-unlock <lock-id>          # Interactive confirmation
tfui force-unlock --force <lock-id>  # Skip confirmation (CI)
```

| Code | Meaning |
|------|---------|
| 0 | Unlock succeeded |
| 1 | Unlock failed |

## Configuration

```hcl
# tfui.hcl
plugin "forceunlock" {
  enabled = true
}
```

## Screenshots

```
Force Unlock

Lock detected:
  ID:      abc123-def456-789
  Who:     user@host
  Created: 5m ago

Remove this lock? [y]es / [n]o
```

When a lock is active, the header shows a `locked (who Xm ago)` badge on the Project line. After successful unlock, the badge clears automatically.

## Related

- [State Browser](state.md) -- press `u` from lock error to reach this plugin
- [Plan](plan.md) -- press `u` from lock error during planning
