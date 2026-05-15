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
- On completion: full terraform output in a scrollable view

### Completion states

| State | Behavior |
|-------|----------|
| Success | Scrollable output, normal styling |
| Error | Scrollable output + visual error indicator |

User stays on the init plugin viewing output. Navigates away manually (`esc`/`q` go home, `:` goes elsewhere).

### Re-run

`Enter` from the output view reopens the form pre-filled with last-used values. Supports the common workflow: init fails, tweak a flag, retry.

### Cache

Successful init invalidates all cached state/plan data.

### Flow

```
Home → i / :init → Form (upgrade, reconfigure, backend, extra args)
  → Submit → Spinner + elapsed time → Output view
       ├── Success: scrollable output
       └── Error: scrollable output + error indicator
  → Enter → Form (pre-filled) → ...
  → esc / q → Home
```
