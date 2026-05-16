---
title: Terraform Init Plugin
status: planned
priority: medium
created: 2026-05-14
effort: small
tags: [ux, cli, tui, plugin]
depends_on: []
---

## Summary

`tfui init` mirrors `terraform init` across both interfaces: a CLI subcommand with full flag passthrough and a TUI plugin with a form-driven experience for common flags.

## Need

Users expect `tfui init` to work like every other terraform command tfui wraps. Without it, they context-switch to bare terraform for initialization — breaking the "tfui is a superset" contract. The TUI side adds value by surfacing common flags in a form instead of requiring users to remember them.

## CLI

Follows the established output contract:

```bash
tfui init
tfui init -upgrade
tfui init -backend-config=path/to/config.hcl
tfui init -reconfigure -upgrade
tfui init -- -plugin-dir=/opt/plugins
```

- stdout: terraform output (passthrough)
- stderr: spinner (if TTY, suppressed with `--ci`)
- Exit codes: `0` success, `1` error
- Full flag passthrough via `splitPassthrough()` + `normalizeArgs()`

## TUI

### Navigation

- Keybinding: `i` (menu visible)
- Command: `:init`
- Nav behavior: Replace

### Form (on activation)

| Field | Type | Default | Maps to |
|-------|------|---------|---------|
| upgrade | toggle | off | `-upgrade` |
| reconfigure | toggle | off | `-reconfigure` |
| backend | toggle | on | `-backend=false` when toggled off |
| extra args | free text | empty | appended raw |

Defaults mirror terraform's own defaults.

Users needing `-backend-config`, `-migrate-state`, or other rare flags use the extra args field.

### Execution

- Submit starts `terraform init` with selected flags
- Shows: spinner + elapsed time
- Form is the plugin's resting state; results are transient feedback

### Completion states

| State | Behavior |
|-------|----------|
| Success | Auto-returns home (emits `DeactivateMsg` + `PlanInvalidatedEvent`) |
| Error | Shows error message. `Enter` acknowledges → back to form (pre-filled for retry) |

Init is a one-shot setup action. On success, the user's intent is satisfied — auto-return avoids lingering on a "done" screen. On error, the user needs to fix and retry, so the form re-appears ready.

### Cache

Successful init invalidates all cached state/plan data.

### Flow

```
Home → i / :init → Form (upgrade, reconfigure, backend, extra args)
  → Submit → Spinner + elapsed time
       ├── Success → auto-return home (cache invalidated)
       └── Error → "Init failed: ..." → Enter → Form (pre-filled) → retry
  → Esc (on form) → Home
```
