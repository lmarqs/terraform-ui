---
layout: default
parent: Plugins
title: Force Unlock
id: forceunlock
key: —
category: utility
---

# Force Unlock

Remove a stale terraform state lock.

## Interactive (TUI)

Access via `:forceunlock` from the command bar, or press `u` from any plugin showing a lock error.

### Flow

1. If a lock was detected (via state/plan error), the plugin shows lock details and immediately prompts for confirmation
2. If no lock is detected, offers manual lock ID entry
3. On confirmation, shows loading state while the unlock RPC executes
4. On success, emits `LockClearedEvent` (clears header badge) and `PlanInvalidatedEvent` (triggers refresh)
5. NavPush returns the user to the previous plugin

### Keybindings

| Key | Action |
|-----|--------|
| `ctrl+r` | Retry (in error state) |
| `q` / `Esc` | Back to previous plugin |

### Lock Awareness

The plugin subscribes to `LockDetectedEvent` — when any plugin encounters a lock error, the forceunlock plugin already knows the lock ID. No manual entry needed in the common case.

## CLI

```bash
tfui force-unlock <lock-id>          # interactive confirmation
tfui force-unlock --force <lock-id>  # skip confirmation (CI)
```

| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

| stdout | stderr | Exit |
|--------|--------|------|
| — | Progress + result | 0/1 |

## Header Integration

When a lock is detected anywhere in the TUI, the header shows a `locked (who Xm ago)` badge on the Project line. After successful unlock, the badge clears automatically.
